package sqlcache

import (
	"database/sql/driver"
	"math"
	"strconv"
	"strings"
	"testing"
	"time"
)

func TestArgumentKeyFramesValuesAndIdentity(t *testing.T) {
	seen := make(map[[32]byte]bool)
	values := []interface{}{nil, false, true, int64(0), int64(1), int64(-1), int64(math.MaxInt64),
		float64(0), math.Copysign(0, -1), float64(1), "", "1", "a", "ab", "bc", "c",
		"\x00", "\xff", "\xfe", "\ufffd", []byte(nil), []byte{}, []byte("1"), []byte("a"),
		[]byte("ab"), []byte("bc"), []byte("c"), strings.Repeat("long", 1024)}
	for _, value := range values {
		for _, name := range []string{"", "first", "first\x00second"} {
			for _, ordinal := range []int{-1, 0, 1, 2} {
				args := []driver.NamedValue{{Name: name, Ordinal: ordinal, Value: value}}
				k, err := key("SELECT $1", args)
				if err != nil || seen[k] {
					t.Fatalf("collision/error for %T, name %q, ordinal %d: %v", value, name, ordinal, err)
				}
				seen[k] = true
				again, err := key("SELECT $1", args)
				if err != nil || again != k {
					t.Fatal("unstable key")
				}
			}
		}
	}
	for _, args := range [][]driver.NamedValue{
		nil, {}, {{Value: "ab"}, {Value: "c"}}, {{Value: "a"}, {Value: "bc"}},
		{{Value: []byte("ab")}, {Value: []byte("c")}}, {{Value: []byte("a")}, {Value: []byte("bc")}},
	} {
		k, err := key("SELECT $1", args)
		if err != nil {
			t.Fatal(err)
		}
		if len(args) == 0 {
			continue // nil and empty argument lists describe the same query.
		}
		if seen[k] {
			t.Fatal("ambiguous argument boundaries")
		}
		seen[k] = true
	}
}

func TestArgumentKeyRetainsOtherTypesAndErrors(t *testing.T) {
	type namedInt int64
	queryHash := hashSQL("SELECT $1,$2")
	for _, value := range []interface{}{time.Now(), namedInt(1), int(1), map[string]int{"a": 1},
		math.NaN(), math.Inf(1), math.Inf(-1), make(chan int)} {
		args := []driver.NamedValue{{Ordinal: 1, Value: int64(7)}, {Ordinal: 2, Value: value}}
		want, wantErr := legacyKeyForSQLHash(queryHash, args)
		got, err := keyForSQLHash(queryHash, args)
		if got != want || (err == nil) != (wantErr == nil) {
			t.Fatalf("changed behavior for %T: %v / %v", value, err, wantErr)
		}
	}
}

func BenchmarkArgumentKey(b *testing.B) {
	hash := hashSQL("SELECT parameterized_expression")
	for _, count := range []int{2, 32, 128} {
		args := make([]driver.NamedValue, count)
		for i := range args {
			args[i] = driver.NamedValue{Ordinal: i + 1, Value: int64(i)}
		}
		for name, build := range map[string]func([32]byte, []driver.NamedValue) ([32]byte, error){
			"binary": keyForSQLHash, "json": legacyKeyForSQLHash,
		} {
			b.Run(name+"/"+strconv.Itoa(count), func(b *testing.B) {
				b.ReportAllocs()
				for i := 0; i < b.N; i++ {
					if _, err := build(hash, args); err != nil {
						b.Fatal(err)
					}
				}
			})
		}
	}
}
