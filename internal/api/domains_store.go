package api

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"strings"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
)

const domainColumns = `
	domain.id, domain.subdomain, domain.custom_hostname, domain.user_id,
	domain.entities, domain.theme, domain.navbar_links,
	domain.site_name, domain.site_description, domain.active,
	domain.created_at, domain.updated_at, domain.widgets,
	domain.campaign_policy`

type domainQuerier interface {
	Query(context.Context, string, ...any) (pgx.Rows, error)
}

type domainCampaignOptions struct {
	Search      string
	Limit       int
	OwnOnly     bool
	IDs         []string
	RestrictIDs bool
}

func domainQueryMaps(
	ctx context.Context,
	db domainQuerier,
	query string,
	args ...any,
) ([]map[string]any, error) {
	if db == nil {
		return nil, apiError(
			http.StatusServiceUnavailable,
			"Domain storage is not configured",
		)
	}
	rows, err := db.Query(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	return pgx.CollectRows(rows, pgx.RowToMap)
}

func domainQueryMap(
	ctx context.Context,
	db domainQuerier,
	query string,
	args ...any,
) (map[string]any, error) {
	rows, err := domainQueryMaps(ctx, db, query, args...)
	if err != nil || len(rows) == 0 {
		return nil, err
	}
	return rows[0], nil
}

func normalizeDomainRecord(row map[string]any) map[string]any {
	if row == nil {
		return nil
	}
	row["entities"] = jsonArray(row["entities"])
	row["theme"] = jsonObject(row["theme"])
	row["navbar_links"] = jsonArray(row["navbar_links"])
	if row["widgets"] == nil {
		row["widgets"] = defaultDomainWidgets()
	} else {
		row["widgets"] = jsonObject(row["widgets"])
	}
	return row
}

func defaultDomainWidgets() map[string]any {
	return map[string]any{
		"top": []any{
			map[string]any{"type": "mostValuable", "enabled": true},
		},
		"left": []any{
			map[string]any{"type": "topCharacters", "enabled": true},
			map[string]any{"type": "topCorporations", "enabled": true},
			map[string]any{"type": "topAlliances", "enabled": true},
			map[string]any{"type": "topShips", "enabled": true},
			map[string]any{"type": "topSystems", "enabled": true},
			map[string]any{"type": "topRegions", "enabled": true},
		},
		"right": []any{
			map[string]any{
				"type": "killList", "enabled": true,
				"killlistType": "latest",
			},
		},
		"columnRatio": "250px_1fr",
	}
}

func (s *domainService) loadOwnedDomains(
	ctx context.Context,
	characterID int32,
) ([]map[string]any, error) {
	rows, err := domainQueryMaps(ctx, s.db, `
		SELECT `+domainColumns+`
		FROM custom_domains domain
		WHERE domain.user_id = $1
		ORDER BY domain.created_at`, characterID)
	if err != nil || len(rows) == 0 {
		if rows == nil {
			rows = []map[string]any{}
		}
		return rows, err
	}
	ids := make([]int32, 0, len(rows))
	for _, row := range rows {
		normalizeDomainRecord(row)
		ids = append(ids, int32From(row["id"]))
	}

	related, err := queryMapsConcurrent(
		ctx,
		s.db,
		databaseQuery{
			SQL: `
				SELECT selection.domain_id, campaign.campaign_id, campaign.name,
				       campaign.description, campaign.visibility, campaign.status,
				       campaign.start_time, campaign.end_time,
				       campaign.estimated_killmails, selection.public_on_domain
				FROM custom_domain_campaigns selection
				JOIN campaigns campaign
				  ON campaign.campaign_id = selection.campaign_id
				WHERE selection.domain_id = ANY($1::integer[])
				ORDER BY selection.created_at, selection.campaign_id`,
			Args: []any{ids},
		},
		databaseQuery{
			SQL: `
				SELECT id, domain_id, status, created_at
				FROM domain_assets
				WHERE domain_id = ANY($1::integer[])
				  AND type = 'background'
				  AND status <> 'rejected'
				ORDER BY id`,
			Args: []any{ids},
		},
		databaseQuery{
			SQL: `
				SELECT id, domain_id, type, status, reject_reason, created_at
				FROM domain_assets
				WHERE domain_id = ANY($1::integer[])
				  AND type = ANY('{banner,logo}'::text[])
				  AND status = ANY('{pending,rejected}'::text[])
				ORDER BY id DESC`,
			Args: []any{ids},
		},
	)
	if err != nil {
		return nil, err
	}
	campaigns, backgrounds, slots := related[0], related[1], related[2]

	campaignsByDomain := make(map[int32][]map[string]any)
	for _, campaign := range campaigns {
		id := int32From(campaign["domain_id"])
		campaignsByDomain[id] = append(campaignsByDomain[id], campaign)
	}
	backgroundsByDomain := make(map[int32][]map[string]any)
	for _, asset := range backgrounds {
		id := int32From(asset["domain_id"])
		backgroundsByDomain[id] = append(backgroundsByDomain[id], asset)
	}
	slotsByDomain := make(map[int32]map[string]map[string]any)
	for _, asset := range slots {
		id := int32From(asset["domain_id"])
		kind, _ := asset["type"].(string)
		if slotsByDomain[id] == nil {
			slotsByDomain[id] = make(map[string]map[string]any)
		}
		if slotsByDomain[id][kind] == nil {
			slotsByDomain[id][kind] = map[string]any{
				"id":            asset["id"],
				"status":        asset["status"],
				"reject_reason": asset["reject_reason"],
				"created_at":    asset["created_at"],
			}
		}
	}
	for _, row := range rows {
		id := int32From(row["id"])
		row["backgrounds"] = emptyDomainRows(backgroundsByDomain[id])
		row["bannerAsset"] = slotsByDomain[id]["banner"]
		row["logoAsset"] = slotsByDomain[id]["logo"]
		row["campaigns"] = emptyDomainRows(campaignsByDomain[id])
	}
	return rows, nil
}

func emptyDomainRows(rows []map[string]any) []map[string]any {
	if rows == nil {
		return []map[string]any{}
	}
	return rows
}

func (s *domainService) domainSubdomainExists(
	ctx context.Context,
	subdomain string,
) (bool, error) {
	row, err := domainQueryMap(ctx, s.db, `
		SELECT id
		FROM custom_domains
		WHERE subdomain = $1
		LIMIT 1`, subdomain)
	return row != nil, err
}

func (s *domainService) createDomain(
	ctx context.Context,
	userID int32,
	input domainCreateInput,
) (map[string]any, error) {
	tx, err := s.mutations.Begin(ctx)
	if err != nil {
		return nil, err
	}
	defer func() { _ = tx.Rollback(ctx) }()

	// The schema's historical uniqueness index only covers active rows. These
	// locks preserve the API's stronger "all rows" uniqueness promise and the
	// per-user cap under concurrent creates without changing production schema.
	if _, err = tx.Exec(ctx, `
		SELECT pg_advisory_xact_lock($1::bigint)`,
		int64(userID)); err != nil {
		return nil, err
	}
	if _, err = tx.Exec(ctx, `
		SELECT pg_advisory_xact_lock(hashtextextended($1, 0))`,
		"evekill-domain:"+input.Subdomain); err != nil {
		return nil, err
	}
	countRow, err := domainQueryMap(ctx, tx, `
		SELECT COUNT(*)::integer AS count
		FROM custom_domains
		WHERE user_id = $1`, userID)
	if err != nil {
		return nil, err
	}
	if int32From(countRow["count"]) >= domainMaximumPerUser {
		return nil, apiError(
			http.StatusBadRequest,
			fmt.Sprintf(
				"Maximum %d domains per user",
				domainMaximumPerUser,
			),
		)
	}
	conflict, err := domainQueryMap(ctx, tx, `
		SELECT id
		FROM custom_domains
		WHERE subdomain = $1
		LIMIT 1`, input.Subdomain)
	if err != nil {
		return nil, err
	}
	if conflict != nil {
		return nil, apiError(
			http.StatusConflict, "This subdomain is already taken",
		)
	}

	if !hasDomainThemeChoice(input.Theme) {
		for _, entity := range input.Entities {
			if entity.Type != "corporation" {
				continue
			}
			palette, paletteErr := domainQueryMap(ctx, tx, `
				SELECT palette
				FROM corporations
				WHERE corporation_id = $1
				LIMIT 1`, entity.ID)
			if paletteErr != nil {
				return nil, paletteErr
			}
			if palette != nil {
				seedDomainTheme(input.Theme, jsonObject(palette["palette"]))
			}
			break
		}
	}
	entities, _ := json.Marshal(input.Entities)
	theme, _ := json.Marshal(input.Theme)
	navbar, _ := json.Marshal(input.NavbarLinks)
	args := []any{
		input.Subdomain, userID, entities, theme, navbar,
		input.SiteName, input.SiteDescription,
	}
	query := `
		INSERT INTO custom_domains (
		  subdomain, user_id, entities, theme, navbar_links,
		  site_name, site_description`
	values := `
		VALUES ($1, $2, $3::jsonb, $4::jsonb, $5::jsonb, $6, $7`
	if input.WidgetsPresent {
		widgets, _ := json.Marshal(input.Widgets)
		args = append(args, widgets)
		query += `, widgets`
		values += `, $8::jsonb`
	}
	query += `)` + values + `)
		RETURNING id, subdomain, custom_hostname, user_id, entities, theme,
		          navbar_links, site_name, site_description, active,
		          created_at, updated_at, widgets, campaign_policy`
	row, err := domainQueryMap(ctx, tx, query, args...)
	if err != nil {
		return nil, domainMutationError(err)
	}
	if err := tx.Commit(ctx); err != nil {
		return nil, domainMutationError(err)
	}
	return normalizeDomainRecord(row), nil
}

func (s *domainService) loadOwnedDomain(
	ctx context.Context,
	db domainQuerier,
	id int32,
	userID int32,
	forUpdate bool,
) (map[string]any, error) {
	suffix := ""
	if forUpdate {
		suffix = " FOR UPDATE"
	}
	row, err := domainQueryMap(ctx, db, `
		SELECT `+domainColumns+`
		FROM custom_domains domain
		WHERE domain.id = $1 AND domain.user_id = $2
		LIMIT 1`+suffix, id, userID)
	return normalizeDomainRecord(row), err
}

func (s *domainService) updateDomain(
	ctx context.Context,
	id int32,
	userID int32,
	principal Principal,
	input domainUpdateInput,
) (map[string]any, error) {
	tx, err := s.mutations.Begin(ctx)
	if err != nil {
		return nil, err
	}
	defer func() { _ = tx.Rollback(ctx) }()
	existing, err := s.loadOwnedDomain(ctx, tx, id, userID, true)
	if err != nil {
		return nil, err
	}
	if existing == nil {
		return nil, apiError(http.StatusNotFound, "Domain not found")
	}
	if input.ActivePresent && input.Active &&
		!boolFrom(existing["active"]) {
		if err := ensureDomainActivationAvailable(ctx, tx, existing); err != nil {
			return nil, err
		}
	}

	effective := make(map[string]any, len(existing))
	for key, value := range existing {
		effective[key] = value
	}
	if input.EntitiesPresent {
		effective["entities"] = domainEntitiesJSON(input.Entities)
	}
	if input.CampaignIDsPresent {
		eligible, eligibleErr := s.loadEligibleDomainCampaigns(
			ctx, tx, effective, principal, domainCampaignOptions{
				IDs:         input.CampaignIDs,
				RestrictIDs: true,
				Limit:       domainMaximumCampaigns,
			},
		)
		if eligibleErr != nil {
			return nil, eligibleErr
		}
		allowed := make(map[string]struct{}, len(eligible))
		for _, campaign := range eligible {
			allowed[fmt.Sprint(campaign["campaign_id"])] = struct{}{}
		}
		for _, campaignID := range input.CampaignIDs {
			if _, ok := allowed[campaignID]; !ok {
				return nil, apiError(
					http.StatusForbidden,
					fmt.Sprintf(
						"Campaign %s cannot be added to this domain",
						campaignID,
					),
				)
			}
		}
	}

	assignments := make([]string, 0, 10)
	args := make([]any, 0, 12)
	add := func(column, suffix string, value any) {
		args = append(args, value)
		assignments = append(assignments, fmt.Sprintf(
			"%s = $%d%s", column, len(args), suffix,
		))
	}
	if input.EntitiesPresent {
		encoded, _ := json.Marshal(input.Entities)
		add("entities", "::jsonb", encoded)
	}
	if input.ThemePresent {
		merged := mergeDomainTheme(
			jsonObject(existing["theme"]), input.Theme,
		)
		encoded, _ := json.Marshal(merged)
		add("theme", "::jsonb", encoded)
	}
	if input.NavbarPresent {
		encoded, _ := json.Marshal(input.NavbarLinks)
		add("navbar_links", "::jsonb", encoded)
	}
	if input.WidgetsPresent {
		encoded, _ := json.Marshal(input.Widgets)
		add("widgets", "::jsonb", encoded)
	}
	if input.SiteNamePresent {
		add("site_name", "", input.SiteName)
	}
	if input.SiteDescriptionPresent {
		add("site_description", "", input.SiteDescription)
	}
	if input.ActivePresent {
		add("active", "", input.Active)
	}
	if input.CampaignPolicyPresent {
		add("campaign_policy", "", input.CampaignPolicy)
	}
	add("updated_at", "", s.now().UTC())
	args = append(args, id)
	row, err := domainQueryMap(ctx, tx, `
		UPDATE custom_domains domain
		SET `+strings.Join(assignments, ", ")+`
		WHERE domain.id = $`+strconv.Itoa(len(args))+`
		RETURNING domain.id, domain.subdomain, domain.custom_hostname,
		          domain.user_id, domain.entities, domain.theme,
		          domain.navbar_links, domain.site_name,
		          domain.site_description, domain.active,
		          domain.created_at, domain.updated_at, domain.widgets,
		          domain.campaign_policy`, args...)
	if err != nil {
		return nil, domainMutationError(err)
	}
	if input.CampaignIDsPresent {
		if _, err := tx.Exec(ctx, `
			DELETE FROM custom_domain_campaigns
			WHERE domain_id = $1`, id); err != nil {
			return nil, err
		}
		if len(input.CampaignIDs) > 0 {
			if _, err := tx.Exec(ctx, `
				INSERT INTO custom_domain_campaigns (
				  domain_id, campaign_id, added_by_character_id,
				  public_on_domain
				)
				SELECT $1, campaign_id, $2,
				       campaign_id = ANY($4::text[])
				FROM unnest($3::text[]) AS campaign_id`,
				id, userID, input.CampaignIDs,
				input.PublicCampaignIDs,
			); err != nil {
				return nil, err
			}
		}
	}
	if err := tx.Commit(ctx); err != nil {
		return nil, domainMutationError(err)
	}
	return normalizeDomainRecord(row), nil
}

func (s *domainService) deleteDomain(
	ctx context.Context,
	id int32,
	userID int32,
) ([]domainStorageReference, error) {
	tx, err := s.mutations.Begin(ctx)
	if err != nil {
		return nil, err
	}
	defer func() { _ = tx.Rollback(ctx) }()
	domain, err := s.loadOwnedDomain(ctx, tx, id, userID, true)
	if err != nil {
		return nil, err
	}
	if domain == nil {
		return nil, apiError(http.StatusNotFound, "Domain not found")
	}
	rows, err := domainQueryMaps(ctx, tx, `
		SELECT id, domain_id, type, storage_key
		FROM domain_assets
		WHERE domain_id = $1`, id)
	if err != nil {
		return nil, err
	}
	refs := make([]domainStorageReference, 0, len(rows))
	for _, row := range rows {
		refs = append(refs, domainStorageReference{
			AssetID:  int32From(row["id"]),
			DomainID: int32From(row["domain_id"]),
			Type:     fmt.Sprint(row["type"]),
			Key:      fmt.Sprint(row["storage_key"]),
		})
	}
	if _, err := tx.Exec(ctx, `
		DELETE FROM domain_assets WHERE domain_id = $1`, id); err != nil {
		return nil, err
	}
	tag, err := tx.Exec(ctx, `
		DELETE FROM custom_domains
		WHERE id = $1 AND user_id = $2`, id, userID)
	if err != nil {
		return nil, err
	}
	if tag.RowsAffected() != 1 {
		return nil, apiError(http.StatusNotFound, "Domain not found")
	}
	if err := tx.Commit(ctx); err != nil {
		return nil, err
	}
	return refs, nil
}

func (s *domainService) loadEligibleDomainCampaigns(
	ctx context.Context,
	db domainQuerier,
	domain map[string]any,
	principal Principal,
	options domainCampaignOptions,
) ([]map[string]any, error) {
	entities, _ := json.Marshal(jsonArray(domain["entities"]))
	searchRunes := []rune(strings.TrimSpace(options.Search))
	if len(searchRunes) > domainCampaignSearchLength {
		searchRunes = searchRunes[:domainCampaignSearchLength]
	}
	search := string(searchRunes)
	limit := options.Limit
	if limit < 1 {
		limit = 20
	}
	if limit > domainMaximumCampaigns {
		limit = domainMaximumCampaigns
	}
	var ids any
	if options.RestrictIDs {
		ids = options.IDs
	}
	rows, err := domainQueryMaps(ctx, db, `
		WITH raw_pairs(entity_type, entity_id) AS (
		  SELECT CASE entity->>'type'
		           WHEN 'character' THEN 0
		           WHEN 'corporation' THEN 1
		           WHEN 'alliance' THEN 2
		         END::smallint,
		         CASE WHEN entity->>'id' ~ '^[0-9]+$'
		                   AND CASE
		                     WHEN entity->>'id' ~ '^[0-9]+$'
		                     THEN (entity->>'id')::numeric
		                   END BETWEEN 1 AND 2147483647
		           THEN (entity->>'id')::integer END
		  FROM jsonb_array_elements($1::jsonb) entity
		  WHERE entity->>'type' IN (
		    'character', 'corporation', 'alliance'
		  )
		),
		pairs(entity_type, entity_id) AS (
		  SELECT entity_type, entity_id
		  FROM raw_pairs
		  WHERE entity_id > 0
		  UNION
		  SELECT 1::smallint, character.corporation_id
		  FROM raw_pairs raw
		  JOIN characters character
		    ON raw.entity_type = 0
		   AND character.character_id = raw.entity_id
		  WHERE character.corporation_id IS NOT NULL
		  UNION
		  SELECT 2::smallint, character.alliance_id
		  FROM raw_pairs raw
		  JOIN characters character
		    ON raw.entity_type = 0
		   AND character.character_id = raw.entity_id
		  WHERE character.alliance_id IS NOT NULL
		  UNION
		  SELECT 2::smallint, corporation.alliance_id
		  FROM raw_pairs raw
		  JOIN corporations corporation
		    ON raw.entity_type = 1
		   AND corporation.corporation_id = raw.entity_id
		  WHERE corporation.alliance_id IS NOT NULL
		)
		SELECT campaign.campaign_id, campaign.name, campaign.description,
		       campaign.visibility, campaign.status, campaign.start_time,
		       campaign.end_time, campaign.created_by_character_id,
		       campaign.estimated_killmails
		FROM campaigns campaign
		WHERE (
		  $3::boolean
		  OR campaign.visibility = 0
		  OR campaign.created_by_character_id = $2
		  OR (
		    campaign.visibility = 2
		    AND (
		      EXISTS (
		        SELECT 1
		        FROM campaign_side_entities side_entity
		        JOIN pairs pair
		          ON pair.entity_type = side_entity.entity_type
		         AND pair.entity_id = side_entity.entity_id
		        WHERE side_entity.campaign_id = campaign.campaign_id
		      )
		      OR EXISTS (
		        SELECT 1
		        FROM jsonb_array_elements(campaign.allowed_entities) allowed
		        JOIN pairs pair
		          ON allowed->>'type' = CASE pair.entity_type
		            WHEN 0 THEN 'character'
		            WHEN 1 THEN 'corporation'
		            WHEN 2 THEN 'alliance'
		          END
		         AND CASE
		               WHEN allowed->>'id' ~ '^[0-9]+$'
		               THEN (allowed->>'id')::numeric
		             END = pair.entity_id
		      )
		    )
		  )
		)
		  AND ($4 = '' OR campaign.campaign_id = $4
		       OR campaign.name ILIKE '%' || $4 || '%')
		  AND (NOT $5::boolean
		       OR campaign.created_by_character_id = $2)
		  AND ($6::text[] IS NULL
		       OR campaign.campaign_id = ANY($6::text[]))
		ORDER BY
		  CASE WHEN campaign.created_by_character_id = $2 THEN 0 ELSE 1 END,
		  campaign.created_at DESC
		LIMIT $7`,
		entities, principal.CharacterID, principal.IsAdmin, search,
		options.OwnOnly, ids, limit,
	)
	if err != nil {
		return nil, err
	}
	return emptyDomainRows(rows), nil
}

func domainEntitiesJSON(entities []domainEntity) []any {
	result := make([]any, 0, len(entities))
	for _, entity := range entities {
		result = append(result, map[string]any{
			"type": entity.Type, "id": entity.ID, "name": entity.Name,
		})
	}
	return result
}

func ensureDomainActivationAvailable(
	ctx context.Context,
	db domainQuerier,
	domain map[string]any,
) error {
	id := int32From(domain["id"])
	subdomain := fmt.Sprint(domain["subdomain"])
	customHostname, _ := domain["custom_hostname"].(string)
	row, err := domainQueryMap(ctx, db, `
		SELECT id
		FROM custom_domains
		WHERE id <> $1 AND active IS TRUE
		  AND (
		    subdomain = $2
		    OR ($3 <> '' AND LOWER(custom_hostname) = LOWER($3))
		  )
		LIMIT 1`, id, subdomain, customHostname)
	if err != nil {
		return err
	}
	if row != nil {
		return apiError(
			http.StatusConflict,
			"Another active domain already uses this hostname",
		)
	}
	return nil
}

func (s *domainService) loadAdminDomains(
	ctx context.Context,
) ([]map[string]any, error) {
	rows, err := domainQueryMaps(ctx, s.db, `
		SELECT domain.id, domain.subdomain, domain.custom_hostname,
		       domain.user_id, domain.entities, domain.theme,
		       domain.site_name, domain.active, domain.created_at,
		       domain.updated_at, owner.character_name AS owner_name,
		       (
		         SELECT COUNT(*)::integer
		         FROM domain_assets asset
		         WHERE asset.domain_id = domain.id
		           AND asset.status = 'pending'
		       ) AS pending_assets
		FROM custom_domains domain
		LEFT JOIN users owner ON owner.character_id = domain.user_id
		ORDER BY domain.created_at DESC`)
	if err != nil {
		return nil, err
	}
	for _, row := range rows {
		row["entities"] = jsonArray(row["entities"])
		row["theme"] = jsonObject(row["theme"])
	}
	return emptyDomainRows(rows), nil
}

func (s *domainService) loadAdminDomain(
	ctx context.Context,
	id int32,
) (map[string]any, []map[string]any, error) {
	domain, err := domainQueryMap(ctx, s.db, `
		SELECT domain.id, domain.subdomain, domain.custom_hostname,
		       domain.user_id, domain.entities, domain.theme,
		       domain.navbar_links, domain.site_name,
		       domain.site_description, domain.active,
		       domain.created_at, domain.updated_at,
		       owner.character_name AS owner_name
		FROM custom_domains domain
		LEFT JOIN users owner ON owner.character_id = domain.user_id
		WHERE domain.id = $1
		LIMIT 1`, id)
	if err != nil || domain == nil {
		return domain, []map[string]any{}, err
	}
	domain["entities"] = jsonArray(domain["entities"])
	domain["theme"] = jsonObject(domain["theme"])
	domain["navbar_links"] = jsonArray(domain["navbar_links"])
	assets, err := domainQueryMaps(ctx, s.db, `
		SELECT id, domain_id, type, status, storage_key, content_type,
		       file_size, uploaded_by, reviewed_by, reviewed_at,
		       reject_reason, created_at, file_hash
		FROM domain_assets
		WHERE domain_id = $1
		ORDER BY created_at, id`, id)
	return domain, emptyDomainRows(assets), err
}

func (s *domainService) toggleDomain(
	ctx context.Context,
	id int32,
) (map[string]any, error) {
	tx, err := s.mutations.Begin(ctx)
	if err != nil {
		return nil, err
	}
	defer func() { _ = tx.Rollback(ctx) }()
	row, err := domainQueryMap(ctx, tx, `
		SELECT `+domainColumns+`
		FROM custom_domains domain
		WHERE domain.id = $1
		LIMIT 1
		FOR UPDATE`, id)
	if err != nil {
		return nil, err
	}
	if row == nil {
		return nil, apiError(http.StatusNotFound, "Domain not found")
	}
	if !boolFrom(row["active"]) {
		if err := ensureDomainActivationAvailable(ctx, tx, row); err != nil {
			return nil, err
		}
	}
	updated, err := domainQueryMap(ctx, tx, `
		UPDATE custom_domains domain
		SET active = NOT active, updated_at = $2
		WHERE domain.id = $1
		RETURNING domain.id, domain.subdomain, domain.custom_hostname,
		          domain.user_id, domain.entities, domain.theme,
		          domain.navbar_links, domain.site_name,
		          domain.site_description, domain.active,
		          domain.created_at, domain.updated_at, domain.widgets,
		          domain.campaign_policy`, id, s.now().UTC())
	if err != nil {
		return nil, domainMutationError(err)
	}
	if err := tx.Commit(ctx); err != nil {
		return nil, domainMutationError(err)
	}
	return normalizeDomainRecord(updated), nil
}

func domainMutationError(err error) error {
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) && pgErr.Code == "23505" {
		return apiError(
			http.StatusConflict,
			"A domain with this hostname already exists",
		)
	}
	return err
}
