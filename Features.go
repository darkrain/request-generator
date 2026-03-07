package module

import "github.com/portalenergy/pe-request-generator/actions"

type Features struct {
	ModuleName string                     `json:"module_name"`
	Actions    map[string]FeaturesActions `json:"actions"`
}

type FeaturesActions struct {
	Label string         `json:"label"`
	Url   string         `json:"url"`
	Type  string         `json:"type"`
	Roles []actions.Role `json:"roles"`
}
