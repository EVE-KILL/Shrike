// Package api builds Shrike's HTTP surfaces.
//
// A surface is one coherent audience for one set of endpoints. Shrike serves
// four of them from a single process, and they are separate APIs rather than
// route groups on one API for a reason: the split is what keeps the public
// OpenAPI document from listing the frontend's private endpoints. A group is a
// convention somebody has to remember; a separate API is a wall.
//
//	private  eve-kill.com/api, /auth   the Nuxt frontend, and nothing else
//	public   api.eve-kill.com          third-party consumers, documented
//	ws       ws.eve-kill.com           live killmail stream
//	images   images.eve-kill.com       entity portraits and renders
//
// Only the first two are Huma. WebSocket upgrades hijack the connection and
// the image server streams bytes, so neither gains anything from an operation
// wrapper — they are plain handlers that happen to live in the same map.
//
// Every surface here returns an http.Handler, which is all the ingress layer
// wants. Nothing in this package knows Caddy exists.
package api

import (
	"net/http"

	"github.com/danielgtaylor/huma/v2"
	"github.com/danielgtaylor/huma/v2/adapters/humago"
)

// Options carries what the surfaces need from the process.
//
// Deliberately not the whole *config.Config: a surface that wants a database
// pool should be handed a pool, so the dependency shows up in the signature
// rather than being fished out of a bag at request time.
type Options struct {
	Version string
	Commit  string
}

// Private is the frontend's API.
//
// "Private" is a policy rather than a mechanism — the endpoints are reachable
// from the internet, they are simply not promised to anyone but our own
// frontend, so their shapes can change without a deprecation cycle. Keeping
// them off the public document is what makes that claim credible.
//
// The OpenAPI document is still served, at /api/openapi.json. Endpoint shapes
// are not secrets, the frontend's generated client is built from this spec,
// and hiding the catalogue while leaving the endpoints open would buy nothing.
// Authorisation belongs on the operations.
func Private(opts Options) http.Handler {
	mux := http.NewServeMux()

	cfg := huma.DefaultConfig("EVE-KILL Frontend API", opts.Version)
	cfg.Info.Description = "Endpoints backing the EVE-KILL frontend. " +
		"Not a public contract: shapes change with the UI and without notice. " +
		"Use api.eve-kill.com instead."
	// Every meta path sits under /api because that is the only prefix the
	// ingress routes to this surface. Huma stamps a $schema URL into each
	// response body built from SchemasPath, so leaving it at the default
	// would put a link to /schemas/... in every payload — a path this surface
	// never receives, and therefore a dead link in every response.
	cfg.OpenAPIPath = "/api/openapi"
	cfg.DocsPath = "/api/docs"
	cfg.SchemasPath = "/api/schemas"

	a := humago.New(mux, cfg)
	registerHealth(a, opts)

	return mux
}

// Public is api.eve-kill.com.
//
// Stubbed to /health for now. It exists this early so the host routing is
// proven end to end — a surface added later is a change to the route table,
// which is exactly the part worth having under test before it carries traffic.
func Public(opts Options) http.Handler {
	mux := http.NewServeMux()

	cfg := huma.DefaultConfig("EVE-KILL API", opts.Version)
	cfg.Info.Description = "The public EVE-KILL API."

	a := humago.New(mux, cfg)
	registerHealth(a, opts)

	return mux
}

// WS is ws.eve-kill.com.
//
// A placeholder that answers honestly. Registered rather than omitted so the
// host route exists and a request to it fails as "not built yet" instead of
// falling through to the frontend and rendering a 404 page at a WebSocket
// client.
func WS(Options) http.Handler {
	return notImplemented("websocket")
}

// Images is images.eve-kill.com. Placeholder, as WS.
func Images(Options) http.Handler {
	return notImplemented("image server")
}

func notImplemented(surface string) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/problem+json")
		w.WriteHeader(http.StatusNotImplemented)
		_, _ = w.Write([]byte(`{"title":"Not Implemented","status":501,` +
			`"detail":"the ` + surface + ` surface is routed but not yet ported"}`))
	})
}
