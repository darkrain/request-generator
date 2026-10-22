package sqlcache

import (
	"context"
	"database/sql/driver"
	"errors"
	"io"
	"sync/atomic"
	"testing"
	"time"
)

func primeRefresh(t *testing.T, c *Cache) [32]byte {
	t.Helper()
	rows, err := c.query(context.Background(), "SELECT value", nil, true, func() (driver.Rows, error) { return &testRows{n: 1}, nil })
	if err != nil {
		t.Fatal(err)
	}
	rows.Close()
	k, _ := key("SELECT value", nil)
	c.mu.Lock()
	e := c.entries[k]
	e.refreshAt = time.Now().Add(-time.Millisecond)
	c.entries[k] = e
	c.mu.Unlock()
	return k
}

func TestRefreshServesOnlyUnexpiredRowsAndReplacesWithinBounds(t *testing.T) {
	c := testCache()
	c.options.MaxEntries = 1
	k := primeRefresh(t, c)
	started, release, done := make(chan struct{}), make(chan struct{}), make(chan error, 1)
	var calls atomic.Int32
	run := func() (driver.Rows, error) { calls.Add(1); close(started); <-release; return &testRows{n: 2}, nil }
	go func() {
		rows, err := c.query(context.Background(), "SELECT value", nil, true, run)
		if rows != nil {
			rows.Close()
		}
		done <- err
	}()
	<-started
	// A refresh must not make concurrent readers wait while the old TTL is valid.
	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()
	rows, err := c.query(ctx, "SELECT value", nil, true, run)
	if err != nil {
		close(release)
		t.Fatal(err)
	}
	if err := rows.Next(make([]driver.Value, 1)); err != nil {
		t.Fatal(err)
	}
	if err := rows.Next(make([]driver.Value, 1)); err != io.EOF {
		t.Fatal("did not receive old snapshot")
	}
	rows.Close()
	// Once the original deadline passes, readers must wait rather than get stale rows.
	c.mu.Lock()
	e := c.entries[k]
	e.expires = time.Now().Add(-time.Second)
	c.entries[k] = e
	c.mu.Unlock()
	expired, cancelExpired := context.WithTimeout(context.Background(), 10*time.Millisecond)
	defer cancelExpired()
	if _, err := c.query(expired, "SELECT value", nil, true, run); !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("stale rows returned: %v", err)
	}
	close(release)
	if err := <-done; err != nil {
		t.Fatal(err)
	}
	if calls.Load() != 1 {
		t.Fatal("duplicate refresh")
	}
	c.mu.RLock()
	current := c.entries[k]
	retained := c.bytes
	count := len(c.entries)
	c.mu.RUnlock()
	if count != 1 || retained != current.data.bytes || len(current.data.rows) != 2 {
		t.Fatal("refresh did not replace within cache limits")
	}
	if current.expires.Sub(current.refreshAt) > c.options.TTL/4 {
		t.Fatal("refresh started too early")
	}
}

func TestFailedRefreshNeverExtendsTTL(t *testing.T) {
	c := testCache()
	k := primeRefresh(t, c)
	before := c.entries[k]
	rows, err := c.query(context.Background(), "SELECT value", nil, true, func() (driver.Rows, error) { return nil, io.ErrUnexpectedEOF })
	if err != nil {
		t.Fatal(err)
	}
	rows.Close()
	rows, err = c.query(context.Background(), "SELECT value", nil, true, func() (driver.Rows, error) {
		t.Fatal("failed refresh retried before original expiry")
		return nil, io.ErrUnexpectedEOF
	})
	if err != nil {
		t.Fatal(err)
	}
	rows.Close()
	after := c.entries[k]
	if after.expires != before.expires || after.data != before.data {
		t.Fatal("failed refresh changed old snapshot")
	}
	// A global TTL reduction is applied even to an entry scheduled for refresh.
	c.SetTTL(time.Nanosecond)
	var calls int
	rows, err = c.query(context.Background(), "SELECT value", nil, true, func() (driver.Rows, error) { calls++; return &testRows{n: 2}, nil })
	if err != nil {
		t.Fatal(err)
	}
	rows.Close()
	if calls != 1 {
		t.Fatal("shorter global TTL ignored")
	}
}

func TestPoolRefreshDoesNotBlockFreshHitsOnBackendPool(t *testing.T) {
	c, front, backend := postgresPool(t, 1)
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	var value int
	if err := front.QueryRowContext(ctx, "SELECT 42").Scan(&value); err != nil {
		t.Fatal(err)
	}
	key, _ := key("SELECT 42", nil)
	held, err := backend.Conn(ctx)
	if err != nil {
		t.Fatal(err)
	}
	defer held.Close()
	c.mu.Lock()
	e := c.entries[key]
	e.refreshAt = time.Now().Add(-time.Millisecond)
	c.entries[key] = e
	c.mu.Unlock()
	leader, cancelLeader := context.WithCancel(ctx)
	defer cancelLeader()
	done := make(chan error, 1)
	go func() { var value int; done <- front.QueryRowContext(leader, "SELECT 42").Scan(&value) }()
	deadline := time.Now().Add(time.Second)
	for backend.Stats().WaitCount == 0 && time.Now().Before(deadline) {
		time.Sleep(time.Millisecond)
	}
	if backend.Stats().WaitCount == 0 {
		t.Fatal("refresh did not acquire backend")
	}
	short, stop := context.WithTimeout(ctx, 100*time.Millisecond)
	defer stop()
	if err := front.QueryRowContext(short, "SELECT 42").Scan(&value); err != nil || value != 42 {
		t.Fatalf("fresh hit blocked behind refresh: %d %v", value, err)
	}
	cancelLeader()
	if err := <-done; !errors.Is(err, context.Canceled) {
		t.Fatalf("refresh cancellation: %v", err)
	}
	if c.entries[key].expires != e.expires {
		t.Fatal("canceled refresh extended TTL")
	}
}
