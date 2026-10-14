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

func TestPreparedDigestUsesSameCacheEntriesAsRawSQL(t *testing.T) {
	query := "SELECT " + strings.Repeat("/* exact SQL bytes */", 8192) + "$1"
	digest := hashSQL(query)
	c := testCache()
	calls := 0
	run := func() (driver.Rows, error) { calls++; return &testRows{n: 1}, nil }
	args := []driver.NamedValue{{Ordinal: 1, Value: int64(7)}}
	raw, err := c.query(context.Background(), query, args, true, run)
	if err != nil {
		t.Fatal(err)
	}
	raw.Close()
	prepared, err := c.queryWithHash(context.Background(), query, args, true, &digest, run)
	if err != nil {
		t.Fatal(err)
	}
	prepared.Close()
	if calls != 1 || c.Stats().Hits != 1 {
		t.Fatalf("raw/prepared results diverged: calls=%d stats=%+v", calls, c.Stats())
	}
	args[0].Value = int64(8)
	changed, err := c.queryWithHash(context.Background(), query, args, true, &digest, run)
	if err != nil {
		t.Fatal(err)
	}
	changed.Close()
	if calls != 2 {
		t.Fatal("different parameter reused a result")
	}
	for _, value := range []driver.Value{nil, int64(1), float64(1), "1", []byte("1"), true, time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)} {
		args[0].Value = value
		before, e1 := key(query, args)
		after, e2 := keyForSQLHash(digest, args)
		if e1 != nil || e2 != nil || before != after {
			t.Fatalf("key encoding changed for %T", value)
		}
	}
}

func BenchmarkPreparedSQLDigest(b *testing.B) {
	for _, size := range []int{128, 16 << 10, 160 << 10} {
		query := "SELECT " + strings.Repeat("x", size)
		digest := hashSQL(query)
		args := []driver.NamedValue{{Ordinal: 1, Value: int64(1)}, {Ordinal: 2, Value: "value"}}
		b.Run(fmt.Sprint(size), func(b *testing.B) {
			b.Run("raw", func(b *testing.B) {
				b.ReportAllocs()
				for i := 0; i < b.N; i++ {
					if _, err := key(query, args); err != nil {
						b.Fatal(err)
					}
				}
			})
			b.Run("prepared", func(b *testing.B) {
				b.ReportAllocs()
				for i := 0; i < b.N; i++ {
					if _, err := keyForSQLHash(digest, args); err != nil {
						b.Fatal(err)
					}
				}
			})
		})
	}
}
