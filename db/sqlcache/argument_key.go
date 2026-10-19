package sqlcache

import (
	"crypto/sha256"
	"database/sql/driver"
	"encoding/binary"
	"math"
)

// Standard driver values need no JSON reflection or temporary type-name strings.
// Every variable-length value is framed; names, ordinals and concrete types stay
// part of the key. Other values retain the existing JSON path, including its
// errors (which make the caller bypass caching). Keys live only in this process.
func keyForSQLHash(queryHash [32]byte, args []driver.NamedValue) ([32]byte, error) {
	size := 13 // domain marker and argument count
	for _, arg := range args {
		size += 17 + len(arg.Name) // name length, ordinal and value type
		switch value := arg.Value.(type) {
		case nil, bool:
		case int64:
			size += 8
		case string:
			size += 8 + len(value)
		case []byte:
			if value != nil {
				size += 8 + len(value)
			}
		case float64:
			if math.IsNaN(value) || math.IsInf(value, 0) {
				return legacyKeyForSQLHash(queryHash, args)
			}
			size += 8
		default:
			return legacyKeyForSQLHash(queryHash, args)
		}
	}
	var scratch [1024]byte
	encoded := scratch[:0]
	if size > len(scratch) {
		encoded = make([]byte, 0, size)
	}
	// Domain separation from the legacy JSON array representation.
	encoded = append(encoded, 0, 'S', 'Q', 'L', 1)
	encoded = binary.LittleEndian.AppendUint64(encoded, uint64(len(args)))
	for _, arg := range args {
		encoded = binary.LittleEndian.AppendUint64(encoded, uint64(len(arg.Name)))
		encoded = append(encoded, arg.Name...)
		encoded = binary.LittleEndian.AppendUint64(encoded, uint64(arg.Ordinal))
		switch value := arg.Value.(type) {
		case nil:
			encoded = append(encoded, 'n')
		case bool:
			if value {
				encoded = append(encoded, 't')
			} else {
				encoded = append(encoded, 'f')
			}
		case int64:
			encoded = append(encoded, 'i')
			encoded = binary.LittleEndian.AppendUint64(encoded, uint64(value))
		case float64:
			encoded = append(encoded, 'd')
			encoded = binary.LittleEndian.AppendUint64(encoded, math.Float64bits(value))
		case string:
			encoded = append(encoded, 's')
			encoded = binary.LittleEndian.AppendUint64(encoded, uint64(len(value)))
			encoded = append(encoded, value...)
		case []byte:
			if value == nil {
				encoded = append(encoded, 'N')
				continue
			}
			encoded = append(encoded, 'b')
			encoded = binary.LittleEndian.AppendUint64(encoded, uint64(len(value)))
			encoded = append(encoded, value...)
		}
	}
	argsHash := sha256.Sum256(encoded)
	var combined [64]byte
	copy(combined[:32], queryHash[:])
	copy(combined[32:], argsHash[:])
	return sha256.Sum256(combined[:]), nil
}
