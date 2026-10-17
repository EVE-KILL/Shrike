package ingress

import (
	"errors"
	"fmt"
	"net/http"
	"strings"

	"github.com/caddyserver/caddy/v2"
	"github.com/caddyserver/caddy/v2/modules/caddyhttp"

	// Only the modules the generated config actually names are linked. Caddy's
	// standard bundle carries templates, tracing, an ACME server and a dozen
	// other handlers that Shrike's closed config could never reach, and every
	// one of them is binary weight and dependency surface for nothing.
	//
	// caddyhttp already brings in the TLS and PKI apps. File storage is linked
	// explicitly so the development CA and leaf certificates stay inside
	// Shrike's data directory instead of Caddy's process-wide default.
	_ "github.com/caddyserver/caddy/v2/modules/filestorage"
)

func init() {
	caddy.RegisterModule(shrikeHandler{})
	caddy.RegisterModule(accessLogHandler{})
}

// shrikeHandler hands a request to one of Shrike's surfaces.
//
// This is the whole reason Caddy is embedded rather than run alongside. The
// handler is the terminal element of a Caddy route, so a request arriving at
// the edge reaches Huma through a function call — no loopback socket, no
// second serialisation of a body Caddy has already parsed.
//
// It carries a surface name rather than the handler itself because Caddy
// constructs modules from JSON and cannot be handed a Go value. Provision
// resolves the name against the active Manager, which is what the process-wide
// activeManager pointer exists for.
type shrikeHandler struct {
	// Surface selects which of Manager's handlers serves the request.
	Surface string `json:"surface,omitempty"`
	// PathPrefix adapts legacy subdomain URLs before invoking the surface.
	PathPrefix string `json:"path_prefix,omitempty"`
	// RootPath is the legacy MCP root alias, whose canonical URL is /api/mcp.
	RootPath string `json:"root_path,omitempty"`

	handler http.Handler
}

func (shrikeHandler) CaddyModule() caddy.ModuleInfo {
	return caddy.ModuleInfo{
		ID:  "http.handlers.shrike",
		New: func() caddy.Module { return new(shrikeHandler) },
	}
}

// Provision binds the named surface.
//
// A name with no registered handler fails the config load rather than
// defaulting to something. Caddy rejects the whole configuration and Manager
// rolls back to the previous one, so a typo'd surface is a startup error with
// the bad name in it — not a route that quietly answers 404 in production.
func (h *shrikeHandler) Provision(caddy.Context) error {
	m := activeManager.Load()
	if m == nil {
		return errors.New("ingress: no active manager; the shrike handler cannot be used outside a running Manager")
	}
	handler, ok := m.surfaces[h.Surface]
	if !ok {
		return fmt.Errorf("ingress: unknown surface %q", h.Surface)
	}
	if handler == nil {
		return fmt.Errorf("ingress: surface %q is registered with a nil handler", h.Surface)
	}
	h.handler = handler
	return nil
}

func (h *shrikeHandler) ServeHTTP(w http.ResponseWriter, r *http.Request, _ caddyhttp.Handler) error {
	// cloudflared preserves the public Host. Do not let a client-supplied
	// forwarding header override it now that Nginx no longer replaces it.
	r.Header.Set("X-Forwarded-Host", r.Host)
	if h.PathPrefix != "" {
		path := r.URL.Path
		if path == "" || path == "/" {
			if h.RootPath != "" {
				path = h.RootPath
			} else {
				path = h.PathPrefix + "/"
			}
		} else if path != h.PathPrefix && !strings.HasPrefix(path, h.PathPrefix+"/") {
			path = h.PathPrefix + path
		}
		if path != r.URL.Path {
			// Preserve Host, query, body and context. Clone the URL so the access
			// logger still records the client's public path. Already-prefixed
			// requests keep working during the rollout behind the old proxy.
			originalPath := r.URL.Path
			r = r.Clone(r.Context())
			r.URL.Path = path
			if r.URL.RawPath != "" {
				if originalPath == "" || originalPath == "/" {
					r.URL.RawPath = ""
				} else {
					r.URL.RawPath = h.PathPrefix + r.URL.RawPath
				}
			}
			r.RequestURI = r.URL.RequestURI()
		}
	}
	h.handler.ServeHTTP(w, r)
	return nil
}

var (
	_ caddy.Provisioner           = (*shrikeHandler)(nil)
	_ caddyhttp.MiddlewareHandler = (*shrikeHandler)(nil)
)
