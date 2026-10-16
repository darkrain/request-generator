package sqlcache

import (
	"context"
	"crypto/sha256"
	"database/sql/driver"
	"fmt"
	"strings"
	"sync"
	"testing"
	"time"
)

func TestHashSQLPreservesExactBytes(t *testing.T) {
	for _, size := range []int{0, 1, 1023, 1024, 1025, 32768} {
		query := strings.Repeat("\xff", size)
		if hashSQL(query) != sha256.Sum256([]byte(query)) {
			t.Fatalf("wrong digest at length %d", size)
		}
	}
}

func TestKeySeparatesSQLAndTypedArguments(t *testing.T) {
	seen := map[[32]byte]bool{}
	for _, query := range []string{"SELECT $1", "SELECT  $1", "SELECT '\xff'", "SELECT '\xfe'"} {
		for _, value := range []driver.Value{nil, int64(1), float64(1), "1", []byte("1"), true} {
			k, err := key(query, []driver.NamedValue{{Ordinal: 1, Value: value}})
			if err != nil || seen[k] {
				t.Fatalf("key collision or error: %v", err)
			}
			seen[k] = true
		}
	}
}

func TestConcurrentCacheHitsStatsAndTTL(t *testing.T) {
	c := testCache()
	run := func() (driver.Rows, error) { return &testRows{n: 1}, nil }
	var wg sync.WaitGroup
	for worker := 0; worker < 8; worker++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for i := 0; i < 100; i++ {
				c.SetTTL(time.Second)
				r, err := c.query(context.Background(), "SELECT value", nil, true, run)
				if err != nil {
					t.Error(err)
					return
				}
				r.Close()
				c.Stats()
			}
		}()
	}
	wg.Wait()
	stats := c.Stats()
	if stats.Misses != 1 || stats.Hits+stats.Shared+stats.Misses != 800 {
		t.Fatalf("inconsistent counters: %+v", stats)
	}
	// Shortening TTL must reject even a hit whose original expiry is later.
	c.mu.Lock()
	for _, e := range c.entries {
		e.data.completed = time.Now().Add(-time.Second)
	}
	c.mu.Unlock()
	c.SetTTL(time.Millisecond)
	r, err := c.query(context.Background(), "SELECT value", nil, true, run)
	if err != nil {
		t.Fatal(err)
	}
	r.Close()
	if c.Stats().Misses != 2 {
		t.Fatal("stale fast-path hit after TTL shortening")
	}
}

func BenchmarkKeyLargeQuery(b *testing.B) {
	query := "SELECT " + strings.Repeat(`CASE WHEN "value"=$1 THEN $2 ELSE NULL END, `, 512) + "id FROM records"
	args := []driver.NamedValue{{Ordinal: 1, Value: int64(1)}, {Ordinal: 2, Value: "value"}}
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		if _, err := key(query, args); err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkCachedQueryParallel(b *testing.B) {
	c := New(Options{TTL: time.Hour, MaxEntries: 512, MaxBytes: 32 << 20, MaxResultBytes: 1 << 20})
	queries := make([]string, 128)
	run := func() (driver.Rows, error) { return &testRows{n: 1}, nil }
	for i := range queries {
		queries[i] = fmt.Sprintf("SELECT value FROM records WHERE id=%d", i)
		r, err := c.query(context.Background(), queries[i], nil, true, run)
		if err != nil {
			b.Fatal(err)
		}
		r.Close()
	}
	b.ReportAllocs()
	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		i := 0
		for pb.Next() {
			r, err := c.query(context.Background(), queries[i%len(queries)], nil, true, run)
			if err != nil {
				b.Error(err)
				return
			}
			r.Close()
			i++
		}
	})
}

func TestPreparedSQLDigestMatchesDirectCacheKey(t *testing.T) {
	for _, query := range []string{"SELECT $1", "SELECT  $1", "SELECT '\xff'", strings.Repeat("SELECT $1;", 4096)} {
		digest := hashSQL(query)
		for _, value := range []driver.Value{nil, int64(1), float64(1), "1", []byte("1"), true, time.Unix(123, 456).UTC()} {
			args := []driver.NamedValue{{Name: "input", Ordinal: 1, Value: value}}
			direct, err := key(query, args)
			prepared, preparedErr := keyWithSQLHash(digest, args)
			if err != nil || preparedErr != nil || direct != prepared {
				t.Fatalf("prepared and direct cache keys differ: %v %v", err, preparedErr)
			}
		}
	}
}

func TestPreparedAndDirectQueriesShareResult(t *testing.T) {
	cache := testCache()
	query := "SELECT value WHERE id=$1"
	digest := hashSQL(query)
	args := []driver.NamedValue{{Ordinal: 1, Value: int64(7)}}
	calls := 0
	run := func() (driver.Rows, error) { calls++; return &testRows{n: 1}, nil }
	first, err := cache.query(context.Background(), query, args, true, run)
	if err != nil {
		t.Fatal(err)
	}
	first.Close()
	second, err := cache.queryWithSQLHash(context.Background(), query, &digest, args, true, run)
	if err != nil {
		t.Fatal(err)
	}
	second.Close()
	if calls != 1 {
		t.Fatalf("prepared query missed direct result: calls=%d", calls)
	}
	args[0].Value = int64(8)
	third, err := cache.queryWithSQLHash(context.Background(), query, &digest, args, true, run)
	if err != nil {
		t.Fatal(err)
	}
	third.Close()
	if calls != 2 {
		t.Fatal("prepared query reused a result for different arguments")
	}
}

func BenchmarkKeyPreparedLargeQuery(b *testing.B) {
	query := "SELECT " + strings.Repeat(`CASE WHEN "value"=$1 THEN $2 ELSE NULL END, `, 512) + "id FROM records"
	digest := hashSQL(query)
	args := []driver.NamedValue{{Ordinal: 1, Value: int64(1)}, {Ordinal: 2, Value: "value"}}
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		if _, err := keyWithSQLHash(digest, args); err != nil {
			b.Fatal(err)
		}
	}
}
