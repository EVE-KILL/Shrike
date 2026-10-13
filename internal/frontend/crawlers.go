package frontend

import (
	"encoding/json"
	"encoding/xml"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"

	"github.com/eve-kill/shrike/internal/killtype"
)

const sitemapNamespace = "http://www.sitemaps.org/schemas/sitemap/0.9"

type sitemapEntry struct {
	Location        string `xml:"loc"`
	LastModified    string `xml:"lastmod,omitempty"`
	ChangeFrequency string `xml:"changefreq,omitempty"`
	Priority        string `xml:"priority,omitempty"`
}
type sitemapDocument struct {
	XMLName   xml.Name       `xml:"urlset"`
	Namespace string         `xml:"xmlns,attr"`
	Entries   []sitemapEntry `xml:"url"`
}
type sitemapIndexDocument struct {
	XMLName   xml.Name       `xml:"sitemapindex"`
	Namespace string         `xml:"xmlns,attr"`
	Entries   []sitemapEntry `xml:"sitemap"`
}

var dynamicSitemapKinds = []string{
	"kills", "characters", "coalitions", "corporations", "alliances",
	"systems", "regions", "ships", "items", "wars", "battles",
}

type sitemapSourceEntry struct {
	Location        string  `json:"loc"`
	LastModified    string  `json:"lastmod,omitempty"`
	ChangeFrequency string  `json:"changefreq,omitempty"`
	Priority        float64 `json:"priority,omitempty"`
}

func validSitemapKind(kind string) bool {
	for _, candidate := range dynamicSitemapKinds {
		if kind == candidate {
			return true
		}
	}
	return false
}

func (s *site) readSitemapSource(r *http.Request, kind string) ([]sitemapSourceEntry, error) {
	if !validSitemapKind(kind) || s.options.SitemapAPI == nil {
		return nil, fmt.Errorf("invalid sitemap kind")
	}
	request := r.Clone(r.Context())
	address := *r.URL
	address.Path = "/api/__sitemap__/" + kind
	address.RawPath = ""
	address.RawQuery = ""
	request.URL = &address
	request.Method = http.MethodGet
	request.RequestURI = ""
	request.Host = "eve-kill.com"
	request.Header.Set("X-Forwarded-Host", "eve-kill.com")
	w := httptest.NewRecorder()
	s.options.SitemapAPI.ServeHTTP(w, request)
	if w.Code != http.StatusOK {
		return nil, fmt.Errorf("sitemap API returned %d", w.Code)
	}
	var entries []sitemapSourceEntry
	if err := json.Unmarshal(w.Body.Bytes(), &entries); err != nil {
		return nil, fmt.Errorf("decode sitemap source: %w", err)
	}
	for _, entry := range entries {
		if _, err := canonicalURL(entry.Location); err != nil {
			return nil, fmt.Errorf("invalid sitemap location: %w", err)
		}
	}
	return entries, nil
}

func (s *site) sitemapData(w http.ResponseWriter, r *http.Request) {
	kind := r.PathValue("kind")
	if !validSitemapKind(kind) {
		http.NotFound(w, r)
		return
	}
	entries, err := s.readSitemapSource(r, kind)
	if err != nil {
		http.Error(w, "Could not load sitemap data", http.StatusServiceUnavailable)
		return
	}
	data, err := json.Marshal(entries)
	if err != nil {
		http.Error(w, "Could not encode sitemap data", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Cache-Control", "public, max-age=3600")
	writeCrawlerResponse(w, r, "application/json; charset=utf-8", data)
}

func (s *site) sitemapCategory(w http.ResponseWriter, r *http.Request) {
	if !s.options.Production || !canonicalHost(r.Host) {
		http.NotFound(w, r)
		return
	}
	file := r.PathValue("file")
	if file == "pages.xml" {
		s.sitemap(w, r)
		return
	}
	kind, ok := strings.CutSuffix(file, ".xml")
	if !ok || !validSitemapKind(kind) {
		http.NotFound(w, r)
		return
	}
	source, err := s.readSitemapSource(r, kind)
	if err != nil {
		http.Error(w, "Could not load sitemap", http.StatusServiceUnavailable)
		return
	}
	entries := make([]sitemapEntry, 0, len(source))
	for _, entry := range source {
		location, _ := canonicalURL(entry.Location)
		priority := ""
		if entry.Priority > 0 {
			priority = strconv.FormatFloat(entry.Priority, 'f', 1, 64)
		}
		entries = append(entries, sitemapEntry{
			Location: location, LastModified: entry.LastModified,
			ChangeFrequency: entry.ChangeFrequency, Priority: priority,
		})
	}
	data, err := xml.Marshal(sitemapDocument{Namespace: sitemapNamespace, Entries: entries})
	if err != nil {
		http.Error(w, "Could not render sitemap", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Cache-Control", "public, max-age=3600")
	writeCrawlerResponse(w, r, "application/xml; charset=utf-8", append([]byte(xml.Header), data...))
}

func (s *site) sitemapEntries() []sitemapEntry {
	var entries []sitemapEntry
	seen := map[string]bool{}
	for _, route := range s.pages {
		if !route.Metadata.Indexable {
			continue
		}
		location, err := canonicalURL(route.Metadata.CanonicalPath)
		if err != nil || seen[location] {
			continue
		}
		entries = append(entries, sitemapEntry{Location: location})
		seen[location] = true
	}
	if s.options.Homepage != nil {
		entries = append(entries, sitemapEntry{Location: canonicalOrigin + "/categories"})
		for _, kind := range killtype.Types {
			entries = append(entries, sitemapEntry{Location: canonicalOrigin + "/kills/" + kind})
		}
	}
	if s.options.FittingsAPI != nil {
		entries = append(entries, sitemapEntry{Location: canonicalOrigin + "/fits"}, sitemapEntry{Location: canonicalOrigin + "/fits/search"})
	}
	return entries
}

func (s *site) robots(w http.ResponseWriter, r *http.Request) {
	// Let crawlers read noindex metadata on noncanonical hosts. A robots block
	// would prevent them from seeing that directive. API/auth aren't page routes.
	body := "User-agent: *\nAllow: /\nDisallow: /api/\nDisallow: /auth/\n"
	if s.options.Production && canonicalHost(r.Host) && len(s.sitemapEntries()) > 0 {
		body += "\nSitemap: " + canonicalOrigin + "/sitemap_index.xml\n"
	}
	writeCrawlerResponse(w, r, "text/plain; charset=utf-8", []byte(body))
}

func (s *site) sitemap(w http.ResponseWriter, r *http.Request) {
	if !s.options.Production || !canonicalHost(r.Host) {
		s.pageHandler(func(*http.Request) (Page, error) { return Page{}, ErrNotFound }).ServeHTTP(w, r)
		return
	}
	data, err := xml.Marshal(sitemapDocument{Namespace: sitemapNamespace, Entries: s.sitemapEntries()})
	if err != nil {
		http.Error(w, "Could not render sitemap", 500)
		return
	}
	writeCrawlerResponse(w, r, "application/xml; charset=utf-8", append([]byte(xml.Header), data...))
}

func (s *site) sitemapIndex(w http.ResponseWriter, r *http.Request) {
	if !s.options.Production || !canonicalHost(r.Host) {
		s.pageHandler(func(*http.Request) (Page, error) { return Page{}, ErrNotFound }).ServeHTTP(w, r)
		return
	}
	var entries []sitemapEntry
	if len(s.sitemapEntries()) > 0 {
		entries = []sitemapEntry{{Location: canonicalOrigin + "/__sitemap__/pages.xml"}}
		if s.options.SitemapAPI != nil {
			for _, kind := range dynamicSitemapKinds {
				entries = append(entries, sitemapEntry{Location: canonicalOrigin + "/__sitemap__/" + kind + ".xml"})
			}
		}
	}
	data, err := xml.Marshal(sitemapIndexDocument{Namespace: sitemapNamespace, Entries: entries})
	if err != nil {
		http.Error(w, "Could not render sitemap", 500)
		return
	}
	writeCrawlerResponse(w, r, "application/xml; charset=utf-8", append([]byte(xml.Header), data...))
}

func writeCrawlerResponse(w http.ResponseWriter, r *http.Request, contentType string, data []byte) {
	w.Header().Set("Content-Type", contentType)
	w.Header().Set("Content-Length", strconv.Itoa(len(data)))
	if w.Header().Get("Cache-Control") == "" {
		w.Header().Set("Cache-Control", "public, max-age=0, must-revalidate")
	}
	w.Header().Set("X-Robots-Tag", "noindex")
	if r.Method != http.MethodHead {
		_, _ = w.Write(data)
	}
}
