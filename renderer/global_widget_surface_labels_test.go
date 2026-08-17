package renderer

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestLocalizeGlobalWidgetSurfaceLabels(t *testing.T) {
	widget := GlobalWidget{Surface: WidgetSurface{
		Kind:       WidgetSurfaceDrawer,
		Placement:  WidgetPlacementShellEnd,
		LoadPolicy: WidgetLoadOnOpen,
		CloseLabel: "widget.close",
		BackLabel:  "widget.back",
		MoreLabel:  "widget.more",
	}}

	localized := LocalizeGlobalWidget(widget, func(key, _ string) string { return "ru:" + key })

	require.Equal(t, "ru:widget.close", localized.Surface.CloseLabel)
	require.Equal(t, "ru:widget.back", localized.Surface.BackLabel)
	require.Equal(t, "ru:widget.more", localized.Surface.MoreLabel)
}
