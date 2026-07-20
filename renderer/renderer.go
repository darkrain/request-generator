package renderer

import "fmt"

type Universal struct {
	List         *ListPage         `json:"list_page,omitempty"`
	Form         *FormPage         `json:"form_page,omitempty"`
	Record       *RecordPage       `json:"record_page,omitempty"`
	ResourceGrid *ResourceGridPage `json:"resource_grid_page,omitempty"`
}

func (r Universal) Identity() *Identity {
	if r.IsZero() {
		return nil
	}
	identity := UniversalIdentity()
	return &identity
}

func (r Universal) ListIdentity() *Identity {
	if r.List == nil && r.ResourceGrid == nil {
		return nil
	}
	return r.Identity()
}

func (r Universal) FormIdentity() *Identity {
	if r.Form == nil {
		return nil
	}
	return r.Identity()
}

func (r Universal) RecordIdentity() *Identity {
	if r.Record == nil {
		return nil
	}
	return r.Identity()
}

func (r Universal) ListRoutePageType() PageType {
	if r.List != nil {
		return PageTypeList
	}
	if r.ResourceGrid != nil {
		return PageTypeResourceGrid
	}
	return ""
}

func (r Universal) FormRoutePageType() PageType {
	if r.Form != nil {
		return PageTypeForm
	}
	return ""
}

func (r Universal) RecordRoutePageType() PageType {
	if r.Record != nil {
		return PageTypeRecord
	}
	return ""
}

func (r Universal) IsZero() bool {
	return r.List == nil && r.Form == nil && r.Record == nil && r.ResourceGrid == nil
}

func (r Universal) Validate() error {
	if r.List != nil && r.ResourceGrid != nil {
		return fmt.Errorf("renderer.Universal: List and ResourceGrid are mutually exclusive for one list route")
	}
	return nil
}

type Layout struct {
	Type     LayoutType   `json:"type,omitempty"`
	Align    AlignToken   `json:"align,omitempty"`
	MaxWidth MaxWidth     `json:"max_width,omitempty"`
	Gap      SpacingToken `json:"gap,omitempty"`
}

type Filters struct {
	Renderer         string                 `json:"renderer,omitempty"`
	Enabled          bool                   `json:"enabled"`
	PrimaryPlacement string                 `json:"primary_placement,omitempty"`
	Primary          []string               `json:"primary,omitempty"`
	Secondary        []string               `json:"secondary,omitempty"`
	More             []string               `json:"more,omitempty"`
	Nested           []string               `json:"nested,omitempty"`
	Reset            *FilterReset           `json:"reset,omitempty"`
	Extra            map[string]interface{} `json:"extra,omitempty"`
}

type FilterReset struct {
	Preserve []string `json:"preserve,omitempty"`
}

type Grid struct {
	Enabled bool     `json:"enabled"`
	Mode    GridMode `json:"mode,omitempty"`
}

type Pagination struct {
	Renderer string         `json:"renderer,omitempty"`
	Mode     PaginationMode `json:"mode,omitempty"`
}

type ListPage struct {
	ID         string                 `json:"id,omitempty"`
	Title      string                 `json:"title,omitempty"`
	Subtitle   string                 `json:"subtitle,omitempty"`
	ShowHeader *bool                  `json:"show_header,omitempty"`
	Layout     *Layout                `json:"layout,omitempty"`
	Filters    *Filters               `json:"filters,omitempty"`
	Summary    *Summary               `json:"summary,omitempty"`
	Grid       *Grid                  `json:"grid,omitempty"`
	Pagination *Pagination            `json:"pagination,omitempty"`
	CardSchema *CardSchema            `json:"card_schema,omitempty"`
	Context    map[string]interface{} `json:"context,omitempty"`
	Actions    []Action               `json:"actions,omitempty"`
}

type Summary struct {
	Title         string `json:"title,omitempty"`
	TitleFallback string `json:"title_fallback,omitempty"`
}

type CardSchema struct {
	Type           string                 `json:"type,omitempty"`
	Variant        CardVariant            `json:"variant,omitempty"`
	Size           SizeToken              `json:"size,omitempty"`
	SurfaceVariant SurfaceVariant         `json:"surface_variant,omitempty"`
	SurfaceEffect  SurfaceEffect          `json:"surface_effect,omitempty"`
	BadgeSize      SizeToken              `json:"badge_size,omitempty"`
	ActionSize     SizeToken              `json:"action_size,omitempty"`
	PrimaryAction  string                 `json:"primary_action,omitempty"`
	Media          *Media                 `json:"media,omitempty"`
	Title          *TextBinding           `json:"title,omitempty"`
	Subtitle       *TextBinding           `json:"subtitle,omitempty"`
	Description    *TextBinding           `json:"description,omitempty"`
	Status         *StatusBinding         `json:"status,omitempty"`
	Badges         []Badge                `json:"badges,omitempty"`
	Stats          []Stat                 `json:"stats,omitempty"`
	Actions        []Action               `json:"actions,omitempty"`
	Extra          map[string]interface{} `json:"extra,omitempty"`
}

type Media struct {
	Field    string                 `json:"field,omitempty"`
	Ratio    MediaRatio             `json:"ratio,omitempty"`
	Size     MediaSize              `json:"size,omitempty"`
	Fallback string                 `json:"fallback,omitempty"`
	Extra    map[string]interface{} `json:"extra,omitempty"`
}

type TextBinding struct {
	Field    string `json:"field,omitempty"`
	Template string `json:"template,omitempty"`
}

type StatusBinding struct {
	ID    string `json:"id,omitempty"`
	Field string `json:"field,omitempty"`
	Type  string `json:"type,omitempty"`
}

type Badge struct {
	ID    string `json:"id,omitempty"`
	Field string `json:"field,omitempty"`
	Tone  string `json:"tone,omitempty"`
}

type Stat struct {
	ID    string       `json:"id,omitempty"`
	Label string       `json:"label,omitempty"`
	Field string       `json:"field,omitempty"`
	Icon  string       `json:"icon,omitempty"`
	Size  SizeToken    `json:"size,omitempty"`
	Tone  string       `json:"tone,omitempty"`
	Value *TextBinding `json:"value,omitempty"`
}

type FormPage struct {
	ID       string                 `json:"id,omitempty"`
	Title    string                 `json:"title,omitempty"`
	Subtitle string                 `json:"subtitle,omitempty"`
	Layout   LayoutType             `json:"layout,omitempty"`
	Actions  []Action               `json:"actions,omitempty"`
	Sections []FormSection          `json:"sections,omitempty"`
	Fields   []string               `json:"fields,omitempty"`
	Context  map[string]interface{} `json:"context,omitempty"`
}

type FormSection struct {
	ID         string                 `json:"id,omitempty"`
	Title      string                 `json:"title,omitempty"`
	PanelTitle string                 `json:"panel_title,omitempty"`
	Subtitle   string                 `json:"subtitle,omitempty"`
	Renderer   string                 `json:"renderer,omitempty"`
	Group      string                 `json:"group,omitempty"`
	GroupTitle string                 `json:"group_title,omitempty"`
	Icon       string                 `json:"icon,omitempty"`
	Block      *Block                 `json:"block,omitempty"`
	Fields     []string               `json:"fields,omitempty"`
	Extra      map[string]interface{} `json:"extra,omitempty"`
}

type Block struct {
	Type    BlockType              `json:"type,omitempty"`
	Variant BlockVariant           `json:"variant,omitempty"`
	Extra   map[string]interface{} `json:"extra,omitempty"`
}

type RecordPage struct {
	Layout      *Layout                `json:"layout,omitempty"`
	Sections    []RecordSection        `json:"sections,omitempty"`
	DisplayData map[string]interface{} `json:"display_data,omitempty"`
	Theme       map[string]interface{} `json:"theme,omitempty"`
	Actions     []Action               `json:"actions,omitempty"`
	Context     map[string]interface{} `json:"context,omitempty"`
}

type RecordSection struct {
	ID         string                 `json:"id,omitempty"`
	Renderer   string                 `json:"renderer,omitempty"`
	LayoutSlot string                 `json:"layout_slot,omitempty"`
	Order      int                    `json:"order,omitempty"`
	Block      *Block                 `json:"block,omitempty"`
	Extra      map[string]interface{} `json:"extra,omitempty"`
}

type ResourceGridPage struct {
	Endpoint string                 `json:"endpoint,omitempty"`
	List     map[string]interface{} `json:"list,omitempty"`
	Create   *Action                `json:"create,omitempty"`
	Delete   *Action                `json:"delete,omitempty"`
	Update   *Action                `json:"update,omitempty"`
	Card     *CardSchema            `json:"card,omitempty"`
	Status   map[string]interface{} `json:"status,omitempty"`
	Context  map[string]interface{} `json:"context,omitempty"`
}

type Action struct {
	ID           string                 `json:"id,omitempty"`
	Type         ActionType             `json:"type,omitempty"`
	LabelKey     string                 `json:"label_key,omitempty"`
	Icon         string                 `json:"icon,omitempty"`
	Variant      ActionVariant          `json:"variant,omitempty"`
	Appearance   ActionAppearance       `json:"appearance,omitempty"`
	VisibleIf    *Condition             `json:"visible_if,omitempty"`
	HiddenIf     *Condition             `json:"hidden_if,omitempty"`
	DisabledIf   *Condition             `json:"disabled_if,omitempty"`
	Route        *RouteAction           `json:"route,omitempty"`
	API          *APIAction             `json:"api,omitempty"`
	Modal        *ModalAction           `json:"modal,omitempty"`
	Confirm      *Confirm               `json:"confirm,omitempty"`
	AfterSuccess *ActionResult          `json:"after_success,omitempty"`
	AfterError   *ActionResult          `json:"after_error,omitempty"`
	AriaLabelKey string                 `json:"aria_label_key,omitempty"`
	TitleKey     string                 `json:"title_key,omitempty"`
	Test         string                 `json:"test,omitempty"`
	Extra        map[string]interface{} `json:"extra,omitempty"`
}

type RouteAction struct {
	Path   string                 `json:"path,omitempty"`
	Params map[string]string      `json:"params,omitempty"`
	Query  map[string]interface{} `json:"query,omitempty"`
}

type APIAction struct {
	Method   string                 `json:"method,omitempty"`
	Endpoint string                 `json:"endpoint,omitempty"`
	Params   map[string]string      `json:"params,omitempty"`
	Query    map[string]interface{} `json:"query,omitempty"`
	Payload  map[string]interface{} `json:"payload,omitempty"`
}

type ModalAction struct {
	Renderer string                 `json:"renderer,omitempty"`
	Title    string                 `json:"title,omitempty"`
	Data     map[string]interface{} `json:"data,omitempty"`
}

type Confirm struct {
	Title        string `json:"title,omitempty"`
	Message      string `json:"message,omitempty"`
	ConfirmLabel string `json:"confirm_label,omitempty"`
}

type ActionResult struct {
	Reload string `json:"reload,omitempty"`
	Toast  string `json:"toast,omitempty"`
	Route  string `json:"route,omitempty"`
	Emit   string `json:"emit,omitempty"`
}

type Condition struct {
	Path      string        `json:"path,omitempty"`
	Equals    interface{}   `json:"equals,omitempty"`
	NotEquals interface{}   `json:"not_equals,omitempty"`
	In        []interface{} `json:"in,omitempty"`
	NotIn     []interface{} `json:"not_in,omitempty"`
	Empty     *bool         `json:"empty,omitempty"`
	NotEmpty  *bool         `json:"not_empty,omitempty"`
	Truthy    *bool         `json:"truthy,omitempty"`
	Falsy     *bool         `json:"falsy,omitempty"`
	All       []Condition   `json:"all,omitempty"`
	Any       []Condition   `json:"any,omitempty"`
	Not       *Condition    `json:"not,omitempty"`
}
