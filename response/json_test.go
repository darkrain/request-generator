package response

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"strings"
	"sync"
	"testing"
	"time"
)

type brokenJSON struct{}

func (brokenJSON) MarshalJSON() ([]byte, error) { return []byte(`{"unfinished":`), nil }

type failingJSON struct{}

func (failingJSON) MarshalJSON() ([]byte, error) { return nil, errors.New("encode failure") }

type errorWriter struct{ err error }

func (w errorWriter) Write([]byte) (int, error) { return 0, w.err }

func TestEncodeJSONPreservesStandardBytes(t *testing.T) {
	large := strings.Repeat("<script>日本語 & \"text\"\n", 8192)
	values := []interface{}{
		nil, []string(nil), []string{}, map[string]interface{}(nil), map[string]interface{}{},
		map[string]interface{}{"z": nil, "a": large, "line": "\u2028\u2029", "invalid": "\xff"},
		[]interface{}{int64(-9223372036854775807), uint64(18446744073709551615), -0.125, json.Number("1.234567890123456789")},
		map[int]string{10: "ten", 2: "two"},
		json.RawMessage(` { "raw": [null,true,1.25], "text": "<>&" } `),
		struct {
			Null []int     `json:"null"`
			Omit []int     `json:"omit,omitempty"`
			Zero int       `json:"zero,string"`
			When time.Time `json:"when"`
		}{When: time.Date(2026, 1, 2, 3, 4, 5, 123456789, time.UTC)},
	}
	for i, value := range values {
		t.Run(fmt.Sprint(i), func(t *testing.T) {
			var want, got bytes.Buffer
			if err := json.NewEncoder(&want).Encode(value); err != nil {
				t.Fatal(err)
			}
			if err := encodeJSON(&got, value); err != nil {
				t.Fatal(err)
			}
			if !bytes.Equal(want.Bytes(), got.Bytes()) {
				t.Fatal("JSON bytes differ from standard encoder")
			}
		})
	}
}

func TestEncodeJSONErrorsDoNotWritePartialResponse(t *testing.T) {
	cycle := map[string]interface{}{}
	cycle["cycle"] = cycle
	for _, value := range []interface{}{
		map[string]interface{}{"prefix": strings.Repeat("x", 100000), "suffix": failingJSON{}},
		brokenJSON{}, json.RawMessage(`{`), math.Inf(1), math.NaN(), make(chan int), cycle,
	} {
		var got bytes.Buffer
		if err := encodeJSON(&got, value); err == nil {
			t.Fatalf("expected error for %T", value)
		}
		if got.Len() != 0 {
			t.Fatal("partial response escaped on encoding failure")
		}
		if err := encodeJSON(&got, "next"); err != nil {
			t.Fatal(err)
		}
		if got.String() != "\"next\"\n" {
			t.Fatal("previous response bytes leaked")
		}
	}
	err := errors.New("write failure")
	if got := encodeJSON(errorWriter{err}, map[string]string{"value": "text"}); got != err {
		t.Fatalf("writer error: %v", got)
	}
}

func TestEncodeJSONConcurrentResponsesStayIndependent(t *testing.T) {
	var wg sync.WaitGroup
	for i := 0; i < 32; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			value := map[string]interface{}{"id": i, "text": strings.Repeat(fmt.Sprint(i), 2000)}
			var want bytes.Buffer
			if err := json.NewEncoder(&want).Encode(value); err != nil {
				t.Error(err)
				return
			}
			for n := 0; n < 40; n++ {
				var got bytes.Buffer
				if err := encodeJSON(&got, value); err != nil {
					t.Error(err)
					return
				}
				if !bytes.Equal(want.Bytes(), got.Bytes()) {
					t.Error("response bytes differ")
					return
				}
			}
		}(i)
	}
	wg.Wait()
}
