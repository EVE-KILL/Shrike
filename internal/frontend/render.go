package frontend

import (
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"time"

	"github.com/a-h/templ"
	"github.com/eve-kill/shrike/internal/requeststats"
	"github.com/eve-kill/shrike/internal/telemetry"
	"github.com/rs/zerolog/log"
)

var ErrNotFound = errors.New("frontend: page not found")

// Page separates fetching and presentation. Loaders call Go services directly;
// templates receive finished view models and never perform database or HTTP IO.
type Page struct {
	Metadata Metadata
	Content  templ.Component
	Status   int
	// RedirectPath lets a loader resolve a canonical target inside the same
	// measured request context as ordinary SSR data loading.
	RedirectPath string
	// SharedCacheTTL is an explicit promise that the page contains no personal
	// data. Requests with cookies or Authorization always bypass shared caching.
	SharedCacheTTL time.Duration
}

type pageLoader func(*http.Request) (Page, error)

func (s *site) pageHandler(load pageLoader) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !s.options.Production {
			ctx, _ := requeststats.Start(r.Context())
			r = r.WithContext(ctx)
		}
		loaded := time.Now()
		parentCtx := r.Context()
		loadCtx, loadSpan := telemetry.StartSpan(parentCtx, "ui.load", "Page data")
		r = r.WithContext(loadCtx)
		page, err := load(r)
		telemetry.Observe(r.Context(), "frontend.load", time.Since(loaded), err)
		telemetry.FinishSpan(loadSpan, err)
		r = r.WithContext(parentCtx)
		requeststats.From(r.Context()).AddLoad(time.Since(loaded))
		if err == nil && page.RedirectPath != "" {
			if _, err = canonicalURL(page.RedirectPath); err == nil {
				http.Redirect(w, r, page.RedirectPath, http.StatusFound)
				return
			}
		}
		if err == nil {
			err = s.render(w, r, page)
		}
		if err == nil {
			return
		}
		status := http.StatusInternalServerError
		title, description := "Something went wrong", "We could not load this page. Please try again."
		var statusError interface{ GetStatus() int }
		if errors.As(err, &statusError) && statusError.GetStatus() >= 400 && statusError.GetStatus() < 600 {
			status = statusError.GetStatus()
		}
		if errors.Is(err, ErrNotFound) || status == http.StatusNotFound {
			status, title, description = http.StatusNotFound, "Page not found", "This page is not available."
		} else if status < http.StatusInternalServerError {
			title, description = http.StatusText(status), "We could not complete this request."
		} else {
			telemetry.Capture(r.Context(), err)
			log.Error().Ctx(r.Context()).Bool("sentry_handled", true).Err(err).Str("path", r.URL.Path).Msg("render frontend page")
		}
		if renderErr := s.render(w, r, Page{
			Status: status, Metadata: Metadata{Title: title}, Content: errorPage(title, description),
		}); renderErr != nil {
			telemetry.Capture(r.Context(), renderErr)
			log.Error().Ctx(r.Context()).Bool("sentry_handled", true).Err(renderErr).Msg("render frontend error page")
			w.Header().Set("Cache-Control", "private, no-store")
			w.Header().Set("X-Robots-Tag", "noindex, nofollow")
			http.Error(w, http.StatusText(status), status)
		}
	})
}

func (s *site) render(w http.ResponseWriter, r *http.Request, page Page) error {
	status := page.Status
	if status == 0 {
		status = http.StatusOK
	}
	if status < 200 || status >= 600 || status == 204 || status == 304 {
		return fmt.Errorf("unsupported HTML status %d", status)
	}
	if page.Content == nil {
		return errors.New("frontend: missing page content")
	}
	meta, err := resolveMetadata(r, s.options.Production, status, page.Metadata)
	if err != nil {
		return err
	}
	buffer := templ.GetBuffer()
	defer templ.ReleaseBuffer(buffer)
	// No headers or body are committed until the complete document renders.
	renderStarted := time.Now()
	renderCtx, renderSpan := telemetry.StartSpan(r.Context(), "ui.render", "Page templates")
	err = layout(meta, page.Content, !s.options.Production).Render(renderCtx, buffer)
	telemetry.Observe(renderCtx, "frontend.render", time.Since(renderStarted), err, "kind", "page")
	telemetry.FinishSpan(renderSpan, err)
	requeststats.From(r.Context()).AddRender(time.Since(renderStarted))
	if err != nil {
		return err
	}
	cache := "private, no-store"
	if s.options.Production && canonicalHost(r.Host) && status == http.StatusOK &&
		page.SharedCacheTTL >= time.Second {
		// A shared cache can answer before this handler sees the request. Its
		// anonymous entry must therefore vary on the headers that disable
		// caching here, including browser theme/default-tab preferences.
		w.Header().Add("Vary", "Cookie")
		w.Header().Add("Vary", "Authorization")
		if r.Header.Get("Cookie") == "" && r.Header.Get("Authorization") == "" &&
			len(w.Header().Values("Set-Cookie")) == 0 {
			seconds := int64(page.SharedCacheTTL / time.Second)
			cache = fmt.Sprintf("public, max-age=0, s-maxage=%d, stale-while-revalidate=%d", seconds, seconds)
		}
	}
	w.Header().Set("Cache-Control", cache)
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Header().Set("Content-Length", strconv.Itoa(buffer.Len()))
	w.Header().Set("Content-Security-Policy", meta.CSP)
	w.Header().Set("X-Robots-Tag", meta.Robots)
	if stats := requeststats.From(r.Context()); !s.options.Production && stats != nil {
		w.Header().Set("Server-Timing", stats.Snapshot().ServerTiming())
	}
	w.WriteHeader(status)
	if r.Method != http.MethodHead {
		if _, err := w.Write(buffer.Bytes()); err != nil {
			// The response has already started; do not attempt to append an error page.
			log.Debug().Err(err).Msg("frontend response connection closed")
		}
	}
	return nil
}
