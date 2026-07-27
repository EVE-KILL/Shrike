package api

import (
	"context"
	"crypto/rand"
	"encoding/json"
	"fmt"
	"io"
	"math"
	"net/http"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/danielgtaylor/huma/v2"
	campaignengine "github.com/eve-kill/shrike/internal/campaign"
	"github.com/eve-kill/shrike/internal/queue"
	"github.com/jackc/pgx/v5/pgxpool"
)

const (
	campaignIDLength                = 14
	campaignMaximumNameLength       = 100
	campaignMaximumDescription      = 2000
	campaignMaximumSides            = 4
	campaignMaximumEntitiesPerSide  = 15
	campaignMaximumAllowedEntities  = 10
	campaignMaximumLivePerUser      = 20
	campaignMaximumCreatedPerHour   = 5
	campaignMaximumPublicOngoing    = 3
	campaignMaximumPublicWindowDays = 365
	campaignMaximumOtherWindowDays  = 365
	campaignMaximumFutureStartDays  = 30
	campaignListPageSize            = 24
	campaignAdminPageSize           = 40
	campaignMaximumPage             = 10000
	campaignWeeklyThreshold         = 180
	campaignBodyLimit               = 1 << 20

	campaignVisibilityPublic    = 0
	campaignVisibilityPrivate   = 1
	campaignVisibilityKillboard = 2

	campaignPrizeFunding = 0
	campaignPrizeReady   = 1
	campaignPrizePaid    = 2
)

const campaignIDAlphabet = "0123456789ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz"

var campaignPrizeMetricLabels = map[int16]string{
	0: "Most kills",
	1: "Most losses",
	2: "Most ISK destroyed",
	3: "Most ISK lost",
}

type campaignService struct {
	auth      *authService
	db        Database
	mutations MutationDatabase
	storeErr  error
	queue     *queue.Client
	queueErr  error
	now       func() time.Time
}

func newCampaignService(opts Options) *campaignService {
	service := &campaignService{
		auth: newAuthService(opts),
		db:   opts.DB,
		now:  time.Now,
	}
	service.mutations, service.storeErr = mutationDatabase(opts)
	if pool, ok := opts.DB.(*pgxpool.Pool); ok && pool != nil {
		service.queue, service.queueErr = queue.New(queue.Options{Pool: pool})
	} else {
		service.queueErr = fmt.Errorf("campaign queue is not configured")
	}
	return service
}

// registerCampaignRoutes installs campaign reads, owner mutations, prize
// actions, and administration in one domain group. Access level is expressed
// by each operation's security declaration rather than by a separate registry.
func registerCampaignRoutes(a huma.API, opts Options) {
	service := newCampaignService(opts)
	requiredSession := []map[string][]string{{"eveSession": {}}}
	optionalSession := []map[string][]string{{}, {"eveSession": {}}}

	registerLegacy(a, huma.Operation{
		OperationID: "campaigns",
		Method:      http.MethodGet,
		Path:        "/campaigns",
		Summary:     "Campaign listing",
		Tags:        []string{"campaigns"},
		Security:    optionalSession,
	}, service.campaignListHandler())

	registerLegacy(a, documentJSONBody[campaignCreateBody](a, huma.Operation{
		OperationID: "campaign-create",
		Method:      http.MethodPost,
		Path:        "/campaigns",
		Summary:     "Create a campaign",
		Tags:        []string{"account", "campaigns"},
		Security:    requiredSession,
	}), service.campaignCreateHandler())
	registerLegacy(a, documentJSONBody[campaignCreateBody](a, huma.Operation{
		OperationID: "campaign-create-legacy",
		Method:      http.MethodPost,
		Path:        "/campaign/create",
		Summary:     "Create a campaign",
		Tags:        []string{"account", "campaigns"},
		Security:    requiredSession,
	}), service.campaignCreateHandler())

	detail := service.campaignDetailHandler()
	for _, route := range []struct{ id, path string }{
		{"campaign-detail", "/campaigns/{id}"},
		{"campaign-detail-legacy", "/campaign/{id}"},
	} {
		registerLegacy(a, huma.Operation{
			OperationID: route.id,
			Method:      http.MethodGet,
			Path:        route.path,
			Summary:     "Campaign detail",
			Tags:        []string{"campaigns"},
			Security:    optionalSession,
		}, detail)
	}

	update := service.campaignUpdateHandler()
	for _, route := range []struct{ id, method, path string }{
		{"campaign-update", http.MethodPatch, "/campaigns/{id}"},
		{"campaign-update-legacy", http.MethodPatch, "/campaign/{id}"},
		{"campaign-update-browser-legacy", http.MethodPost, "/campaign/{id}/update"},
	} {
		registerLegacy(a, documentJSONBody[campaignUpdateBody](a, huma.Operation{
			OperationID: route.id,
			Method:      route.method,
			Path:        route.path,
			Summary:     "Update a campaign",
			Tags:        []string{"account", "campaigns"},
			Security:    requiredSession,
		}), update)
	}

	remove := service.campaignDeleteHandler()
	for _, route := range []struct{ id, path string }{
		{"campaign-delete", "/campaigns/{id}"},
		{"campaign-delete-legacy", "/campaign/{id}"},
	} {
		registerLegacy(a, huma.Operation{
			OperationID: route.id,
			Method:      http.MethodDelete,
			Path:        route.path,
			Summary:     "Delete an unfunded campaign",
			Tags:        []string{"account", "campaigns"},
			Security:    requiredSession,
		}, remove)
	}

	killlist := service.campaignKilllistHandler()
	for _, route := range []struct{ id, path string }{
		{"campaign-killmails", "/campaigns/{id}/killmails"},
		{"campaign-killlist-legacy", "/campaign/{id}/killlist"},
	} {
		registerLegacy(a, huma.Operation{
			OperationID: route.id,
			Method:      http.MethodGet,
			Path:        route.path,
			Summary:     "Campaign killmail feed",
			Tags:        []string{"campaigns", "killmails"},
			Security:    optionalSession,
		}, killlist)
	}

	contribute := service.campaignContributeHandler()
	for _, route := range []struct{ id, path string }{
		{"campaign-prize-contribute", "/campaigns/{id}/prizes/contributions"},
		{"campaign-prize-contribute-legacy", "/campaign/{id}/prize/contribute"},
	} {
		registerLegacy(a, documentJSONBody[campaignContributeBody](a, huma.Operation{
			OperationID: route.id,
			Method:      http.MethodPost,
			Path:        route.path,
			Summary:     "Fund a campaign prize pool from EK Wallet",
			Tags:        []string{"account", "campaigns", "wallet"},
			Security:    requiredSession,
		}), contribute)
	}

	claim := service.campaignClaimHandler()
	for _, route := range []struct{ id, path string }{
		{"campaign-prize-claim", "/campaigns/{id}/prizes/claim"},
		{"campaign-prize-claim-legacy", "/campaign/{id}/prize/claim"},
	} {
		registerLegacy(a, huma.Operation{
			OperationID: route.id,
			Method:      http.MethodPost,
			Path:        route.path,
			Summary:     "Claim a campaign prize",
			Tags:        []string{"account", "campaigns", "wallet"},
			Security:    requiredSession,
		}, claim)
	}

	registerLegacy(a, huma.Operation{
		OperationID: "campaign-admin-list",
		Method:      http.MethodGet,
		Path:        "/admin/campaigns",
		Summary:     "Campaign administration",
		Tags:        []string{"admin", "campaigns"},
		Security:    requiredSession,
	}, service.adminCampaignListHandler())

	adminAction := service.adminCampaignActionHandler()
	for _, route := range []struct{ id, path string }{
		{"campaign-admin-action", "/admin/campaigns/{id}/actions"},
		{"campaign-admin-action-legacy", "/admin/campaigns/{id}/action"},
	} {
		registerLegacy(a, documentJSONBody[campaignActionBody](a, huma.Operation{
			OperationID: route.id,
			Method:      http.MethodPost,
			Path:        route.path,
			Summary:     "Apply an administrative campaign action",
			Tags:        []string{"admin", "campaigns"},
			Security:    requiredSession,
		}), adminAction)
	}

	markPaid := service.adminCampaignPrizePaidHandler()
	for _, route := range []struct{ id, path string }{
		{
			"campaign-prize-paid",
			"/admin/campaigns/{id}/prizes/{rank}/payment",
		},
		{
			"campaign-prize-paid-legacy",
			"/admin/campaign-prizes/{id}/{rank}/paid",
		},
	} {
		registerLegacy(a, documentJSONBody[campaignPrizePaidBody](a, huma.Operation{
			OperationID: route.id,
			Method:      http.MethodPost,
			Path:        route.path,
			Summary:     "Mark a claimed campaign prize paid",
			Tags:        []string{"admin", "campaigns", "wallet"},
			Security:    requiredSession,
		}), markPaid)
	}
}

func (s *campaignService) campaignListHandler() legacyHandler {
	return func(ctx context.Context, req *legacyRequest) (legacyPayload, error) {
		principal, err := s.auth.resolvePrincipal(ctx, req)
		if err != nil {
			return legacyPayload{}, err
		}
		domain, err := loadCampaignDomain(ctx, s.db, req)
		if err != nil {
			return legacyPayload{}, err
		}
		page := boundedCampaignPage(req.Query.Get("page"))
		entityType := strings.TrimSpace(req.Query.Get("entityType"))
		entityID, _ := strconv.ParseInt(req.Query.Get("entityId"), 10, 32)
		entityScoped := entityType != "" && entityID > 0
		if entityScoped &&
			entityType != "character" &&
			entityType != "corporation" &&
			entityType != "alliance" {
			return legacyPayload{}, apiError(
				http.StatusBadRequest, "Invalid entityType",
			)
		}

		status := strings.TrimSpace(req.Query.Get("status"))
		if status == "" {
			if entityScoped {
				status = "all"
			} else {
				status = "active"
			}
		}
		search := truncateRunes(strings.TrimSpace(req.Query.Get("q")), 100)
		mode := strings.TrimSpace(req.Query.Get("mode"))
		viewerID := int32(0)
		if principal != nil {
			viewerID = principal.CharacterID
		}

		scopeCTE, scopeCondition, scopeArgs := campaignListingScope(
			domain, entityType, int32(entityID),
		)
		args := append([]any(nil), scopeArgs...)
		arg := func(value any) string {
			args = append(args, value)
			return fmt.Sprintf("$%d", len(args))
		}

		viewerPlaceholder := arg(viewerID)
		var visibility string
		if domain == nil {
			visibility = fmt.Sprintf(
				"(c.visibility = %d OR c.created_by_character_id = %s)",
				campaignVisibilityPublic, viewerPlaceholder,
			)
		} else {
			domainID := arg(domain.ID)
			privateAccess := "selection.public_on_domain = true"
			if campaignDomainMember(principal, domain) {
				privateAccess = "true"
			}
			visibility = fmt.Sprintf(`(
				c.visibility IN (%d, %d)
				OR c.created_by_character_id = %s
				OR (
					c.visibility = %d
					AND EXISTS (
						SELECT 1 FROM custom_domain_campaigns selection
						WHERE selection.domain_id = %s
						  AND selection.campaign_id = c.campaign_id
						  AND %s
					)
				)
			)`,
				campaignVisibilityPublic,
				campaignVisibilityKillboard,
				viewerPlaceholder,
				campaignVisibilityPrivate,
				domainID,
				privateAccess,
			)
		}

		filters := []string{visibility}
		if scopeCondition != "" {
			filters = append(filters, scopeCondition)
		}
		if search != "" {
			filters = append(filters, "c.name ILIKE "+arg("%"+escapeLike(search)+"%")+" ESCAPE '\\'")
		}
		switch mode {
		case "conflict":
			filters = append(filters,
				"EXISTS (SELECT 1 FROM campaign_sides side WHERE side.campaign_id = c.campaign_id)")
		case "area":
			filters = append(filters,
				"NOT EXISTS (SELECT 1 FROM campaign_sides side WHERE side.campaign_id = c.campaign_id)")
		}
		where := strings.Join(filters, " AND ")
		rowWhere := where
		if status == "private" {
			rowWhere += fmt.Sprintf(
				" AND c.visibility = %d AND c.created_by_character_id = %s",
				campaignVisibilityPrivate, viewerPlaceholder,
			)
		}

		countSQL := scopeCTE + fmt.Sprintf(`
			SELECT
				COUNT(*) FILTER (
					WHERE c.status IN (%d, %d)
				)::bigint AS active,
				COUNT(*) FILTER (
					WHERE c.status = %d
				)::bigint AS archived,
				COUNT(*) FILTER (
					WHERE c.visibility = %d
					  AND c.created_by_character_id = %s
				)::bigint AS private
			FROM campaigns c
			WHERE %s`,
			campaignengine.StatusPending,
			campaignengine.StatusActive,
			campaignengine.StatusArchived,
			campaignVisibilityPrivate,
			viewerPlaceholder,
			where,
		)
		countRows, err := queryMaps(ctx, s.db, countSQL, args...)
		if err != nil {
			return legacyPayload{}, err
		}
		counts := campaignCounts(firstRow(countRows))
		total := counts["active"]
		switch status {
		case "archived":
			total = counts["archived"]
		case "private":
			total = counts["private"]
		case "all":
			total = counts["active"] + counts["archived"]
		}

		var statuses []int16
		switch status {
		case "archived":
			statuses = []int16{campaignengine.StatusArchived}
		case "all", "private":
			statuses = []int16{
				campaignengine.StatusPending,
				campaignengine.StatusActive,
				campaignengine.StatusArchived,
			}
		default:
			statuses = []int16{
				campaignengine.StatusPending,
				campaignengine.StatusActive,
			}
		}
		statusPlaceholder := arg(statuses)
		limitPlaceholder := arg(campaignListPageSize + 1)
		offsetPlaceholder := arg((page - 1) * campaignListPageSize)
		rows, err := queryMaps(ctx, s.db, scopeCTE+fmt.Sprintf(`
			SELECT c.campaign_id, c.name, c.description, c.status,
			       c.visibility, c.start_time, c.end_time, c.location,
			       c.last_activity_at, c.stats_updated_at, c.created_at,
			       c.created_by_character_id,
			       creator.name AS creator_name,
			       c.stats->'totals' AS totals,
			       COALESCE((
			          SELECT jsonb_agg(
			            jsonb_build_object(
			              'side_index', side.side_index,
			              'name', side.name,
			              'kills', side.kills,
			              'losses', side.losses,
			              'isk_destroyed', side.isk_destroyed,
			              'isk_lost', side.isk_lost,
			              'palette', dominant.palette,
			              'entities', COALESCE((
			                SELECT jsonb_agg(
			                  jsonb_build_object(
			                    'entity_type', entity.entity_type,
			                    'entity_id', entity.entity_id
			                  ) ORDER BY entity.kills DESC, entity.id
			                )
			                FROM campaign_side_entities entity
			                WHERE entity.campaign_id = side.campaign_id
			                  AND entity.side_index = side.side_index
			              ), '[]'::jsonb)
			            ) ORDER BY side.side_index
			          )
			          FROM campaign_sides side
			          LEFT JOIN LATERAL (
			            SELECT COALESCE(
			              direct_corporation.palette,
			              character_corporation.palette,
			              executor_corporation.palette
			            ) AS palette
			            FROM campaign_side_entities entity
			            LEFT JOIN characters character
			              ON entity.entity_type = 0
			             AND character.character_id = entity.entity_id
			            LEFT JOIN corporations character_corporation
			              ON character_corporation.corporation_id = character.corporation_id
			            LEFT JOIN corporations direct_corporation
			              ON entity.entity_type = 1
			             AND direct_corporation.corporation_id = entity.entity_id
			            LEFT JOIN alliances alliance
			              ON entity.entity_type = 2
			             AND alliance.alliance_id = entity.entity_id
			            LEFT JOIN corporations executor_corporation
			              ON executor_corporation.corporation_id =
			                 alliance.executor_corporation_id
			            WHERE entity.campaign_id = side.campaign_id
			              AND entity.side_index = side.side_index
			            ORDER BY entity.kills DESC, entity.losses DESC, entity.id
			            LIMIT 1
			          ) dominant ON true
			          WHERE side.campaign_id = c.campaign_id
			       ), '[]'::jsonb) AS sides
			FROM campaigns c
			LEFT JOIN characters creator
			  ON creator.character_id = c.created_by_character_id
			WHERE %s
			  AND c.status = ANY(%s::smallint[])
			ORDER BY c.last_activity_at DESC NULLS LAST, c.created_at DESC
			LIMIT %s OFFSET %s`,
			rowWhere,
			statusPlaceholder,
			limitPlaceholder,
			offsetPlaceholder,
		), args...)
		if err != nil {
			return legacyPayload{}, err
		}
		hasMore := len(rows) > campaignListPageSize
		if hasMore {
			rows = rows[:campaignListPageSize]
		}
		for _, row := range rows {
			sides := jsonArray(row["sides"])
			row["mode"] = "conflict"
			if len(sides) == 0 {
				row["mode"] = "area"
			}
			row["sides"] = sides
			row["creator"] = map[string]any{
				"character_id": row["created_by_character_id"],
				"name":         row["creator_name"],
			}
			delete(row, "created_by_character_id")
			delete(row, "creator_name")
			if row["totals"] == nil {
				row["totals"] = nil
			}
		}
		return jsonPayload(map[string]any{
			"campaigns": rows,
			"hasMore":   hasMore,
			"page":      page,
			"total":     total,
			"counts":    counts,
		}), nil
	}
}

func (s *campaignService) campaignDetailHandler() legacyHandler {
	return func(ctx context.Context, req *legacyRequest) (legacyPayload, error) {
		id, err := campaignID(req.Param("id"))
		if err != nil {
			return legacyPayload{}, err
		}
		campaignRow, err := queryMap(ctx, s.db, `
			SELECT c.*,
			       creator.name AS creator_name
			FROM campaigns c
			LEFT JOIN characters creator
			  ON creator.character_id = c.created_by_character_id
			WHERE c.campaign_id = $1
			LIMIT 1`, id)
		if err != nil {
			return legacyPayload{}, err
		}
		if campaignRow == nil {
			return legacyPayload{}, apiError(
				http.StatusNotFound, "Campaign not found",
			)
		}
		principal, err := s.auth.resolvePrincipal(ctx, req)
		if err != nil {
			return legacyPayload{}, err
		}
		if err := requireCampaignView(
			ctx, s.db, req, campaignRow, principal,
		); err != nil {
			return legacyPayload{}, err
		}

		location := campaignLocationFrom(campaignRow["location"])
		results, err := queryMapsConcurrent(ctx, s.db,
			databaseQuery{
				SQL: `
					SELECT side.side_index, side.name, side.kills, side.losses,
					       side.isk_destroyed, side.isk_lost,
					       dominant.palette
					FROM campaign_sides side
					LEFT JOIN LATERAL (
					  SELECT COALESCE(
					    direct_corporation.palette,
					    character_corporation.palette,
					    executor_corporation.palette
					  ) AS palette
					  FROM campaign_side_entities entity
					  LEFT JOIN characters character
					    ON entity.entity_type = 0
					   AND character.character_id = entity.entity_id
					  LEFT JOIN corporations character_corporation
					    ON character_corporation.corporation_id =
					       character.corporation_id
					  LEFT JOIN corporations direct_corporation
					    ON entity.entity_type = 1
					   AND direct_corporation.corporation_id = entity.entity_id
					  LEFT JOIN alliances alliance
					    ON entity.entity_type = 2
					   AND alliance.alliance_id = entity.entity_id
					  LEFT JOIN corporations executor_corporation
					    ON executor_corporation.corporation_id =
					       alliance.executor_corporation_id
					  WHERE entity.campaign_id = side.campaign_id
					    AND entity.side_index = side.side_index
					  ORDER BY entity.kills DESC, entity.losses DESC, entity.id
					  LIMIT 1
					) dominant ON true
					WHERE side.campaign_id = $1
					ORDER BY side.side_index`,
				Args: []any{id},
			},
			databaseQuery{
				SQL: `
					SELECT entity.id, entity.side_index, entity.entity_type,
					       entity.entity_id, entity.kills, entity.losses,
					       entity.isk_destroyed, entity.isk_lost,
					       CASE entity.entity_type
					         WHEN 0 THEN character.name
					         WHEN 1 THEN corporation.name
					         WHEN 2 THEN alliance.name
					       END AS name
					FROM campaign_side_entities entity
					LEFT JOIN characters character
					  ON entity.entity_type = 0
					 AND character.character_id = entity.entity_id
					LEFT JOIN corporations corporation
					  ON entity.entity_type = 1
					 AND corporation.corporation_id = entity.entity_id
					LEFT JOIN alliances alliance
					  ON entity.entity_type = 2
					 AND alliance.alliance_id = entity.entity_id
					WHERE entity.campaign_id = $1
					ORDER BY entity.side_index, entity.kills DESC,
					         entity.losses DESC, entity.id`,
				Args: []any{id},
			},
			databaseQuery{
				SQL: `
					SELECT side_index, day, kills, losses,
					       isk_destroyed, isk_lost
					FROM campaign_stats_daily
					WHERE campaign_id = $1
					ORDER BY day, side_index`,
				Args: []any{id},
			},
			databaseQuery{
				SQL: `
					SELECT campaign_id, metric, winner_count,
					       payout_percentages, status,
					       funded_total::text AS funded_total,
					       rules_locked_at, finalized_at, created_at, updated_at
					FROM campaign_prize_pools
					WHERE campaign_id = $1 LIMIT 1`,
				Args: []any{id},
			},
			databaseQuery{
				SQL: `
					SELECT rank, character_id, character_name,
					       metric_value::text AS metric_value,
					       secondary_value::text AS secondary_value,
					       payout_percentage,
					       payout_amount::text AS payout_amount,
					       claimed_at, paid_at
					FROM campaign_prize_results
					WHERE campaign_id = $1
					ORDER BY rank`,
				Args: []any{id},
			},
			databaseQuery{
				SQL: `
					WITH contributions AS (
					  SELECT 'ek_wallet'::text AS source,
					         'ek:' || wallet_tx.id::text AS contribution_id,
					         wallet_tx.character_id::bigint AS contributor_id,
					         COALESCE(
					           character.name,
					           'Character ' || wallet_tx.character_id::text
					         ) AS contributor_name,
					         'character'::text AS contributor_type,
					         (-wallet_tx.amount -
					           COALESCE(refunds.amount, 0))::text AS amount,
					         wallet_tx.created_at AS contributed_at
					  FROM ek_wallet_transactions wallet_tx
					  LEFT JOIN characters character
					    ON character.character_id = wallet_tx.character_id
					  LEFT JOIN (
					    SELECT related_transaction_id, SUM(amount) AS amount
					    FROM ek_wallet_transactions
					    WHERE type = 2 AND related_transaction_id IS NOT NULL
					    GROUP BY related_transaction_id
					  ) refunds ON refunds.related_transaction_id = wallet_tx.id
					  WHERE wallet_tx.type = 3
					    AND wallet_tx.campaign_id = $1
					    AND -wallet_tx.amount > COALESCE(refunds.amount, 0)

					  UNION ALL

					  SELECT 'eve_wallet'::text AS source,
					         'eve:' || reference.division::text || ':' ||
					           reference.journal_id::text AS contribution_id,
					         journal.first_party_id AS contributor_id,
					         COALESCE(
					           character.name, corporation.name, alliance.name,
					           CASE WHEN journal.first_party_id IS NOT NULL
					             THEN 'Entity ' || journal.first_party_id::text
					             ELSE 'Anonymous' END
					         ) AS contributor_name,
					         CASE
					           WHEN character.character_id IS NOT NULL THEN 'character'
					           WHEN corporation.corporation_id IS NOT NULL THEN 'corporation'
					           WHEN alliance.alliance_id IS NOT NULL THEN 'alliance'
					           ELSE 'unknown'
					         END AS contributor_type,
					         reference.amount::text AS amount,
					         journal.date AS contributed_at
					  FROM wallet_journal_references reference
					  JOIN corporation_wallet_journal journal
					    ON journal.corporation_id = reference.corporation_id
					   AND journal.division = reference.division
					   AND journal.journal_id = reference.journal_id
					  LEFT JOIN characters character
					    ON character.character_id = journal.first_party_id
					  LEFT JOIN corporations corporation
					    ON corporation.corporation_id = journal.first_party_id
					  LEFT JOIN alliances alliance
					    ON alliance.alliance_id = journal.first_party_id
					  WHERE reference.reference_type = 'campaign'
					    AND reference.reference_id = $1
					    AND reference.status = 0
					)
					SELECT *, COUNT(*) OVER ()::int AS contribution_count
					FROM contributions
					ORDER BY contributed_at DESC, contribution_id DESC
					LIMIT 100`,
				Args: []any{id},
			},
			databaseQuery{
				SQL: `
					SELECT last_journal_sync
					FROM corporation_wallet_tokens
					WHERE corporation_id = $1 LIMIT 1`,
				Args: []any{campaignengine.EveKillCorporationID},
			},
			databaseQuery{
				SQL: `
					SELECT solar_system_id AS id, system_name AS name
					FROM solar_systems
					WHERE solar_system_id = ANY($1::int[])
					ORDER BY system_name`,
				Args: []any{location.SystemIDs},
			},
			databaseQuery{
				SQL: `
					SELECT constellation_id AS id, constellation_name AS name
					FROM constellations
					WHERE constellation_id = ANY($1::int[])
					ORDER BY constellation_name`,
				Args: []any{location.ConstellationIDs},
			},
			databaseQuery{
				SQL: `
					SELECT region_id AS id, name
					FROM regions
					WHERE region_id = ANY($1::int[])
					ORDER BY name`,
				Args: []any{location.RegionIDs},
			},
		)
		if err != nil {
			return legacyPayload{}, err
		}

		canManage := principal != nil &&
			(principal.IsAdmin ||
				int32From(campaignRow["created_by_character_id"]) ==
					principal.CharacterID)
		sides := buildCampaignSides(results[0], results[1])
		daily := buildCampaignDaily(results[2])
		prize := buildCampaignPrize(
			campaignRow,
			results[3],
			results[4],
			results[5],
			results[6],
			principal,
			s.now().UTC(),
		)
		processing := map[string]any{
			"paused": boolFrom(campaignRow["processing_paused"]),
			"note":   nil, "estimated_killmails": nil, "last_started_at": nil,
			"last_duration_ms": nil, "last_killmails": nil, "last_error": nil,
		}
		if canManage {
			processing["note"] = campaignRow["processing_note"]
			processing["estimated_killmails"] = campaignRow["estimated_killmails"]
			processing["last_started_at"] = campaignRow["last_processing_started_at"]
			processing["last_duration_ms"] = campaignRow["last_processing_duration_ms"]
			processing["last_killmails"] = campaignRow["last_processing_killmails"]
			processing["last_error"] = campaignRow["last_processing_error"]
		}
		return jsonPayload(map[string]any{
			"campaign_id":      campaignRow["campaign_id"],
			"mode":             campaignMode(sides),
			"name":             campaignRow["name"],
			"description":      campaignRow["description"],
			"status":           campaignRow["status"],
			"visibility":       campaignRow["visibility"],
			"allowed_entities": jsonArray(campaignRow["allowed_entities"]),
			"start_time":       campaignRow["start_time"],
			"end_time":         campaignRow["end_time"],
			"location":         campaignRow["location"],
			"location_details": map[string]any{
				"systems":        results[7],
				"constellations": results[8],
				"regions":        results[9],
			},
			"stats":             campaignRow["stats"],
			"processed_through": campaignRow["processed_through"],
			"last_activity_at":  campaignRow["last_activity_at"],
			"stats_updated_at":  campaignRow["stats_updated_at"],
			"processing":        processing,
			"created_at":        campaignRow["created_at"],
			"creator": map[string]any{
				"character_id": campaignRow["created_by_character_id"],
				"name":         campaignRow["creator_name"],
			},
			"prize_pool": prize,
			"sides":      sides,
			"daily":      daily,
		}), nil
	}
}

func (s *campaignService) campaignKilllistHandler() legacyHandler {
	return func(ctx context.Context, req *legacyRequest) (legacyPayload, error) {
		id, err := campaignID(req.Param("id"))
		if err != nil {
			return legacyPayload{}, err
		}
		campaignRow, err := queryMap(ctx, s.db, `
			SELECT campaign_id, visibility, created_by_character_id,
			       start_time, end_time, location
			FROM campaigns WHERE campaign_id = $1 LIMIT 1`, id)
		if err != nil {
			return legacyPayload{}, err
		}
		if campaignRow == nil {
			return legacyPayload{}, apiError(
				http.StatusNotFound, "Campaign not found",
			)
		}
		principal, err := s.auth.resolvePrincipal(ctx, req)
		if err != nil {
			return legacyPayload{}, err
		}
		if err := requireCampaignView(
			ctx, s.db, req, campaignRow, principal,
		); err != nil {
			return legacyPayload{}, err
		}
		entityRows, err := queryMaps(ctx, s.db, `
			SELECT side_index, entity_type, entity_id
			FROM campaign_side_entities
			WHERE campaign_id = $1`, id)
		if err != nil {
			return legacyPayload{}, err
		}
		var side *int16
		if raw := strings.TrimSpace(req.Query.Get("side")); raw != "" {
			value, parseErr := strconv.ParseInt(raw, 10, 16)
			if parseErr != nil {
				return legacyPayload{}, apiError(
					http.StatusBadRequest, "Invalid campaign side",
				)
			}
			parsed := int16(value)
			side = &parsed
			found := false
			for _, entity := range entityRows {
				if int16From(entity["side_index"]) == parsed {
					found = true
					break
				}
			}
			if !found {
				return legacyPayload{}, apiError(
					http.StatusBadRequest, "Invalid campaign side",
				)
			}
		}
		limit := campaignKilllistLimit(req.Query.Get("limit"))
		after := positiveInt4(req.Query.Get("after"))
		end := s.now().UTC()
		if value, ok := timeFrom(campaignRow["end_time"]); ok && value.Before(end) {
			end = value
		}
		start, ok := timeFrom(campaignRow["start_time"])
		if !ok || !start.Before(end) {
			return jsonPayload(map[string]any{
				"kills": []any{}, "hasMore": false, "cursor": nil,
			}), nil
		}
		location := campaignLocationFrom(campaignRow["location"])
		where, args := campaignKilllistConditions(
			entityRows, side, start, end, location,
		)
		if after != nil {
			args = append(args, *after)
			where = append(where, fmt.Sprintf(
				"k.killmail_id < $%d", len(args),
			))
		}
		payload, err := loadCampaignKilllistPage(
			ctx, s.db, where, args, limit,
		)
		if err != nil {
			return legacyPayload{}, err
		}
		return payload, nil
	}
}

func campaignMode(sides []map[string]any) string {
	if len(sides) == 0 {
		return "area"
	}
	return "conflict"
}

func campaignCounts(row map[string]any) map[string]int64 {
	return map[string]int64{
		"active":   int64From(row["active"]),
		"archived": int64From(row["archived"]),
		"private":  int64From(row["private"]),
	}
}

func boundedCampaignPage(raw string) int {
	value, err := strconv.ParseFloat(strings.TrimSpace(raw), 64)
	if err != nil || math.IsNaN(value) || math.IsInf(value, 0) || value < 1 {
		return 1
	}
	page := int(math.Floor(value))
	if page > campaignMaximumPage {
		return campaignMaximumPage
	}
	return page
}

func campaignID(raw string) (string, error) {
	if len(raw) != campaignIDLength {
		return "", apiError(http.StatusBadRequest, "Invalid campaign ID")
	}
	for _, character := range raw {
		if character >= '0' && character <= '9' ||
			character >= 'A' && character <= 'Z' ||
			character >= 'a' && character <= 'z' {
			continue
		}
		return "", apiError(http.StatusBadRequest, "Invalid campaign ID")
	}
	return raw, nil
}

func generateCampaignID() (string, error) {
	random := make([]byte, campaignIDLength)
	if _, err := io.ReadFull(rand.Reader, random); err != nil {
		return "", err
	}
	result := make([]byte, campaignIDLength)
	for index, value := range random {
		result[index] = campaignIDAlphabet[int(value)%len(campaignIDAlphabet)]
	}
	return string(result), nil
}

func escapeLike(value string) string {
	replacer := strings.NewReplacer(`\`, `\\`, `%`, `\%`, `_`, `\_`)
	return replacer.Replace(value)
}

func int16From(value any) int16 {
	number, _ := int64Value(value)
	return int16(number)
}

func int32From(value any) int32 {
	number, _ := int64Value(value)
	return int32(number)
}

func int64From(value any) int64 {
	number, _ := int64Value(value)
	return number
}

func boolFrom(value any) bool {
	result, _ := boolValue(value)
	return result
}

func timeFrom(value any) (time.Time, bool) {
	switch typed := value.(type) {
	case time.Time:
		return typed, true
	case *time.Time:
		if typed != nil {
			return *typed, true
		}
	case string:
		parsed, err := time.Parse(time.RFC3339Nano, typed)
		return parsed, err == nil
	}
	return time.Time{}, false
}

func jsonArray(value any) []any {
	switch typed := value.(type) {
	case nil:
		return []any{}
	case []any:
		return typed
	case []map[string]any:
		result := make([]any, len(typed))
		for index := range typed {
			result[index] = typed[index]
		}
		return result
	case []byte:
		var result []any
		if json.Unmarshal(typed, &result) == nil {
			return result
		}
	case string:
		var result []any
		if json.Unmarshal([]byte(typed), &result) == nil {
			return result
		}
	}
	return []any{}
}

func jsonObject(value any) map[string]any {
	switch typed := value.(type) {
	case map[string]any:
		return typed
	case []byte:
		var result map[string]any
		if json.Unmarshal(typed, &result) == nil {
			return result
		}
	case string:
		var result map[string]any
		if json.Unmarshal([]byte(typed), &result) == nil {
			return result
		}
	}
	return map[string]any{}
}

func campaignLocationFrom(value any) campaignengine.Location {
	object := jsonObject(value)
	return campaignengine.Location{
		SystemIDs:        cleanCampaignIDs(object["systemIds"], 10),
		ConstellationIDs: cleanCampaignIDs(object["constellationIds"], 5),
		RegionIDs:        cleanCampaignIDs(object["regionIds"], 5),
	}
}

func cleanCampaignIDs(raw any, limit int) []int32 {
	items := jsonArray(raw)
	result := make([]int32, 0, min(len(items), limit))
	seen := map[int32]bool{}
	for _, item := range items {
		number, ok := jsonNumberInt64(item)
		if !ok || number <= 0 || number > math.MaxInt32 {
			continue
		}
		value := int32(number)
		if seen[value] {
			continue
		}
		seen[value] = true
		result = append(result, value)
		if len(result) == limit {
			break
		}
	}
	return result
}

func jsonNumberInt64(value any) (int64, bool) {
	if number, ok := int64Value(value); ok {
		return number, true
	}
	switch typed := value.(type) {
	case json.Number:
		number, err := typed.Int64()
		return number, err == nil
	case string:
		number, err := strconv.ParseInt(typed, 10, 64)
		return number, err == nil
	}
	return 0, false
}

func buildCampaignSides(
	sides []map[string]any,
	entities []map[string]any,
) []map[string]any {
	result := make([]map[string]any, 0, len(sides))
	for _, side := range sides {
		index := int16From(side["side_index"])
		sideEntities := make([]map[string]any, 0)
		for _, entity := range entities {
			if int16From(entity["side_index"]) != index {
				continue
			}
			copy := make(map[string]any, len(entity)-2)
			for key, value := range entity {
				if key != "id" && key != "side_index" {
					copy[key] = value
				}
			}
			sideEntities = append(sideEntities, copy)
		}
		result = append(result, map[string]any{
			"side_index":    side["side_index"],
			"name":          side["name"],
			"kills":         side["kills"],
			"losses":        side["losses"],
			"isk_destroyed": side["isk_destroyed"],
			"isk_lost":      side["isk_lost"],
			"palette":       side["palette"],
			"entities":      sideEntities,
		})
	}
	return result
}

func buildCampaignDaily(rows []map[string]any) map[string]any {
	distinct := map[string]bool{}
	normalized := make([]map[string]any, 0, len(rows))
	for _, row := range rows {
		period := campaignDay(row["day"])
		distinct[period] = true
		normalized = append(normalized, map[string]any{
			"period":        period,
			"side_index":    row["side_index"],
			"kills":         row["kills"],
			"losses":        row["losses"],
			"isk_destroyed": row["isk_destroyed"],
			"isk_lost":      row["isk_lost"],
		})
	}
	if len(distinct) <= campaignWeeklyThreshold {
		return map[string]any{"granularity": "day", "rows": normalized}
	}

	type bucketKey struct {
		Period string
		Side   int16
	}
	buckets := map[bucketKey]map[string]any{}
	for _, row := range normalized {
		day, err := time.Parse("2006-01-02", row["period"].(string))
		if err != nil {
			continue
		}
		monday := day.AddDate(0, 0, -((int(day.Weekday()) + 6) % 7))
		key := bucketKey{
			Period: monday.Format("2006-01-02"),
			Side:   int16From(row["side_index"]),
		}
		bucket := buckets[key]
		if bucket == nil {
			bucket = map[string]any{
				"period": key.Period, "side_index": key.Side,
				"kills": int64(0), "losses": int64(0),
				"isk_destroyed": float64(0), "isk_lost": float64(0),
			}
			buckets[key] = bucket
		}
		bucket["kills"] = int64From(bucket["kills"]) + int64From(row["kills"])
		bucket["losses"] = int64From(bucket["losses"]) + int64From(row["losses"])
		destroyed, _ := float64Value(row["isk_destroyed"])
		lost, _ := float64Value(row["isk_lost"])
		bucket["isk_destroyed"] = bucket["isk_destroyed"].(float64) + destroyed
		bucket["isk_lost"] = bucket["isk_lost"].(float64) + lost
	}
	weekly := make([]map[string]any, 0, len(buckets))
	for _, bucket := range buckets {
		weekly = append(weekly, bucket)
	}
	sort.Slice(weekly, func(left, right int) bool {
		lPeriod := weekly[left]["period"].(string)
		rPeriod := weekly[right]["period"].(string)
		if lPeriod != rPeriod {
			return lPeriod < rPeriod
		}
		return int16From(weekly[left]["side_index"]) <
			int16From(weekly[right]["side_index"])
	})
	return map[string]any{"granularity": "week", "rows": weekly}
}

func campaignDay(value any) string {
	if parsed, ok := timeFrom(value); ok {
		return parsed.UTC().Format("2006-01-02")
	}
	text := fmt.Sprint(value)
	if len(text) >= 10 {
		return text[:10]
	}
	return text
}

func buildCampaignPrize(
	campaignRow map[string]any,
	poolRows []map[string]any,
	resultRows []map[string]any,
	contributionRows []map[string]any,
	walletRows []map[string]any,
	principal *Principal,
	now time.Time,
) any {
	if len(poolRows) == 0 {
		return nil
	}
	pool := poolRows[0]
	status := int16From(pool["status"])
	funded, _ := strconv.ParseFloat(mapString(pool, "funded_total", "0"), 64)
	percentages := campaignPercentages(pool["payout_percentages"])
	payouts := make([]float64, len(resultRows))
	if status == campaignPrizeFunding {
		present := make([]int16, 0, len(resultRows))
		for _, row := range resultRows {
			present = append(present, int16From(row["payout_percentage"]))
		}
		if len(present) > 0 {
			percentages = present
		}
		payouts = campaignPrizePayouts(funded, percentages)
	} else {
		for index, row := range resultRows {
			payouts[index], _ = strconv.ParseFloat(
				mapString(row, "payout_amount", "0"), 64,
			)
		}
	}
	results := make([]map[string]any, 0, len(resultRows))
	for index, row := range resultRows {
		claimed := row["claimed_at"] != nil
		paid := row["paid_at"] != nil
		results = append(results, map[string]any{
			"rank":              row["rank"],
			"character_id":      row["character_id"],
			"character_name":    row["character_name"],
			"metric_value":      row["metric_value"],
			"secondary_value":   row["secondary_value"],
			"payout_percentage": row["payout_percentage"],
			"payout_amount":     payouts[index],
			"claimed":           claimed,
			"paid":              paid,
			"can_claim": status == campaignPrizeReady &&
				principal != nil &&
				int32From(row["character_id"]) == principal.CharacterID &&
				!claimed,
		})
	}
	contributions := make([]map[string]any, 0, len(contributionRows))
	for _, row := range contributionRows {
		contributions = append(contributions, map[string]any{
			"id":               fmt.Sprint(row["contribution_id"]),
			"source":           fmt.Sprint(row["source"]),
			"contributor_id":   row["contributor_id"],
			"contributor_name": fmt.Sprint(row["contributor_name"]),
			"contributor_type": fmt.Sprint(row["contributor_type"]),
			"amount":           fmt.Sprint(row["amount"]),
			"contributed_at":   row["contributed_at"],
		})
	}
	var settlementAt any
	var fundingCloses any
	var projectedLead any
	if end, ok := timeFrom(campaignRow["end_time"]); ok {
		fundingCloses = end
		settlementAt = end.Add(campaignengine.PrizeEndGraceHours * time.Hour)
		if !now.Before(end) && len(resultRows) > 0 {
			first, _ := strconv.ParseFloat(
				mapString(resultRows[0], "metric_value", "0"), 64,
			)
			second := float64(0)
			if len(resultRows) > 1 {
				second, _ = strconv.ParseFloat(
					mapString(resultRows[1], "metric_value", "0"), 64,
				)
			}
			if first > 0 && second > 0 {
				projectedLead = ((first - second) / second) * 100
			}
		}
	}
	contributionCount := int64(0)
	if len(contributionRows) > 0 {
		contributionCount = int64From(contributionRows[0]["contribution_count"])
	}
	return map[string]any{
		"metric":                 pool["metric"],
		"metric_label":           campaignPrizeMetricLabels[int16From(pool["metric"])],
		"winner_count":           pool["winner_count"],
		"payout_percentages":     pool["payout_percentages"],
		"status":                 pool["status"],
		"funded_total":           pool["funded_total"],
		"contribution_count":     contributionCount,
		"contributions":          contributions,
		"rules_locked_at":        pool["rules_locked_at"],
		"finalized_at":           pool["finalized_at"],
		"funding_reference":      "campaign:" + fmt.Sprint(campaignRow["campaign_id"]),
		"funding_closes_at":      fundingCloses,
		"settles_at":             settlementAt,
		"last_wallet_sync":       firstValue(walletRows, "last_journal_sync"),
		"discord_url":            "https://discord.gg/R9gZRc4Jtn",
		"projected_lead_percent": projectedLead,
		"results":                results,
	}
}

func campaignPercentages(value any) []int16 {
	switch typed := value.(type) {
	case []int16:
		return append([]int16(nil), typed...)
	case []int32:
		result := make([]int16, len(typed))
		for index := range typed {
			result[index] = int16(typed[index])
		}
		return result
	case []any:
		result := make([]int16, 0, len(typed))
		for _, item := range typed {
			result = append(result, int16From(item))
		}
		return result
	}
	return nil
}

func campaignPrizePayouts(funded float64, percentages []int16) []float64 {
	total := int64(math.Floor(math.Max(0, funded)))
	result := make([]float64, len(percentages))
	var weight int64
	for _, percentage := range percentages {
		if percentage > 0 {
			weight += int64(percentage)
		}
	}
	if total == 0 || weight == 0 || len(percentages) == 0 {
		return result
	}
	var remainder = total
	for index := 1; index < len(percentages); index++ {
		value := int64(math.Floor(
			float64(total*int64(percentages[index])) / float64(weight),
		))
		result[index] = float64(value)
		remainder -= value
	}
	result[0] = float64(remainder)
	return result
}

func campaignKilllistLimit(raw string) int {
	value, err := strconv.ParseFloat(strings.TrimSpace(raw), 64)
	if err != nil || math.IsNaN(value) || math.IsInf(value, 0) || value == 0 {
		return 50
	}
	limit := int(math.Floor(value))
	if limit < 10 {
		return 10
	}
	if limit > 100 {
		return 100
	}
	return limit
}
