package db

import (
	"encoding/json"
	"fmt"
	"testing"

	pg "github.com/go-jet/jet/v2/postgres"
	"github.com/stretchr/testify/require"
)

// Reference for the previous transport conversion, kept independent of the
// optimized decoder to catch type, null, duplicate and ordering changes.
func oldListJoin(raw []byte, columns []pg.Column) ([]map[string]interface{}, error) {
	var rows [][]interface{}
	if err := json.Unmarshal(raw, &rows); err != nil {
		return nil, err
	}
	check := ""
	for _, row := range rows {
		for _, v := range row {
			if v != nil {
				check = fmt.Sprintf("%v%v", check, v)
			}
		}
	}
	result := make([]map[string]interface{}, 0, 10)
	if check == "" {
		return result, nil
	}
	var encoded []string
	for _, row := range rows {
		value := map[string]interface{}{}
		for i, column := range columns {
			value[column.Name()] = row[i]
		}
		data, err := json.Marshal(value)
		if err != nil {
			return nil, err
		}
		encoded = append(encoded, string(data))
	}
	seen := map[string]bool{}
	for _, value := range encoded {
		if seen[value] {
			continue
		}
		seen[value] = true
		var decoded map[string]interface{}
		if err := json.Unmarshal([]byte(value), &decoded); err != nil {
			return nil, err
		}
		result = append(result, decoded)
	}
	return result, nil
}

func TestListJoinTransportEquivalence(t *testing.T) {
	for _, raw := range []string{
		`null`, `[]`, `[[null,null]]`, `[["",""],[null,""]]`,
		`[["a",1],["a",1],["b",2],[null,null],["",""]]`,
		`[[false,0],[true,-0],[0.125,1e20]]`,
		`[[{"nested":[null,1,"x"]},[]],[{"nested":[null,1,"x"]},[]],[{},{}]]`,
		`[["\\\"\n<>&", "юникод"],["\\\"\n<>&", "юникод"]]`,
	} {
		for _, names := range [][]string{{"left", "right"}, {"same", "same"}} {
			columns := []pg.Column{pg.StringColumn(names[0]), pg.StringColumn(names[1])}
			want, err := oldListJoin([]byte(raw), columns)
			require.NoError(t, err)
			got, err := decodeListJoin([]byte(raw), columns)
			require.NoError(t, err)
			require.Equal(t, want, got, "%s, %v", raw, names)
			wantJSON, err := json.Marshal(want)
			require.NoError(t, err)
			gotJSON, err := json.Marshal(got)
			require.NoError(t, err)
			require.Equal(t, string(wantJSON), string(gotJSON))
		}
	}
	_, err := decodeListJoin([]byte(`[["short"]]`), []pg.Column{pg.StringColumn("a"), pg.StringColumn("b")})
	require.Error(t, err)
}

func BenchmarkListJoinConversion(b *testing.B) {
	columns := []pg.Column{pg.StringColumn("label"), pg.StringColumn("value")}
	rows := make([][]interface{}, 100)
	for i := range rows {
		rows[i] = []interface{}{fmt.Sprintf("label %d", i%50), map[string]interface{}{"number": i % 50, "nested": []interface{}{true, "text"}}}
	}
	raw, err := json.Marshal(rows)
	if err != nil {
		b.Fatal(err)
	}
	for name, decode := range map[string]func([]byte, []pg.Column) ([]map[string]interface{}, error){"before": oldListJoin, "after": decodeListJoin} {
		b.Run(name, func(b *testing.B) {
			b.ReportAllocs()
			for i := 0; i < b.N; i++ {
				if _, err := decode(raw, columns); err != nil {
					b.Fatal(err)
				}
			}
		})
	}
}
