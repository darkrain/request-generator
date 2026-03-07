package fields

import (
	"database/sql"
	"errors"
	"fmt"

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

type ModuleField struct {
	Column               pg.Column                                       `json:"-"`
	SelectExpression     pg.Projection                                   `json:"-"`
	Title                string                                          `json:"title"`
	Type                 ModuleFieldType                                 `json:"type"`
	FormType             ModuleFieldFormType                             `json:"form_type,omitempty"`
	Example              string                                          `json:"example,omitempty"`
	Options              []ModuleFieldOptions                            `json:"options,omitempty"`
	OptionsFunc          func(context *gin.Context) []ModuleFieldOptions `json:"-"`
	Check                []CheckRules                                    `json:"check,omitempty"`
	CheckFunc            func(context *gin.Context) []CheckRules         `json:"-"`
	Convert              func(value interface{}) (interface{}, error)    `json:"-"`
	ResultValueConverter func(value interface{}) interface{}             `json:"-"`
}

// ColumnName returns the database column name from the Jet column.
func (f ModuleField) ColumnName() string {
	if f.Column != nil {
		return f.Column.Name()
	}
	return ""
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
	Value interface{} `json:"value"`
	Label string      `json:"label"`
}

type CheckRules interface {
	Validate(obj interface{}) error
	GetScenarios() []Scenario
}

func RequiredRule(field string, scenarios []Scenario) requiredRule {
	return requiredRule{
		Field:     field,
		Scenarios: scenarios,
	}
}

func InRule(field string, values []interface{}, scenarios []Scenario) inRule {
	return inRule{
		Field:     field,
		Values:    values,
		Scenarios: scenarios,
	}
}

func InDBRule(field string, values func() []interface{}, scenarios []Scenario) inRule {
	return inRule{
		Field:     field,
		Values:    values(),
		Scenarios: scenarios,
	}
}

func LenRule(field string, min int, max int, scenarios []Scenario) lengthRule {
	return lengthRule{
		Min:       min,
		Max:       max,
		Field:     field,
		Scenarios: scenarios,
	}
}

func UrlRule(field string, scenarios []Scenario) urlRule {
	return urlRule{
		Field:     field,
		Scenarios: scenarios,
	}
}

type requiredRule struct {
	CheckRules `json:"-"`
	Field      string     `json:"field"`
	Scenarios  []Scenario `json:"scenarios"`
}

type inRule struct {
	Field     string        `json:"field"`
	Values    []interface{} `json:"values"`
	Scenarios []Scenario    `json:"scenarios"`
}

type emailRule struct {
	CheckRules `json:"-"`
	Type       string     `json:"type"`
	Field      string     `json:"field"`
	Scenarios  []Scenario `json:"scenarios"`
}

type urlRule struct {
	CheckRules `json:"-"`
	Type       string     `json:"type"`
	Field      string     `json:"field"`
	Scenarios  []Scenario `json:"scenarios"`
}

type lengthRule struct {
	CheckRules `json:"-"`
	Type       string     `json:"type"`
	Min        int        `json:"min"`
	Max        int        `json:"max"`
	Field      string     `json:"field"`
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

func (rule requiredRule) Validate(obj interface{}) error {
	return validation.Required.Error(fmt.Sprintf("%s - не может быть пустым", rule.Field)).Validate(obj)
}

func (rule inRule) Validate(obj interface{}) error {
	if obj == nil {
		return nil
	}
	stringValues := make([]interface{}, 0, 10)
	for _, validationVal := range rule.Values {
		stringValues = append(stringValues, fmt.Sprintf("%v", validationVal))
	}
	return validation.In(stringValues...).Error(fmt.Sprintf("%s - должен быть одним из %v", rule.Field, rule.Values)).Validate(fmt.Sprintf("%v", obj))
}

func (rule emailRule) Validate(obj interface{}) error {
	return is.Email.Error(fmt.Sprintf("%s неправильный Email адрес", rule.Field)).Validate(obj)
}

func (rule urlRule) Validate(obj interface{}) error {
	return is.URL.Error(fmt.Sprintf("%s неправильный URL адрес", rule.Field)).Validate(obj)
}

func (rule lengthRule) Validate(obj interface{}) error {
	return validation.Length(
		rule.Min,
		rule.Max,
	).Error(
		fmt.Sprintf("%s должен быть в пределах %v - %v", rule.Field, rule.Min, rule.Max),
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
