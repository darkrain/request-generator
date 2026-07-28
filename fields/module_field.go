package fields

import (
	"database/sql"
	"errors"
	"fmt"

	"github.com/darkrain/request-generator/locale"
	"github.com/gin-gonic/gin"
	pg "github.com/go-jet/jet/v2/postgres"
	validation "github.com/go-ozzo/ozzo-validation/v4"
	"github.com/go-ozzo/ozzo-validation/v4/is"
)

const (
	ErrorUnknownType     string = "Unknown type"
	ErrorUnknownFormType string = "Unknown formType"
)

type ModuleFieldType string

const (
	ModuleFieldTypeString ModuleFieldType = "string"
	ModuleFieldTypeInt    ModuleFieldType = "int"
	ModuleFieldTypeFloat  ModuleFieldType = "float"
	ModuleFieldTypeArray  ModuleFieldType = "array"
	ModuleFieldTypeObject ModuleFieldType = "object"
)

func ModuleFieldTypeOf(value string) (ModuleFieldType, error) {
	switch value {
	case string(ModuleFieldTypeString):
		return ModuleFieldTypeString, nil
	case string(ModuleFieldTypeInt):
		return ModuleFieldTypeInt, nil
	case string(ModuleFieldTypeArray):
		return ModuleFieldTypeArray, nil
	case string(ModuleFieldTypeObject):
		return ModuleFieldTypeObject, nil
	}
	return ModuleFieldTypeString, errors.New(ErrorUnknownFormType)
}

type ModuleFieldFormType string

const (
	ModuleFieldFormTypeText        ModuleFieldFormType = "text"
	ModuleFieldFormTypeNumber      ModuleFieldFormType = "number"
	ModuleFieldFormTypeTextArea    ModuleFieldFormType = "textarea"
	ModuleFieldFormTypeSelect      ModuleFieldFormType = "select"
	ModuleFieldFormTypeCheckBox    ModuleFieldFormType = "checkbox"
	ModuleFieldFormTypeMultiselect ModuleFieldFormType = "multiselect"
	ModuleFieldFormTypeMap         ModuleFieldFormType = "map"
	ModuleFieldFormTypeHidden      ModuleFieldFormType = "hidden"
	ModuleFieldFormTypeOnlyView    ModuleFieldFormType = "onlyview"
)

func ModuleFieldFormTypeOf(value string) (ModuleFieldFormType, error) {
	switch value {
	case string(ModuleFieldFormTypeText):
		return ModuleFieldFormTypeMap, nil
	case string(ModuleFieldFormTypeNumber):
		return ModuleFieldFormTypeNumber, nil
	case string(ModuleFieldFormTypeTextArea):
		return ModuleFieldFormTypeTextArea, nil
	case string(ModuleFieldFormTypeSelect):
		return ModuleFieldFormTypeSelect, nil
	case string(ModuleFieldFormTypeCheckBox):
		return ModuleFieldFormTypeCheckBox, nil
	case string(ModuleFieldFormTypeMultiselect):
		return ModuleFieldFormTypeMultiselect, nil
	case string(ModuleFieldFormTypeMap):
		return ModuleFieldFormTypeMap, nil
	case string(ModuleFieldFormTypeHidden):
		return ModuleFieldFormTypeHidden, nil
	case string(ModuleFieldFormTypeOnlyView):
		return ModuleFieldFormTypeOnlyView, nil
	}
	return ModuleFieldFormTypeMap, errors.New(ErrorUnknownFormType)
}

type Scenario string

const (
	ScenarioAdd    Scenario = "add"
	ScenarioUpdate Scenario = "update"
)

type RoleCheck struct {
	Role  string
	Rules []CheckRules
}

type RoleOptions struct {
	Role    string
	Options []ModuleFieldOptions
}

// FieldExtra holds per-context extra metadata for a field.
type FieldExtra struct {
	View   interface{} `json:"-"`
	List   interface{} `json:"-"`
	Defrec interface{} `json:"-"`
}

type ModuleFieldMatrixBinding struct {
	Row    string `json:"row,omitempty"`
	Column string `json:"column,omitempty"`
}

type ModuleFieldLocationPicker struct {
	CountryLabel       string   `json:"country_label,omitempty"`
	CityLabel          string   `json:"city_label,omitempty"`
	CountryPlaceholder string   `json:"country_placeholder,omitempty"`
	CityPlaceholder    string   `json:"city_placeholder,omitempty"`
	CityMode           string   `json:"city_mode,omitempty"`
	HideInnerLabels    bool     `json:"hide_inner_labels,omitempty"`
	AllowedCountries   []string `json:"allowed_countries,omitempty"`
}

type ModuleField struct {
	Column           pg.Column                                       `json:"-"`
	SelectExpression pg.Projection                                   `json:"-"`
	Title            string                                          `json:"title"`
	Titles           map[string]string                               `json:"-"`
	Type             ModuleFieldType                                 `json:"type"`
	FormType         ModuleFieldFormType                             `json:"form_type,omitempty"`
	Example          string                                          `json:"example,omitempty"`
	AllLabel         string                                          `json:"all_label,omitempty"`
	Extra            *FieldExtra                                     `json:"-"`
	Options          []ModuleFieldOptions                            `json:"options,omitempty"`
	OptionsURL       string                                          `json:"options_url,omitempty"`
	OptionsFunc      func(context *gin.Context) []ModuleFieldOptions `json:"-"`
	RoleOptions      []RoleOptions                                   `json:"-"`
	Check            []CheckRules                                    `json:"-"`
	CheckFunc        func(context *gin.Context) []CheckRules         `json:"-"`
	RoleCheck        []RoleCheck                                     `json:"-"`
	// DefaultFunc is called during Add when the field is absent from the request body.
	// The returned value is injected into the input before validation and DB insert.
	DefaultFunc          func(c *gin.Context) interface{}                             `json:"-"`
	Convert              func(c *gin.Context, value interface{}) (interface{}, error) `json:"-"`
	ResultValueConverter func(value interface{}) interface{}                          `json:"-"`
	Translatable         bool                                                         `json:"-"`
	Group                string                                                       `json:"-"`
	Order                int                                                          `json:"-"`
	FieldName            string                                                       `json:"-"`
	FilterCondition      func(c *gin.Context) bool                                    `json:"-"`
	Roles                []string                                                     `json:"roles,omitempty"`
	Section              string                                                       `json:"section,omitempty"`
	RoleSection          map[string]string                                            `json:"-"`
	RoleFormType         map[string]ModuleFieldFormType                               `json:"-"`
	RoleLocationPicker   map[string]ModuleFieldLocationPicker                         `json:"-"`
	VisualKind           string                                                       `json:"visual_kind,omitempty"`
	Width                string                                                       `json:"width,omitempty"`
	Span                 string                                                       `json:"span,omitempty"`
	HideLabel            bool                                                         `json:"hide_label,omitempty"`
	Hint                 string                                                       `json:"hint,omitempty"`
	Description          string                                                       `json:"description,omitempty"`
	Meta                 string                                                       `json:"meta,omitempty"`
	Prefix               string                                                       `json:"prefix,omitempty"`
	Suffix               string                                                       `json:"suffix,omitempty"`
	Icon                 string                                                       `json:"icon,omitempty"`
	IconSVG              string                                                       `json:"icon_svg,omitempty"`
	Glow                 string                                                       `json:"glow,omitempty"`
	MaxLength            int                                                          `json:"max_length,omitempty"`
	Rows                 int                                                          `json:"rows,omitempty"`
	Searchable           *bool                                                        `json:"searchable,omitempty"`
	OptionIcons          map[string]string                                            `json:"option_icons,omitempty"`
	OptionsParams        map[string]interface{}                                       `json:"options_params,omitempty"`
	VisibleWhen          map[string]interface{}                                       `json:"visible_when,omitempty"`
	Matrix               *ModuleFieldMatrixBinding                                    `json:"matrix,omitempty"`
	LocationPicker       *ModuleFieldLocationPicker                                   `json:"location_picker,omitempty"`
	UploadLabel          string                                                       `json:"upload_label,omitempty"`
	UploadingLabel       string                                                       `json:"uploading_label,omitempty"`
	RecenterLabel        string                                                       `json:"recenter_label,omitempty"`
	RemoveLabel          string                                                       `json:"remove_label,omitempty"`
}

// ColumnName returns the database column name from the Jet column.
func (f ModuleField) ColumnName() string {
	if f.Column != nil {
		return f.Column.Name()
	}
	return ""
}

// Name returns the logical field name. For translatable fields this is
// FieldName (e.g. "name"); for regular fields it falls back to ColumnName().
func (f ModuleField) Name() string {
	if f.FieldName != "" {
		return f.FieldName
	}
	return f.ColumnName()
}

func (f ModuleField) UIMap() map[string]interface{} {
	return f.UIMapForRole("")
}

func (f ModuleField) UIMapForRole(role string) map[string]interface{} {
	item := map[string]interface{}{}
	if f.VisualKind != "" {
		item["visual_kind"] = f.VisualKind
	}
	if f.Width != "" {
		item["width"] = f.Width
	}
	if f.Span != "" {
		item["span"] = f.Span
	}
	if f.HideLabel {
		item["hide_label"] = true
	}
	if f.Hint != "" {
		item["hint"] = f.Hint
	}
	if f.Description != "" {
		item["description"] = f.Description
	}
	if f.Meta != "" {
		item["meta"] = f.Meta
	}
	if f.Prefix != "" {
		item["prefix"] = f.Prefix
	}
	if f.Suffix != "" {
		item["suffix"] = f.Suffix
	}
	if f.Icon != "" {
		item["icon"] = f.Icon
	}
	if f.IconSVG != "" {
		item["icon_svg"] = f.IconSVG
	}
	if f.Glow != "" {
		item["glow"] = f.Glow
	}
	if f.MaxLength > 0 {
		item["max_length"] = f.MaxLength
	}
	if f.Rows > 0 {
		item["rows"] = f.Rows
	}
	if f.Searchable != nil {
		item["searchable"] = *f.Searchable
	}
	if f.OptionsURL != "" {
		item["options_url"] = f.OptionsURL
	}
	if len(f.OptionIcons) > 0 {
		item["option_icons"] = f.OptionIcons
	}
	if len(f.OptionsParams) > 0 {
		item["options_params"] = f.OptionsParams
	}
	if len(f.VisibleWhen) > 0 {
		item["visible_when"] = f.VisibleWhen
	}
	if f.Matrix != nil {
		item["matrix"] = f.Matrix
	}
	if role != "" && f.RoleLocationPicker != nil {
		if locationPicker, ok := f.RoleLocationPicker[role]; ok {
			item["location_picker"] = locationPicker
		}
	}
	if _, ok := item["location_picker"]; !ok && f.LocationPicker != nil {
		item["location_picker"] = f.LocationPicker
	}
	if f.UploadLabel != "" {
		item["upload_label"] = f.UploadLabel
	}
	if f.UploadingLabel != "" {
		item["uploading_label"] = f.UploadingLabel
	}
	if f.RecenterLabel != "" {
		item["recenter_label"] = f.RecenterLabel
	}
	if f.RemoveLabel != "" {
		item["remove_label"] = f.RemoveLabel
	}
	if len(item) == 0 {
		return nil
	}
	return item
}

// GetProjection returns the SELECT expression for this field.
// If SelectExpression is set (e.g. a function wrapper), it is used instead of the raw column.
func (f ModuleField) GetProjection() pg.Projection {
	if f.SelectExpression != nil {
		return f.SelectExpression
	}
	return f.Column
}

// NewScanValue returns a fresh sql scan destination appropriate for this column's type.
func (f ModuleField) NewScanValue() interface{} {
	switch f.Column.(type) {
	case pg.ColumnBool:
		return &sql.NullBool{}
	case pg.ColumnInteger:
		return &sql.NullInt64{}
	case pg.ColumnFloat:
		return &sql.NullFloat64{}
	case pg.ColumnTimestamp, pg.ColumnTimestampz, pg.ColumnDate, pg.ColumnTime, pg.ColumnTimez:
		return &sql.NullTime{}
	default:
		return &sql.NullString{}
	}
}

type ModuleFilterField struct {
	Column          pg.Column                                                    `json:"-"`
	FieldName       string                                                       `json:"-"`
	Title           string                                                       `json:"title"`
	Titles          map[string]string                                            `json:"-"`
	Type            ModuleFieldType                                              `json:"type"`
	FormType        ModuleFieldFormType                                          `json:"form_type,omitempty"`
	Example         string                                                       `json:"example,omitempty"`
	AllLabel        string                                                       `json:"all_label,omitempty"`
	Options         []ModuleFieldOptions                                         `json:"options,omitempty"`
	Check           []CheckRules                                                 `json:"-"`
	Convert         func(c *gin.Context, value interface{}) (interface{}, error) `json:"-"`
	Group           string                                                       `json:"group,omitempty"`
	Order           int                                                          `json:"order,omitempty"`
	Extra           interface{}                                                  `json:"extra,omitempty"`
	FilterCondition func(c *gin.Context) bool                                    `json:"-"`
}

func (f ModuleFilterField) ColumnName() string {
	if f.Column != nil {
		return f.Column.Name()
	}
	return ""
}

type ModuleFieldOptions struct {
	Value  interface{}       `json:"value"`
	Label  string            `json:"label"`
	Labels map[string]string `json:"-"`
}

type CheckRules interface {
	Validate(obj interface{}, lang string) error
	GetScenarios() []Scenario
}

// DataCheckRule is a CheckRules variant that has access to the full input map,
// gin.Context, and the raw *sql.DB. Use it for composite validation spanning
// multiple fields (e.g. uniqueness across (who_add, whom_add, tag)).
// Implementors must also satisfy CheckRules to be stored in a Check slice;
// the Validate method can be a no-op since the generator calls ValidateData instead.
type DataCheckRule interface {
	CheckRules
	ValidateData(c *gin.Context, db *sql.DB, data map[string]interface{}, lang string) error
}

// dataRule is a function-based DataCheckRule for inline use.
type dataRule struct {
	fn        func(c *gin.Context, db *sql.DB, data map[string]interface{}, lang string) error
	scenarios []Scenario
}

func (r dataRule) Validate(_ interface{}, _ string) error { return nil }
func (r dataRule) GetScenarios() []Scenario               { return r.scenarios }
func (r dataRule) ValidateData(c *gin.Context, db *sql.DB, data map[string]interface{}, lang string) error {
	return r.fn(c, db, data, lang)
}

// DataRule creates a CheckRules entry for composite, context-aware validation.
// The fn receives gin.Context, *sql.DB, and the full input map — suitable for
// multi-field uniqueness checks or any validation that needs DB access.
func DataRule(fn func(c *gin.Context, db *sql.DB, data map[string]interface{}, lang string) error, scenarios []Scenario) CheckRules {
	return dataRule{fn: fn, scenarios: scenarios}
}

// RuleInfo holds validation rule metadata for OpenAPI spec generation.
type RuleInfo struct {
	Type      string // "required", "in", "length", "url", "email"
	Field     string
	Values    []interface{} // for "in" rules
	Min       int           // for "length" rules
	Max       int           // for "length" rules
	Scenarios []Scenario
}

// CheckRuleIntrospectable exposes rule metadata for documentation generation.
type CheckRuleIntrospectable interface {
	RuleInfo() RuleInfo
}

func RequiredRule(field pg.Column, scenarios []Scenario) requiredRule {
	return requiredRule{
		Field:     field,
		Scenarios: scenarios,
	}
}

func InRule(field pg.Column, values []interface{}, scenarios []Scenario) inRule {
	return inRule{
		Field:     field,
		Values:    values,
		Scenarios: scenarios,
	}
}

func InDBRule(field pg.Column, values func() []interface{}, scenarios []Scenario) inRule {
	return inRule{
		Field:     field,
		Values:    values(),
		Scenarios: scenarios,
	}
}

func LenRule(field pg.Column, min int, max int, scenarios []Scenario) lengthRule {
	return lengthRule{
		Min:       min,
		Max:       max,
		Field:     field,
		Scenarios: scenarios,
	}
}

func UrlRule(field pg.Column, scenarios []Scenario) urlRule {
	return urlRule{
		Field:     field,
		Scenarios: scenarios,
	}
}

type requiredRule struct {
	CheckRules `json:"-"`
	Field      pg.Column  `json:"-"`
	Scenarios  []Scenario `json:"scenarios"`
}

type inRule struct {
	Field     pg.Column     `json:"-"`
	Values    []interface{} `json:"values"`
	Scenarios []Scenario    `json:"scenarios"`
}

type emailRule struct {
	CheckRules `json:"-"`
	Type       string     `json:"type"`
	Field      pg.Column  `json:"-"`
	Scenarios  []Scenario `json:"scenarios"`
}

type urlRule struct {
	CheckRules `json:"-"`
	Type       string     `json:"type"`
	Field      pg.Column  `json:"-"`
	Scenarios  []Scenario `json:"scenarios"`
}

type lengthRule struct {
	CheckRules `json:"-"`
	Type       string     `json:"type"`
	Min        int        `json:"min"`
	Max        int        `json:"max"`
	Field      pg.Column  `json:"-"`
	Scenarios  []Scenario `json:"scenarios"`
}

func (rule requiredRule) GetScenarios() []Scenario {
	return rule.Scenarios
}

func (rule inRule) GetScenarios() []Scenario {
	return rule.Scenarios
}

func (rule emailRule) GetScenarios() []Scenario {
	return rule.Scenarios
}

func (rule urlRule) GetScenarios() []Scenario {
	return rule.Scenarios
}

func (rule lengthRule) GetScenarios() []Scenario {
	return rule.Scenarios
}

func (rule requiredRule) RuleInfo() RuleInfo {
	return RuleInfo{Type: "required", Field: rule.Field.Name(), Scenarios: rule.Scenarios}
}

func (rule requiredRule) Validate(obj interface{}, lang string) error {
	return validation.Required.Error(fmt.Sprintf(locale.Message(locale.Lang(lang), "required"), rule.Field.Name())).Validate(obj)
}

func (rule inRule) RuleInfo() RuleInfo {
	return RuleInfo{Type: "in", Field: rule.Field.Name(), Values: rule.Values, Scenarios: rule.Scenarios}
}

func (rule inRule) Validate(obj interface{}, lang string) error {
	if obj == nil {
		return nil
	}
	stringValues := make([]interface{}, 0, 10)
	for _, validationVal := range rule.Values {
		stringValues = append(stringValues, fmt.Sprintf("%v", validationVal))
	}
	return validation.In(stringValues...).Error(fmt.Sprintf(locale.Message(locale.Lang(lang), "in"), rule.Field.Name(), rule.Values)).Validate(fmt.Sprintf("%v", obj))
}

func (rule emailRule) RuleInfo() RuleInfo {
	return RuleInfo{Type: "email", Field: rule.Field.Name(), Scenarios: rule.Scenarios}
}

func (rule emailRule) Validate(obj interface{}, lang string) error {
	return is.Email.Error(fmt.Sprintf(locale.Message(locale.Lang(lang), "email"), rule.Field.Name())).Validate(obj)
}

func (rule urlRule) RuleInfo() RuleInfo {
	return RuleInfo{Type: "url", Field: rule.Field.Name(), Scenarios: rule.Scenarios}
}

func (rule urlRule) Validate(obj interface{}, lang string) error {
	return is.URL.Error(fmt.Sprintf(locale.Message(locale.Lang(lang), "url"), rule.Field.Name())).Validate(obj)
}

func (rule lengthRule) RuleInfo() RuleInfo {
	return RuleInfo{Type: "length", Field: rule.Field.Name(), Min: rule.Min, Max: rule.Max, Scenarios: rule.Scenarios}
}

func (rule lengthRule) Validate(obj interface{}, lang string) error {
	return validation.Length(
		rule.Min,
		rule.Max,
	).Error(
		fmt.Sprintf(locale.Message(locale.Lang(lang), "length"), rule.Field.Name(), rule.Min, rule.Max),
	).Validate(obj)
}

// ContainsColumn checks if a column is present in the list by name.
func ContainsColumn(columns []pg.Column, target pg.Column) bool {
	targetName := target.Name()
	for _, c := range columns {
		if c.Name() == targetName {
			return true
		}
	}
	return false
}
