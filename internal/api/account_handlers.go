package api

import (
	"context"
	"encoding/json"
	"math"
	"net/http"
	"slices"
	"sort"
	"strconv"
	"strings"
)

type preferenceWriteMode int

const (
	preferenceWriteCombined preferenceWriteMode = iota
	preferenceWriteDefaultTabs
	preferenceWriteTheme
	preferenceWriteBoards
)

func (s *accountService) preferencesHandler() legacyHandler {
	return func(ctx context.Context, req *legacyRequest) (legacyPayload, error) {
		setAccountNoStore(req.Huma)
		principal, err := s.principal(ctx, req)
		if err != nil {
			return legacyPayload{}, err
		}
		preferences, err := s.store.LoadPreferences(ctx, principal.CharacterID)
		if err != nil {
			return legacyPayload{}, err
		}
		return accountNoStorePayload(preferences), nil
	}
}

// Wire types for the account write routes.
type accountPreferencesBody struct {
	Theme       json.RawMessage `json:"theme,omitempty" doc:"Preferred site theme."`
	DefaultTabs json.RawMessage `json:"defaultTabs,omitempty" doc:"Default tab per entity page."`
	Boards      json.RawMessage `json:"boards,omitempty" doc:"Killboards pinned in the navigation."`

	// whole is the untouched object. The boards-only route treats the entire
	// body as the board state rather than reading a boards key, so that shape
	// has to survive decoding into the combined type.
	whole json.RawMessage `json:"-"`
}

func (b *accountPreferencesBody) UnmarshalJSON(data []byte) error {
	type alias accountPreferencesBody
	var decoded alias
	if err := json.Unmarshal(data, &decoded); err != nil {
		return err
	}
	*b = accountPreferencesBody(decoded)
	b.whole = append(json.RawMessage(nil), data...)
	return nil
}

type accountDescriptionBody struct {
	Entity      string `json:"entity" enum:"character,corporation,alliance" doc:"Which description to write."`
	Description string `json:"description" maxLength:"4000" doc:"Description text."`
	Format      string `json:"format" enum:"markdown,eve_html" doc:"How to interpret the text."`
}

type accountNotificationsReadBody struct {
	ID json.RawMessage `json:"id,omitempty" doc:"Read cursor. Marks every notification up to this identifier."`
}

func (s *accountService) savePreferencesHandler(
	mode preferenceWriteMode,
) legacyHandler {
	return func(ctx context.Context, req *legacyRequest) (legacyPayload, error) {
		setAccountNoStore(req.Huma)
		if err := requireSameOriginMutation(req.Huma); err != nil {
			return legacyPayload{}, err
		}
		principal, err := s.principal(ctx, req)
		if err != nil {
			return legacyPayload{}, err
		}
		body, err := decodeJSONBody[accountPreferencesBody](req, accountBodyLimit)
		if err != nil {
			return legacyPayload{}, err
		}

		updates := make(map[string]any)
		if mode == preferenceWriteCombined || mode == preferenceWriteDefaultTabs {
			if raw, present := rawJSONField(body.DefaultTabs); present {
				tabs, valid := sanitizeDefaultTabs(raw)
				if !valid {
					return legacyPayload{}, apiError(
						http.StatusBadRequest, "Missing defaultTabs",
					)
				}
				updates["defaultTabs"] = tabs
			} else if mode == preferenceWriteDefaultTabs {
				return legacyPayload{}, apiError(
					http.StatusBadRequest, "Missing defaultTabs",
				)
			}
		}
		if mode == preferenceWriteCombined || mode == preferenceWriteTheme {
			if raw, present := rawJSONField(body.Theme); present {
				theme, valid := sanitizeTheme(raw)
				if !valid {
					return legacyPayload{}, apiError(
						http.StatusBadRequest, "Missing theme object",
					)
				}
				updates["theme"] = theme
			} else if mode == preferenceWriteTheme {
				return legacyPayload{}, apiError(
					http.StatusBadRequest, "Missing theme object",
				)
			}
		}
		if mode == preferenceWriteCombined || mode == preferenceWriteBoards {
			if raw, present := rawJSONField(body.Boards); present && mode == preferenceWriteCombined {
				updates["boards"] = sanitizeBoardState(raw)
			} else if mode == preferenceWriteBoards {
				updates["boards"] = sanitizeBoardState(rawJSONValue(body.whole))
			}
		}
		if len(updates) == 0 {
			return legacyPayload{}, apiError(
				http.StatusBadRequest,
				"At least one of defaultTabs, theme, or boards is required",
			)
		}
		if err := s.store.SavePreferences(
			ctx, principal.CharacterID, updates, s.now().UTC(),
		); err != nil {
			return legacyPayload{}, err
		}

		switch mode {
		case preferenceWriteDefaultTabs:
			return accountNoStorePayload(map[string]any{
				"defaultTabs": updates["defaultTabs"],
			}), nil
		case preferenceWriteTheme:
			return accountNoStorePayload(map[string]any{
				"theme": updates["theme"],
			}), nil
		case preferenceWriteBoards:
			return accountNoStorePayload(updates["boards"]), nil
		default:
			return accountNoStorePayload(map[string]any{
				"preferences": updates,
			}), nil
		}
	}
}

func (s *accountService) boardsHandler() legacyHandler {
	return func(ctx context.Context, req *legacyRequest) (legacyPayload, error) {
		setAccountNoStore(req.Huma)
		if err := s.requireStore(); err != nil {
			return legacyPayload{}, err
		}
		principal, err := s.auth.resolvePrincipal(ctx, req)
		if err != nil {
			return legacyPayload{}, err
		}
		data, err := s.store.LoadBoardData(ctx, principal)
		if err != nil {
			return legacyPayload{}, err
		}
		rawCookie, _ := requestCookie(req.Huma, "ek_boards")
		state := mergeBoardStates(parseBoardCookie(rawCookie), data.Account)
		pinned := make(map[string]bool, len(state.Pinned))
		dismissed := make(map[string]bool, len(state.Dismissed))
		for _, key := range state.Pinned {
			pinned[key] = true
		}
		for _, key := range state.Dismissed {
			dismissed[key] = true
		}

		byKey := make(map[string]accountBoardDomain, len(data.Domains))
		tracked := make(map[string]bool)
		var orderedKeys []string
		for _, domain := range data.Domains {
			key := boardDomainKey(domain)
			byKey[key] = domain
			if principal != nil && domainTracksPrincipal(domain, *principal) {
				tracked[key] = true
				orderedKeys = appendUnique(orderedKeys, key)
			}
		}
		for _, key := range state.Pinned {
			orderedKeys = appendUnique(orderedKeys, key)
		}

		boards := make([]map[string]any, 0, len(orderedKeys))
		for _, key := range orderedKeys {
			if dismissed[key] {
				continue
			}
			domain, found := byKey[key]
			if !found {
				continue
			}
			name := domain.Subdomain
			if domain.SiteName != nil && *domain.SiteName != "" {
				name = *domain.SiteName
			}
			host := boardKeyHost(key)
			boards = append(boards, map[string]any{
				"key": key, "host": host, "url": "https://" + host,
				"name": name, "tracked": tracked[key], "pinned": pinned[key],
			})
		}
		sort.SliceStable(boards, func(i, j int) bool {
			left := strings.ToLower(boards[i]["name"].(string))
			right := strings.ToLower(boards[j]["name"].(string))
			if left == right {
				return boards[i]["name"].(string) < boards[j]["name"].(string)
			}
			return left < right
		})

		var current any
		if key := hostBoardKey(req.Huma.Host()); key != nil {
			if domain, found := byKey[*key]; found {
				name := domain.Subdomain
				if domain.SiteName != nil && *domain.SiteName != "" {
					name = *domain.SiteName
				}
				listed := false
				for _, board := range boards {
					if board["key"] == *key {
						listed = true
						break
					}
				}
				current = map[string]any{
					"key": *key, "name": name, "listed": listed,
				}
			}
		}
		return accountNoStorePayload(map[string]any{
			"boards": boards, "current": current,
			"authenticated": principal != nil,
			"atCapacity":    len(state.Pinned) >= maxBoardEntries,
		}), nil
	}
}

func domainTracksPrincipal(domain accountBoardDomain, principal Principal) bool {
	for _, entity := range domain.Entities {
		switch entity.Type {
		case "character":
			if entity.ID == int64(principal.CharacterID) {
				return true
			}
		case "corporation":
			if principal.CorporationID != nil &&
				entity.ID == int64(*principal.CorporationID) {
				return true
			}
		case "alliance":
			if principal.AllianceID != nil &&
				entity.ID == int64(*principal.AllianceID) {
				return true
			}
		}
	}
	return false
}

func appendUnique(values []string, value string) []string {
	if slices.Contains(values, value) {
		return values
	}
	return append(values, value)
}

func (s *accountService) overviewHandler() legacyHandler {
	return func(ctx context.Context, req *legacyRequest) (legacyPayload, error) {
		setAccountNoStore(req.Huma)
		principal, err := s.principal(ctx, req)
		if err != nil {
			return legacyPayload{}, err
		}
		overview, err := s.store.LoadOverview(
			ctx, principal.CharacterID, s.now().UTC(),
		)
		if err != nil {
			return legacyPayload{}, err
		}
		var token any
		if overview.TokenFound {
			token = map[string]any{
				"scopeCount":  overview.TokenScopeCount,
				"tokenExpiry": overview.TokenExpiry,
				"lastFetched": overview.TokenLastFetch,
			}
		}
		return accountNoStorePayload(map[string]any{
			"account": map[string]any{
				"characterId":   principal.CharacterID,
				"characterName": principal.CharacterName,
				"isAdmin":       principal.IsAdmin,
				"lastLogin":     overview.LastLogin,
				"createdAt":     overview.CreatedAt,
			},
			"esiStats": map[string]any{
				"total_requests":  overview.TotalRequests,
				"total_errors":    overview.TotalErrors,
				"total_new_items": overview.TotalNewItems,
				"last_request":    overview.LastRequest,
				"requests_24h":    overview.Requests24Hours,
				"errors_24h":      overview.Errors24Hours,
				"new_items_24h":   overview.NewItems24Hours,
			},
			"esiToken": token,
		}), nil
	}
}

func (s *accountService) manageableEntitiesHandler() legacyHandler {
	return func(ctx context.Context, req *legacyRequest) (legacyPayload, error) {
		setAccountNoStore(req.Huma)
		principal, err := s.principal(ctx, req)
		if err != nil {
			return legacyPayload{}, err
		}
		entities, err := s.store.LoadManageableEntities(
			ctx, principal.CharacterID,
		)
		if err != nil {
			return legacyPayload{}, err
		}
		body := map[string]any{
			"character": map[string]any{
				"id":                        entities.Character.ID,
				"name":                      entities.Character.Name,
				"esi_description":           entities.Character.ESIDescription,
				"custom_description":        entities.Character.CustomDescription,
				"custom_description_format": formatOrMarkdown(entities.Character.CustomFormat),
				"canEdit":                   true,
				"pending_submission":        entities.Character.Pending,
			},
			"corporation": nil,
			"alliance":    nil,
		}
		if corporation := entities.Corporation; corporation != nil {
			canEdit := corporation.CEOID != nil &&
				*corporation.CEOID == principal.CharacterID
			var pending any
			if canEdit {
				pending = corporation.Pending
			}
			body["corporation"] = map[string]any{
				"id": corporation.ID, "name": corporation.Name,
				"ticker": corporation.Ticker, "ceo_id": corporation.CEOID,
				"ceo_name":                  corporation.CEOName,
				"esi_description":           corporation.ESIDescription,
				"custom_description":        corporation.CustomDescription,
				"custom_description_format": formatOrMarkdown(corporation.CustomFormat),
				"canEdit":                   canEdit, "pending_submission": pending,
			}
		}
		if alliance := entities.Alliance; alliance != nil {
			canEdit := alliance.ExecutorCEOID != nil &&
				*alliance.ExecutorCEOID == principal.CharacterID
			var pending any
			if canEdit {
				pending = alliance.Pending
			}
			body["alliance"] = map[string]any{
				"id": alliance.ID, "name": alliance.Name,
				"ticker":                    alliance.Ticker,
				"executor_corporation_id":   alliance.ExecutorCorporationID,
				"executor_ceo_id":           alliance.ExecutorCEOID,
				"executor_ceo_name":         alliance.ExecutorCEOName,
				"custom_description":        alliance.CustomDescription,
				"custom_description_format": formatOrMarkdown(alliance.CustomFormat),
				"canEdit":                   canEdit, "pending_submission": pending,
			}
		}
		return accountNoStorePayload(body), nil
	}
}

func formatOrMarkdown(value *string) string {
	if value == nil || *value == "" {
		return "markdown"
	}
	return *value
}

func (s *accountService) saveDescriptionHandler() legacyHandler {
	return func(ctx context.Context, req *legacyRequest) (legacyPayload, error) {
		setAccountNoStore(req.Huma)
		if err := requireSameOriginMutation(req.Huma); err != nil {
			return legacyPayload{}, err
		}
		principal, err := s.principal(ctx, req)
		if err != nil {
			return legacyPayload{}, err
		}
		body, err := decodeJSONBody[accountDescriptionBody](req, accountBodyLimit)
		if err != nil {
			return legacyPayload{}, err
		}
		entity, entityOK := body.Entity, true
		description, descriptionOK := body.Description, true
		format, formatOK := body.Format, body.Format != ""
		if !entityOK || (entity != "character" &&
			entity != "corporation" && entity != "alliance") {
			return legacyPayload{}, apiError(
				http.StatusBadRequest,
				"entity must be one of: character, corporation, alliance",
			)
		}
		if !formatOK || (format != "markdown" && format != "eve_html") {
			return legacyPayload{}, apiError(
				http.StatusBadRequest,
				"format must be one of: markdown, eve_html",
			)
		}
		if !descriptionOK {
			return legacyPayload{}, apiError(
				http.StatusBadRequest, "description must be a string",
			)
		}
		if len([]rune(description)) > maxDescriptionLength {
			return legacyPayload{}, apiError(
				http.StatusBadRequest,
				"description exceeds 4000 characters",
			)
		}
		target, err := s.store.ResolveBioTarget(
			ctx, principal.CharacterID, entity,
		)
		if err != nil {
			return legacyPayload{}, err
		}
		description = strings.TrimSpace(description)
		if description == "" {
			if err := s.store.ClearBio(ctx, target); err != nil {
				return legacyPayload{}, err
			}
			return accountNoStorePayload(map[string]any{
				"ok": true, "status": "cleared",
				"entity": entity, "entity_id": target.ID,
			}), nil
		}

		submission := accountBioSubmission{
			Target: target, Body: description, BodyFormat: format,
			RenderedHTML:    renderAccountBio(description, format),
			CharacterID:     principal.CharacterID,
			CharacterName:   principal.CharacterName,
			CorporationID:   principal.CorporationID,
			CorporationName: principal.CorporationName,
			AllianceID:      principal.AllianceID,
			AllianceName:    principal.AllianceName,
			SubmittedAt:     s.now().UTC(),
		}
		queueID, err := s.store.EnqueueBio(ctx, submission)
		if err != nil {
			return legacyPayload{}, err
		}
		if s.dispatch != nil {
			s.dispatch.BioPending(ctx, submission, queueID)
		}
		return accountNoStorePayload(map[string]any{
			"ok": true, "status": "pending_review",
			"entity": entity, "entity_id": target.ID,
			"queue_id": queueID,
		}), nil
	}
}

func (s *accountService) esiMetricsHandler() legacyHandler {
	return func(ctx context.Context, req *legacyRequest) (legacyPayload, error) {
		setAccountNoStore(req.Huma)
		principal, err := s.principal(ctx, req)
		if err != nil {
			return legacyPayload{}, err
		}
		metrics, err := s.store.LoadESIMetrics(
			ctx, principal.CharacterID, s.now().UTC(),
		)
		if err != nil {
			return legacyPayload{}, err
		}
		if metrics.Volume == nil {
			metrics.Volume = []accountESIVolume{}
		}
		return accountNoStorePayload(map[string]any{
			"volumeByHour": metrics.Volume,
			"rateLimit": map[string]any{
				"request_count": metrics.RequestCount,
			},
			"responseTime": map[string]any{
				"avg_ms": metrics.AverageMS, "p95_ms": metrics.P95MS,
			},
		}), nil
	}
}

func (s *accountService) esiLogsHandler() legacyHandler {
	return func(ctx context.Context, req *legacyRequest) (legacyPayload, error) {
		setAccountNoStore(req.Huma)
		principal, err := s.principal(ctx, req)
		if err != nil {
			return legacyPayload{}, err
		}
		query := accountESILogQuery{
			CharacterID: principal.CharacterID,
			Limit:       boundedAccountQueryInt(req.Query.Get("limit"), 50, 1, 100),
			Page:        boundedAccountQueryInt(req.Query.Get("page"), 1, 1, math.MaxInt32),
			Source:      truncateRunes(strings.TrimSpace(req.Query.Get("source")), 200),
			Status:      req.Query.Get("status"),
			Endpoint:    req.Query.Get("endpoint_type"),
		}
		if after, ok := positiveQueryInt64(req.Query.Get("after_id")); ok {
			query.AfterID = &after
		}
		result, err := s.store.LoadESILogs(ctx, query)
		if err != nil {
			return legacyPayload{}, err
		}
		if result.Rows == nil {
			result.Rows = []accountESILogRow{}
		}
		if query.AfterID != nil {
			return accountNoStorePayload(map[string]any{
				"rows": result.Rows, "newRows": true,
			}), nil
		}
		if result.Sources == nil {
			result.Sources = []string{}
		}
		pages := int64(0)
		if result.Total > 0 {
			pages = (result.Total + int64(query.Limit) - 1) /
				int64(query.Limit)
		}
		return accountNoStorePayload(map[string]any{
			"rows": result.Rows, "total": result.Total,
			"page": query.Page, "limit": query.Limit,
			"pages": pages, "sources": result.Sources,
		}), nil
	}
}

func boundedAccountQueryInt(raw string, fallback, minimum, maximum int) int {
	if raw == "" {
		return fallback
	}
	value, err := strconv.Atoi(raw)
	if err != nil || value == 0 {
		return fallback
	}
	if value < minimum {
		return minimum
	}
	if value > maximum {
		return maximum
	}
	return value
}

func positiveQueryInt64(raw string) (int64, bool) {
	value, err := strconv.ParseInt(strings.TrimSpace(raw), 10, 64)
	return value, err == nil && value > 0
}

func (s *accountService) activeAnnouncementsHandler() legacyHandler {
	return func(ctx context.Context, _ *legacyRequest) (legacyPayload, error) {
		if err := s.requireStore(); err != nil {
			return legacyPayload{}, err
		}
		now := s.now().UTC()
		stored, err := s.store.LoadActiveAnnouncements(ctx, now)
		if err != nil {
			return legacyPayload{}, err
		}
		announcements := make([]any, 0, len(stored)+4)
		for _, item := range stored {
			announcements = append(announcements, item)
		}
		if s.loadEphemeral != nil {
			for _, item := range s.loadEphemeral(ctx, now) {
				announcements = append(announcements, item)
			}
		}
		return jsonPayload(map[string]any{
			"announcements": announcements,
		}), nil
	}
}

func (s *accountService) dismissedAnnouncementsHandler() legacyHandler {
	return func(ctx context.Context, req *legacyRequest) (legacyPayload, error) {
		setAccountNoStore(req.Huma)
		principal, err := s.principal(ctx, req)
		if err != nil {
			return legacyPayload{}, err
		}
		ids, err := s.store.LoadDismissedAnnouncementIDs(
			ctx, principal.CharacterID,
		)
		if err != nil {
			return legacyPayload{}, err
		}
		if ids == nil {
			ids = []int64{}
		}
		return accountNoStorePayload(map[string]any{
			"dismissedIds": ids,
		}), nil
	}
}

func (s *accountService) dismissAnnouncementHandler() legacyHandler {
	return func(ctx context.Context, req *legacyRequest) (legacyPayload, error) {
		setAccountNoStore(req.Huma)
		if err := requireSameOriginMutation(req.Huma); err != nil {
			return legacyPayload{}, err
		}
		principal, err := s.principal(ctx, req)
		if err != nil {
			return legacyPayload{}, err
		}
		id, err := strconv.ParseInt(strings.TrimSpace(req.Param("id")), 10, 64)
		if err != nil || id < 1 {
			return legacyPayload{}, apiError(
				http.StatusBadRequest, "Invalid announcement id",
			)
		}
		if err := s.store.DismissAnnouncement(
			ctx, principal.CharacterID, id,
		); err != nil {
			return legacyPayload{}, err
		}
		return accountNoStorePayload(map[string]any{"ok": true}), nil
	}
}

func (s *accountService) notificationRepliesHandler() legacyHandler {
	return func(ctx context.Context, req *legacyRequest) (legacyPayload, error) {
		setAccountNoStore(req.Huma)
		principal, err := s.principal(ctx, req)
		if err != nil {
			return legacyPayload{}, err
		}
		domainID, err := s.store.ResolveCommentDomainID(
			ctx, req.Huma.Host(),
		)
		if err != nil {
			return legacyPayload{}, err
		}
		since := int64(0)
		if parsed, ok := nonNegativeQueryInt64(req.Query.Get("since")); ok {
			since = parsed
		}
		limit := boundedAccountQueryInt(req.Query.Get("limit"), 50, 1, 100)
		replies, err := s.store.LoadNotificationReplies(
			ctx,
			accountNotificationQuery{
				CharacterID: principal.CharacterID,
				DomainID:    domainID, Since: since, Limit: limit,
			},
		)
		if err != nil {
			return legacyPayload{}, err
		}
		if replies == nil {
			replies = []accountNotificationReply{}
		}
		highest := since
		if len(replies) > 0 {
			highest = replies[0].ID
		}
		return accountNoStorePayload(map[string]any{
			"replies": replies, "highestId": highest,
		}), nil
	}
}

func nonNegativeQueryInt64(raw string) (int64, bool) {
	if strings.TrimSpace(raw) == "" {
		return 0, false
	}
	value, err := strconv.ParseInt(strings.TrimSpace(raw), 10, 64)
	return value, err == nil && value >= 0
}

func (s *accountService) markNotificationsReadHandler() legacyHandler {
	return func(ctx context.Context, req *legacyRequest) (legacyPayload, error) {
		setAccountNoStore(req.Huma)
		if err := requireSameOriginMutation(req.Huma); err != nil {
			return legacyPayload{}, err
		}
		principal, err := s.principal(ctx, req)
		if err != nil {
			return legacyPayload{}, err
		}
		body, err := decodeJSONBody[accountNotificationsReadBody](req, accountBodyLimit)
		if err != nil {
			return legacyPayload{}, err
		}
		id, ok := nonNegativeJSONInt64(rawJSONValue(body.ID))
		if !ok {
			return legacyPayload{}, apiError(
				http.StatusBadRequest,
				"id must be a non-negative integer",
			)
		}
		updated, err := s.store.MarkNotificationsRead(
			ctx, principal.CharacterID, id,
		)
		if err != nil {
			return legacyPayload{}, err
		}
		return accountNoStorePayload(map[string]any{
			"lastSeenNotificationId": updated,
		}), nil
	}
}

func nonNegativeJSONInt64(raw any) (int64, bool) {
	switch value := raw.(type) {
	case json.Number:
		parsed, err := strconv.ParseInt(value.String(), 10, 64)
		return parsed, err == nil && parsed >= 0
	case string:
		parsed, err := strconv.ParseInt(strings.TrimSpace(value), 10, 64)
		return parsed, err == nil && parsed >= 0
	case float64:
		if value < 0 || math.Trunc(value) != value || value > math.MaxInt64 {
			return 0, false
		}
		return int64(value), true
	default:
		return 0, false
	}
}
