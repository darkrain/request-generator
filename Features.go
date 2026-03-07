package module

import "github.com/portalenergy/pe-request-generator/actions"

type Features struct {
	ModuleName       string                     `json:"module_name"`
	ModuleNameLabels map[string]string          `json:"-"`
	Actions          map[string]FeaturesActions `json:"actions"`
}

type FeaturesActions struct {
	Label  string            `json:"label"`
	Labels map[string]string `json:"-"`
	Url    string            `json:"url"`
	Type   string            `json:"type"`
	Roles  []actions.Role    `json:"roles"`
}

type FeaturesResponse struct {
	Locale  string                              `json:"locale"`
	Locales []string                            `json:"locales"`
	Modules []Features                          `json:"modules"`
	I18n    map[string]map[string]FeaturesI18n  `json:"i18n"`
}

type FeaturesI18n struct {
	ModuleName string            `json:"module_name"`
	Actions    map[string]string `json:"actions"`
}
