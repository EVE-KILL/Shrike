package api

import (
	"net/http"
	"time"

	"github.com/danielgtaylor/huma/v2"
)

// registerAccountRoutes installs account preferences, account diagnostics,
// announcements, and reply notifications. Authentication itself remains in
// registerAuthRoutes so either domain can be tested independently.
func registerAccountRoutes(a huma.API, opts Options) {
	registerAccountServiceRoutes(a, opts, newAccountService(opts))
}

func registerAccountServiceRoutes(
	a huma.API,
	opts Options,
	service *accountService,
) {
	if a.OpenAPI().Components.SecuritySchemes == nil {
		a.OpenAPI().Components.SecuritySchemes =
			make(map[string]*huma.SecurityScheme)
	}
	if a.OpenAPI().Components.SecuritySchemes["eveSession"] == nil {
		a.OpenAPI().Components.SecuritySchemes["eveSession"] =
			&huma.SecurityScheme{
				Type: "apiKey", In: "cookie", Name: authSessionCookie,
				Description: "EVE-KILL browser session for account and admin operations.",
			}
	}
	requiredSession := []map[string][]string{{"eveSession": {}}}
	optionalSession := []map[string][]string{{}, {"eveSession": {}}}

	preferences := service.preferencesHandler()
	registerLegacy(a, huma.Operation{
		OperationID: "account-preferences",
		Method:      http.MethodGet,
		Path:        "/me/preferences",
		Summary:     "Current account preferences",
		Tags:        []string{"account", "settings"},
		Security:    requiredSession,
	}, preferences)
	registerLegacy(a, documentJSONBody[accountPreferencesBody](a, huma.Operation{
		OperationID: "account-preferences-update",
		Method:      http.MethodPut,
		Path:        "/me/preferences",
		Summary:     "Update account preferences",
		Description: "Updates default tabs, theme colors, and board preferences in one request.",
		Tags:        []string{"account", "settings"},
		Security:    requiredSession,
	}), service.savePreferencesHandler(preferenceWriteCombined))

	boards := service.boardsHandler()
	registerLegacy(a, huma.Operation{
		OperationID: "account-boards",
		Method:      http.MethodGet,
		Path:        "/me/boards",
		Summary:     "Available custom killboards",
		Tags:        []string{"account", "boards"},
		Security:    optionalSession,
	}, boards)

	overview := service.overviewHandler()
	registerLegacy(a, huma.Operation{
		OperationID: "account-overview",
		Method:      http.MethodGet,
		Path:        "/me/overview",
		Summary:     "Account and ESI overview",
		Tags:        []string{"account", "settings"},
		Security:    requiredSession,
	}, overview)

	descriptions := service.manageableEntitiesHandler()
	registerLegacy(a, huma.Operation{
		OperationID: "account-descriptions",
		Method:      http.MethodGet,
		Path:        "/me/descriptions",
		Summary:     "Descriptions managed by this account",
		Tags:        []string{"account", "settings"},
		Security:    requiredSession,
	}, descriptions)
	registerLegacy(a, documentJSONBody[accountDescriptionBody](a, huma.Operation{
		OperationID: "account-description-update",
		Method:      http.MethodPut,
		Path:        "/me/descriptions",
		Summary:     "Submit or clear an entity description",
		Tags:        []string{"account", "settings"},
		Security:    requiredSession,
	}), service.saveDescriptionHandler())

	esiMetrics := service.esiMetricsHandler()
	registerLegacy(a, huma.Operation{
		OperationID: "account-esi",
		Method:      http.MethodGet,
		Path:        "/me/esi",
		Summary:     "Account ESI request metrics",
		Tags:        []string{"account", "esi"},
		Security:    requiredSession,
	}, esiMetrics)
	esiLogs := service.esiLogsHandler()
	registerLegacy(a, huma.Operation{
		OperationID: "account-esi-logs",
		Method:      http.MethodGet,
		Path:        "/me/esi/logs",
		Summary:     "Account ESI request log",
		Tags:        []string{"account", "esi"},
		Security:    requiredSession,
	}, esiLogs)

	registerCanonicalAnnouncementsRoute(a, opts, service)
	registerCanonicalAnnouncementAccountRoutes(a, service, requiredSession)

	activeAnnouncements := routeJSONCacheBy(
		opts,
		30*time.Second,
		"public, max-age=30, s-maxage=30",
		func(*legacyRequest) string { return "announcements:active" },
		service.activeAnnouncementsHandler(),
	)
	dismissedAnnouncements := service.dismissedAnnouncementsHandler()
	dismissAnnouncement := service.dismissAnnouncementHandler()

	notificationReplies := service.notificationRepliesHandler()
	registerLegacy(a, documentJSONBody[accountNotificationsReadBody](a, huma.Operation{
		OperationID: "account-notification-replies",
		Method:      http.MethodGet,
		Path:        "/me/notifications/replies",
		Summary:     "Replies to this account's comments",
		Tags:        []string{"account", "notifications"},
		Security:    requiredSession,
	}), notificationReplies)
	markNotificationsRead := service.markNotificationsReadHandler()
	registerLegacy(a, documentJSONBody[accountNotificationsReadBody](a, huma.Operation{
		OperationID: "account-notification-read-cursor",
		Method:      http.MethodPut,
		Path:        "/me/notifications/read-cursor",
		Summary:     "Advance the reply notification read cursor",
		Tags:        []string{"account", "notifications"},
		Security:    requiredSession,
	}), markNotificationsRead)

	// Compatibility aliases are limited to routes the current Nuxt application
	// actually calls. The copied frontend can move to the canonical paths
	// without carrying every historical server endpoint into the Go service.
	registerLegacy(a, huma.Operation{
		OperationID: "user-preferences-compat",
		Method:      http.MethodGet,
		Path:        "/user/preferences",
		Summary:     "Current account preferences",
		Tags:        []string{"account", "settings"},
		Security:    requiredSession,
	}, preferences)
	registerLegacy(a, documentJSONBody[accountPreferencesBody](a, huma.Operation{
		OperationID: "user-preferences-update-compat",
		Method:      http.MethodPut,
		Path:        "/user/preferences",
		Summary:     "Update default tabs",
		Tags:        []string{"account", "settings"},
		Security:    requiredSession,
	}), service.savePreferencesHandler(preferenceWriteDefaultTabs))
	registerLegacy(a, documentJSONBody[accountPreferencesBody](a, huma.Operation{
		OperationID: "user-theme-update-compat",
		Method:      http.MethodPut,
		Path:        "/user/theme",
		Summary:     "Update account theme",
		Tags:        []string{"account", "settings"},
		Security:    requiredSession,
	}), service.savePreferencesHandler(preferenceWriteTheme))
	registerLegacy(a, documentJSONBody[accountPreferencesBody](a, huma.Operation{
		OperationID: "user-boards-update-compat",
		Method:      http.MethodPut,
		Path:        "/user/boards",
		Summary:     "Update board preferences",
		Tags:        []string{"account", "boards"},
		Security:    requiredSession,
	}), service.savePreferencesHandler(preferenceWriteBoards))
	registerLegacy(a, huma.Operation{
		OperationID: "boards-mine-compat",
		Method:      http.MethodGet,
		Path:        "/boards/mine",
		Summary:     "Available custom killboards",
		Tags:        []string{"account", "boards"},
		Security:    optionalSession,
	}, boards)
	registerLegacy(a, huma.Operation{
		OperationID: "user-overview-compat",
		Method:      http.MethodGet,
		Path:        "/user/overview",
		Summary:     "Account and ESI overview",
		Tags:        []string{"account", "settings"},
		Security:    requiredSession,
	}, overview)
	registerLegacy(a, huma.Operation{
		OperationID: "user-manageable-entities-compat",
		Method:      http.MethodGet,
		Path:        "/user/manageable-entities",
		Summary:     "Descriptions managed by this account",
		Tags:        []string{"account", "settings"},
		Security:    requiredSession,
	}, descriptions)
	registerLegacy(a, documentJSONBody[accountDescriptionBody](a, huma.Operation{
		OperationID: "user-description-update-compat",
		Method:      http.MethodPut,
		Path:        "/user/descriptions",
		Summary:     "Submit or clear an entity description",
		Tags:        []string{"account", "settings"},
		Security:    requiredSession,
	}), service.saveDescriptionHandler())
	registerLegacy(a, huma.Operation{
		OperationID: "user-esi-compat",
		Method:      http.MethodGet,
		Path:        "/user/esi",
		Summary:     "Account ESI request metrics",
		Tags:        []string{"account", "esi"},
		Security:    requiredSession,
	}, esiMetrics)
	registerLegacy(a, huma.Operation{
		OperationID: "user-esi-logs-compat",
		Method:      http.MethodGet,
		Path:        "/user/esi-logs",
		Summary:     "Account ESI request log",
		Tags:        []string{"account", "esi"},
		Security:    requiredSession,
	}, esiLogs)
	registerLegacy(a, huma.Operation{
		OperationID: "announcements-active-compat",
		Method:      http.MethodGet,
		Path:        "/announcements/active",
		Summary:     "Active site announcements",
		Tags:        []string{"announcements"},
	}, activeAnnouncements)
	registerLegacy(a, huma.Operation{
		OperationID: "announcements-dismissed-compat",
		Method:      http.MethodGet,
		Path:        "/announcements/dismissed",
		Summary:     "Announcements dismissed by this account",
		Tags:        []string{"account", "announcements"},
		Security:    requiredSession,
	}, dismissedAnnouncements)
	registerLegacy(a, huma.Operation{
		OperationID: "announcement-dismiss-compat",
		Method:      http.MethodPost,
		Path:        "/announcements/{id}/dismiss",
		Summary:     "Dismiss an announcement",
		Tags:        []string{"account", "announcements"},
		Security:    requiredSession,
	}, dismissAnnouncement)
	registerLegacy(a, huma.Operation{
		OperationID: "notification-replies-compat",
		Method:      http.MethodGet,
		Path:        "/notifications/replies",
		Summary:     "Replies to this account's comments",
		Tags:        []string{"account", "notifications"},
		Security:    requiredSession,
	}, notificationReplies)
	registerLegacy(a, documentJSONBody[accountNotificationsReadBody](a, huma.Operation{
		OperationID: "notification-mark-read-compat",
		Method:      http.MethodPost,
		Path:        "/notifications/mark-read",
		Summary:     "Advance the reply notification read cursor",
		Tags:        []string{"account", "notifications"},
		Security:    requiredSession,
	}), markNotificationsRead)
}
