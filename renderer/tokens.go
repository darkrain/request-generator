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
