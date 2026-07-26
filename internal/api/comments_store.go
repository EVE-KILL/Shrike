package api

import (
	"context"
	"encoding/json"
	"net/http"
	"time"

	"github.com/eve-kill/shrike/internal/queue"
	"github.com/jackc/pgx/v5"
)

type commentInsert struct {
	TargetType  int16
	TargetID    int64
	TargetSlug  *string
	Domain      *commentDomain
	ParentID    *int64
	RootID      *int64
	Depth       int16
	BodyMD      string
	BodyHTML    string
	Principal   *Principal
	Visibility  int16
	Verdict     commentModerationResult
	QueueStatus int16
}

func (s *commentService) insertComment(
	ctx context.Context,
	input commentInsert,
) (map[string]any, int64, error) {
	tx, err := s.db.Begin(ctx)
	if err != nil {
		return nil, 0, err
	}
	defer func() { _ = tx.Rollback(ctx) }()

	var domainID any
	if input.Domain != nil {
		domainID = input.Domain.ID
	}
	corporationID := int64(0)
	corporationName := ""
	if input.Principal.CorporationID != nil {
		corporationID = int64(*input.Principal.CorporationID)
	}
	if input.Principal.CorporationName != nil {
		corporationName = *input.Principal.CorporationName
	}
	var allianceID any
	var allianceName any
	if input.Principal.AllianceID != nil {
		allianceID = int64(*input.Principal.AllianceID)
	}
	if input.Principal.AllianceName != nil {
		allianceName = *input.Principal.AllianceName
	}
	row, err := queryMap(ctx, txDatabase{Tx: tx}, `
		INSERT INTO comments (
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
			moderation_status,
			visibility
		)
		VALUES (
			$1, $2, $3, $4, $5, $6, $7, $8, $9,
			$10, $11, $12, $13, $14, $15, $16, $17
		)
		RETURNING `+commentColumns,
		input.TargetType,
		input.TargetID,
		input.TargetSlug,
		domainID,
		input.ParentID,
		input.RootID,
		input.Depth,
		input.BodyMD,
		input.BodyHTML,
		input.Principal.CharacterID,
		input.Principal.CharacterName,
		corporationID,
		corporationName,
		allianceID,
		allianceName,
		commentModerationForVerdict(input.Verdict.Action),
		input.Visibility,
	)
	if err != nil {
		return nil, 0, err
	}
	queueID, err := insertCommentModerationQueue(
		ctx,
		tx,
		row,
		input.BodyMD,
		input.BodyHTML,
		input.Principal,
		input.Verdict,
		input.QueueStatus,
	)
	if err != nil {
		return nil, 0, err
	}
	if err := tx.Commit(ctx); err != nil {
		return nil, 0, err
	}
	return row, queueID, nil
}

func (s *commentService) updateCommentBody(
	ctx context.Context,
	current map[string]any,
	principal *Principal,
	bodyMD string,
	bodyHTML string,
	visibility int16,
	verdict commentModerationResult,
	queueStatus int16,
	now time.Time,
) (map[string]any, int64, error) {
	tx, err := s.db.Begin(ctx)
	if err != nil {
		return nil, 0, err
	}
	defer func() { _ = tx.Rollback(ctx) }()
	id := mapInt64(current, "id", 0)
	if _, err := tx.Exec(ctx, `
		INSERT INTO comment_edits (
			comment_id,
			body_md,
			edited_by
		)
		VALUES ($1, $2, $3)`,
		id,
		mapString(current, "body_md", ""),
		principal.CharacterID,
	); err != nil {
		return nil, 0, err
	}
	row, err := queryMap(ctx, txDatabase{Tx: tx}, `
		UPDATE comments
		SET body_md = $2,
			body_html = $3,
			edited_at = $4,
			updated_at = $4,
			moderation_status = $5,
			visibility = $6
		WHERE id = $1
		  AND deleted_at IS NULL
		RETURNING `+commentColumns,
		id,
		bodyMD,
		bodyHTML,
		now,
		commentModerationForVerdict(verdict.Action),
		visibility,
	)
	if err != nil {
		return nil, 0, err
	}
	if row == nil {
		return nil, 0, apiError(
			http.StatusConflict,
			"Comment changed while it was being edited",
		)
	}
	queueID, err := insertCommentModerationQueue(
		ctx,
		tx,
		row,
		bodyMD,
		bodyHTML,
		principal,
		verdict,
		queueStatus,
	)
	if err != nil {
		return nil, 0, err
	}
	if err := tx.Commit(ctx); err != nil {
		return nil, 0, err
	}
	return row, queueID, nil
}

func insertCommentModerationQueue(
	ctx context.Context,
	tx pgx.Tx,
	comment map[string]any,
	bodyMD string,
	bodyHTML string,
	principal *Principal,
	verdict commentModerationResult,
	status int16,
) (int64, error) {
	scores, err := json.Marshal(verdict.Scores)
	if err != nil {
		return 0, err
	}
	var corporationID any
	var corporationName any
	var allianceID any
	var allianceName any
	if principal.CorporationID != nil {
		corporationID = *principal.CorporationID
	}
	if principal.CorporationName != nil {
		corporationName = *principal.CorporationName
	}
	if principal.AllianceID != nil {
		allianceID = *principal.AllianceID
	}
	if principal.AllianceName != nil {
		allianceName = *principal.AllianceName
	}
	var id int64
	err = tx.QueryRow(ctx, `
		INSERT INTO moderation_queue (
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
			status
		)
		VALUES (
			$1, $2, $3, NULL, $4, $5, $6, $7, $8,
			$9, $10, $11, $12, $13, $14::jsonb, $15, $16
		)
		RETURNING id`,
		moderationKindComment,
		mapInt64(comment, "id", 0),
		bodyMD,
		bodyHTML,
		principal.CharacterID,
		principal.CharacterName,
		corporationID,
		corporationName,
		allianceID,
		allianceName,
		verdict.Action,
		verdict.Category,
		verdict.MaxScore,
		string(scores),
		verdict.Source,
		status,
	).Scan(&id)
	return id, err
}

func (s *commentService) insertCommentReport(
	ctx context.Context,
	comment map[string]any,
	reporter *Principal,
	reason string,
	message *string,
	now time.Time,
) (map[string]any, error) {
	tx, err := s.db.Begin(ctx)
	if err != nil {
		return nil, err
	}
	defer func() { _ = tx.Rollback(ctx) }()
	commentID := mapInt64(comment, "id", 0)
	// Serialize reports for one comment. Under Postgres READ COMMITTED, a
	// transaction alone does not make two concurrently inserted reports
	// visible to each other's recount query.
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
	if _, err := tx.Exec(ctx, `
		INSERT INTO comment_reports (
			comment_id,
			reporter_id,
			reporter_name,
			reason,
			message,
			created_at
		)
		VALUES ($1, $2, $3, $4, $5, $6)`,
		commentID,
		reporter.CharacterID,
		reporter.CharacterName,
		reason,
		message,
		now,
	); err != nil {
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
	fresh, err := queryMap(ctx, txDatabase{Tx: tx}, `
		UPDATE comments
		SET reports_count = $2,
			flagged = ($2 >= $3)
		WHERE id = $1
		RETURNING `+commentColumns,
		commentID,
		count,
		commentReportFlagAt,
	)
	if err != nil {
		return nil, err
	}
	if err := tx.Commit(ctx); err != nil {
		return nil, err
	}
	return fresh, nil
}

func commentModerationForVerdict(
	action commentModerationAction,
) int16 {
	switch action {
	case commentActionFlag:
		return commentModerationPending
	case commentActionDelete:
		return commentModerationRejectedAI
	default:
		return commentModerationOK
	}
}

type discordContentEventArgs map[string]any

func (discordContentEventArgs) Kind() string { return "discord_events" }

type riverCommentEventDispatcher struct {
	client *queue.Client
}

func (d *riverCommentEventDispatcher) Comment(
	ctx context.Context,
	event string,
	row map[string]any,
	ancestors []commentAncestor,
) {
	if d == nil || d.client == nil || row == nil {
		return
	}
	payload := buildCommentEventPayload(event, row, ancestors)
	raw, err := json.Marshal(normalizeJSON(payload))
	if err != nil {
		return
	}
	_, _ = queue.Dispatch(
		ctx,
		d.client,
		queue.CommentEventArgs{Payload: raw},
		queue.Live,
	)
}

func buildCommentEventPayload(
	event string,
	row map[string]any,
	ancestors []commentAncestor,
) map[string]any {
	if ancestors == nil {
		ancestors = []commentAncestor{}
	}
	if event != "new" || row["parent_id"] == nil {
		ancestors = []commentAncestor{}
	}
	var parentCharacterID any
	var parentCharacterName any
	if len(ancestors) != 0 {
		parent := ancestors[len(ancestors)-1]
		parentCharacterID = parent.CharacterID
		parentCharacterName = parent.CharacterName
	}
	var comment any
	if event != "deleted" {
		minimal := map[string]any{}
		for _, key := range []string{
			"id", "target_type", "target_id", "target_slug",
			"domain_id", "parent_id", "root_id", "depth",
			"body_html", "character_id", "character_name",
			"corporation_id", "corporation_name", "alliance_id",
			"alliance_name", "created_at", "edited_at",
		} {
			minimal[key] = row[key]
		}
		comment = minimal
	}
	payload := map[string]any{
		"event_type":            event,
		"comment_id":            row["id"],
		"target_type":           row["target_type"],
		"target_id":             row["target_id"],
		"target_slug":           row["target_slug"],
		"domain_id":             row["domain_id"],
		"parent_id":             row["parent_id"],
		"parent_character_id":   parentCharacterID,
		"parent_character_name": parentCharacterName,
		"ancestors":             ancestors,
	}
	if event != "deleted" {
		payload["comment"] = comment
	}
	return payload
}

func (d *riverCommentEventDispatcher) ModerationPending(
	ctx context.Context,
	queueID int64,
	row map[string]any,
	verdict commentModerationResult,
) {
	if d == nil || d.client == nil || row == nil {
		return
	}
	event := discordContentEventArgs{
		"type":        "moderation.comment",
		"queueItemId": queueID,
		"commentId":   row["id"],
		"character": map[string]any{
			"id":               row["character_id"],
			"name":             row["character_name"],
			"corporation_id":   row["corporation_id"],
			"corporation_name": row["corporation_name"],
			"alliance_id":      row["alliance_id"],
			"alliance_name":    row["alliance_name"],
		},
		"bodySnippet": truncateRunes(
			mapString(row, "body_md", ""),
			1800,
		),
		"aiAction":   verdict.Action,
		"aiCategory": verdict.Category,
		"aiMaxScore": verdict.MaxScore,
	}
	_, _ = queue.Dispatch(ctx, d.client, event, queue.Live)
}

func (d *riverCommentEventDispatcher) Reported(
	ctx context.Context,
	row map[string]any,
	reporter *Principal,
	reason string,
	message *string,
) {
	if d == nil || d.client == nil || row == nil || reporter == nil {
		return
	}
	var reporterCorporationID any
	var reporterCorporationName any
	var reporterAllianceID any
	var reporterAllianceName any
	if reporter.CorporationID != nil {
		reporterCorporationID = *reporter.CorporationID
	}
	if reporter.CorporationName != nil {
		reporterCorporationName = *reporter.CorporationName
	}
	if reporter.AllianceID != nil {
		reporterAllianceID = *reporter.AllianceID
	}
	if reporter.AllianceName != nil {
		reporterAllianceName = *reporter.AllianceName
	}
	var snippet any
	if html := mapString(row, "body_html", ""); html != "" {
		snippet = truncateRunes(html, 400)
	}
	event := discordContentEventArgs{
		"type":      "moderation.comment-reported",
		"commentId": row["id"],
		"commentAuthor": map[string]any{
			"id":               row["character_id"],
			"name":             row["character_name"],
			"corporation_id":   row["corporation_id"],
			"corporation_name": row["corporation_name"],
			"alliance_id":      row["alliance_id"],
			"alliance_name":    row["alliance_name"],
		},
		"reporter": map[string]any{
			"id":               reporter.CharacterID,
			"name":             reporter.CharacterName,
			"corporation_id":   reporterCorporationID,
			"corporation_name": reporterCorporationName,
			"alliance_id":      reporterAllianceID,
			"alliance_name":    reporterAllianceName,
		},
		"reason":       reason,
		"message":      message,
		"reportsCount": row["reports_count"],
		"flagged":      row["flagged"],
		"bodySnippet":  snippet,
	}
	_, _ = queue.Dispatch(ctx, d.client, event, queue.Live)
}

func mapInt64(
	row map[string]any,
	key string,
	fallback int64,
) int64 {
	if row == nil {
		return fallback
	}
	value, ok := int64Value(row[key])
	if !ok {
		return fallback
	}
	return value
}
