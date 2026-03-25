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

type ModuleField struct {
	Column               pg.Column                                       `json:"-"`
	SelectExpression     pg.Projection                                   `json:"-"`
	Title                string                                          `json:"title"`
	Titles               map[string]string                               `json:"-"`
	Type                 ModuleFieldType                                 `json:"type"`
	FormType             ModuleFieldFormType                             `json:"form_type,omitempty"`
	Example              string                                          `json:"example,omitempty"`
	Extra                *FieldExtra                                     `json:"-"`
	Options              []ModuleFieldOptions                            `json:"options,omitempty"`
	OptionsFunc          func(context *gin.Context) []ModuleFieldOptions `json:"-"`
	RoleOptions          []RoleOptions                                   `json:"-"`
	Check                []CheckRules                                    `json:"-"`
	CheckFunc            func(context *gin.Context) []CheckRules         `json:"-"`
	RoleCheck            []RoleCheck                                     `json:"-"`
	Convert              func(value interface{}) (interface{}, error)    `json:"-"`
	ResultValueConverter func(value interface{}) interface{}             `json:"-"`
	Translatable         bool                                            `json:"-"`
	FieldName            string                                          `json:"-"`
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
	Column   pg.Column                                    `json:"-"`
	Title    string                                       `json:"title"`
	Titles   map[string]string                            `json:"-"`
	Type     ModuleFieldType                              `json:"type"`
	FormType ModuleFieldFormType                          `json:"form_type,omitempty"`
	Example  string                                       `json:"example,omitempty"`
	Options  []ModuleFieldOptions                         `json:"options,omitempty"`
	Check    []CheckRules                                 `json:"-"`
	Convert  func(value interface{}) (interface{}, error) `json:"-"`
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

// RuleInfo holds validation rule metadata for OpenAPI spec generation.
type RuleInfo struct {
	Type      string        // "required", "in", "length", "url", "email"
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
