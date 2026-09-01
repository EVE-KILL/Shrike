package api

import (
	"context"
	"net/http"

	"github.com/danielgtaylor/huma/v2"
)

type adminService struct {
	opts Options
	auth *authService
}

// registerAdminRoutes installs the operational administration surface. Every
// route is session-authenticated and also verifies the principal's admin bit;
// OpenAPI security alone is documentation, not authorization.
func registerAdminRoutes(a huma.API, opts Options) {
	opts.DB = primaryDatabase(opts)
	service := &adminService{opts: opts, auth: newAuthService(opts)}
	requiredSession := []map[string][]string{{"eveSession": {}}}
	for _, route := range []struct {
		id, method, path, summary string
		handler                   legacyHandler
	}{
		{
			"admin-overview", http.MethodGet, "/admin/overview",
			"Administration overview", service.overviewHandler(),
		},
		{
			"admin-users-list", http.MethodGet, "/admin/users",
			"List users", service.usersHandler(),
		},
		{
			"admin-users-detail", http.MethodGet, "/admin/users/{id}",
			"User administration detail", service.userDetailHandler(),
		},
		{
			"admin-users-set-discord", http.MethodPost,
			"/admin/users/{id}/set-discord",
			"Set a user's Discord identity", service.setDiscordHandler(),
		},
		{
			"admin-users-toggle-admin", http.MethodPost,
			"/admin/users/{id}/toggle-admin",
			"Toggle a user's administrator status", service.toggleAdminHandler(),
		},
		{
			"admin-esi-overview", http.MethodGet, "/admin/esi",
			"ESI request overview", service.esiOverviewHandler(),
		},
		{
			"admin-esi-logs", http.MethodGet, "/admin/esi-logs",
			"Search ESI request logs", service.esiLogsHandler(),
		},
		{
			"admin-esi-entities", http.MethodGet, "/admin/esi-entities",
			"Search entities with ESI request logs", service.esiEntitiesHandler(),
		},
		{
			"admin-river-overview", http.MethodGet, "/admin/river",
			"River queues and workers", service.riverOverviewHandler(),
		},
		{
			"admin-river-jobs", http.MethodGet, "/admin/river/jobs",
			"List River jobs", service.riverJobsHandler(),
		},
		{
			"admin-river-job", http.MethodGet, "/admin/river/jobs/{id}",
			"Get a River job", service.riverJobHandler(),
		},
		{
			"admin-river-job-action", http.MethodPost, "/admin/river/jobs/{id}/action",
			"Cancel, retry, or delete a River job", service.riverJobActionHandler(),
		},
		{
			"admin-river-queue-action", http.MethodPost, "/admin/river/queues/{name}/action",
			"Pause or resume a River queue", service.riverQueueActionHandler(),
		},
		{
			"admin-river-queue-clear", http.MethodPost, "/admin/river/queues/{name}/clear",
			"Delete matching jobs from a River queue", service.riverClearHandler(),
		},
	} {
		operation := huma.Operation{
			OperationID: route.id,
			Method:      route.method,
			Path:        route.path,
			Summary:     route.summary,
			Tags:        []string{"admin"},
			Security:    requiredSession,
		}
		switch route.id {
		case "admin-users-set-discord":
			operation = documentJSONBody[adminSetDiscordDocument](a, operation)
		case "admin-river-job-action":
			operation = documentJSONBody[adminRiverJobActionBody](a, operation)
		case "admin-river-queue-action":
			operation = documentJSONBody[adminRiverQueueActionBody](a, operation)
		case "admin-river-queue-clear":
			operation = documentJSONBody[adminRiverClearBody](a, operation)
		}
		registerLegacy(a, operation, route.handler)
	}
}

func (s *adminService) requireAdmin(
	ctx context.Context,
	req *legacyRequest,
	mutation bool,
) (*Principal, error) {
	setAccountNoStore(req.Huma)
	if mutation {
		if err := requireSameOriginMutation(req.Huma); err != nil {
			return nil, err
		}
	}
	principal, err := s.auth.requirePrincipal(ctx, req)
	if err != nil {
		return nil, err
	}
	if !principal.IsAdmin {
		return nil, apiError(
			http.StatusForbidden, "Administrator access required",
		)
	}
	return principal, nil
}
