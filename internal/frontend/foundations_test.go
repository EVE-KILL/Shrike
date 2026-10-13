package frontend

import (
	"context"
	"crypto/sha256"
	"encoding/base64"
	"encoding/xml"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"regexp"
	"strings"
	"testing"
	"time"

	"github.com/a-h/templ"
)

func TestMetadataIndexingPolicy(t *testing.T) {
	for _, tt := range []struct {
		name, host            string
		production, indexable bool
		status                int
		want                  string
	}{
		{"production apex", "eve-kill.com", true, true, 200, "index, follow"},
		{"canonical host normalization", "EVE-KILL.COM.:443", true, true, 200, "index, follow"},
		{"development", "eve-kill.com", false, true, 200, "noindex, nofollow"},
		{"localhost", "localhost:4000", true, true, 200, "noindex, nofollow"},
		{"tenant", "void.eve-kill.com", true, true, 200, "noindex, nofollow"},
		{"unported page", "eve-kill.com", true, false, 200, "noindex, nofollow"},
		{"missing page", "eve-kill.com", true, true, 404, "noindex, nofollow"},
	} {
		t.Run(tt.name, func(t *testing.T) {
			r := httptest.NewRequest("GET", "http://"+tt.host+"/kills?utm_source=mail", nil)
			r.Header.Set("X-Forwarded-Host", "eve-kill.com")
			meta, err := resolveMetadata(r, tt.production, tt.status, Metadata{Title: "Kills", CanonicalPath: "/kills?page=2", Indexable: tt.indexable})
			if err != nil {
				t.Fatal(err)
			}
			if meta.Robots != tt.want {
				t.Fatalf("robots = %q, want %q", meta.Robots, tt.want)
			}
			wantURL := canonicalOrigin + "/kills?page=2"
			if tt.status != 200 {
				wantURL = ""
			}
			if meta.CanonicalURL != wantURL {
				t.Fatalf("canonical = %q, want %q", meta.CanonicalURL, wantURL)
			}
		})
	}
}

func TestCanonicalRejectsExternalAndMalformedURLs(t *testing.T) {
	for _, value := range []string{"", "https://other.test/", "//other.test/", "/\\other.test", "/x#fragment", "/%zz", "/x\n"} {
		if _, err := canonicalURL(value); err == nil {
			t.Errorf("accepted %q", value)
		}
	}
}

func TestStructuredDataAndMetadataEscaping(t *testing.T) {
	s := &site{options: Options{Production: true}}
	payload := `</script><script>alert("injected")</script>`
	h := s.pageHandler(func(*http.Request) (Page, error) {
		return Page{Metadata: Metadata{Title: payload, Description: `" onload="bad`, CanonicalPath: "/", Indexable: true,
			StructuredData: map[string]any{"@context": "https://schema.org", "@type": "WebSite", "name": payload}}, Content: home()}, nil
	})
	w := httptest.NewRecorder()
	h.ServeHTTP(w, httptest.NewRequest("GET", canonicalOrigin+"/", nil))
	if w.Code != 200 {
		t.Fatalf("status = %d: %s", w.Code, w.Body)
	}
	if strings.Contains(w.Body.String(), payload) {
		t.Fatal("unescaped HTML in metadata")
	}
	re := regexp.MustCompile(`(?s)<script id="structured-data" type="application/ld\+json">(.*?)</script>`)
	match := re.FindStringSubmatch(w.Body.String())
	if len(match) != 2 {
		t.Fatal("missing structured data")
	}
	sum := sha256.Sum256([]byte(match[1]))
	wantHash := "'sha256-" + base64.StdEncoding.EncodeToString(sum[:]) + "'"
	if !strings.Contains(w.Header().Get("Content-Security-Policy"), wantHash) {
		t.Fatal("CSP does not match rendered JSON-LD")
	}
	if !strings.Contains(w.Header().Get("Content-Security-Policy"), "style-src 'self' 'unsafe-inline'") {
		t.Fatal("CSP blocks rendered dynamic styles")
	}
	if !strings.Contains(w.Body.String(), `content="index, follow"`) || w.Header().Get("X-Robots-Tag") != "index, follow" {
		t.Fatal("HTML and header indexing policy disagree")
	}
}

func TestRenderFailureDoesNotLeakPartialContent(t *testing.T) {
	s := &site{}
	h := s.pageHandler(func(*http.Request) (Page, error) {
		return Page{Content: templ.ComponentFunc(func(_ context.Context, w io.Writer) error {
			_, _ = io.WriteString(w, "private partial content")
			return errors.New("internal database detail")
		})}, nil
	})
	w := httptest.NewRecorder()
	h.ServeHTTP(w, httptest.NewRequest("GET", "http://localhost/", nil))
	if w.Code != 500 || !strings.Contains(w.Body.String(), "Something went wrong") {
		t.Fatalf("response = %d %s", w.Code, w.Body)
	}
	for _, secret := range []string{"private partial content", "internal database detail"} {
		if strings.Contains(w.Body.String(), secret) {
			t.Errorf("response leaked %q", secret)
		}
	}
	if w.Header().Get("Cache-Control") != "private, no-store" || w.Header().Get("X-Robots-Tag") != "noindex, nofollow" {
		t.Fatal("error page may be cached or indexed")
	}
}

func TestPageHEADAndCachePolicy(t *testing.T) {
	s := &site{options: Options{Production: true}}
	h := s.pageHandler(func(*http.Request) (Page, error) { return Page{Content: home(), SharedCacheTTL: time.Minute}, nil })
	for _, tt := range []struct {
		name, origin, header string
		setCookie            bool
		shared               bool
	}{
		{"anonymous", canonicalOrigin, "", false, true},
		{"browser preferences", canonicalOrigin, "Cookie", false, false},
		{"authorization", canonicalOrigin, "Authorization", false, false},
		{"response sets cookie", canonicalOrigin, "", true, false},
		{"tenant", "https://board.eve-kill.com", "", false, false},
		{"preview", "http://localhost:4200", "", false, false},
	} {
		r := httptest.NewRequest("HEAD", tt.origin+"/", nil)
		if tt.header != "" {
			r.Header.Set(tt.header, "session=private")
		}
		w := httptest.NewRecorder()
		w.Header().Set("Vary", "Accept-Encoding")
		if tt.setCookie {
			w.Header().Set("Set-Cookie", "fixture=value; HttpOnly")
		}
		h.ServeHTTP(w, r)
		if w.Body.Len() != 0 || w.Header().Get("Content-Length") == "" {
			t.Fatal("HEAD returned a body or omitted content length")
		}
		want := "public, max-age=0, s-maxage=60, stale-while-revalidate=60"
		if !tt.shared {
			want = "private, no-store"
		}
		if w.Header().Get("Cache-Control") != want {
			t.Errorf("%s: cache = %q", tt.name, w.Header().Get("Cache-Control"))
		}
		vary := strings.Join(w.Header().Values("Vary"), ", ")
		if !strings.Contains(vary, "Accept-Encoding") {
			t.Errorf("%s: previous cache variation lost: %q", tt.name, vary)
		}
		if tt.origin == canonicalOrigin && (!strings.Contains(vary, "Cookie") || !strings.Contains(vary, "Authorization")) {
			t.Errorf("%s: anonymous cached HTML can be reused for personalized requests: %q", tt.name, vary)
		}
	}
}

func TestVersionedAssetsAndConditionalRequests(t *testing.T) {
	h := New(Options{})
	for _, name := range []string{"site.css", "site.js", "preferences.js", "icons.svg", "fonts/Exo2-Latin-v1.woff2"} {
		url := assetURL(name)
		w := httptest.NewRecorder()
		h.ServeHTTP(w, httptest.NewRequest("GET", url, nil))
		if w.Code != 200 || !strings.Contains(w.Header().Get("Cache-Control"), "immutable") {
			t.Fatalf("%s: %d %s", name, w.Code, w.Header())
		}
		if strings.HasSuffix(name, ".css") || strings.HasSuffix(name, ".js") {
			re := regexp.MustCompile(`/assets/([^/'"\s]+)`)
			for _, match := range re.FindAllStringSubmatch(w.Body.String(), -1) {
				if match[1] != manifest.revision {
					t.Errorf("unversioned nested URL: %s", match[0])
				}
			}
		}
		req := httptest.NewRequest("GET", url, nil)
		req.Header.Set("If-None-Match", w.Header().Get("ETag"))
		conditional := httptest.NewRecorder()
		h.ServeHTTP(conditional, req)
		if conditional.Code != 304 || conditional.Body.Len() != 0 {
			t.Fatal("conditional asset request did not return empty 304")
		}
	}
	for _, url := range []string{"/assets/", "/assets/fonts/", "/assets/not-this-build/site.css"} {
		w := httptest.NewRecorder()
		h.ServeHTTP(w, httptest.NewRequest("GET", url, nil))
		if w.Code != 404 {
			t.Errorf("%s = %d, want 404", url, w.Code)
		}
	}
	w := httptest.NewRecorder()
	h.ServeHTTP(w, httptest.NewRequest("GET", "/assets/site.css", nil))
	if w.Code != 200 || strings.Contains(w.Header().Get("Cache-Control"), "immutable") {
		t.Fatal("unversioned asset alias must revalidate")
	}
}

func TestIconSpriteIsOneValidDocumentWithUniqueSymbols(t *testing.T) {
	data, err := assets.ReadFile("static/icons.svg")
	if err != nil {
		t.Fatal(err)
	}
	decoder := xml.NewDecoder(strings.NewReader(string(data)))
	depth, roots := 0, 0
	symbols := map[string]bool{}
	for {
		token, err := decoder.Token()
		if err == io.EOF {
			break
		}
		if err != nil {
			t.Fatal(err)
		}
		switch token := token.(type) {
		case xml.StartElement:
			if depth == 0 {
				roots++
				if roots != 1 || token.Name.Local != "svg" {
					t.Fatal("icons appended outside the SVG root")
				}
			}
			depth++
			if token.Name.Local == "symbol" {
				id := ""
				for _, attr := range token.Attr {
					if attr.Name.Local == "id" {
						id = attr.Value
					}
				}
				if id == "" || symbols[id] {
					t.Fatalf("missing or duplicate symbol id %q", id)
				}
				symbols[id] = true
			}
		case xml.EndElement:
			depth--
		}
	}
	if roots != 1 || len(symbols) == 0 {
		t.Fatal("missing SVG sprite")
	}
}

func TestSitemapsUseOnlyPublishedStaticRoutes(t *testing.T) {
	s := &site{options: Options{Production: true}, pages: []staticPage{
		{Path: "/", Metadata: Metadata{CanonicalPath: "/"}, Content: home()},
		{Path: "/about", Metadata: Metadata{CanonicalPath: "/about", Indexable: true}, Content: home()},
		{Path: "/about/", Metadata: Metadata{CanonicalPath: "/about", Indexable: true}, Content: home()},
	}}
	w := httptest.NewRecorder()
	s.sitemap(w, httptest.NewRequest("GET", canonicalOrigin+"/sitemap.xml", nil))
	var doc sitemapDocument
	if err := xml.Unmarshal(w.Body.Bytes(), &doc); err != nil {
		t.Fatal(err)
	}
	if len(doc.Entries) != 1 || doc.Entries[0].Location != canonicalOrigin+"/about" {
		t.Fatalf("entries = %+v", doc.Entries)
	}
	robots := httptest.NewRecorder()
	s.robots(robots, httptest.NewRequest("GET", canonicalOrigin+"/robots.txt", nil))
	if !strings.Contains(robots.Body.String(), "Sitemap: "+canonicalOrigin+"/sitemap_index.xml") {
		t.Fatal("published sitemap not advertised")
	}
	for _, host := range []string{"localhost:4000", "void.eve-kill.com", "other.test"} {
		r := httptest.NewRequest("GET", "http://"+host+"/sitemap.xml", nil)
		r.Header.Set("X-Forwarded-Host", "eve-kill.com")
		other := httptest.NewRecorder()
		s.sitemap(other, r)
		if other.Code != 404 {
			t.Errorf("sitemap exposed on %s", host)
		}
	}
	current := httptest.NewRecorder()
	New(Options{Production: true}).ServeHTTP(current, httptest.NewRequest("GET", canonicalOrigin+"/sitemap.xml", nil))
	var published sitemapDocument
	if err := xml.Unmarshal(current.Body.Bytes(), &published); err != nil {
		t.Fatal(err)
	}
	if len(published.Entries) != 9 || published.Entries[0].Location != canonicalOrigin+"/about" || published.Entries[1].Location != canonicalOrigin+"/donate" || published.Entries[2].Location != canonicalOrigin+"/faq" || published.Entries[3].Location != canonicalOrigin+"/tools" || published.Entries[4].Location != canonicalOrigin+"/tools/dscan" || published.Entries[5].Location != canonicalOrigin+"/tools/localscan" || published.Entries[6].Location != canonicalOrigin+"/post" || published.Entries[7].Location != canonicalOrigin+"/docs" || published.Entries[8].Location != canonicalOrigin+"/mcp" {
		t.Fatalf("published static entries = %+v", published.Entries)
	}
}

func TestPageAssetReferencesExist(t *testing.T) {
	w := httptest.NewRecorder()
	h := New(Options{})
	h.ServeHTTP(w, httptest.NewRequest("GET", "/", nil))
	re := regexp.MustCompile(`(?:src|href)="(/assets/[^"#]+)`)
	for _, match := range re.FindAllStringSubmatch(w.Body.String(), -1) {
		assetResponse := httptest.NewRecorder()
		h.ServeHTTP(assetResponse, httptest.NewRequest("GET", match[1], nil))
		if assetResponse.Code != 200 {
			t.Errorf("page references missing asset %s", match[1])
		}
	}
}
