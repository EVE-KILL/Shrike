package api

import (
	"context"
	"net/http"

	"github.com/danielgtaylor/huma/v2"
)

// Health is the liveness payload. It reports what build is answering, not
// whether the database is reachable — a readiness probe that fails on a
// transient database blip takes the pod out of rotation and makes an outage
// worse. Dependency checks belong in `shrike doctor`, which is allowed to be
// slow and is read by a human.
type Health struct {
	Status  string `json:"status" example:"ok" doc:"Always \"ok\" — reaching this handler is the check"`
	Version string `json:"version" example:"1.4.0" doc:"Release version this process was built from"`
	Commit  string `json:"commit" example:"dcfc9af" doc:"Git commit this process was built from"`
}

type healthOutput struct {
	Body Health
}

// registerHealth mounts GET /health on a surface.
//
// Mounted on both Huma surfaces so each API documents and serves a response
// whose generated schema link belongs to that surface. Ingress sends the
// public hostname through the public surface, then uses the private surface as
// a path-only fallback so Kubernetes can still probe by pod IP without DNS.
func registerHealth(a huma.API, opts Options) {
	huma.Register(a, huma.Operation{
		OperationID: "health",
		Method:      http.MethodGet,
		Path:        "/health",
		Summary:     "Liveness check",
		Description: "Reports the running build. Does not touch the database or Redis.",
		Tags:        []string{"meta"},
	}, func(context.Context, *struct{}) (*healthOutput, error) {
		return &healthOutput{Body: Health{
			Status:  "ok",
			Version: opts.Version,
			Commit:  opts.Commit,
		}}, nil
	})
}
