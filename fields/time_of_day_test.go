package fields

import (
	"database/sql"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestTimeOfDayConverter(t *testing.T) {
	value, err := TimeOfDayConverter(nil, " 08:05 ")
	require.NoError(t, err)
	require.Equal(t, "08:05:00", value)

	value, err = TimeOfDayConverter(nil, "23:59:10")
	require.NoError(t, err)
	require.Equal(t, "23:59:10", value)

	_, err = TimeOfDayConverter(nil, "8pm")
	require.EqualError(t, err, "time value must use HH:MM")
}

func TestTimeOfDayResultValue(t *testing.T) {
	value := TimeOfDayResultValue(&sql.NullTime{Time: time.Date(0, 1, 1, 23, 5, 0, 0, time.UTC), Valid: true})
	require.Equal(t, "23:05", value)
	require.Nil(t, TimeOfDayResultValue(&sql.NullTime{}))
}

func TestModuleFieldFormTypeOfRecognizesTime(t *testing.T) {
	value, err := ModuleFieldFormTypeOf(string(ModuleFieldFormTypeTime))
	require.NoError(t, err)
	require.Equal(t, ModuleFieldFormTypeTime, value)
}
