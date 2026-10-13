package requeststats

import (
	"io"
	"net/http"

	"github.com/felixge/httpsnoop"
)

// Middleware instruments non-streaming API responses in development. The
// frontend has its own timing boundaries for data loading and rendering.
// httpsnoop preserves the optional ResponseWriter interfaces used by handlers.
func Middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/api/feed/stream" {
			next.ServeHTTP(w, r)
			return
		}
		ctx, stats := Start(r.Context())
		r = r.WithContext(ctx)
		committed := false
		finish := func() {
			if committed {
				return
			}
			committed = true
			w.Header().Set("Server-Timing", stats.Snapshot().ServerTiming())
			// Browser-cached diagnostic headers describe an earlier request.
			w.Header().Set("Cache-Control", "private, no-store")
		}
		wrapped := httpsnoop.Wrap(w, httpsnoop.Hooks{
			ReadFrom: func(next httpsnoop.ReadFromFunc) httpsnoop.ReadFromFunc {
				return func(reader io.Reader) (int64, error) { finish(); return next(reader) }
			},
			Flush: func(next httpsnoop.FlushFunc) httpsnoop.FlushFunc {
				return func() { finish(); next() }
			},
			WriteHeader: func(next httpsnoop.WriteHeaderFunc) httpsnoop.WriteHeaderFunc {
				return func(status int) {
					if status >= 200 {
						finish()
					}
					next(status)
				}
			},
			Write: func(next httpsnoop.WriteFunc) httpsnoop.WriteFunc {
				return func(data []byte) (int, error) { finish(); return next(data) }
			},
		})
		next.ServeHTTP(wrapped, r)
		if !committed {
			finish()
		}
	})
}
