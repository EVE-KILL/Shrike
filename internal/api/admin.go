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
	} {
		operation := huma.Operation{
			OperationID: route.id,
			Method:      route.method,
			Path:        route.path,
			Summary:     route.summary,
			Tags:        []string{"admin"},
			Security:    requiredSession,
		}
		if route.id == "admin-users-set-discord" {
			operation = documentJSONBody[adminSetDiscordDocument](a, operation)
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
