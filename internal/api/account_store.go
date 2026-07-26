package api

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net"
	"sort"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
)

type postgresAccountStore struct {
	db MutationDatabase
}

func (s *postgresAccountStore) LoadPreferences(
	ctx context.Context,
	characterID int32,
) (map[string]any, error) {
	rows, err := s.db.Query(ctx, `
		SELECT key, value
		FROM user_config
		WHERE character_id = $1
		ORDER BY key`,
		characterID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	result := make(map[string]any)
	for rows.Next() {
		var key string
		var raw []byte
		if err := rows.Scan(&key, &raw); err != nil {
			return nil, err
		}
		var value any
		if err := json.Unmarshal(raw, &value); err != nil {
			return nil, fmt.Errorf("decode user preference %q: %w", key, err)
		}
		result[key] = value
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return result, nil
}

func (s *postgresAccountStore) SavePreferences(
	ctx context.Context,
	characterID int32,
	updates map[string]any,
	now time.Time,
) (err error) {
	tx, err := s.db.Begin(ctx)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback(ctx) }()

	keys := make([]string, 0, len(updates))
	for key := range updates {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	for _, key := range keys {
		raw, err := json.Marshal(updates[key])
		if err != nil {
			return fmt.Errorf("encode user preference %q: %w", key, err)
		}
		if _, err := tx.Exec(ctx, `
			INSERT INTO user_config (character_id, key, value, updated_at)
			VALUES ($1, $2, $3::jsonb, $4)
			ON CONFLICT (character_id, key) DO UPDATE SET
				value = excluded.value,
				updated_at = excluded.updated_at`,
			characterID, key, raw, now,
		); err != nil {
			return err
		}
	}
	return tx.Commit(ctx)
}

func (s *postgresAccountStore) LoadBoardData(
	ctx context.Context,
	principal *Principal,
) (accountBoardData, error) {
	result := accountBoardData{
		Account: accountBoardState{
			Pinned: []string{}, Dismissed: []string{},
		},
		Domains: []accountBoardDomain{},
	}
	if principal != nil {
		var raw []byte
		err := s.db.QueryRow(ctx, `
			SELECT value
			FROM user_config
			WHERE character_id = $1
			  AND key = 'boards'
			LIMIT 1`,
			principal.CharacterID,
		).Scan(&raw)
		switch {
		case errors.Is(err, pgx.ErrNoRows):
		case err != nil:
			return accountBoardData{}, err
		default:
			var value any
			if err := json.Unmarshal(raw, &value); err != nil {
				return accountBoardData{}, fmt.Errorf(
					"decode stored boards: %w", err,
				)
			}
			result.Account = sanitizeBoardState(value)
		}
	}

	rows, err := s.db.Query(ctx, `
		SELECT id, subdomain, custom_hostname, site_name, entities
		FROM custom_domains
		WHERE active IS TRUE
		ORDER BY id`)
	if err != nil {
		return accountBoardData{}, err
	}
	defer rows.Close()
	for rows.Next() {
		var domain accountBoardDomain
		var entitiesJSON []byte
		if err := rows.Scan(
			&domain.ID,
			&domain.Subdomain,
			&domain.CustomHostname,
			&domain.SiteName,
			&entitiesJSON,
		); err != nil {
			return accountBoardData{}, err
		}
		if err := json.Unmarshal(entitiesJSON, &domain.Entities); err != nil {
			// A malformed board configuration must not break the switcher for
			// every visitor. It simply cannot track an entity until repaired.
			domain.Entities = []accountBoardEntity{}
		}
		result.Domains = append(result.Domains, domain)
	}
	if err := rows.Err(); err != nil {
		return accountBoardData{}, err
	}
	return result, nil
}

func (s *postgresAccountStore) LoadOverview(
	ctx context.Context,
	characterID int32,
	now time.Time,
) (accountOverview, error) {
	var result accountOverview
	var scopeCount *int32
	err := s.db.QueryRow(ctx, `
		SELECT
			u.last_login,
			u.created_at,
			(
				SELECT count(*)
				FROM esi_request_logs l
				WHERE l.character_id = u.character_id
			),
			(
				SELECT count(*)
				FROM esi_request_logs l
				WHERE l.character_id = u.character_id
				  AND l.success IS FALSE
				  AND (l.status_code IS NULL OR l.status_code <> 304)
			),
			(
				SELECT coalesce(sum(l.new_items), 0)
				FROM esi_request_logs l
				WHERE l.character_id = u.character_id
			),
			(
				SELECT max(l.created_at)
				FROM esi_request_logs l
				WHERE l.character_id = u.character_id
			),
			(
				SELECT count(*)
				FROM esi_request_logs l
				WHERE l.character_id = u.character_id
				  AND l.created_at >= $2 - interval '24 hours'
			),
			(
				SELECT count(*)
				FROM esi_request_logs l
				WHERE l.character_id = u.character_id
				  AND l.success IS FALSE
				  AND (l.status_code IS NULL OR l.status_code <> 304)
				  AND l.created_at >= $2 - interval '24 hours'
			),
			(
				SELECT coalesce(sum(l.new_items), 0)
				FROM esi_request_logs l
				WHERE l.character_id = u.character_id
				  AND l.created_at >= $2 - interval '24 hours'
			),
			cardinality(t.scopes),
			t.token_expiry,
			t.last_fetched
		FROM users u
		LEFT JOIN user_esi_tokens t
		  ON t.character_id = u.character_id
		WHERE u.character_id = $1
		LIMIT 1`,
		characterID, now,
	).Scan(
		&result.LastLogin,
		&result.CreatedAt,
		&result.TotalRequests,
		&result.TotalErrors,
		&result.TotalNewItems,
		&result.LastRequest,
		&result.Requests24Hours,
		&result.Errors24Hours,
		&result.NewItems24Hours,
		&scopeCount,
		&result.TokenExpiry,
		&result.TokenLastFetch,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return accountOverview{}, errNoAuthSession
	}
	if err != nil {
		return accountOverview{}, err
	}
	if scopeCount != nil {
		result.TokenFound = true
		result.TokenScopeCount = int(*scopeCount)
	}
	return result, nil
}

func (s *postgresAccountStore) LoadManageableEntities(
	ctx context.Context,
	characterID int32,
) (accountManageableEntities, error) {
	var result accountManageableEntities
	var corporationID, allianceID *int32
	var corporationName, corporationTicker *string
	var corporationCEOID *int32
	var corporationCEOName *string
	var corporationESI, corporationCustom, corporationFormat *string
	var allianceName, allianceTicker *string
	var executorCorporationID, executorCEOID *int32
	var executorCEOName *string
	var allianceCustom, allianceFormat *string
	var charPendingBody, charPendingFormat *string
	var charPendingAt *time.Time
	var corpPendingBody, corpPendingFormat *string
	var corpPendingAt *time.Time
	var alliancePendingBody, alliancePendingFormat *string
	var alliancePendingAt *time.Time

	err := s.db.QueryRow(ctx, `
		SELECT
			c.character_id,
			c.name,
			c.description,
			c.custom_description,
			c.custom_description_format,
			corp.corporation_id,
			corp.name,
			corp.ticker,
			corp.ceo_id,
			corp_ceo.name,
			corp.description,
			corp.custom_description,
			corp.custom_description_format,
			ally.alliance_id,
			ally.name,
			ally.ticker,
			ally.executor_corporation_id,
			exec_corp.ceo_id,
			exec_ceo.name,
			ally.custom_description,
			ally.custom_description_format,
			pending_char.body,
			pending_char.body_format,
			pending_char.submitted_at,
			pending_corp.body,
			pending_corp.body_format,
			pending_corp.submitted_at,
			pending_ally.body,
			pending_ally.body_format,
			pending_ally.submitted_at
		FROM characters c
		LEFT JOIN corporations corp
		  ON corp.corporation_id = c.corporation_id
		LEFT JOIN characters corp_ceo
		  ON corp_ceo.character_id = corp.ceo_id
		LEFT JOIN alliances ally
		  ON ally.alliance_id = c.alliance_id
		LEFT JOIN corporations exec_corp
		  ON exec_corp.corporation_id = ally.executor_corporation_id
		LEFT JOIN characters exec_ceo
		  ON exec_ceo.character_id = exec_corp.ceo_id
		LEFT JOIN LATERAL (
			SELECT body, body_format, submitted_at
			FROM moderation_queue
			WHERE target_kind = $2
			  AND target_id = c.character_id
			  AND status = $5
			LIMIT 1
		) pending_char ON TRUE
		LEFT JOIN LATERAL (
			SELECT body, body_format, submitted_at
			FROM moderation_queue
			WHERE target_kind = $3
			  AND target_id = corp.corporation_id
			  AND status = $5
			  AND corp.ceo_id = c.character_id
			LIMIT 1
		) pending_corp ON TRUE
		LEFT JOIN LATERAL (
			SELECT body, body_format, submitted_at
			FROM moderation_queue
			WHERE target_kind = $4
			  AND target_id = ally.alliance_id
			  AND status = $5
			  AND exec_corp.ceo_id = c.character_id
			LIMIT 1
		) pending_ally ON TRUE
		WHERE c.character_id = $1
		LIMIT 1`,
		characterID,
		bioKindCharacter,
		bioKindCorporation,
		bioKindAlliance,
		bioStatusPending,
	).Scan(
		&result.Character.ID,
		&result.Character.Name,
		&result.Character.ESIDescription,
		&result.Character.CustomDescription,
		&result.Character.CustomFormat,
		&corporationID,
		&corporationName,
		&corporationTicker,
		&corporationCEOID,
		&corporationCEOName,
		&corporationESI,
		&corporationCustom,
		&corporationFormat,
		&allianceID,
		&allianceName,
		&allianceTicker,
		&executorCorporationID,
		&executorCEOID,
		&executorCEOName,
		&allianceCustom,
		&allianceFormat,
		&charPendingBody,
		&charPendingFormat,
		&charPendingAt,
		&corpPendingBody,
		&corpPendingFormat,
		&corpPendingAt,
		&alliancePendingBody,
		&alliancePendingFormat,
		&alliancePendingAt,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return accountManageableEntities{}, apiError(
			404, "Character not found",
		)
	}
	if err != nil {
		return accountManageableEntities{}, err
	}
	result.Character.Pending = pendingBio(
		charPendingBody, charPendingFormat, charPendingAt,
	)
	if corporationID != nil {
		result.Corporation = &accountCorporationBio{
			ID: *corporationID, Name: stringPointerValue(corporationName),
			Ticker: stringPointerValue(corporationTicker),
			CEOID:  corporationCEOID, CEOName: corporationCEOName,
			ESIDescription:    corporationESI,
			CustomDescription: corporationCustom,
			CustomFormat:      corporationFormat,
			Pending: pendingBio(
				corpPendingBody, corpPendingFormat, corpPendingAt,
			),
		}
	}
	if allianceID != nil {
		result.Alliance = &accountAllianceBio{
			ID: *allianceID, Name: stringPointerValue(allianceName),
			Ticker:                stringPointerValue(allianceTicker),
			ExecutorCorporationID: executorCorporationID,
			ExecutorCEOID:         executorCEOID,
			ExecutorCEOName:       executorCEOName,
			CustomDescription:     allianceCustom,
			CustomFormat:          allianceFormat,
			Pending: pendingBio(
				alliancePendingBody, alliancePendingFormat, alliancePendingAt,
			),
		}
	}
	return result, nil
}

func pendingBio(
	body *string,
	format *string,
	submittedAt *time.Time,
) *accountPendingBio {
	if body == nil || submittedAt == nil {
		return nil
	}
	return &accountPendingBio{
		Body:       *body,
		BodyFormat: formatOrMarkdown(format),
		Submitted:  *submittedAt,
	}
}

func (s *postgresAccountStore) ResolveBioTarget(
	ctx context.Context,
	characterID int32,
	entity string,
) (accountBioTarget, error) {
	var corporationID, allianceID *int32
	var corporationExists, allianceExists bool
	var corporationCEOID, executorCorporationID, executorCEOID *int32
	err := s.db.QueryRow(ctx, `
		SELECT
			c.corporation_id,
			c.alliance_id,
			(corp.corporation_id IS NOT NULL),
			corp.ceo_id,
			(ally.alliance_id IS NOT NULL),
			ally.executor_corporation_id,
			exec_corp.ceo_id
		FROM characters c
		LEFT JOIN corporations corp
		  ON corp.corporation_id = c.corporation_id
		LEFT JOIN alliances ally
		  ON ally.alliance_id = c.alliance_id
		LEFT JOIN corporations exec_corp
		  ON exec_corp.corporation_id = ally.executor_corporation_id
		WHERE c.character_id = $1
		LIMIT 1`,
		characterID,
	).Scan(
		&corporationID,
		&allianceID,
		&corporationExists,
		&corporationCEOID,
		&allianceExists,
		&executorCorporationID,
		&executorCEOID,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return accountBioTarget{}, apiError(404, "Character not found")
	}
	if err != nil {
		return accountBioTarget{}, err
	}
	switch entity {
	case "character":
		return accountBioTarget{
			Entity: entity, Kind: bioKindCharacter, ID: int64(characterID),
		}, nil
	case "corporation":
		if corporationID == nil {
			return accountBioTarget{}, apiError(
				403, "You are not in a corporation",
			)
		}
		if !corporationExists {
			return accountBioTarget{}, apiError(404, "Corporation not found")
		}
		if corporationCEOID == nil || *corporationCEOID != characterID {
			return accountBioTarget{}, apiError(
				403, "Only the CEO may edit the corporation description",
			)
		}
		return accountBioTarget{
			Entity: entity, Kind: bioKindCorporation,
			ID: int64(*corporationID),
		}, nil
	case "alliance":
		if allianceID == nil {
			return accountBioTarget{}, apiError(
				403, "You are not in an alliance",
			)
		}
		if !allianceExists {
			return accountBioTarget{}, apiError(404, "Alliance not found")
		}
		if executorCorporationID == nil {
			return accountBioTarget{}, apiError(
				403, "Alliance has no executor corporation",
			)
		}
		if executorCEOID == nil || *executorCEOID != characterID {
			return accountBioTarget{}, apiError(
				403, "Only the executor corp CEO may edit the alliance description",
			)
		}
		return accountBioTarget{
			Entity: entity, Kind: bioKindAlliance, ID: int64(*allianceID),
		}, nil
	default:
		return accountBioTarget{}, apiError(400, "Invalid entity")
	}
}

func (s *postgresAccountStore) ClearBio(
	ctx context.Context,
	target accountBioTarget,
) (err error) {
	tx, err := s.db.Begin(ctx)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback(ctx) }()

	var query string
	switch target.Kind {
	case bioKindCharacter:
		query = `UPDATE characters SET custom_description = NULL WHERE character_id = $1`
	case bioKindCorporation:
		query = `UPDATE corporations SET custom_description = NULL WHERE corporation_id = $1`
	case bioKindAlliance:
		query = `UPDATE alliances SET custom_description = NULL WHERE alliance_id = $1`
	default:
		return errors.New("unknown bio target kind")
	}
	if _, err := tx.Exec(ctx, query, target.ID); err != nil {
		return err
	}
	if _, err := tx.Exec(ctx, `
		DELETE FROM moderation_queue
		WHERE target_kind = $1
		  AND target_id = $2
		  AND status = $3`,
		target.Kind, target.ID, bioStatusPending,
	); err != nil {
		return err
	}
	return tx.Commit(ctx)
}

func (s *postgresAccountStore) EnqueueBio(
	ctx context.Context,
	submission accountBioSubmission,
) (int64, error) {
	var id int64
	err := s.db.QueryRow(ctx, `
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
			status,
			submitted_at
		) VALUES (
			$1, $2, $3, $4, $5, $6, $7,
			$8, $9, $10, $11, $12, $13
		)
		ON CONFLICT (target_kind, target_id)
			WHERE status = 0
		DO UPDATE SET
			body = excluded.body,
			body_format = excluded.body_format,
			rendered_html = excluded.rendered_html,
			character_id = excluded.character_id,
			character_name = excluded.character_name,
			corporation_id = excluded.corporation_id,
			corporation_name = excluded.corporation_name,
			alliance_id = excluded.alliance_id,
			alliance_name = excluded.alliance_name,
			submitted_at = excluded.submitted_at
		RETURNING id`,
		submission.Target.Kind,
		submission.Target.ID,
		submission.Body,
		submission.BodyFormat,
		submission.RenderedHTML,
		submission.CharacterID,
		submission.CharacterName,
		nullableInt32(submission.CorporationID),
		nullableString(submission.CorporationName),
		nullableInt32(submission.AllianceID),
		nullableString(submission.AllianceName),
		bioStatusPending,
		submission.SubmittedAt,
	).Scan(&id)
	return id, err
}

func nullableInt32(value *int32) any {
	if value == nil {
		return nil
	}
	return *value
}

func stringPointerValue(value *string) string {
	if value == nil {
		return ""
	}
	return *value
}

func (s *postgresAccountStore) LoadESIMetrics(
	ctx context.Context,
	characterID int32,
	now time.Time,
) (accountESIMetrics, error) {
	var result accountESIMetrics
	var volumeJSON []byte
	err := s.db.QueryRow(ctx, `
		WITH recent AS MATERIALIZED (
			SELECT
				created_at,
				success,
				status_code,
				new_items,
				request_duration_ms
			FROM esi_request_logs
			WHERE character_id = $1
			  AND created_at >= $2 - interval '24 hours'
		),
		hourly AS (
			SELECT
				date_trunc('hour', created_at)::text AS hour,
				count(*)::bigint AS total,
				count(*) FILTER (
					WHERE success IS FALSE
					  AND (status_code IS NULL OR status_code <> 304)
				)::bigint AS errors,
				coalesce(sum(new_items), 0)::bigint AS new_items
			FROM recent
			GROUP BY date_trunc('hour', created_at)
		)
		SELECT
			coalesce((
				SELECT jsonb_agg(
					jsonb_build_object(
						'hour', hour,
						'total', total,
						'errors', errors,
						'new_items', new_items
					)
					ORDER BY hour
				)
				FROM hourly
			), '[]'::jsonb),
			(
				SELECT count(*)::bigint
				FROM recent
				WHERE created_at >= $2 - interval '1 hour'
			),
			(
				SELECT round(avg(request_duration_ms))::int
				FROM recent
				WHERE created_at >= $2 - interval '1 hour'
				  AND request_duration_ms IS NOT NULL
			),
			(
				SELECT round(
					percentile_cont(0.95) WITHIN GROUP (
						ORDER BY request_duration_ms
					)
				)::int
				FROM recent
				WHERE created_at >= $2 - interval '1 hour'
				  AND request_duration_ms IS NOT NULL
			)`,
		characterID,
		now,
	).Scan(
		&volumeJSON,
		&result.RequestCount,
		&result.AverageMS,
		&result.P95MS,
	)
	if err != nil {
		return accountESIMetrics{}, err
	}
	if err := json.Unmarshal(volumeJSON, &result.Volume); err != nil {
		return accountESIMetrics{}, fmt.Errorf(
			"decode ESI hourly metrics: %w", err,
		)
	}
	return result, nil
}

type accountSQLFilter struct {
	clauses []string
	args    []any
}

func newAccountESILogFilter(query accountESILogQuery) accountSQLFilter {
	filter := accountSQLFilter{
		clauses: []string{"character_id = $1"},
		args:    []any{query.CharacterID},
	}
	if query.Source != "" {
		filter.add("source = %s", query.Source)
	}
	switch query.Status {
	case "success":
		filter.clauses = append(
			filter.clauses,
			"(success IS TRUE OR status_code = 304)",
		)
	case "error":
		filter.clauses = append(
			filter.clauses,
			"success IS FALSE AND (status_code IS NULL OR status_code <> 304)",
		)
	}
	switch query.Endpoint {
	case "character":
		filter.clauses = append(
			filter.clauses,
			"endpoint ILIKE '%/characters/%'",
		)
	case "corporation":
		filter.clauses = append(
			filter.clauses,
			"endpoint ILIKE '%/corporations/%'",
		)
	}
	if query.AfterID != nil {
		filter.add("id > %s", *query.AfterID)
	}
	return filter
}

func (f *accountSQLFilter) add(template string, value any) {
	f.args = append(f.args, value)
	placeholder := fmt.Sprintf("$%d", len(f.args))
	f.clauses = append(
		f.clauses,
		fmt.Sprintf(template, placeholder),
	)
}

func (f accountSQLFilter) where() string {
	return strings.Join(f.clauses, " AND ")
}

const accountESILogColumns = `
	id,
	endpoint,
	method,
	status_code,
	success,
	error_message,
	items_returned,
	new_items,
	source,
	request_duration_ms,
	created_at,
	CASE
		WHEN endpoint ILIKE '%/characters/%' THEN 'character'
		WHEN endpoint ILIKE '%/corporations/%' THEN 'corporation'
		ELSE 'other'
	END AS endpoint_type,
	regexp_replace(
		regexp_replace(
			endpoint,
			'.*/(?:characters|corporations)/[0-9]+/',
			''
		),
		'[/?].*',
		''
	) AS endpoint_action`

func (s *postgresAccountStore) LoadESILogs(
	ctx context.Context,
	query accountESILogQuery,
) (accountESILogResult, error) {
	filter := newAccountESILogFilter(query)
	args := append([]any{}, filter.args...)
	limit := query.Limit
	if query.AfterID != nil {
		limit = 200
	}
	args = append(args, limit)
	pagination := fmt.Sprintf("LIMIT $%d", len(args))
	if query.AfterID == nil {
		offset := (int64(query.Page) - 1) * int64(query.Limit)
		args = append(args, offset)
		pagination += fmt.Sprintf(" OFFSET $%d", len(args))
	}
	rows, err := s.db.Query(ctx, fmt.Sprintf(`
		SELECT %s
		FROM esi_request_logs
		WHERE %s
		ORDER BY id DESC
		%s`,
		accountESILogColumns,
		filter.where(),
		pagination,
	), args...)
	if err != nil {
		return accountESILogResult{}, err
	}
	result := accountESILogResult{
		Rows: []accountESILogRow{},
	}
	for rows.Next() {
		var row accountESILogRow
		if err := rows.Scan(
			&row.ID,
			&row.Endpoint,
			&row.Method,
			&row.StatusCode,
			&row.Success,
			&row.ErrorMessage,
			&row.ItemsReturned,
			&row.NewItems,
			&row.Source,
			&row.RequestDurationMS,
			&row.CreatedAt,
			&row.EndpointType,
			&row.EndpointAction,
		); err != nil {
			rows.Close()
			return accountESILogResult{}, err
		}
		result.Rows = append(result.Rows, row)
	}
	if err := rows.Err(); err != nil {
		rows.Close()
		return accountESILogResult{}, err
	}
	rows.Close()
	if query.AfterID != nil {
		return result, nil
	}

	baseFilter := query
	baseFilter.AfterID = nil
	filter = newAccountESILogFilter(baseFilter)
	if err := s.db.QueryRow(ctx, `
		SELECT count(*)::bigint
		FROM esi_request_logs
		WHERE `+filter.where(),
		filter.args...,
	).Scan(&result.Total); err != nil {
		return accountESILogResult{}, err
	}

	sourceRows, err := s.db.Query(ctx, `
		SELECT DISTINCT source
		FROM esi_request_logs
		WHERE character_id = $1
		ORDER BY source`,
		query.CharacterID,
	)
	if err != nil {
		return accountESILogResult{}, err
	}
	result.Sources = []string{}
	for sourceRows.Next() {
		var source string
		if err := sourceRows.Scan(&source); err != nil {
			sourceRows.Close()
			return accountESILogResult{}, err
		}
		result.Sources = append(result.Sources, source)
	}
	if err := sourceRows.Err(); err != nil {
		sourceRows.Close()
		return accountESILogResult{}, err
	}
	sourceRows.Close()
	return result, nil
}

func (s *postgresAccountStore) LoadActiveAnnouncements(
	ctx context.Context,
	now time.Time,
) ([]accountAnnouncement, error) {
	rows, err := s.db.Query(ctx, `
		SELECT
			id,
			tier,
			title,
			body_md,
			body_html,
			color,
			icon,
			link_url,
			link_label,
			starts_at,
			expires_at,
			created_by,
			created_at,
			updated_at,
			archived_at
		FROM announcements
		WHERE archived_at IS NULL
		  AND starts_at <= $1
		  AND expires_at > $1
		ORDER BY tier DESC, created_at DESC`,
		now,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	result := []accountAnnouncement{}
	for rows.Next() {
		var item accountAnnouncement
		if err := rows.Scan(
			&item.ID,
			&item.Tier,
			&item.Title,
			&item.BodyMD,
			&item.BodyHTML,
			&item.Color,
			&item.Icon,
			&item.LinkURL,
			&item.LinkLabel,
			&item.StartsAt,
			&item.ExpiresAt,
			&item.CreatedBy,
			&item.CreatedAt,
			&item.UpdatedAt,
			&item.ArchivedAt,
		); err != nil {
			return nil, err
		}
		result = append(result, item)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return result, nil
}

func (s *postgresAccountStore) LoadDismissedAnnouncementIDs(
	ctx context.Context,
	characterID int32,
) ([]int64, error) {
	rows, err := s.db.Query(ctx, `
		SELECT announcement_id
		FROM announcement_dismissals
		WHERE character_id = $1
		ORDER BY announcement_id`,
		characterID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	result := []int64{}
	for rows.Next() {
		var id int64
		if err := rows.Scan(&id); err != nil {
			return nil, err
		}
		result = append(result, id)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return result, nil
}

func (s *postgresAccountStore) DismissAnnouncement(
	ctx context.Context,
	characterID int32,
	announcementID int64,
) error {
	_, err := s.db.Exec(ctx, `
		INSERT INTO announcement_dismissals (
			character_id,
			announcement_id
		)
		VALUES ($1, $2)
		ON CONFLICT (character_id, announcement_id) DO NOTHING`,
		characterID,
		announcementID,
	)
	return err
}

func (s *postgresAccountStore) ResolveCommentDomainID(
	ctx context.Context,
	rawHost string,
) (*int32, error) {
	host := strings.ToLower(strings.TrimSpace(rawHost))
	if parsed, _, err := net.SplitHostPort(host); err == nil {
		host = parsed
	} else if strings.Count(host, ":") == 1 {
		host, _, _ = strings.Cut(host, ":")
	}
	host = strings.TrimSuffix(host, ".")
	if host == "" ||
		host == "eve-kill.com" ||
		host == "www.eve-kill.com" ||
		host == "zkillboard.co" ||
		host == "localhost" ||
		host == "127.0.0.1" ||
		net.ParseIP(host) != nil ||
		!strings.Contains(host, ".") {
		return nil, nil
	}

	var query string
	var lookup string
	switch {
	case strings.HasSuffix(host, ".eve-kill.com"):
		lookup = strings.TrimSuffix(host, ".eve-kill.com")
		if lookup == "" || strings.Contains(lookup, ".") {
			return nil, nil
		}
		query = `
			SELECT id
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
			SELECT id
			FROM custom_domains
			WHERE active IS TRUE
			  AND lower(subdomain) = $1
			LIMIT 1`
	default:
		lookup = host
		query = `
			SELECT id
			FROM custom_domains
			WHERE active IS TRUE
			  AND lower(custom_hostname) = $1
			LIMIT 1`
	}

	var id int32
	err := s.db.QueryRow(ctx, query, lookup).Scan(&id)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &id, nil
}

func (s *postgresAccountStore) LoadNotificationReplies(
	ctx context.Context,
	query accountNotificationQuery,
) ([]accountNotificationReply, error) {
	rows, err := s.db.Query(ctx, `
		SELECT
			c.id,
			c.target_type,
			c.target_id,
			c.parent_id,
			c.root_id,
			c.body_html,
			c.created_at,
			c.character_id,
			c.character_name,
			c.corporation_id,
			c.corporation_name,
			c.alliance_id,
			c.alliance_name,
			p.id,
			p.body_md
		FROM comments c
		INNER JOIN comments p
		  ON c.parent_id = p.id
		WHERE p.character_id = $1
		  AND c.character_id <> $1
		  AND c.deleted_at IS NULL
		  AND p.deleted_at IS NULL
		  AND c.visibility = 0
		  AND c.domain_id IS NOT DISTINCT FROM $2
		  AND p.domain_id IS NOT DISTINCT FROM $2
		  AND c.id > $3
		ORDER BY c.id DESC
		LIMIT $4`,
		query.CharacterID,
		nullableInt32(query.DomainID),
		query.Since,
		query.Limit,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	result := []accountNotificationReply{}
	for rows.Next() {
		var item accountNotificationReply
		var createdAt time.Time
		var parentSnippet string
		if err := rows.Scan(
			&item.ID,
			&item.TargetType,
			&item.TargetID,
			&item.ParentID,
			&item.RootID,
			&item.BodyHTML,
			&createdAt,
			&item.CharacterID,
			&item.CharacterName,
			&item.CorporationID,
			&item.CorporationName,
			&item.AllianceID,
			&item.AllianceName,
			&item.ParentCommentID,
			&parentSnippet,
		); err != nil {
			return nil, err
		}
		item.CreatedAt = createdAt.UTC().Format(
			"2006-01-02T15:04:05.000Z",
		)
		item.ParentSnippet = truncateRunes(parentSnippet, 140)
		result = append(result, item)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return result, nil
}

func (s *postgresAccountStore) MarkNotificationsRead(
	ctx context.Context,
	characterID int32,
	id int64,
) (int64, error) {
	var updated int64
	err := s.db.QueryRow(ctx, `
		UPDATE users
		SET last_seen_notification_id = greatest(
			last_seen_notification_id,
			$2
		)
		WHERE character_id = $1
		RETURNING last_seen_notification_id`,
		characterID,
		id,
	).Scan(&updated)
	if errors.Is(err, pgx.ErrNoRows) {
		// Match the frontend fallback. A session can outlive a manually removed
		// users row, and marking the cursor remains harmless in that case.
		return id, nil
	}
	return updated, err
}
