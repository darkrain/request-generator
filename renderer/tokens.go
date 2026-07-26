package renderer

const (
	Name    = "UniversalRenderer"
	Version = "1.0.0"
)

type Identity struct {
	Name    string `json:"name"`
	Version string `json:"version"`
}

func UniversalIdentity() Identity {
	return Identity{Name: Name, Version: Version}
}

type PageType string

const (
	PageTypeList         PageType = "list"
	PageTypeForm         PageType = "form"
	PageTypeRecord       PageType = "record"
	PageTypeResourceGrid PageType = "resource_grid"
)

type SpacingToken string

const (
	SpacingNone SpacingToken = "none"
	SpacingXS   SpacingToken = "xs"
	SpacingSM   SpacingToken = "sm"
	SpacingMD   SpacingToken = "md"
	SpacingLG   SpacingToken = "lg"
	SpacingXL   SpacingToken = "xl"
)

type RadiusToken string

const (
	RadiusNone RadiusToken = "none"
	RadiusSM   RadiusToken = "sm"
	RadiusMD   RadiusToken = "md"
	RadiusLG   RadiusToken = "lg"
	RadiusXL   RadiusToken = "xl"
	RadiusFull RadiusToken = "full"
)

type InsetToken string

const (
	InsetNone       InsetToken = "none"
	InsetInlineMD   InsetToken = "inline-md"
	InsetMediaFrame InsetToken = "media-frame"
)

type ComponentRadiusToken string

const (
	ComponentRadiusNone     ComponentRadiusToken = "none"
	ComponentRadiusMediaTop ComponentRadiusToken = "media-top"
)

type SizeToken string

const (
	SizeXS SizeToken = "xs"
	SizeSM SizeToken = "sm"
	SizeMD SizeToken = "md"
	SizeLG SizeToken = "lg"
	SizeXL SizeToken = "xl"
)

type WeightToken string

const (
	WeightRegular  WeightToken = "regular"
	WeightMedium   WeightToken = "medium"
	WeightSemibold WeightToken = "semibold"
	WeightBold     WeightToken = "bold"
)

type AlignToken string

const (
	AlignStart   AlignToken = "start"
	AlignCenter  AlignToken = "center"
	AlignEnd     AlignToken = "end"
	AlignStretch AlignToken = "stretch"
)

type ToneToken string

const (
	ToneDefault ToneToken = "default"
	ToneMuted   ToneToken = "muted"
	ToneSoft    ToneToken = "soft"
	ToneAccent  ToneToken = "accent"
	ToneSuccess ToneToken = "success"
	ToneWarning ToneToken = "warning"
	ToneDanger  ToneToken = "danger"
)

type LayoutType string

const (
	LayoutOneColumn   LayoutType = "one_column"
	LayoutTwoColumn   LayoutType = "two_column"
	LayoutThreeColumn LayoutType = "three_column"
)

type MaxWidth string

const (
	MaxWidthNone MaxWidth = "none"
	MaxWidthSM   MaxWidth = "sm"
	MaxWidthMD   MaxWidth = "md"
	MaxWidthLG   MaxWidth = "lg"
	MaxWidthXL   MaxWidth = "xl"
	MaxWidthFull MaxWidth = "full"
)

type GridMode string

const (
	GridModeTable GridMode = "table"
	GridModeCards GridMode = "cards"
)

type PaginationMode string

const (
	PaginationServer PaginationMode = "server"
	PaginationClient PaginationMode = "client"
)

type CardVariant string

const (
	CardVariantDefault CardVariant = "default"
	CardVariantMedia   CardVariant = "media"
	CardVariantCompact CardVariant = "compact"
)

type SurfaceVariant string

const (
	SurfaceDefault   SurfaceVariant = "default"
	SurfacePrimary   SurfaceVariant = "primary"
	SurfaceSecondary SurfaceVariant = "secondary"
)

type SurfaceEffect string

const (
	SurfaceEffectNone     SurfaceEffect = "none"
	SurfaceEffectFlat     SurfaceEffect = "flat"
	SurfaceEffectElevated SurfaceEffect = "elevated"
)

type MediaRatio string

const (
	MediaRatioSquare    MediaRatio = "square"
	MediaRatioPortrait  MediaRatio = "portrait"
	MediaRatioLandscape MediaRatio = "landscape"
	MediaRatioWide      MediaRatio = "wide"
)

type MediaSize string

const (
	MediaSizeThumb MediaSize = "thumb"
	MediaSizeCard  MediaSize = "card"
	MediaSizeHero  MediaSize = "hero"
)

type BlockType string

const (
	BlockNone  BlockType = "none"
	BlockPanel BlockType = "panel"
	BlockCard  BlockType = "card"
)

type BlockVariant string

const (
	BlockVariantDefault BlockVariant = "default"
	BlockVariantCompact BlockVariant = "compact"
)

type TitleDecorToken string

const (
	TitleDecorNone    TitleDecorToken = "none"
	TitleDecorSection TitleDecorToken = "section"
)

type RendererKey string

const (
	RendererUniversalDisplay     RendererKey = "universal.display"
	RendererUniversalSection     RendererKey = "universal.section"
	RendererUniversalPreferences RendererKey = "universal.preferences"
	RendererUniversalFilters     RendererKey = "universal.filters"
	RendererUniversalPagination  RendererKey = "universal.pagination"
	RendererMediaGallery         RendererKey = "media.gallery"
	RendererCollectionManager    RendererKey = "collection.manager"
	RendererAvatar               RendererKey = "avatar"
)

type RecordSectionRenderer = RendererKey

const (
	RecordRendererDisplay RecordSectionRenderer = RendererUniversalDisplay
)

type LayoutSlotToken string

const (
	LayoutSlotLeft   LayoutSlotToken = "left"
	LayoutSlotCenter LayoutSlotToken = "center"
	LayoutSlotRight  LayoutSlotToken = "right"
)

type ActionType string

const (
	ActionRoute    ActionType = "route"
	ActionAPI      ActionType = "api"
	ActionModal    ActionType = "modal"
	ActionEmit     ActionType = "emit"
	ActionExternal ActionType = "external"
)

type ActionVariant string

const (
	ActionVariantDefault   ActionVariant = "default"
	ActionVariantPrimary   ActionVariant = "primary"
	ActionVariantSecondary ActionVariant = "secondary"
	ActionVariantSuccess   ActionVariant = "success"
	ActionVariantWarning   ActionVariant = "warning"
	ActionVariantDanger    ActionVariant = "danger"
)

type ActionAppearance string

const (
	ActionAppearanceSolid   ActionAppearance = "solid"
	ActionAppearanceOutline ActionAppearance = "outline"
	ActionAppearanceGhost   ActionAppearance = "ghost"
	ActionAppearanceSoft    ActionAppearance = "soft"
	ActionAppearanceLink    ActionAppearance = "link"
)

type DisplayComponentType string

const (
	DisplayMediaGallery    DisplayComponentType = "media_gallery"
	DisplayActions         DisplayComponentType = "actions"
	DisplayIdentity        DisplayComponentType = "identity"
	DisplayStatList        DisplayComponentType = "stat_list"
	DisplayDataList        DisplayComponentType = "data_list"
	DisplayBadgeGroupBlock DisplayComponentType = "badge_group_block"
	DisplayText            DisplayComponentType = "text"
	DisplayBadgeList       DisplayComponentType = "badge_list"
	DisplayRateGroups      DisplayComponentType = "rate_groups"
	DisplayAccordionGroups DisplayComponentType = "accordion_groups"
)

type DataPath string

const (
	DataGalleryItems       DataPath = "gallery.items"
	DataGalleryCurrent     DataPath = "gallery.current"
	DataGalleryOverlays    DataPath = "gallery.overlays"
	DataGalleryActions     DataPath = "gallery.actions"
	DataHeroIdentity       DataPath = "hero.identity"
	DataHeroStats          DataPath = "hero.stats"
	DataDetailsItems       DataPath = "details.items"
	DataDetailsCommercial  DataPath = "details.commercial"
	DataAboutDescription   DataPath = "about.description"
	DataMeetingsItems      DataPath = "meetings.items"
	DataMeetingsCommission DataPath = "meetings.commission"
	DataMeetingsWorkArea   DataPath = "meetings.workArea"
	DataMeetingsPayments   DataPath = "meetings.payments"
	DataRatesGroups        DataPath = "rates.groups"
	DataServicesGroups     DataPath = "services.groups"
	DataContactsItems      DataPath = "contacts.items"
)

type ComponentAction string

const (
	ComponentActionGallerySet ComponentAction = "gallery:set"
)

type ComponentDisplayType string

const (
	ComponentDisplayKeyValueGrid ComponentDisplayType = "key_value_grid"
)

type ComponentRatio string

const (
	ComponentRatioSquare   ComponentRatio = "square"
	ComponentRatioPortrait ComponentRatio = "portrait"
)

type DirectionToken string

const (
	DirectionRow    DirectionToken = "row"
	DirectionColumn DirectionToken = "column"
)

type JustifyToken string

const (
	JustifyStart   JustifyToken = "start"
	JustifyCenter  JustifyToken = "center"
	JustifyEnd     JustifyToken = "end"
	JustifyStretch JustifyToken = "stretch"
	JustifyBetween JustifyToken = "between"
)

type SeparatorAppearance string

const (
	SeparatorSolid    SeparatorAppearance = "solid"
	SeparatorGradient SeparatorAppearance = "gradient"
)
