package frontend

import "github.com/a-h/templ"

// Source PageHeader slots, including the optional background and metadata row.
type utilityHeaderView struct {
	Title, Eyebrow, Symbol, Description, Background string
	DescriptionContent, Meta                        templ.Component
}
