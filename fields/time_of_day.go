package fields

import (
	"database/sql"
	"fmt"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
)

const timeOfDayLayout = "15:04"

// TimeOfDayConverter normalizes a form time value for a PostgreSQL TIME
// column. The renderer contract uses HH:MM; the database receives HH:MM:SS.
func TimeOfDayConverter(_ *gin.Context, value interface{}) (interface{}, error) {
	raw, ok := value.(string)
	if !ok {
		return nil, fmt.Errorf("time value must be a string")
	}
	raw = strings.TrimSpace(raw)
	for _, layout := range []string{timeOfDayLayout, "15:04:05"} {
		parsed, err := time.Parse(layout, raw)
		if err == nil {
			return parsed.Format("15:04:05"), nil
		}
	}
	return nil, fmt.Errorf("time value must use HH:MM")
}

// TimeOfDayResultValue converts the sql TIME scan value to the stable
// renderer value HH:MM. Invalid and absent values remain nil.
func TimeOfDayResultValue(value interface{}) interface{} {
	var parsed time.Time
	switch typed := value.(type) {
	case time.Time:
		parsed = typed
	case *time.Time:
		if typed == nil {
			return nil
		}
		parsed = *typed
	case sql.NullTime:
		if !typed.Valid {
			return nil
		}
		parsed = typed.Time
	case *sql.NullTime:
		if typed == nil || !typed.Valid {
			return nil
		}
		parsed = typed.Time
	case string:
		for _, layout := range []string{timeOfDayLayout, "15:04:05", time.RFC3339, time.RFC3339Nano} {
			if result, err := time.Parse(layout, typed); err == nil {
				return result.Format(timeOfDayLayout)
			}
		}
		return nil
	default:
		return nil
	}
	return parsed.Format(timeOfDayLayout)
}
