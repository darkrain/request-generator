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
	cp.ShowHeader = clonePtr(v.ShowHeader)
	cp.Navigation = cloneRecordNavigation(v.Navigation)
	cp.Layout = cloneLayout(v.Layout)
	cp.Sections = cloneRecordSections(v.Sections)
	cp.DisplayData = cloneRecordDisplayData(v.DisplayData)
	cp.CardSchema = cloneCardSchema(v.CardSchema)
	cp.Theme = cloneRecordTheme(v.Theme)
	cp.Actions = cloneActions(v.Actions)
	cp.Context = cloneMap(v.Context)
	return &cp
}

func cloneRecordNavigation(v *RecordNavigation) *RecordNavigation {
	if v == nil {
		return nil
	}
	cp := *v
	return &cp
}

func cloneRecordTheme(v *RecordTheme) *RecordTheme {
	if v == nil {
		return nil
	}
	cp := *v
	if v.Profile != nil {
		profile := *v.Profile
		cp.Profile = &profile
	}
	return &cp
}

func cloneResourceGridPage(v *ResourceGridPage) *ResourceGridPage {
	if v == nil {
		return nil
	}
	cp := *v
	cp.List = cloneResourceGridListConfig(v.List)
	cp.Create = cloneAction(v.Create)
	cp.Delete = cloneAction(v.Delete)
	cp.Update = cloneAction(v.Update)
	cp.Card = cloneCardSchema(v.Card)
	cp.Status = cloneResourceGridStatusConfig(v.Status)
	cp.Actions = cloneResourceGridActionsConfig(v.Actions)
	cp.Text = cloneMap(v.Text)
	cp.Context = cloneMap(v.Context)
	return &cp
}

func cloneResourceGridListConfig(v *ResourceGridListConfig) *ResourceGridListConfig {
	if v == nil {
		return nil
	}
	cp := *v
	cp.Filters = cloneMap(v.Filters)
	return &cp
}

func cloneResourceGridStatusConfig(v *ResourceGridStatusConfig) *ResourceGridStatusConfig {
	if v == nil {
		return nil
	}
	cp := *v
	cp.DraftValues = cloneSlice(v.DraftValues)
	cp.PendingPayload = cloneInterface(v.PendingPayload)
	return &cp
}

func cloneResourceGridActionsConfig(v *ResourceGridActionsConfig) *ResourceGridActionsConfig {
	if v == nil {
		return nil
	}
	cp := *v
	cp.EditRoute = cloneRouteValue(v.EditRoute)
	return &cp
}

func cloneLayout(v *Layout) *Layout {
	if v == nil {
		return nil
	}
	cp := *v
	cp.Slots = cloneSlice(v.Slots)
	return &cp
}

func cloneFilters(v *Filters) *Filters {
	if v == nil {
		return nil
	}
	cp := *v
	cp.SecondaryEnabled = clonePtr(v.SecondaryEnabled)
	cp.Levels = cloneSlice(v.Levels)
	cp.Primary = cloneSlice(v.Primary)
	cp.Secondary = cloneSlice(v.Secondary)
	cp.More = cloneSlice(v.More)
	cp.Nested = cloneSlice(v.Nested)
	cp.PillRows = cloneMapRows(v.PillRows)
	cp.SecondaryPillRows = cloneMapRows(v.SecondaryPillRows)
	cp.Reset = cloneFilterReset(v.Reset)
	return &cp
}

func cloneMapRows(values [][]FilterPill) [][]FilterPill {
	if values == nil {
		return nil
	}
	out := make([][]FilterPill, len(values))
	for i, row := range values {
		if row == nil {
			continue
		}
		out[i] = make([]FilterPill, len(row))
		for j, item := range row {
			out[i][j] = item
		}
	}
	return out
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
	cp.ShowOnline = clonePtr(v.ShowOnline)
	cp.ShowAction = clonePtr(v.ShowAction)
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
	cp.Badges = cloneBadges(v.Badges)
	cp.Stats = cloneStats(v.Stats)
	cp.Actions = cloneActions(v.Actions)
	return &cp
}

func cloneMedia(v *Media) *Media {
	if v == nil {
		return nil
	}
	cp := *v
	cp.GlowEnabled = clonePtr(v.GlowEnabled)
	return &cp
}

func cloneTextBinding(v *TextBinding) *TextBinding {
	if v == nil {
		return nil
	}
	cp := *v
	return &cp
}

func cloneBadges(values []Badge) []Badge {
	if values == nil {
		return nil
	}
	out := make([]Badge, len(values))
	for i, v := range values {
		out[i] = v
		out[i].Marker = clonePtr(v.Marker)
		out[i].ToneMap = cloneMap(v.ToneMap)
		out[i].Then = cloneBadgeState(v.Then)
		out[i].Else = cloneBadgeState(v.Else)
	}
	return out
}

func cloneBadgeState(v *BadgeState) *BadgeState {
	if v == nil {
		return nil
	}
	cp := *v
	cp.Marker = clonePtr(v.Marker)
	return &cp
}

func cloneStatusBinding(v *StatusBinding) *StatusBinding {
	if v == nil {
		return nil
	}
	cp := *v
	cp.Marker = clonePtr(v.Marker)
	cp.ToneMap = cloneMap(v.ToneMap)
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
		out[i].ListPage = cloneListPage(v.ListPage)
		out[i].Collection = cloneCollectionConfig(v.Collection)
		out[i].Preferences = clonePreferencesConfig(v.Preferences)
	}
	return out
}

func cloneCollectionConfig(v *CollectionConfig) *CollectionConfig {
	if v == nil {
		return nil
	}
	cp := *v
	cp.Collections = cloneSlice(v.Collections)
	cp.Modal = cloneCollectionModal(v.Modal)
	return &cp
}

func cloneCollectionModal(v *CollectionModal) *CollectionModal {
	if v == nil {
		return nil
	}
	cp := *v
	return &cp
}

func clonePreferencesConfig(v *PreferencesConfig) *PreferencesConfig {
	if v == nil {
		return nil
	}
	cp := *v
	cp.Channels = cloneSlice(v.Channels)
	cp.Blocks = cloneSlice(v.Blocks)
	cp.ConnectionPrompts = cloneSlice(v.ConnectionPrompts)
	return &cp
}

func cloneRecordDisplayData(v *RecordDisplayData) *RecordDisplayData {
	if v == nil {
		return nil
	}
	cp := *v
	if v.Gallery != nil {
		gallery := *v.Gallery
		gallery.Items = cloneSlice(v.Gallery.Items)
		gallery.Actions = cloneSlice(v.Gallery.Actions)
		gallery.Overlays = cloneSlice(v.Gallery.Overlays)
		cp.Gallery = &gallery
	}
	if v.Hero != nil {
		hero := *v.Hero
		if v.Hero.Identity != nil {
			identity := *v.Hero.Identity
			if v.Hero.Identity.Avatar != nil {
				avatar := *v.Hero.Identity.Avatar
				identity.Avatar = &avatar
			}
			identity.CornerBadges = cloneSlice(v.Hero.Identity.CornerBadges)
			if v.Hero.Identity.Location != nil {
				location := *v.Hero.Identity.Location
				identity.Location = &location
			}
			hero.Identity = &identity
		}
		hero.Stats = cloneSlice(v.Hero.Stats)
		cp.Hero = &hero
	}
	if v.Details != nil {
		details := *v.Details
		details.Items = cloneSlice(v.Details.Items)
		details.Commercial = cloneSlice(v.Details.Commercial)
		cp.Details = &details
	}
	if v.About != nil {
		about := *v.About
		cp.About = &about
	}
	if v.Meetings != nil {
		meetings := *v.Meetings
		meetings.Items = cloneSlice(v.Meetings.Items)
		meetings.Commission = cloneSlice(v.Meetings.Commission)
		meetings.WorkArea = cloneSlice(v.Meetings.WorkArea)
		meetings.Payments = cloneSlice(v.Meetings.Payments)
		cp.Meetings = &meetings
	}
	if v.Rates != nil {
		rates := *v.Rates
		rates.Items = cloneSlice(v.Rates.Items)
		rates.Groups = cloneSlice(v.Rates.Groups)
		cp.Rates = &rates
	}
	if v.Services != nil {
		services := *v.Services
		services.Included = cloneSlice(v.Services.Included)
		services.Extra = cloneSlice(v.Services.Extra)
		services.Groups = cloneSlice(v.Services.Groups)
		cp.Services = &services
	}
	if v.Contacts != nil {
		contacts := *v.Contacts
		contacts.Items = cloneSlice(v.Contacts.Items)
		cp.Contacts = &contacts
	}
	return &cp
}

func cloneRecordSections(values []RecordSection) []RecordSection {
	if values == nil {
		return nil
	}
	out := make([]RecordSection, len(values))
	for i, v := range values {
		out[i] = v
		out[i].Block = cloneBlock(v.Block)
		out[i].Stack = cloneStack(v.Stack)
		out[i].Components = cloneDisplayComponents(v.Components)
	}
	return out
}

func cloneBlock(v *Block) *Block {
	if v == nil {
		return nil
	}
	cp := *v
	return &cp
}

func cloneStack(v *Stack) *Stack {
	if v == nil {
		return nil
	}
	cp := *v
	if v.Wrap != nil {
		wrap := *v.Wrap
		cp.Wrap = &wrap
	}
	return &cp
}

func cloneDisplayComponents(values []DisplayComponent) []DisplayComponent {
	if values == nil {
		return nil
	}
	out := make([]DisplayComponent, len(values))
	for i, v := range values {
		out[i] = v
		out[i].Block = cloneBlock(v.Block)
		if v.Visible != nil {
			visible := *v.Visible
			out[i].Visible = &visible
		}
		if v.Wrap != nil {
			wrap := *v.Wrap
			out[i].Wrap = &wrap
		}
		if v.VideoControls != nil {
			videoControls := *v.VideoControls
			out[i].VideoControls = &videoControls
		}
		if v.MatrixColumns != nil {
			out[i].MatrixColumns = make([]map[string]interface{}, len(v.MatrixColumns))
			for j, column := range v.MatrixColumns {
				out[i].MatrixColumns[j] = cloneMap(column)
			}
		}
	}
	return out
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
	v.AfterRoute = cloneRouteValue(v.AfterRoute)
	v.Route = cloneRouteValue(v.Route)
	v.API = cloneAPIAction(v.API)
	v.Modal = cloneModalAction(v.Modal)
	v.Confirm = cloneConfirm(v.Confirm)
	v.AfterSuccess = cloneActionResult(v.AfterSuccess)
	v.AfterError = cloneActionResult(v.AfterError)
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

func cloneRouteValue(value interface{}) interface{} {
	switch typed := value.(type) {
	case nil:
		return nil
	case string:
		return typed
	case *RouteAction:
		return cloneRouteAction(typed)
	case RouteAction:
		cloned := cloneRouteAction(&typed)
		if cloned == nil {
			return nil
		}
		return *cloned
	case map[string]interface{}:
		return cloneMap(typed)
	default:
		return typed
	}
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
	cp.Not = cloneConditionValue(v.Not)
	return &cp
}

func cloneConditionValue(value interface{}) interface{} {
	switch typed := value.(type) {
	case nil:
		return nil
	case *Condition:
		return cloneCondition(typed)
	case Condition:
		cloned := cloneCondition(&typed)
		if cloned == nil {
			return nil
		}
		return *cloned
	case map[string]interface{}:
		return cloneMap(typed)
	default:
		return typed
	}
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
