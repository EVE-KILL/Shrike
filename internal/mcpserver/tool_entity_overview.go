package mcpserver

import (
	"context"
	"fmt"
	"math"
	"strings"
	"time"
)

const (
	statsCharacter     = 0
	statsCorporation   = 1
	statsAlliance      = 2
	statsShip          = 3
	statsSystem        = 4
	statsConstellation = 5
	statsRegion        = 6

	dimensionShipFlown         = 0
	dimensionShipLost          = 1
	dimensionSystem            = 10
	dimensionRegion            = 12
	dimensionDiesToCorporation = 21
	dimensionKilledCorporation = 31
)

type EntityOverviewInput struct {
	Entity StringOrInt64 `json:"entity" jsonschema:"Entity name or numeric identifier."`
	Type   *EntityType   `json:"type,omitempty" enum:"character,corporation,alliance,ship,system,constellation,region" jsonschema:"Entity type, recommended for identifiers and ambiguous names."`
	Format string        `json:"format,omitempty" enum:"json,summary" default:"json" jsonschema:"Use summary for a concise narrative response."`
}

type EntityLifetime struct {
	Kills         int64      `json:"kills"`
	Losses        int64      `json:"losses"`
	SoloKills     int64      `json:"solo_kills"`
	SoloLosses    int64      `json:"solo_losses"`
	NPCLosses     int64      `json:"npc_losses"`
	FinalBlows    int64      `json:"final_blows"`
	Points        int64      `json:"points"`
	ISKDestroyed  float64    `json:"isk_destroyed"`
	ISKLost       float64    `json:"isk_lost"`
	ISKEfficiency float64    `json:"isk_efficiency"`
	FirstSeenYear *time.Time `json:"first_seen_year"`
	LastSeenYear  *time.Time `json:"last_seen_year"`
}

type EntityBreakdown struct {
	ID           int64   `json:"id"`
	Name         *string `json:"name"`
	Kills        int64   `json:"kills"`
	Losses       int64   `json:"losses"`
	ISKDestroyed float64 `json:"isk_destroyed"`
	ISKLost      float64 `json:"isk_lost"`
}

type EntityOverviewOutput struct {
	Entity        Entity            `json:"entity"`
	Lifetime      *EntityLifetime   `json:"lifetime,omitempty"`
	TopShipsFlown []EntityBreakdown `json:"top_ships_flown,omitempty"`
	TopShipsLost  []EntityBreakdown `json:"top_ships_lost,omitempty"`
	TopSystems    []EntityBreakdown `json:"top_systems,omitempty"`
	TopRegions    []EntityBreakdown `json:"top_regions,omitempty"`
	TopTormentors []EntityBreakdown `json:"top_tormentors,omitempty"`
	TopPrey       []EntityBreakdown `json:"top_prey,omitempty"`
	Summary       string            `json:"summary,omitempty"`
}

func registerEntityOverviewTool(registry *Registry) error {
	return addTool(registry, ToolDefinition{
		Name:  "entity_overview",
		Title: "Get entity overview",
		Description: "Return lifetime kills, losses, ISK totals, and common " +
			"breakdowns for a character, corporation, alliance, ship, system, " +
			"constellation, or region.",
	}, func(ctx context.Context, input EntityOverviewInput) (EntityOverviewOutput, error) {
		return entityOverview(ctx, registry.deps, input)
	})
}

func entityOverview(
	ctx context.Context,
	deps Dependencies,
	input EntityOverviewInput,
) (EntityOverviewOutput, error) {
	resolved, err := resolveEntity(ctx, deps, input.Entity, input.Type)
	if err != nil {
		return EntityOverviewOutput{}, fmt.Errorf("resolve entity: %w", err)
	}
	if resolved == nil {
		return EntityOverviewOutput{}, fmt.Errorf("no entity found for %q", input.Entity.String())
	}
	statsType, ok := map[EntityType]int{
		EntityCharacter: statsCharacter, EntityCorporation: statsCorporation,
		EntityAlliance: statsAlliance, EntityShip: statsShip,
		EntitySystem: statsSystem, EntityConstellation: statsConstellation,
		EntityRegion: statsRegion,
	}[resolved.Type]
	if !ok {
		return EntityOverviewOutput{}, fmt.Errorf(
			"stats not tracked for type %s",
			resolved.Type,
		)
	}
	rows, err := queryMaps(ctx, deps.DB, `
		SELECT
			COALESCE(SUM(kills), 0)::bigint AS kills,
			COALESCE(SUM(losses), 0)::bigint AS losses,
			COALESCE(SUM(solo_kills), 0)::bigint AS solo_kills,
			COALESCE(SUM(solo_losses), 0)::bigint AS solo_losses,
			COALESCE(SUM(npc_losses), 0)::bigint AS npc_losses,
			COALESCE(SUM(final_blows), 0)::bigint AS final_blows,
			COALESCE(SUM(points), 0)::bigint AS points,
			COALESCE(SUM(isk_destroyed), 0)::double precision AS isk_destroyed,
			COALESCE(SUM(isk_lost), 0)::double precision AS isk_lost,
			MIN(period_start) AS first_year,
			MAX(period_start) AS last_year
		FROM stats
		WHERE entity_type = $1 AND entity_id = $2 AND period_type = 2`,
		statsType, resolved.ID)
	if err != nil {
		return EntityOverviewOutput{}, fmt.Errorf("load entity totals: %w", err)
	}
	totals := firstMap(rows)
	destroyed := valueFloat64(totals["isk_destroyed"])
	lost := valueFloat64(totals["isk_lost"])
	lifetime := &EntityLifetime{
		Kills:         valueInt64(totals["kills"]),
		Losses:        valueInt64(totals["losses"]),
		SoloKills:     valueInt64(totals["solo_kills"]),
		SoloLosses:    valueInt64(totals["solo_losses"]),
		NPCLosses:     valueInt64(totals["npc_losses"]),
		FinalBlows:    valueInt64(totals["final_blows"]),
		Points:        valueInt64(totals["points"]),
		ISKDestroyed:  destroyed,
		ISKLost:       lost,
		ISKEfficiency: iskEfficiency(destroyed, lost),
		FirstSeenYear: nullableTime(totals["first_year"]),
		LastSeenYear:  nullableTime(totals["last_year"]),
	}
	output := EntityOverviewOutput{
		Entity:   resolved.Public(deps.BaseURL),
		Lifetime: lifetime,
	}
	if statsType <= statsAlliance {
		specs := []struct {
			target    *[]EntityBreakdown
			dimension int
			order     string
			table     string
			id        string
			name      string
		}{
			{&output.TopShipsFlown, dimensionShipFlown, "kills", "inv_types", "type_id", "name"},
			{&output.TopShipsLost, dimensionShipLost, "losses", "inv_types", "type_id", "name"},
			{&output.TopSystems, dimensionSystem, "kills", "solar_systems", "solar_system_id", "system_name"},
			{&output.TopRegions, dimensionRegion, "kills", "regions", "region_id", "name"},
			{&output.TopTormentors, dimensionDiesToCorporation, "losses", "corporations", "corporation_id", "name"},
			{&output.TopPrey, dimensionKilledCorporation, "kills", "corporations", "corporation_id", "name"},
		}
		for _, spec := range specs {
			breakdown, loadErr := topBreakdown(
				ctx, deps, statsType, resolved.ID, spec.dimension,
				spec.order, spec.table, spec.id, spec.name, 5,
			)
			if loadErr != nil {
				return EntityOverviewOutput{}, loadErr
			}
			*spec.target = breakdown
		}
	}
	if input.Format == "summary" {
		return EntityOverviewOutput{
			Entity:  output.Entity,
			Summary: renderEntitySummary(output),
		}, nil
	}
	return output, nil
}

func topBreakdown(
	ctx context.Context,
	deps Dependencies,
	entityType int,
	entityID int64,
	dimension int,
	orderBy, table, idColumn, nameColumn string,
	limit int,
) ([]EntityBreakdown, error) {
	orderColumn := "kills"
	if orderBy == "losses" {
		orderColumn = "losses"
	}
	query := fmt.Sprintf(`
		SELECT
			b.dim_id AS id,
			lookup.%s AS name,
			SUM(b.kills)::bigint AS kills,
			SUM(b.losses)::bigint AS losses,
			SUM(b.isk_destroyed)::double precision AS isk_destroyed,
			SUM(b.isk_lost)::double precision AS isk_lost
		FROM stats_breakdowns b
		LEFT JOIN %s lookup ON lookup.%s = b.dim_id
		WHERE b.entity_type = $1 AND b.entity_id = $2
		  AND b.period_type = 2 AND b.dim_category = $3
		GROUP BY b.dim_id, lookup.%s
		ORDER BY SUM(b.%s) DESC
		LIMIT $4`, nameColumn, table, idColumn, nameColumn, orderColumn)
	rows, err := queryMaps(ctx, deps.DB, query, entityType, entityID, dimension, limit)
	if err != nil {
		return nil, fmt.Errorf("load entity breakdown: %w", err)
	}
	result := make([]EntityBreakdown, 0, len(rows))
	for _, row := range rows {
		result = append(result, EntityBreakdown{
			ID:           valueInt64(row["id"]),
			Name:         nullableString(row["name"]),
			Kills:        valueInt64(row["kills"]),
			Losses:       valueInt64(row["losses"]),
			ISKDestroyed: valueFloat64(row["isk_destroyed"]),
			ISKLost:      valueFloat64(row["isk_lost"]),
		})
	}
	return result, nil
}

func iskEfficiency(destroyed, lost float64) float64 {
	total := destroyed + lost
	if total <= 0 {
		return 0
	}
	return math.Round(destroyed/total*10000) / 100
}

func renderEntitySummary(output EntityOverviewOutput) string {
	lifetime := output.Lifetime
	if lifetime == nil {
		return output.Entity.Name
	}
	ticker := ""
	if output.Entity.Ticker != nil && *output.Entity.Ticker != "" {
		ticker = " [" + *output.Entity.Ticker + "]"
	}
	seen := ""
	if lifetime.FirstSeenYear != nil {
		seen = fmt.Sprintf(" (active since %d)", lifetime.FirstSeenYear.Year())
	}
	parts := []string{fmt.Sprintf(
		"%s%s (%s%s): %d kills / %d losses, %s destroyed vs %s lost, %.1f%% ISK efficiency.",
		output.Entity.Name, ticker, output.Entity.Type, seen,
		lifetime.Kills, lifetime.Losses,
		formatISK(lifetime.ISKDestroyed), formatISK(lifetime.ISKLost),
		lifetime.ISKEfficiency,
	)}
	appendNames := func(label string, rows []EntityBreakdown) {
		names := make([]string, 0, 3)
		for _, row := range rows {
			if row.Name != nil && *row.Name != "" {
				names = append(names, *row.Name)
			}
			if len(names) == 3 {
				break
			}
		}
		if len(names) > 0 {
			parts = append(parts, label+": "+strings.Join(names, ", ")+".")
		}
	}
	appendNames("Favored ships", output.TopShipsFlown)
	appendNames("Most active in", output.TopSystems)
	appendNames("Frequent targets", output.TopPrey)
	appendNames("Frequent killers", output.TopTormentors)
	return strings.Join(parts, " ")
}

func formatISK(value float64) string {
	switch {
	case value >= 1e12:
		return fmt.Sprintf("%.2fT ISK", value/1e12)
	case value >= 1e9:
		return fmt.Sprintf("%.2fB ISK", value/1e9)
	case value >= 1e6:
		return fmt.Sprintf("%.1fM ISK", value/1e6)
	case value >= 1e3:
		return fmt.Sprintf("%.1fK ISK", value/1e3)
	default:
		return fmt.Sprintf("%.0f ISK", value)
	}
}
