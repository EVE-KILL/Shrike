package frontend

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/eve-kill/shrike/internal/api"
)

type campaignSessionTestReader struct {
	user   *api.Principal
	cookie string
}

func TestCampaignEditorFragmentRequiresOwnershipAndLocksFundedRules(t *testing.T) {
	reader := &campaignSessionTestReader{}
	detail := campaignDetailData{ID: "ABC123abc456XY", Name: "<Untrusted name>", Mode: "conflict", EndTime: "2026-09-29T12:00:00Z", PrizePool: &campaignPrizePool{RulesLockedAt: "2026-09-28T12:00:00Z"}, Sides: []campaignDetailSide{{Name: "Fleet", Entities: []campaignDetailMember{{EntityType: 0, EntityID: 42, Name: "Pilot"}, {EntityType: 2, EntityID: 99}}}}}
	detail.Creator.CharacterID = 42
	apiReads := 0
	handler := New(Options{Sessions: reader, CampaignsAPI: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { apiReads++; json.NewEncoder(w).Encode(detail) })})
	call := func() *httptest.ResponseRecorder {
		r := httptest.NewRequest(http.MethodGet, "/_fragments/campaign/ABC123abc456XY/edit", nil)
		r.AddCookie(&http.Cookie{Name: "evelogin", Value: "session"})
		w := httptest.NewRecorder()
		handler.ServeHTTP(w, r)
		return w
	}
	if got := call(); got.Code != 401 || apiReads != 0 {
		t.Fatalf("anonymous editor: %d, reads=%d", got.Code, apiReads)
	}
	reader.user = &api.Principal{CharacterID: 7}
	if got := call(); got.Code != 403 {
		t.Fatalf("non-owner editor: %d", got.Code)
	}
	reader.user = &api.Principal{CharacterID: 42}
	got := call()
	html := got.Body.String()
	if got.Code != 200 || !strings.Contains(html, "Participants are locked") || strings.Contains(html, "<Untrusted name>") {
		t.Fatal("funded editor escaped/lock rendering failed")
	}
	visibleMarkup, _, _ := strings.Cut(html, "<template")
	if strings.Contains(visibleMarkup, `data-picker-name="edit-side"`) {
		t.Fatal("funded participant picker was rendered")
	}
	if !strings.Contains(html, `data-edit-ongoing type="checkbox" disabled`) || !strings.Contains(html, `data-date-time-trigger`) {
		t.Fatal("fixed prize end did not use the locked shared picker")
	}
	if got.Header().Get("Cache-Control") != "private, no-store" || !strings.Contains(got.Header().Get("X-Robots-Tag"), "noindex") {
		t.Fatal("private editor fragment was cacheable/indexable")
	}
	if !strings.Contains(visibleMarkup, "#99") {
		t.Fatal("participant without a resolved name lost its ID label")
	}
}

func (r *campaignSessionTestReader) BrowserSession(_ context.Context, cookie string) (*api.Principal, error) {
	r.cookie = cookie
	return r.user, nil
}

func TestCampaignCreatorUsesTheAuthoritativeSessionAndEscapesItsPreview(t *testing.T) {
	reader := &campaignSessionTestReader{user: &api.Principal{CharacterID: 42, CharacterName: "<script>test</script>"}}
	handler := New(Options{Production: true, Sessions: reader, CampaignsAPI: http.NotFoundHandler()})
	for _, authenticated := range []bool{false, true} {
		request := httptest.NewRequest(http.MethodGet, canonicalOrigin+"/campaigncreator", nil)
		if authenticated {
			request.AddCookie(&http.Cookie{Name: "evelogin", Value: "session"})
		} else {
			request.AddCookie(&http.Cookie{Name: "ek_auth", Value: "42:untrusted hint"})
		}
		response := httptest.NewRecorder()
		handler.ServeHTTP(response, request)
		if response.Code != 200 {
			t.Fatalf("creator status: %d", response.Code)
		}
		html := response.Body.String()
		if strings.Contains(html, "<script>test</script>") {
			t.Fatal("unescaped principal name")
		}
		if authenticated {
			if reader.cookie != "session" || !strings.Contains(html, `data-campaign-login hidden`) {
				t.Fatal("valid session did not render the form")
			}
		} else {
			if !strings.Contains(html, `data-campaign-form hidden`) {
				t.Fatal("display hint authenticated a request")
			}
		}
		for _, text := range []string{"Campaign Creator", "Tracking mode", "Who can find", "Funded prize pool", "Search battlefield", "Ranking metric", "Create Campaign", "campaign_start"} {
			if !strings.Contains(html, text) {
				t.Errorf("missing control %q", text)
			}
		}
		if response.Header().Get("Cache-Control") != "private, no-store" {
			t.Fatal("creator HTML was cacheable")
		}
	}
}
