package renderer

import (
	"encoding/json"
	"math"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

type legacyTypedValue TypedValue

func (v legacyTypedValue) MarshalJSON() ([]byte, error) {
	payload := struct {
		Type   TypedValueType `json:"type"`
		String *string        `json:"string,omitempty"`
		Number *float64       `json:"number,omitempty"`
		Bool   *bool          `json:"bool,omitempty"`
	}{Type: v.Type}
	switch v.Type {
	case TypedValueString:
		payload.String = &v.String
	case TypedValueNumber:
		payload.Number = &v.Number
	case TypedValueBool:
		payload.Bool = v.Bool
	}
	return json.Marshal(payload)
}

func TestTypedValueEncodingMatchesLegacy(t *testing.T) {
	flag := false
	values := []TypedValue{
		{Type: TypedValueString},
		{Type: TypedValueString, String: "active_1 / ? = :", Number: 42, Bool: &flag},
		{Type: TypedValueString, String: "Кириллица 日本語 🙂"},
		{Type: TypedValueString, String: "<script>&\"\\\n\t\u2028\u2029"},
		{Type: TypedValueString, String: strings.Repeat("large", 20000)},
		{Type: TypedValueNumber, Number: math.Copysign(0, -1)},
		{Type: TypedValueNumber, Number: 1e30},
		{Type: TypedValueNumber, Number: math.NaN()},
		{Type: TypedValueNumber, Number: math.Inf(1)},
		{Type: TypedValueBool, Bool: &flag},
		{Type: TypedValueBool},
		{Type: TypedValueNull, String: "ignored", Number: 42},
		{Type: "invalid<type>"},
	}
	// Every byte covers control characters, quote/backslash, HTML delimiters,
	// DEL and invalid UTF-8, including escapes in the middle of a value.
	for i := 0; i < 256; i++ {
		values = append(values, TypedValue{Type: TypedValueString, String: "prefix" + string([]byte{byte(i)}) + "suffix"})
	}
	for _, value := range values {
		want, wantErr := legacyTypedValue(value).MarshalJSON()
		got, gotErr := value.MarshalJSON()
		require.Equal(t, wantErr != nil, gotErr != nil)
		if wantErr == nil {
			require.Equal(t, string(want), string(got))
		}
	}
}

func BenchmarkTypedValuesEnvelope(b *testing.B) {
	values := []TypedValue{}
	for i := 0; i < 8; i++ {
		values = append(values, TypedValue{Type: TypedValueString, String: "ready"})
	}
	values = append(values, TypedValue{Type: TypedValueString, String: "日本語"}, TypedValue{Type: TypedValueString, String: "<text>"})
	legacy := make([]legacyTypedValue, len(values))
	for i, v := range values {
		legacy[i] = legacyTypedValue(v)
	}
	for _, tc := range []struct {
		name   string
		values interface{}
	}{{"legacy", legacy}, {"current", values}} {
		b.Run(tc.name, func(b *testing.B) {
			payload := struct {
				Data   string
				Values interface{}
			}{strings.Repeat("data", 16384), tc.values}
			b.ReportAllocs()
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				if _, err := json.Marshal(payload); err != nil {
					b.Fatal(err)
				}
			}
		})
	}
}
