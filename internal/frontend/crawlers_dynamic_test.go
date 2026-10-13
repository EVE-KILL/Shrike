package frontend

import (
	"encoding/json"
	"encoding/xml"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestDynamicSitemapSourcesAndIndex(t *testing.T) {
	apiHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/__sitemap__/regions" || r.Host != "eve-kill.com" || r.Header.Get("X-Forwarded-Host") != "eve-kill.com" {
			t.Errorf("unexpected sitemap API request: path=%s host=%s forwarded=%s", r.URL.Path, r.Host, r.Header.Get("X-Forwarded-Host"))
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`[{"loc":"/region/10000042","lastmod":"2026-09-27T12:00:00Z","changefreq":"daily","priority":0.6}]`))
	})
	handler := New(Options{Production: true, SitemapAPI: apiHandler})
	request := func(path string) *httptest.ResponseRecorder {
		w := httptest.NewRecorder()
		handler.ServeHTTP(w, httptest.NewRequest(http.MethodGet, canonicalOrigin+path, nil))
		return w
	}

	data := request("/_sitemap-data/regions")
	if data.Code != http.StatusOK || !strings.Contains(data.Header().Get("Content-Type"), "application/json") {
		t.Fatalf("source status=%d content-type=%q", data.Code, data.Header().Get("Content-Type"))
	}
	var source []sitemapSourceEntry
	if err := json.Unmarshal(data.Body.Bytes(), &source); err != nil || len(source) != 1 || source[0].Location != "/region/10000042" {
		t.Fatalf("source=%+v, error=%v", source, err)
	}
	category := request("/__sitemap__/regions.xml")
	var doc sitemapDocument
	if err := xml.Unmarshal(category.Body.Bytes(), &doc); err != nil || category.Code != http.StatusOK || len(doc.Entries) != 1 {
		t.Fatalf("category status=%d entries=%+v error=%v", category.Code, doc.Entries, err)
	}
	if entry := doc.Entries[0]; entry.Location != canonicalOrigin+"/region/10000042" || entry.LastModified != "2026-09-27T12:00:00Z" || entry.ChangeFrequency != "daily" || entry.Priority != "0.6" {
		t.Fatalf("category entry=%+v", entry)
	}
	index := request("/sitemap_index.xml")
	var indexDoc sitemapIndexDocument
	if err := xml.Unmarshal(index.Body.Bytes(), &indexDoc); err != nil || len(indexDoc.Entries) != 1+len(dynamicSitemapKinds) {
		t.Fatalf("index entries=%+v error=%v", indexDoc.Entries, err)
	}
	if indexDoc.Entries[0].Location != canonicalOrigin+"/__sitemap__/pages.xml" || indexDoc.Entries[len(indexDoc.Entries)-1].Location != canonicalOrigin+"/__sitemap__/battles.xml" {
		t.Fatalf("index endpoints=%+v", indexDoc.Entries)
	}
	if pages := request("/__sitemap__/pages.xml"); pages.Code != http.StatusOK || !strings.Contains(pages.Body.String(), canonicalOrigin+"/about") {
		t.Fatalf("pages status=%d body=%s", pages.Code, pages.Body.String())
	}
	if invalid := request("/_sitemap-data/unknown"); invalid.Code != http.StatusNotFound {
		t.Fatalf("unknown kind status=%d", invalid.Code)
	}
	other := httptest.NewRecorder()
	handler.ServeHTTP(other, httptest.NewRequest(http.MethodGet, "https://other.test/__sitemap__/regions.xml", nil))
	if other.Code != http.StatusNotFound {
		t.Fatalf("noncanonical sitemap status=%d", other.Code)
	}
}
