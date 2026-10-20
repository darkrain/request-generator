// Package sqlcache caches complete PostgreSQL read results below database/sql.
// It preserves database/sql scanning and the application's *sql.DB API. Enable
// only on pools with invariant session state whose reads tolerate TTL staleness.
// SQL functions used in cacheable queries must be side-effect free. Transactions,
// locking reads, mutations and known session/sequence functions bypass the cache.
package sqlcache

import (
	"container/heap"
	"context"
	"crypto/sha256"
	"database/sql/driver"
	"encoding/json"
	"fmt"
	"sync"
	"sync/atomic"
	"time"
)

type Options struct {
	TTL                      time.Duration
	MaxEntries               int
	MaxBytes, MaxResultBytes int64
}

type Stats struct {
	TTL                            time.Duration
	Hits, Misses, Shared, Bypassed uint64
	Entries                        int
	Bytes                          int64
	// Plan counters describe WrapDB's shared statement registry, including raw
	// reads. They are separate from the caller's prepared-read/result caches.
	// A prepare counts backend.PrepareContext, not PostgreSQL's per-connection
	// lazy prepares or planner invocations.
	PlanPrepares, PlanReuses, PlanEvictions, PlanErrors uint64
}

type bypassKey struct{}

// WithoutCache is for control-plane reads (for example the cache TTL itself).
func WithoutCache(ctx context.Context) context.Context {
	return context.WithValue(ctx, bypassKey{}, true)
}

type entry struct {
	data      *snapshot
	expires   time.Time
	refreshAt time.Time
	expiry    *expiryItem
}

func (e entry) usable(now time.Time, ttl time.Duration) bool {
	return now.Before(e.expires) && now.Before(e.data.completed.Add(ttl))
}

type flight struct {
	done     chan struct{}
	data     *snapshot
	err      error
	canceled bool
}

// Cache is shared by connections in one pool, never across databases.
type Cache struct {
	mu                                                  sync.RWMutex
	options                                             Options
	entries                                             map[[32]byte]entry
	expirations                                         expiryQueue
	flights                                             map[[32]byte]*flight
	hits, misses, shared, bypassed                      atomic.Uint64
	bytes                                               int64
	planPrepares, planReuses, planEvictions, planErrors atomic.Uint64
}

func New(options Options) *Cache {
	return &Cache{options: options, entries: make(map[[32]byte]entry), flights: make(map[[32]byte]*flight)}
}

func (c *Cache) Stats() Stats {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return Stats{TTL: c.options.TTL, Hits: c.hits.Load(), Misses: c.misses.Load(),
		Shared: c.shared.Load(), Bypassed: c.bypassed.Load(), Entries: len(c.entries), Bytes: c.bytes,
		PlanPrepares: c.planPrepares.Load(), PlanReuses: c.planReuses.Load(),
		PlanEvictions: c.planEvictions.Load(), PlanErrors: c.planErrors.Load()}
}

// SetTTL changes the global window. Existing entries keep their original expiry;
// a shorter TTL also caps their age at lookup. Zero disables reuse immediately.
// Writes never invalidate entries.
func (c *Cache) SetTTL(ttl time.Duration) {
	c.mu.Lock()
	if c.options.TTL != ttl {
		// The original deadline never moves. A shorter global TTL additionally
		// caps every entry's age; rebuilding is needed only on a setting change.
		for _, e := range c.entries {
			e.expiry.deadline = minExpiry(e.expires, e.data.completed.Add(ttl))
		}
		heap.Init(&c.expirations)
	}
	c.options.TTL = ttl
	c.mu.Unlock()
}

func key(query string, args []driver.NamedValue) ([32]byte, error) {
	return keyForSQLHash(hashSQL(query), args)
}

func legacyKeyForSQLHash(queryHash [32]byte, args []driver.NamedValue) ([32]byte, error) {
	type argument struct {
		Name, Type string
		Ordinal    int
		Value      interface{}
	}
	values := make([]argument, len(args))
	for i, arg := range args {
		values[i] = argument{arg.Name, fmt.Sprintf("%T", arg.Value), arg.Ordinal, arg.Value}
	}
	encoded, err := json.Marshal(values)
	if err != nil {
		return [32]byte{}, err
	}
	// Hash exact SQL bytes separately: quoting/escaping a large SQL string as
	// JSON made every cache hit allocate another copy of the whole statement.
	argsHash := sha256.Sum256(encoded)
	var combined [64]byte
	copy(combined[:32], queryHash[:])
	copy(combined[32:], argsHash[:])
	return sha256.Sum256(combined[:]), nil
}

func hashSQL(query string) [32]byte {
	// A fixed scratch buffer avoids allocating len(query) bytes just to pass
	// a string to SHA-256. Small statements fit entirely on the stack.
	var block [1024]byte
	if len(query) <= len(block) {
		return sha256.Sum256(block[:copy(block[:], query)])
	}
	h := sha256.New()
	for len(query) > 0 {
		n := copy(block[:], query)
		h.Write(block[:n])
		query = query[n:]
	}
	var result [32]byte
	h.Sum(result[:0])
	return result
}

func (c *Cache) query(ctx context.Context, query string, args []driver.NamedValue, eligible bool, run func() (driver.Rows, error)) (driver.Rows, error) {
	return c.queryWithHash(ctx, query, args, eligible, nil, run)
}

// A prepared statement retains only immutable SQL metadata. Arguments, session
// state, cancellation and the current global result TTL are checked per call.
func (c *Cache) queryWithHash(ctx context.Context, query string, args []driver.NamedValue, eligible bool, preparedHash *[32]byte, run func() (driver.Rows, error)) (driver.Rows, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	if !eligible || ctx.Value(bypassKey{}) == true {
		c.bypassed.Add(1)
		return run()
	}
	var queryHash [32]byte
	if preparedHash == nil {
		queryHash = hashSQL(query)
	} else {
		queryHash = *preparedHash
	}
	k, keyErr := keyForSQLHash(queryHash, args)
	if keyErr != nil {
		c.bypassed.Add(1)
		return run()
	}
retry:
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	// The common hit path neither sweeps unrelated entries nor takes an
	// exclusive lock. Expired entries are reclaimed on the bounded miss path.
	c.mu.RLock()
	opts := c.options
	if opts.TTL > 0 {
		if e, ok := c.entries[k]; ok {
			now := time.Now()
			if e.usable(now, opts.TTL) && (now.Before(e.refreshAt) || c.flights[k] != nil) {
				c.hits.Add(1)
				c.mu.RUnlock()
				return e.data.reader(), nil
			}
		}
		if f := c.flights[k]; f != nil {
			c.shared.Add(1)
			c.mu.RUnlock()
			rows, err, canceledLeader := awaitFlight(ctx, f, run)
			if canceledLeader {
				goto retry
			}
			return rows, err
		}
	}
	c.mu.RUnlock()
	c.mu.Lock()
	opts = c.options
	if opts.TTL <= 0 || opts.MaxEntries <= 0 || opts.MaxBytes <= 0 || opts.MaxResultBytes <= 0 {
		c.bypassed.Add(1)
		c.mu.Unlock()
		return run()
	}
	now := time.Now()
	if e, ok := c.entries[k]; ok && e.usable(now, opts.TTL) && (now.Before(e.refreshAt) || c.flights[k] != nil) {
		c.hits.Add(1)
		c.mu.Unlock()
		return e.data.reader(), nil
	}
	if f := c.flights[k]; f != nil {
		c.shared.Add(1)
		c.mu.Unlock()
		rows, err, canceledLeader := awaitFlight(ctx, f, run)
		if canceledLeader {
			goto retry
		}
		return rows, err
	}
	// Only expired entries need work. Live entries are not scanned under the
	// shared write lock on every miss.
	c.reclaimExpired(now)
	if len(c.flights) >= opts.MaxEntries {
		c.bypassed.Add(1)
		c.mu.Unlock()
		return run()
	}
	f := &flight{done: make(chan struct{})}
	c.flights[k] = f
	c.misses.Add(1)
	c.mu.Unlock()
	started := time.Now()
	rows, err := run()
	var data *snapshot
	if err == nil {
		data, rows, err = capture(ctx, rows, opts.MaxResultBytes)
	}
	c.mu.Lock()
	delete(c.flights, k)
	f.data, f.err = data, err
	f.canceled = err != nil && ctx.Err() != nil
	var fallback driver.Rows
	if err != nil && !f.canceled && c.options.TTL > 0 {
		if previous, ok := c.entries[k]; ok && previous.usable(time.Now(), c.options.TTL) {
			// An opportunistic refresh failure must not break a still-valid hit
			// or trigger retries on every request. Keep the original deadline;
			// after it passes the ordinary miss/error behavior applies.
			previous.refreshAt = previous.expires
			c.entries[k] = previous
			fallback = previous.data.reader()
		}
	}
	if err == nil && data != nil {
		entries, retained := len(c.entries), c.bytes
		if previous, ok := c.entries[k]; ok {
			entries--
			retained -= previous.data.bytes
		}
		if entries < opts.MaxEntries && retained+data.bytes <= opts.MaxBytes {
			// One foreground caller refreshes shortly before expiry. Concurrent
			// callers may still read the OLD result only inside its original TTL.
			// No driver connection escapes the caller and no stale window is added.
			// Base the head start on observed read cost, capped at 1/4 of the TTL.
			expires := data.completed.Add(opts.TTL)
			ahead := min(2*data.completed.Sub(started), opts.TTL/4)
			if previous, ok := c.entries[k]; ok {
				heap.Remove(&c.expirations, previous.expiry.index)
			}
			expiry := &expiryItem{key: k, deadline: minExpiry(expires, data.completed.Add(c.options.TTL))}
			heap.Push(&c.expirations, expiry)
			c.entries[k] = entry{data: data, expires: expires, refreshAt: expires.Add(-ahead), expiry: expiry}
			c.bytes = retained + data.bytes
		} else {
			c.bypassed.Add(1)
		}
	} else if err == nil {
		c.bypassed.Add(1)
	}
	c.mu.Unlock()
	// Waking a large group can reschedule the leader. Unrelated cache hits
	// must be able to proceed while the waiters become runnable.
	close(f.done)
	if fallback != nil {
		return fallback, nil
	}
	return rows, err
}

func awaitFlight(ctx context.Context, f *flight, run func() (driver.Rows, error)) (driver.Rows, error, bool) {
	select {
	case <-ctx.Done():
		return nil, ctx.Err(), false
	case <-f.done:
	}
	if f.err != nil {
		// An ordinary database error is shared; a canceled leader is retried
		// by live waiters under their own context, preserving request coalescing.
		return nil, f.err, f.canceled
	}
	if f.data != nil {
		return f.data.reader(), nil, false
	}
	rows, err := run()
	return rows, err, false
}
