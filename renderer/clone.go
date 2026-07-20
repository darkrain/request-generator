package renderer

func (r Universal) Clone() Universal {
	return Universal{
		List:         cloneListPage(r.List),
		Form:         cloneFormPage(r.Form),
		Record:       cloneRecordPage(r.Record),
		ResourceGrid: cloneResourceGridPage(r.ResourceGrid),
	}
}

func cloneListPage(v *ListPage) *ListPage {
	if v == nil {
		return nil
	}
	cp := *v
	cp.ShowHeader = clonePtr(v.ShowHeader)
	cp.Layout = cloneLayout(v.Layout)
	cp.Filters = cloneFilters(v.Filters)
	cp.Summary = cloneSummary(v.Summary)
	cp.Grid = cloneGrid(v.Grid)
	cp.Pagination = clonePagination(v.Pagination)
	cp.CardSchema = cloneCardSchema(v.CardSchema)
	cp.Context = cloneMap(v.Context)
	cp.Actions = cloneActions(v.Actions)
	return &cp
}

func cloneFormPage(v *FormPage) *FormPage {
	if v == nil {
		return nil
	}
	cp := *v
	cp.Actions = cloneActions(v.Actions)
	cp.Sections = cloneFormSections(v.Sections)
	cp.Fields = cloneSlice(v.Fields)
	cp.Context = cloneMap(v.Context)
	return &cp
}

func cloneRecordPage(v *RecordPage) *RecordPage {
	if v == nil {
		return nil
	}
	cp := *v
	cp.Layout = cloneLayout(v.Layout)
	cp.Sections = cloneRecordSections(v.Sections)
	cp.DisplayData = cloneMap(v.DisplayData)
	cp.Theme = cloneMap(v.Theme)
	cp.Actions = cloneActions(v.Actions)
	cp.Context = cloneMap(v.Context)
	return &cp
}

func cloneResourceGridPage(v *ResourceGridPage) *ResourceGridPage {
	if v == nil {
		return nil
	}
	cp := *v
	cp.List = cloneMap(v.List)
	cp.Create = cloneAction(v.Create)
	cp.Delete = cloneAction(v.Delete)
	cp.Update = cloneAction(v.Update)
	cp.Card = cloneCardSchema(v.Card)
	cp.Status = cloneMap(v.Status)
	cp.Context = cloneMap(v.Context)
	return &cp
}

func cloneLayout(v *Layout) *Layout {
	if v == nil {
		return nil
	}
	cp := *v
	return &cp
}

func cloneFilters(v *Filters) *Filters {
	if v == nil {
		return nil
	}
	cp := *v
	cp.Primary = cloneSlice(v.Primary)
	cp.Secondary = cloneSlice(v.Secondary)
	cp.More = cloneSlice(v.More)
	cp.Nested = cloneSlice(v.Nested)
	cp.Reset = cloneFilterReset(v.Reset)
	cp.Extra = cloneMap(v.Extra)
	return &cp
}

func cloneFilterReset(v *FilterReset) *FilterReset {
	if v == nil {
		return nil
	}
	cp := *v
	cp.Preserve = cloneSlice(v.Preserve)
	return &cp
}

func cloneGrid(v *Grid) *Grid {
	if v == nil {
		return nil
	}
	cp := *v
	return &cp
}

func clonePagination(v *Pagination) *Pagination {
	if v == nil {
		return nil
	}
	cp := *v
	return &cp
}

func cloneSummary(v *Summary) *Summary {
	if v == nil {
		return nil
	}
	cp := *v
	return &cp
}

func cloneCardSchema(v *CardSchema) *CardSchema {
	if v == nil {
		return nil
	}
	cp := *v
	cp.Media = cloneMedia(v.Media)
	cp.Title = cloneTextBinding(v.Title)
	cp.Subtitle = cloneTextBinding(v.Subtitle)
	cp.Description = cloneTextBinding(v.Description)
	cp.Status = cloneStatusBinding(v.Status)
	cp.Badges = cloneSlice(v.Badges)
	cp.Stats = cloneStats(v.Stats)
	cp.Actions = cloneActions(v.Actions)
	cp.Extra = cloneMap(v.Extra)
	return &cp
}

func cloneMedia(v *Media) *Media {
	if v == nil {
		return nil
	}
	cp := *v
	cp.Extra = cloneMap(v.Extra)
	return &cp
}

func cloneTextBinding(v *TextBinding) *TextBinding {
	if v == nil {
		return nil
	}
	cp := *v
	return &cp
}

func cloneStatusBinding(v *StatusBinding) *StatusBinding {
	if v == nil {
		return nil
	}
	cp := *v
	return &cp
}

func cloneStats(values []Stat) []Stat {
	if values == nil {
		return nil
	}
	out := make([]Stat, len(values))
	for i, v := range values {
		out[i] = v
		out[i].Value = cloneTextBinding(v.Value)
	}
	return out
}

func cloneFormSections(values []FormSection) []FormSection {
	if values == nil {
		return nil
	}
	out := make([]FormSection, len(values))
	for i, v := range values {
		out[i] = v
		out[i].Block = cloneBlock(v.Block)
		out[i].Fields = cloneSlice(v.Fields)
		out[i].Extra = cloneMap(v.Extra)
	}
	return out
}

func cloneRecordSections(values []RecordSection) []RecordSection {
	if values == nil {
		return nil
	}
	out := make([]RecordSection, len(values))
	for i, v := range values {
		out[i] = v
		out[i].Block = cloneBlock(v.Block)
		out[i].Extra = cloneMap(v.Extra)
	}
	return out
}

func cloneBlock(v *Block) *Block {
	if v == nil {
		return nil
	}
	cp := *v
	cp.Extra = cloneMap(v.Extra)
	return &cp
}

func cloneActions(values []Action) []Action {
	if values == nil {
		return nil
	}
	out := make([]Action, len(values))
	for i := range values {
		out[i] = cloneActionValue(values[i])
	}
	return out
}

func cloneAction(v *Action) *Action {
	if v == nil {
		return nil
	}
	cp := cloneActionValue(*v)
	return &cp
}

func cloneActionValue(v Action) Action {
	v.VisibleIf = cloneCondition(v.VisibleIf)
	v.HiddenIf = cloneCondition(v.HiddenIf)
	v.DisabledIf = cloneCondition(v.DisabledIf)
	v.Route = cloneRouteAction(v.Route)
	v.API = cloneAPIAction(v.API)
	v.Modal = cloneModalAction(v.Modal)
	v.Confirm = cloneConfirm(v.Confirm)
	v.AfterSuccess = cloneActionResult(v.AfterSuccess)
	v.AfterError = cloneActionResult(v.AfterError)
	v.Extra = cloneMap(v.Extra)
	return v
}

func cloneRouteAction(v *RouteAction) *RouteAction {
	if v == nil {
		return nil
	}
	cp := *v
	cp.Params = cloneMap(v.Params)
	cp.Query = cloneMap(v.Query)
	return &cp
}

func cloneAPIAction(v *APIAction) *APIAction {
	if v == nil {
		return nil
	}
	cp := *v
	cp.Params = cloneMap(v.Params)
	cp.Query = cloneMap(v.Query)
	cp.Payload = cloneMap(v.Payload)
	return &cp
}

func cloneModalAction(v *ModalAction) *ModalAction {
	if v == nil {
		return nil
	}
	cp := *v
	cp.Data = cloneMap(v.Data)
	return &cp
}

func cloneConfirm(v *Confirm) *Confirm {
	if v == nil {
		return nil
	}
	cp := *v
	return &cp
}

func cloneActionResult(v *ActionResult) *ActionResult {
	if v == nil {
		return nil
	}
	cp := *v
	return &cp
}

func cloneCondition(v *Condition) *Condition {
	if v == nil {
		return nil
	}
	cp := *v
	cp.In = cloneSlice(v.In)
	cp.NotIn = cloneSlice(v.NotIn)
	cp.Empty = clonePtr(v.Empty)
	cp.NotEmpty = clonePtr(v.NotEmpty)
	cp.Truthy = clonePtr(v.Truthy)
	cp.Falsy = clonePtr(v.Falsy)
	cp.All = cloneConditions(v.All)
	cp.Any = cloneConditions(v.Any)
	cp.Not = cloneCondition(v.Not)
	return &cp
}

func cloneConditions(values []Condition) []Condition {
	if values == nil {
		return nil
	}
	out := make([]Condition, len(values))
	for i := range values {
		out[i] = *cloneCondition(&values[i])
	}
	return out
}

func clonePtr[T any](v *T) *T {
	if v == nil {
		return nil
	}
	cp := *v
	return &cp
}

func cloneSlice[T any](values []T) []T {
	if values == nil {
		return nil
	}
	out := make([]T, len(values))
	for i, v := range values {
		out[i] = cloneValue(v)
	}
	return out
}

func cloneMap[K comparable, V any](values map[K]V) map[K]V {
	if values == nil {
		return nil
	}
	out := make(map[K]V, len(values))
	for k, v := range values {
		out[k] = cloneValue(v)
	}
	return out
}

func cloneValue[T any](value T) T {
	cloned := cloneInterface(any(value))
	typed, ok := cloned.(T)
	if !ok {
		return value
	}
	return typed
}

func cloneInterface(value interface{}) interface{} {
	switch v := value.(type) {
	case map[string]interface{}:
		return cloneMap(v)
	case []interface{}:
		return cloneSlice(v)
	case map[string]string:
		return cloneMap(v)
	case []string:
		return cloneSlice(v)
	case map[string][]string:
		return cloneMapStringSlice(v)
	case map[string]bool:
		return cloneMap(v)
	case []bool:
		return cloneSlice(v)
	case map[string]int:
		return cloneMap(v)
	case []int:
		return cloneSlice(v)
	case map[string]int64:
		return cloneMap(v)
	case []int64:
		return cloneSlice(v)
	case map[string]float64:
		return cloneMap(v)
	case []float64:
		return cloneSlice(v)
	default:
		return value
	}
}

func cloneMapStringSlice(values map[string][]string) map[string][]string {
	if values == nil {
		return nil
	}
	out := make(map[string][]string, len(values))
	for k, v := range values {
		out[k] = cloneSlice(v)
	}
	return out
}
