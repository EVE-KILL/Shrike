package api

import (
	"bytes"
	"context"
	"encoding/json"
	stdhtml "html"
	"net"
	"net/url"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"time"

	xhtml "golang.org/x/net/html"
	"golang.org/x/net/html/atom"
)

var validDefaultTabs = map[string]map[string]bool{
	"character":   valuesSet("dashboard", "combined", "kills", "losses", "history", "achievements"),
	"corporation": valuesSet("dashboard", "combined", "kills", "losses", "members", "history"),
	"alliance":    valuesSet("dashboard", "combined", "kills", "losses", "corporations", "members"),
	// The TypeScript validator accidentally omitted faction even though the
	// settings control offers it. Accepting the rendered choices is the useful
	// behavior and prevents every faction preference from being discarded.
	"faction":       valuesSet("dashboard", "combined", "kills", "losses"),
	"item":          valuesSet("dashboard", "kills"),
	"system":        valuesSet("dashboard", "kills"),
	"constellation": valuesSet("dashboard", "kills"),
	"region":        valuesSet("dashboard", "kills"),
}

var validThemeKeys = valuesSet(
	"brandPrimary", "brandPrimaryHover", "brandSecondary", "brandAccent",
	"bgPrimary", "bgSecondary", "bgTertiary", "bgHover",
	"textPrimary", "textSecondary", "textTertiary",
	"borderLight", "borderMedium", "borderFocus",
	"surfaceAlpha", "surfaceHover",
	"colorSuccess", "colorWarning", "colorError", "colorInfo",
	"lossBg", "lossHover", "lossBorder",
	"colorHighsec", "colorLowsec", "colorNullsec",
	"scrollbarThumb",
	"iskColor", "npcColor", "selectionBg", "selectionText",
)

var (
	cssColorPattern = regexp.MustCompile(
		`^(#[0-9a-fA-F]{3,8}|rgba?\(\s*[\d.%,\s/]+\)|hsla?\(\s*[\d.%,\s/]+\)|[a-zA-Z]{3,30})$`,
	)
	boardKeyPattern       = regexp.MustCompile(`^[a-z0-9-]+$`)
	boardHostPattern      = regexp.MustCompile(`^[a-z0-9-]+(\.[a-z0-9-]+)+$`)
	markdownLinkPattern   = regexp.MustCompile(`\[([^\]\n]+)\]\(([^)\s]+)\)`)
	markdownCodePattern   = regexp.MustCompile("`([^`\\n]+)`")
	markdownBoldPattern   = regexp.MustCompile(`\*\*([^*\n]+)\*\*`)
	markdownStrikePattern = regexp.MustCompile(`~~([^~\n]+)~~`)
	markdownEmPattern     = regexp.MustCompile(`\*([^*\n]+)\*`)
	eveURLPattern         = regexp.MustCompile(`(?is)<url=([^>]+)>(.*?)</url>`)
	eveBareURLPattern     = regexp.MustCompile(`(?is)<url>(.*?)</url>`)
	eveSizePattern        = regexp.MustCompile(`(?is)<size=["']?([^"'\s>]+)["']?>(.*?)</size>`)
	eveLocPattern         = regexp.MustCompile(`(?is)<loc>(.*?)</loc>`)
	safeColorStyle        = regexp.MustCompile(`(?i)^color:\s*(#[0-9a-f]{3,8}|[a-z]{3,30})\s*;?$`)
	safeFontSizeStyle     = regexp.MustCompile(`(?i)^font-size:\s*([0-9]{1,2})px\s*;?$`)
)

func valuesSet(values ...string) map[string]bool {
	result := make(map[string]bool, len(values))
	for _, value := range values {
		result[value] = true
	}
	return result
}

func sanitizeDefaultTabs(raw any) (map[string]string, bool) {
	input, ok := raw.(map[string]any)
	if !ok || input == nil {
		return nil, false
	}
	result := make(map[string]string)
	for entity, value := range input {
		tab, ok := value.(string)
		if ok && validDefaultTabs[entity][tab] {
			result[entity] = tab
		}
	}
	return result, true
}

func sanitizeTheme(raw any) (map[string]string, bool) {
	input, ok := raw.(map[string]any)
	if !ok || input == nil {
		return nil, false
	}
	result := make(map[string]string)
	for key, value := range input {
		color, ok := value.(string)
		if !ok || !validThemeKeys[key] {
			continue
		}
		color = strings.TrimSpace(color)
		if cssColorPattern.MatchString(color) {
			result[key] = color
		}
	}
	return result, true
}

func sanitizeBoardState(raw any) accountBoardState {
	input, _ := raw.(map[string]any)
	return accountBoardState{
		Pinned:    cleanBoardList(input["pinned"]),
		Dismissed: cleanBoardList(input["dismissed"]),
	}
}

func cleanBoardList(raw any) []string {
	list, ok := raw.([]any)
	if !ok {
		if strings, ok := raw.([]string); ok {
			list = make([]any, len(strings))
			for i := range strings {
				list[i] = strings[i]
			}
		} else {
			return []string{}
		}
	}
	result := make([]string, 0, min(len(list), maxBoardEntries))
	seen := make(map[string]bool)
	for _, item := range list {
		value := strings.ToLower(strings.TrimSpace(stringifyJSONValue(item)))
		if !validBoardKey(value) || seen[value] {
			continue
		}
		seen[value] = true
		result = append(result, value)
		if len(result) == maxBoardEntries {
			break
		}
	}
	return result
}

func stringifyJSONValue(value any) string {
	switch typed := value.(type) {
	case nil:
		return "null"
	case string:
		return typed
	case json.Number:
		return typed.String()
	case bool:
		return strconv.FormatBool(typed)
	case float64:
		return strconv.FormatFloat(typed, 'f', -1, 64)
	default:
		encoded, err := json.Marshal(value)
		if err != nil {
			return ""
		}
		return string(encoded)
	}
}

func validBoardKey(value string) bool {
	if value == "" || len(value) > maxBoardKeyLength {
		return false
	}
	if strings.Contains(value, ".") {
		return boardHostPattern.MatchString(value)
	}
	return boardKeyPattern.MatchString(value)
}

func parseBoardCookie(raw string) accountBoardState {
	if raw == "" {
		return accountBoardState{Pinned: []string{}, Dismissed: []string{}}
	}
	pinned, dismissed, _ := strings.Cut(raw, "|")
	return accountBoardState{
		Pinned:    cleanBoardList(stringsToAny(strings.Split(pinned, ","))),
		Dismissed: cleanBoardList(stringsToAny(strings.Split(dismissed, ","))),
	}
}

func stringsToAny(values []string) []any {
	result := make([]any, len(values))
	for i := range values {
		result[i] = values[i]
	}
	return result
}

func mergeBoardStates(left, right accountBoardState) accountBoardState {
	return accountBoardState{
		Pinned: cleanBoardList(stringsToAny(append(
			append([]string{}, left.Pinned...),
			right.Pinned...,
		))),
		Dismissed: cleanBoardList(stringsToAny(append(
			append([]string{}, left.Dismissed...),
			right.Dismissed...,
		))),
	}
}

func boardDomainKey(domain accountBoardDomain) string {
	if domain.CustomHostname != nil {
		return strings.ToLower(*domain.CustomHostname)
	}
	return strings.ToLower(domain.Subdomain)
}

func boardKeyHost(key string) string {
	if strings.Contains(key, ".") {
		return key
	}
	return key + ".eve-kill.com"
}

func hostBoardKey(raw string) *string {
	host := strings.ToLower(strings.TrimSpace(raw))
	if parsedHost, _, err := net.SplitHostPort(host); err == nil {
		host = parsedHost
	} else if strings.Count(host, ":") == 1 {
		host, _, _ = strings.Cut(host, ":")
	}
	host = strings.TrimSuffix(host, ".")
	switch host {
	case "", "eve-kill.com", "www.eve-kill.com":
		return nil
	}
	const suffix = ".eve-kill.com"
	if before, ok := strings.CutSuffix(host, suffix); ok {
		host = before
	}
	if !validBoardKey(host) {
		return nil
	}
	return &host
}

func truncateRunes(value string, limit int) string {
	runes := []rune(value)
	if len(runes) <= limit {
		return value
	}
	return string(runes[:limit])
}

func renderAccountBio(text, format string) *string {
	text = strings.TrimSpace(text)
	if text == "" {
		return nil
	}
	var rendered string
	if format == "eve_html" {
		rendered = renderEVEHTML(text)
	} else {
		rendered = renderBasicMarkdown(text)
	}
	return &rendered
}

func renderBasicMarkdown(source string) string {
	lines := strings.Split(strings.ReplaceAll(source, "\r\n", "\n"), "\n")
	var output strings.Builder
	var paragraph []string
	inCode := false
	var code []string
	flushParagraph := func() {
		if len(paragraph) == 0 {
			return
		}
		output.WriteString("<p>")
		output.WriteString(renderMarkdownInline(strings.Join(paragraph, "\n")))
		output.WriteString("</p>\n")
		paragraph = paragraph[:0]
	}
	flushCode := func() {
		output.WriteString("<pre><code>")
		output.WriteString(stdhtml.EscapeString(strings.Join(code, "\n")))
		output.WriteString("</code></pre>\n")
		code = code[:0]
	}
	for _, line := range lines {
		if strings.HasPrefix(strings.TrimSpace(line), "```") {
			if inCode {
				flushCode()
			} else {
				flushParagraph()
			}
			inCode = !inCode
			continue
		}
		if inCode {
			code = append(code, line)
			continue
		}
		trimmed := strings.TrimSpace(line)
		if trimmed == "" {
			flushParagraph()
			continue
		}
		if trimmed == "---" || trimmed == "***" {
			flushParagraph()
			output.WriteString("<hr>\n")
			continue
		}
		if level := headingLevel(trimmed); level > 0 {
			flushParagraph()
			body := strings.TrimSpace(trimmed[level:])
			output.WriteString("<h" + strconv.Itoa(level) + ">")
			output.WriteString(renderMarkdownInline(body))
			output.WriteString("</h" + strconv.Itoa(level) + ">\n")
			continue
		}
		if strings.HasPrefix(trimmed, "> ") {
			flushParagraph()
			output.WriteString("<blockquote><p>")
			output.WriteString(renderMarkdownInline(strings.TrimSpace(trimmed[2:])))
			output.WriteString("</p></blockquote>\n")
			continue
		}
		paragraph = append(paragraph, line)
	}
	if inCode {
		flushCode()
	} else {
		flushParagraph()
	}
	return strings.TrimSpace(output.String())
}

func headingLevel(value string) int {
	for level := 6; level >= 1; level-- {
		prefix := strings.Repeat("#", level) + " "
		if strings.HasPrefix(value, prefix) {
			return level
		}
	}
	return 0
}

func renderMarkdownInline(source string) string {
	escaped := stdhtml.EscapeString(source)
	escaped = markdownLinkPattern.ReplaceAllStringFunc(escaped, func(match string) string {
		parts := markdownLinkPattern.FindStringSubmatch(match)
		if len(parts) != 3 {
			return match
		}
		href := stdhtml.UnescapeString(parts[2])
		if !safeBioURL(href) {
			return parts[1]
		}
		return hardenedAnchor(href, parts[1])
	})
	escaped = markdownCodePattern.ReplaceAllString(escaped, "<code>$1</code>")
	escaped = markdownBoldPattern.ReplaceAllString(escaped, "<strong>$1</strong>")
	escaped = markdownStrikePattern.ReplaceAllString(escaped, "<del>$1</del>")
	escaped = markdownEmPattern.ReplaceAllString(escaped, "<em>$1</em>")
	return strings.ReplaceAll(escaped, "\n", "<br>\n")
}

func renderEVEHTML(source string) string {
	translated := eveURLPattern.ReplaceAllString(source, `<a href="$1">$2</a>`)
	translated = eveBareURLPattern.ReplaceAllString(translated, `<a href="$1">$1</a>`)
	translated = eveSizePattern.ReplaceAllString(
		translated,
		`<span style="font-size: $1px">$2</span>`,
	)
	translated = eveLocPattern.ReplaceAllString(translated, "$1")
	translated = strings.ReplaceAll(translated, "</color>", "</span>")
	translated = strings.ReplaceAll(translated, "\r\n", "\n")
	translated = strings.ReplaceAll(translated, "\r", "\n")
	translated = strings.ReplaceAll(translated, "\n", "<br>")

	contextNode := &xhtml.Node{Type: xhtml.ElementNode, Data: "div", DataAtom: atom.Div}
	nodes, err := xhtml.ParseFragment(strings.NewReader(translated), contextNode)
	if err != nil {
		return strings.ReplaceAll(stdhtml.EscapeString(source), "\n", "<br>")
	}
	var output bytes.Buffer
	for _, node := range nodes {
		if clean := sanitizeBioNode(node); clean != nil {
			_ = xhtml.Render(&output, clean)
		}
	}
	return output.String()
}

var allowedBioElements = valuesSet(
	"a", "b", "i", "em", "strong", "u", "s", "del", "code", "pre",
	"br", "hr", "p", "blockquote", "ul", "ol", "li", "h1", "h2",
	"h3", "h4", "h5", "h6", "table", "thead", "tbody", "tr", "th",
	"td", "img", "div", "span", "font",
)

func sanitizeBioNode(node *xhtml.Node) *xhtml.Node {
	switch node.Type {
	case xhtml.TextNode:
		return &xhtml.Node{Type: xhtml.TextNode, Data: node.Data}
	case xhtml.ElementNode:
		tag := strings.ToLower(node.Data)
		if tag == "script" || tag == "style" || tag == "iframe" ||
			tag == "form" || tag == "object" || tag == "embed" {
			return nil
		}
		if !allowedBioElements[tag] {
			wrapper := &xhtml.Node{Type: xhtml.ElementNode, Data: "span", DataAtom: atom.Span}
			appendSanitizedChildren(wrapper, node)
			return wrapper
		}
		if tag == "font" {
			tag = "span"
		}
		clean := &xhtml.Node{Type: xhtml.ElementNode, Data: tag}
		for _, attr := range node.Attr {
			name, value := strings.ToLower(attr.Key), strings.TrimSpace(attr.Val)
			switch {
			case node.Data == "font" && name == "color" && cssColorPattern.MatchString(value):
				if strings.HasPrefix(value, "#") && len(value) == 9 {
					value = value[:7]
				}
				clean.Attr = append(clean.Attr, xhtml.Attribute{
					Key: "style", Val: "color: " + value,
				})
			case tag == "span" && name == "style":
				if style := sanitizeBioStyle(value); style != "" {
					clean.Attr = append(clean.Attr, xhtml.Attribute{Key: "style", Val: style})
				}
			case tag == "a" && name == "href":
				if rewritten := rewriteEVEHref(value); safeBioURL(rewritten) {
					clean.Attr = append(clean.Attr, xhtml.Attribute{Key: "href", Val: rewritten})
				}
			case (name == "title" || name == "class" || name == "lang") &&
				len(value) <= 500:
				clean.Attr = append(clean.Attr, xhtml.Attribute{Key: name, Val: value})
			case tag == "img" && name == "src" && safeBioURL(value):
				clean.Attr = append(clean.Attr, xhtml.Attribute{Key: "src", Val: value})
			case tag == "img" && (name == "alt" || name == "loading") && len(value) <= 500:
				clean.Attr = append(clean.Attr, xhtml.Attribute{Key: name, Val: value})
			case (tag == "td" || tag == "th") && (name == "colspan" || name == "rowspan"):
				if number, err := strconv.Atoi(value); err == nil && number >= 1 && number <= 100 {
					clean.Attr = append(clean.Attr, xhtml.Attribute{Key: name, Val: value})
				}
			}
		}
		if tag == "a" {
			href := nodeAttribute(clean, "href")
			if href == "" {
				clean.Attr = nil
			} else if !strings.HasPrefix(href, "/") && !strings.HasPrefix(href, "#") {
				clean.Attr = append(clean.Attr,
					xhtml.Attribute{Key: "target", Val: "_blank"},
					xhtml.Attribute{Key: "rel", Val: "noopener noreferrer nofollow"},
				)
			} else {
				clean.Attr = append(clean.Attr,
					xhtml.Attribute{Key: "rel", Val: "noopener noreferrer nofollow"},
				)
			}
		}
		appendSanitizedChildren(clean, node)
		return clean
	default:
		return nil
	}
}

func appendSanitizedChildren(target, source *xhtml.Node) {
	for child := source.FirstChild; child != nil; child = child.NextSibling {
		if clean := sanitizeBioNode(child); clean != nil {
			target.AppendChild(clean)
		}
	}
}

func nodeAttribute(node *xhtml.Node, name string) string {
	for _, attr := range node.Attr {
		if attr.Key == name {
			return attr.Val
		}
	}
	return ""
}

func sanitizeBioStyle(value string) string {
	value = strings.TrimSpace(value)
	if safeColorStyle.MatchString(value) {
		return strings.TrimSuffix(value, ";")
	}
	match := safeFontSizeStyle.FindStringSubmatch(value)
	if len(match) == 2 {
		size, _ := strconv.Atoi(match[1])
		if size >= 1 && size <= 48 {
			return "font-size: " + strconv.Itoa(size) + "px"
		}
	}
	return ""
}

func rewriteEVEHref(value string) string {
	for prefix, local := range map[string]string{
		"killReport:": "/kill/",
		"warReport:":  "/war/",
	} {
		if after, ok := strings.CutPrefix(value, prefix); ok {
			id, _, _ := strings.Cut(after, ":")
			if _, err := strconv.ParseInt(id, 10, 64); err == nil {
				return local + id
			}
		}
	}
	if after, ok := strings.CutPrefix(value, "showinfo:"); ok {
		parts := strings.Split(after, "//")
		typeID, err := strconv.ParseInt(parts[0], 10, 64)
		if err != nil {
			return ""
		}
		if len(parts) == 1 {
			return "/item/" + strconv.FormatInt(typeID, 10)
		}
		id, err := strconv.ParseInt(parts[1], 10, 64)
		if err != nil {
			return ""
		}
		prefix := "/item/"
		switch typeID {
		case 2:
			prefix = "/corporation/"
		case 3:
			prefix = "/region/"
		case 4:
			prefix = "/constellation/"
		case 5:
			prefix = "/system/"
		case 16159:
			prefix = "/alliance/"
		}
		return prefix + strconv.FormatInt(id, 10)
	}
	return value
}

func safeBioURL(value string) bool {
	value = strings.TrimSpace(stdhtml.UnescapeString(value))
	if strings.HasPrefix(value, "/") && !strings.HasPrefix(value, "//") {
		return true
	}
	if strings.HasPrefix(value, "#") {
		return true
	}
	parsed, err := url.Parse(value)
	if err != nil {
		return false
	}
	if parsed.Scheme == "mailto" {
		return parsed.Opaque != "" || parsed.Path != ""
	}
	return parsed.Host != "" &&
		(parsed.Scheme == "https" || parsed.Scheme == "http")
}

func hardenedAnchor(href, text string) string {
	href = stdhtml.EscapeString(href)
	if strings.HasPrefix(href, "/") || strings.HasPrefix(href, "#") {
		return `<a href="` + href + `" rel="noopener noreferrer nofollow">` + text + `</a>`
	}
	return `<a href="` + href + `" target="_blank" rel="noopener noreferrer nofollow">` +
		text + `</a>`
}

func (s *accountService) ephemeralAnnouncements(
	ctx context.Context,
	now time.Time,
) []map[string]any {
	if s.cache == nil {
		return []map[string]any{}
	}
	var cursor uint64
	var keys []string
	for {
		batch, next, err := s.cache.Scan(
			ctx, cursor, "announcements:ephemeral:*", 100,
		).Result()
		if err != nil {
			return []map[string]any{}
		}
		keys = append(keys, batch...)
		cursor = next
		if cursor == 0 {
			break
		}
	}
	if len(keys) == 0 {
		return []map[string]any{}
	}
	sort.Strings(keys)
	values, err := s.cache.MGet(ctx, keys...).Result()
	if err != nil {
		return []map[string]any{}
	}
	result := make([]map[string]any, 0, len(values))
	for _, value := range values {
		raw, ok := value.(string)
		if !ok || raw == "" {
			continue
		}
		var item map[string]any
		if json.Unmarshal([]byte(raw), &item) != nil {
			continue
		}
		start, startOK := timeValue(item["starts_at"])
		expires, expiresOK := timeValue(item["expires_at"])
		if !startOK || !expiresOK || start.After(now) || !expires.After(now) {
			continue
		}
		result = append(result, item)
	}
	return result
}

func timeValue(raw any) (time.Time, bool) {
	switch value := raw.(type) {
	case time.Time:
		return value, true
	case string:
		parsed, err := time.Parse(time.RFC3339Nano, value)
		return parsed, err == nil
	default:
		return time.Time{}, false
	}
}
