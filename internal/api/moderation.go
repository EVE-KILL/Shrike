package api

import (
	"context"
	"encoding/json"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/danielgtaylor/huma/v2"
)

const moderationQueueColumns = `
	id,
	target_kind,
	target_id,
	body,
	body_format,
	rendered_html,
	character_id,
	character_name,
	corporation_id,
	corporation_name,
	alliance_id,
	alliance_name,
	ai_action,
	ai_category,
	ai_max_score,
	ai_scores,
	ai_source,
	status,
	submitted_at,
	reviewed_at,
	reviewed_by,
	review_notes`

const commentReportColumns = `
	id,
	comment_id,
	reporter_id,
	reporter_name,
	reason,
	message,
	created_at,
	resolved_at,
	resolved_by,
	resolution`

type moderationService struct {
	comments *commentService
}

func newModerationService(opts Options) *moderationService {
	return &moderationService{comments: newCommentService(opts)}
}

// registerModerationRoutes installs the consolidated administration contract
// and retains only the paths called by the current Nuxt application.
func registerModerationRoutes(a huma.API, opts Options) {
	registerModerationServiceRoutes(a, opts, newModerationService(opts))
}

func registerModerationServiceRoutes(
	a huma.API,
	opts Options,
	service *moderationService,
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

	commentList := service.commentQueueHandler()
	registerLegacy(a, huma.Operation{
		OperationID: "admin-comments",
		Method:      http.MethodGet,
		Path:        "/admin/comments",
		Summary:     "Reported and flagged comments",
		Tags:        []string{"admin", "comments"},
		Security:    required,
	}, commentList)
	registerLegacy(a, huma.Operation{
		OperationID: "admin-comments-live-queue-alias",
		Method:      http.MethodGet,
		Path:        "/admin/comments/queue",
		Summary:     "Reported and flagged comments (live frontend alias)",
		Tags:        []string{"admin", "comments"},
		Security:    required,
	}, commentList)
	registerLegacy(a, documentJSONBody[moderationActionBody](a, huma.Operation{
		OperationID: "admin-comment-moderation",
		Method:      http.MethodPatch,
		Path:        "/admin/comments/{id}",
		Summary:     "Hide or restore a comment",
		Tags:        []string{"admin", "comments"},
		Security:    required,
	}), service.commentVisibilityHandler(""))
	registerLegacy(a, huma.Operation{
		OperationID: "admin-comment-hide-live-alias",
		Method:      http.MethodPost,
		Path:        "/admin/comments/{id}/hide",
		Summary:     "Hide a comment (live frontend alias)",
		Tags:        []string{"admin", "comments"},
		Security:    required,
	}, service.commentVisibilityHandler("hide"))
	registerLegacy(a, huma.Operation{
		OperationID: "admin-comment-restore-live-alias",
		Method:      http.MethodPost,
		Path:        "/admin/comments/{id}/restore",
		Summary:     "Restore a comment (live frontend alias)",
		Tags:        []string{"admin", "comments"},
		Security:    required,
	}, service.commentVisibilityHandler("restore"))
	registerLegacy(a, documentJSONBody[moderationResolutionBody](a, huma.Operation{
		OperationID: "admin-comment-report-resolution",
		Method:      http.MethodPatch,
		Path:        "/admin/comment-reports/{id}",
		Summary:     "Resolve a comment report",
		Tags:        []string{"admin", "comments"},
		Security:    required,
	}), service.reportResolutionHandler())
	registerLegacy(a, huma.Operation{
		OperationID: "admin-comment-report-resolution-live-alias",
		Method:      http.MethodPost,
		Path:        "/admin/comments/reports/{id}/resolve",
		Summary:     "Resolve a comment report (live frontend alias)",
		Tags:        []string{"admin", "comments"},
		Security:    required,
	}, service.reportResolutionHandler())

	queueList := service.queueHandler()
	registerLegacy(a, huma.Operation{
		OperationID: "admin-moderation",
		Method:      http.MethodGet,
		Path:        "/admin/moderation",
		Summary:     "Unified moderation queue",
		Tags:        []string{"admin", "moderation"},
		Security:    required,
	}, queueList)
	registerLegacy(a, huma.Operation{
		OperationID: "admin-moderation-live-queue-alias",
		Method:      http.MethodGet,
		Path:        "/admin/moderation/queue",
		Summary:     "Unified moderation queue (live frontend alias)",
		Tags:        []string{"admin", "moderation"},
		Security:    required,
	}, queueList)
	registerLegacy(a, documentJSONBody[moderationDecisionBody](a, huma.Operation{
		OperationID: "admin-moderation-review",
		Method:      http.MethodPatch,
		Path:        "/admin/moderation/{id}",
		Summary:     "Approve or reject a moderation item",
		Tags:        []string{"admin", "moderation"},
		Security:    required,
	}), service.reviewHandler("", false))
	registerLegacy(a, huma.Operation{
		OperationID: "admin-moderation-approve-live-alias",
		Method:      http.MethodPost,
		Path:        "/admin/moderation/{id}/approve",
		Summary:     "Approve a moderation item (live frontend alias)",
		Tags:        []string{"admin", "moderation"},
		Security:    required,
	}, service.reviewHandler("approve", false))
	registerLegacy(a, huma.Operation{
		OperationID: "admin-moderation-reject-live-alias",
		Method:      http.MethodPost,
		Path:        "/admin/moderation/{id}/reject",
		Summary:     "Reject a moderation item (live frontend alias)",
		Tags:        []string{"admin", "moderation"},
		Security:    required,
	}, service.reviewHandler("reject", true))
}

func (s *moderationService) authorize(
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
	if err := s.comments.requireDB(); err != nil {
		return nil, err
	}
	principal, err := s.comments.auth.requirePrincipal(ctx, req)
	if err != nil {
		return nil, err
	}
	if !principal.IsAdmin {
		return nil, apiError(
			http.StatusForbidden,
			"Admin access required",
		)
	}
	return principal, nil
}

func (s *moderationService) commentQueueHandler() legacyHandler {
	return func(ctx context.Context, req *legacyRequest) (legacyPayload, error) {
		if _, err := s.authorize(ctx, req, false); err != nil {
			return legacyPayload{}, err
		}
		limit := boundedContentInt(req.Query.Get("limit"), 50, 1, 200)
		filter := req.Query.Get("filter")
		if filter == "" {
			filter = "flagged"
		}
		filters := []string{"deleted_at IS NULL"}
		switch filter {
		case "flagged":
			filters = append(filters, "flagged IS TRUE")
		case "reported":
			filters = append(filters, "reports_count > 0")
		case "all":
		default:
			// The TypeScript route treats unknown values like "all".
		}
		rows, err := queryMaps(ctx, s.comments.db, `
			SELECT `+commentColumns+`
			FROM comments
			WHERE `+strings.Join(filters, " AND ")+`
			ORDER BY reports_count DESC, id DESC
			LIMIT $1`,
			limit,
		)
		if err != nil {
			return legacyPayload{}, err
		}
		return accountNoStorePayload(
			map[string]any{"comments": rows},
		), nil
	}
}

type moderationActionBody struct {
	Action json.RawMessage `json:"action,omitempty" doc:"Moderation action to apply."`
}

type moderationResolutionBody struct {
	Resolution json.RawMessage `json:"resolution,omitempty" doc:"How the report was resolved."`
}

type moderationDecisionBody struct {
	Decision json.RawMessage `json:"decision,omitempty" doc:"Moderator decision for the queued item."`
	Notes    json.RawMessage `json:"notes,omitempty" doc:"Operator note, at most 1000 characters."`
}

func (s *moderationService) commentVisibilityHandler(
	forcedAction string,
) legacyHandler {
	return func(ctx context.Context, req *legacyRequest) (legacyPayload, error) {
		if _, err := s.authorize(ctx, req, true); err != nil {
			return legacyPayload{}, err
		}
		id, err := commentID(req.Param("id"), "Invalid comment id")
		if err != nil {
			return legacyPayload{}, err
		}
		action := forcedAction
		if action == "" {
			body, bodyErr := decodeJSONBody[moderationActionBody](req, contentBodyLimit)
			if bodyErr != nil {
				return legacyPayload{}, bodyErr
			}
			action = strings.ToLower(
				strings.TrimSpace(stringField(rawJSONValue(body.Action))),
			)
			if action == "hidden" {
				action = "hide"
			}
			if action == "published" {
				action = "restore"
			}
		}
		var query string
		switch action {
		case "hide":
			// visibility is the field every public read actually checks. The
			// old handler changed only moderation_status and therefore did
			// not hide anything.
			query = `
				UPDATE comments
				SET moderation_status = 3,
					visibility = 3,
					updated_at = $2
				WHERE id = $1
				RETURNING ` + commentColumns
		case "restore":
			query = `
				UPDATE comments
				SET moderation_status = 0,
					visibility = 0,
					deleted_at = NULL,
					deleted_by = NULL,
					flagged = FALSE,
					updated_at = $2
				WHERE id = $1
				RETURNING ` + commentColumns
		default:
			return legacyPayload{}, apiError(
				http.StatusBadRequest,
				"action must be hide or restore",
			)
		}
		row, err := queryMap(
			ctx,
			s.comments.db,
			query,
			id,
			s.comments.now().UTC(),
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
		event := "edited"
		if action == "hide" {
			// An edited event replaces the comment in connected clients and
			// therefore leaves an administratively hidden comment visible
			// until the next page load. Deleted is the wire-level removal
			// event; restoring can use edited to put the row back.
			event = "deleted"
		}
		s.comments.dispatchComment(ctx, event, row, nil)
		return accountNoStorePayload(map[string]any{
			"ok": true, "comment": row,
		}), nil
	}
}

func (s *moderationService) reportResolutionHandler() legacyHandler {
	valid := map[string]bool{
		"dismissed": true, "deleted": true, "warned": true,
	}
	return func(ctx context.Context, req *legacyRequest) (legacyPayload, error) {
		principal, err := s.authorize(ctx, req, true)
		if err != nil {
			return legacyPayload{}, err
		}
		id, err := commentID(req.Param("id"), "Invalid report id")
		if err != nil {
			return legacyPayload{}, err
		}
		body, err := decodeJSONBody[moderationResolutionBody](req, contentBodyLimit)
		if err != nil {
			return legacyPayload{}, err
		}
		resolution := strings.ToLower(
			strings.TrimSpace(stringField(rawJSONValue(body.Resolution))),
		)
		if !valid[resolution] {
			return legacyPayload{}, apiError(
				http.StatusBadRequest,
				"resolution must be one of: dismissed, deleted, warned",
			)
		}
		row, err := s.resolveReport(
			ctx,
			id,
			principal.CharacterID,
			resolution,
			s.comments.now().UTC(),
		)
		if err != nil {
			return legacyPayload{}, err
		}
		if row == nil {
			return legacyPayload{}, apiError(
				http.StatusNotFound,
				"Report not found",
			)
		}
		return accountNoStorePayload(map[string]any{
			"ok": true, "report": row,
		}), nil
	}
}

func (s *moderationService) resolveReport(
	ctx context.Context,
	reportID int64,
	reviewerID int32,
	resolution string,
	now time.Time,
) (map[string]any, error) {
	tx, err := s.comments.db.Begin(ctx)
	if err != nil {
		return nil, err
	}
	defer func() { _ = tx.Rollback(ctx) }()
	report, err := queryMap(ctx, txDatabase{Tx: tx}, `
		UPDATE comment_reports
		SET resolved_at = $2,
			resolved_by = $3,
			resolution = $4
		WHERE id = $1
		RETURNING `+commentReportColumns,
		reportID,
		now,
		reviewerID,
		resolution,
	)
	if err != nil || report == nil {
		return report, err
	}
	commentID := mapInt64(report, "comment_id", 0)
	var lockedID int64
	if err := tx.QueryRow(ctx, `
		SELECT id
		FROM comments
		WHERE id = $1
		FOR UPDATE`,
		commentID,
	).Scan(&lockedID); err != nil {
		return nil, err
	}
	var count int32
	if err := tx.QueryRow(ctx, `
		SELECT COUNT(*)::int
		FROM comment_reports
		WHERE comment_id = $1
		  AND resolved_at IS NULL`,
		commentID,
	).Scan(&count); err != nil {
		return nil, err
	}
	// reports_count is documented as unresolved reports. Keeping it in sync
	// here fixes the old path where a resolved report remained permanently
	// counted and the comment could stay flagged forever.
	if _, err := tx.Exec(ctx, `
		UPDATE comments
		SET reports_count = $2,
			flagged = ($2 >= $3)
		WHERE id = $1`,
		commentID,
		count,
		commentReportFlagAt,
	); err != nil {
		return nil, err
	}
	if err := tx.Commit(ctx); err != nil {
		return nil, err
	}
	return report, nil
}

func (s *moderationService) queueHandler() legacyHandler {
	kindGroups := map[string][]int16{
		"comments":        {moderationKindComment},
		"bios":            {bioKindCharacter, bioKindCorporation, bioKindAlliance},
		"bio_character":   {bioKindCharacter},
		"bio_corporation": {bioKindCorporation},
		"bio_alliance":    {bioKindAlliance},
	}
	statuses := map[string]int16{
		"pending":       moderationStatusPending,
		"auto_approved": moderationStatusAutoApprove,
		"auto_rejected": moderationStatusAutoReject,
		"approved":      moderationStatusApproved,
		"rejected":      moderationStatusRejected,
	}
	return func(ctx context.Context, req *legacyRequest) (legacyPayload, error) {
		if _, err := s.authorize(ctx, req, false); err != nil {
			return legacyPayload{}, err
		}
		kindKey := req.Query.Get("kind")
		if kindKey == "" {
			kindKey = "all"
		}
		statusKey := req.Query.Get("status")
		if statusKey == "" {
			statusKey = "pending"
		}
		var kinds []int16
		if kindKey != "all" {
			var found bool
			kinds, found = kindGroups[kindKey]
			if !found {
				return legacyPayload{}, apiError(
					http.StatusBadRequest,
					"invalid kind: "+kindKey,
				)
			}
		}
		var status *int16
		if statusKey != "all" {
			value, found := statuses[statusKey]
			if !found {
				return legacyPayload{}, apiError(
					http.StatusBadRequest,
					"invalid status: "+statusKey,
				)
			}
			status = &value
		}
		limit := boundedContentInt(req.Query.Get("limit"), 50, 1, 200)
		cursor := looseCommentInteger(req.Query.Get("cursor"))
		args := []any{}
		filters := []string{}
		add := func(value any) string {
			args = append(args, value)
			return "$" + strconv.Itoa(len(args))
		}
		if len(kinds) != 0 {
			filters = append(
				filters,
				"target_kind = ANY("+add(kinds)+"::smallint[])",
			)
		}
		if status != nil {
			filters = append(
				filters,
				"status = "+add(*status),
			)
		}
		if cursor != nil && *cursor != 0 {
			filters = append(
				filters,
				"id < "+add(*cursor),
			)
		}
		where := ""
		if len(filters) != 0 {
			where = "WHERE " + strings.Join(filters, " AND ")
		}
		args = append(args, limit+1)
		rows, err := queryMaps(ctx, s.comments.db, `
			SELECT `+moderationQueueColumns+`
			FROM moderation_queue
			`+where+`
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
		if err := s.enrichModerationComments(ctx, rows); err != nil {
			return legacyPayload{}, err
		}
		counts, err := queryMap(ctx, s.comments.db, `
			SELECT
				COUNT(*) FILTER (WHERE status = 0)::int
					AS pending_count,
				COUNT(*) FILTER (
					WHERE status = 0 AND target_kind = 0
				)::int AS pending_comments,
				COUNT(*) FILTER (
					WHERE status = 0 AND target_kind IN (1, 2, 3)
				)::int AS pending_bios,
				COUNT(*)::int AS total_count
			FROM moderation_queue`,
		)
		if err != nil {
			return legacyPayload{}, err
		}
		var next any
		if hasMore && len(rows) != 0 {
			next = rows[len(rows)-1]["id"]
		}
		return accountNoStorePayload(map[string]any{
			"items":      rows,
			"nextCursor": next,
			"counts": map[string]any{
				"pending":          mapInt64(counts, "pending_count", 0),
				"pending_comments": mapInt64(counts, "pending_comments", 0),
				"pending_bios":     mapInt64(counts, "pending_bios", 0),
				"total":            mapInt64(counts, "total_count", 0),
			},
		}), nil
	}
}

func (s *moderationService) enrichModerationComments(
	ctx context.Context,
	rows []map[string]any,
) error {
	ids := []int64{}
	for _, row := range rows {
		if mapInt64(row, "target_kind", -1) == moderationKindComment {
			ids = append(ids, mapInt64(row, "target_id", 0))
		}
	}
	contexts := map[int64]map[string]any{}
	if len(ids) != 0 {
		contextRows, err := queryMaps(ctx, s.comments.db, `
			SELECT id, target_type, target_id, target_slug
			FROM comments
			WHERE id = ANY($1::bigint[])`,
			ids,
		)
		if err != nil {
			return err
		}
		for _, row := range contextRows {
			contexts[mapInt64(row, "id", 0)] = map[string]any{
				"target_type": row["target_type"],
				"target_id":   row["target_id"],
				"target_slug": row["target_slug"],
			}
		}
	}
	for _, row := range rows {
		row["comment_context"] = nil
		if mapInt64(row, "target_kind", -1) ==
			moderationKindComment {
			if value := contexts[mapInt64(row, "target_id", 0)]; value != nil {
				row["comment_context"] = value
			}
		}
	}
	return nil
}

func (s *moderationService) reviewHandler(
	forcedDecision string,
	legacyRejectNotFound bool,
) legacyHandler {
	return func(ctx context.Context, req *legacyRequest) (legacyPayload, error) {
		principal, err := s.authorize(ctx, req, true)
		if err != nil {
			return legacyPayload{}, err
		}
		id, err := commentID(req.Param("id"), "Invalid queue id")
		if err != nil {
			return legacyPayload{}, err
		}
		body, err := decodeJSONBody[moderationDecisionBody](req, contentBodyLimit)
		if err != nil {
			if forcedDecision == "" {
				return legacyPayload{}, err
			}
			// The live approve/reject handlers intentionally treated an
			// unreadable optional body as {}.
			body = &moderationDecisionBody{}
		}
		decision := forcedDecision
		if decision == "" {
			decision = strings.ToLower(
				strings.TrimSpace(stringField(rawJSONValue(body.Decision))),
			)
		}
		if decision != "approve" && decision != "reject" {
			return legacyPayload{}, apiError(
				http.StatusBadRequest,
				"decision must be approve or reject",
			)
		}
		notes := optionalTrimmedString(rawJSONValue(body.Notes), 1000)
		queueRow, comment, state, err := s.review(
			ctx,
			id,
			principal.CharacterID,
			decision,
			notes,
			s.comments.now().UTC(),
		)
		if err != nil {
			return legacyPayload{}, err
		}
		if state == reviewMissing {
			return legacyPayload{}, apiError(
				http.StatusNotFound,
				"Queue row not found",
			)
		}
		if state == reviewAlreadyHandled {
			if legacyRejectNotFound {
				return legacyPayload{}, apiError(
					http.StatusNotFound,
					"Queue row not found or already reviewed",
				)
			}
			return legacyPayload{}, apiError(
				http.StatusConflict,
				"Queue row is not pending — already reviewed",
			)
		}
		if decision == "approve" && comment != nil {
			ancestors := []commentAncestor{}
			if parentID := mapOptionalInt64(comment, "parent_id"); parentID != nil {
				ancestors, err = s.comments.commentAncestry(
					ctx,
					*parentID,
				)
				if err != nil {
					return legacyPayload{}, err
				}
			}
			s.comments.dispatchComment(
				ctx,
				"new",
				comment,
				ancestors,
			)
		}
		_ = queueRow
		status := "approved"
		if decision == "reject" {
			status = "rejected"
		}
		return accountNoStorePayload(map[string]any{
			"ok": true, "id": id, "status": status,
		}), nil
	}
}

type moderationReviewState int

const (
	reviewApplied moderationReviewState = iota
	reviewMissing
	reviewAlreadyHandled
)

func (s *moderationService) review(
	ctx context.Context,
	id int64,
	reviewerID int32,
	decision string,
	notes *string,
	now time.Time,
) (
	map[string]any,
	map[string]any,
	moderationReviewState,
	error,
) {
	tx, err := s.comments.db.Begin(ctx)
	if err != nil {
		return nil, nil, reviewMissing, err
	}
	defer func() { _ = tx.Rollback(ctx) }()
	queueRow, err := queryMap(ctx, txDatabase{Tx: tx}, `
		SELECT `+moderationQueueColumns+`
		FROM moderation_queue
		WHERE id = $1
		FOR UPDATE`,
		id,
	)
	if err != nil {
		return nil, nil, reviewMissing, err
	}
	if queueRow == nil {
		return nil, nil, reviewMissing, nil
	}
	if mapInt64(queueRow, "status", -1) !=
		moderationStatusPending {
		return queueRow, nil, reviewAlreadyHandled, nil
	}
	kind := mapInt64(queueRow, "target_kind", -1)
	targetID := mapInt64(queueRow, "target_id", 0)
	var comment map[string]any
	if decision == "approve" {
		switch kind {
		case moderationKindComment:
			comment, err = queryMap(ctx, txDatabase{Tx: tx}, `
				UPDATE comments
				SET visibility = 0,
					moderation_status = 0,
					updated_at = $2
				WHERE id = $1
				RETURNING `+commentColumns,
				targetID,
				now,
			)
		case int64(bioKindCharacter):
			_, err = tx.Exec(ctx, `
				UPDATE characters
				SET custom_description = $2,
					custom_description_format =
						COALESCE($3, 'markdown')
				WHERE character_id = $1`,
				targetID,
				queueRow["body"],
				queueRow["body_format"],
			)
		case int64(bioKindCorporation):
			_, err = tx.Exec(ctx, `
				UPDATE corporations
				SET custom_description = $2,
					custom_description_format =
						COALESCE($3, 'markdown')
				WHERE corporation_id = $1`,
				targetID,
				queueRow["body"],
				queueRow["body_format"],
			)
		case int64(bioKindAlliance):
			_, err = tx.Exec(ctx, `
				UPDATE alliances
				SET custom_description = $2,
					custom_description_format =
						COALESCE($3, 'markdown')
				WHERE alliance_id = $1`,
				targetID,
				queueRow["body"],
				queueRow["body_format"],
			)
		}
	} else if kind == moderationKindComment {
		comment, err = queryMap(ctx, txDatabase{Tx: tx}, `
			UPDATE comments
			SET visibility = 2,
				moderation_status = 2,
				updated_at = $2
			WHERE id = $1
			RETURNING `+commentColumns,
			targetID,
			now,
		)
	}
	if err != nil {
		return nil, nil, reviewMissing, err
	}
	status := moderationStatusApproved
	if decision == "reject" {
		status = moderationStatusRejected
	}
	queueRow, err = queryMap(ctx, txDatabase{Tx: tx}, `
		UPDATE moderation_queue
		SET status = $2,
			reviewed_at = $3,
			reviewed_by = $4,
			review_notes = $5
		WHERE id = $1
		RETURNING `+moderationQueueColumns,
		id,
		status,
		now,
		reviewerID,
		notes,
	)
	if err != nil {
		return nil, nil, reviewMissing, err
	}
	if err := tx.Commit(ctx); err != nil {
		return nil, nil, reviewMissing, err
	}
	return queueRow, comment, reviewApplied, nil
}
