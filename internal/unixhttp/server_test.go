package unixhttp

import (
	"context"
	"io"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"testing"
)

func TestServerRoundTripAndCleanup(t *testing.T) {
	socket := filepath.Join(t.TempDir(), "api.sock")
	server, err := Listen(socket, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("X-Request-Host", r.Host)
		_, _ = io.WriteString(w, r.URL.RequestURI())
	}))
	if err != nil {
		t.Fatalf("Listen: %v", err)
	}

	transport := &http.Transport{
		DialContext: func(ctx context.Context, _, _ string) (net.Conn, error) {
			var dialer net.Dialer
			return dialer.DialContext(ctx, "unix", socket)
		},
	}
	client := &http.Client{Transport: transport}
	defer transport.CloseIdleConnections()

	request, err := http.NewRequest(http.MethodGet, "http://shrike.internal/api/kills?limit=5", nil)
	if err != nil {
		t.Fatal(err)
	}
	request.Host = "boring.eve-kill.com"
	response, err := client.Do(request)
	if err != nil {
		t.Fatalf("request over Unix socket: %v", err)
	}
	body, err := io.ReadAll(response.Body)
	_ = response.Body.Close()
	if err != nil {
		t.Fatal(err)
	}
	if got, want := string(body), "/api/kills?limit=5"; got != want {
		t.Fatalf("body = %q, want %q", got, want)
	}
	if got, want := response.Header.Get("X-Request-Host"), "boring.eve-kill.com"; got != want {
		t.Fatalf("forwarded host = %q, want %q", got, want)
	}

	if err := server.Close(); err != nil {
		t.Fatalf("Close: %v", err)
	}
	if _, err := os.Lstat(socket); !os.IsNotExist(err) {
		t.Fatalf("socket still exists after Close: %v", err)
	}
}

func TestListenNeverReplacesOrdinaryFile(t *testing.T) {
	path := filepath.Join(t.TempDir(), "important")
	if err := os.WriteFile(path, []byte("keep me"), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := Listen(path, http.HandlerFunc(func(http.ResponseWriter, *http.Request) {})); err == nil {
		t.Fatal("Listen should reject an existing ordinary file")
	}
	body, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if got, want := string(body), "keep me"; got != want {
		t.Fatalf("file changed to %q, want %q", got, want)
	}
}

func TestListenRejectsUnsafePaths(t *testing.T) {
	handler := http.HandlerFunc(func(http.ResponseWriter, *http.Request) {})
	for _, path := range []string{"", "relative.sock", string(filepath.Separator)} {
		t.Run(path, func(t *testing.T) {
			if _, err := Listen(path, handler); err == nil {
				t.Fatalf("Listen(%q) should fail", path)
			}
		})
	}
}
