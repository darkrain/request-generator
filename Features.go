package module

import "github.com/darkrain/request-generator/actions"

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
	Modules []Features `json:"modules"`
}
