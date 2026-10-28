package renderer

import (
	"encoding/json"
	"math"
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
	source := &FieldMediaConfig{
		Item:    &MediaGalleryItem{Actions: []Action{{ID: "publish", Label: "media.publish", Type: ActionEmit}}},
		Cropper: validMediaCropper(),
	}
	translations := map[string]string{
		"cropper.title":    "Adjust image",
		"cropper.subtitle": "Move and scale the image",
		"cropper.hint":     "Drag or pinch to zoom",
		"cropper.choose":   "Choose image",
		"cropper.cancel":   "Cancel",
		"cropper.confirm":  "Use image",
		"cropper.close":    "Close",
		"media.publish":    "Publish",
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
	require.Equal(t, "Publish", localized.Item.Actions[0].Label)
	require.Equal(t, "media.publish", source.Item.Actions[0].Label)
}

// A frame changes selection semantics, without sharing mutable cropper state.
func TestMediaCropperFrameSurvivesLocalizationAndSerialization(t *testing.T) {
	source := &FieldMediaConfig{Cropper: validMediaCropper()}
	source.Cropper.Frame = true
	localized := LocalizeFieldMedia(source, func(value, _ string) string { return value })
	require.NoError(t, localized.Cropper.Validate())
	require.True(t, localized.Cropper.Frame)
	data, err := json.Marshal(localized.Cropper)
	require.NoError(t, err)
	require.Contains(t, string(data), `"frame":true`)
	localized.Cropper.Frame = false
	require.True(t, source.Cropper.Frame)
	data, err = json.Marshal(localized.Cropper)
	require.NoError(t, err)
	require.NotContains(t, string(data), `"frame"`)
	localized.Cropper.Viewport.AspectRatio = 0
	require.Error(t, localized.Cropper.Validate())
}

func TestGalleryFrameContract(t *testing.T) {
	original := &FieldMediaConfig{Item: &MediaGalleryItem{
		Fit: "crop", Crop: &MediaCrop{X: 0.1, Y: 0.2, W: 0.5, H: 0.5}, FocalPoint: &MediaFocalPoint{X: 0.4, Y: 0.6},
	}}
	require.NoError(t, validateMediaGalleryItems("test", []MediaGalleryItem{*original.Item}))
	clone := LocalizeFieldMedia(original, func(value, _ string) string { return value })
	data, err := json.Marshal(clone.Item)
	require.NoError(t, err)
	require.Contains(t, string(data), `"focal_point":{"x":0.4,"y":0.6}`)
	require.Contains(t, string(data), `"crop":{"x":0.1,"y":0.2,"w":0.5,"h":0.5}`)
	clone.Item.Crop.X = 0.3
	clone.Item.FocalPoint.X = 0.8
	require.Equal(t, 0.1, original.Item.Crop.X)
	require.Equal(t, 0.4, original.Item.FocalPoint.X)
	for _, bad := range []MediaGalleryItem{
		{FocalPoint: &MediaFocalPoint{X: math.NaN()}},
		{FocalPoint: &MediaFocalPoint{Y: 1.1}},
		{Crop: &MediaCrop{W: 0, H: 0.5}},
		{Crop: &MediaCrop{X: 0.8, W: 0.5, H: 0.5}},
	} {
		require.Error(t, validateMediaGalleryItems("test", []MediaGalleryItem{bad}))
	}
	items := cloneMediaGalleryItems([]MediaGalleryItem{*original.Item})
	items[0].Crop.X = 0.7
	require.Equal(t, 0.1, original.Item.Crop.X)
}
