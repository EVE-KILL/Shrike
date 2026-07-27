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

// The development listener carries the same authenticated surface as the public
// one, without the edge in front of it. Binding it anywhere routable would
// publish an unguarded copy of the whole site, so a non-loopback address is
// refused rather than accepted with a warning.
func TestListenLoopbackRejectsRoutableAddresses(t *testing.T) {
	handler := http.HandlerFunc(func(http.ResponseWriter, *http.Request) {})
	for _, address := range []string{
		"0.0.0.0:0", "192.168.1.10:0", "::0", "example.test:80", "4002",
	} {
		if server, err := ListenLoopback(address, handler); err == nil {
			_ = server.Close()
			t.Errorf("ListenLoopback(%q) was accepted", address)
		}
	}
}

func TestListenLoopbackServesAndLeavesNoSocket(t *testing.T) {
	server, err := ListenLoopback("127.0.0.1:0", http.HandlerFunc(
		func(w http.ResponseWriter, r *http.Request) {
			_, _ = io.WriteString(w, r.URL.RequestURI())
		},
	))
	if err != nil {
		t.Fatalf("ListenLoopback: %v", err)
	}
	if server.Socket() != "" {
		t.Errorf("Socket() = %q, want empty for a loopback listener", server.Socket())
	}

	response, err := http.Get("http://" + server.Addr() + "/site")
	if err != nil {
		t.Fatalf("get: %v", err)
	}
	defer response.Body.Close() //nolint:errcheck
	body, err := io.ReadAll(response.Body)
	if err != nil {
		t.Fatalf("read: %v", err)
	}
	if string(body) != "/site" {
		t.Errorf("body = %q, want %q", body, "/site")
	}

	// Close must not fail trying to unlink a socket that never existed.
	if err := server.Close(); err != nil {
		t.Fatalf("Close: %v", err)
	}
}
