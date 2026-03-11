package actions

import pg "github.com/go-jet/jet/v2/postgres"

type SortDirection string

const (
	SortASC  SortDirection = "asc"
	SortDESC SortDirection = "desc"
)

type SortOption struct {
	Column    pg.Column     `json:"-"`
	Direction SortDirection `json:"-"`
}

type SortResponseItem struct {
	Value string `json:"value"`
	Text  string `json:"text"`
}
