//go:build !go1.27 || !goexperiment.jsonv2

package response

import (
	"encoding/json"
	"io"
)

func encodeJSON(w io.Writer, value interface{}) error {
	return json.NewEncoder(w).Encode(value)
}
