package main

import (
	"archive/tar"
	"compress/gzip"
	"context"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"regexp"
	"strings"

	"github.com/eve-kill/shrike/internal/api"
)

type remoteCoalitions struct{ origin string }

func (r remoteCoalitions) CoalitionDirectory(ctx context.Context) (api.CoalitionDirectoryData, error) {
	return fixtureRead[api.CoalitionDirectoryData](ctx, r.origin, "/api/coalitions")
}

// The disposable API predates the overview endpoint. This facade is for browser
// parity, not production timing; production uses HomepageReader's overview SQL.
func (r remoteCoalitions) CoalitionDirectoryOverview(ctx context.Context) (api.CoalitionDirectoryData, error) {
	return r.CoalitionDirectory(ctx)
}

func (r remoteCoalitions) CoalitionDetail(ctx context.Context, slug string) (api.CoalitionDetailData, bool, error) {
	data, err := fixtureRead[api.CoalitionDetailData](ctx, r.origin, "/api/coalitions/"+url.PathEscape(slug))
	var failure *fixtureHTTPError
	if errors.As(err, &failure) && failure.status == http.StatusNotFound {
		return data, false, nil
	}
	return data, err == nil, err
}

var coalitionModuleScript = regexp.MustCompile(`src="/assets/[^/"<>]+/coalitions\.js"`)
var referenceDevInstanceTag = regexp.MustCompile(`<meta[^>]*name="shrike-dev-instance"[^>]*>`)

func archivedCoalitionScripts(path string) (map[string][]byte, error) {
	return archivedComponentScripts(path, []string{"coalitions.js", "coalition-activity.js"})
}

func archivedComponentScripts(path string, names []string) (map[string][]byte, error) {
	// #nosec G304 -- callers select fixed local source archives, never request input.
	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer func() { _ = file.Close() }()
	gz, err := gzip.NewReader(file)
	if err != nil {
		return nil, err
	}
	defer func() { _ = gz.Close() }()
	return readComponentScripts(tar.NewReader(gz), names)
}

func readCoalitionScripts(archive *tar.Reader) (map[string][]byte, error) {
	return readComponentScripts(archive, []string{"coalitions.js", "coalition-activity.js"})
}

func readComponentScripts(archive *tar.Reader, names []string) (map[string][]byte, error) {
	allowed := make(map[string]bool)
	for _, name := range names {
		allowed[name] = true
	}
	scripts := make(map[string][]byte)
	for {
		header, err := archive.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, err
		}
		name := strings.TrimPrefix(header.Name, "internal/frontend/static/")
		if header.Name != "internal/frontend/static/"+name || !allowed[name] {
			continue
		}
		if header.Typeflag != tar.TypeReg || header.Size > 1<<20 {
			return nil, errors.New("invalid archived component script")
		}
		body, err := io.ReadAll(archive)
		if err != nil {
			return nil, err
		}
		scripts[name] = body
	}
	if len(scripts) != len(allowed) {
		return nil, errors.New("original component scripts are missing from source archive")
	}
	return scripts, nil
}

// The retained binary predates the coalition activity split and never registered
// its Go page routes. Opt-in comparisons reuse the actual Go component HTML and
// run the untouched scripts from the pre-migration source archive. Only those two
// named JS files are served; all other archive content stays private.
// Other reference pages/assets remain the retained binary;
// all reference API requests remain anonymous and read-only.
func coalitionReferenceRenderer(reference, renderer http.Handler, scripts map[string][]byte) http.Handler {
	return componentReferenceRenderer(reference, renderer, scripts, "coalition", coalitionModuleScript,
		func(path string) bool {
			return path == "/coalitions" || strings.HasPrefix(path, "/coalitions/") || strings.HasPrefix(path, "/_fragments/coalitions/")
		},
		[]string{"coalition-activity coalitions", "coalitions"}, "coalitions.js")
}

func componentReferenceRenderer(reference, renderer http.Handler, scripts map[string][]byte, name string, module *regexp.Regexp, match func(string) bool, controllers []string, entry string) http.Handler {
	prefix := "/__fixture/" + name + "-source/"
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if body, ok := scripts[strings.TrimPrefix(r.URL.Path, prefix)]; ok && strings.HasPrefix(r.URL.Path, prefix) {
			if r.Method != http.MethodGet && r.Method != http.MethodHead {
				w.WriteHeader(405)
				return
			}
			w.Header().Set("Content-Type", "text/javascript; charset=utf-8")
			w.Header().Set("Cache-Control", "private, no-store")
			if r.Method != http.MethodHead {
				_, _ = w.Write(body)
			}
			return
		}
		if strings.HasPrefix(r.URL.Path, "/assets/") && (r.Method == http.MethodGet || r.Method == http.MethodHead) {
			request := r.Clone(r.Context())
			request.Header.Del("Cookie")
			request.Header.Del("Authorization")
			response := httptest.NewRecorder()
			renderer.ServeHTTP(response, request)
			if response.Code != http.StatusNotFound {
				for name, values := range response.Header() {
					for _, value := range values {
						w.Header().Add(name, value)
					}
				}
				w.WriteHeader(response.Code)
				if r.Method != http.MethodHead {
					_, _ = w.Write(response.Body.Bytes())
				}
				return
			}
		}
		coalition := match(r.URL.Path)
		if !coalition || (r.Method != http.MethodGet && r.Method != http.MethodHead) {
			reference.ServeHTTP(w, r)
			return
		}
		request := r.Clone(r.Context())
		request.Header.Del("Cookie")
		request.Header.Del("Authorization")
		response := httptest.NewRecorder()
		renderer.ServeHTTP(response, request)
		for name, values := range response.Header() {
			if name == "Set-Cookie" || name == "Content-Length" {
				continue
			}
			for _, value := range values {
				w.Header().Add(name, value)
			}
		}
		body := response.Body.String()
		if module.MatchString(body) {
			body = module.ReplaceAllString(body, `src="`+prefix+entry+`"`)
		} else if strings.Contains(body, "</body>") {
			// Shared shell controllers have no individual script tag in the
			// built page. Their stripped scope still needs its original script.
			body = strings.Replace(body, "</body>", `<script type="module" src="`+prefix+entry+`"></script></body>`, 1)
		}
		// The retained reload endpoint belongs to a different process. The
		// comparison page must not repeatedly reload against its instance ID.
		body = referenceDevInstanceTag.ReplaceAllString(body, "")
		for _, controller := range controllers {
			body = strings.ReplaceAll(body, `data-controller="`+controller+`"`, `data-controller=""`)
		}
		w.WriteHeader(response.Code)
		if r.Method != http.MethodHead {
			_, _ = io.WriteString(w, body)
		}
	})
}
