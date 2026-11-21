package ingress

import (
	"bufio"
	"context"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/eve-kill/shrike/internal/frontend"
)

func TestDevelopmentReloadStreamThroughIngress(t *testing.T) {
	surfaces := testSurfaces()
	surfaces[SurfaceFrontend] = frontend.New(frontend.Options{})
	base := startTestServer(t, testConfig(), surfaces)
	ctx, cancel := context.WithTimeout(t.Context(), 3*time.Second)
	defer cancel()
	request, err := http.NewRequestWithContext(ctx, http.MethodGet, base+"/__dev/reload", nil)
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
