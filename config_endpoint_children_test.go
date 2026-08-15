package module

import (
	"testing"

	"github.com/darkrain/request-generator/actions"
	pg "github.com/go-jet/jet/v2/postgres"
	"github.com/stretchr/testify/require"
)

func TestRecordChildRouteBindsStandardSelector(t *testing.T) {
	id := pg.IntegerColumn("id")
	mod := &BaseModule{Name: "records", Path: "/api", PrimaryKey: id}
	query := standardRecordActionRouteQuery(mod, actions.ViewModuleAction{By: []pg.Column{pg.StringColumn("nick"), id}}, []pg.Column{pg.StringColumn("nick"), id})

	require.NotNil(t, query)
	require.Equal(t, "/api/records/view/:bykey/:value", query.Url)
	require.Equal(t, "id", query.Params["bykey"])
	require.Equal(t, "{id}", query.Params["value"])
}

func TestRecordChildRouteUsesFirstAllowedSelector(t *testing.T) {
	mod := &BaseModule{Name: "records", Path: "/api", PrimaryKey: pg.IntegerColumn("id")}
	by := []pg.Column{pg.StringColumn("nick")}
	query := standardRecordActionRouteQuery(mod, actions.UpdateModuleAction{By: by}, by)

	require.NotNil(t, query)
	require.Equal(t, "/api/records/:bykey/:value", query.Url)
	require.Equal(t, "nick", query.Params["bykey"])
	require.Equal(t, "{id}", query.Params["value"])
}
