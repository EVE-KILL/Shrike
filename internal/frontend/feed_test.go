package frontend

import (
	"context"
	"encoding/xml"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/eve-kill/shrike/internal/api"
)

type rssReaderStub struct{ homepageReaderStub }

func (rssReaderStub) LatestKills(context.Context, int, int) (api.KilllistPage, error) {
	return api.KilllistPage{Kills: []api.KilllistRow{{
		KillmailID: 42, KillmailTime: time.Date(2026, 9, 24, 12, 0, 0, 0, time.UTC),
		ShipTypeID: 670, ShipName: "Cruiser <A>", VictimCharacterName: "A & B",
		SolarSystemName: "Jita", TotalValue: 1_500_000,
	}}}, nil
}

func TestRSSFeedUsesLatestKillsAndEscapesContent(t *testing.T) {
	handler := New(Options{Homepage: rssReaderStub{}})
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/feed.xml", nil))
	if w.Code != http.StatusOK || !strings.HasPrefix(w.Header().Get("Content-Type"), "application/rss+xml") {
		t.Fatalf("feed response = %d %q", w.Code, w.Header().Get("Content-Type"))
	}
	var doc rssFeed
	if err := xml.Unmarshal(w.Body.Bytes(), &doc); err != nil {
		t.Fatalf("invalid RSS XML: %v", err)
	}
	if len(doc.Channel.Items) != 1 || doc.Channel.Items[0].Link != canonicalOrigin+"/kill/42" || doc.Channel.Items[0].Enclosure == nil ||
		doc.Channel.Items[0].Title != "Cruiser <A> | A & B | 1.50M ISK" ||
		!strings.Contains(w.Body.String(), "Cruiser &lt;A&gt; | A &amp; B") {
		t.Fatalf("feed items = %+v", doc.Channel.Items)
	}
	icon := httptest.NewRecorder()
	handler.ServeHTTP(icon, httptest.NewRequest(http.MethodGet, "/icon.png", nil))
	if icon.Code != http.StatusOK || !strings.HasPrefix(icon.Header().Get("Content-Type"), "image/png") {
		t.Fatalf("RSS icon response = %d %q", icon.Code, icon.Header().Get("Content-Type"))
	}
}
