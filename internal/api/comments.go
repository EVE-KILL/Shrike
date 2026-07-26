package api

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"net"
	"net/http"
	"strconv"
	"strings"
	"time"
	"unicode/utf16"

	"github.com/danielgtaylor/huma/v2"
	"github.com/eve-kill/shrike/internal/queue"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/redis/go-redis/v9"
)

const (
	commentTargetKillmail    = 1
	commentTargetCharacter   = 2
	commentTargetCorporation = 3
	commentTargetAlliance    = 4
	commentTargetSystem      = 5
	commentTargetPage        = 6
	commentTargetBattle      = 7
	commentTargetFit         = 8
	commentTargetBlog        = 9
	commentTargetCampaign    = 10

	commentVisibilityPublished = 0
	commentVisibilityPending   = 1
	commentVisibilityRejected  = 2
	commentVisibilityHidden    = 3

	commentModerationOK          = 0
	commentModerationPending     = 1
	commentModerationRejectedAI  = 2
	commentModerationAdminHidden = 3

	moderationKindComment       = 0
	moderationStatusPending     = 0
	moderationStatusApproved    = 1
	moderationStatusRejected    = 2
	moderationStatusAutoApprove = 3
	moderationStatusAutoReject  = 4

	commentMinimumLength = 2
	commentMaximumLength = 2000
	commentReplyHardCap  = 500
	commentReportFlagAt  = 3
	commentEditWindow    = 24 * time.Hour
)

const commentColumns = `
	id,
	target_type,
	target_id,
	target_slug,
	domain_id,
	parent_id,
	root_id,
	depth,
	body_md,
	body_html,
	character_id,
	character_name,
	corporation_id,
	corporation_name,
	alliance_id,
	alliance_name,
	created_at,
	updated_at,
	edited_at,
	deleted_at,
	deleted_by,
	reports_count,
	flagged,
	moderation_status,
	visibility`

type commentDomainEntity struct {
	Type string `json:"type"`
	ID   int64  `json:"id"`
}

type commentDomain struct {
	ID       int32
	OwnerID  int32
	Entities []commentDomainEntity
}

type commentAncestor struct {
	ID            int64  `json:"id"`
	CharacterID   int64  `json:"character_id"`
	CharacterName string `json:"character_name"`
}

type commentModerationAction string

const (
	commentActionPass   commentModerationAction = "pass"
	commentActionFlag   commentModerationAction = "flag"
	commentActionDelete commentModerationAction = "delete"
)

type commentModerationResult struct {
	Action      commentModerationAction `json:"action"`
	Category    *string                 `json:"category"`
	UserMessage *string                 `json:"userMessage"`
	MaxScore    float64                 `json:"maxScore"`
	Scores      map[string]float64      `json:"scores"`
	Source      string                  `json:"source"`
}

type commentModerator interface {
	Moderate(
		context.Context,
		string,
		[]string,
	) commentModerationResult
}

type commentEventDispatcher interface {
	Comment(
		context.Context,
		string,
		map[string]any,
		[]commentAncestor,
	)
	ModerationPending(
		context.Context,
		int64,
		map[string]any,
		commentModerationResult,
	)
	Reported(
		context.Context,
		map[string]any,
		*Principal,
		string,
		*string,
	)
}

type commentService struct {
	auth       *authService
	db         MutationDatabase
	dbErr      error
	cache      *redis.Client
	now        func() time.Time
	renderer   *commentRenderer
	moderator  commentModerator
	dispatcher commentEventDispatcher
	klipy      *klipyClient
}

func newCommentService(opts Options) *commentService {
	service := &commentService{
		auth:  newAuthService(opts),
		cache: opts.Cache,
		now:   time.Now,
	}
	db, err := mutationDatabase(opts)
	if err != nil {
		service.dbErr = err
	} else {
		service.db = db
	}
	httpClient := opts.Auth.HTTPClient
	service.renderer = newCommentRenderer(opts.Cache, httpClient)
	service.moderator = newOpenAICommentModerator(opts.Cache, httpClient)
	service.klipy = newKlipyClient(opts.Cache, httpClient)
	if pool, ok := opts.DB.(*pgxpool.Pool); ok && pool != nil {
		if client, queueErr := queue.New(queue.Options{Pool: pool}); queueErr == nil {
			service.dispatcher = &riverCommentEventDispatcher{client: client}
		}
	}
	return service
}

func (s *commentService) requireDB() error {
	if s.dbErr != nil {
		return s.dbErr
	}
	if s.db == nil {
		return apiError(
			http.StatusServiceUnavailable,
			"Comment storage is not configured",
		)
	}
	return nil
}

// registerCommentRoutes installs the public discussion surface and the
// signed-in user's cross-domain comment history. The /user aliases remain
// until the Nuxt caller is moved to the canonical /me contract.
func registerCommentRoutes(a huma.API, opts Options) {
	registerCommentServiceRoutes(a, opts, newCommentService(opts))
}

func registerCommentServiceRoutes(
	a huma.API,
	opts Options,
	service *commentService,
) {
	if a.OpenAPI().Components.SecuritySchemes == nil {
		a.OpenAPI().Components.SecuritySchemes =
			make(map[string]*huma.SecurityScheme)
	}
	if a.OpenAPI().Components.SecuritySchemes["eveSession"] == nil {
		a.OpenAPI().Components.SecuritySchemes["eveSession"] =
			&huma.SecurityScheme{
				Type: "apiKey", In: "cookie", Name: authSessionCookie,
				Description: "EVE-KILL browser session.",
			}
	}
	required := []map[string][]string{{"eveSession": {}}}
	optional := []map[string][]string{{}, {"eveSession": {}}}

	registerLegacy(a, huma.Operation{
		OperationID: "comments-feed",
		Method:      http.MethodGet,
		Path:        "/comments",
		Summary:     "Recent top-level comments",
		Tags:        []string{"comments"},
		Security:    optional,
	}, commentCached(opts, 30*time.Second, service.listHandler()))
	registerLegacy(a, huma.Operation{
		OperationID: "comments-create",
		Method:      http.MethodPost,
		Path:        "/comments",
		Summary:     "Publish a comment or reply",
		Tags:        []string{"comments"},
		Security:    required,
	}, service.createHandler())
	registerLegacy(a, huma.Operation{
		OperationID: "comments-thread",
		Method:      http.MethodGet,
		Path:        "/comments/thread",
		Summary:     "Comments for one target",
		Tags:        []string{"comments"},
		Security:    optional,
	}, commentCached(opts, 30*time.Second, service.threadHandler()))
	registerLegacy(a, huma.Operation{
		OperationID: "comments-preview",
		Method:      http.MethodPost,
		Path:        "/comments/preview",
		Summary:     "Render a comment preview",
		Tags:        []string{"comments"},
		Security:    required,
	}, service.previewHandler())
	registerLegacy(a, huma.Operation{
		OperationID: "comments-klipy-search",
		Method:      http.MethodGet,
		Path:        "/comments/klipy/search",
		Summary:     "Search comment GIFs",
		Tags:        []string{"comments"},
		Security:    required,
	}, service.klipyHandler("search"))
	registerLegacy(a, huma.Operation{
		OperationID: "comments-klipy-trending",
		Method:      http.MethodGet,
		Path:        "/comments/klipy/trending",
		Summary:     "Trending comment GIFs",
		Tags:        []string{"comments"},
		Security:    required,
	}, service.klipyHandler("trending"))
	registerLegacy(a, huma.Operation{
		OperationID: "comment-detail",
		Method:      http.MethodGet,
		Path:        "/comments/{id}",
		Summary:     "One published comment",
		Tags:        []string{"comments"},
		Security:    optional,
	}, commentCached(opts, 15*time.Second, service.detailHandler()))
	registerLegacy(a, huma.Operation{
		OperationID: "comment-edit",
		Method:      http.MethodPatch,
		Path:        "/comments/{id}",
		Summary:     "Edit a recent comment",
		Tags:        []string{"comments"},
		Security:    required,
	}, service.editHandler())
	registerLegacy(a, huma.Operation{
		OperationID: "comment-delete",
		Method:      http.MethodDelete,
		Path:        "/comments/{id}",
		Summary:     "Delete a comment",
		Tags:        []string{"comments"},
		Security:    required,
	}, service.deleteHandler(false))
	registerLegacy(a, huma.Operation{
		OperationID: "comment-report",
		Method:      http.MethodPost,
		Path:        "/comments/{id}/report",
		Summary:     "Report a published comment",
		Tags:        []string{"comments"},
		Security:    required,
	}, service.reportHandler())

	myList := service.myCommentsHandler()
	registerLegacy(a, huma.Operation{
		OperationID: "my-comments",
		Method:      http.MethodGet,
		Path:        "/me/comments",
		Summary:     "Current user's comments",
		Tags:        []string{"account", "comments"},
		Security:    required,
	}, myList)
	registerLegacy(a, huma.Operation{
		OperationID: "my-comment-delete",
		Method:      http.MethodDelete,
		Path:        "/me/comments/{id}",
		Summary:     "Delete one of the current user's comments",
		Tags:        []string{"account", "comments"},
		Security:    required,
	}, service.deleteHandler(true))
	registerLegacy(a, huma.Operation{
		OperationID: "my-comments-live-alias",
		Method:      http.MethodGet,
		Path:        "/user/comments",
		Summary:     "Current user's comments (live frontend alias)",
		Tags:        []string{"account", "comments"},
		Security:    required,
	}, myList)
	registerLegacy(a, huma.Operation{
		OperationID: "my-comment-delete-live-alias",
		Method:      http.MethodDelete,
		Path:        "/user/comments/{id}",
		Summary:     "Delete one of the current user's comments (live frontend alias)",
		Tags:        []string{"account", "comments"},
		Security:    required,
	}, service.deleteHandler(true))
}

func commentCached(
	opts Options,
	ttl time.Duration,
	next legacyHandler,
) legacyHandler {
	cached := routeJSONCacheBy(
		opts,
		ttl,
		fmt.Sprintf(
			"public, max-age=0, s-maxage=%d, stale-while-revalidate=%d",
			int(ttl.Seconds()),
			int(ttl.Seconds()),
		),
		func(req *legacyRequest) string {
			host := commentRequestHost(req)
			requestURL := req.Huma.URL()
			return "comments:" + host + ":" + requestURL.RequestURI()
		},
		next,
	)
	return func(
		ctx context.Context,
		req *legacyRequest,
	) (legacyPayload, error) {
		req.Huma.AppendHeader("Vary", "Cookie")
		req.Huma.AppendHeader("Vary", "Host")
		if raw, ok := requestCookie(req.Huma, authSessionCookie); ok && raw != "" {
			// Session revocation and affiliation changes are authorization
			// events for private campaigns. Do not let a cookie-keyed response
			// outlive either event; anonymous traffic remains shared-cached.
			payload, err := next(ctx, req)
			if payload.Headers == nil {
				payload.Headers = make(http.Header)
			}
			payload.Headers.Set("Cache-Control", "private, no-store")
			return payload, err
		}
		return cached(ctx, req)
	}
}

func (s *commentService) listHandler() legacyHandler {
	return func(ctx context.Context, req *legacyRequest) (legacyPayload, error) {
		if err := s.requireDB(); err != nil {
			return legacyPayload{}, err
		}
		principal, domain, err := s.scope(ctx, req)
		if err != nil {
			return legacyPayload{}, err
		}
		limit := boundedContentInt(req.Query.Get("limit"), 50, 1, 100)
		cursor := looseCommentInteger(req.Query.Get("cursor"))
		filters := commentPublicScope("c")
		args := []any{}
		add := func(value any) string {
			args = append(args, value)
			return fmt.Sprintf("$%d", len(args))
		}
		domainValue := any(nil)
		if domain != nil {
			domainValue = domain.ID
		}
		filters = append(
			filters,
			"c.domain_id IS NOT DISTINCT FROM "+add(domainValue),
			"c.parent_id IS NULL",
		)
		if cursor != nil && *cursor != 0 {
			filters = append(filters, "c.id < "+add(*cursor))
		}
		for _, filter := range []struct {
			name   string
			column string
		}{
			{"target_type", "c.target_type"},
			{"character_id", "c.character_id"},
			{"corporation_id", "c.corporation_id"},
			{"alliance_id", "c.alliance_id"},
		} {
			if raw := req.Query.Get(filter.name); raw != "" {
				value := looseCommentIntegerValue(raw)
				filters = append(
					filters,
					filter.column+" = "+add(value),
				)
			}
		}
		search := strings.TrimSpace(req.Query.Get("q"))
		order := "c.id DESC"
		if utf16Length(search) >= 2 {
			placeholder := add(search)
			filters = append(
				filters,
				"c.body_md % "+placeholder,
			)
			order = "similarity(c.body_md, " + placeholder +
				") DESC, c.id DESC"
		}
		visibilitySQL, visibilityArgs := campaignCommentVisibilitySQL(
			len(args),
			domain,
			principal,
		)
		args = append(args, visibilityArgs...)
		filters = append(filters, visibilitySQL)
		args = append(args, limit+1)
		rows, err := queryMaps(ctx, s.db, `
			SELECT `+prefixedCommentColumns("c")+`
			FROM comments c
			WHERE `+strings.Join(filters, " AND ")+`
			ORDER BY `+order+`
			LIMIT $`+strconv.Itoa(len(args)),
			args...,
		)
		if err != nil {
			return legacyPayload{}, err
		}
		hasMore := len(rows) > limit
		if hasMore {
			rows = rows[:limit]
		}
		if err := s.addReplyCounts(ctx, rows, false); err != nil {
			return legacyPayload{}, err
		}
		var next any
		if hasMore && len(rows) != 0 {
			next = rows[len(rows)-1]["id"]
		}
		return jsonPayload(map[string]any{
			"comments": rows, "nextCursor": next,
		}), nil
	}
}

func (s *commentService) threadHandler() legacyHandler {
	return func(ctx context.Context, req *legacyRequest) (legacyPayload, error) {
		if err := s.requireDB(); err != nil {
			return legacyPayload{}, err
		}
		targetType, targetID, targetSlug, err :=
			commentTargetFromQuery(req)
		if err != nil {
			return legacyPayload{}, err
		}
		principal, domain, err := s.scope(ctx, req)
		if err != nil {
			return legacyPayload{}, err
		}
		if err := s.requireCampaignTarget(
			ctx, targetType, targetSlug, domain, principal,
		); err != nil {
			return legacyPayload{}, err
		}
		limit := boundedContentInt(req.Query.Get("limit"), 50, 1, 100)
		cursor := looseCommentInteger(req.Query.Get("cursor"))
		domainValue := any(nil)
		if domain != nil {
			domainValue = domain.ID
		}
		args := []any{domainValue, targetType, targetID}
		filters := []string{
			"c.deleted_at IS NULL",
			"c.visibility = 0",
			"c.domain_id IS NOT DISTINCT FROM $1",
			"c.target_type = $2",
			"c.target_id = $3",
			"c.parent_id IS NULL",
		}
		if targetSlug != nil {
			args = append(args, *targetSlug)
			filters = append(
				filters,
				"c.target_slug = $"+strconv.Itoa(len(args)),
			)
		}
		if cursor != nil && *cursor != 0 {
			args = append(args, *cursor)
			filters = append(
				filters,
				"c.id < $"+strconv.Itoa(len(args)),
			)
		}
		args = append(args, limit+1)
		roots, err := queryMaps(ctx, s.db, `
			SELECT `+prefixedCommentColumns("c")+`
			FROM comments c
			WHERE `+strings.Join(filters, " AND ")+`
			ORDER BY c.id DESC
			LIMIT $`+strconv.Itoa(len(args)),
			args...,
		)
		if err != nil {
			return legacyPayload{}, err
		}
		hasMore := len(roots) > limit
		if hasMore {
			roots = roots[:limit]
		}
		replies := []map[string]any{}
		truncated := false
		rootIDs := commentMapIDs(roots)
		if len(rootIDs) != 0 {
			replies, err = queryMaps(ctx, s.db, `
				SELECT `+prefixedCommentColumns("c")+`
				FROM comments c
				WHERE c.deleted_at IS NULL
				  AND c.visibility = 0
				  AND c.domain_id IS NOT DISTINCT FROM $1
				  AND c.root_id = ANY($2::bigint[])
				ORDER BY c.created_at ASC
				LIMIT $3`,
				domainValue,
				rootIDs,
				commentReplyHardCap+1,
			)
			if err != nil {
				return legacyPayload{}, err
			}
			truncated = len(replies) > commentReplyHardCap
			if truncated {
				replies = replies[:commentReplyHardCap]
			}
		}
		var next any
		if hasMore && len(roots) != 0 {
			next = roots[len(roots)-1]["id"]
		}
		return jsonPayload(map[string]any{
			"roots":            roots,
			"replies":          replies,
			"repliesTruncated": truncated,
			"nextCursor":       next,
		}), nil
	}
}

func (s *commentService) detailHandler() legacyHandler {
	return func(ctx context.Context, req *legacyRequest) (legacyPayload, error) {
		if err := s.requireDB(); err != nil {
			return legacyPayload{}, err
		}
		id, err := commentID(req.Param("id"), "Invalid comment id")
		if err != nil {
			return legacyPayload{}, err
		}
		principal, domain, err := s.scope(ctx, req)
		if err != nil {
			return legacyPayload{}, err
		}
		domainValue := any(nil)
		if domain != nil {
			domainValue = domain.ID
		}
		row, err := queryMap(ctx, s.db, `
			SELECT `+prefixedCommentColumns("c")+`
			FROM comments c
			WHERE c.id = $1
			  AND c.deleted_at IS NULL
			  AND c.visibility = 0
			  AND c.domain_id IS NOT DISTINCT FROM $2
			LIMIT 1`,
			id,
			domainValue,
		)
		if err != nil {
			return legacyPayload{}, err
		}
		if row == nil {
			return legacyPayload{}, apiError(
				http.StatusNotFound,
				"Comment not found",
			)
		}
		if err := s.requireCampaignTarget(
			ctx,
			int16(mapInt64(row, "target_type", 0)),
			mapOptionalString(row, "target_slug"),
			domain,
			principal,
		); err != nil {
			return legacyPayload{}, err
		}
		return jsonPayload(map[string]any{"comment": row}), nil
	}
}

func (s *commentService) createHandler() legacyHandler {
	return func(ctx context.Context, req *legacyRequest) (legacyPayload, error) {
		setAccountNoStore(req.Huma)
		if err := requireSameOriginMutation(req.Huma); err != nil {
			return legacyPayload{}, err
		}
		if err := s.requireDB(); err != nil {
			return legacyPayload{}, err
		}
		principal, err := s.auth.requirePrincipal(ctx, req)
		if err != nil {
			return legacyPayload{}, err
		}
		body, err := decodeContentBody(req, false)
		if err != nil {
			return legacyPayload{}, err
		}
		targetType, targetID, targetSlug, err :=
			commentTargetFromBody(body)
		if err != nil {
			return legacyPayload{}, err
		}
		domain, err := s.resolveDomain(ctx, req)
		if err != nil {
			return legacyPayload{}, err
		}
		if err := s.requireCampaignTarget(
			ctx, targetType, targetSlug, domain, principal,
		); err != nil {
			return legacyPayload{}, err
		}
		bodyMD, err := validateCommentBodyValue(body["body_md"])
		if err != nil {
			return legacyPayload{}, err
		}
		rate := checkCommentRateLimit(
			ctx, s.cache, principal.CharacterID, bodyMD,
		)
		if !rate.OK {
			req.Huma.SetHeader(
				"Retry-After",
				strconv.Itoa(rate.RetryAfter),
			)
			message := "Rate limit exceeded"
			if rate.Reason == "duplicate" {
				message = "Duplicate comment"
			}
			return legacyPayload{}, apiError(
				http.StatusTooManyRequests,
				message,
			)
		}
		cooldown := checkCommentModerationCooldown(
			ctx, s.cache, principal.CharacterID,
		)
		if !cooldown.OK {
			req.Huma.SetHeader(
				"Retry-After",
				strconv.Itoa(cooldown.RetryAfter),
			)
			return legacyPayload{}, apiError(
				http.StatusTooManyRequests,
				"Too many rejected comments. Please wait before trying again.",
			)
		}
		rendered := s.renderer.Render(ctx, bodyMD)
		images := commentImageURLs(rendered)
		verdict := s.moderator.Moderate(ctx, bodyMD, images)
		if verdict.Action != commentActionPass {
			recordCommentModerationRejection(
				ctx, s.cache, principal.CharacterID,
			)
		}
		visibility, queueStatus := commentVerdictStates(verdict.Action)

		var parentID *int64
		if raw, found := body["parent_id"]; found && raw != nil {
			value, ok := contentInteger(raw)
			if !ok || value <= 0 {
				return legacyPayload{}, apiError(
					http.StatusBadRequest,
					"Invalid parent_id",
				)
			}
			parentID = &value
		}
		rootID, depth, ancestors, err := s.resolveCommentParent(
			ctx,
			parentID,
			targetType,
			targetID,
			targetSlug,
			domain,
		)
		if err != nil {
			return legacyPayload{}, err
		}
		row, queueID, err := s.insertComment(
			ctx,
			commentInsert{
				TargetType:  targetType,
				TargetID:    targetID,
				TargetSlug:  targetSlug,
				Domain:      domain,
				ParentID:    parentID,
				RootID:      rootID,
				Depth:       depth,
				BodyMD:      bodyMD,
				BodyHTML:    rendered,
				Principal:   principal,
				Visibility:  visibility,
				Verdict:     verdict,
				QueueStatus: queueStatus,
			},
		)
		if err != nil {
			return legacyPayload{}, err
		}
		if queueStatus == moderationStatusPending && s.dispatcher != nil {
			s.dispatcher.ModerationPending(
				context.WithoutCancel(ctx),
				queueID,
				row,
				verdict,
			)
		}
		if verdict.Action == commentActionDelete {
			message := "Your comment was rejected by automated moderation."
			if verdict.UserMessage != nil && *verdict.UserMessage != "" {
				message = *verdict.UserMessage
			}
			return legacyPayload{}, apiError(
				http.StatusUnprocessableEntity,
				message,
			)
		}
		if verdict.Action == commentActionPass {
			s.dispatchComment(ctx, "new", row, ancestors)
			return accountNoStorePayload(map[string]any{
				"comment": row, "status": "published",
			}), nil
		}
		return accountNoStorePayload(map[string]any{
			"comment": row, "status": "pending_review",
		}), nil
	}
}

func (s *commentService) editHandler() legacyHandler {
	return func(ctx context.Context, req *legacyRequest) (legacyPayload, error) {
		setAccountNoStore(req.Huma)
		if err := requireSameOriginMutation(req.Huma); err != nil {
			return legacyPayload{}, err
		}
		if err := s.requireDB(); err != nil {
			return legacyPayload{}, err
		}
		principal, err := s.auth.requirePrincipal(ctx, req)
		if err != nil {
			return legacyPayload{}, err
		}
		id, err := commentID(req.Param("id"), "Invalid comment id")
		if err != nil {
			return legacyPayload{}, err
		}
		body, err := decodeContentBody(req, false)
		if err != nil {
			return legacyPayload{}, err
		}
		bodyMD, err := validateCommentBodyValue(body["body_md"])
		if err != nil {
			return legacyPayload{}, err
		}
		domain, err := s.resolveDomain(ctx, req)
		if err != nil {
			return legacyPayload{}, err
		}
		current, err := s.activeComment(ctx, id, domain)
		if err != nil {
			return legacyPayload{}, err
		}
		if current == nil {
			return legacyPayload{}, apiError(
				http.StatusNotFound,
				"Comment not found",
			)
		}
		if mapInt64(current, "character_id", 0) !=
			int64(principal.CharacterID) {
			return legacyPayload{}, apiError(
				http.StatusForbidden,
				"Not your comment",
			)
		}
		if mapInt64(current, "moderation_status", 0) ==
			commentModerationAdminHidden ||
			mapInt64(current, "visibility", 0) ==
				commentVisibilityHidden {
			return legacyPayload{}, apiError(
				http.StatusForbidden,
				"Comment is hidden by moderation",
			)
		}
		createdAt, ok := timeFrom(current["created_at"])
		if !ok || s.now().UTC().Sub(createdAt) > commentEditWindow {
			return legacyPayload{}, apiError(
				http.StatusForbidden,
				"Edit window expired (24h)",
			)
		}
		if bodyMD == mapString(current, "body_md", "") {
			return accountNoStorePayload(
				map[string]any{"comment": current},
			), nil
		}
		rendered := s.renderer.Render(ctx, bodyMD)
		verdict := s.moderator.Moderate(
			ctx,
			bodyMD,
			commentImageURLs(rendered),
		)
		visibility, queueStatus := commentVerdictStates(verdict.Action)
		row, queueID, err := s.updateCommentBody(
			ctx,
			current,
			principal,
			bodyMD,
			rendered,
			visibility,
			verdict,
			queueStatus,
			s.now().UTC(),
		)
		if err != nil {
			return legacyPayload{}, err
		}
		if queueStatus == moderationStatusPending && s.dispatcher != nil {
			s.dispatcher.ModerationPending(
				context.WithoutCancel(ctx),
				queueID,
				row,
				verdict,
			)
		}
		// The TypeScript edit path only changed moderation_status, leaving a
		// harmful replacement body publicly visible. Hidden edits deliberately
		// do not go onto the public relay; an approval dispatches them later.
		if verdict.Action == commentActionPass {
			s.dispatchComment(ctx, "edited", row, nil)
		}
		return accountNoStorePayload(
			map[string]any{"comment": row},
		), nil
	}
}

func (s *commentService) deleteHandler(crossDomain bool) legacyHandler {
	return func(ctx context.Context, req *legacyRequest) (legacyPayload, error) {
		setAccountNoStore(req.Huma)
		if err := requireSameOriginMutation(req.Huma); err != nil {
			return legacyPayload{}, err
		}
		if err := s.requireDB(); err != nil {
			return legacyPayload{}, err
		}
		principal, err := s.auth.requirePrincipal(ctx, req)
		if err != nil {
			return legacyPayload{}, err
		}
		id, err := commentID(req.Param("id"), "Invalid comment id")
		if err != nil {
			return legacyPayload{}, err
		}
		var current map[string]any
		// Admin moderation is global. The old shared delete endpoint applied
		// the current host's domain scope before checking IsAdmin, so the
		// global moderation page could not delete a custom-domain comment.
		if crossDomain || principal.IsAdmin {
			current, err = queryMap(ctx, s.db, `
				SELECT `+commentColumns+`
				FROM comments
				WHERE id = $1
				  AND deleted_at IS NULL
				LIMIT 1`,
				id,
			)
		} else {
			var domain *commentDomain
			domain, err = s.resolveDomain(ctx, req)
			if err == nil {
				current, err = s.activeComment(ctx, id, domain)
			}
		}
		if err != nil {
			return legacyPayload{}, err
		}
		if current == nil {
			return legacyPayload{}, apiError(
				http.StatusNotFound,
				"Comment not found",
			)
		}
		if mapInt64(current, "character_id", 0) !=
			int64(principal.CharacterID) &&
			(!principal.IsAdmin || crossDomain) {
			return legacyPayload{}, apiError(
				http.StatusForbidden,
				"Not allowed",
			)
		}
		row, err := queryMap(ctx, s.db, `
			UPDATE comments
			SET deleted_at = $2,
				deleted_by = $3,
				updated_at = $2
			WHERE id = $1
			RETURNING `+commentColumns,
			id,
			s.now().UTC(),
			principal.CharacterID,
		)
		if err != nil {
			return legacyPayload{}, err
		}
		s.dispatchComment(ctx, "deleted", row, nil)
		return accountNoStorePayload(map[string]any{"ok": true}), nil
	}
}

func (s *commentService) reportHandler() legacyHandler {
	validReasons := map[string]bool{
		"spam": true, "harassment": true, "nsfw": true,
		"offtopic": true, "other": true,
	}
	return func(ctx context.Context, req *legacyRequest) (legacyPayload, error) {
		setAccountNoStore(req.Huma)
		if err := requireSameOriginMutation(req.Huma); err != nil {
			return legacyPayload{}, err
		}
		if err := s.requireDB(); err != nil {
			return legacyPayload{}, err
		}
		principal, err := s.auth.requirePrincipal(ctx, req)
		if err != nil {
			return legacyPayload{}, err
		}
		id, err := commentID(req.Param("id"), "Invalid comment id")
		if err != nil {
			return legacyPayload{}, err
		}
		body, err := decodeContentBody(req, false)
		if err != nil {
			return legacyPayload{}, err
		}
		reason := strings.ToLower(
			strings.TrimSpace(stringField(body["reason"])),
		)
		if !validReasons[reason] {
			return legacyPayload{}, apiError(
				http.StatusBadRequest,
				"reason must be one of: spam, harassment, nsfw, offtopic, other",
			)
		}
		message := optionalTrimmedString(body["message"], 1000)
		domain, err := s.resolveDomain(ctx, req)
		if err != nil {
			return legacyPayload{}, err
		}
		domainValue := any(nil)
		if domain != nil {
			domainValue = domain.ID
		}
		row, err := queryMap(ctx, s.db, `
			SELECT `+commentColumns+`
			FROM comments
			WHERE id = $1
			  AND deleted_at IS NULL
			  AND visibility = 0
			  AND domain_id IS NOT DISTINCT FROM $2
			LIMIT 1`,
			id,
			domainValue,
		)
		if err != nil {
			return legacyPayload{}, err
		}
		if row == nil {
			return legacyPayload{}, apiError(
				http.StatusNotFound,
				"Comment not found",
			)
		}
		if err := s.requireCampaignTarget(
			ctx,
			int16(mapInt64(row, "target_type", 0)),
			mapOptionalString(row, "target_slug"),
			domain,
			principal,
		); err != nil {
			return legacyPayload{}, err
		}
		if mapInt64(row, "character_id", 0) ==
			int64(principal.CharacterID) {
			return legacyPayload{}, apiError(
				http.StatusBadRequest,
				"Cannot report your own comment",
			)
		}
		fresh, err := s.insertCommentReport(
			ctx,
			row,
			principal,
			reason,
			message,
			s.now().UTC(),
		)
		if isUniqueViolation(err) {
			return legacyPayload{}, apiError(
				http.StatusConflict,
				"Already reported",
			)
		}
		if err != nil {
			return legacyPayload{}, err
		}
		if s.dispatcher != nil {
			s.dispatcher.Reported(
				context.WithoutCancel(ctx),
				fresh,
				principal,
				reason,
				message,
			)
		}
		return accountNoStorePayload(map[string]any{
			"ok":            true,
			"reports_count": fresh["reports_count"],
			"flagged":       fresh["flagged"],
		}), nil
	}
}

func (s *commentService) previewHandler() legacyHandler {
	return func(ctx context.Context, req *legacyRequest) (legacyPayload, error) {
		setAccountNoStore(req.Huma)
		if err := requireSameOriginMutation(req.Huma); err != nil {
			return legacyPayload{}, err
		}
		principal, err := s.auth.requirePrincipal(ctx, req)
		if err != nil {
			return legacyPayload{}, err
		}
		body, err := decodeContentBody(req, false)
		if err != nil {
			return legacyPayload{}, err
		}
		raw := stringField(body["body_md"])
		if strings.TrimSpace(raw) == "" {
			return accountNoStorePayload(
				map[string]any{"html": ""},
			), nil
		}
		rate := checkCommentPreviewRateLimit(
			ctx,
			s.cache,
			principal.CharacterID,
		)
		if !rate.OK {
			req.Huma.SetHeader(
				"Retry-After",
				strconv.Itoa(rate.RetryAfter),
			)
			return legacyPayload{}, apiError(
				http.StatusTooManyRequests,
				"Preview rate limit exceeded",
			)
		}
		value, validationErr := validateCommentBody(raw)
		if validationErr != nil {
			return accountNoStorePayload(map[string]any{
				"html": "", "error": validationErr.Error(),
			}), nil
		}
		return accountNoStorePayload(map[string]any{
			"html": s.renderer.Render(ctx, value),
		}), nil
	}
}

func (s *commentService) myCommentsHandler() legacyHandler {
	return func(ctx context.Context, req *legacyRequest) (legacyPayload, error) {
		setAccountNoStore(req.Huma)
		if err := s.requireDB(); err != nil {
			return legacyPayload{}, err
		}
		principal, err := s.auth.requirePrincipal(ctx, req)
		if err != nil {
			return legacyPayload{}, err
		}
		limit := boundedContentInt(req.Query.Get("limit"), 25, 1, 100)
		args := []any{principal.CharacterID}
		filters := []string{
			"character_id = $1",
			"deleted_at IS NULL",
		}
		if cursor := looseCommentInteger(req.Query.Get("cursor")); cursor != nil && *cursor != 0 {
			args = append(args, *cursor)
			filters = append(
				filters,
				"id < $"+strconv.Itoa(len(args)),
			)
		}
		args = append(args, limit+1)
		rows, err := queryMaps(ctx, s.db, `
			SELECT
				id, target_type, target_id, target_slug, domain_id,
				parent_id, root_id, depth, body_md, body_html,
				character_id, character_name, corporation_id,
				corporation_name, alliance_id, alliance_name,
				created_at, updated_at, edited_at,
				moderation_status, visibility
			FROM comments
			WHERE `+strings.Join(filters, " AND ")+`
			ORDER BY id DESC
			LIMIT $`+strconv.Itoa(len(args)),
			args...,
		)
		if err != nil {
			return legacyPayload{}, err
		}
		hasMore := len(rows) > limit
		if hasMore {
			rows = rows[:limit]
		}
		if err := s.addReplyCounts(ctx, rows, true); err != nil {
			return legacyPayload{}, err
		}
		var next any
		if hasMore && len(rows) != 0 {
			next = rows[len(rows)-1]["id"]
		}
		return accountNoStorePayload(map[string]any{
			"comments": rows, "nextCursor": next,
		}), nil
	}
}

func (s *commentService) klipyHandler(kind string) legacyHandler {
	return func(ctx context.Context, req *legacyRequest) (legacyPayload, error) {
		setAccountNoStore(req.Huma)
		principal, err := s.auth.requirePrincipal(ctx, req)
		if err != nil {
			return legacyPayload{}, err
		}
		rate := checkCommentKlipyRateLimit(
			ctx,
			s.cache,
			principal.CharacterID,
		)
		if !rate.OK {
			req.Huma.SetHeader(
				"Retry-After",
				strconv.Itoa(rate.RetryAfter),
			)
			return legacyPayload{}, apiError(
				http.StatusTooManyRequests,
				"Klipy rate limit exceeded",
			)
		}
		page := boundedContentInt(
			req.Query.Get("page"), 1, 1, math.MaxInt32,
		)
		perPage := boundedContentInt(
			req.Query.Get("per_page"), 24, 1, 48,
		)
		search := ""
		if kind == "search" {
			search = strings.ToLower(
				strings.TrimSpace(req.Query.Get("q")),
			)
			if utf16Length(search) < 2 {
				return legacyPayload{}, apiError(
					http.StatusBadRequest,
					"q must be at least 2 characters",
				)
			}
		}
		payload, err := s.klipy.Fetch(
			ctx, kind, search, page, perPage,
		)
		if err != nil {
			return legacyPayload{}, err
		}
		return accountNoStorePayload(payload), nil
	}
}

func (s *commentService) scope(
	ctx context.Context,
	req *legacyRequest,
) (*Principal, *commentDomain, error) {
	principal, err := s.optionalPrincipal(ctx, req)
	if err != nil {
		return nil, nil, err
	}
	domain, err := s.resolveDomain(ctx, req)
	if err != nil {
		return nil, nil, err
	}
	return principal, domain, nil
}

func (s *commentService) optionalPrincipal(
	ctx context.Context,
	req *legacyRequest,
) (*Principal, error) {
	if raw, ok := requestCookie(req.Huma, authSessionCookie); !ok || raw == "" {
		return nil, nil
	}
	return s.auth.resolvePrincipal(ctx, req)
}

func (s *commentService) resolveDomain(
	ctx context.Context,
	req *legacyRequest,
) (*commentDomain, error) {
	host := normalizeCommentHost(commentRequestHost(req))
	if host == "" || isMainCommentHost(host) {
		return nil, nil
	}
	var lookup string
	var query string
	switch {
	case strings.HasSuffix(host, ".eve-kill.com"):
		lookup = strings.TrimSuffix(host, ".eve-kill.com")
		if lookup == "" || strings.Contains(lookup, ".") {
			return nil, nil
		}
		query = `
			SELECT id, user_id, entities
			FROM custom_domains
			WHERE active IS TRUE
			  AND lower(subdomain) = $1
			LIMIT 1`
	case strings.HasSuffix(host, ".localhost"):
		lookup = strings.TrimSuffix(host, ".localhost")
		if lookup == "" || strings.Contains(lookup, ".") {
			return nil, nil
		}
		query = `
			SELECT id, user_id, entities
			FROM custom_domains
			WHERE active IS TRUE
			  AND lower(subdomain) = $1
			LIMIT 1`
	default:
		lookup = host
		query = `
			SELECT id, user_id, entities
			FROM custom_domains
			WHERE active IS TRUE
			  AND lower(custom_hostname) = $1
			LIMIT 1`
	}
	var domain commentDomain
	var raw []byte
	err := s.db.QueryRow(ctx, query, lookup).Scan(
		&domain.ID,
		&domain.OwnerID,
		&raw,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	_ = json.Unmarshal(raw, &domain.Entities)
	return &domain, nil
}

func (s *commentService) requireCampaignTarget(
	ctx context.Context,
	targetType int16,
	targetSlug *string,
	domain *commentDomain,
	principal *Principal,
) error {
	if targetType != commentTargetCampaign {
		return nil
	}
	slug := ""
	if targetSlug != nil {
		slug = *targetSlug
	}
	var visibility int16
	var creator int32
	err := s.db.QueryRow(ctx, `
		SELECT visibility, created_by_character_id
		FROM campaigns
		WHERE campaign_id = $1
		LIMIT 1`,
		slug,
	).Scan(&visibility, &creator)
	if errors.Is(err, pgx.ErrNoRows) {
		return apiError(http.StatusNotFound, "Campaign not found")
	}
	if err != nil {
		return err
	}
	if visibility != campaignVisibilityPrivate {
		return nil
	}
	if principal != nil &&
		(principal.IsAdmin || principal.CharacterID == creator) {
		return nil
	}
	if domain == nil {
		return apiError(http.StatusNotFound, "Campaign not found")
	}
	var public bool
	err = s.db.QueryRow(ctx, `
		SELECT public_on_domain
		FROM custom_domain_campaigns
		WHERE domain_id = $1
		  AND campaign_id = $2
		LIMIT 1`,
		domain.ID,
		slug,
	).Scan(&public)
	if errors.Is(err, pgx.ErrNoRows) {
		return apiError(http.StatusNotFound, "Campaign not found")
	}
	if err != nil {
		return err
	}
	if !public && !commentDomainMember(domain, principal) {
		return apiError(http.StatusNotFound, "Campaign not found")
	}
	return nil
}

func (s *commentService) activeComment(
	ctx context.Context,
	id int64,
	domain *commentDomain,
) (map[string]any, error) {
	var domainValue any
	if domain != nil {
		domainValue = domain.ID
	}
	return queryMap(ctx, s.db, `
		SELECT `+commentColumns+`
		FROM comments
		WHERE id = $1
		  AND deleted_at IS NULL
		  AND domain_id IS NOT DISTINCT FROM $2
		LIMIT 1`,
		id,
		domainValue,
	)
}

func (s *commentService) resolveCommentParent(
	ctx context.Context,
	parentID *int64,
	targetType int16,
	targetID int64,
	targetSlug *string,
	domain *commentDomain,
) (*int64, int16, []commentAncestor, error) {
	if parentID == nil {
		return nil, 0, nil, nil
	}
	parent, err := queryMap(ctx, s.db, `
		SELECT id, root_id, depth, target_type, target_id,
		       target_slug, domain_id
		FROM comments
		WHERE id = $1
		  AND deleted_at IS NULL
		LIMIT 1`,
		*parentID,
	)
	if err != nil {
		return nil, 0, nil, err
	}
	if parent == nil {
		return nil, 0, nil, apiError(
			http.StatusNotFound,
			"Parent comment not found",
		)
	}
	if int16(mapInt64(parent, "target_type", 0)) != targetType ||
		mapInt64(parent, "target_id", -1) != targetID {
		return nil, 0, nil, apiError(
			http.StatusBadRequest,
			"Parent target mismatch",
		)
	}
	if targetSlug != nil &&
		!optionalStringsEqual(
			mapOptionalString(parent, "target_slug"),
			targetSlug,
		) {
		return nil, 0, nil, apiError(
			http.StatusBadRequest,
			"Parent target mismatch",
		)
	}
	var expectedDomain *int64
	if domain != nil {
		value := int64(domain.ID)
		expectedDomain = &value
	}
	if !optionalInt64Equal(
		mapOptionalInt64(parent, "domain_id"),
		expectedDomain,
	) {
		return nil, 0, nil, apiError(
			http.StatusBadRequest,
			"Parent domain mismatch",
		)
	}
	rootID := mapOptionalInt64(parent, "root_id")
	if rootID == nil {
		value := mapInt64(parent, "id", *parentID)
		rootID = &value
	}
	depth := mapInt64(parent, "depth", 0) + 1
	if depth > math.MaxInt16 {
		return nil, 0, nil, apiError(
			http.StatusBadRequest,
			"Maximum comment nesting reached",
		)
	}
	ancestors, err := s.commentAncestry(ctx, *parentID)
	return rootID, int16(depth), ancestors, err
}

func (s *commentService) commentAncestry(
	ctx context.Context,
	parentID int64,
) ([]commentAncestor, error) {
	rows, err := s.db.Query(ctx, `
		WITH RECURSIVE ancestry(
			id, parent_id, character_id, character_name, depth_up
		) AS (
			SELECT id, parent_id, character_id, character_name, 0
			FROM comments
			WHERE id = $1
			UNION ALL
			SELECT c.id, c.parent_id, c.character_id,
			       c.character_name, a.depth_up + 1
			FROM comments c
			INNER JOIN ancestry a ON c.id = a.parent_id
		)
		SELECT id, character_id, character_name
		FROM ancestry
		ORDER BY depth_up DESC`,
		parentID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	result := []commentAncestor{}
	for rows.Next() {
		var item commentAncestor
		if err := rows.Scan(
			&item.ID,
			&item.CharacterID,
			&item.CharacterName,
		); err != nil {
			return nil, err
		}
		result = append(result, item)
	}
	return result, rows.Err()
}

func (s *commentService) addReplyCounts(
	ctx context.Context,
	rows []map[string]any,
	rootsOnly bool,
) error {
	rootIDs := []int64{}
	for _, row := range rows {
		if rootsOnly && row["parent_id"] != nil {
			continue
		}
		if id := mapInt64(row, "id", 0); id > 0 {
			rootIDs = append(rootIDs, id)
		}
	}
	counts := map[int64]int64{}
	if len(rootIDs) != 0 {
		countRows, err := s.db.Query(ctx, `
			SELECT root_id, COUNT(*)::int
			FROM comments
			WHERE root_id = ANY($1::bigint[])
			  AND deleted_at IS NULL
			  AND visibility = 0
			GROUP BY root_id`,
			rootIDs,
		)
		if err != nil {
			return err
		}
		defer countRows.Close()
		for countRows.Next() {
			var id int64
			var count int32
			if err := countRows.Scan(&id, &count); err != nil {
				return err
			}
			counts[id] = int64(count)
		}
		if err := countRows.Err(); err != nil {
			return err
		}
	}
	for _, row := range rows {
		count := int64(0)
		if !rootsOnly || row["parent_id"] == nil {
			count = counts[mapInt64(row, "id", 0)]
		}
		row["reply_count"] = count
	}
	return nil
}

func (s *commentService) dispatchComment(
	ctx context.Context,
	event string,
	row map[string]any,
	ancestors []commentAncestor,
) {
	if s.dispatcher == nil || row == nil {
		return
	}
	if mapInt64(row, "target_type", 0) == commentTargetCampaign {
		slug := mapString(row, "target_slug", "")
		var visibility int16
		err := s.db.QueryRow(ctx, `
			SELECT visibility
			FROM campaigns
			WHERE campaign_id = $1`,
			slug,
		).Scan(&visibility)
		if err != nil || visibility == campaignVisibilityPrivate {
			return
		}
	}
	s.dispatcher.Comment(
		context.WithoutCancel(ctx),
		event,
		row,
		ancestors,
	)
}

func commentRequestHost(req *legacyRequest) string {
	return legacyRequestHost(req)
}

func normalizeCommentHost(raw string) string {
	host := strings.ToLower(strings.TrimSpace(raw))
	if parsed, _, err := net.SplitHostPort(host); err == nil {
		host = parsed
	} else if strings.Count(host, ":") == 1 {
		host, _, _ = strings.Cut(host, ":")
	}
	return strings.TrimSuffix(host, ".")
}

func isMainCommentHost(host string) bool {
	return host == "" ||
		host == "eve-kill.com" ||
		host == "www.eve-kill.com" ||
		host == "zkillboard.co" ||
		host == "localhost" ||
		host == "127.0.0.1" ||
		net.ParseIP(host) != nil ||
		!strings.Contains(host, ".")
}

func commentDomainMember(
	domain *commentDomain,
	principal *Principal,
) bool {
	if domain == nil || principal == nil {
		return false
	}
	if principal.IsAdmin || principal.CharacterID == domain.OwnerID {
		return true
	}
	for _, entity := range domain.Entities {
		switch strings.ToLower(entity.Type) {
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

func campaignCommentVisibilitySQL(
	argOffset int,
	domain *commentDomain,
	principal *Principal,
) (string, []any) {
	args := []any{}
	add := func(value any) string {
		args = append(args, value)
		return "$" + strconv.Itoa(argOffset+len(args))
	}
	characterID := int32(0)
	isAdmin := false
	if principal != nil {
		characterID = principal.CharacterID
		isAdmin = principal.IsAdmin
	}
	admin := add(isAdmin)
	character := add(characterID)
	if domain == nil {
		return `(c.target_type <> 10 OR EXISTS (
			SELECT 1
			FROM campaigns campaign
			WHERE campaign.campaign_id = c.target_slug
			  AND (
				campaign.visibility <> 1
				OR ` + admin + `
				OR campaign.created_by_character_id = ` + character + `
			  )
		))`, args
	}
	domainID := add(domain.ID)
	member := add(commentDomainMember(domain, principal))
	return `(c.target_type <> 10 OR EXISTS (
		SELECT 1
		FROM campaigns campaign
		WHERE campaign.campaign_id = c.target_slug
		  AND (
			campaign.visibility <> 1
			OR ` + admin + `
			OR campaign.created_by_character_id = ` + character + `
			OR EXISTS (
				SELECT 1
				FROM custom_domain_campaigns selection
				WHERE selection.domain_id = ` + domainID + `
				  AND selection.campaign_id = campaign.campaign_id
				  AND (` + member + ` OR selection.public_on_domain IS TRUE)
			)
		  )
	))`, args
}

func commentPublicScope(
	alias string,
) []string {
	return []string{
		alias + ".deleted_at IS NULL",
		alias + ".visibility = 0",
	}
}

func prefixedCommentColumns(alias string) string {
	parts := strings.Split(commentColumns, ",")
	for index, part := range parts {
		parts[index] = alias + "." + strings.TrimSpace(part)
	}
	return strings.Join(parts, ", ")
}

func commentMapIDs(rows []map[string]any) []int64 {
	result := make([]int64, 0, len(rows))
	for _, row := range rows {
		if id := mapInt64(row, "id", 0); id > 0 {
			result = append(result, id)
		}
	}
	return result
}

func commentTargetFromQuery(
	req *legacyRequest,
) (int16, int64, *string, error) {
	targetType := looseCommentIntegerValue(
		req.Query.Get("target_type"),
	)
	targetID := looseCommentIntegerValue(req.Query.Get("target_id"))
	var slug *string
	if raw := req.Query.Get("target_slug"); raw != "" {
		slug = &raw
	}
	err := validateCommentTarget(targetType, targetID, slug)
	return int16(targetType), targetID, slug, err
}

func commentTargetFromBody(
	body map[string]any,
) (int16, int64, *string, error) {
	targetType, typeOK := contentInteger(body["target_type"])
	targetID, idOK := contentInteger(body["target_id"])
	if !typeOK {
		targetType = 0
	}
	if !idOK {
		targetID = math.MinInt64
	}
	var slug *string
	if raw, ok := body["target_slug"].(string); ok && raw != "" {
		value := raw
		slug = &value
	}
	err := validateCommentTarget(targetType, targetID, slug)
	return int16(targetType), targetID, slug, err
}

func validateCommentTarget(
	targetType int64,
	targetID int64,
	targetSlug *string,
) error {
	if targetType < commentTargetKillmail ||
		targetType > commentTargetCampaign {
		return apiError(
			http.StatusBadRequest,
			fmt.Sprintf("Invalid target_type: %d", targetType),
		)
	}
	if targetID < 0 {
		return apiError(
			http.StatusBadRequest,
			"target_id must be a non-negative integer",
		)
	}
	if targetType == commentTargetPage ||
		targetType == commentTargetFit ||
		targetType == commentTargetCampaign {
		if targetSlug == nil || *targetSlug == "" {
			return apiError(
				http.StatusBadRequest,
				"target_slug is required for this target_type",
			)
		}
		if utf16Length(*targetSlug) > 255 {
			return apiError(
				http.StatusBadRequest,
				"target_slug too long",
			)
		}
	}
	return nil
}

func validateCommentBodyValue(value any) (string, error) {
	if value == nil {
		return validateCommentBody("")
	}
	raw, ok := value.(string)
	if !ok {
		return "", apiError(
			http.StatusBadRequest,
			"body_md must be a string",
		)
	}
	return validateCommentBody(raw)
}

func validateCommentBody(raw string) (string, error) {
	trimmed := strings.TrimSpace(raw)
	length := utf16Length(trimmed)
	if length < commentMinimumLength {
		return "", apiError(
			http.StatusBadRequest,
			"Comment is too short",
		)
	}
	if length > commentMaximumLength {
		return "", apiError(
			http.StatusBadRequest,
			fmt.Sprintf(
				"Comment exceeds %d characters",
				commentMaximumLength,
			),
		)
	}
	return trimmed, nil
}

func utf16Length(value string) int {
	return len(utf16.Encode([]rune(value)))
}

func commentID(raw, message string) (int64, error) {
	value, err := strconv.ParseInt(strings.TrimSpace(raw), 10, 64)
	if err != nil || value <= 0 {
		return 0, apiError(http.StatusBadRequest, message)
	}
	return value, nil
}

func looseCommentInteger(raw string) *int64 {
	if strings.TrimSpace(raw) == "" {
		return nil
	}
	value := looseCommentIntegerValue(raw)
	return &value
}

func looseCommentIntegerValue(raw string) int64 {
	value, err := strconv.ParseFloat(strings.TrimSpace(raw), 64)
	if err != nil || math.IsNaN(value) || math.IsInf(value, 0) {
		return 0
	}
	return int64(value)
}

func commentVerdictStates(
	action commentModerationAction,
) (int16, int16) {
	switch action {
	case commentActionFlag:
		return commentVisibilityPending, moderationStatusPending
	case commentActionDelete:
		return commentVisibilityRejected, moderationStatusAutoReject
	default:
		return commentVisibilityPublished, moderationStatusAutoApprove
	}
}

func optionalStringsEqual(first, second *string) bool {
	if first == nil || second == nil {
		return first == nil && second == nil
	}
	return *first == *second
}

func optionalInt64Equal(first, second *int64) bool {
	if first == nil || second == nil {
		return first == nil && second == nil
	}
	return *first == *second
}

func mapOptionalString(row map[string]any, key string) *string {
	if row == nil || row[key] == nil {
		return nil
	}
	value := stringField(row[key])
	return &value
}

func mapOptionalInt64(row map[string]any, key string) *int64 {
	if row == nil || row[key] == nil {
		return nil
	}
	value, ok := int64Value(row[key])
	if !ok {
		return nil
	}
	return &value
}
