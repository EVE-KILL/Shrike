package mcpserver

import (
	"context"
	"time"
)

type GlobalPulseInput struct {
	Hours float64 `json:"hours,omitempty" default:"1" minimum:"0.25" maximum:"24" doc:"Lookback window in hours (max 24)."`
	TopN  int     `json:"top_n,omitempty" default:"10" minimum:"1" maximum:"20" doc:"How many rows in each list."`
}

type GlobalPulseTotals struct {
	Kills         int64   `json:"kills"`
	ISKDestroyed  float64 `json:"isk_destroyed"`
	SystemsActive int64   `json:"systems_active"`
	SoloKills     int64   `json:"solo_kills"`
}

type GlobalPulseSystem struct {
	SolarSystemID int64      `json:"solar_system_id"`
	Name          *string    `json:"name"`
	RegionName    *string    `json:"region_name"`
	Security      *float64   `json:"security"`
	Kills         int64      `json:"kills"`
	ISKDestroyed  float64    `json:"isk_destroyed"`
	LatestKill    *time.Time `json:"latest_kill"`
	URL           string     `json:"url"`
}

type GlobalPulseAlliance struct {
	AllianceID    int64   `json:"alliance_id"`
	Name          *string `json:"name"`
	Ticker        *string `json:"ticker"`
	Kills         int64   `json:"kills"`
	SystemsActive int64   `json:"systems_active"`
	URL           string  `json:"url"`
}

type GlobalPulseCorporation struct {
	CorporationID int64   `json:"corporation_id"`
	Name          *string `json:"name"`
	Ticker        *string `json:"ticker"`
	AllianceID    *int64  `json:"alliance_id"`
	AllianceName  *string `json:"alliance_name"`
	Kills         int64   `json:"kills"`
	URL           string  `json:"url"`
}

type GlobalPulseOutput struct {
	WindowHours            float64                  `json:"window_hours"`
	Since                  string                   `json:"since"`
	Totals                 GlobalPulseTotals        `json:"totals"`
	HottestSystems         []GlobalPulseSystem      `json:"hottest_systems"`
	MostActiveAlliances    []GlobalPulseAlliance    `json:"most_active_alliances"`
	MostActiveCorporations []GlobalPulseCorporation `json:"most_active_corporations"`
}

func registerGlobalPulseTool(registry *Registry) error {
	return addTool(registry, ToolDefinition{
		Name:  "global_pulse",
		Title: "Show current global combat activity",
		Description: "Global activity firehose: hottest solar systems, most active alliances and corporations over a recent window (default 1 hour). " +
			"Scouting entry point when you don't know where to start. Use system_pulse to drill into a specific hot system.",
	}, func(ctx context.Context, input GlobalPulseInput) (GlobalPulseOutput, error) {
		return globalPulse(ctx, registry.deps, input)
	})
}

func globalPulse(ctx context.Context, deps Dependencies, input GlobalPulseInput) (GlobalPulseOutput, error) {
	hours := input.Hours
	if hours == 0 {
		hours = 1
	}
	if hours < 0.25 {
		hours = 0.25
	}
	if hours > 24 {
		hours = 24
	}
	limit := input.TopN
	if limit == 0 {
		limit = 10
	}
	limit = clamp(limit, 1, 20)
	sinceTime := time.Now().UTC().Add(-time.Duration(hours * float64(time.Hour)))
	since := sinceTime.Format(time.RFC3339Nano)

	totalsRows, err := queryMaps(ctx, deps.DB, `
		SELECT COUNT(*)::bigint AS kills,
		       COALESCE(SUM(total_value), 0)::double precision AS isk_destroyed,
		       COUNT(DISTINCT solar_system_id)::bigint AS systems_active,
		       COUNT(*) FILTER (WHERE is_solo = true)::bigint AS solo_kills
		FROM killmails
		WHERE killmail_time >= $1`, sinceTime)
	if err != nil {
		return GlobalPulseOutput{}, err
	}
	systemRows, err := queryMaps(ctx, deps.DB, `
		SELECT k.solar_system_id, s.system_name, r.name AS region_name, s.security,
		       COUNT(*)::bigint AS kills,
		       COALESCE(SUM(k.total_value), 0)::double precision AS isk_destroyed,
		       MAX(k.killmail_time) AS latest_kill
		FROM killmails k
		LEFT JOIN solar_systems s ON s.solar_system_id = k.solar_system_id
		LEFT JOIN regions r ON r.region_id = k.region_id
		WHERE k.killmail_time >= $1
		GROUP BY k.solar_system_id, s.system_name, r.name, s.security
		ORDER BY COUNT(*) DESC, SUM(k.total_value) DESC NULLS LAST
		LIMIT $2`, sinceTime, limit)
	if err != nil {
		return GlobalPulseOutput{}, err
	}
	allianceRows, err := queryMaps(ctx, deps.DB, `
		SELECT a.alliance_id, al.name, al.ticker,
		       COUNT(DISTINCT a.killmail_id)::bigint AS kills,
		       COUNT(DISTINCT k.solar_system_id)::bigint AS systems_active
		FROM killmail_attackers a
		JOIN killmails k ON k.killmail_id = a.killmail_id
		LEFT JOIN alliances al ON al.alliance_id = a.alliance_id
		WHERE a.killmail_time >= $1 AND a.alliance_id IS NOT NULL
		GROUP BY a.alliance_id, al.name, al.ticker
		ORDER BY kills DESC
		LIMIT $2`, sinceTime, limit)
	if err != nil {
		return GlobalPulseOutput{}, err
	}
	corporationRows, err := queryMaps(ctx, deps.DB, `
		SELECT a.corporation_id, co.name, co.ticker, co.alliance_id, al.name AS alliance_name,
		       COUNT(DISTINCT a.killmail_id)::bigint AS kills
		FROM killmail_attackers a
		LEFT JOIN corporations co ON co.corporation_id = a.corporation_id
		LEFT JOIN alliances al ON al.alliance_id = co.alliance_id
		WHERE a.killmail_time >= $1 AND a.corporation_id IS NOT NULL
		GROUP BY a.corporation_id, co.name, co.ticker, co.alliance_id, al.name
		ORDER BY kills DESC
		LIMIT $2`, sinceTime, limit)
	if err != nil {
		return GlobalPulseOutput{}, err
	}

	out := GlobalPulseOutput{
		WindowHours:            hours,
		Since:                  since,
		HottestSystems:         make([]GlobalPulseSystem, 0, len(systemRows)),
		MostActiveAlliances:    make([]GlobalPulseAlliance, 0, len(allianceRows)),
		MostActiveCorporations: make([]GlobalPulseCorporation, 0, len(corporationRows)),
	}
	if row := firstMap(totalsRows); row != nil {
		out.Totals = GlobalPulseTotals{
			Kills:         valueInt64(row["kills"]),
			ISKDestroyed:  valueFloat64(row["isk_destroyed"]),
			SystemsActive: valueInt64(row["systems_active"]),
			SoloKills:     valueInt64(row["solo_kills"]),
		}
	}
	for _, row := range systemRows {
		id := valueInt64(row["solar_system_id"])
		out.HottestSystems = append(out.HottestSystems, GlobalPulseSystem{
			SolarSystemID: id,
			Name:          nullableString(row["system_name"]),
			RegionName:    nullableString(row["region_name"]),
			Security:      nullableFloat64(row["security"]),
			Kills:         valueInt64(row["kills"]),
			ISKDestroyed:  valueFloat64(row["isk_destroyed"]),
			LatestKill:    nullableTime(row["latest_kill"]),
			URL:           entityURL(deps.BaseURL, EntitySystem, id),
		})
	}
	for _, row := range allianceRows {
		id := valueInt64(row["alliance_id"])
		out.MostActiveAlliances = append(out.MostActiveAlliances, GlobalPulseAlliance{
			AllianceID: id,
			Name:       nullableString(row["name"]), Ticker: nullableString(row["ticker"]),
			Kills: valueInt64(row["kills"]), SystemsActive: valueInt64(row["systems_active"]),
			URL: entityURL(deps.BaseURL, EntityAlliance, id),
		})
	}
	for _, row := range corporationRows {
		id := valueInt64(row["corporation_id"])
		out.MostActiveCorporations = append(out.MostActiveCorporations, GlobalPulseCorporation{
			CorporationID: id,
			Name:          nullableString(row["name"]), Ticker: nullableString(row["ticker"]),
			AllianceID: nullableInt64(row["alliance_id"]), AllianceName: nullableString(row["alliance_name"]),
			Kills: valueInt64(row["kills"]), URL: entityURL(deps.BaseURL, EntityCorporation, id),
		})
	}
	return out, nil
}
