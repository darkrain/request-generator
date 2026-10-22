package db

import (
	"container/list"
	"context"
	"database/sql"
	"errors"
	"sync"
)

type preparedReadEntry struct {
	query    string
	stmt     *sql.Stmt
	position *list.Element
	users    int
	retired  bool
}

type preparedViewCache struct {
	mu         sync.Mutex
	limit      int
	statements map[string]*preparedReadEntry
	pending    map[string]chan struct{}
	recent     list.List
	stats      PreparedReadCacheStats
}

// PreparedReadCacheStats reports bounded statement retention without exposing
// SQL text, arguments or results. Counters cover this executor's lifetime.
type PreparedReadCacheStats struct {
	Capacity      int
	Entries       int
	Pending       int
	Hits          uint64
	Misses        uint64
	Bypasses      uint64
	Evictions     uint64
	PrepareErrors uint64
}

// NewDBWithPreparedReads reuses parameterized read statements in a bounded LRU
// cache. Values and results are never retained here. A new shape may replace an
// idle least-recently-used statement; if every slot is preparing or in use, it
// executes normally. This keeps later hot queries eligible after a broad scan.
// A nonpositive limit disables preparation. Call ClosePreparedViews after
// draining requests; the supplied connection pool remains owned by the caller.
func NewDBWithPreparedReads(pool *sql.DB, maxStatements int) *DB {
	db := NewDB(pool)
	if maxStatements > 0 {
		db.preparedViews = &preparedViewCache{limit: maxStatements, statements: make(map[string]*preparedReadEntry), pending: make(map[string]chan struct{})}
	}
	return db
}

// NewDBWithPreparedViews is retained for existing callers. Prepared reads also
// include generated List/COUNT and parameterized QueryContext reads.
func NewDBWithPreparedViews(pool *sql.DB, maxStatements int) *DB {
	return NewDBWithPreparedReads(pool, maxStatements)
}

func (db *DB) PreparedReadCacheStats() PreparedReadCacheStats {
	cache := db.preparedViews
	if cache == nil {
		return PreparedReadCacheStats{}
	}
	cache.mu.Lock()
	defer cache.mu.Unlock()
	stats := cache.stats
	stats.Capacity = cache.limit
	stats.Entries = len(cache.statements)
	stats.Pending = len(cache.pending)
	return stats
}

func (db *DB) queryView(query string, args ...interface{}) (*sql.Rows, error) {
	return db.QueryContext(db.queryContext(), query, args...)
}
func (db *DB) queryList(query string, args ...interface{}) (selectRows, error) {
	if db.listSelectCache != nil {
		return db.queryCachedList(query, args...)
	}
	return db.queryListSQL(query, args...)
}
func (db *DB) queryListSQL(query string, args ...interface{}) (*sql.Rows, error) {
	return db.QueryContext(db.queryContext(), query, args...)
}

// QueryContext shares the prepared-read cache. The lease protects the interval
// between retrieving a statement and beginning its query, so eviction cannot
// race with QueryContext and return "statement is closed". database/sql itself
// retains the driver's statement for any returned rows until they are closed.
func (db *DB) QueryContext(ctx context.Context, query string, args ...interface{}) (*sql.Rows, error) {
	cache := db.preparedViews
	if cache == nil {
		return db.sql.QueryContext(ctx, query, args...)
	}
	entry, err := cache.statement(ctx, db.sql, query)
	if err != nil {
		return nil, err
	}
	if entry == nil {
		return db.sql.QueryContext(ctx, query, args...)
	}
	defer cache.release(entry)
	return entry.stmt.QueryContext(ctx, args...)
}

func (cache *preparedViewCache) statement(ctx context.Context, pool *sql.DB, query string) (*preparedReadEntry, error) {
	for {
		if err := ctx.Err(); err != nil {
			return nil, err
		}
		cache.mu.Lock()
		if entry := cache.statements[query]; entry != nil {
			entry.users++
			cache.recent.MoveToFront(entry.position)
			cache.stats.Hits++
			cache.mu.Unlock()
			return entry, nil
		}
		if ready := cache.pending[query]; ready != nil {
			cache.mu.Unlock()
			select {
			case <-ctx.Done():
				return nil, ctx.Err()
			case <-ready:
			}
			continue
		}
		var evicted *sql.Stmt
		if len(cache.statements)+len(cache.pending) >= cache.limit {
			// Pinned entries may still be waiting for a pool connection. Never
			// close them or wait for them while holding the cache mutex.
			for position := cache.recent.Back(); position != nil; position = position.Prev() {
				entry := position.Value.(*preparedReadEntry)
				if entry.users != 0 {
					continue
				}
				delete(cache.statements, entry.query)
				cache.recent.Remove(position)
				entry.retired = true
				evicted = entry.stmt
				cache.stats.Evictions++
				break
			}
			if evicted == nil {
				cache.stats.Bypasses++
				cache.mu.Unlock()
				return nil, nil
			}
		}
		ready := make(chan struct{})
		cache.pending[query] = ready
		cache.stats.Misses++
		cache.mu.Unlock()
		if evicted != nil {
			_ = evicted.Close()
		}
		// Only the same SQL shape waits for this preparation. Reserve its slot
		// before releasing the mutex so concurrent misses remain bounded.
		stmt, err := pool.PrepareContext(ctx, query)
		cache.mu.Lock()
		delete(cache.pending, query)
		var entry *preparedReadEntry
		if err == nil {
			entry = &preparedReadEntry{query: query, stmt: stmt, users: 1}
			if cache.limit > 0 {
				entry.position = cache.recent.PushFront(entry)
				cache.statements[query] = entry
			} else {
				entry.retired = true
			}
		} else {
			cache.stats.PrepareErrors++
		}
		close(ready)
		cache.mu.Unlock()
		return entry, err
	}
}

func (cache *preparedViewCache) release(entry *preparedReadEntry) {
	cache.mu.Lock()
	entry.users--
	closeNow := entry.retired && entry.users == 0
	cache.mu.Unlock()
	if closeNow {
		_ = entry.stmt.Close()
	}
}

// ClosePreparedViews disables retention and releases idle statements. Leased
// statements finish safely and are closed on release; pending preparations
// also become retired. Normal shutdown should first drain this executor.
func (db *DB) ClosePreparedViews() error {
	cache := db.preparedViews
	if cache == nil {
		return nil
	}
	cache.mu.Lock()
	cache.limit = 0
	var closeNow []*sql.Stmt
	for query, entry := range cache.statements {
		entry.retired = true
		if entry.users == 0 {
			closeNow = append(closeNow, entry.stmt)
		}
		delete(cache.statements, query)
	}
	cache.recent.Init()
	cache.mu.Unlock()
	var result error
	for _, stmt := range closeNow {
		result = errors.Join(result, stmt.Close())
	}
	return result
}

// WithContext returns an immutable request-scoped copy sharing the pool and
// statement cache. Generated View and List/COUNT honor cancellation; writes
// are unchanged.
func (db *DB) WithContext(ctx context.Context) DBExecutor {
	scoped := *db
	scoped.readContext = ctx
	return &scoped
}

func (db *DB) queryContext() context.Context {
	if db.readContext != nil {
		return db.readContext
	}
	return context.Background()
}
