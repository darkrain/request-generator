//go:build go1.27 && goexperiment.jsonv2

package response

import (
	"bytes"
	"testing"
)

func TestResponseBufferRetentionIsBounded(t *testing.T) {
	pool := make(responseBufferPool, 2)
	for i := 0; i < 3; i++ {
		b := new(bytes.Buffer)
		b.WriteString("private response")
		pool.put(b)
	}
	if len(pool) != 2 {
		t.Fatal("pool must not retain excess buffers")
	}
	for i := 0; i < 2; i++ {
		if b := pool.get(); b.Len() != 0 {
			t.Fatal("retained response has nonzero length")
		}
	}
	large := new(bytes.Buffer)
	large.Grow(maxResponseBufferCapacity + 1)
	pool.put(large)
	if len(pool) != 0 {
		t.Fatal("oversized buffer retained")
	}
}
