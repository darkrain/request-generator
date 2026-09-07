package module

import (
	"testing"

	"github.com/darkrain/request-generator/renderer"
	"github.com/stretchr/testify/require"
)

func TestExplicitListPageTypeKeepsListRendererDiscovery(t *testing.T) {
	render := renderer.Universal{List: &renderer.ListPage{ID: "items-list"}}

	require.NotNil(t, viewRouteIdentity(render, renderer.PageTypeList))
	require.Equal(t, renderer.PageTypeList, viewRoutePageType(render, renderer.PageTypeList))
}

func TestExplicitResourceGridPageTypeKeepsListRendererDiscovery(t *testing.T) {
	render := renderer.Universal{ResourceGrid: &renderer.ResourceGridPage{Endpoint: "/api/items"}}

	require.NotNil(t, viewRouteIdentity(render, renderer.PageTypeResourceGrid))
	require.Equal(t, renderer.PageTypeResourceGrid, viewRoutePageType(render, renderer.PageTypeResourceGrid))
}
