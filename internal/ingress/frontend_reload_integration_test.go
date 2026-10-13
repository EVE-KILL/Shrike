package ingress

import (
	"bufio"
	"context"
	"fmt"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/eve-kill/shrike/internal/frontend"
	"github.com/rs/zerolog"
)

func TestDevelopmentReloadStreamThroughCaddy(t *testing.T) {
	port := freePort(t)
	cfg := testConfig()
	cfg.Address = fmt.Sprintf("127.0.0.1:%d", port)
	surfaces := testSurfaces()
	surfaces[SurfaceFrontend] = frontend.New(frontend.Options{})
	manager := New(surfaces, zerolog.Nop())
	if err := manager.Start(context.Background(), cfg); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = manager.Close() })
	ctx, cancel := context.WithTimeout(t.Context(), 3*time.Second)
	defer cancel()
	request, err := http.NewRequestWithContext(ctx, http.MethodGet, fmt.Sprintf("http://127.0.0.1:%d/__dev/reload", port), nil)
	if err != nil {
		t.Fatal(err)
	}
	response, err := http.DefaultClient.Do(request)
	if err != nil {
		t.Fatal(err)
	}
	defer response.Body.Close()
	if response.StatusCode != http.StatusOK || !strings.HasPrefix(response.Header.Get("Content-Type"), "text/event-stream") {
		t.Fatalf("reload response = %d %v", response.StatusCode, response.Header)
	}
	reader := bufio.NewReader(response.Body)
	for _, want := range []string{"retry: 1000\n", "event: build\n"} {
		line, err := reader.ReadString('\n')
		if err != nil || line != want {
			t.Fatalf("line = %q, error = %v, want %q", line, err, want)
		}
	}
	if line, err := reader.ReadString('\n'); err != nil || !strings.HasPrefix(line, "data: ") {
		t.Fatalf("build event = %q, error = %v", line, err)
	}
	cancel()
}
