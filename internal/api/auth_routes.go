package api

import (
	"context"
	"net/http"
	"net/url"
	"strconv"
	"strings"

	"github.com/danielgtaylor/huma/v2"
	"github.com/google/uuid"
)

// registerAuthRoutes installs the canonical account contract and the
// temporary Nuxt aliases in the shared API catalogue.
func registerAuthRoutes(a huma.API, opts Options) {
	service := newAuthService(opts)
	if a.OpenAPI().Components.SecuritySchemes == nil {
		a.OpenAPI().Components.SecuritySchemes =
			make(map[string]*huma.SecurityScheme)
	}
	a.OpenAPI().Components.SecuritySchemes["eveSession"] =
		&huma.SecurityScheme{
			Type: "apiKey", In: "cookie", Name: authSessionCookie,
			Description: "EVE-KILL browser session for account and admin operations.",
		}
	requiredSession := []map[string][]string{{"eveSession": {}}}
	optionalSession := []map[string][]string{{}, {"eveSession": {}}}

	registerLegacy(a, huma.Operation{
		OperationID:   "eve-login-start",
		Method:        http.MethodGet,
		Path:          "/auth/eve/start",
		Summary:       "Start EVE SSO login",
		Tags:          []string{"auth"},
		DefaultStatus: http.StatusFound,
	}, service.loginStartHandler())
	registerLegacy(a, huma.Operation{
		OperationID: "auth-login-legacy",
		Method:      http.MethodGet,
		Path:        "/auth/login",
		Summary:     "Create an EVE SSO login URL",
		Tags:        []string{"auth"},
	}, service.loginURLHandler())

	callback := service.callbackHandler()
	registerLegacy(a, huma.Operation{
		OperationID:   "eve-login-callback",
		Method:        http.MethodGet,
		Path:          "/auth/eve/callback",
		Summary:       "Complete EVE SSO login",
		Tags:          []string{"auth"},
		DefaultStatus: http.StatusFound,
	}, callback)
	registerLegacy(a, huma.Operation{
		OperationID:   "eve-login-callback-legacy",
		Method:        http.MethodGet,
		Path:          "/auth/callback",
		Summary:       "Complete EVE SSO login",
		Tags:          []string{"auth"},
		DefaultStatus: http.StatusFound,
	}, callback)

	registerLegacy(a, huma.Operation{
		OperationID: "me",
		Method:      http.MethodGet,
		Path:        "/me",
		Summary:     "Current authenticated identity",
		Tags:        []string{"auth"},
		Security:    optionalSession,
	}, service.meHandler(false))
	registerLegacy(a, huma.Operation{
		OperationID: "auth-me-legacy",
		Method:      http.MethodGet,
		Path:        "/auth/me",
		Summary:     "Current authenticated identity",
		Tags:        []string{"auth"},
		Security:    optionalSession,
	}, service.meHandler(true))
	registerLegacy(a, huma.Operation{
		OperationID: "me-settings",
		Method:      http.MethodGet,
		Path:        "/me/settings",
		Summary:     "Account settings bootstrap",
		Tags:        []string{"auth", "settings"},
		Security:    requiredSession,
	}, service.settingsHandler())
	registerLegacy(a, huma.Operation{
		OperationID: "auth-token-info-legacy",
		Method:      http.MethodGet,
		Path:        "/auth/token-info",
		Summary:     "Current ESI token summary",
		Tags:        []string{"auth", "settings"},
		Security:    requiredSession,
	}, service.tokenInfoHandler())

	logout := service.logoutHandler()
	registerLegacy(a, huma.Operation{
		OperationID: "session-delete",
		Method:      http.MethodDelete,
		Path:        "/me/session",
		Summary:     "Log out this browser session",
		Tags:        []string{"auth"},
		Security:    requiredSession,
	}, logout)
	registerLegacy(a, huma.Operation{
		OperationID: "auth-logout-legacy",
		Method:      http.MethodPost,
		Path:        "/auth/logout",
		Summary:     "Log out this browser session",
		Tags:        []string{"auth"},
		Security:    requiredSession,
	}, logout)

	sessions := service.sessionsHandler()
	registerLegacy(a, huma.Operation{
		OperationID: "sessions",
		Method:      http.MethodGet,
		Path:        "/me/sessions",
		Summary:     "List logged-in browser sessions",
		Tags:        []string{"auth", "settings"},
		Security:    requiredSession,
	}, sessions)
	registerLegacy(a, huma.Operation{
		OperationID: "user-sessions-legacy",
		Method:      http.MethodGet,
		Path:        "/user/sessions",
		Summary:     "List logged-in browser sessions",
		Tags:        []string{"auth", "settings"},
		Security:    requiredSession,
	}, sessions)

	revoke := service.revokeSessionHandler()
	registerLegacy(a, huma.Operation{
		OperationID: "session-revoke",
		Method:      http.MethodDelete,
		Path:        "/me/sessions/{id}",
		Summary:     "Revoke one browser session",
		Tags:        []string{"auth", "settings"},
		Security:    requiredSession,
	}, revoke)
	registerLegacy(a, huma.Operation{
		OperationID: "user-session-revoke-legacy",
		Method:      http.MethodPost,
		Path:        "/user/sessions/{id}/revoke",
		Summary:     "Revoke one browser session",
		Tags:        []string{"auth", "settings"},
		Security:    requiredSession,
	}, revoke)

	registerLegacy(a, huma.Operation{
		OperationID: "other-sessions-revoke",
		Method:      http.MethodDelete,
		Path:        "/me/sessions",
		Summary:     "Revoke every other browser session",
		Tags:        []string{"auth", "settings"},
		Security:    requiredSession,
	}, service.revokeOtherSessionsHandler(true))
	registerLegacy(a, huma.Operation{
		OperationID: "other-sessions-revoke-legacy",
		Method:      http.MethodPost,
		Path:        "/user/sessions/revoke-others",
		Summary:     "Revoke every other browser session",
		Tags:        []string{"auth", "settings"},
		Security:    requiredSession,
	}, service.revokeOtherSessionsHandler(false))
}

func (s *authService) loginStartHandler() legacyHandler {
	return func(ctx context.Context, req *legacyRequest) (legacyPayload, error) {
		setAccountNoStore(req.Huma)
		authorizationURL, cookie, err := s.beginLogin(ctx, req)
		if err != nil {
			return legacyPayload{}, err
		}
		headers := make(http.Header)
		headers.Set("Location", authorizationURL)
		headers.Set("Cache-Control", "private, no-store")
		headers.Add("Set-Cookie", cookie.String())
		return legacyPayload{
			Status: http.StatusFound, Headers: headers, RawBody: []byte{},
		}, nil
	}
}

func (s *authService) loginURLHandler() legacyHandler {
	return func(ctx context.Context, req *legacyRequest) (legacyPayload, error) {
		setAccountNoStore(req.Huma)
		authorizationURL, cookie, err := s.beginLogin(ctx, req)
		if err != nil {
			return legacyPayload{}, err
		}
		headers := make(http.Header)
		headers.Set("Cache-Control", "private, no-store")
		headers.Add("Set-Cookie", cookie.String())
		return legacyPayload{
			Headers: headers,
			Body:    map[string]any{"url": authorizationURL},
		}, nil
	}
}

func (s *authService) callbackHandler() legacyHandler {
	return func(ctx context.Context, req *legacyRequest) (legacyPayload, error) {
		return s.completeCallback(ctx, req)
	}
}

func (s *authService) meHandler(legacy bool) legacyHandler {
	return func(ctx context.Context, req *legacyRequest) (legacyPayload, error) {
		setAccountNoStore(req.Huma)
		principal, err := s.resolvePrincipal(ctx, req)
		if err != nil {
			return legacyPayload{}, err
		}
		if principal == nil {
			return accountNoStorePayload(map[string]any{"user": nil}), nil
		}
		user := principalJSON(*principal)
		if legacy {
			details, err := s.store.LoadMeDetails(ctx, principal.CharacterID)
			if err != nil {
				return legacyPayload{}, err
			}
			user["characterOwnerHash"] = details.CharacterOwnerHash
			user["settings"] = details.Settings
		}
		return accountNoStorePayload(map[string]any{"user": user}), nil
	}
}

func (s *authService) settingsHandler() legacyHandler {
	return func(ctx context.Context, req *legacyRequest) (legacyPayload, error) {
		setAccountNoStore(req.Huma)
		principal, err := s.requirePrincipal(ctx, req)
		if err != nil {
			return legacyPayload{}, err
		}
		details, err := s.store.LoadMeDetails(ctx, principal.CharacterID)
		if err != nil {
			return legacyPayload{}, err
		}
		token, found, err := s.store.LoadTokenInfo(ctx, principal.CharacterID)
		if err != nil {
			return legacyPayload{}, err
		}

		var tokenBody any
		if found {
			effective := effectiveScopes(token.Scopes, token.RevokedScopes)
			tokenBody = map[string]any{
				"scopes":          token.Scopes,
				"effectiveScopes": effective,
				"revokedScopes":   token.RevokedScopes,
				"scopeCount":      len(effective),
				"tokenExpiry":     token.TokenExpiry,
				"lastFetched":     token.LastFetched,
				"disabled":        token.Disabled,
			}
		}
		return accountNoStorePayload(map[string]any{
			"account": map[string]any{
				"characterId":        principal.CharacterID,
				"characterName":      principal.CharacterName,
				"characterOwnerHash": details.CharacterOwnerHash,
				"isAdmin":            principal.IsAdmin,
				"lastLogin":          details.LastLogin,
				"createdAt":          details.CreatedAt,
			},
			"preferences": details.Settings,
			"esiToken":    tokenBody,
		}), nil
	}
}

func (s *authService) tokenInfoHandler() legacyHandler {
	return func(ctx context.Context, req *legacyRequest) (legacyPayload, error) {
		setAccountNoStore(req.Huma)
		principal, err := s.requirePrincipal(ctx, req)
		if err != nil {
			return legacyPayload{}, err
		}
		token, found, err := s.store.LoadTokenInfo(ctx, principal.CharacterID)
		if err != nil {
			return legacyPayload{}, err
		}
		if !found {
			return accountNoStorePayload(map[string]any{
				"scopes": []string{}, "token_expiry": nil,
			}), nil
		}
		return accountNoStorePayload(map[string]any{
			"scopes": token.Scopes, "token_expiry": token.TokenExpiry,
		}), nil
	}
}

func (s *authService) logoutHandler() legacyHandler {
	return func(ctx context.Context, req *legacyRequest) (legacyPayload, error) {
		setAccountNoStore(req.Huma)
		if err := requireSameOriginMutation(req.Huma); err != nil {
			return legacyPayload{}, err
		}
		raw, found := requestCookie(req.Huma, authSessionCookie)
		clearAuthCookies(req.Huma, s.production)
		if !found || raw == "" {
			return accountNoStorePayload(map[string]any{"success": true}), nil
		}
		if _, err := uuid.Parse(raw); err != nil {
			return accountNoStorePayload(map[string]any{"success": true}), nil
		}
		if s.storeErr != nil {
			return legacyPayload{}, s.storeErr
		}
		if err := s.store.DeleteSession(ctx, raw); err != nil {
			return legacyPayload{}, err
		}
		return accountNoStorePayload(map[string]any{"success": true}), nil
	}
}

func (s *authService) sessionsHandler() legacyHandler {
	return func(ctx context.Context, req *legacyRequest) (legacyPayload, error) {
		setAccountNoStore(req.Huma)
		principal, err := s.requirePrincipal(ctx, req)
		if err != nil {
			return legacyPayload{}, err
		}
		current, ok := requestCookie(req.Huma, authSessionCookie)
		if !ok {
			return legacyPayload{}, apiError(
				http.StatusUnauthorized, "Authentication required",
			)
		}
		rows, err := s.store.ListSessions(
			ctx, principal.CharacterID, current, s.now().UTC(),
		)
		if err != nil {
			return legacyPayload{}, err
		}
		sessions := make([]map[string]any, 0, len(rows))
		for _, row := range rows {
			browser, operatingSystem, device := describeSessionClient(
				row.UserAgent, row.ClientHint,
			)
			sessions = append(sessions, map[string]any{
				"id":              row.ID,
				"current":         row.Current,
				"ipAddress":       pointerValue(row.IPAddress),
				"countryCode":     pointerValue(row.CountryCode),
				"browser":         browser,
				"operatingSystem": operatingSystem,
				"device":          device,
				"createdAt":       row.CreatedAt,
				"lastSeenAt":      row.LastSeenAt,
				"expiresAt":       row.ExpiresAt,
			})
		}
		return accountNoStorePayload(
			map[string]any{"sessions": sessions},
		), nil
	}
}

func (s *authService) revokeSessionHandler() legacyHandler {
	return func(ctx context.Context, req *legacyRequest) (legacyPayload, error) {
		setAccountNoStore(req.Huma)
		if err := requireSameOriginMutation(req.Huma); err != nil {
			return legacyPayload{}, err
		}
		principal, err := s.requirePrincipal(ctx, req)
		if err != nil {
			return legacyPayload{}, err
		}
		id, err := strconv.ParseInt(strings.TrimSpace(req.Param("id")), 10, 64)
		if err != nil || id <= 0 {
			return legacyPayload{}, apiError(
				http.StatusBadRequest, "Invalid session id",
			)
		}
		currentSessionID, ok := requestCookie(req.Huma, authSessionCookie)
		if !ok {
			return legacyPayload{}, apiError(
				http.StatusUnauthorized, "Authentication required",
			)
		}
		found, current, err := s.store.RevokeSession(
			ctx, principal.CharacterID, id, currentSessionID,
		)
		if err != nil {
			return legacyPayload{}, err
		}
		if !found {
			return legacyPayload{}, apiError(
				http.StatusNotFound, "Session not found",
			)
		}
		if current {
			clearAuthCookies(req.Huma, s.production)
		}
		return accountNoStorePayload(map[string]any{
			"revoked": true, "current": current,
		}), nil
	}
}

func (s *authService) revokeOtherSessionsHandler(
	canonical bool,
) legacyHandler {
	return func(ctx context.Context, req *legacyRequest) (legacyPayload, error) {
		setAccountNoStore(req.Huma)
		if err := requireSameOriginMutation(req.Huma); err != nil {
			return legacyPayload{}, err
		}
		if canonical && req.Query.Get("except") != "current" {
			return legacyPayload{}, apiError(
				http.StatusBadRequest,
				"except=current is required",
			)
		}
		principal, err := s.requirePrincipal(ctx, req)
		if err != nil {
			return legacyPayload{}, err
		}
		currentSessionID, ok := requestCookie(req.Huma, authSessionCookie)
		if !ok {
			return legacyPayload{}, apiError(
				http.StatusUnauthorized, "Authentication required",
			)
		}
		revoked, err := s.store.RevokeOtherSessions(
			ctx, principal.CharacterID, currentSessionID,
		)
		if err != nil {
			return legacyPayload{}, err
		}
		return accountNoStorePayload(
			map[string]any{"revoked": revoked},
		), nil
	}
}

func accountNoStorePayload(body any) legacyPayload {
	headers := make(http.Header)
	headers.Set("Cache-Control", "private, no-store")
	headers.Set("Pragma", "no-cache")
	return legacyPayload{Body: body, Headers: headers}
}

func effectiveScopes(granted, revoked []string) []string {
	blocked := make(map[string]bool, len(revoked))
	for _, scope := range revoked {
		blocked[scope] = true
	}
	result := make([]string, 0, len(granted))
	for _, scope := range granted {
		if !blocked[scope] {
			result = append(result, scope)
		}
	}
	return result
}

func describeSessionClient(
	userAgent *string,
	clientHint *string,
) (browser string, operatingSystem string, device string) {
	ua := ""
	if userAgent != nil {
		ua = *userAgent
	}
	hints := ""
	if clientHint != nil {
		hints = *clientHint
	}

	browser = "Unknown browser"
	switch {
	case containsFold(hints, "Brave"):
		browser = "Brave"
	case containsFold(ua, "Edg/"):
		browser = "Microsoft Edge"
	case containsFold(ua, "Firefox/") || containsFold(ua, "FxiOS/"):
		browser = "Firefox"
	case containsFold(ua, "CriOS/"):
		browser = "Chrome"
	case containsFold(ua, "Chrome/") || containsFold(ua, "Chromium/"):
		browser = "Chrome"
	case containsFold(ua, "Safari/"):
		browser = "Safari"
	}

	operatingSystem = "Unknown OS"
	switch {
	case containsFold(ua, "Windows NT"):
		operatingSystem = "Windows"
	case containsFold(ua, "Android"):
		operatingSystem = "Android"
	case containsAnyFold(ua, "iPhone", "iPad", "iPod"):
		operatingSystem = "iOS"
	case containsFold(ua, "Macintosh") || containsFold(ua, "Mac OS X"):
		operatingSystem = "macOS"
	case containsFold(ua, "Linux"):
		operatingSystem = "Linux"
	}

	device = "Desktop"
	switch {
	case containsFold(ua, "iPad") || containsFold(ua, "Tablet"):
		device = "Tablet"
	case containsAnyFold(ua, "Mobile", "iPhone", "iPod", "Android"):
		device = "Mobile"
	}
	return browser, operatingSystem, device
}

func containsFold(value, fragment string) bool {
	return strings.Contains(
		strings.ToLower(value),
		strings.ToLower(fragment),
	)
}

func containsAnyFold(value string, fragments ...string) bool {
	for _, fragment := range fragments {
		if containsFold(value, fragment) {
			return true
		}
	}
	return false
}

func requireSameOriginMutation(ctx huma.Context) error {
	if strings.EqualFold(ctx.Header("Sec-Fetch-Site"), "cross-site") {
		return apiError(http.StatusForbidden, "Cross-site request rejected")
	}
	rawOrigin := strings.TrimSpace(ctx.Header("Origin"))
	if rawOrigin == "" {
		return nil
	}
	origin, err := url.Parse(rawOrigin)
	if err != nil || origin.Host == "" {
		return apiError(http.StatusForbidden, "Cross-site request rejected")
	}
	// Caddy preserves the original Host header for the in-process handler.
	// Do not let a client-supplied forwarding header redefine same-origin.
	host := ctx.Host()
	if !strings.EqualFold(origin.Host, host) {
		return apiError(http.StatusForbidden, "Cross-site request rejected")
	}
	return nil
}
