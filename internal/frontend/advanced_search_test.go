package frontend

import (
	"compress/gzip"
	"context"
	"encoding/base64"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"

	"github.com/eve-kill/shrike/internal/api"
)

type advancedTestReader struct{ queries []url.Values }

func (r *advancedTestReader) Labels(context.Context) ([]api.LabelEntry, error) {
	return []api.LabelEntry{{ID: "solo", Name: "Solo", Category: "Engagement"}}, nil
}
func (r *advancedTestReader) AdvancedSearch(_ context.Context, query url.Values) (api.AdvancedSearchResult, error) {
	r.queries = append(r.queries, query)
	cursor := int64(1234)
	return api.AdvancedSearchResult{Kills: []api.KilllistRow{{KillmailID: 2345, ShipTypeID: 587, ShipName: "Rifter", ShipGroupID: 25, ShipGroupName: "Frigate", KillmailTime: time.Date(2026, 8, 18, 12, 0, 0, 0, time.UTC)}}, Fits: []api.AdvancedSearchFit{{KillmailID: 2345, ShipTypeID: 587, ShipName: "Rifter", KillmailTime: time.Date(2026, 8, 18, 12, 0, 0, 0, time.UTC), Count: 3, Hash: strings.Repeat("a", 64), DedupMode: "family", Modules: []api.UniverseFitModule{{SlotGroup: 1, TypeID: 425, Name: "Autocannon"}}}}, HasMore: true, Cursor: &cursor}, nil
}

func TestAdvancedSearchRendersControlsWithoutAnUnfilteredQuery(t *testing.T) {
	reader := &advancedTestReader{}
	handler := New(Options{Production: true, AdvancedSearch: reader})
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, httptest.NewRequest(http.MethodGet, canonicalOrigin+"/advancedsearch", nil))
	if w.Code != 200 || len(reader.queries) != 0 {
		t.Fatalf("initial search status=%d reads=%d", w.Code, len(reader.queries))
	}
	for _, text := range []string{"Advanced Search", "Killmail Label", "Security Space", "Attacker Count", "Attacker Type", "ISK Value", "Ship Category", "Tech Level", "Time Range", "Sort descending", "Saved searches", "Add filters above to find killmails", "data-date-time-picker", `data-controller="advanced-search saved-searches"`, "saved-searches:current->advanced-search#savedCurrent"} {
		if !strings.Contains(w.Body.String(), text) {
			t.Errorf("missing control %q", text)
		}
	}
}

func TestAdvancedSearchSSRAndFragmentsUseTheSameFilterContract(t *testing.T) {
	reader := &advancedTestReader{}
	handler := New(Options{Production: true, AdvancedSearch: reader})
	q := `{"entities":{"attacker":[{"id":2116635241,"type":"character","name":"<Pilot>","role":"attacker","exclude":true}]},"items":[{"typeId":425,"name":"Autocannon"}],"location":{"systemId":30000142,"systemName":"Jita"},"iskMin":"500m","iskMax":2000000000,"timeRange":{"from":"2026-08-18T12:00","to":"2026-08-18T16:00"},"sort":{"field":"total_value","direction":"asc"}}`
	query := url.Values{"q": {q}, "view": {"fits"}, "dedup": {"exact"}, "dh": {strings.Repeat("b", 64)}, "dm": {"family"}, "after": {"1234"}, "limit": {"25"}}
	for _, path := range []string{"/advancedsearch", "/_fragments/advancedsearch"} {
		w := httptest.NewRecorder()
		handler.ServeHTTP(w, httptest.NewRequest(http.MethodGet, canonicalOrigin+path+"?"+query.Encode(), nil))
		if w.Code != 200 || !strings.Contains(w.Body.String(), "data-fit-preview") || !strings.Contains(w.Body.String(), "data-fit-show-kills") {
			t.Fatalf("%s failed to render fitting results: %d", path, w.Code)
		}
		if strings.Contains(w.Body.String(), "<Pilot>") {
			t.Fatal("untrusted display name was not escaped")
		}
		if w.Header().Get("Cache-Control") != "private, no-store" || !strings.Contains(w.Header().Get("X-Robots-Tag"), "noindex") {
			t.Fatal("query result was cacheable/indexable")
		}
		got := reader.queries[len(reader.queries)-1]
		if got.Get("view") != "fits" || got.Get("dedup") != "exact" || got.Get("familyHash") != strings.Repeat("b", 64) || got.Get("after") != "1234" || got.Get("limit") != "25" {
			t.Fatalf("lost result controls: %s", got.Encode())
		}
		var filters map[string]any
		if err := json.Unmarshal([]byte(got.Get("filters")), &filters); err != nil {
			t.Fatal(err)
		}
		if filters["iskMin"] != float64(500000000) || filters["iskMax"] != float64(2000000000) {
			t.Fatal("ISK shorthand/numeric saved search changed")
		}
		if strings.Contains(got.Get("filters"), `"role"`) || strings.Contains(got.Get("filters"), `"systemName"`) {
			t.Fatal("display-only state reached the query")
		}
		items := filters["items"].([]any)
		item := items[0].(map[string]any)
		if item["slot"] != "any" || item["side"] != "victim" {
			t.Fatal("older item search lost its default slot/side")
		}
	}
}

func TestAdvancedSearchRejectsMalformedLinksWithoutQuerying(t *testing.T) {
	reader := &advancedTestReader{}
	handler := New(Options{AdvancedSearch: reader})
	for _, path := range []string{"/advancedsearch", "/_fragments/advancedsearch"} {
		w := httptest.NewRecorder()
		handler.ServeHTTP(w, httptest.NewRequest(http.MethodGet, path+"?q=%5B1%5D", nil))
		if w.Code != 200 || !strings.Contains(w.Body.String(), "invalid filters") {
			t.Fatalf("invalid link not explained: %d", w.Code)
		}
	}
	if len(reader.queries) != 0 {
		t.Fatal("malformed query reached the database")
	}
}

func TestFitPreviewEditorLinkPreservesSlotsChargesAndDroneQuantities(t *testing.T) {
	link := observedFitEditorURL(587, "Rifter", "", []api.UniverseFitModule{{SlotGroup: 1, Ordinal: 7, TypeID: 425, ChargeTypeID: 199}, {SlotGroup: 1, Ordinal: 3, TypeID: 425}, {SlotGroup: 3, Ordinal: 5, TypeID: 2048}}, []api.UniverseFitDrone{{TypeID: 2456, Quantity: 7}, {TypeID: 2464, Quantity: 2}})
	u, err := url.Parse(link)
	if err != nil {
		t.Fatal(err)
	}
	encoded := u.Query().Get("fit")
	if !strings.HasPrefix(encoded, "v3:") {
		t.Fatal("editor link is not interoperable ESF v3")
	}
	compressed, err := base64.StdEncoding.DecodeString(strings.TrimPrefix(encoded, "v3:"))
	if err != nil {
		t.Fatal(err)
	}
	reader, err := gzip.NewReader(strings.NewReader(string(compressed)))
	if err != nil {
		t.Fatal(err)
	}
	defer reader.Close()
	plain, err := io.ReadAll(reader)
	if err != nil {
		t.Fatal(err)
	}
	want := "ship,587,Rifter,\nmodule,High,1,425,Active,199\nmodule,High,2,425,Active,\nmodule,Low,1,2048,Active,\ndrone,2456,5,2\ndrone,2464,0,2\n"
	if string(plain) != want {
		t.Fatalf("encoded fitting changed slots/ammo/active drone budget:\n%s", plain)
	}
}
