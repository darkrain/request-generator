package module

import (
	"testing"

	"github.com/darkrain/request-generator/locale"
	"github.com/darkrain/request-generator/renderer"
	"github.com/stretchr/testify/require"
)

func TestLocalizeRenderer_LocalizesPublicTextOnly(t *testing.T) {
	generator := &Generator{translations: map[locale.Lang]map[string]string{
		locale.RU: {
			"list.title":         "Список",
			"list.subtitle":      "Описание списка",
			"pill.all":           "Все",
			"filter.search":      "Поиск",
			"filter.reset":       "Сбросить фильтр",
			"filter.reset_all":   "Сбросить всё",
			"filter.apply":       "Применить",
			"filter.loading":     "Загрузка",
			"filter.empty":       "Нет данных",
			"filter.no_results":  "Нет результатов",
			"filter.cancel":      "Отмена",
			"filter.close":       "Закрыть",
			"filter.price":       "Цена",
			"summary.title":      "Результаты",
			"badge.verified":     "Проверен",
			"action.open":        "Открыть",
			"action.open.aria":   "Открыть карточку",
			"action.open.title":  "Открыть профиль",
			"action.saving":      "Сохранение",
			"action.saved":       "Сохранено",
			"form.title":         "Настройки",
			"section.title":      "Тарифы",
			"matrix.duration":    "Длительность",
			"matrix.none":        "Недоступно",
			"collection.loading": "Загрузка",
			"bucket.title":       "Активные",
			"modal.search":       "Поиск",
			"media.upload":       "Загрузить файл",
			"media.empty":        "Нет файлов",
			"record.title":       "Профиль",
			"record.section":     "О пользователе",
			"component.value":    "Значение",
			"resource.create":    "Создать",
			"resource.empty":     "Пусто",
			"confirm.title":      "Подтвердить",
			"confirm.message":    "Продолжить действие?",
			"confirm.label":      "Подтвердить",
			"toast.success":      "Сохранено",
		},
	}}

	render := renderer.Universal{
		List: &renderer.ListPage{
			ID:       "profiles",
			Title:    "list.title",
			Subtitle: "list.subtitle",
			Filters: &renderer.Filters{PillRows: [][]renderer.FilterPill{{
				{Label: "All", LabelKey: "pill.all", Key: "status", Val: "all"},
			}}, Groups: []renderer.FilterGroup{{
				ID: "price", Label: "Price", LabelKey: "filter.price", Placement: renderer.FilterGroupPlacementPrimary, Fields: []string{"price"},
			}}, Text: &renderer.FilterText{
				SearchPlaceholder: "filter.search",
				ResetLabel:        "filter.reset",
				ResetAllLabel:     "filter.reset_all",
				ApplyLabel:        "filter.apply",
				LoadingLabel:      "filter.loading",
				EmptyLabel:        "filter.empty",
				NoResultsLabel:    "filter.no_results",
				CancelLabel:       "filter.cancel",
				CloseLabel:        "filter.close",
			}},
			Summary: &renderer.Summary{Title: "summary.title"},
			CardSchema: &renderer.CardSchema{
				Badges: []renderer.Badge{{ID: "verification", Label: "Verified", LabelKey: "badge.verified", Field: "verify_status"}},
				Actions: []renderer.Action{{
					ID:           "open",
					Label:        "Open",
					LabelKey:     "action.open",
					AriaLabelKey: "action.open.aria",
					TitleKey:     "action.open.title",
					SavingLabel:  "action.saving",
					SavedLabel:   "action.saved",
					Endpoint:     "/api/profiles/view/id/1",
					Confirm:      &renderer.Confirm{Title: "confirm.title", Message: "confirm.message", ConfirmLabel: "confirm.label"},
					AfterSuccess: &renderer.ActionResult{Toast: "toast.success"},
				}},
			},
		},
		Form: &renderer.FormPage{
			ID:    "settings",
			Title: "form.title",
			Sections: []renderer.FormSection{{
				ID:       "rates",
				Title:    "section.title",
				Renderer: renderer.RendererFieldMatrix,
				Matrix: &renderer.FieldMatrix{Type: renderer.FieldMatrixTypeTable, Table: &renderer.FieldMatrixTable{
					Heads: []string{"matrix.duration", "Incall"},
					Rows:  []renderer.FieldMatrixRow{{Label: "1 hour", Cells: []renderer.FieldMatrixCell{{Text: "matrix.none"}}}},
				}},
				Collection: &renderer.CollectionConfig{
					Module:       "services",
					LoadingLabel: "collection.loading",
					Buckets:      []renderer.CollectionBucket{{ID: "active", Title: "bucket.title"}},
					Modal:        &renderer.CollectionModal{SearchPlaceholder: "modal.search"},
				},
				MediaUpload: &renderer.MediaUploadConfig{Title: "media.upload"},
				MediaItems:  []renderer.MediaGalleryItem{{ID: "photo-1", Title: "User supplied title"}},
				MediaLabels: &renderer.MediaGalleryLabels{Empty: "media.empty"},
			}},
		},
		Record: &renderer.RecordPage{
			ID:    "profile",
			Title: "record.title",
			Badge: "badge.verified",
			Sections: []renderer.RecordSection{{
				ID:    "about",
				Title: "record.section",
				Components: []renderer.DisplayComponent{{
					ID:         "details",
					ValueLabel: "component.value",
				}},
			}},
		},
		ResourceGrid: &renderer.ResourceGridPage{
			Endpoint: "/api/profiles",
			Create:   &renderer.Action{ID: "create", Label: "resource.create"},
			Text:     map[string]string{"empty": "resource.empty"},
		},
	}

	localized := generator.localizeRenderer(locale.RU, render)

	require.Equal(t, "Список", localized.List.Title)
	require.Equal(t, "Все", localized.List.Filters.PillRows[0][0].Label)
	require.Empty(t, localized.List.Filters.PillRows[0][0].LabelKey)
	require.Equal(t, "Поиск", localized.List.Filters.Text.SearchPlaceholder)
	require.Equal(t, "Сбросить фильтр", localized.List.Filters.Text.ResetLabel)
	require.Equal(t, "Сбросить всё", localized.List.Filters.Text.ResetAllLabel)
	require.Equal(t, "Применить", localized.List.Filters.Text.ApplyLabel)
	require.Equal(t, "Загрузка", localized.List.Filters.Text.LoadingLabel)
	require.Equal(t, "Нет данных", localized.List.Filters.Text.EmptyLabel)
	require.Equal(t, "Нет результатов", localized.List.Filters.Text.NoResultsLabel)
	require.Equal(t, "Отмена", localized.List.Filters.Text.CancelLabel)
	require.Equal(t, "Закрыть", localized.List.Filters.Text.CloseLabel)
	require.Equal(t, "Цена", localized.List.Filters.Groups[0].Label)
	require.Empty(t, localized.List.Filters.Groups[0].LabelKey)
	require.Equal(t, "Проверен", localized.List.CardSchema.Badges[0].Label)
	require.Empty(t, localized.List.CardSchema.Badges[0].LabelKey)
	action := localized.List.CardSchema.Actions[0]
	require.Equal(t, "Открыть", action.Label)
	require.Equal(t, "Открыть карточку", action.AriaLabel)
	require.Equal(t, "Открыть профиль", action.Title)
	require.Equal(t, "Сохранение", action.SavingLabel)
	require.Equal(t, "Сохранено", action.SavedLabel)
	require.Empty(t, action.LabelKey)
	require.Empty(t, action.AriaLabelKey)
	require.Empty(t, action.TitleKey)
	require.Equal(t, "/api/profiles/view/id/1", action.Endpoint)
	require.Equal(t, "Подтвердить", action.Confirm.Title)
	require.Equal(t, "Продолжить действие?", action.Confirm.Message)
	require.Equal(t, "Сохранено", action.AfterSuccess.Toast)

	section := localized.Form.Sections[0]
	require.Equal(t, "Настройки", localized.Form.Title)
	require.Equal(t, "Тарифы", section.Title)
	require.Equal(t, "Длительность", section.Matrix.Table.Heads[0])
	require.Equal(t, "Недоступно", section.Matrix.Table.Rows[0].Cells[0].Text)
	require.Equal(t, "Загрузка", section.Collection.LoadingLabel)
	require.Equal(t, "Активные", section.Collection.Buckets[0].Title)
	require.Equal(t, "Поиск", section.Collection.Modal.SearchPlaceholder)
	require.Equal(t, "Загрузить файл", section.MediaUpload.Title)
	require.Equal(t, "Нет файлов", section.MediaLabels.Empty)
	require.Equal(t, "User supplied title", section.MediaItems[0].Title)

	require.Equal(t, "Профиль", localized.Record.Title)
	require.Equal(t, "Проверен", localized.Record.Badge)
	require.Equal(t, "О пользователе", localized.Record.Sections[0].Title)
	require.Equal(t, "Значение", localized.Record.Sections[0].Components[0].ValueLabel)
	require.Equal(t, "Создать", localized.ResourceGrid.Create.Label)
	require.Equal(t, "Пусто", localized.ResourceGrid.Text["empty"])
}
