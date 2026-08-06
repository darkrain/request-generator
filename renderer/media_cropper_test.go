package renderer

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func validMediaCropper() *MediaCropperConfig {
	return &MediaCropperConfig{
		Title:        "cropper.title",
		Subtitle:     "cropper.subtitle",
		Hint:         "cropper.hint",
		ChooseLabel:  "cropper.choose",
		CancelLabel:  "cropper.cancel",
		ConfirmLabel: "cropper.confirm",
		CloseLabel:   "cropper.close",
		Accept:       "image/jpeg,image/png,image/webp",
		Viewport: MediaCropperViewportConfig{
			Shape:       MediaCropperViewportCircle,
			AspectRatio: 1,
		},
		Output: MediaCropperOutputConfig{
			Width:    512,
			Height:   512,
			MIMEType: MediaCropperOutputMIMETypeJPEG,
			Quality:  0.92,
		},
	}
}

func TestMediaCropperConfigValidate(t *testing.T) {
	tests := []struct {
		name   string
		mutate func(*MediaCropperConfig)
		err    string
	}{
		{name: "valid"},
		{name: "unsupported shape", mutate: func(config *MediaCropperConfig) { config.Viewport.Shape = "oval" }, err: `renderer.MediaCropperConfig: unsupported viewport shape "oval"`},
		{name: "nonpositive aspect ratio", mutate: func(config *MediaCropperConfig) { config.Viewport.AspectRatio = 0 }, err: "renderer.MediaCropperConfig: viewport aspect ratio must be positive"},
		{name: "nonpositive dimensions", mutate: func(config *MediaCropperConfig) { config.Output.Width = 0 }, err: "renderer.MediaCropperConfig: output dimensions must be positive"},
		{name: "missing title", mutate: func(config *MediaCropperConfig) { config.Title = " " }, err: "renderer.MediaCropperConfig: title is required"},
		{name: "missing confirm label", mutate: func(config *MediaCropperConfig) { config.ConfirmLabel = "" }, err: "renderer.MediaCropperConfig: confirm label is required"},
		{name: "unsupported mime type", mutate: func(config *MediaCropperConfig) { config.Output.MIMEType = "image/avif" }, err: `renderer.MediaCropperConfig: unsupported output mime type "image/avif"`},
		{name: "invalid quality", mutate: func(config *MediaCropperConfig) { config.Output.Quality = 1.01 }, err: "renderer.MediaCropperConfig: output quality must be between 0 and 1"},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			cropper := validMediaCropper()
			if test.mutate != nil {
				test.mutate(cropper)
			}
			if test.err == "" {
				require.NoError(t, cropper.Validate())
				return
			}
			require.EqualError(t, cropper.Validate(), test.err)
		})
	}
}

func TestLocalizeFieldMediaCropperClonesAndLocalizes(t *testing.T) {
	source := &FieldMediaConfig{Cropper: validMediaCropper()}
	translations := map[string]string{
		"cropper.title":    "Adjust image",
		"cropper.subtitle": "Move and scale the image",
		"cropper.hint":     "Drag or pinch to zoom",
		"cropper.choose":   "Choose image",
		"cropper.cancel":   "Cancel",
		"cropper.confirm":  "Use image",
		"cropper.close":    "Close",
	}

	localized := LocalizeFieldMedia(source, func(value, _ string) string {
		if translated, ok := translations[value]; ok {
			return translated
		}
		return value
	})

	require.Equal(t, "Adjust image", localized.Cropper.Title)
	require.Equal(t, "Use image", localized.Cropper.ConfirmLabel)
	require.Equal(t, "cropper.title", source.Cropper.Title)
	require.Equal(t, "cropper.confirm", source.Cropper.ConfirmLabel)
	require.Equal(t, MediaCropperViewportCircle, localized.Cropper.Viewport.Shape)
	require.Equal(t, 512, localized.Cropper.Output.Width)
}
