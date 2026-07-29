package mcpserver

import (
	"context"
	"fmt"
	"strings"
)

type MeInput struct {
	Me string `json:"me" minLength:"1" doc:"Your EVE character name as typed in-game."`
}

type MeKillsInput struct {
	Me     string  `json:"me" minLength:"1"`
	Role   string  `json:"role,omitempty" enum:"kills,losses,all" default:"all"`
	Limit  int     `json:"limit,omitempty" minimum:"1" maximum:"50" default:"20"`
	Before *int64  `json:"before,omitempty"`
	From   *string `json:"from,omitempty"`
	To     *string `json:"to,omitempty"`
}

type MeIntelInput struct {
	Me    string `json:"me" minLength:"1"`
	Limit int    `json:"limit,omitempty" minimum:"1" maximum:"50" default:"10"`
}

type MeShipsUsedInput struct {
	Me      string         `json:"me" minLength:"1"`
	Ship    *StringOrInt64 `json:"ship,omitempty"`
	Role    string         `json:"role,omitempty" enum:"kills,losses,all" default:"all"`
	From    *string        `json:"from,omitempty"`
	To      *string        `json:"to,omitempty"`
	System  *StringOrInt64 `json:"system,omitempty"`
	Region  *StringOrInt64 `json:"region,omitempty"`
	GroupBy string         `json:"group_by,omitempty" enum:"none,ship,victim_ship,system,region,month" default:"ship"`
	Limit   int            `json:"limit,omitempty" minimum:"1" maximum:"50" default:"10"`
}

type MeKillsWithInput struct {
	Me           string         `json:"me" minLength:"1"`
	Partner      StringOrInt64  `json:"partner"`
	EntityShip   *StringOrInt64 `json:"entity_ship,omitempty"`
	PartnerShip  *StringOrInt64 `json:"partner_ship,omitempty"`
	VictimShip   *StringOrInt64 `json:"victim_ship,omitempty"`
	VictimEntity *StringOrInt64 `json:"victim_entity,omitempty"`
	System       *StringOrInt64 `json:"system,omitempty"`
	Region       *StringOrInt64 `json:"region,omitempty"`
	From         *string        `json:"from,omitempty"`
	To           *string        `json:"to,omitempty"`
	GroupBy      string         `json:"group_by,omitempty" enum:"none,victim_ship,system,region,month,partner_ship,entity_ship" default:"none"`
	Limit        int            `json:"limit,omitempty" minimum:"1" maximum:"50" default:"10"`
}

type MeTimelineInput struct {
	Me     string         `json:"me" minLength:"1"`
	Bucket string         `json:"bucket,omitempty" enum:"day,month,year" default:"month"`
	Since  *string        `json:"since,omitempty"`
	Until  *string        `json:"until,omitempty"`
	VS     *StringOrInt64 `json:"vs,omitempty"`
}

func registerMeTools(registry *Registry) error {
	if err := addTool(registry, ToolDefinition{
		Name: "me_dossier", Title: "Build my capsuleer dossier",
		Description: "Your own full intelligence dossier with lifetime stats, archetype tags, playstyle, ships, systems, and wingmates.",
	}, func(ctx context.Context, input MeInput) (DossierOutput, error) {
		id, err := resolveMe(ctx, registry.deps, input.Me)
		if err != nil {
			return DossierOutput{}, err
		}
		return capsuleerDossier(ctx, registry.deps, DossierInput{Entity: IntRef(id), Type: entityTypePointer(EntityCharacter)})
	}); err != nil {
		return err
	}
	if err := addTool(registry, ToolDefinition{
		Name: "me_overview", Title: "Get my combat overview",
		Description: "Your lifetime kill and loss statistics with top ships, systems, prey, and tormentors.",
	}, func(ctx context.Context, input MeInput) (EntityOverviewOutput, error) {
		id, err := resolveMe(ctx, registry.deps, input.Me)
		if err != nil {
			return EntityOverviewOutput{}, err
		}
		return entityOverview(ctx, registry.deps, EntityOverviewInput{Entity: IntRef(id), Type: entityTypePointer(EntityCharacter)})
	}); err != nil {
		return err
	}
	if err := addTool(registry, ToolDefinition{
		Name: "me_kills", Title: "List my recent killmails",
		Description: "Your recent killmails, optionally filtered to kills or losses and bounded by time or pagination cursor.",
	}, func(ctx context.Context, input MeKillsInput) (EntityKillsOutput, error) {
		id, err := resolveMe(ctx, registry.deps, input.Me)
		if err != nil {
			return EntityKillsOutput{}, err
		}
		return entityKills(ctx, registry.deps, EntityKillsInput{Entity: IntRef(id), Type: entityTypePointer(EntityCharacter), Role: input.Role, Limit: input.Limit, Before: input.Before, From: input.From, To: input.To})
	}); err != nil {
		return err
	}
	if err := registerMeIntelTools(registry); err != nil {
		return err
	}
	if err := addTool(registry, ToolDefinition{
		Name: "me_ships_used", Title: "Analyze my ship usage",
		Description: "Your kills and losses broken down by ship, victim ship, location, or month with optional filters.",
	}, func(ctx context.Context, input MeShipsUsedInput) (ShipsUsedOutput, error) {
		id, err := resolveMe(ctx, registry.deps, input.Me)
		if err != nil {
			return ShipsUsedOutput{}, err
		}
		return shipsUsed(ctx, registry.deps, ShipsUsedInput{
			Entity: IntRef(id), Type: entityTypePointer(EntityCharacter), Ship: input.Ship,
			Role: input.Role, From: input.From, To: input.To, System: input.System,
			Region: input.Region, GroupBy: input.GroupBy, Limit: input.Limit,
		})
	}); err != nil {
		return err
	}
	if err := addTool(registry, ToolDefinition{
		Name: "me_kills_with", Title: "Count my shared kills",
		Description: "Count killmails you share with a partner, with optional ship, victim, location, time, and breakdown filters.",
	}, func(ctx context.Context, input MeKillsWithInput) (KillsWithOutput, error) {
		id, err := resolveMe(ctx, registry.deps, input.Me)
		if err != nil {
			return KillsWithOutput{}, err
		}
		return killsWith(ctx, registry.deps, KillsWithInput{
			Entity: IntRef(id), Type: entityTypePointer(EntityCharacter), Partner: input.Partner,
			EntityShip: input.EntityShip, PartnerShip: input.PartnerShip, VictimShip: input.VictimShip,
			VictimEntity: input.VictimEntity, System: input.System, Region: input.Region,
			From: input.From, To: input.To, GroupBy: input.GroupBy, Limit: input.Limit,
		})
	}); err != nil {
		return err
	}
	return addTool(registry, ToolDefinition{
		Name: "me_timeline", Title: "Get my combat activity timeline",
		Description: "Your combat activity over time, bucketed by day, month, or year and optionally restricted to one opponent.",
	}, func(ctx context.Context, input MeTimelineInput) (EntityTimelineOutput, error) {
		id, err := resolveMe(ctx, registry.deps, input.Me)
		if err != nil {
			return EntityTimelineOutput{}, err
		}
		return entityTimeline(ctx, registry.deps, EntityTimelineInput{Entity: IntRef(id), Type: entityTypePointer(EntityCharacter), Bucket: input.Bucket, Since: input.Since, Until: input.Until, VS: input.VS})
	})
}

func registerMeIntelTools(registry *Registry) error {
	if err := addTool(registry, ToolDefinition{Name: "me_flies_with", Title: "Find my frequent wingmates", Description: "Your frequent wingmates from the last 90 days."},
		func(ctx context.Context, input MeIntelInput) (FliesWithOutput, error) {
			id, err := resolveMe(ctx, registry.deps, input.Me)
			if err != nil {
				return FliesWithOutput{}, err
			}
			return fliesWith(ctx, registry.deps, CharacterIntelInput{Entity: IntRef(id), Type: entityTypePointer(EntityCharacter), Limit: input.Limit})
		}); err != nil {
		return err
	}
	if err := addTool(registry, ToolDefinition{Name: "me_hunts_in", Title: "Find my hunting grounds", Description: "Your preferred hunting systems from the last 90 days."},
		func(ctx context.Context, input MeIntelInput) (HuntsInOutput, error) {
			id, err := resolveMe(ctx, registry.deps, input.Me)
			if err != nil {
				return HuntsInOutput{}, err
			}
			return huntsIn(ctx, registry.deps, CharacterIntelInput{Entity: IntRef(id), Type: entityTypePointer(EntityCharacter), Limit: input.Limit})
		}); err != nil {
		return err
	}
	if err := addTool(registry, ToolDefinition{Name: "me_hunted_by", Title: "Find who hunts me", Description: "Characters who most often killed you in the last 90 days."},
		func(ctx context.Context, input MeIntelInput) (CharacterIntelOutput, error) {
			id, err := resolveMe(ctx, registry.deps, input.Me)
			if err != nil {
				return CharacterIntelOutput{}, err
			}
			return huntedBy(ctx, registry.deps, CharacterIntelInput{Entity: IntRef(id), Type: entityTypePointer(EntityCharacter), Limit: input.Limit})
		}); err != nil {
		return err
	}
	return addTool(registry, ToolDefinition{Name: "me_preys_on", Title: "Find who I prey on", Description: "Characters you most often killed in the last 90 days."},
		func(ctx context.Context, input MeIntelInput) (CharacterIntelOutput, error) {
			id, err := resolveMe(ctx, registry.deps, input.Me)
			if err != nil {
				return CharacterIntelOutput{}, err
			}
			return preysOn(ctx, registry.deps, CharacterIntelInput{Entity: IntRef(id), Type: entityTypePointer(EntityCharacter), Limit: input.Limit})
		})
}

func resolveMe(ctx context.Context, deps Dependencies, name string) (int64, error) {
	name = strings.TrimSpace(name)
	if name == "" {
		return 0, fmt.Errorf("missing me: pass your EVE character name")
	}
	resolved, err := resolveEntity(ctx, deps, StringRef(name), entityTypePointer(EntityCharacter))
	if err != nil {
		return 0, err
	}
	if resolved == nil || resolved.Type != EntityCharacter {
		return 0, fmt.Errorf("could not resolve %q to an EVE character", name)
	}
	return resolved.ID, nil
}
