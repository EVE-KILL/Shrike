package frontend

import (
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

type pageStatusError int

func (e pageStatusError) Error() string  { return "private backend detail" }
func (e pageStatusError) GetStatus() int { return int(e) }

func TestPageReaderPreservesHTTPErrorStatus(t *testing.T) {
	for _, status := range []int{400, 401, 403, 404, 429, 503} {
		t.Run(http.StatusText(status), func(t *testing.T) {
			s := &site{options: Options{Production: true}}
			w := httptest.NewRecorder()
			s.pageHandler(func(*http.Request) (Page, error) {
				return Page{}, fmt.Errorf("reader: %w", pageStatusError(status))
			}).ServeHTTP(w, httptest.NewRequest(http.MethodGet, "https://eve-kill.com/item/74434/market", nil))
			if w.Code != status || strings.Contains(w.Body.String(), "private backend detail") {
				t.Fatalf("status/detail handling: status=%d body=%s", w.Code, w.Body.String())
			}
			if !strings.Contains(w.Header().Get("Cache-Control"), "no-store") {
				t.Fatal("error page can enter the shared cache")
			}
		})
	}
	if errors.Is(pageStatusError(404), ErrNotFound) {
		t.Fatal("test must exercise a reader HTTP error, not the frontend sentinel")
	}
}
