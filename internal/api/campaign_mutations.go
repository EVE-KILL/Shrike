package api

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"regexp"
	"strconv"
	"strings"
	"time"

	campaignengine "github.com/eve-kill/shrike/internal/campaign"
	"github.com/eve-kill/shrike/internal/queue"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

var campaignSlugPattern = regexp.MustCompile(`^[0-9A-Za-z]{14}$`)

type campaignSideInput struct {
	Index    int16
	Name     string
	Entities []campaignEntityInput
}

type campaignEntityInput struct {
	Type int16
	ID   int32
}

type campaignPrizeInput struct {
	Metric      int16
	WinnerCount int16
	Percentages []int16
}

// Runtime decode types for the campaign write routes.
//
// Structured fields stay json.RawMessage and are handed to the compatibility
// parsers unchanged. openapi_body_types.go carries their concrete documented
// wire shapes without narrowing the values these parsers accept.
type campaignCreateBody struct {
	Name            string          `json:"name" doc:"Campaign name."`
	Description     *string         `json:"description,omitempty" doc:"Free text shown on the campaign page."`
	Visibility      json.RawMessage `json:"visibility,omitempty" doc:"Who may see the campaign."`
	StartTime       json.RawMessage `json:"startTime" doc:"Campaign start, as a timestamp or ISO 8601 string."`
	EndTime         json.RawMessage `json:"endTime,omitempty" doc:"Campaign end. Omit for an open-ended campaign."`
	Location        json.RawMessage `json:"location,omitempty" doc:"Location filter: system, constellation or region identifiers."`
	Sides           json.RawMessage `json:"sides,omitempty" doc:"Participating sides, each naming its entities."`
	AllowedEntities json.RawMessage `json:"allowedEntities,omitempty" doc:"Entities permitted to view a restricted campaign."`
	PrizePool       json.RawMessage `json:"prizePool,omitempty" doc:"Prize pool definition, including any initial contribution."`
}

type campaignUpdateBody struct {
	Name             json.RawMessage `json:"name,omitempty" doc:"New campaign name."`
	Description      json.RawMessage `json:"description,omitempty" doc:"New description. An empty string clears it."`
	Visibility       json.RawMessage `json:"visibility,omitempty" doc:"New visibility."`
	EndTime          json.RawMessage `json:"endTime,omitempty" doc:"New end time."`
	AllowedEntities  json.RawMessage `json:"allowedEntities,omitempty" doc:"Replacement viewer list."`
	Archived         json.RawMessage `json:"archived,omitempty" doc:"Archive or restore the campaign."`
	ResumeProcessing json.RawMessage `json:"resumeProcessing,omitempty" doc:"Resume killmail processing after an edit."`
}

type campaignContributeBody struct {
	RequestID string          `json:"requestId" doc:"Caller-supplied idempotency key for the contribution."`
	Amount    json.RawMessage `json:"amount" doc:"ISK amount to contribute."`
}

type campaignActionBody struct {
	Action string `json:"action" doc:"Administrative action to apply to the campaign."`
	Reason string `json:"reason,omitempty" maxLength:"500" doc:"Operator note recorded with the action."`
}

type campaignPrizePaidBody struct {
	Note string `json:"note,omitempty" maxLength:"500" doc:"Operator note recorded with the payout."`
}

func (s *campaignService) campaignCreateHandler() legacyHandler {
	return func(ctx context.Context, req *legacyRequest) (legacyPayload, error) {
		setAccountNoStore(req.Huma)
		if err := requireSameOriginMutation(req.Huma); err != nil {
			return legacyPayload{}, err
		}
		principal, err := s.auth.requirePrincipal(ctx, req)
		if err != nil {
			return legacyPayload{}, err
		}
		if s.storeErr != nil {
			return legacyPayload{}, s.storeErr
		}
		body, err := decodeJSONBody[campaignCreateBody](req, campaignBodyLimit)
		if err != nil {
			return legacyPayload{}, err
		}
		name := strings.TrimSpace(body.Name)
		if runeLength(name) < 3 || runeLength(name) > campaignMaximumNameLength {
			return legacyPayload{}, apiError(
				http.StatusBadRequest,
				fmt.Sprintf("Name must be 3–%d characters", campaignMaximumNameLength),
			)
		}
		var description *string
		if body.Description != nil {
			value := strings.TrimSpace(*body.Description)
			if runeLength(value) > campaignMaximumDescription {
				return legacyPayload{}, apiError(
					http.StatusBadRequest,
					fmt.Sprintf(
						"Description must be at most %d characters",
						campaignMaximumDescription,
					),
				)
			}
			if value != "" {
				description = &value
			}
		}
		visibility, err := parseCampaignVisibility(rawJSONValue(body.Visibility))
		if err != nil {
			return legacyPayload{}, err
		}
		start, err := parseCampaignTime(rawJSONValue(body.StartTime))
		if err != nil || start.Before(time.Date(
			2003, 5, 6, 0, 0, 0, 0, time.UTC,
		)) {
			return legacyPayload{}, apiError(
				http.StatusBadRequest, "Invalid start time",
			)
		}
		now := s.now().UTC()
		if start.After(now.Add(campaignMaximumFutureStartDays * 24 * time.Hour)) {
			return legacyPayload{}, apiError(
				http.StatusBadRequest,
				"Start time cannot be more than 30 days in the future",
			)
		}
		end, err := parseOptionalCampaignTime(rawJSONValue(body.EndTime))
		if err != nil || end != nil && !end.After(start) {
			return legacyPayload{}, apiError(
				http.StatusBadRequest, "End time must be after start time",
			)
		}
		if err := validateCampaignWindow(
			start, end, visibility, principal.IsAdmin, now,
		); err != nil {
			return legacyPayload{}, err
		}
		location := campaignLocationFrom(rawJSONValue(body.Location))
		sides, err := parseCampaignSides(rawJSONValue(body.Sides), location.HasFilter())
		if err != nil {
			return legacyPayload{}, err
		}
		allowed, err := parseCampaignAllowedEntities(
			rawJSONValue(body.AllowedEntities), visibility,
		)
		if err != nil {
			return legacyPayload{}, err
		}
		prize, initialContribution, requestID, err := parseCampaignPrizeInput(
			rawJSONValue(body.PrizePool), end, now,
		)
		if err != nil {
			return legacyPayload{}, err
		}
		fundingReference := ""
		if initialContribution != "" {
			fundingReference = fmt.Sprintf(
				"campaign-create:%d:%s",
				principal.CharacterID,
				requestID,
			)
			existing, findErr := s.findFundedCampaign(
				ctx, fundingReference, principal.CharacterID, initialContribution,
			)
			if findErr != nil {
				return legacyPayload{}, findErr
			}
			if existing != nil {
				return accountNoStorePayload(map[string]any{
					"campaign_id":          existing["campaign_id"],
					"estimated_killmails":  existing["estimated_killmails"],
					"initial_contribution": initialContribution,
					"replayed":             true,
				}), nil
			}
		}

		id, err := generateCampaignID()
		if err != nil {
			return legacyPayload{}, err
		}
		tx, err := s.mutations.Begin(ctx)
		if err != nil {
			return legacyPayload{}, err
		}
		defer tx.Rollback(ctx) //nolint:errcheck

		if _, err := tx.Exec(ctx, `
			SELECT pg_advisory_xact_lock(
			  (6010::bigint << 32) | ($1::bigint & 4294967295)
			)`, principal.CharacterID); err != nil {
			return legacyPayload{}, err
		}
		if !principal.IsAdmin {
			var live, recent, publicOngoing int
			if err := tx.QueryRow(ctx, `
				SELECT
				  COUNT(*) FILTER (WHERE status <> $2)::int,
				  COUNT(*) FILTER (
				    WHERE created_at >= now() - interval '1 hour'
				  )::int,
				  COUNT(*) FILTER (
				    WHERE visibility = $3
				      AND end_time IS NULL
				      AND status <> $2
				  )::int
				FROM campaigns
				WHERE created_by_character_id = $1`,
				principal.CharacterID,
				campaignengine.StatusArchived,
				campaignVisibilityPublic,
			).Scan(&live, &recent, &publicOngoing); err != nil {
				return legacyPayload{}, err
			}
			switch {
			case live >= campaignMaximumLivePerUser:
				return legacyPayload{}, apiError(
					http.StatusBadRequest,
					fmt.Sprintf(
						"You can have at most %d non-archived campaigns",
						campaignMaximumLivePerUser,
					),
				)
			case recent >= campaignMaximumCreatedPerHour:
				return legacyPayload{}, apiError(
					http.StatusTooManyRequests,
					fmt.Sprintf(
						"You can create at most %d campaigns per hour",
						campaignMaximumCreatedPerHour,
					),
				)
			case visibility == campaignVisibilityPublic &&
				end == nil &&
				publicOngoing >= campaignMaximumPublicOngoing:
				return legacyPayload{}, apiError(
					http.StatusBadRequest,
					fmt.Sprintf(
						"You can have at most %d active public ongoing campaigns",
						campaignMaximumPublicOngoing,
					),
				)
			}
		}

		locationJSON, err := nullableCampaignLocationJSON(location)
		if err != nil {
			return legacyPayload{}, err
		}
		allowedJSON, err := json.Marshal(allowed)
		if err != nil {
			return legacyPayload{}, err
		}
		_, err = tx.Exec(ctx, `
			INSERT INTO campaigns (
			  campaign_id, name, description, created_by_character_id,
			  status, visibility, allowed_entities, start_time, end_time,
			  location, estimated_killmails
			) VALUES (
			  $1, $2, $3, $4, $5, $6, $7::jsonb, $8, $9, $10::jsonb, 0
			)`,
			id,
			name,
			nullableString(description),
			principal.CharacterID,
			campaignengine.StatusPending,
			visibility,
			allowedJSON,
			start,
			end,
			locationJSON,
		)
		if err != nil {
			return legacyPayload{}, err
		}
		if prize != nil {
			if _, err := tx.Exec(ctx, `
				INSERT INTO campaign_prize_pools (
				  campaign_id, metric, winner_count, payout_percentages
				) VALUES ($1, $2, $3, $4)`,
				id, prize.Metric, prize.WinnerCount, prize.Percentages,
			); err != nil {
				return legacyPayload{}, err
			}
			if initialContribution != "" {
				_, err := contributeCampaignWallet(
					ctx,
					tx,
					principal.CharacterID,
					id,
					initialContribution,
					"Initial prize funding for "+name,
					fundingReference,
					map[string]any{
						"source":           "campaign_creation",
						"fundingRequestId": requestID,
					},
				)
				if err != nil {
					var operation *walletOperationError
					if errors.As(err, &operation) &&
						operation.Code == "idempotency_conflict" {
						existing, findErr := s.findFundedCampaign(
							ctx,
							fundingReference,
							principal.CharacterID,
							initialContribution,
						)
						if findErr == nil && existing != nil {
							return accountNoStorePayload(map[string]any{
								"campaign_id":          existing["campaign_id"],
								"estimated_killmails":  existing["estimated_killmails"],
								"initial_contribution": initialContribution,
								"replayed":             true,
							}), nil
						}
					}
					return legacyPayload{}, walletAPIError(err)
				}
			}
		}
		for _, side := range sides {
			if _, err := tx.Exec(ctx, `
				INSERT INTO campaign_sides (
				  campaign_id, side_index, name
				) VALUES ($1, $2, $3)`,
				id, side.Index, side.Name,
			); err != nil {
				return legacyPayload{}, err
			}
			for _, entity := range side.Entities {
				if _, err := tx.Exec(ctx, `
					INSERT INTO campaign_side_entities (
					  campaign_id, side_index, entity_type, entity_id
					) VALUES ($1, $2, $3, $4)`,
					id, side.Index, entity.Type, entity.ID,
				); err != nil {
					return legacyPayload{}, err
				}
			}
		}
		if err := tx.Commit(ctx); err != nil {
			return legacyPayload{}, err
		}
		s.dispatchCampaign(ctx, id)
		if initialContribution == "" {
			initialContribution = "0.00"
		}
		return accountNoStorePayload(map[string]any{
			"campaign_id": id,
			// Workload size is informational only. The worker intentionally
			// has no killmail ceiling; the time-window guard is the bound.
			"estimated_killmails":  0,
			"initial_contribution": initialContribution,
			"replayed":             false,
		}), nil
	}
}

func (s *campaignService) campaignUpdateHandler() legacyHandler {
	return func(ctx context.Context, req *legacyRequest) (legacyPayload, error) {
		setAccountNoStore(req.Huma)
		if err := requireSameOriginMutation(req.Huma); err != nil {
			return legacyPayload{}, err
		}
		principal, err := s.auth.requirePrincipal(ctx, req)
		if err != nil {
			return legacyPayload{}, err
		}
		if s.storeErr != nil {
			return legacyPayload{}, s.storeErr
		}
		id, err := campaignID(req.Param("id"))
		if err != nil {
			return legacyPayload{}, err
		}
		body, err := decodeJSONBody[campaignUpdateBody](req, campaignBodyLimit)
		if err != nil {
			return legacyPayload{}, err
		}
		campaign, err := queryMap(ctx, s.consistent, `
			SELECT c.*,
			       pool.campaign_id IS NOT NULL AS has_prize_pool,
			       pool.rules_locked_at
			FROM campaigns c
			LEFT JOIN campaign_prize_pools pool
			  ON pool.campaign_id = c.campaign_id
			WHERE c.campaign_id = $1
			LIMIT 1`, id)
		if err != nil {
			return legacyPayload{}, err
		}
		if campaign == nil {
			return legacyPayload{}, apiError(
				http.StatusNotFound, "Campaign not found",
			)
		}
		creator := int32From(campaign["created_by_character_id"])
		if !principal.IsAdmin && creator != principal.CharacterID {
			return legacyPayload{}, apiError(
				http.StatusForbidden,
				"Only the creator or an admin can edit this campaign",
			)
		}
		hasPrize := boolFrom(campaign["has_prize_pool"])
		updates := map[string]any{}
		needsRecompute := false
		if value, found := rawJSONField(body.Name); found {
			name := strings.TrimSpace(jsonString(value))
			if runeLength(name) < 3 || runeLength(name) > campaignMaximumNameLength {
				return legacyPayload{}, apiError(
					http.StatusBadRequest,
					fmt.Sprintf(
						"Name must be 3–%d characters",
						campaignMaximumNameLength,
					),
				)
			}
			updates["name"] = name
		}
		if value, found := rawJSONField(body.Description); found {
			description := strings.TrimSpace(jsonString(value))
			if runeLength(description) > campaignMaximumDescription {
				return legacyPayload{}, apiError(
					http.StatusBadRequest,
					fmt.Sprintf(
						"Description must be at most %d characters",
						campaignMaximumDescription,
					),
				)
			}
			if description == "" {
				updates["description"] = nil
			} else {
				updates["description"] = description
			}
		}
		start, _ := timeFrom(campaign["start_time"])
		nextEnd, _ := optionalTimeFrom(campaign["end_time"])
		if raw, found := rawJSONField(body.EndTime); found {
			parsed, parseErr := parseOptionalCampaignTime(raw)
			if parseErr != nil || parsed != nil && !parsed.After(start) {
				return legacyPayload{}, apiError(
					http.StatusBadRequest, "End time must be after start time",
				)
			}
			if !sameOptionalTime(nextEnd, parsed) {
				if campaign["rules_locked_at"] != nil {
					return legacyPayload{}, apiError(
						http.StatusBadRequest,
						"The campaign end time is locked because its prize pool has received funding",
					)
				}
				if hasPrize && parsed == nil {
					return legacyPayload{}, apiError(
						http.StatusBadRequest,
						"Prize campaigns require an end time",
					)
				}
				nextEnd = parsed
				updates["end_time"] = parsed
				needsRecompute = true
			}
		}
		nextVisibility := int16From(campaign["visibility"])
		visibilityRaw, visibilityFound := rawJSONField(body.Visibility)
		allowedRaw, allowedFound := rawJSONField(body.AllowedEntities)
		if visibilityFound || allowedFound {
			if visibilityFound {
				nextVisibility, err = parseCampaignVisibility(visibilityRaw)
				if err != nil {
					return legacyPayload{}, err
				}
				updates["visibility"] = nextVisibility
			}
			if !allowedFound {
				allowedRaw = campaign["allowed_entities"]
			}
			allowed, parseErr := parseCampaignAllowedEntities(
				allowedRaw, nextVisibility,
			)
			if parseErr != nil {
				return legacyPayload{}, parseErr
			}
			encoded, _ := json.Marshal(allowed)
			updates["allowed_entities"] = encoded
		}
		if raw, found := rawJSONField(body.Archived); found {
			archived, _ := raw.(bool)
			if hasPrize {
				return legacyPayload{}, apiError(
					http.StatusBadRequest,
					"Prize campaigns settle automatically after their end time and cannot be archived manually",
				)
			}
			if archived {
				updates["status"] = campaignengine.StatusArchived
			} else if int16From(campaign["status"]) ==
				campaignengine.StatusArchived {
				needsRecompute = true
			}
		}
		resume, _ := rawJSONValue(body.ResumeProcessing).(bool)
		if resume && boolFrom(campaign["processing_paused"]) {
			updates["processing_paused"] = false
			updates["processing_note"] = nil
			updates["last_processing_error"] = nil
			needsRecompute = true
		}
		if needsRecompute {
			updates["status"] = campaignengine.StatusPending
			updates["estimated_killmails"] = 0
			resumeExplicitlyFalse := rawJSONValue(body.ResumeProcessing) == false
			if boolFrom(campaign["processing_paused"]) && !resumeExplicitlyFalse {
				updates["processing_paused"] = false
				updates["processing_note"] = nil
			}
		}
		if needsRecompute ||
			nextVisibility != int16From(campaign["visibility"]) {
			if err := validateCampaignWindow(
				start,
				nextEnd,
				nextVisibility,
				principal.IsAdmin,
				s.now().UTC(),
			); err != nil {
				return legacyPayload{}, err
			}
		}
		if len(updates) == 0 {
			return accountNoStorePayload(map[string]any{"updated": false}), nil
		}

		tx, err := s.mutations.Begin(ctx)
		if err != nil {
			return legacyPayload{}, err
		}
		defer tx.Rollback(ctx) //nolint:errcheck
		becomingPublicOngoing := nextVisibility == campaignVisibilityPublic &&
			nextEnd == nil &&
			(int16From(campaign["visibility"]) != campaignVisibilityPublic ||
				campaign["end_time"] != nil)
		if needsRecompute || !principal.IsAdmin && becomingPublicOngoing {
			if _, err := tx.Exec(ctx, `
				SELECT pg_advisory_xact_lock(
				  (6010::bigint << 32) | ($1::bigint & 4294967295)
				)`, creator); err != nil {
				return legacyPayload{}, err
			}
		}
		if !principal.IsAdmin && becomingPublicOngoing {
			var count int
			if err := tx.QueryRow(ctx, `
				SELECT COUNT(*)::int
				FROM campaigns
				WHERE created_by_character_id = $1
				  AND campaign_id <> $2
				  AND visibility = $3
				  AND end_time IS NULL
				  AND status <> $4`,
				principal.CharacterID,
				id,
				campaignVisibilityPublic,
				campaignengine.StatusArchived,
			).Scan(&count); err != nil {
				return legacyPayload{}, err
			}
			if count >= campaignMaximumPublicOngoing {
				return legacyPayload{}, apiError(
					http.StatusBadRequest,
					fmt.Sprintf(
						"You can have at most %d active public ongoing campaigns",
						campaignMaximumPublicOngoing,
					),
				)
			}
		}
		if err := updateCampaignColumns(ctx, tx, id, updates); err != nil {
			return legacyPayload{}, err
		}
		if err := tx.Commit(ctx); err != nil {
			return legacyPayload{}, err
		}
		if needsRecompute {
			s.dispatchCampaign(ctx, id)
		}
		return accountNoStorePayload(map[string]any{
			"updated": true, "recompute": needsRecompute,
		}), nil
	}
}

func (s *campaignService) campaignDeleteHandler() legacyHandler {
	return func(ctx context.Context, req *legacyRequest) (legacyPayload, error) {
		setAccountNoStore(req.Huma)
		if err := requireSameOriginMutation(req.Huma); err != nil {
			return legacyPayload{}, err
		}
		principal, err := s.auth.requirePrincipal(ctx, req)
		if err != nil {
			return legacyPayload{}, err
		}
		if s.storeErr != nil {
			return legacyPayload{}, s.storeErr
		}
		id, err := campaignID(req.Param("id"))
		if err != nil {
			return legacyPayload{}, err
		}
		tx, err := s.mutations.Begin(ctx)
		if err != nil {
			return legacyPayload{}, err
		}
		defer tx.Rollback(ctx) //nolint:errcheck
		var creator int32
		err = tx.QueryRow(ctx, `
			SELECT created_by_character_id
			FROM campaigns WHERE campaign_id = $1
			FOR UPDATE`, id).Scan(&creator)
		if errors.Is(err, pgx.ErrNoRows) {
			return legacyPayload{}, apiError(
				http.StatusNotFound, "Campaign not found",
			)
		}
		if err != nil {
			return legacyPayload{}, err
		}
		if !principal.IsAdmin && creator != principal.CharacterID {
			return legacyPayload{}, apiError(
				http.StatusForbidden,
				"Only the creator or an admin can delete this campaign",
			)
		}
		var funded string
		err = tx.QueryRow(ctx, `
			SELECT funded_total::text
			FROM campaign_prize_pools
			WHERE campaign_id = $1
			FOR UPDATE`, id).Scan(&funded)
		if err == nil {
			cents, parseErr := parseWalletCentsBig(funded)
			if parseErr != nil {
				return legacyPayload{}, parseErr
			}
			if cents.Sign() > 0 {
				return legacyPayload{}, apiError(
					http.StatusBadRequest,
					"Funded prize campaigns cannot be deleted",
				)
			}
		} else if !errors.Is(err, pgx.ErrNoRows) {
			return legacyPayload{}, err
		}
		if _, err := tx.Exec(
			ctx, `DELETE FROM campaigns WHERE campaign_id = $1`, id,
		); err != nil {
			return legacyPayload{}, err
		}
		if err := tx.Commit(ctx); err != nil {
			return legacyPayload{}, err
		}
		return accountNoStorePayload(map[string]any{"deleted": true}), nil
	}
}

func (s *campaignService) campaignContributeHandler() legacyHandler {
	return func(ctx context.Context, req *legacyRequest) (legacyPayload, error) {
		setAccountNoStore(req.Huma)
		if err := requireSameOriginMutation(req.Huma); err != nil {
			return legacyPayload{}, err
		}
		principal, err := s.auth.requirePrincipal(ctx, req)
		if err != nil {
			return legacyPayload{}, err
		}
		if s.storeErr != nil {
			return legacyPayload{}, s.storeErr
		}
		id, err := campaignID(req.Param("id"))
		if err != nil {
			return legacyPayload{}, err
		}
		body, err := decodeJSONBody[campaignContributeBody](req, campaignBodyLimit)
		if err != nil {
			return legacyPayload{}, err
		}
		requestID := strings.TrimSpace(body.RequestID)
		if parsed, parseErr := uuid.Parse(requestID); parseErr != nil ||
			parsed.Version() < 1 || parsed.Version() > 8 {
			return legacyPayload{}, apiError(
				http.StatusBadRequest,
				"Campaign funding request is missing a valid retry token",
			)
		}
		amount, err := normalizeWalletAmount(rawJSONValue(body.Amount))
		if err != nil {
			return legacyPayload{}, walletAPIError(err)
		}
		campaign, err := queryMap(ctx, s.consistent, `
			SELECT campaign_id, name, visibility, created_by_character_id
			FROM campaigns WHERE campaign_id = $1 LIMIT 1`, id)
		if err != nil {
			return legacyPayload{}, err
		}
		if campaign == nil {
			return legacyPayload{}, apiError(
				http.StatusNotFound, "Campaign not found",
			)
		}
		if err := requireCampaignView(
			ctx, s.consistent, req, campaign, principal,
		); err != nil {
			return legacyPayload{}, err
		}
		tx, err := s.mutations.Begin(ctx)
		if err != nil {
			return legacyPayload{}, err
		}
		defer tx.Rollback(ctx) //nolint:errcheck
		result, err := contributeCampaignWallet(
			ctx,
			tx,
			principal.CharacterID,
			id,
			amount,
			"Prize funding for "+jsonString(campaign["name"]),
			fmt.Sprintf(
				"campaign-contribute:%d:%s:%s",
				principal.CharacterID,
				id,
				requestID,
			),
			map[string]any{
				"source":    "campaign_page",
				"requestId": requestID,
			},
		)
		if err != nil {
			return legacyPayload{}, walletAPIError(err)
		}
		if err := tx.Commit(ctx); err != nil {
			return legacyPayload{}, err
		}
		return accountNoStorePayload(map[string]any{
			"contributed": amount,
			"replayed":    result.Replayed,
			"balance":     result.Balance,
		}), nil
	}
}

func (s *campaignService) campaignClaimHandler() legacyHandler {
	return func(ctx context.Context, req *legacyRequest) (legacyPayload, error) {
		setAccountNoStore(req.Huma)
		if err := requireSameOriginMutation(req.Huma); err != nil {
			return legacyPayload{}, err
		}
		principal, err := s.auth.requirePrincipal(ctx, req)
		if err != nil {
			return legacyPayload{}, err
		}
		id, err := campaignID(req.Param("id"))
		if err != nil {
			return legacyPayload{}, err
		}
		campaign, err := queryMap(ctx, s.consistent, `
			SELECT campaign_id, visibility, created_by_character_id
			FROM campaigns WHERE campaign_id = $1 LIMIT 1`, id)
		if err != nil {
			return legacyPayload{}, err
		}
		if campaign == nil {
			return legacyPayload{}, apiError(
				http.StatusNotFound, "Campaign not found",
			)
		}
		if err := requireCampaignView(
			ctx, s.consistent, req, campaign, principal,
		); err != nil {
			return legacyPayload{}, err
		}
		if s.storeErr != nil {
			return legacyPayload{}, s.storeErr
		}
		var rank int16
		err = s.mutations.QueryRow(ctx, `
			UPDATE campaign_prize_results result
			SET claimed_at = now(),
			    claimed_by_character_id = $2,
			    updated_at = now()
			WHERE result.campaign_id = $1
			  AND result.character_id = $2
			  AND result.claimed_at IS NULL
			  AND EXISTS (
			    SELECT 1 FROM campaign_prize_pools pool
			    WHERE pool.campaign_id = result.campaign_id
			      AND pool.status = $3
			  )
			RETURNING result.rank`,
			id, principal.CharacterID, campaignPrizeReady,
		).Scan(&rank)
		if errors.Is(err, pgx.ErrNoRows) {
			var status *int16
			_ = s.mutations.QueryRow(ctx, `
				SELECT status FROM campaign_prize_pools
				WHERE campaign_id = $1`, id).Scan(&status)
			if status == nil || *status != campaignPrizeReady {
				return legacyPayload{}, apiError(
					http.StatusBadRequest,
					"Campaign prizes are not ready to claim",
				)
			}
			return legacyPayload{}, apiError(
				http.StatusBadRequest,
				"Sign in with a winning character to claim this prize",
			)
		}
		if err != nil {
			return legacyPayload{}, err
		}
		return accountNoStorePayload(map[string]any{
			"claimed": true, "rank": rank,
		}), nil
	}
}

func (s *campaignService) adminCampaignListHandler() legacyHandler {
	return func(ctx context.Context, req *legacyRequest) (legacyPayload, error) {
		setAccountNoStore(req.Huma)
		if _, err := s.requireAdmin(ctx, req); err != nil {
			return legacyPayload{}, err
		}
		page := boundedCampaignPage(req.Query.Get("page"))
		search := truncateRunes(strings.TrimSpace(req.Query.Get("q")), 100)
		state := strings.TrimSpace(req.Query.Get("state"))
		args := []any{}
		add := func(value any) string {
			args = append(args, value)
			return fmt.Sprintf("$%d", len(args))
		}
		filters := []string{"true"}
		switch state {
		case "paused":
			filters = append(filters, "c.processing_paused = true")
		case "pending":
			filters = append(filters, "c.status = 0")
		case "active":
			filters = append(filters, "c.status = 1")
		case "archived":
			filters = append(filters, "c.status = 2")
		case "failed":
			filters = append(filters, "c.last_processing_error IS NOT NULL")
		}
		if search != "" {
			placeholder := add("%" + escapeLike(search) + "%")
			exact := add(search)
			filters = append(filters, fmt.Sprintf(`(
				c.campaign_id = %s
				OR c.name ILIKE %s ESCAPE '\'
				OR creator.character_name ILIKE %s ESCAPE '\'
			)`, exact, placeholder, placeholder))
		}
		limit := add(campaignAdminPageSize + 1)
		offset := add((page - 1) * campaignAdminPageSize)
		rows, err := queryMaps(ctx, s.consistent, fmt.Sprintf(`
			SELECT c.campaign_id, c.name, c.status, c.visibility,
			       c.created_by_character_id,
			       creator.character_name AS creator_name,
			       c.start_time, c.end_time, c.created_at, c.updated_at,
			       c.processing_paused, c.processing_note,
			       c.estimated_killmails,
			       c.last_processing_started_at,
			       c.last_processing_duration_ms,
			       c.last_processing_killmails,
			       c.last_processing_error,
			       c.stats->'totals' AS totals
			FROM campaigns c
			LEFT JOIN users creator
			  ON creator.character_id = c.created_by_character_id
			WHERE %s
			ORDER BY c.processing_paused DESC, c.created_at DESC
			LIMIT %s OFFSET %s`,
			strings.Join(filters, " AND "), limit, offset,
		), args...)
		if err != nil {
			return legacyPayload{}, err
		}
		hasMore := len(rows) > campaignAdminPageSize
		if hasMore {
			rows = rows[:campaignAdminPageSize]
		}
		return accountNoStorePayload(map[string]any{
			"campaigns": rows,
			"page":      page,
			"hasMore":   hasMore,
		}), nil
	}
}

func (s *campaignService) adminCampaignActionHandler() legacyHandler {
	return func(ctx context.Context, req *legacyRequest) (legacyPayload, error) {
		setAccountNoStore(req.Huma)
		if err := requireSameOriginMutation(req.Huma); err != nil {
			return legacyPayload{}, err
		}
		admin, err := s.requireAdmin(ctx, req)
		if err != nil {
			return legacyPayload{}, err
		}
		if s.storeErr != nil {
			return legacyPayload{}, s.storeErr
		}
		id, err := campaignID(req.Param("id"))
		if err != nil {
			return legacyPayload{}, err
		}
		body, err := decodeJSONBody[campaignActionBody](req, campaignBodyLimit)
		if err != nil {
			return legacyPayload{}, err
		}
		action := body.Action
		reason := truncateRunes(strings.TrimSpace(body.Reason), 500)
		tx, err := s.mutations.Begin(ctx)
		if err != nil {
			return legacyPayload{}, err
		}
		defer tx.Rollback(ctx) //nolint:errcheck
		var lockedID string
		err = tx.QueryRow(ctx, `
			SELECT campaign_id
			FROM campaigns
			WHERE campaign_id = $1
			FOR UPDATE`, id).Scan(&lockedID)
		if errors.Is(err, pgx.ErrNoRows) {
			return legacyPayload{}, apiError(
				http.StatusNotFound, "Campaign not found",
			)
		}
		if err != nil {
			return legacyPayload{}, err
		}
		var funded *string
		var fundedValue string
		err = tx.QueryRow(ctx, `
			SELECT funded_total::text
			FROM campaign_prize_pools
			WHERE campaign_id = $1
			FOR UPDATE`, id).Scan(&fundedValue)
		if err == nil {
			funded = &fundedValue
		} else if !errors.Is(err, pgx.ErrNoRows) {
			return legacyPayload{}, err
		}
		hasPrize := funded != nil
		dispatched := false
		switch action {
		case "pause":
			if reason == "" {
				reason = "Paused by " + admin.CharacterName
			}
			_, err = tx.Exec(ctx, `
				UPDATE campaigns
				SET processing_paused = true,
				    processing_note = $2,
				    updated_at = now()
				WHERE campaign_id = $1`, id, reason)
		case "resume", "reprocess":
			_, err = tx.Exec(ctx, `
				UPDATE campaigns
				SET processing_paused = false,
				    processing_note = NULL,
				    last_processing_error = NULL,
				    status = $2,
				    updated_at = now()
				WHERE campaign_id = $1`,
				id, campaignengine.StatusPending)
			dispatched = true
		case "archive":
			if hasPrize {
				return legacyPayload{}, apiError(
					http.StatusBadRequest,
					"Prize campaigns settle automatically and cannot be archived manually",
				)
			}
			if reason == "" {
				reason = "Archived by " + admin.CharacterName
			}
			_, err = tx.Exec(ctx, `
				UPDATE campaigns
				SET status = $2,
				    processing_paused = true,
				    processing_note = $3,
				    updated_at = now()
				WHERE campaign_id = $1`,
				id, campaignengine.StatusArchived, reason)
		case "delete":
			if funded != nil {
				cents, parseErr := parseWalletCentsBig(*funded)
				if parseErr != nil {
					return legacyPayload{}, parseErr
				}
				if cents.Sign() > 0 {
					return legacyPayload{}, apiError(
						http.StatusBadRequest,
						"Funded prize campaigns cannot be deleted",
					)
				}
			}
			_, err = tx.Exec(
				ctx, `DELETE FROM campaigns WHERE campaign_id = $1`, id,
			)
		default:
			return legacyPayload{}, apiError(
				http.StatusBadRequest, "Invalid campaign action",
			)
		}
		if err != nil {
			return legacyPayload{}, err
		}
		if err := tx.Commit(ctx); err != nil {
			return legacyPayload{}, err
		}
		if dispatched {
			s.dispatchCampaign(ctx, id)
		}
		return accountNoStorePayload(map[string]any{
			"ok": true, "action": action, "dispatched": dispatched,
		}), nil
	}
}

func (s *campaignService) adminCampaignPrizePaidHandler() legacyHandler {
	return func(ctx context.Context, req *legacyRequest) (legacyPayload, error) {
		setAccountNoStore(req.Huma)
		if err := requireSameOriginMutation(req.Huma); err != nil {
			return legacyPayload{}, err
		}
		admin, err := s.requireAdmin(ctx, req)
		if err != nil {
			return legacyPayload{}, err
		}
		if s.storeErr != nil {
			return legacyPayload{}, s.storeErr
		}
		id, err := campaignID(req.Param("id"))
		if err != nil {
			return legacyPayload{}, apiError(
				http.StatusBadRequest, "Invalid prize result",
			)
		}
		rankValue, err := strconv.ParseInt(req.Param("rank"), 10, 16)
		if err != nil {
			return legacyPayload{}, apiError(
				http.StatusBadRequest, "Invalid prize result",
			)
		}
		body, err := decodeJSONBody[campaignPrizePaidBody](req, campaignBodyLimit)
		if err != nil {
			return legacyPayload{}, err
		}
		note := truncateRunes(strings.TrimSpace(body.Note), 500)
		var noteValue any
		if note != "" {
			noteValue = note
		}
		tx, err := s.mutations.Begin(ctx)
		if err != nil {
			return legacyPayload{}, err
		}
		defer tx.Rollback(ctx) //nolint:errcheck
		var status int16
		err = tx.QueryRow(ctx, `
			SELECT status FROM campaign_prize_pools
			WHERE campaign_id = $1
			FOR UPDATE`, id).Scan(&status)
		if errors.Is(err, pgx.ErrNoRows) ||
			err == nil &&
				status != campaignPrizeReady &&
				status != campaignPrizePaid {
			return legacyPayload{}, apiError(
				http.StatusBadRequest,
				"Campaign prizes are not ready for payment",
			)
		}
		if err != nil {
			return legacyPayload{}, err
		}
		command, err := tx.Exec(ctx, `
			UPDATE campaign_prize_results
			SET paid_at = now(),
			    paid_by_admin_character_id = $3,
			    payment_note = $4,
			    updated_at = now()
			WHERE campaign_id = $1
			  AND rank = $2
			  AND claimed_at IS NOT NULL
			  AND paid_at IS NULL`,
			id, int16(rankValue), admin.CharacterID, noteValue,
		)
		if err != nil {
			return legacyPayload{}, err
		}
		if command.RowsAffected() == 0 {
			return legacyPayload{}, apiError(
				http.StatusBadRequest,
				"The winner must claim this prize before it can be marked paid",
			)
		}
		var unpaid int
		if err := tx.QueryRow(ctx, `
			SELECT COUNT(*)::int
			FROM campaign_prize_results
			WHERE campaign_id = $1 AND paid_at IS NULL`,
			id,
		).Scan(&unpaid); err != nil {
			return legacyPayload{}, err
		}
		if unpaid == 0 {
			if _, err := tx.Exec(ctx, `
				UPDATE campaign_prize_pools
				SET status = $2, updated_at = now()
				WHERE campaign_id = $1`,
				id, campaignPrizePaid,
			); err != nil {
				return legacyPayload{}, err
			}
		}
		if err := tx.Commit(ctx); err != nil {
			return legacyPayload{}, err
		}
		return accountNoStorePayload(map[string]any{"paid": true}), nil
	}
}

func (s *campaignService) requireAdmin(
	ctx context.Context,
	req *legacyRequest,
) (*Principal, error) {
	principal, err := s.auth.requirePrincipal(ctx, req)
	if err != nil {
		return nil, err
	}
	if !principal.IsAdmin {
		return nil, apiError(
			http.StatusForbidden, "Administrator access required",
		)
	}
	return principal, nil
}

func (s *campaignService) dispatchCampaign(ctx context.Context, id string) {
	if s.queue == nil || s.queueErr != nil {
		return
	}
	_, _ = queue.Dispatch(
		context.WithoutCancel(ctx),
		s.queue,
		queue.CampaignProcessingArgs{CampaignID: id},
		queue.Immediate,
	)
}

func (s *campaignService) findFundedCampaign(
	ctx context.Context,
	externalReference string,
	characterID int32,
	amount string,
) (map[string]any, error) {
	row, err := queryMap(ctx, s.consistent, `
		SELECT wallet_tx.character_id, wallet_tx.type,
		       wallet_tx.amount::text AS amount, wallet_tx.campaign_id,
		       campaign.estimated_killmails
		FROM ek_wallet_transactions wallet_tx
		LEFT JOIN campaigns campaign
		  ON campaign.campaign_id = wallet_tx.campaign_id
		WHERE wallet_tx.external_reference = $1
		LIMIT 1`, externalReference)
	if err != nil || row == nil {
		return row, err
	}
	if int32From(row["character_id"]) != characterID ||
		int16From(row["type"]) != walletCampaignContributionType ||
		mapString(row, "amount", "") != "-"+amount ||
		row["campaign_id"] == nil {
		return nil, apiError(
			http.StatusConflict,
			"This campaign funding retry token was already used for another operation",
		)
	}
	if row["estimated_killmails"] == nil {
		return nil, apiError(
			http.StatusInternalServerError,
			"The funded campaign transaction exists without its campaign",
		)
	}
	return row, nil
}

func parseCampaignVisibility(raw any) (int16, error) {
	if raw == nil {
		return campaignVisibilityPublic, nil
	}
	number, ok := jsonNumberInt64(raw)
	if !ok || number < campaignVisibilityPublic ||
		number > campaignVisibilityKillboard {
		return 0, apiError(http.StatusBadRequest, "Invalid visibility")
	}
	return int16(number), nil
}

func parseCampaignSides(
	raw any,
	hasLocation bool,
) ([]campaignSideInput, error) {
	items := jsonArray(raw)
	if len(items) > campaignMaximumSides {
		return nil, apiError(
			http.StatusBadRequest,
			fmt.Sprintf(
				"Campaigns support at most %d sides",
				campaignMaximumSides,
			),
		)
	}
	if len(items) == 0 && !hasLocation {
		return nil, apiError(
			http.StatusBadRequest,
			"Area campaigns need at least one region or system",
		)
	}
	seen := map[string]bool{}
	sides := make([]campaignSideInput, 0, len(items))
	for index, rawSide := range items {
		side := jsonObject(rawSide)
		name := strings.TrimSpace(jsonString(side["name"]))
		if name == "" {
			name = fmt.Sprintf("Side %d", index+1)
		}
		if runeLength(name) > 50 {
			return nil, apiError(
				http.StatusBadRequest,
				"Side names must be at most 50 characters",
			)
		}
		rawEntities := jsonArray(side["entities"])
		if len(rawEntities) < 1 ||
			len(rawEntities) > campaignMaximumEntitiesPerSide {
			return nil, apiError(
				http.StatusBadRequest,
				fmt.Sprintf(
					"Each side needs 1–%d entities",
					campaignMaximumEntitiesPerSide,
				),
			)
		}
		parsed := campaignSideInput{Index: int16(index), Name: name}
		for _, rawEntity := range rawEntities {
			entity := jsonObject(rawEntity)
			entityType := campaignengineEntityType(jsonString(entity["type"]))
			entityID, valid := jsonNumberInt64(entity["id"])
			if entityType < 0 || !valid ||
				entityID <= 0 || entityID > int64(pgInt4Max) {
				return nil, apiError(
					http.StatusBadRequest, "Invalid entity reference",
				)
			}
			key := fmt.Sprintf("%d:%d", entityType, entityID)
			if seen[key] {
				return nil, apiError(
					http.StatusBadRequest,
					"An entity can only appear on one side",
				)
			}
			seen[key] = true
			parsed.Entities = append(parsed.Entities, campaignEntityInput{
				Type: int16(entityType), ID: int32(entityID),
			})
		}
		sides = append(sides, parsed)
	}
	return sides, nil
}

func parseCampaignAllowedEntities(
	raw any,
	visibility int16,
) ([]map[string]any, error) {
	if visibility != campaignVisibilityKillboard {
		return []map[string]any{}, nil
	}
	items := jsonArray(raw)
	if len(items) > campaignMaximumAllowedEntities {
		return nil, apiError(
			http.StatusBadRequest,
			fmt.Sprintf(
				"At most %d allowed entities",
				campaignMaximumAllowedEntities,
			),
		)
	}
	seen := map[string]bool{}
	result := make([]map[string]any, 0, len(items))
	for _, item := range items {
		entity := jsonObject(item)
		entityType := jsonString(entity["type"])
		entityID, valid := jsonNumberInt64(entity["id"])
		if campaignengineEntityType(entityType) < 0 ||
			!valid || entityID <= 0 || entityID > int64(pgInt4Max) {
			return nil, apiError(
				http.StatusBadRequest, "Invalid allowed entity reference",
			)
		}
		key := fmt.Sprintf("%s:%d", entityType, entityID)
		if seen[key] {
			continue
		}
		seen[key] = true
		var name any
		if entity["name"] != nil {
			name = truncateRunes(jsonString(entity["name"]), 100)
		}
		result = append(result, map[string]any{
			"type": entityType, "id": entityID, "name": name,
		})
	}
	return result, nil
}

func parseCampaignPrizeInput(
	raw any,
	end *time.Time,
	now time.Time,
) (*campaignPrizeInput, string, string, error) {
	input := jsonObject(raw)
	enabled, _ := input["enabled"].(bool)
	if !enabled {
		if contribution := input["initialContribution"]; contribution != nil &&
			strings.TrimSpace(jsonString(contribution)) != "" &&
			!zeroWalletAmount(contribution) {
			return nil, "", "", apiError(
				http.StatusBadRequest,
				"Enable campaign prizes before adding initial funding",
			)
		}
		return nil, "", "", nil
	}
	if end == nil {
		return nil, "", "", apiError(
			http.StatusBadRequest, "Prize campaigns require an end time",
		)
	}
	if !end.After(now) {
		return nil, "", "", apiError(
			http.StatusBadRequest, "Prize campaigns must end in the future",
		)
	}
	metric, metricOK := jsonNumberInt64(input["metric"])
	winners, winnersOK := jsonNumberInt64(input["winnerCount"])
	if !metricOK || metric < 0 || metric > 3 {
		return nil, "", "", apiError(
			http.StatusBadRequest, "Choose a valid campaign prize metric",
		)
	}
	if !winnersOK || winners < 3 || winners > 10 {
		return nil, "", "", apiError(
			http.StatusBadRequest,
			"Campaign prizes must reward 3–10 pilots",
		)
	}
	rawPercentages := jsonArray(input["payoutPercentages"])
	if len(rawPercentages) != int(winners) {
		return nil, "", "", apiError(
			http.StatusBadRequest,
			"Provide one payout percentage for every winning place",
		)
	}
	percentages := make([]int16, 0, len(rawPercentages))
	total := int64(0)
	for _, rawPercentage := range rawPercentages {
		percentage, ok := jsonNumberInt64(rawPercentage)
		if !ok || percentage < 1 || percentage > 100 {
			return nil, "", "", apiError(
				http.StatusBadRequest,
				"Every winning place must receive at least 1%",
			)
		}
		total += percentage
		percentages = append(percentages, int16(percentage))
	}
	if total != 100 {
		return nil, "", "", apiError(
			http.StatusBadRequest,
			"Campaign prize percentages must total exactly 100%",
		)
	}
	var amount, requestID string
	if contribution := input["initialContribution"]; contribution != nil &&
		strings.TrimSpace(jsonString(contribution)) != "" &&
		!zeroWalletAmount(contribution) {
		var err error
		amount, err = normalizeWalletAmount(contribution)
		if err != nil {
			return nil, "", "", walletAPIError(err)
		}
		requestID = strings.TrimSpace(jsonString(input["fundingRequestId"]))
		parsed, err := uuid.Parse(requestID)
		if err != nil || parsed.Version() < 1 || parsed.Version() > 8 {
			return nil, "", "", apiError(
				http.StatusBadRequest,
				"Initial campaign funding request is missing a valid retry token",
			)
		}
	}
	return &campaignPrizeInput{
		Metric: int16(metric), WinnerCount: int16(winners),
		Percentages: percentages,
	}, amount, requestID, nil
}

func validateCampaignWindow(
	start time.Time,
	end *time.Time,
	visibility int16,
	admin bool,
	now time.Time,
) error {
	if admin {
		return nil
	}
	until := now
	if end != nil {
		until = *end
	}
	days := campaignMaximumOtherWindowDays
	label := "Non-public"
	if visibility == campaignVisibilityPublic {
		days = campaignMaximumPublicWindowDays
		label = "Public"
	}
	if until.Sub(start) > time.Duration(days)*24*time.Hour {
		return apiError(
			http.StatusBadRequest,
			fmt.Sprintf("%s campaigns may cover at most %d days", label, days),
		)
	}
	return nil
}

func updateCampaignColumns(
	ctx context.Context,
	tx pgx.Tx,
	id string,
	updates map[string]any,
) error {
	order := []string{
		"name", "description", "end_time", "visibility", "allowed_entities",
		"status", "processing_paused", "processing_note",
		"last_processing_error", "estimated_killmails",
	}
	args := []any{id}
	sets := make([]string, 0, len(updates)+1)
	for _, column := range order {
		value, found := updates[column]
		if !found {
			continue
		}
		args = append(args, value)
		cast := ""
		if column == "allowed_entities" {
			cast = "::jsonb"
		}
		sets = append(sets, fmt.Sprintf(
			"%s = $%d%s", column, len(args), cast,
		))
	}
	sets = append(sets, "updated_at = now()")
	_, err := tx.Exec(ctx,
		"UPDATE campaigns SET "+strings.Join(sets, ", ")+" WHERE campaign_id = $1",
		args...,
	)
	return err
}

func nullableCampaignLocationJSON(
	location campaignengine.Location,
) (any, error) {
	if !location.HasFilter() {
		return nil, nil
	}
	return json.Marshal(location)
}

func parseCampaignTime(raw any) (time.Time, error) {
	text := strings.TrimSpace(jsonString(raw))
	if text == "" {
		return time.Time{}, fmt.Errorf("missing campaign time")
	}
	parsed, err := time.Parse(time.RFC3339Nano, text)
	if err != nil {
		return time.Time{}, err
	}
	return parsed.UTC(), nil
}

func parseOptionalCampaignTime(raw any) (*time.Time, error) {
	if raw == nil || strings.TrimSpace(jsonString(raw)) == "" {
		return nil, nil
	}
	parsed, err := parseCampaignTime(raw)
	if err != nil {
		return nil, err
	}
	return &parsed, nil
}

func optionalTimeFrom(value any) (*time.Time, bool) {
	if value == nil {
		return nil, true
	}
	parsed, ok := timeFrom(value)
	if !ok {
		return nil, false
	}
	return &parsed, true
}

func sameOptionalTime(left, right *time.Time) bool {
	if left == nil || right == nil {
		return left == nil && right == nil
	}
	return left.Equal(*right)
}

func jsonString(value any) string {
	switch typed := value.(type) {
	case nil:
		return ""
	case string:
		return typed
	case json.Number:
		return typed.String()
	default:
		return fmt.Sprint(value)
	}
}

func runeLength(value string) int { return len([]rune(value)) }

func zeroWalletAmount(raw any) bool {
	amount, err := normalizeWalletAmount(raw)
	if err != nil {
		text := strings.TrimSpace(jsonString(raw))
		return regexp.MustCompile(`^0+(?:\.0{1,2})?$`).MatchString(text)
	}
	return amount == "0.00"
}

func walletAPIError(err error) error {
	var operation *walletOperationError
	if !errors.As(err, &operation) {
		return err
	}
	status := http.StatusBadRequest
	if operation.Code == "campaign_not_fundable" ||
		operation.Code == "idempotency_conflict" {
		status = http.StatusConflict
	}
	return apiError(status, operation.Message)
}
