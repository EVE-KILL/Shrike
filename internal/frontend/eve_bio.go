package frontend

import (
	"fmt"
	"html"
	"regexp"
	"strconv"
	"strings"

	xhtml "golang.org/x/net/html"
	"golang.org/x/net/html/atom"
)

var eveUnicodeEscape = regexp.MustCompile(`\\u([0-9a-fA-F]{4})`)

func entityBioHTML(custom, description string) string {
	if custom != "" {
		// The API's renderEntityBio strips raw HTML and constructs only its
		// allowlisted anchors, emphasis, paragraphs, and line breaks.
		return custom
	}
	if description == "" {
		return ""
	}
	return renderEVEBio(description)
}

func renderEVEBio(raw string) string {
	raw = strings.TrimSpace(raw)
	if len(raw) >= 3 && raw[0] == 'u' && (raw[1] == '\'' || raw[1] == '"') && raw[len(raw)-1] == raw[1] {
		raw = raw[2 : len(raw)-1]
	}
	raw = strings.ReplaceAll(raw, `\'`, `'`)
	raw = eveUnicodeEscape.ReplaceAllStringFunc(raw, func(match string) string {
		code, err := strconv.ParseUint(match[2:], 16, 16)
		if err != nil {
			return match
		}
		return string(rune(code))
	})
	context := &xhtml.Node{Type: xhtml.ElementNode, DataAtom: atom.Div, Data: "div"}
	nodes, err := xhtml.ParseFragment(strings.NewReader(raw), context)
	if err != nil {
		return html.EscapeString(raw)
	}
	var out strings.Builder
	for _, node := range nodes {
		writeEVENode(&out, node)
	}
	return out.String()
}

func writeEVENode(out *strings.Builder, node *xhtml.Node) {
	switch node.Type {
	case xhtml.TextNode:
		text := html.EscapeString(node.Data)
		text = strings.ReplaceAll(strings.ReplaceAll(text, "\r\n", "\n"), "\n", "<br>")
		out.WriteString(text)
	case xhtml.ElementNode:
		tag := strings.ToLower(node.Data)
		if tag == "script" || tag == "style" || tag == "iframe" || tag == "svg" || tag == "object" {
			return
		}
		if tag == "br" {
			out.WriteString("<br>")
			return
		}
		open, close := "", ""
		switch tag {
		case "b", "strong":
			open, close = "<strong>", "</strong>"
		case "i", "em":
			open, close = "<em>", "</em>"
		case "u":
			open, close = "<u>", "</u>"
		case "font", "color", "size":
			classes := eveFontClasses(node)
			if len(classes) > 0 {
				open, close = `<span class="`+strings.Join(classes, " ")+`">`, "</span>"
			}
		case "a":
			if href := eveLink(node); href != "" {
				open = `<a href="` + html.EscapeString(href) + `"`
				if strings.HasPrefix(href, "https://") || strings.HasPrefix(href, "http://") {
					open += ` target="_blank" rel="noopener noreferrer nofollow"`
				}
				open += `>`
				close = "</a>"
			}
		case "p", "div":
			open, close = "<p>", "</p>"
		}
		out.WriteString(open)
		for child := node.FirstChild; child != nil; child = child.NextSibling {
			writeEVENode(out, child)
		}
		out.WriteString(close)
	}
}

func eveFontClasses(node *xhtml.Node) []string {
	classes := []string{}
	for _, attr := range node.Attr {
		switch attr.Key {
		case "size":
			if size, err := strconv.Atoi(attr.Val); err == nil {
				switch {
				case size >= 23:
					classes = append(classes, "eve-bio-xl")
				case size >= 17:
					classes = append(classes, "eve-bio-lg")
				case size <= 10:
					classes = append(classes, "eve-bio-sm")
				}
			}
		case "color":
			value := strings.ToLower(strings.TrimPrefix(attr.Val, "#"))
			if len(value) == 8 {
				// Match Nuxt's EVE parser: the first six digits are RGB and
				// the final pair is the transparency channel.
				value = value[:6]
			}
			switch value {
			case "ff00ff":
				classes = append(classes, "eve-bio-magenta")
			case "ff7f7f":
				classes = append(classes, "eve-bio-coral")
			case "ffff00":
				classes = append(classes, "eve-bio-yellow")
			case "ff007f":
				classes = append(classes, "eve-bio-pink")
			case "ff7f00":
				classes = append(classes, "eve-bio-orange")
			case "ffd98d":
				classes = append(classes, "eve-bio-peach")
			}
		}
	}
	return classes
}

func eveLink(node *xhtml.Node) string {
	var raw string
	for _, attr := range node.Attr {
		if attr.Key == "href" {
			raw = strings.TrimSpace(attr.Val)
			break
		}
	}
	if strings.HasPrefix(raw, "killReport:") || strings.HasPrefix(raw, "warReport:") {
		parts := strings.Split(raw, ":")
		if len(parts) >= 2 && digits(parts[1]) {
			if parts[0] == "killReport" {
				return "/kill/" + parts[1]
			}
			return "/war/" + parts[1]
		}
	}
	if strings.HasPrefix(raw, "showinfo:") {
		parts := strings.SplitN(strings.TrimPrefix(raw, "showinfo:"), "//", 2)
		if len(parts) == 1 && digits(parts[0]) {
			return "/item/" + parts[0]
		}
		if len(parts) == 2 && digits(parts[0]) && digits(parts[1]) {
			kind := map[string]string{"2": "corporation", "3": "region", "4": "constellation", "5": "system", "16159": "alliance"}[parts[0]]
			if kind != "" {
				return fmt.Sprintf("/%s/%s", kind, parts[1])
			}
			return "/item/" + parts[1]
		}
	}
	if strings.HasPrefix(raw, "/") && !strings.HasPrefix(raw, "//") && !strings.ContainsAny(raw, "\\\r\n") {
		return raw
	}
	if strings.HasPrefix(raw, "https://") || strings.HasPrefix(raw, "http://") {
		return raw
	}
	return ""
}

func digits(value string) bool {
	if value == "" {
		return false
	}
	for _, char := range value {
		if char < '0' || char > '9' {
			return false
		}
	}
	return true
}
