package frontend

import (
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"
)

const testCampaignID = "AbCdEf12345678"

func TestCampaignDetailRoutesAndScopedFragments(t *testing.T) {
	var forwarded []string
	apiHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Host != "eve-kill.com" || r.Header.Get("Cookie") != "session=test" {
			t.Errorf("campaign API scope host=%q cookie=%q", r.Host, r.Header.Get("Cookie"))
		}
		forwarded = append(forwarded, r.URL.String())
		switch r.URL.Path {
		case "/api/campaign/" + testCampaignID:
			_, _ = w.Write([]byte(`{"campaign_id":"` + testCampaignID + `","mode":"area","name":"Test Campaign","status":1,"visibility":0,"start_time":"2026-01-01T00:00:00Z","end_time":"2026-12-31T00:00:00Z","creator":{"character_id":42,"name":"Pilot"},"location_details":{"systems":[{"id":30000163,"name":"Akora"}],"constellations":[],"regions":[]},"stats":{"totals":{"killCount":12,"iskDestroyed":2000000000,"charactersInvolved":8,"corporationsInvolved":3,"alliancesInvolved":2},"mostValuable":[]},"daily":{"granularity":"day","rows":[]},"sides":[]}`))
		case "/api/campaign/" + testCampaignID + "/killlist":
			_, _ = w.Write([]byte(`{"kills":[],"hasMore":true,"cursor":42}`))
		default:
			http.NotFound(w, r)
		}
	})
	handler := New(Options{Production: true, CampaignsAPI: apiHandler})
	request := func(path string) *httptest.ResponseRecorder {
		w := httptest.NewRecorder()
		r := httptest.NewRequest(http.MethodGet, canonicalOrigin+path, nil)
		r.Header.Set("Cookie", "session=test")
		handler.ServeHTTP(w, r)
		return w
	}
	page := request("/campaign/" + testCampaignID)
	if page.Code != http.StatusOK || !strings.Contains(page.Body.String(), "Test Campaign") || !strings.Contains(page.Body.String(), ">Akora</a>") || !strings.Contains(page.Body.String(), `href="`+canonicalOrigin+`/campaign/`+testCampaignID+`"`) {
		t.Fatalf("campaign page status=%d markers missing", page.Code)
	}
	comments := request("/campaign/" + testCampaignID + "/comments")
	if comments.Code != http.StatusOK || !strings.Contains(comments.Body.String(), `data-target-slug="`+testCampaignID+`"`) {
		t.Fatalf("comments status=%d missing shared thread", comments.Code)
	}
	fragment := request("/_fragments/campaign/" + testCampaignID + "/comments")
	if fragment.Code != http.StatusOK || strings.Contains(fragment.Body.String(), "campaign-detail-header") || !strings.Contains(fragment.Body.String(), "data-comment-thread") {
		t.Fatalf("tab fragment status=%d unexpected content", fragment.Code)
	}
	feed := request("/_fragments/campaign/" + testCampaignID + "/killfeed?page=2&limit=10&after=42")
	if feed.Code != http.StatusOK || !strings.Contains(feed.Body.String(), `data-page="2"`) || !strings.Contains(feed.Body.String(), `data-limit="10"`) || !strings.Contains(feed.Body.String(), `data-next-cursor="42"`) {
		t.Fatalf("killfeed status=%d missing paging state", feed.Code)
	}
	last := forwarded[len(forwarded)-1]
	parsed, err := url.Parse(last)
	if err != nil || parsed.Query().Get("after") != "42" || parsed.Query().Get("limit") != "10" {
		t.Fatalf("forwarded killfeed %q: %v", last, err)
	}
	for _, path := range []string{
		"/campaign/" + testCampaignID + "/intel",
		"/campaign/" + testCampaignID + "/other",
		"/campaign/not-an-id",
	} {
		if got := request(path); got.Code != http.StatusNotFound {
			t.Errorf("%s status=%d", path, got.Code)
		}
	}
	for _, path := range []string{
		"/_fragments/campaign/" + testCampaignID + "/killfeed?page=2",
		"/_fragments/campaign/" + testCampaignID + "/killfeed?limit=500",
		"/_fragments/campaign/" + testCampaignID + "/killfeed?after=-1",
	} {
		if got := request(path); got.Code != http.StatusBadRequest {
			t.Errorf("%s status=%d", path, got.Code)
		}
	}
}

func TestPrivateCampaignIsNotIndexed(t *testing.T) {
	apiHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`{"campaign_id":"` + testCampaignID + `","mode":"area","name":"Private Campaign","status":1,"visibility":1,"location_details":{"systems":[],"constellations":[],"regions":[]},"sides":[]}`))
	})
	handler := New(Options{Production: true, CampaignsAPI: apiHandler})
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, httptest.NewRequest(http.MethodGet, canonicalOrigin+"/campaign/"+testCampaignID, nil))
	if w.Code != http.StatusOK || !strings.Contains(w.Body.String(), `name="robots" content="noindex, nofollow"`) {
		t.Fatalf("private campaign status=%d robots missing", w.Code)
	}
}

func TestCampaignPrizePoolRendersAndPreservesFundedDeleteGuard(t *testing.T) {
	closes := time.Now().Add(24 * time.Hour).UTC().Format(time.RFC3339)
	apiHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Host != "eve-kill.com" || r.Header.Get("Cookie") != "session=test" {
			t.Errorf("prize API scope host=%q cookie=%q", r.Host, r.Header.Get("Cookie"))
		}
		_, _ = w.Write([]byte(`{"campaign_id":"` + testCampaignID + `","mode":"conflict","name":"Prize Campaign","status":1,"visibility":0,"creator":{"character_id":42},"location_details":{"systems":[],"constellations":[],"regions":[]},"sides":[{"side_index":0,"name":"A","isk_destroyed":1,"isk_lost":1}],"prize_pool":{"metric":0,"metric_label":"Most kills","winner_count":1,"payout_percentages":[100],"status":0,"funded_total":"0.01","contribution_count":1,"funding_reference":"campaign:` + testCampaignID + `","funding_closes_at":"` + closes + `","results":[],"contributions":[{"id":"one","source":"ek_wallet","contributor_name":"Pilot","contributor_type":"character","contributor_id":42,"amount":"0.01","contributed_at":"2026-09-27T12:00:00Z"}]}}`))
	})
	handler := New(Options{Production: true, CampaignsAPI: apiHandler})
	request := func(path string) *httptest.ResponseRecorder {
		w := httptest.NewRecorder()
		r := httptest.NewRequest(http.MethodGet, canonicalOrigin+path, nil)
		r.Header.Set("Cookie", "session=test")
		handler.ServeHTTP(w, r)
		return w
	}
	page := request("/campaign/" + testCampaignID)
	if page.Code != http.StatusOK || !strings.Contains(page.Body.String(), "data-campaign-prize-open") || !strings.Contains(page.Body.String(), "campaign:"+testCampaignID) || strings.Contains(page.Body.String(), "data-campaign-delete title=") {
		t.Fatalf("funded campaign page status=%d prize/guard mismatch", page.Code)
	}
	fragment := request("/_fragments/campaign/" + testCampaignID + "/prize")
	if fragment.Code != http.StatusOK || !strings.Contains(fragment.Body.String(), "data-campaign-prize-dialog") || !strings.Contains(fragment.Body.String(), "data-campaign-prize-copy") || strings.Contains(fragment.Body.String(), "campaign-detail-header") {
		t.Fatalf("prize fragment status=%d expected modal only", fragment.Code)
	}
}
