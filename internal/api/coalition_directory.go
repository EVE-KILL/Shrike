package api

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"regexp"
	"sort"
	"strings"
	"unicode/utf8"

	"github.com/danielgtaylor/huma/v2"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
)

const (
	coalitionBodyLimit      = 32 << 10
	coalitionNameMax        = 100
	coalitionDescriptionMax = 2000
	coalitionSourceURLMax   = 2048
	coalitionAllianceMax    = 200
	coalitionStatsDays      = 30
)

var requiredCoalitionSession = []map[string][]string{{"eveSession": {}}}
var coalitionSlugPattern = regexp.MustCompile(`^[a-z0-9]+(?:-[a-z0-9]+)*$`)

type coalitionCreateBody struct {
	Name        *string            `json:"name" minLength:"2" maxLength:"100" doc:"Public coalition name."`
	Description string             `json:"description,omitempty" maxLength:"2000" doc:"Short description of the coalition."`
	SourceURL   optional[string]   `json:"source_url,omitempty" maxLength:"2048" doc:"Public source used to verify the membership."`
	AllianceIDs requestList[jsInt] `json:"alliance_ids" minItems:"1" maxItems:"200" doc:"Current member alliance IDs."`
}

type coalitionUpdateBody struct {
	Revision    *int64                       `json:"revision" minimum:"1" doc:"Current revision returned by the API. Prevents overwriting a newer edit."`
	Name        optional[string]             `json:"name,omitempty" minLength:"2" maxLength:"100" doc:"Replacement public name."`
	Description optional[string]             `json:"description,omitempty" maxLength:"2000" doc:"Replacement description. Null clears it."`
	SourceURL   optional[string]             `json:"source_url,omitempty" maxLength:"2048" doc:"Replacement verification source. Null clears it."`
	AllianceIDs optional[requestList[jsInt]] `json:"alliance_ids,omitempty" maxItems:"200" doc:"Complete replacement member list."`
}

type coalitionSnapshot struct {
	Name        string  `json:"name"`
	Description string  `json:"description"`
	SourceURL   *string `json:"source_url"`
	AllianceIDs []int32 `json:"alliance_ids"`
}

const coalitionDirectorySummarySQL = `
	WITH membership AS (
		SELECT cm.coalition_id,
		       count(DISTINCT cm.alliance_id)::bigint AS alliance_count,
		       count(ch.character_id)::bigint AS member_count
		FROM coalition_memberships cm
		LEFT JOIN characters ch
		  ON ch.alliance_id = cm.alliance_id
		 AND ch.deleted IS NOT TRUE
		GROUP BY cm.coalition_id
	), systems AS (
		SELECT cm.coalition_id, count(*)::bigint AS system_count
		FROM coalition_memberships cm
		JOIN sovereignty s ON s.alliance_id = cm.alliance_id
		GROUP BY cm.coalition_id
	), coalition_kill_ids AS (
		SELECT DISTINCT cm.coalition_id, ka.killmail_id
		FROM coalition_memberships cm
		JOIN killmail_attackers ka ON ka.alliance_id = cm.alliance_id
		WHERE ka.killmail_time >= now() - make_interval(days => $2::integer)
	), kills AS (
		SELECT ids.coalition_id,
		       count(*)::bigint AS kills,
		       coalesce(sum(k.total_value), 0)::double precision AS isk_destroyed
		FROM coalition_kill_ids ids
		JOIN killmails k ON k.killmail_id = ids.killmail_id
		GROUP BY ids.coalition_id
	), losses AS (
		SELECT cm.coalition_id,
		       count(*)::bigint AS losses,
		       coalesce(sum(k.total_value), 0)::double precision AS isk_lost
		FROM coalition_memberships cm
		JOIN killmails k ON k.victim_alliance_id = cm.alliance_id
		WHERE k.killmail_time >= now() - make_interval(days => $2::integer)
		GROUP BY cm.coalition_id
	)
	SELECT c.coalition_id, c.slug, c.name, c.description, c.source_url,
	       c.revision, c.created_at, c.updated_at,
	       c.created_by_character_id, creator.character_name AS created_by_character_name,
	       c.updated_by_character_id, editor.character_name AS updated_by_character_name,
	       coalesce(membership.alliance_count, 0)::bigint AS alliance_count,
	       coalesce(membership.member_count, 0)::bigint AS member_count,
	       coalesce(systems.system_count, 0)::bigint AS system_count,
	       coalesce(kills.kills, 0)::bigint AS kills,
	       coalesce(losses.losses, 0)::bigint AS losses,
	       coalesce(kills.isk_destroyed, 0)::double precision AS isk_destroyed,
	       coalesce(losses.isk_lost, 0)::double precision AS isk_lost
	FROM coalitions c
	LEFT JOIN users creator ON creator.character_id = c.created_by_character_id
	LEFT JOIN users editor ON editor.character_id = c.updated_by_character_id
	LEFT JOIN membership ON membership.coalition_id = c.coalition_id
	LEFT JOIN systems ON systems.coalition_id = c.coalition_id
	LEFT JOIN kills ON kills.coalition_id = c.coalition_id
	LEFT JOIN losses ON losses.coalition_id = c.coalition_id
	WHERE ($1::text = '' OR c.slug = $1)
	ORDER BY membership.member_count DESC NULLS LAST, c.name ASC`

func registerCoalitionDirectoryRoutes(a huma.API, opts Options) {
	auth := newAuthService(opts)

	registerLegacy(a, huma.Operation{
		OperationID: "coalitions", Method: http.MethodGet, Path: "/coalitions",
		Summary: "List community-maintained coalitions",
	}, coalitionDirectoryListHandler(opts))

	registerLegacy(a, huma.Operation{
		OperationID: "coalition", Method: http.MethodGet, Path: "/coalitions/{slug}",
		Summary:    "Get a coalition, its alliances, statistics, and edit history",
		Parameters: []*huma.Param{coalitionSlugParameter()},
	}, coalitionDirectoryDetailHandler(opts))

	registerLegacy(a, documentJSONBody[coalitionCreateBody](a, huma.Operation{
		OperationID: "coalition-create", Method: http.MethodPost, Path: "/coalitions",
		Summary:  "Create a community-maintained coalition",
		Security: requiredCoalitionSession, DefaultStatus: http.StatusCreated,
	}), coalitionDirectoryCreateHandler(opts, auth))

	registerLegacy(a, documentJSONBody[coalitionUpdateBody](a, huma.Operation{
		OperationID: "coalition-update", Method: http.MethodPatch, Path: "/coalitions/{slug}",
		Summary:    "Update coalition metadata or replace its alliance membership",
		Security:   requiredCoalitionSession,
		Parameters: []*huma.Param{coalitionSlugParameter()},
	}), coalitionDirectoryUpdateHandler(opts, auth))
}

func coalitionSlugParameter() *huma.Param {
	return &huma.Param{
		Name: "slug", In: "path", Required: true,
		Description: "Stable coalition slug.",
		Schema:      &huma.Schema{Type: huma.TypeString, Pattern: "^[a-z0-9]+(?:-[a-z0-9]+)*$"},
	}
}

func coalitionDirectoryListHandler(opts Options) legacyHandler {
	return func(ctx context.Context, _ *legacyRequest) (legacyPayload, error) {
		rows, err := queryMaps(ctx, opts.DB, coalitionDirectorySummarySQL, "", coalitionStatsDays)
		if err != nil {
			return legacyPayload{}, err
		}
		return jsonPayload(map[string]any{
			"coalitions": rows, "stats_window_days": coalitionStatsDays,
		}), nil
	}
}

func coalitionDirectoryDetailHandler(opts Options) legacyHandler {
	return func(ctx context.Context, req *legacyRequest) (legacyPayload, error) {
		return loadCoalitionDirectoryDetail(ctx, opts.DB, req.Param("slug"))
	}
}

func loadCoalitionDirectoryDetail(
	ctx context.Context,
	db Database,
	slug string,
) (legacyPayload, error) {
	if !coalitionSlugPattern.MatchString(slug) {
		return legacyPayload{}, apiError(http.StatusNotFound, "Coalition not found")
	}
	results, err := queryMapsConcurrent(ctx, db,
		databaseQuery{SQL: coalitionDirectorySummarySQL, Args: []any{slug, coalitionStatsDays}},
		databaseQuery{SQL: `
			SELECT a.alliance_id, a.name, a.ticker,
			       coalesce(members.member_count, 0)::bigint AS member_count,
			       coalesce(corporations.corporation_count, 0)::bigint AS corporation_count,
			       coalesce(systems.system_count, 0)::bigint AS system_count,
			       cm.added_at, cm.added_by_character_id,
			       added_by.character_name AS added_by_character_name
			FROM coalitions c
			JOIN coalition_memberships cm ON cm.coalition_id = c.coalition_id
			JOIN alliances a ON a.alliance_id = cm.alliance_id
			LEFT JOIN LATERAL (
				SELECT count(*)::bigint AS member_count
				FROM characters ch
				WHERE ch.alliance_id = a.alliance_id AND ch.deleted IS NOT TRUE
			) members ON true
			LEFT JOIN LATERAL (
				SELECT count(*)::bigint AS corporation_count
				FROM corporations corporation
				WHERE corporation.alliance_id = a.alliance_id
				  AND corporation.deleted IS NOT TRUE
			) corporations ON true
			LEFT JOIN LATERAL (
				SELECT count(*)::bigint AS system_count
				FROM sovereignty sovereignty_claim
				WHERE sovereignty_claim.alliance_id = a.alliance_id
			) systems ON true
			LEFT JOIN users added_by ON added_by.character_id = cm.added_by_character_id
			WHERE c.slug = $1
			ORDER BY members.member_count DESC NULLS LAST, a.name ASC`, Args: []any{slug}},
		databaseQuery{SQL: `
			SELECT e.edit_id, e.editor_character_id, e.editor_character_name,
			       e.action, e.summary, e.changes, e.created_at
			FROM coalitions c
			JOIN coalition_edits e ON e.coalition_id = c.coalition_id
			WHERE c.slug = $1
			ORDER BY e.created_at DESC, e.edit_id DESC
			LIMIT 100`, Args: []any{slug}},
	)
	if err != nil {
		return legacyPayload{}, err
	}
	if len(results[0]) == 0 {
		return legacyPayload{}, apiError(http.StatusNotFound, "Coalition not found")
	}
	return jsonPayload(map[string]any{
		"coalition": results[0][0], "alliances": results[1], "edits": results[2],
		"stats_window_days": coalitionStatsDays,
	}), nil
}

func coalitionDirectoryMutationPrincipal(
	ctx context.Context,
	req *legacyRequest,
	auth *authService,
) (*Principal, error) {
	setAccountNoStore(req.Huma)
	if err := requireSameOriginMutation(req.Huma); err != nil {
		return nil, err
	}
	return auth.requirePrincipal(ctx, req)
}

func coalitionDirectoryCreateHandler(opts Options, auth *authService) legacyHandler {
	return func(ctx context.Context, req *legacyRequest) (legacyPayload, error) {
		principal, err := coalitionDirectoryMutationPrincipal(ctx, req, auth)
		if err != nil {
			return legacyPayload{}, err
		}
		wire, err := decodeJSONBody[coalitionCreateBody](req, coalitionBodyLimit)
		if err != nil {
			return legacyPayload{}, err
		}
		body, err := validateCoalitionCreate(wire)
		if err != nil {
			return legacyPayload{}, err
		}
		db, err := mutationDatabase(opts)
		if err != nil {
			return legacyPayload{}, err
		}
		tx, err := db.Begin(ctx)
		if err != nil {
			return legacyPayload{}, err
		}
		defer tx.Rollback(ctx) //nolint:errcheck

		if err := validateCoalitionAllianceRows(ctx, tx, body.AllianceIDs, 0); err != nil {
			return legacyPayload{}, err
		}
		var coalitionID int64
		err = tx.QueryRow(ctx, `
			INSERT INTO coalitions (
				slug, name, description, source_url,
				created_by_character_id, updated_by_character_id
			) VALUES ($1, $2, $3, $4, $5, $5)
			RETURNING coalition_id`, body.Slug, body.Name, body.Description,
			body.SourceURL, principal.CharacterID).Scan(&coalitionID)
		if err != nil {
			return legacyPayload{}, coalitionDirectoryWriteError(err)
		}
		if err := insertCoalitionMemberships(ctx, tx, coalitionID, body.AllianceIDs, principal.CharacterID); err != nil {
			return legacyPayload{}, coalitionDirectoryWriteError(err)
		}
		after := coalitionSnapshot{
			Name: body.Name, Description: body.Description,
			SourceURL: body.SourceURL, AllianceIDs: body.AllianceIDs,
		}
		if err := insertCoalitionEdit(ctx, tx, coalitionID, principal, "create",
			fmt.Sprintf("Created the coalition with %d alliances", len(body.AllianceIDs)),
			nil, &after); err != nil {
			return legacyPayload{}, err
		}
		if err := tx.Commit(ctx); err != nil {
			return legacyPayload{}, coalitionDirectoryWriteError(err)
		}
		invalidateCoalitionDirectoryCache(ctx, opts)
		payload, err := loadCoalitionDirectoryDetail(ctx, primaryDatabase(opts), body.Slug)
		if err != nil {
			return legacyPayload{}, err
		}
		payload.Status = http.StatusCreated
		payload.Headers = noStoreHeaders()
		return payload, nil
	}
}

type validatedCoalitionCreate struct {
	Slug, Name, Description string
	SourceURL               *string
	AllianceIDs             []int32
}

func validateCoalitionCreate(body *coalitionCreateBody) (validatedCoalitionCreate, error) {
	if body == nil || body.Name == nil {
		return validatedCoalitionCreate{}, apiError(http.StatusUnprocessableEntity, "name is required")
	}
	name, err := validateCoalitionName(*body.Name)
	if err != nil {
		return validatedCoalitionCreate{}, err
	}
	description, err := validateCoalitionDescription(body.Description)
	if err != nil {
		return validatedCoalitionCreate{}, err
	}
	sourceURL, err := validateCoalitionSourceURL(body.SourceURL)
	if err != nil {
		return validatedCoalitionCreate{}, err
	}
	alliances, err := validateCoalitionAllianceIDs([]jsInt(body.AllianceIDs))
	if err != nil {
		return validatedCoalitionCreate{}, err
	}
	slug := slugifyBlogTitle(name)
	if slug == "" {
		return validatedCoalitionCreate{}, apiError(http.StatusUnprocessableEntity, "name must produce a usable slug")
	}
	return validatedCoalitionCreate{
		Slug: slug, Name: name, Description: description,
		SourceURL: sourceURL, AllianceIDs: alliances,
	}, nil
}

func coalitionDirectoryUpdateHandler(opts Options, auth *authService) legacyHandler {
	return func(ctx context.Context, req *legacyRequest) (legacyPayload, error) {
		principal, err := coalitionDirectoryMutationPrincipal(ctx, req, auth)
		if err != nil {
			return legacyPayload{}, err
		}
		slug := req.Param("slug")
		if !coalitionSlugPattern.MatchString(slug) {
			return legacyPayload{}, apiError(http.StatusNotFound, "Coalition not found")
		}
		wire, err := decodeJSONBody[coalitionUpdateBody](req, coalitionBodyLimit)
		if err != nil {
			return legacyPayload{}, err
		}
		patch, err := validateCoalitionUpdate(wire)
		if err != nil {
			return legacyPayload{}, err
		}
		db, err := mutationDatabase(opts)
		if err != nil {
			return legacyPayload{}, err
		}
		tx, err := db.Begin(ctx)
		if err != nil {
			return legacyPayload{}, err
		}
		defer tx.Rollback(ctx) //nolint:errcheck

		var coalitionID int64
		var currentName, currentDescription string
		var currentSourceURL *string
		var currentRevision int64
		err = tx.QueryRow(ctx, `
			SELECT coalition_id, name, description, source_url, revision
			FROM coalitions WHERE slug = $1 FOR UPDATE`, slug).Scan(
			&coalitionID, &currentName, &currentDescription, &currentSourceURL, &currentRevision,
		)
		if errors.Is(err, pgx.ErrNoRows) {
			return legacyPayload{}, apiError(http.StatusNotFound, "Coalition not found")
		}
		if err != nil {
			return legacyPayload{}, err
		}
		if currentRevision != patch.Revision {
			return legacyPayload{}, apiError(http.StatusConflict,
				fmt.Sprintf("Coalition changed since revision %d; reload before editing", patch.Revision))
		}
		currentIDs, err := loadCoalitionAllianceIDs(ctx, tx, coalitionID)
		if err != nil {
			return legacyPayload{}, err
		}
		before := coalitionSnapshot{
			Name: currentName, Description: currentDescription,
			SourceURL: currentSourceURL, AllianceIDs: currentIDs,
		}
		after := before
		if patch.Name != nil {
			after.Name = *patch.Name
		}
		if patch.HasDescription {
			after.Description = patch.Description
		}
		if patch.HasSourceURL {
			after.SourceURL = patch.SourceURL
		}
		if patch.AllianceIDs != nil {
			after.AllianceIDs = patch.AllianceIDs
			if err := validateCoalitionAllianceRows(ctx, tx, after.AllianceIDs, coalitionID); err != nil {
				return legacyPayload{}, err
			}
		}
		summary := summarizeCoalitionEdit(before, after)
		if summary == "" {
			if err := tx.Rollback(ctx); err != nil {
				return legacyPayload{}, err
			}
			payload, err := loadCoalitionDirectoryDetail(ctx, primaryDatabase(opts), slug)
			if err != nil {
				return legacyPayload{}, err
			}
			payload.Headers = noStoreHeaders()
			return payload, nil
		}
		_, err = tx.Exec(ctx, `
			UPDATE coalitions
			SET name = $2, description = $3, source_url = $4,
			    revision = revision + 1, updated_by_character_id = $5,
			    updated_at = now()
			WHERE coalition_id = $1`, coalitionID, after.Name, after.Description,
			after.SourceURL, principal.CharacterID)
		if err != nil {
			return legacyPayload{}, coalitionDirectoryWriteError(err)
		}
		if patch.AllianceIDs != nil {
			if _, err := tx.Exec(ctx, `
				DELETE FROM coalition_memberships
				WHERE coalition_id = $1 AND NOT (alliance_id = ANY($2::int[]))`,
				coalitionID, after.AllianceIDs); err != nil {
				return legacyPayload{}, err
			}
			if err := insertCoalitionMemberships(ctx, tx, coalitionID, after.AllianceIDs, principal.CharacterID); err != nil {
				return legacyPayload{}, coalitionDirectoryWriteError(err)
			}
		}
		if err := insertCoalitionEdit(ctx, tx, coalitionID, principal, "update", summary, &before, &after); err != nil {
			return legacyPayload{}, err
		}
		if err := tx.Commit(ctx); err != nil {
			return legacyPayload{}, coalitionDirectoryWriteError(err)
		}
		invalidateCoalitionDirectoryCache(ctx, opts)
		payload, err := loadCoalitionDirectoryDetail(ctx, primaryDatabase(opts), slug)
		if err != nil {
			return legacyPayload{}, err
		}
		payload.Headers = noStoreHeaders()
		return payload, nil
	}
}

type validatedCoalitionUpdate struct {
	Revision       int64
	Name           *string
	Description    string
	HasDescription bool
	SourceURL      *string
	HasSourceURL   bool
	AllianceIDs    []int32
}

func validateCoalitionUpdate(body *coalitionUpdateBody) (validatedCoalitionUpdate, error) {
	if body == nil || body.Revision == nil || *body.Revision < 1 {
		return validatedCoalitionUpdate{}, apiError(http.StatusUnprocessableEntity, "revision is required")
	}
	result := validatedCoalitionUpdate{Revision: *body.Revision}
	if body.Name.present() {
		if body.Name.Value == nil {
			return result, apiError(http.StatusUnprocessableEntity, "name cannot be null")
		}
		name, err := validateCoalitionName(*body.Name.Value)
		if err != nil {
			return result, err
		}
		result.Name = &name
	}
	if body.Description.present() {
		value := body.Description.valueOr("")
		description, err := validateCoalitionDescription(value)
		if err != nil {
			return result, err
		}
		result.Description, result.HasDescription = description, true
	}
	if body.SourceURL.present() {
		sourceURL, err := validateCoalitionSourceURL(body.SourceURL)
		if err != nil {
			return result, err
		}
		result.SourceURL, result.HasSourceURL = sourceURL, true
	}
	if body.AllianceIDs.present() {
		if body.AllianceIDs.Value == nil {
			return result, apiError(http.StatusUnprocessableEntity, "alliance_ids cannot be null")
		}
		alliances, err := validateCoalitionAllianceIDs([]jsInt(*body.AllianceIDs.Value))
		if err != nil {
			return result, err
		}
		result.AllianceIDs = alliances
	}
	if result.Name == nil && !result.HasDescription && !result.HasSourceURL && !body.AllianceIDs.present() {
		return result, apiError(http.StatusUnprocessableEntity, "No coalition changes supplied")
	}
	return result, nil
}

func validateCoalitionName(raw string) (string, error) {
	name := strings.TrimSpace(raw)
	length := utf8.RuneCountInString(name)
	if length < 2 || length > coalitionNameMax {
		return "", apiError(http.StatusUnprocessableEntity, "name must be between 2 and 100 characters")
	}
	return name, nil
}

func validateCoalitionDescription(raw string) (string, error) {
	description := strings.TrimSpace(raw)
	if utf8.RuneCountInString(description) > coalitionDescriptionMax {
		return "", apiError(http.StatusUnprocessableEntity, "description must be at most 2000 characters")
	}
	return description, nil
}

func validateCoalitionSourceURL(value optional[string]) (*string, error) {
	if value.Value == nil || strings.TrimSpace(*value.Value) == "" {
		return nil, nil
	}
	raw := strings.TrimSpace(*value.Value)
	if utf8.RuneCountInString(raw) > coalitionSourceURLMax {
		return nil, apiError(http.StatusUnprocessableEntity, "source_url must be at most 2048 characters")
	}
	parsed, err := url.Parse(raw)
	if err != nil || parsed.Host == "" || (parsed.Scheme != "https" && parsed.Scheme != "http") {
		return nil, apiError(http.StatusUnprocessableEntity, "source_url must be an absolute HTTP or HTTPS URL")
	}
	return &raw, nil
}

func validateCoalitionAllianceIDs(raw []jsInt) ([]int32, error) {
	if len(raw) == 0 {
		return nil, apiError(http.StatusUnprocessableEntity, "alliance_ids must contain at least one alliance")
	}
	if len(raw) > coalitionAllianceMax {
		return nil, apiError(http.StatusUnprocessableEntity, "alliance_ids must not exceed 200 entries")
	}
	ids, err := coalitionIDs(raw, "alliance_ids")
	if err != nil {
		return nil, err
	}
	sort.Slice(ids, func(i, j int) bool { return ids[i] < ids[j] })
	return ids, nil
}

type coalitionQueryer interface {
	Query(context.Context, string, ...any) (pgx.Rows, error)
	QueryRow(context.Context, string, ...any) pgx.Row
}

func validateCoalitionAllianceRows(ctx context.Context, q coalitionQueryer, ids []int32, ownID int64) error {
	rows, err := q.Query(ctx, `
		SELECT requested.alliance_id,
		       a.name,
		       c.coalition_id,
		       c.name AS coalition_name
		FROM unnest($1::int[]) AS requested(alliance_id)
		LEFT JOIN alliances a
		  ON a.alliance_id = requested.alliance_id AND a.deleted IS NOT TRUE
		LEFT JOIN coalition_memberships cm ON cm.alliance_id = requested.alliance_id
		LEFT JOIN coalitions c ON c.coalition_id = cm.coalition_id
		ORDER BY requested.alliance_id`, ids)
	if err != nil {
		return err
	}
	defer rows.Close()
	for rows.Next() {
		var allianceID int32
		var allianceName *string
		var coalitionID *int64
		var coalitionName *string
		if err := rows.Scan(&allianceID, &allianceName, &coalitionID, &coalitionName); err != nil {
			return err
		}
		if allianceName == nil {
			return apiError(http.StatusUnprocessableEntity, fmt.Sprintf("Alliance %d does not exist", allianceID))
		}
		if coalitionID != nil && *coalitionID != ownID {
			return apiError(http.StatusConflict, fmt.Sprintf("%s already belongs to %s", *allianceName, pointerStringOr(coalitionName, "another coalition")))
		}
	}
	return rows.Err()
}

func loadCoalitionAllianceIDs(ctx context.Context, q coalitionQueryer, coalitionID int64) ([]int32, error) {
	rows, err := q.Query(ctx, `
		SELECT alliance_id FROM coalition_memberships
		WHERE coalition_id = $1 ORDER BY alliance_id`, coalitionID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	ids := []int32{}
	for rows.Next() {
		var id int32
		if err := rows.Scan(&id); err != nil {
			return nil, err
		}
		ids = append(ids, id)
	}
	return ids, rows.Err()
}

func insertCoalitionMemberships(ctx context.Context, tx pgx.Tx, coalitionID int64, ids []int32, actorID int32) error {
	_, err := tx.Exec(ctx, `
		INSERT INTO coalition_memberships (
			coalition_id, alliance_id, added_by_character_id
		)
		SELECT $1, alliance_id, $3
		FROM unnest($2::int[]) AS member(alliance_id)
		ON CONFLICT (coalition_id, alliance_id) DO NOTHING`, coalitionID, ids, actorID)
	return err
}

func insertCoalitionEdit(
	ctx context.Context,
	tx pgx.Tx,
	coalitionID int64,
	principal *Principal,
	action, summary string,
	before, after *coalitionSnapshot,
) error {
	changes, err := json.Marshal(map[string]any{"before": before, "after": after})
	if err != nil {
		return err
	}
	_, err = tx.Exec(ctx, `
		INSERT INTO coalition_edits (
			coalition_id, editor_character_id, editor_character_name,
			action, summary, changes
		) VALUES ($1, $2, $3, $4, $5, $6::jsonb)`,
		coalitionID, principal.CharacterID, principal.CharacterName,
		action, summary, string(changes))
	return err
}

func summarizeCoalitionEdit(before, after coalitionSnapshot) string {
	changes := []string{}
	if before.Name != after.Name {
		changes = append(changes, "renamed the coalition")
	}
	if before.Description != after.Description {
		changes = append(changes, "updated the description")
	}
	if pointerStringOr(before.SourceURL, "") != pointerStringOr(after.SourceURL, "") {
		changes = append(changes, "updated the verification source")
	}
	beforeSet, afterSet := map[int32]bool{}, map[int32]bool{}
	for _, id := range before.AllianceIDs {
		beforeSet[id] = true
	}
	for _, id := range after.AllianceIDs {
		afterSet[id] = true
	}
	added, removed := 0, 0
	for id := range afterSet {
		if !beforeSet[id] {
			added++
		}
	}
	for id := range beforeSet {
		if !afterSet[id] {
			removed++
		}
	}
	if added > 0 {
		changes = append(changes, fmt.Sprintf("added %d alliance%s", added, pluralSuffix(added)))
	}
	if removed > 0 {
		changes = append(changes, fmt.Sprintf("removed %d alliance%s", removed, pluralSuffix(removed)))
	}
	if len(changes) == 0 {
		return ""
	}
	return strings.ToUpper(changes[0][:1]) + changes[0][1:] + joinEditSummary(changes[1:])
}

func joinEditSummary(changes []string) string {
	if len(changes) == 0 {
		return ""
	}
	return "; " + strings.Join(changes, "; ")
}

func pluralSuffix(count int) string {
	if count == 1 {
		return ""
	}
	return "s"
}

func pointerStringOr(value *string, fallback string) string {
	if value == nil {
		return fallback
	}
	return *value
}

func coalitionDirectoryWriteError(err error) error {
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) {
		switch pgErr.Code {
		case "23505":
			return apiError(http.StatusConflict, "A coalition or alliance membership already uses that value")
		case "23503":
			return apiError(http.StatusUnprocessableEntity, "A referenced alliance or user does not exist")
		}
	}
	return err
}

func invalidateCoalitionDirectoryCache(ctx context.Context, opts Options) {
	if opts.responseCache == nil {
		return
	}
	opts.responseCache.DeleteMatching(ctx, "shrike:api:*:/coalitions*")
	opts.responseCache.DeleteMatching(ctx, "web:shrike:api:*:/coalitions*")
}

func noStoreHeaders() http.Header {
	return http.Header{"Cache-Control": []string{"private, no-store"}}
}
