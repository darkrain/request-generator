//go:build go1.27 && goexperiment.jsonv2

package response

import (
	"bytes"
	"encoding/json"
	jsonv2 "encoding/json/v2"
	"io"
)

// Bound idle storage to 4 MiB. Larger responses still work, but their buffers
// are released after use. Only bytes are reused, never response values or
// authorization results. Each active response owns its buffer exclusively.
const maxResponseBufferCapacity = 256 << 10

type responseBufferPool chan *bytes.Buffer

var responseBuffers = make(responseBufferPool, 16)

func (pool responseBufferPool) get() *bytes.Buffer {
	select {
	case buffer := <-pool:
		return buffer
	default:
		return new(bytes.Buffer)
	}
}

func (pool responseBufferPool) put(buffer *bytes.Buffer) {
	if buffer.Cap() > maxResponseBufferCapacity {
		return
	}
	buffer.Reset()
	select {
	case pool <- buffer:
	default:
	}
}

func encodeJSON(w io.Writer, value interface{}) error {
	buffer := responseBuffers.get()
	defer responseBuffers.put(buffer)
	// MarshalWrite can reuse bytes.Buffer's storage. Keep the complete value
	// private until encoding succeeds, just like json.Encoder.Encode, and use
	// all v1 options (including deterministic keys, escaping and nil values).
	if err := jsonv2.MarshalWrite(buffer, value, json.DefaultOptionsV1()); err != nil {
		return err
	}
	buffer.WriteByte('\n')
	_, err := w.Write(buffer.Bytes())
	return err
}
