package renderer

type TextResolver func(value string, key string) string

type textLocalizer struct {
	resolve TextResolver
}

func (localizer textLocalizer) localizeRendererText(value, key string) string {
	if key != "" {
		return localizer.resolve(value, key)
	}
	return localizer.resolve(value, "")
}

func (localizer textLocalizer) localizeTextFields(fields ...*string) {
	for _, field := range fields {
		*field = localizer.localizeRendererText(*field, "")
	}
}

func (localizer textLocalizer) localizeRendererAction(action *Action) {
	if action == nil {
		return
	}
	action.Label = localizer.localizeRendererText(action.Label, action.LabelKey)
	action.LabelKey = ""
	action.AriaLabel = localizer.localizeRendererText(action.AriaLabel, action.AriaLabelKey)
	action.AriaLabelKey = ""
	action.Title = localizer.localizeRendererText(action.Title, action.TitleKey)
	action.TitleKey = ""
	localizer.localizeTextFields(&action.SavingLabel, &action.SavedLabel)
	if action.Modal != nil {
		localizer.localizeTextFields(&action.Modal.Title)
	}
	if action.Confirm != nil {
		localizer.localizeTextFields(&action.Confirm.Title, &action.Confirm.Message, &action.Confirm.ConfirmLabel)
	}
	if action.AfterSuccess != nil {
		localizer.localizeTextFields(&action.AfterSuccess.Toast)
	}
	if action.AfterError != nil {
		localizer.localizeTextFields(&action.AfterError.Toast)
	}
}

func Localize(render Universal, resolve TextResolver) Universal {
	if resolve == nil {
		return render.Clone()
	}
	localized := render.Clone()
	return (textLocalizer{resolve: resolve}).localizeRenderer(localized)
}

func LocalizeFieldMedia(value *FieldMediaConfig, resolve TextResolver) *FieldMediaConfig {
	localized := CloneFieldMediaConfig(value)
	if localized == nil || resolve == nil {
		return localized
	}
	localizer := textLocalizer{resolve: resolve}
	localized.Upload = localizer.localizeMediaUpload(localized.Upload)
	localized.Labels = localizer.localizeMediaLabels(localized.Labels)
	localizer.localizeMediaActions(localized.Actions)
	localizer.localizeMediaCropper(localized.Cropper)
	return localized
}

func (localizer textLocalizer) localizeRenderer(render Universal) Universal {
	if render.List != nil {
		localizer.localizeListPage(render.List)
	}
	if render.Form != nil {
		localizer.localizeFormPage(render.Form)
	}
	if render.Record != nil {
		localizer.localizeRecordPage(render.Record)
	}
	if render.ResourceGrid != nil {
		localizer.localizeResourceGridPage(render.ResourceGrid)
	}
	return render
}

func (localizer textLocalizer) localizeListPage(page *ListPage) {
	localizer.localizeTextFields(&page.Title, &page.Subtitle)
	for i := range page.Actions {
		localizer.localizeRendererAction(&page.Actions[i])
	}
	if page.Filters != nil {
		localizer.localizeFilterPills(page.Filters.PillRows)
		localizer.localizeFilterPills(page.Filters.SecondaryPillRows)
		localizer.localizeFilterText(page.Filters.Text)
	}
	if page.Summary != nil {
		localizer.localizeTextFields(&page.Summary.Title, &page.Summary.TitleFallback)
	}
	if page.CardSchema != nil {
		localizer.localizeCardSchema(page.CardSchema)
	}
}

func (localizer textLocalizer) localizeFilterText(text *FilterText) {
	if text != nil {
		localizer.localizeTextFields(&text.SearchPlaceholder, &text.ResetLabel, &text.ResetAllLabel, &text.ApplyLabel, &text.LoadingLabel, &text.EmptyLabel, &text.NoResultsLabel, &text.CancelLabel, &text.CloseLabel)
	}
}

func (localizer textLocalizer) localizeFilterPills(rows [][]FilterPill) {
	for i := range rows {
		for j := range rows[i] {
			pill := &rows[i][j]
			pill.Label = localizer.localizeRendererText(pill.Label, pill.LabelKey)
			pill.LabelKey = ""
		}
	}
}

func (localizer textLocalizer) localizeCardSchema(schema *CardSchema) {
	for i := range schema.Badges {
		localizer.localizeBadge(&schema.Badges[i])
	}
	for i := range schema.Stats {
		localizer.localizeBadge(&schema.Stats[i])
	}
	for i := range schema.Actions {
		localizer.localizeRendererAction(&schema.Actions[i])
	}
}

func (localizer textLocalizer) localizeBadge(badge *Badge) {
	badge.Label = localizer.localizeRendererText(badge.Label, badge.LabelKey)
	badge.LabelKey = ""
	if badge.Then != nil {
		badge.Then.Label = localizer.localizeRendererText(badge.Then.Label, badge.Then.LabelKey)
		badge.Then.LabelKey = ""
	}
	if badge.Else != nil {
		badge.Else.Label = localizer.localizeRendererText(badge.Else.Label, badge.Else.LabelKey)
		badge.Else.LabelKey = ""
	}
}

func (localizer textLocalizer) localizeFormPage(page *FormPage) {
	localizer.localizeTextFields(&page.Title, &page.Subtitle)
	for i := range page.Actions {
		localizer.localizeRendererAction(&page.Actions[i])
	}
	for i := range page.Sections {
		localizer.localizeFormSection(&page.Sections[i])
	}
}

func (localizer textLocalizer) localizeFormSection(section *FormSection) {
	localizer.localizeTextFields(&section.Title, &section.PanelTitle, &section.Subtitle, &section.GroupTitle)
	localizer.localizeFieldMatrix(section.Matrix)
	if section.ListPage != nil {
		localizer.localizeListPage(section.ListPage)
	}
	localizer.localizeCollection(section.Collection)
	section.MediaUpload = localizer.localizeMediaUpload(section.MediaUpload)
	section.MediaLabels = localizer.localizeMediaLabels(section.MediaLabels)
	localizer.localizeMediaActions(section.MediaActions)
}

func (localizer textLocalizer) localizeFieldMatrix(matrix *FieldMatrix) {
	if matrix == nil || matrix.Table == nil {
		return
	}
	for i := range matrix.Table.Heads {
		matrix.Table.Heads[i] = localizer.localizeRendererText(matrix.Table.Heads[i], "")
	}
	for i := range matrix.Table.Rows {
		row := &matrix.Table.Rows[i]
		localizer.localizeTextFields(&row.Label)
		for j := range row.Cells {
			localizer.localizeTextFields(&row.Cells[j].Text)
		}
	}
}

func (localizer textLocalizer) localizeMediaUpload(upload *MediaUploadConfig) *MediaUploadConfig {
	if upload != nil {
		localizer.localizeTextFields(&upload.Title, &upload.Subtitle, &upload.LoadingTitle)
	}
	return upload
}

func (localizer textLocalizer) localizeMediaLabels(labels *MediaGalleryLabels) *MediaGalleryLabels {
	if labels != nil {
		localizer.localizeTextFields(&labels.Public, &labels.Private, &labels.Empty, &labels.CoverBadge, &labels.Remove, &labels.Reorder, &labels.FirstIsCover, &labels.PrivateHint)
	}
	return labels
}

func (localizer textLocalizer) localizeMediaActions(actions *MediaGalleryActions) {
	if actions != nil {
		localizer.localizeRendererAction(actions.Upload)
		localizer.localizeRendererAction(actions.Link)
		localizer.localizeRendererAction(actions.Reorder)
		localizer.localizeRendererAction(actions.Recenter)
		localizer.localizeRendererAction(actions.Crop)
		localizer.localizeRendererAction(actions.Remove)
	}
}

func (localizer textLocalizer) localizeMediaCropper(cropper *MediaCropperConfig) {
	if cropper == nil {
		return
	}
	localizer.localizeTextFields(&cropper.Title, &cropper.Subtitle, &cropper.Hint, &cropper.ChooseLabel, &cropper.CancelLabel, &cropper.ConfirmLabel, &cropper.CloseLabel)
}

func (localizer textLocalizer) localizeCollection(collection *CollectionConfig) {
	if collection == nil {
		return
	}
	localizer.localizeTextFields(&collection.LoadingLabel)
	for i := range collection.Actions {
		localizer.localizeRendererAction(&collection.Actions[i])
	}
	for i := range collection.Buckets {
		bucket := &collection.Buckets[i]
		localizer.localizeTextFields(&bucket.Title, &bucket.CountLabel, &bucket.AddLabel, &bucket.ClearLabel, &bucket.ModalTitle, &bucket.ModalSubtitle, &bucket.ConfirmLabel)
		for j := range bucket.Actions {
			localizer.localizeRendererAction(&bucket.Actions[j])
		}
	}
	if collection.Modal != nil {
		localizer.localizeTextFields(&collection.Modal.SearchPlaceholder, &collection.Modal.EmptyLabel, &collection.Modal.SelectedLabel, &collection.Modal.TakenLabel, &collection.Modal.CancelLabel, &collection.Modal.ConfirmLoadingLabel)
	}
}

func (localizer textLocalizer) localizeRecordPage(page *RecordPage) {
	localizer.localizeTextFields(&page.Title, &page.Subtitle, &page.Badge)
	for i := range page.Actions {
		localizer.localizeRendererAction(&page.Actions[i])
	}
	for i := range page.Sections {
		section := &page.Sections[i]
		localizer.localizeTextFields(&section.Title, &section.TitleFallback)
		for j := range section.Components {
			component := &section.Components[j]
			localizer.localizeTextFields(&component.ValueLabel, &component.ValueFallback, &component.MatrixLabel, &component.Title, &component.TitleFallback, &component.Subtitle, &component.SubtitleFallback)
		}
	}
}

func (localizer textLocalizer) localizeResourceGridPage(page *ResourceGridPage) {
	localizer.localizeRendererAction(page.Create)
	localizer.localizeRendererAction(page.Delete)
	localizer.localizeRendererAction(page.Update)
	if page.Card != nil {
		localizer.localizeCardSchema(page.Card)
	}
	for key, value := range page.Text {
		page.Text[key] = localizer.localizeRendererText(value, "")
	}
}
