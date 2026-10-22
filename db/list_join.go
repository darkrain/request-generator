package db

import (
	"encoding/json"
	"fmt"

	pg "github.com/go-jet/jet/v2/postgres"
)

// The SQL join aggregate contains JSON values already decoded into transport
// types. Keep those maps after deduplication instead of decoding their JSON a
// second time. Each call owns its values; no result data is shared across rows.
func decodeListJoin(raw []byte, columns []pg.Column) ([]map[string]interface{}, error) {
	var rows [][]interface{}
	if err := json.Unmarshal(raw, &rows); err != nil {
		return nil, err
	}
	result := make([]map[string]interface{}, 0, len(rows))
	hasValue := false
	for _, row := range rows {
		for _, value := range row {
			if value == nil {
				continue
			}
			if text, ok := value.(string); ok && text == "" {
				continue
			}
			hasValue = true
			break
		}
		if hasValue {
			break
		}
	}
	// Preserve the existing empty-join rule, including all-empty strings.
	if !hasValue {
		return result, nil
	}
	seen := make(map[string]struct{}, len(rows))
	for _, row := range rows {
		if len(row) < len(columns) {
			return nil, fmt.Errorf("join row has %d columns, expected %d", len(row), len(columns))
		}
		value := make(map[string]interface{}, len(columns))
		for i, column := range columns {
			value[column.Name()] = row[i]
		}
		encoded, err := json.Marshal(value)
		if err != nil {
			return nil, err
		}
		key := string(encoded)
		if _, exists := seen[key]; exists {
			continue
		}
		seen[key] = struct{}{}
		result = append(result, value)
	}
	return result, nil
}
