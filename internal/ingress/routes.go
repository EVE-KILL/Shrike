package ingress

import (
	"errors"
	"fmt"
	"net"
	"net/http"
	"path"
	"strings"
)

const (
	SurfaceSameOrigin = "same-origin"
	SurfaceWS         = "ws"
	SurfaceFrontend   = "frontend"
)

type legacyRoute struct {
	handler http.Handler
	prefix  string
	root    string
}

func newHandler(legacyDomain string, surfaces map[string]http.Handler) (http.Handler, error) {
	for _, name := range []string{SurfaceSameOrigin, SurfaceWS, SurfaceFrontend} {
		if surfaces[name] == nil {
			return nil, fmt.Errorf("ingress surface %q has no handler", name)
		}
	}
	site, ws, frontend := surfaces[SurfaceSameOrigin], surfaces[SurfaceWS], surfaces[SurfaceFrontend]
	domain := strings.ToLower(strings.TrimSuffix(strings.TrimSpace(legacyDomain), "."))
	if strings.ContainsAny(domain, "/\\:* \t\r\n") {
		return nil, errors.New("ingress legacy domain must be a hostname without a port")
	}
	aliases := map[string]legacyRoute{}
	if domain != "" {
		aliases["api."+domain] = legacyRoute{handler: site, prefix: "/api"}
		aliases["images."+domain] = legacyRoute{handler: site, prefix: "/images"}
		aliases["ws."+domain] = legacyRoute{handler: ws, prefix: "/ws"}
		aliases["mcp."+domain] = legacyRoute{handler: site, prefix: "/api", root: "/api/mcp"}
	}
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Cloudflared preserves Host. Use it consistently for tenant routing and schema URLs.
		r.Header.Set("X-Forwarded-Host", r.Host)
		requestPath := strings.ToLower(path.Clean(r.URL.Path))
		if (requestPath == "/health" || requestPath == "/ready") && !strings.HasSuffix(r.URL.Path, "/") {
			site.ServeHTTP(w, r)
			return
		}
		host := r.Host
		if hostname, _, err := net.SplitHostPort(host); err == nil {
			host = hostname
		}
		if alias, ok := aliases[strings.ToLower(host)]; ok {
			alias.handler.ServeHTTP(w, rewriteLegacyPath(r, alias))
			return
		}
		segment, _, _ := strings.Cut(strings.TrimPrefix(requestPath, "/"), "/")
		switch segment {
		case "ws":
			ws.ServeHTTP(w, r)
		case "api", "auth", "images":
			site.ServeHTTP(w, r)
		default:
			frontend.ServeHTTP(w, r)
		}
	}), nil
}

func rewriteLegacyPath(r *http.Request, route legacyRoute) *http.Request {
	originalPath := r.URL.Path
	if originalPath == route.prefix || strings.HasPrefix(originalPath, route.prefix+"/") {
		return r
	}
	// Clone the request to preserve the original path in access logs and keep
	// the query, escaped path, headers and body intact through the alias.
	request := r.Clone(r.Context())
	if originalPath == "" || originalPath == "/" {
		request.URL.Path = route.prefix + "/"
		if route.root != "" {
			request.URL.Path = route.root
		}
		request.URL.RawPath = ""
	} else {
		request.URL.Path = route.prefix + originalPath
		if r.URL.RawPath != "" {
			request.URL.RawPath = route.prefix + r.URL.RawPath
		}
	}
	request.RequestURI = request.URL.RequestURI()
	return request
}
