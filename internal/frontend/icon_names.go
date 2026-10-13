package frontend

import "regexp"

var commonIconNames = func() map[string]bool {
	result := map[string]bool{}
	for _, match := range regexp.MustCompile(`<symbol id="([^"]+)"`).FindAllSubmatch(manifest.files["icons.svg"].data, -1) {
		result[string(match[1])] = true
	}
	return result
}()
