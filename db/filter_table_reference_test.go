package db

import (
	"testing"

	pg "github.com/go-jet/jet/v2/postgres"
	"github.com/stretchr/testify/require"
)

func TestColumnTableRefUsesDeclaredColumnTable(t *testing.T) {
	baseID := pg.IntegerColumn("id")
	joinedEnabled := pg.BoolColumn("enabled")
	base := pg.NewTable("public", "base_records", "", baseID)
	joined := pg.NewTable("public", "joined_records", "", joinedEnabled)

	require.Equal(t, base.TableName(), columnTableRef(baseID, "fallback"))
	require.Equal(t, joined.TableName(), columnTableRef(joinedEnabled, base.TableName()))
	require.Equal(t, "fallback", columnTableRef(pg.BoolColumn("computed"), "fallback"))
}
