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
// Mounted on every Huma surface rather than one, because a probe that only
// works through one hostname stops being a check of the process and becomes a
// check of the routing table. Kubernetes reaches it by pod IP with no Host
// header at all, which is why the ingress route for /health matches on path
// alone and ahead of every host route.
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
