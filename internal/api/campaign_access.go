package api

import (
	"context"
	"fmt"
	"net/http"
)

type campaignDomain struct {
	ID                int32
	UserID            int32
	Entities          []campaignDomainEntity
	CampaignPolicy    int16
	SelectedIDs       []string
	PublicSelectedIDs []string
}

type campaignDomainEntity struct {
	Type string
	ID   int32
}

func loadCampaignDomain(
	ctx context.Context,
	db Database,
	req *legacyRequest,
) (*campaignDomain, error) {
	host := conflictRequestHost(req)
	lookup, value, domainHost := customDomainHostQuery(host)
	if !domainHost || lookup == "" {
		return nil, nil
	}

	row, err := queryMap(ctx, db, fmt.Sprintf(`
		SELECT domain.id, domain.user_id, domain.entities,
		       domain.campaign_policy,
		       COALESCE(
		         array_agg(selection.campaign_id ORDER BY selection.created_at)
		           FILTER (WHERE selection.campaign_id IS NOT NULL),
		         '{}'::text[]
		       ) AS selected_ids,
		       COALESCE(
		         array_agg(selection.campaign_id ORDER BY selection.created_at)
		           FILTER (
		             WHERE selection.campaign_id IS NOT NULL
		               AND selection.public_on_domain
		           ),
		         '{}'::text[]
		       ) AS public_selected_ids
		FROM custom_domains domain
		LEFT JOIN custom_domain_campaigns selection
		  ON selection.domain_id = domain.id
		WHERE domain.active IS TRUE AND %s
		GROUP BY domain.id
		LIMIT 1`, lookup), value)
	if err != nil || row == nil {
		return nil, err
	}
	domain := &campaignDomain{
		ID:                int32From(row["id"]),
		UserID:            int32From(row["user_id"]),
		CampaignPolicy:    int16From(row["campaign_policy"]),
		SelectedIDs:       stringSlice(row["selected_ids"]),
		PublicSelectedIDs: stringSlice(row["public_selected_ids"]),
	}
	for _, raw := range jsonArray(row["entities"]) {
		entity := jsonObject(raw)
		entityType, _ := stringValue(entity["type"])
		entityID := int32From(entity["id"])
		if entityID <= 0 ||
			entityType != "character" &&
				entityType != "corporation" &&
				entityType != "alliance" {
			continue
		}
		domain.Entities = append(domain.Entities, campaignDomainEntity{
			Type: entityType, ID: entityID,
		})
	}
	return domain, nil
}

func campaignDomainMember(
	principal *Principal,
	domain *campaignDomain,
) bool {
	if principal == nil || domain == nil {
		return false
	}
	if principal.IsAdmin || principal.CharacterID == domain.UserID {
		return true
	}
	for _, entity := range domain.Entities {
		switch entity.Type {
		case "character":
			if entity.ID == principal.CharacterID {
				return true
			}
		case "corporation":
			if principal.CorporationID != nil &&
				entity.ID == *principal.CorporationID {
				return true
			}
		case "alliance":
			if principal.AllianceID != nil &&
				entity.ID == *principal.AllianceID {
				return true
			}
		}
	}
	return false
}

func requireCampaignView(
	ctx context.Context,
	db Database,
	req *legacyRequest,
	campaign map[string]any,
	principal *Principal,
) error {
	if int16From(campaign["visibility"]) != campaignVisibilityPrivate {
		return nil
	}
	creatorID := int32From(campaign["created_by_character_id"])
	if principal != nil &&
		(principal.IsAdmin || principal.CharacterID == creatorID) {
		return nil
	}
	domain, err := loadCampaignDomain(ctx, db, req)
	if err != nil {
		return err
	}
	if domain == nil {
		return apiError(http.StatusNotFound, "Campaign not found")
	}
	row, err := queryMap(ctx, db, `
		SELECT public_on_domain
		FROM custom_domain_campaigns
		WHERE domain_id = $1 AND campaign_id = $2
		LIMIT 1`, domain.ID, campaign["campaign_id"])
	if err != nil {
		return err
	}
	if row == nil {
		return apiError(http.StatusNotFound, "Campaign not found")
	}
	public, _ := boolValue(row["public_on_domain"])
	if !public && !campaignDomainMember(principal, domain) {
		return apiError(http.StatusNotFound, "Campaign not found")
	}
	return nil
}

func campaignListingScope(
	domain *campaignDomain,
	entityType string,
	entityID int32,
) (cte, condition string, args []any) {
	if entityType != "" && entityID > 0 {
		entityCode := int16(campaignengineEntityType(entityType))
		args = []any{entityCode, entityID}
		cte = `
			WITH raw_scope_pairs(entity_type, entity_id) AS (
			  SELECT $1::smallint, $2::integer
			),
			scope_pairs(entity_type, entity_id) AS (
			  SELECT entity_type, entity_id FROM raw_scope_pairs
			  UNION
			  SELECT 1::smallint, character.corporation_id
			  FROM raw_scope_pairs raw
			  JOIN characters character
			    ON raw.entity_type = 0
			   AND character.character_id = raw.entity_id
			  WHERE character.corporation_id IS NOT NULL
			  UNION
			  SELECT 2::smallint, character.alliance_id
			  FROM raw_scope_pairs raw
			  JOIN characters character
			    ON raw.entity_type = 0
			   AND character.character_id = raw.entity_id
			  WHERE character.alliance_id IS NOT NULL
			  UNION
			  SELECT 2::smallint, corporation.alliance_id
			  FROM raw_scope_pairs raw
			  JOIN corporations corporation
			    ON raw.entity_type = 1
			   AND corporation.corporation_id = raw.entity_id
			  WHERE corporation.alliance_id IS NOT NULL
			)
		`
		condition = `
			EXISTS (
			  SELECT 1
			  FROM campaign_side_entities entity
			  JOIN scope_pairs scope
			    ON scope.entity_type = entity.entity_type
			   AND scope.entity_id = entity.entity_id
			  WHERE entity.campaign_id = c.campaign_id
			)`
		return
	}
	if domain == nil {
		return "", "", nil
	}

	args = []any{domain.ID}
	if domain.CampaignPolicy == 1 {
		return "", `
			EXISTS (
			  SELECT 1 FROM custom_domain_campaigns selection
			  WHERE selection.domain_id = $1
			    AND selection.campaign_id = c.campaign_id
			)`, args
	}
	args = append(args, domain.UserID)
	cte = `
		WITH raw_scope_pairs(entity_type, entity_id) AS (
		  SELECT CASE entity->>'type'
		           WHEN 'character' THEN 0
		           WHEN 'corporation' THEN 1
		           WHEN 'alliance' THEN 2
		         END::smallint,
		         CASE WHEN entity->>'id' ~ '^[0-9]+$'
		           THEN (entity->>'id')::integer END
		  FROM custom_domains domain
		  CROSS JOIN LATERAL jsonb_array_elements(domain.entities) entity
		  WHERE domain.id = $1
		    AND entity->>'type' IN ('character', 'corporation', 'alliance')
		    AND entity->>'id' ~ '^[0-9]+$'
		),
		scope_pairs(entity_type, entity_id) AS (
		  SELECT entity_type, entity_id
		  FROM raw_scope_pairs
		  WHERE entity_id > 0
		  UNION
		  SELECT 1::smallint, character.corporation_id
		  FROM raw_scope_pairs raw
		  JOIN characters character
		    ON raw.entity_type = 0
		   AND character.character_id = raw.entity_id
		  WHERE character.corporation_id IS NOT NULL
		  UNION
		  SELECT 2::smallint, character.alliance_id
		  FROM raw_scope_pairs raw
		  JOIN characters character
		    ON raw.entity_type = 0
		   AND character.character_id = raw.entity_id
		  WHERE character.alliance_id IS NOT NULL
		  UNION
		  SELECT 2::smallint, corporation.alliance_id
		  FROM raw_scope_pairs raw
		  JOIN corporations corporation
		    ON raw.entity_type = 1
		   AND corporation.corporation_id = raw.entity_id
		  WHERE corporation.alliance_id IS NOT NULL
		)
	`
	condition = fmt.Sprintf(`(
		EXISTS (
		  SELECT 1 FROM custom_domain_campaigns selection
		  WHERE selection.domain_id = $1
		    AND selection.campaign_id = c.campaign_id
		)
		OR EXISTS (
		  SELECT 1
		  FROM campaign_side_entities entity
		  JOIN scope_pairs scope
		    ON scope.entity_type = entity.entity_type
		   AND scope.entity_id = entity.entity_id
		  WHERE entity.campaign_id = c.campaign_id
		)
		OR (
		  c.visibility = %d
		  AND (
		    c.created_by_character_id = $2
		    OR EXISTS (
		      SELECT 1
		      FROM jsonb_array_elements(
		        COALESCE(c.allowed_entities, '[]'::jsonb)
		      ) allowed
		      JOIN scope_pairs scope
		        ON CASE allowed->>'type'
		             WHEN 'character' THEN 0
		             WHEN 'corporation' THEN 1
		             WHEN 'alliance' THEN 2
		           END = scope.entity_type
		       AND CASE WHEN allowed->>'id' ~ '^[0-9]+$'
		             THEN (allowed->>'id')::integer END = scope.entity_id
		    )
		  )
		)
	)`, campaignVisibilityKillboard)
	return
}

func campaignengineEntityType(entityType string) int {
	switch entityType {
	case "character":
		return 0
	case "corporation":
		return 1
	case "alliance":
		return 2
	default:
		return -1
	}
}

func stringSlice(value any) []string {
	switch typed := value.(type) {
	case []string:
		return append([]string(nil), typed...)
	case []any:
		result := make([]string, 0, len(typed))
		for _, item := range typed {
			if text, ok := stringValue(item); ok {
				result = append(result, text)
			}
		}
		return result
	}
	return nil
}
