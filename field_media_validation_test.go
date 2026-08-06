package module

import (
	"testing"

	"github.com/darkrain/request-generator/fields"
	"github.com/darkrain/request-generator/renderer"
	"github.com/gin-gonic/gin"
	"github.com/go-jet/jet/v2/postgres"
	"github.com/stretchr/testify/require"
)

func TestRunRejectsInvalidFieldMediaCropper(t *testing.T) {
	gin.SetMode(gin.TestMode)
	id := postgres.IntegerColumn("id")
	avatar := postgres.StringColumn("avatar")
	table := postgres.NewTable("public", "media_items", "", id, avatar)
	base := &BaseModule{
		Name:       "media-items",
		Table:      table,
		PrimaryKey: id,
		Fields: []fields.ModuleField{{
			Column: avatar,
			Media: &renderer.FieldMediaConfig{Cropper: &renderer.MediaCropperConfig{
				Viewport: renderer.MediaCropperViewportConfig{Shape: renderer.MediaCropperViewportCircle, AspectRatio: 1},
				Output:   renderer.MediaCropperOutputConfig{Width: 512, Height: 512, MIMEType: "image/jpeg", Quality: 1.1},
			}},
		}},
	}
	engine := gin.New()
	group := engine.Group("")
	generator := NewGenerator(nil, *group, []*BaseModule{base}, nil, nil)

	require.PanicsWithValue(t,
		`invalid field media config in module media-items: field "avatar": renderer.MediaCropperConfig: output quality must be between 0 and 1`,
		func() { generator.Run() },
	)
}
