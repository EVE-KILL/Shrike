package frontend

import "strings"

func searchPickerPlaceholder(types []string) string {
	names := append([]string(nil), types...)
	for i, name := range names {
		if name == "corporation" {
			names[i] = "corp"
		}
	}
	if len(names) == 0 {
		return "Search..."
	}
	list := names[0]
	if len(names) > 1 {
		list = strings.Join(names[:len(names)-1], ", ") + " or " + names[len(names)-1]
	}
	return "Search " + list + "..."
}

type searchPickerOptions struct {
	Large, LocationImages bool
	TypeImages, Compact   bool
	Limit, Debounce       int
	Placeholder           string
}

func searchPickerLimit(options []searchPickerOptions) int {
	if n := searchPickerConfig(options).Limit; n > 0 && n <= 50 {
		return n
	}
	return 10
}
func searchPickerDebounce(options []searchPickerOptions) int {
	if n := searchPickerConfig(options).Debounce; n > 0 {
		return n
	}
	return 300
}

func searchPickerConfig(options []searchPickerOptions) searchPickerOptions {
	if len(options) > 0 {
		return options[0]
	}
	return searchPickerOptions{}
}
func searchPickerText(types []string, options []searchPickerOptions) string {
	if text := searchPickerConfig(options).Placeholder; text != "" {
		return text
	}
	return searchPickerPlaceholder(types)
}
