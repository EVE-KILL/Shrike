package frontend

import "strings"

type dateTimePickerOptions struct {
	Name, Value, Placeholder     string
	Disabled, Compact, HideClear bool
}

func dateTimePickerLabel(value string) string {
	return strings.ReplaceAll(value, "T", " ")
}
