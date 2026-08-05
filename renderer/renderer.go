package renderer

import (
	"encoding/json"
	"fmt"
)

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
	if r.Form != nil {
		for _, section := range r.Form.Sections {
			if section.Collection == nil {
				continue
			}
			if section.Collection.Module == "" {
				return fmt.Errorf("renderer.Universal: collection section %q must define module", section.ID)
			}
			for _, bucket := range section.Collection.Buckets {
				if bucket.Predicate == nil {
					continue
				}
				if bucket.Predicate.Field == "" {
					return fmt.Errorf("renderer.Universal: collection bucket %q predicate must define field", bucket.ID)
				}
				if bucket.Predicate.Operator == "" {
					return fmt.Errorf("renderer.Universal: collection bucket %q predicate must define operator", bucket.ID)
				}
				if len(bucket.Predicate.Values) > 0 && bucket.Predicate.Value != nil {
					return fmt.Errorf("renderer.Universal: collection bucket %q predicate must not define both value and values", bucket.ID)
				}
			}
		}
	}
	return nil
}

type Layout struct {
	Type     LayoutType   `json:"type,omitempty"`
	Mode     string       `json:"mode,omitempty"`
	Slots    []string     `json:"slots,omitempty"`
	Left     SizeToken    `json:"left,omitempty"`
	Center   SizeToken    `json:"center,omitempty"`
	Right    SizeToken    `json:"right,omitempty"`
	Align    AlignToken   `json:"align,omitempty"`
	MaxWidth MaxWidth     `json:"max_width,omitempty"`
	Gap      SpacingToken `json:"gap,omitempty"`
}

type Filters struct {
	Renderer          RendererKey    `json:"renderer,omitempty"`
	Enabled           bool           `json:"enabled"`
	PrimaryPlacement  string         `json:"primary_placement,omitempty"`
	SecondaryEnabled  *bool          `json:"secondary_enabled,omitempty"`
	ResetPlacement    string         `json:"reset_placement,omitempty"`
	Levels            []string       `json:"levels,omitempty"`
	Primary           []string       `json:"primary,omitempty"`
	Secondary         []string       `json:"secondary,omitempty"`
	More              []string       `json:"more,omitempty"`
	Nested            []string       `json:"nested,omitempty"`
	PillRows          [][]FilterPill `json:"pill_rows,omitempty"`
	SecondaryPillRows [][]FilterPill `json:"secondary_pill_rows,omitempty"`
	Reset             *FilterReset   `json:"reset,omitempty"`
}

type FilterPill struct {
	Label    string `json:"label,omitempty"`
	LabelKey string `json:"label_key,omitempty"`
	Key      string `json:"key,omitempty"`
	Val      string `json:"val,omitempty"`
	Dot      bool   `json:"dot,omitempty"`
}

type FilterReset struct {
	Preserve []string `json:"preserve,omitempty"`
}

type Grid struct {
	Enabled bool     `json:"enabled"`
	Mode    GridMode `json:"mode,omitempty"`
}

type Pagination struct {
	Renderer RendererKey    `json:"renderer,omitempty"`
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
	ShowOnline    *bool  `json:"show_online,omitempty"`
	ShowAction    *bool  `json:"show_action,omitempty"`
}

type CardSchema struct {
	Type             string         `json:"type,omitempty"`
	Variant          CardVariant    `json:"variant,omitempty"`
	Size             SizeToken      `json:"size,omitempty"`
	SurfaceVariant   SurfaceVariant `json:"surface_variant,omitempty"`
	SurfaceEffect    SurfaceEffect  `json:"surface_effect,omitempty"`
	BadgeSize        SizeToken      `json:"badge_size,omitempty"`
	ActionSize       SizeToken      `json:"action_size,omitempty"`
	DeleteActionSize SizeToken      `json:"delete_action_size,omitempty"`
	PrimaryAction    string         `json:"primary_action,omitempty"`
	Media            *Media         `json:"media,omitempty"`
	Title            *TextBinding   `json:"title,omitempty"`
	Subtitle         *TextBinding   `json:"subtitle,omitempty"`
	SubtitleTone     string         `json:"subtitle_tone,omitempty"`
	Description      *TextBinding   `json:"description,omitempty"`
	Status           *StatusBinding `json:"status,omitempty"`
	Badges           []Badge        `json:"badges,omitempty"`
	Stats            []Badge        `json:"stats,omitempty"`
	Actions          []Action       `json:"actions,omitempty"`
}

type Media struct {
	Field        string      `json:"field,omitempty"`
	Renderer     RendererKey `json:"renderer,omitempty"`
	Ratio        MediaRatio  `json:"ratio,omitempty"`
	Size         MediaSize   `json:"size,omitempty"`
	Variant      string      `json:"variant,omitempty"`
	GlowField    string      `json:"glow_field,omitempty"`
	GlowFallback string      `json:"glow_fallback,omitempty"`
	GlowEnabled  *bool       `json:"glow_enabled,omitempty"`
	StatusField  string      `json:"status_field,omitempty"`
	Fallback     string      `json:"fallback,omitempty"`
}

type FieldPresentation struct {
	Renderer RendererKey `json:"renderer,omitempty"`
	Variant  string      `json:"variant,omitempty"`
	Style    string      `json:"style,omitempty"`
	Size     MediaSize   `json:"size,omitempty"`
	Ratio    MediaRatio  `json:"ratio,omitempty"`
}

type FieldMediaConfig struct {
	Item    *MediaGalleryItem    `json:"item,omitempty"`
	Upload  *MediaUploadConfig   `json:"upload,omitempty"`
	Labels  *MediaGalleryLabels  `json:"labels,omitempty"`
	Actions *MediaGalleryActions `json:"actions,omitempty"`
}

type TextBinding struct {
	Field    string `json:"field,omitempty"`
	Template string `json:"template,omitempty"`
}

type StatusBinding struct {
	ID         string            `json:"id,omitempty"`
	Field      string            `json:"field,omitempty"`
	Type       string            `json:"type,omitempty"`
	Option     string            `json:"option,omitempty"`
	Placement  string            `json:"placement,omitempty"`
	Marker     *bool             `json:"marker,omitempty"`
	OnlineTone string            `json:"online_tone,omitempty"`
	ToneMap    map[string]string `json:"tone_map,omitempty"`
}

type Badge struct {
	ID        string            `json:"id,omitempty"`
	Type      string            `json:"type,omitempty"`
	Field     string            `json:"field,omitempty"`
	IfField   string            `json:"if_field,omitempty"`
	Value     *TextBinding      `json:"value,omitempty"`
	Option    string            `json:"option,omitempty"`
	Placement string            `json:"placement,omitempty"`
	Label     string            `json:"label,omitempty"`
	LabelKey  string            `json:"label_key,omitempty"`
	Icon      string            `json:"icon,omitempty"`
	Size      SizeToken         `json:"size,omitempty"`
	Tone      string            `json:"tone,omitempty"`
	ToneMap   map[string]string `json:"tone_map,omitempty"`
	Marker    *bool             `json:"marker,omitempty"`
	Then      *BadgeState       `json:"then,omitempty"`
	Else      *BadgeState       `json:"else,omitempty"`
}

type BadgeState struct {
	ID       string `json:"id,omitempty"`
	Label    string `json:"label,omitempty"`
	LabelKey string `json:"label_key,omitempty"`
	Tone     string `json:"tone,omitempty"`
	Marker   *bool  `json:"marker,omitempty"`
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
	ID           string               `json:"id,omitempty"`
	Title        string               `json:"title,omitempty"`
	PanelTitle   string               `json:"panel_title,omitempty"`
	Subtitle     string               `json:"subtitle,omitempty"`
	Renderer     RendererKey          `json:"renderer,omitempty"`
	Group        string               `json:"group,omitempty"`
	GroupTitle   string               `json:"group_title,omitempty"`
	Icon         string               `json:"icon,omitempty"`
	Action       string               `json:"action,omitempty"`
	Mode         string               `json:"mode,omitempty"`
	Block        *Block               `json:"block,omitempty"`
	Fields       []string             `json:"fields,omitempty"`
	ListPage     *ListPage            `json:"list_page,omitempty"`
	Collection   *CollectionConfig    `json:"collection,omitempty"`
	MediaUpload  *MediaUploadConfig   `json:"media_upload,omitempty"`
	MediaItems   []MediaGalleryItem   `json:"media_items,omitempty"`
	MediaLabels  *MediaGalleryLabels  `json:"media_labels,omitempty"`
	MediaActions *MediaGalleryActions `json:"media_actions,omitempty"`
}

type MediaUploadConfig struct {
	Title        string `json:"title,omitempty"`
	Subtitle     string `json:"subtitle,omitempty"`
	LoadingTitle string `json:"loading_title,omitempty"`
	Accept       string `json:"accept,omitempty"`
	Multiple     bool   `json:"multiple"`
}

type MediaGalleryItem struct {
	ID          string          `json:"id,omitempty"`
	MediaID     int64           `json:"media_id,omitempty"`
	LinkID      int64           `json:"link_id,omitempty"`
	Kind        MediaKind       `json:"kind,omitempty"`
	Src         string          `json:"src,omitempty"`
	Poster      string          `json:"poster,omitempty"`
	Visibility  MediaVisibility `json:"visibility,omitempty"`
	Usage       MediaUsage      `json:"usage,omitempty"`
	SortOrder   int             `json:"sort_order"`
	Title       string          `json:"title,omitempty"`
	Description string          `json:"description,omitempty"`
}

type MediaGalleryLabels struct {
	Public       string `json:"public,omitempty"`
	Private      string `json:"private,omitempty"`
	Empty        string `json:"empty,omitempty"`
	CoverBadge   string `json:"cover_badge,omitempty"`
	Remove       string `json:"remove,omitempty"`
	Reorder      string `json:"reorder,omitempty"`
	FirstIsCover string `json:"first_is_cover,omitempty"`
	PrivateHint  string `json:"private_hint,omitempty"`
}

type MediaGalleryActions struct {
	Upload   *Action `json:"upload,omitempty"`
	Link     *Action `json:"link,omitempty"`
	Reorder  *Action `json:"reorder,omitempty"`
	Recenter *Action `json:"recenter,omitempty"`
	Crop     *Action `json:"crop,omitempty"`
	Remove   *Action `json:"remove,omitempty"`
}

type CollectionConfig struct {
	Module       string             `json:"module,omitempty"`
	Relation     string             `json:"relation,omitempty"`
	Item         *CollectionItem    `json:"item,omitempty"`
	Size         int                `json:"size,omitempty"`
	LoadingLabel string             `json:"loading_label,omitempty"`
	Buckets      []CollectionBucket `json:"buckets,omitempty"`
	EditFields   []string           `json:"edit_fields,omitempty"`
	Modal        *CollectionModal   `json:"modal,omitempty"`
	Actions      []Action           `json:"actions,omitempty"`
}

type CollectionItem struct {
	LabelField       string   `json:"label_field,omitempty"`
	MetaFields       []string `json:"meta_fields,omitempty"`
	DescriptionField string   `json:"description_field,omitempty"`
	MediaField       string   `json:"media_field,omitempty"`
	StatusField      string   `json:"status_field,omitempty"`
}

type CollectionBucket struct {
	ID            string                        `json:"id,omitempty"`
	Title         string                        `json:"title,omitempty"`
	CountLabel    string                        `json:"count_label,omitempty"`
	AddLabel      string                        `json:"add_label,omitempty"`
	ClearLabel    string                        `json:"clear_label,omitempty"`
	ModalTitle    string                        `json:"modal_title,omitempty"`
	ModalSubtitle string                        `json:"modal_subtitle,omitempty"`
	ConfirmLabel  string                        `json:"confirm_label,omitempty"`
	BlockID       string                        `json:"block_id,omitempty"`
	Block         *Block                        `json:"block,omitempty"`
	Predicate     *CollectionPredicate          `json:"predicate,omitempty"`
	Defaults      []CollectionFieldDefaultValue `json:"defaults,omitempty"`
	EditFields    []string                      `json:"edit_fields,omitempty"`
	Actions       []Action                      `json:"actions,omitempty"`
}

type CollectionPredicateOperator string

const (
	CollectionPredicateEquals      CollectionPredicateOperator = "eq"
	CollectionPredicateNotEquals   CollectionPredicateOperator = "ne"
	CollectionPredicateIn          CollectionPredicateOperator = "in"
	CollectionPredicateNotIn       CollectionPredicateOperator = "not_in"
	CollectionPredicateEmpty       CollectionPredicateOperator = "empty"
	CollectionPredicateNotEmpty    CollectionPredicateOperator = "not_empty"
	CollectionPredicateGreaterThan CollectionPredicateOperator = "gt"
	CollectionPredicateLessThan    CollectionPredicateOperator = "lt"
	CollectionPredicateGTE         CollectionPredicateOperator = "gte"
	CollectionPredicateLTE         CollectionPredicateOperator = "lte"
)

type CollectionPredicate struct {
	Field    string                      `json:"field,omitempty"`
	Operator CollectionPredicateOperator `json:"operator,omitempty"`
	Value    *TypedValue                 `json:"value,omitempty"`
	Values   []TypedValue                `json:"values,omitempty"`
}

type CollectionFieldDefaultValue struct {
	Field string     `json:"field,omitempty"`
	Value TypedValue `json:"value"`
}

type TypedValueType string

const (
	TypedValueString TypedValueType = "string"
	TypedValueNumber TypedValueType = "number"
	TypedValueBool   TypedValueType = "bool"
	TypedValueNull   TypedValueType = "null"
)

type TypedValue struct {
	Type   TypedValueType `json:"type"`
	String string         `json:"string,omitempty"`
	Number float64        `json:"number,omitempty"`
	Bool   *bool          `json:"bool,omitempty"`
}

// MarshalJSON keeps a typed zero value on the wire. A plain omitempty tag on
// Number makes number: 0 indistinguishable from an omitted value to clients.
func (v TypedValue) MarshalJSON() ([]byte, error) {
	type typedValueJSON struct {
		Type   TypedValueType `json:"type"`
		String *string        `json:"string,omitempty"`
		Number *float64       `json:"number,omitempty"`
		Bool   *bool          `json:"bool,omitempty"`
	}

	payload := typedValueJSON{Type: v.Type}
	switch v.Type {
	case TypedValueString:
		payload.String = &v.String
	case TypedValueNumber:
		payload.Number = &v.Number
	case TypedValueBool:
		payload.Bool = v.Bool
	}
	return json.Marshal(payload)
}

type CollectionModal struct {
	Icon                string `json:"icon,omitempty"`
	Search              bool   `json:"search"`
	SearchPlaceholder   string `json:"search_placeholder,omitempty"`
	EmptyLabel          string `json:"empty_label,omitempty"`
	SelectedLabel       string `json:"selected_label,omitempty"`
	TakenLabel          string `json:"taken_label,omitempty"`
	CancelLabel         string `json:"cancel_label,omitempty"`
	ConfirmLoadingLabel string `json:"confirm_loading_label,omitempty"`
}

type Block struct {
	Type           BlockType       `json:"type,omitempty"`
	Variant        BlockVariant    `json:"variant,omitempty"`
	TitleDecor     TitleDecorToken `json:"title_decor,omitempty"`
	TitleBar       ToneToken       `json:"title_bar,omitempty"`
	TitleUnderline ToneToken       `json:"title_underline,omitempty"`
	Inset          InsetToken      `json:"inset,omitempty"`
	MaxWidth       string          `json:"max_width,omitempty"`
	BodyClass      string          `json:"body_class,omitempty"`
	BorderStyle    string          `json:"border_style,omitempty"`
	HoverEnabled   *bool           `json:"hover_enabled,omitempty"`
	Effect         string          `json:"effect,omitempty"`
}

type Stack struct {
	Gap       SpacingToken   `json:"gap,omitempty"`
	Direction DirectionToken `json:"direction,omitempty"`
	Wrap      *bool          `json:"wrap,omitempty"`
	Justify   JustifyToken   `json:"justify,omitempty"`
	Align     AlignToken     `json:"align,omitempty"`
	Inset     InsetToken     `json:"inset,omitempty"`
}

type DisplayComponent struct {
	ID                  string                   `json:"id,omitempty"`
	Type                DisplayComponentType     `json:"type,omitempty"`
	Fields              []string                 `json:"fields,omitempty"`
	Value               interface{}              `json:"value,omitempty"`
	Default             interface{}              `json:"default,omitempty"`
	Visible             *bool                    `json:"visible,omitempty"`
	UpdateAction        ComponentAction          `json:"update_action,omitempty"`
	MainRatio           ComponentRatio           `json:"main_ratio,omitempty"`
	MainRadius          ComponentRadiusToken     `json:"main_radius,omitempty"`
	MainRadiusToken     ComponentRadiusToken     `json:"main_radius_token,omitempty"`
	ThumbRatio          ComponentRatio           `json:"thumb_ratio,omitempty"`
	ThumbsInset         InsetToken               `json:"thumbs_inset,omitempty"`
	ThumbsInsetToken    SpacingToken             `json:"thumbs_inset_token,omitempty"`
	VideoControls       *bool                    `json:"video_controls,omitempty"`
	Size                SizeToken                `json:"size,omitempty"`
	Wrap                *bool                    `json:"wrap,omitempty"`
	Gap                 SpacingToken             `json:"gap,omitempty"`
	Direction           DirectionToken           `json:"direction,omitempty"`
	Justify             JustifyToken             `json:"justify,omitempty"`
	Align               AlignToken               `json:"align,omitempty"`
	Inset               InsetToken               `json:"inset,omitempty"`
	Compact             bool                     `json:"compact,omitempty"`
	Columns             int                      `json:"columns,omitempty"`
	ReadonlyColumns     int                      `json:"readonly_columns,omitempty"`
	DisplayType         ComponentDisplayType     `json:"display_type,omitempty"`
	SeparatorVariant    ToneToken                `json:"separator_variant,omitempty"`
	SeparatorAppearance SeparatorAppearance      `json:"separator_appearance,omitempty"`
	MatrixColumns       []map[string]interface{} `json:"matrix_columns,omitempty"`
	ValueLabel          string                   `json:"value_label,omitempty"`
	ValueFallback       string                   `json:"value_fallback,omitempty"`
	MatrixLabel         string                   `json:"matrix_label,omitempty"`
	MatrixLabelIcon     string                   `json:"matrix_label_icon,omitempty"`
	Block               *Block                   `json:"block,omitempty"`
	Title               string                   `json:"title,omitempty"`
	TitleFallback       string                   `json:"title_fallback,omitempty"`
	Subtitle            string                   `json:"subtitle,omitempty"`
	SubtitleFallback    string                   `json:"subtitle_fallback,omitempty"`
	TitleLevel          int                      `json:"title_level,omitempty"`
	TitleTone           ToneToken                `json:"title_tone,omitempty"`
	BodyClass           string                   `json:"body_class,omitempty"`
}

type RecordPage struct {
	ID            string            `json:"id,omitempty"`
	Title         string            `json:"title,omitempty"`
	Subtitle      string            `json:"subtitle,omitempty"`
	ShowHeader    *bool             `json:"show_header,omitempty"`
	Badge         string            `json:"badge,omitempty"`
	BadgeTone     string            `json:"badge_tone,omitempty"`
	BadgeTeleport string            `json:"badge_teleport,omitempty"`
	Navigation    *RecordNavigation `json:"navigation,omitempty"`
	Layout        *Layout           `json:"layout,omitempty"`
	Sections      []RecordSection   `json:"sections,omitempty"`
	Theme         *RecordTheme      `json:"theme,omitempty"`
	Actions       []Action          `json:"actions,omitempty"`
}

type RecordNavigation struct {
	Type    string `json:"type,omitempty"`
	Enabled bool   `json:"enabled"`
}

type RecordTheme struct {
	Surfaces   map[string]string      `json:"surfaces,omitempty"`
	Headings   map[string]interface{} `json:"headings,omitempty"`
	Badges     map[string]string      `json:"badges,omitempty"`
	Buttons    map[string]interface{} `json:"buttons,omitempty"`
	Media      map[string]string      `json:"media,omitempty"`
	Components map[string]interface{} `json:"components,omitempty"`
}

type RecordSection struct {
	ID            string                `json:"id,omitempty"`
	Title         string                `json:"title,omitempty"`
	TitleFallback string                `json:"title_fallback,omitempty"`
	TitleLevel    int                   `json:"title_level,omitempty"`
	TitleTone     ToneToken             `json:"title_tone,omitempty"`
	Renderer      RecordSectionRenderer `json:"renderer,omitempty"`
	LayoutSlot    LayoutSlotToken       `json:"layout_slot,omitempty"`
	Order         int                   `json:"order,omitempty"`
	Block         *Block                `json:"block,omitempty"`
	Stack         *Stack                `json:"stack,omitempty"`
	Components    []DisplayComponent    `json:"components,omitempty"`
}

type ResourceGridPage struct {
	Endpoint string                     `json:"endpoint,omitempty"`
	List     *ResourceGridListConfig    `json:"list,omitempty"`
	Create   *Action                    `json:"create,omitempty"`
	Delete   *Action                    `json:"delete,omitempty"`
	Update   *Action                    `json:"update,omitempty"`
	Card     *CardSchema                `json:"card,omitempty"`
	Status   *ResourceGridStatusConfig  `json:"status,omitempty"`
	Actions  *ResourceGridActionsConfig `json:"actions,omitempty"`
	Text     map[string]string          `json:"text,omitempty"`
	Context  map[string]interface{}     `json:"context,omitempty"`
}

type ResourceGridListConfig struct {
	Size    int                    `json:"size,omitempty"`
	Filters map[string]interface{} `json:"filters,omitempty"`
}

type ResourceGridStatusConfig struct {
	VerifyField         string        `json:"verifyField,omitempty"`
	ActiveField         string        `json:"activeField,omitempty"`
	VerifiedValue       string        `json:"verifiedValue,omitempty"`
	PendingValue        string        `json:"pendingValue,omitempty"`
	InactiveValue       string        `json:"inactiveValue,omitempty"`
	InactiveActionValue string        `json:"inactiveActionValue,omitempty"`
	ActiveActionValue   string        `json:"activeActionValue,omitempty"`
	DraftValues         []interface{} `json:"draftValues,omitempty"`
	PendingPayload      interface{}   `json:"pendingPayload,omitempty"`
}

type ResourceGridActionsConfig struct {
	EditRoute interface{} `json:"editRoute,omitempty"`
}

type Action struct {
	ID               string           `json:"id,omitempty"`
	Type             ActionType       `json:"type,omitempty"`
	Label            string           `json:"label,omitempty"`
	LabelKey         string           `json:"label_key,omitempty"`
	Icon             string           `json:"icon,omitempty"`
	Variant          ActionVariant    `json:"variant,omitempty"`
	Appearance       ActionAppearance `json:"appearance,omitempty"`
	ActiveAppearance ActionAppearance `json:"active_appearance,omitempty"`
	Active           string           `json:"active,omitempty"`
	External         *bool            `json:"external,omitempty"`
	Block            *bool            `json:"block,omitempty"`
	VisibleIf        *Condition       `json:"visible_if,omitempty"`
	HiddenIf         *Condition       `json:"hidden_if,omitempty"`
	DisabledIf       *Condition       `json:"disabled_if,omitempty"`
	Endpoint         string           `json:"endpoint,omitempty"`
	Method           string           `json:"method,omitempty"`
	UniqueEndpoint   string           `json:"uniqueEndpoint,omitempty"`
	AfterRoute       interface{}      `json:"afterRoute,omitempty"`
	Route            interface{}      `json:"route,omitempty"`
	API              *APIAction       `json:"api,omitempty"`
	Modal            *ModalAction     `json:"modal,omitempty"`
	Confirm          *Confirm         `json:"confirm,omitempty"`
	AfterSuccess     *ActionResult    `json:"after_success,omitempty"`
	AfterError       *ActionResult    `json:"after_error,omitempty"`
	AriaLabelKey     string           `json:"aria_label_key,omitempty"`
	TitleKey         string           `json:"title_key,omitempty"`
	Test             string           `json:"test,omitempty"`
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
	Renderer RendererKey            `json:"renderer,omitempty"`
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
	Not       interface{}   `json:"not,omitempty"`
}
