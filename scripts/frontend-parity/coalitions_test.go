package main

import (
	"archive/tar"
	"bytes"
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestCoalitionReferenceUsesOriginalScriptAndKeepsReadsAnonymous(t *testing.T) {
	rendered, forwarded := 0, 0
	renderer := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.HasPrefix(r.URL.Path, "/assets/") {
			if r.URL.Path == "/assets/current/coalitions.css" {
				fmt.Fprint(w, "unchanged coalition CSS")
			} else {
				w.WriteHeader(404)
			}
			return
		}
		rendered++
		if r.Header.Get("Cookie") != "" || r.Header.Get("Authorization") != "" {
			t.Error("reference received credentials")
		}
		w.Header().Set("Set-Cookie", "secret=fixture")
		fmt.Fprint(w, `<meta name="shrike-dev-instance" content="current"><div class="coalition-page" data-controller="coalition-activity coalitions"><a class="coalition-card" href="/coalitions/example">Example</a></div><script type="module" src="/assets/current/site.js"></script><script type="module" src="/assets/current/coalitions.js"></script>`)
	})
	reference := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		forwarded++
		if r.Method != "GET" && r.Method != "HEAD" {
			w.WriteHeader(405)
			return
		}
		fmt.Fprint(w, "retained "+r.URL.Path)
	})
	handler := coalitionReferenceRenderer(reference, renderer, map[string][]byte{"coalitions.js": []byte("original coalition script"), "coalition-activity.js": []byte("original activity script")})
	request := httptest.NewRequest("GET", "/coalitions", nil)
	request.Header.Set("Cookie", "private=session")
	request.Header.Set("Authorization", "private")
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, request)
	if !strings.Contains(response.Body.String(), `src="/__fixture/coalition-source/coalitions.js"`) || !strings.Contains(response.Body.String(), `src="/assets/current/site.js"`) || !strings.Contains(response.Body.String(), `class="coalition-card"`) || strings.Contains(response.Body.String(), `data-controller="coalition-activity coalitions"`) || response.Header().Get("Set-Cookie") != "" {
		t.Fatal(response.Body.String())
	}
	if strings.Contains(response.Body.String(), "shrike-dev-instance") {
		t.Fatal("comparison retained mismatched development instance")
	}
	for _, item := range []struct{ method, path string }{{"POST", "/api/coalitions"}, {"PATCH", "/coalitions/example"}, {"GET", "/about"}, {"GET", "/assets/original/coalitions.js"}} {
		response := httptest.NewRecorder()
		handler.ServeHTTP(response, httptest.NewRequest(item.method, item.path, nil))
		if item.method != "GET" && response.Code != 405 {
			t.Error("reference mutation was not blocked")
		}
	}
	if rendered != 1 || forwarded != 4 {
		t.Fatalf("unexpected delegation: render=%d forward=%d", rendered, forwarded)
	}
	response = httptest.NewRecorder()
	handler.ServeHTTP(response, httptest.NewRequest("GET", "/assets/current/coalitions.css", nil))
	if response.Body.String() != "unchanged coalition CSS" || forwarded != 4 {
		t.Fatal("current coalition styles were not served")
	}
}

func TestCoalitionSourceArchiveReadsOnlyTwoNamedScripts(t *testing.T) {
	var buffer bytes.Buffer
	writer := tar.NewWriter(&buffer)
	for _, name := range []string{"coalitions.js", "coalition-activity.js", "private.env"} {
		body := []byte("original " + name)
		if err := writer.WriteHeader(&tar.Header{Name: "internal/frontend/static/" + name, Mode: 0600, Size: int64(len(body))}); err != nil {
			t.Fatal(err)
		}
		if _, err := writer.Write(body); err != nil {
			t.Fatal(err)
		}
	}
	if err := writer.Close(); err != nil {
		t.Fatal(err)
	}
	scripts, err := readCoalitionScripts(tar.NewReader(bytes.NewReader(buffer.Bytes())))
	if err != nil || len(scripts) != 2 || string(scripts["coalitions.js"]) != "original coalitions.js" || scripts["private.env"] != nil {
		t.Fatal("archive selection failed", err)
	}
	handler := coalitionReferenceRenderer(http.NotFoundHandler(), http.NotFoundHandler(), scripts)
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, httptest.NewRequest("GET", "/__fixture/coalition-source/coalition-activity.js", nil))
	if response.Code != 200 || response.Body.String() != "original coalition-activity.js" {
		t.Fatal(response.Body.String())
	}
	response = httptest.NewRecorder()
	handler.ServeHTTP(response, httptest.NewRequest("GET", "/__fixture/coalition-source/private.env", nil))
	if response.Code != 404 {
		t.Fatal("unexpected archive exposure")
	}
}

func TestRemoteCoalitionReaderUsesPublicLocalAPIAndNotFound(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "GET" || r.Header.Get("Cookie") != "" {
			t.Error("reader should be anonymous GET")
		}
		switch r.URL.Path {
		case "/api/coalitions":
			fmt.Fprint(w, `{"coalitions":[{"coalition_id":1,"slug":"example"}],"stats_window_days":30}`)
		case "/api/coalitions/example":
			fmt.Fprint(w, `{"coalition":{"coalition_id":1,"slug":"example"},"alliances":[],"edits":[],"stats_window_days":30}`)
		default:
			w.WriteHeader(404)
		}
	}))
	defer server.Close()
	reader := remoteCoalitions{server.URL}
	directory, err := reader.CoalitionDirectoryOverview(context.Background())
	if err != nil || len(directory.Coalitions) != 1 || directory.StatsWindowDays != 30 {
		t.Fatal(directory, err)
	}
	detail, found, err := reader.CoalitionDetail(context.Background(), "example")
	if err != nil || !found || detail.Coalition.Slug != "example" {
		t.Fatal(detail, found, err)
	}
	_, found, err = reader.CoalitionDetail(context.Background(), "missing")
	if err != nil || found {
		t.Fatal(found, err)
	}
}
