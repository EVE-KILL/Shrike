package frontend

import (
	"fmt"
	"net/http"
	"os"
	"strconv"
	"time"
)

// Air replaces the Go process after a successful build. A fresh process ID is
// enough to tell an already-open browser that its HTML and assets are stale.
var devInstanceID = strconv.FormatInt(time.Now().UnixNano(), 36) + "-" + strconv.Itoa(os.Getpid())

// A bounded read bootstraps new/BFCache documents and supports browsers that
// cannot share a worker. It never reserves a long-lived HTTP connection.
func (s *site) liveReloadVersion(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	w.Header().Set("Cache-Control", "private, no-store")
	w.Header().Set("X-Robots-Tag", "noindex")
	_, _ = fmt.Fprintln(w, devInstanceID)
}

// liveReload streams the current process identity. EventSource reconnects
// automatically after Air restarts the server; the browser reloads only when
// the new identity differs from the one embedded in its HTML.
func (s *site) liveReload(w http.ResponseWriter, r *http.Request) {
	if s.options.Production {
		http.NotFound(w, r)
		return
	}
	controller := http.NewResponseController(w)
	w.Header().Set("Content-Type", "text/event-stream; charset=utf-8")
	w.Header().Set("Cache-Control", "private, no-store")
	w.Header().Set("X-Robots-Tag", "noindex")
	w.Header().Set("X-Accel-Buffering", "no")
	_, _ = fmt.Fprintf(w, "retry: 1000\nevent: build\ndata: %s\n\n", devInstanceID)
	if err := controller.Flush(); err != nil {
		return
	}

	// A comment keeps idle proxies from silently closing the stream. It has no
	// browser effect and does not cause a page reload.
	ticker := time.NewTicker(20 * time.Second)
	defer ticker.Stop()
	for {
		select {
		case <-r.Context().Done():
			return
		case <-ticker.C:
			if _, err := w.Write([]byte(": keep-alive\n\n")); err != nil {
				return
			}
			if err := controller.Flush(); err != nil {
				return
			}
		}
	}
}
