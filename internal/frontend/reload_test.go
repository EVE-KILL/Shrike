package frontend

import (
	"bufio"
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func TestDevelopmentReloadStreamMatchesPageIdentity(t *testing.T) {
	server := httptest.NewServer(New(Options{}))
	defer server.Close()

	page, err := server.Client().Get(server.URL + "/")
	if err != nil {
		t.Fatal(err)
	}
	body, err := io.ReadAll(page.Body)
	_ = page.Body.Close()
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(body), `name="shrike-dev-instance" content="`+devInstanceID+`"`) ||
		!strings.Contains(string(body), "/reload.js") {
		t.Fatal("development page does not include its instance identity and reload script")
	}

	ctx, cancel := context.WithTimeout(t.Context(), 3*time.Second)
	defer cancel()
	request, err := http.NewRequestWithContext(ctx, http.MethodGet, server.URL+"/__dev/reload", nil)
	if err != nil {
		t.Fatal(err)
	}
	response, err := server.Client().Do(request)
	if err != nil {
		t.Fatal(err)
	}
	defer response.Body.Close()
	if response.StatusCode != http.StatusOK ||
		!strings.HasPrefix(response.Header.Get("Content-Type"), "text/event-stream") ||
		response.Header.Get("Cache-Control") != "private, no-store" {
		t.Fatalf("reload response = %d, headers = %v", response.StatusCode, response.Header)
	}
	reader := bufio.NewReader(response.Body)
	for _, want := range []string{"retry: 1000\n", "event: build\n", "data: " + devInstanceID + "\n", "\n"} {
		line, err := reader.ReadString('\n')
		if err != nil || line != want {
			t.Fatalf("line = %q, error = %v, want %q", line, err, want)
		}
	}
	cancel()
}

func TestProductionDoesNotExposeReload(t *testing.T) {
	handler := New(Options{Production: true})
	page := httptest.NewRecorder()
	handler.ServeHTTP(page, httptest.NewRequest("GET", "https://eve-kill.com/", nil))
	if strings.Contains(page.Body.String(), "shrike-dev-instance") || strings.Contains(page.Body.String(), "/reload.js") {
		t.Fatal("production page includes reload client")
	}
	for _, path := range []string{"/__dev/reload", "/__dev/version"} {
		stream := httptest.NewRecorder()
		handler.ServeHTTP(stream, httptest.NewRequest("GET", "https://eve-kill.com"+path, nil))
		if stream.Code != http.StatusNotFound || strings.Contains(stream.Body.String(), devInstanceID) {
			t.Fatalf("production reload route = %d %s", stream.Code, stream.Header().Get("Content-Type"))
		}
	}
}

func TestDevelopmentVersionIsBoundedPrivateAndMatchesHTML(t *testing.T) {
	w := httptest.NewRecorder()
	New(Options{}).ServeHTTP(w, httptest.NewRequest("GET", "/__dev/version", nil))
	if w.Code != 200 || strings.TrimSpace(w.Body.String()) != devInstanceID || w.Header().Get("Cache-Control") != "private, no-store" || w.Header().Get("X-Robots-Tag") != "noindex" || !strings.HasPrefix(w.Header().Get("Content-Type"), "text/plain") {
		t.Fatal("development version read has a different identity or can be cached")
	}
}
