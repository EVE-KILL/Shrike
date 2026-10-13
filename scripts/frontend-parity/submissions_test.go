package main

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestSubmissionFixtureNeverForwardsQueueJobs(t *testing.T) {
	forwarded := 0
	handler := newFixtureSubmissions().wrap(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) { forwarded++; w.WriteHeader(418) }))
	request := func(method, path, body string) *httptest.ResponseRecorder {
		response := httptest.NewRecorder()
		handler.ServeHTTP(response, httptest.NewRequest(method, path, strings.NewReader(body)))
		return response
	}
	if response := request("POST", "/__fixture/submissions", `{"existing":[900000001]}`); response.Code != 204 {
		t.Fatal(response.Body.String())
	}
	response := request("POST", "/api/killmail/post", `{"text":"https://esi.evetech.net/latest/killmails/900000001/aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa/\nhttps://esi.evetech.net/v1/killmails/900000002/bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb/\nhttps://esi.evetech.net/v1/killmails/900000002/bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb/"}`)
	var result struct {
		Total, Accepted, Existing, Rejected int
		Killmails, ExistingIDs              []int64
	}
	if response.Code != 200 || json.Unmarshal(response.Body.Bytes(), &result) != nil || result.Total != 2 || result.Accepted != 1 || result.Existing != 1 || result.Rejected != 0 || len(result.Killmails) != 2 || len(result.ExistingIDs) != 1 {
		t.Fatalf("invalid acknowledgment: %s", response.Body.String())
	}
	if response := request("GET", "/api/killmail/900000002/exists", ""); response.Body.String() != "{\"exists\":false}\n" {
		t.Fatal(response.Body.String())
	}
	if response := request("GET", "/api/killmail/900000002", ""); response.Code != 404 {
		t.Fatal(response.Code)
	}
	if response := request("POST", "/__fixture/submissions", `{"ready":[900000002]}`); response.Code != 204 {
		t.Fatal(response.Body.String())
	}
	if response := request("GET", "/api/killmail/900000002/exists", ""); response.Body.String() != "{\"exists\":true}\n" {
		t.Fatal(response.Body.String())
	}
	if response := request("GET", "/api/killmail/900000002", ""); !strings.Contains(response.Body.String(), "Fixture Pilot") {
		t.Fatal(response.Body.String())
	}
	for _, body := range []string{`{`, `{"text":""}`} {
		if response := request("POST", "/api/killmail/post", body); response.Code != 400 {
			t.Fatal(response.Code)
		}
	}
	if response := request("PUT", "/api/killmail/post", `{}`); response.Code != 405 {
		t.Fatal(response.Code)
	}
	if forwarded != 0 {
		t.Fatalf("submission reached upstream: %d calls", forwarded)
	}
	if response := request("GET", "/api/killmail/123", ""); response.Code != 418 || forwarded != 1 {
		t.Fatal("unrelated reads should remain unchanged")
	}
}

func TestSubmissionFixtureRejectionsAndInvalidConfiguration(t *testing.T) {
	handler := newFixtureSubmissions().wrap(http.NotFoundHandler())
	request := func(path, body string) *httptest.ResponseRecorder {
		response := httptest.NewRecorder()
		handler.ServeHTTP(response, httptest.NewRequest("POST", path, strings.NewReader(body)))
		return response
	}
	for _, body := range []string{`{"ready":[0]}`, `{"existing":[9007199254740992]}`, `{"detail":[]}`} {
		if response := request("/__fixture/submissions", body); response.Code != 400 {
			t.Fatalf("accepted invalid config %s", body)
		}
	}
	if response := request("/__fixture/submissions", `{"reject":true}`); response.Code != 204 {
		t.Fatal(response.Code)
	}
	response := request("/api/killmail/post", `{"text":"https://esi.evetech.net/latest/killmails/900000002/aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa/"}`)
	if !strings.Contains(response.Body.String(), `"rejected":1`) || !strings.Contains(response.Body.String(), `"accepted":0`) {
		t.Fatal(response.Body.String())
	}
}
