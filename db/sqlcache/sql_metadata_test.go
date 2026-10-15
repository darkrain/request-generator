package sqlcache

import (
	"context"
	"crypto/sha256"
	"database/sql/driver"
	"fmt"
	"math"
	"strings"
	"sync"
	"testing"
	"time"
)

func TestSQLMetadataExactQueriesAndOriginalKeys(t *testing.T) {
	var cache sqlMetadataCache
	queries := []string{"SELECT $1", "SELECT  $1", "SELECT '$1'", "SELECT nextval('x')", "SELECT 1 FOR UPDATE", "WITH x AS (DELETE FROM records RETURNING *) SELECT * FROM x", "SET ROLE x", "SELECT '\xff'", "SELECT '\xfe'"}
	values := []any{nil, int64(1), float64(1), "1", []byte("1"), true, time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC), math.NaN()}
	for _, query := range queries {
		for repeat := 0; repeat < 2; repeat++ {
			m := cache.lookup(query)
			if m.digest != sha256.Sum256([]byte(query)) || m.readOnly != readOnly(query) {
				t.Fatalf("metadata changed SQL classification or bytes: %q", query)
			}
			for _, value := range values {
				for _, name := range []string{"", "arg"} {
					args := []driver.NamedValue{{Name: name, Ordinal: 1, Value: value}}
					old, oe := key(query, args)
					next, ne := keyWithSQLHash(m.digest, args)
					if (oe == nil) != (ne == nil) || old != next {
						t.Fatalf("key changed for %q / %T", query, value)
					}
				}
			}
		}
	}
}

func TestSQLMetadataBoundedAndConcurrent(t *testing.T) {
	var cache sqlMetadataCache
	var wg sync.WaitGroup
	for w := 0; w < 8; w++ {
		wg.Add(1)
		go func(worker int) {
			defer wg.Done()
			for i := 0; i < 600; i++ {
				query := fmt.Sprintf("SELECT %d, %d", worker, i)
				if cache.lookup(query).digest != sha256.Sum256([]byte(query)) {
					t.Error("wrong concurrent digest")
				}
			}
		}(w)
	}
	wg.Wait()
	if len(cache.entries) > maxSQLMetadataEntries || cache.bytes > maxSQLMetadataBytes {
		t.Fatal("unbounded metadata")
	}
	for i := 0; i < 6; i++ {
		cache.lookup("SELECT '" + strings.Repeat("x", 1<<20) + fmt.Sprintf("%d'", i))
	}
	if cache.bytes > maxSQLMetadataBytes {
		t.Fatal("byte limit exceeded")
	}
	count, bytes := len(cache.entries), cache.bytes
	huge := "SELECT '" + strings.Repeat("z", maxSQLMetadataBytes) + "'"
	if cache.lookup(huge).digest != sha256.Sum256([]byte(huge)) {
		t.Fatal("oversized SQL changed")
	}
	if count != len(cache.entries) || bytes != cache.bytes {
		t.Fatal("oversized SQL retained")
	}
}

func TestSQLMetadataEvictionDoesNotEvictResults(t *testing.T) {
	c := testCache()
	run := func() (driver.Rows, error) { return &testRows{n: 1}, nil }
	r, err := c.query(context.Background(), "SELECT value", nil, true, run)
	if err != nil {
		t.Fatal(err)
	}
	r.Close()
	for i := 0; i < maxSQLMetadataEntries+1; i++ {
		c.sqlMetadata.lookup(fmt.Sprintf("SELECT %d", i))
	}
	r, err = c.query(context.Background(), "SELECT value", nil, true, run)
	if err != nil {
		t.Fatal(err)
	}
	r.Close()
	if c.Stats().Misses != 1 || c.Stats().Hits != 1 {
		t.Fatal("metadata eviction invalidated result cache")
	}
}

func BenchmarkSQLMetadataAndKey(b *testing.B) {
	query := "SELECT " + strings.Repeat(`CASE WHEN "value"=$1 THEN $2 ELSE NULL END, `, 512) + "id FROM records"
	args := []driver.NamedValue{{Ordinal: 1, Value: int64(1)}, {Ordinal: 2, Value: "value"}}
	b.Run("uncached", func(b *testing.B) {
		b.ReportAllocs()
		for i := 0; i < b.N; i++ {
			if !readOnly(query) {
				b.Fatal("unsafe")
			}
			if _, e := key(query, args); e != nil {
				b.Fatal(e)
			}
		}
	})
	b.Run("cached", func(b *testing.B) {
		var c sqlMetadataCache
		c.lookup(query)
		b.ReportAllocs()
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			if !c.lookup(query).readOnly {
				b.Fatal("unsafe")
			}
			if _, e := keyWithSQLHash(c.lookup(query).digest, args); e != nil {
				b.Fatal(e)
			}
		}
	})
}
