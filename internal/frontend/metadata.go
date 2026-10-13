package frontend

import (
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"net/url"
	"strings"
)

const canonicalOrigin = "https://eve-kill.com"
const analyticsOrigin = "https://analytics.karbowiak.dk"
const analyticsWebsiteID = "ee10fd70-9e08-4512-9e36-2f2c4c8145ff"
const defaultDescription = "EVE Killboard — the community-driven EVE Online killboard. Real-time killmails, top pilots, corporations, alliances, and battle reports across New Eden."
const contentSecurityPolicy = "default-src 'self'; style-src 'self' 'unsafe-inline'; img-src 'self' https://images.evetech.net; connect-src 'self' ws: wss:; object-src 'none'; base-uri 'self'; frame-ancestors 'none'"

// CommentInput accepts direct image/video URLs; the API sanitizes their schemes.
// Keep those media usable without broadening script or connection permissions.
const commentMediaPolicy = " https: http:; media-src 'self' https: http:; frame-src https://youtube.com https://www.youtube.com https://player.vimeo.com"

// Metadata belongs to the page loader. CanonicalPath is deliberate: include
// meaningful pagination queries, omit tracking queries, and choose entity tab
// canonicals in the route that understands their meaning. Never use Host.
type Metadata struct {
	Title         string
	OGTitle       string
	Description   string
	CanonicalPath string
	Image         string
	ImageAlt      string
	// NoSocialImage suppresses the default fallback for preview modes that
	// intentionally emit no Open Graph or Twitter image.
	NoSocialImage bool
	TwitterCard   string
	TwitterTitle  string
	ThemeColor    string
	ImageWidth    int
	ImageHeight   int
	ImageSources  []string
	CommentMedia  bool
	CommentEditor bool
	Kind          string
	// Indexing is opt-in per page, and only enabled on the production apex.
	Indexable bool
	// StructuredData is encoded as JSON, including HTML-safe string escaping.
	StructuredData any
	// Page-specific assets are referenced by logical name and versioned here.
	Stylesheets   []string
	ModuleScripts []string
	NeedsWASM     bool
}

type pageMetadata struct {
	Title, OGTitle, Description, CanonicalURL, Image, ImageAlt, Kind, Robots string
	TwitterCard, TwitterTitle, ThemeColor                                    string
	ImageWidth, ImageHeight                                                  int
	StructuredData                                                           json.RawMessage
	CSP                                                                      string
	Stylesheets, ModuleScripts                                               []string
	CommentEditor                                                            bool
	AnalyticsHost                                                            string
}

func canonicalURL(relative string) (string, error) {
	u, err := url.Parse(relative)
	if err != nil || !strings.HasPrefix(relative, "/") || strings.HasPrefix(relative, "//") ||
		u == nil || u.IsAbs() || u.Host != "" || u.Fragment != "" || strings.ContainsAny(relative, "\\\r\n") {
		return "", fmt.Errorf("invalid canonical path %q", relative)
	}
	return canonicalOrigin + u.String(), nil
}

func canonicalHost(host string) bool {
	if name, port, err := net.SplitHostPort(host); err == nil {
		if port != "443" && port != "80" {
			return false
		}
		host = name
	}
	return strings.EqualFold(strings.TrimSuffix(host, "."), "eve-kill.com")
}

func resolveMetadata(r *http.Request, production bool, status int, meta Metadata) (pageMetadata, error) {
	result := pageMetadata{
		Title: "EVE-KILL.com", Description: meta.Description, Image: meta.Image,
		ImageAlt: meta.ImageAlt, Kind: meta.Kind, Robots: "noindex, nofollow", CSP: contentSecurityPolicy,
		ThemeColor: "#000000",
		ImageWidth: meta.ImageWidth, ImageHeight: meta.ImageHeight,
	}
	if len(meta.ImageSources) > 0 {
		seen := map[string]bool{"https://images.evetech.net": true}
		origins := make([]string, 0, len(meta.ImageSources))
		for _, raw := range meta.ImageSources {
			candidate, err := url.Parse(strings.TrimSpace(raw))
			if err != nil || candidate.Scheme != "https" || candidate.Host == "" || candidate.User != nil {
				continue
			}
			origin := "https://" + strings.ToLower(candidate.Host)
			if !seen[origin] {
				seen[origin] = true
				origins = append(origins, origin)
			}
			if len(origins) == 32 {
				break
			}
		}
		if len(origins) > 0 {
			result.CSP = strings.Replace(result.CSP, "; connect-src", " "+strings.Join(origins, " ")+"; connect-src", 1)
		}
	}
	if meta.CommentMedia || meta.CommentEditor {
		result.CSP = strings.Replace(result.CSP, "; connect-src", commentMediaPolicy+"; connect-src", 1)
	}
	stylesheets := append([]string(nil), meta.Stylesheets...)
	if meta.CommentEditor {
		result.CommentEditor = true
		stylesheets = append(stylesheets, "utilities.css", "comment-input.css")
	}
	seenStyles := make(map[string]bool)
	for _, name := range stylesheets {
		if seenStyles[name] {
			continue
		}
		seenStyles[name] = true
		result.Stylesheets = append(result.Stylesheets, assetURL(name))
	}
	for _, name := range meta.ModuleScripts {
		result.ModuleScripts = append(result.ModuleScripts, assetURL(name))
	}
	if meta.Title != "" {
		result.Title = meta.Title + " - EVE-KILL.com"
	}
	result.OGTitle = result.Title
	if meta.OGTitle != "" {
		result.OGTitle = meta.OGTitle
	}
	result.TwitterTitle = result.OGTitle
	if rgb, ok := parsePaletteColor(meta.ThemeColor); ok {
		result.ThemeColor = fmt.Sprintf("#%02x%02x%02x", accentByte(rgb[0]), accentByte(rgb[1]), accentByte(rgb[2]))
	}
	if meta.TwitterTitle != "" {
		result.TwitterTitle = meta.TwitterTitle
	}
	result.TwitterCard = meta.TwitterCard
	if result.TwitterCard == "" && !meta.NoSocialImage {
		result.TwitterCard = "summary_large_image"
	}
	if result.Description == "" {
		result.Description = defaultDescription
	}
	if result.Kind == "" {
		result.Kind = "website"
	}
	if result.Image == "" && !meta.NoSocialImage {
		result.Image = assetURL("og-default.png")
		if result.ImageAlt == "" {
			result.ImageAlt = "EVE-KILL — EVE Online Killboard"
		}
	}
	if result.Image == "" {
		// An image-free preview has no URL to validate.
	} else if strings.HasPrefix(result.Image, "/") {
		var err error
		result.Image, err = canonicalURL(result.Image)
		if err != nil {
			return pageMetadata{}, err
		}
	} else {
		u, err := url.Parse(result.Image)
		if err != nil || u.Scheme != "https" || u.Host == "" || u.User != nil {
			return pageMetadata{}, fmt.Errorf("invalid social image URL")
		}
	}
	if status == http.StatusOK && meta.CanonicalPath != "" {
		var err error
		result.CanonicalURL, err = canonicalURL(meta.CanonicalPath)
		if err != nil {
			return pageMetadata{}, err
		}
	}
	if production && canonicalHost(r.Host) && status == http.StatusOK && meta.Indexable && result.CanonicalURL != "" {
		result.Robots = "index, follow"
	}
	if meta.StructuredData != nil && status == http.StatusOK {
		data, err := json.Marshal(meta.StructuredData)
		if err != nil {
			return pageMetadata{}, fmt.Errorf("encode structured data: %w", err)
		}
		result.StructuredData = data
		// templ.JSONScript uses json.Encoder, which appends a newline. A hash
		// permits this exact JSON-LD without per-request nonces or unsafe-inline.
		digest := sha256.Sum256(append(append([]byte(nil), data...), '\n'))
		result.CSP += "; script-src 'self'"
		if meta.NeedsWASM {
			result.CSP += " 'wasm-unsafe-eval'"
		}
		result.CSP += " 'sha256-" + base64.StdEncoding.EncodeToString(digest[:]) + "'"
	} else if meta.NeedsWASM {
		result.CSP += "; script-src 'self' 'wasm-unsafe-eval'"
	}
	if production {
		host := strings.ToLower(strings.TrimSuffix(r.Host, "."))
		if name, _, err := net.SplitHostPort(host); err == nil {
			host = strings.TrimSuffix(name, ".")
		}
		if host != "" && host != "localhost" && !strings.HasSuffix(host, ".localhost") && net.ParseIP(host) == nil {
			result.AnalyticsHost = host
			result.CSP = strings.Replace(result.CSP, "connect-src 'self'", "connect-src 'self' "+analyticsOrigin, 1)
			if strings.Contains(result.CSP, "; script-src ") {
				result.CSP = strings.Replace(result.CSP, "script-src 'self'", "script-src 'self' "+analyticsOrigin, 1)
			} else {
				result.CSP += "; script-src 'self' " + analyticsOrigin
			}
		}
	}
	return result, nil
}
