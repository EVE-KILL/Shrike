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

type generatorTestReader struct {
	input json.RawMessage
	err   error
	calls int
}

func (r *generatorTestReader) BattleGeneratorPreview(_ context.Context, input json.RawMessage) (json.RawMessage, error) {
	r.calls++
	r.input = input
	if r.err != nil {
		return nil, r.err
	}
	return json.RawMessage(`{"solar_system_id":30000142,"solar_system_name":"Jita","region_name":"The Forge","kill_count":2,"total_isk_destroyed":350000000,"duration_minutes":60,"teams":[{"name":"Red <script>alert(1)</script>","total_kills":1,"alliance_count":0,"corp_count":0,"alliances":[]},{"name":"Blue","total_kills":1,"alliance_count":0,"corp_count":0,"alliances":[]}]}`), nil
}

type generatorTestError struct{}

func (generatorTestError) Error() string  { return "A corporation cannot be on both sides" }
func (generatorTestError) GetStatus() int { return 400 }

func TestBattleGeneratorGuestPrefillAndAuthenticatedPreview(t *testing.T) {
	reader := &generatorTestReader{}
	session := &campaignSessionTestReader{}
	handler := New(Options{BattleGenerator: reader, Sessions: session})
	call := func(method, path, body string) *httptest.ResponseRecorder {
		r := httptest.NewRequest(method, path, strings.NewReader(body))
		r.AddCookie(&http.Cookie{Name: "evelogin", Value: "local-session"})
		w := httptest.NewRecorder()
		handler.ServeHTTP(w, r)
		return w
	}
	page := call("GET", "/battlegenerator?system=30000142&systemName=Jita&start=2026-09-24T14:00:00%2B02:00&end=2026-09-24T15:00:00%2B02:00&battle=123&step=sides", "")
	if page.Code != 200 || !strings.Contains(page.Body.String(), "You must be logged in") || !strings.Contains(page.Body.String(), "2026-09-24T12:00") || !strings.Contains(page.Body.String(), `"battleId":123`) {
		t.Fatal("guest/prefill did not retain the source contract")
	}
	if w := call("POST", "/_fragments/battlegenerator/preview", `{}`); w.Code != 401 || reader.calls != 0 {
		t.Fatal("anonymous preview must not call the reader")
	}
	session.user = &api.Principal{CharacterID: 42}
	w := call("POST", "/_fragments/battlegenerator/preview", `{"systemIds":[30000142]}`)
	if w.Code != 200 || !strings.Contains(w.Body.String(), "data-generator-preview-loaded") || strings.Contains(w.Body.String(), "<script>alert(1)</script>") || !strings.Contains(w.Body.String(), "350.00m") || !strings.Contains(w.Header().Get("Cache-Control"), "no-store") {
		t.Fatalf("preview did not render safely: %d", w.Code)
	}
	reader.err = generatorTestError{}
	w = call("POST", "/_fragments/battlegenerator/preview", `{}`)
	if w.Code != 400 || !strings.Contains(w.Body.String(), "both sides") {
		t.Fatal("the authoritative validation error was lost")
	}
}
