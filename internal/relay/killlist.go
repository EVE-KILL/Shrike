package relay

import (
	"context"
	"time"

	"github.com/eve-kill/shrike/internal/eve"
	"github.com/eve-kill/shrike/internal/killmail"
	"github.com/jackc/pgx/v5/pgxpool"
)

// KilllistRow is the flattened killmail summary the killlist UI renders.
//
// Two independent producers emit this shape and they must stay identical: the
// REST resolver behind the /api/killlist/* endpoints, and the live event this
// package publishes and the client prepends to page one. They have drifted
// before — the relay payload was once missing ship_market_path, so a live row
// rendered without its ship link until you reloaded the page, and missing
// meta_group_id, so the client had to download a separate set of ship type ids
// just to evaluate the tech-level filter.
//
// Every field is therefore a pointer or has an explicit zero default matching
// what the REST resolver applies. The nullable ones are pointers because the
// client distinguishes absent from zero: a kill with no recorded attacker count
// is not a kill with zero attackers.
type KilllistRow struct {
	KillmailID   int64  `json:"killmail_id"`
	KillmailHash string `json:"killmail_hash"`

	// KillmailTime is the wire shape — an ISO string, not a timestamp — because
	// this is what the browser receives.
	KillmailTime string `json:"killmail_time"`

	// These four are nullable in the schema but the REST resolver defaults them
	// rather than forwarding null, so a live row and a refetched one agree on a
	// kill with no value or no attacker count recorded.
	TotalValue    float64 `json:"total_value"`
	AttackerCount int32   `json:"attacker_count"`
	IsNPC         bool    `json:"is_npc"`
	IsSolo        bool    `json:"is_solo"`

	ShipTypeID     *int32  `json:"ship_type_id"`
	ShipName       *string `json:"ship_name"`
	ShipGroupName  *string `json:"ship_group_name"`
	ShipMarketPath *string `json:"ship_market_path"`
	MetaGroupID    *int32  `json:"meta_group_id"`

	VictimCharacterID     *int32  `json:"victim_character_id"`
	VictimCharacterName   *string `json:"victim_character_name"`
	VictimCorporationID   *int32  `json:"victim_corporation_id"`
	VictimCorporationName *string `json:"victim_corporation_name"`
	VictimAllianceID      *int32  `json:"victim_alliance_id"`
	VictimAllianceName    *string `json:"victim_alliance_name"`

	FinalBlowCharacterID     *int32  `json:"final_blow_character_id"`
	FinalBlowCharacterName   *string `json:"final_blow_character_name"`
	FinalBlowCorporationID   *int32  `json:"final_blow_corporation_id"`
	FinalBlowCorporationName *string `json:"final_blow_corporation_name"`
	FinalBlowAllianceID      *int32  `json:"final_blow_alliance_id"`
	FinalBlowAllianceName    *string `json:"final_blow_alliance_name"`
	FinalBlowShipTypeID      *int32  `json:"final_blow_ship_type_id"`
	FinalBlowShipName        *string `json:"final_blow_ship_name"`

	SolarSystemID       int32    `json:"solar_system_id"`
	SolarSystemName     *string  `json:"solar_system_name"`
	SolarSystemSecurity *float64 `json:"solar_system_security"`
	RegionID            *int32   `json:"region_id"`
	RegionName          *string  `json:"region_name"`
}

// EntityNames holds the names a killlist row needs.
type EntityNames struct {
	Characters   map[int32]string
	Corporations map[int32]string
	Alliances    map[int32]string
}

// LookupNames reads the names for the entities a killmail row references.
//
// One query per kind regardless of attacker count. Only the victim and the
// final blow appear on a killlist row, so this is at most six ids — but the
// query shape is the same either way and this keeps the call site simple.
func LookupNames(ctx context.Context, pool *pgxpool.Pool, characters, corporations, alliances []int32) (EntityNames, error) {
	out := EntityNames{
		Characters:   map[int32]string{},
		Corporations: map[int32]string{},
		Alliances:    map[int32]string{},
	}

	for _, q := range []struct {
		table, column string
		into          map[int32]string
		ids           []int32
	}{
		{"characters", "character_id", out.Characters, characters},
		{"corporations", "corporation_id", out.Corporations, corporations},
		{"alliances", "alliance_id", out.Alliances, alliances},
	} {
		ids := nonZero(q.ids)
		if len(ids) == 0 {
			continue
		}
		// Identifiers come from this fixed list, never from input.
		rows, err := pool.Query(ctx,
			"SELECT "+q.column+", name FROM "+q.table+" WHERE "+q.column+" = ANY($1::int[])", ids)
		if err != nil {
			return out, err
		}
		for rows.Next() {
			var id int32
			var name string
			if err := rows.Scan(&id, &name); err != nil {
				rows.Close()
				return out, err
			}
			q.into[id] = name
		}
		rows.Close()
		if err := rows.Err(); err != nil {
			return out, err
		}
	}
	return out, nil
}

// BuildKilllistRow flattens a parsed killmail into the wire shape.
func BuildKilllistRow(p *killmail.Parsed, cache *eve.Cache, paths eve.MarketPaths, names EntityNames) KilllistRow {
	km := p.Killmail

	row := KilllistRow{
		KillmailID:    km.KillmailID,
		KillmailHash:  km.KillmailHash,
		KillmailTime:  km.KillmailTime.UTC().Format(time.RFC3339),
		TotalValue:    km.TotalValue,
		AttackerCount: km.AttackerCount,
		IsNPC:         km.IsNPC,
		IsSolo:        km.IsSolo,
		SolarSystemID: km.SolarSystemID,

		ShipTypeID:          idOrNil(km.VictimShipTypeID),
		VictimCharacterID:   idOrNil(km.VictimCharacterID),
		VictimCorporationID: idOrNil(km.VictimCorporationID),
		VictimAllianceID:    idOrNil(km.VictimAllianceID),
		RegionID:            idOrNil(km.RegionID),
	}

	if t, ok := cache.Type(km.VictimShipTypeID); ok {
		row.ShipName = strOrNil(t.Name)
		row.MetaGroupID = idOrNil(t.MetaGroupID)
		row.ShipMarketPath = strOrNil(paths.Path(t.MarketGroupID))
		if g, ok := cache.Group(t.GroupID); ok {
			row.ShipGroupName = strOrNil(g.Name)
		}
	}

	if s, ok := cache.System(km.SolarSystemID); ok {
		row.SolarSystemName = strOrNil(s.Name)
		sec := s.Security
		row.SolarSystemSecurity = &sec
	}
	if r, ok := cache.Region(km.RegionID); ok {
		row.RegionName = strOrNil(r.Name)
	}

	row.VictimCharacterName = strOrNil(names.Characters[km.VictimCharacterID])
	row.VictimCorporationName = strOrNil(names.Corporations[km.VictimCorporationID])
	row.VictimAllianceName = strOrNil(names.Alliances[km.VictimAllianceID])

	// The final blow is the attacker the UI credits the kill to. A killmail
	// always has one, but a malformed or NPC-only mail may not, so this stays
	// entirely absent rather than defaulting to the first attacker.
	for _, a := range p.Attackers {
		if !a.FinalBlow {
			continue
		}
		row.FinalBlowCharacterID = idOrNil(a.CharacterID)
		row.FinalBlowCorporationID = idOrNil(a.CorporationID)
		row.FinalBlowAllianceID = idOrNil(a.AllianceID)
		row.FinalBlowShipTypeID = idOrNil(a.ShipTypeID)

		row.FinalBlowCharacterName = strOrNil(names.Characters[a.CharacterID])
		row.FinalBlowCorporationName = strOrNil(names.Corporations[a.CorporationID])
		row.FinalBlowAllianceName = strOrNil(names.Alliances[a.AllianceID])
		if t, ok := cache.Type(a.ShipTypeID); ok {
			row.FinalBlowShipName = strOrNil(t.Name)
		}
		break
	}

	return row
}

// KilllistEvent is what the killlist channel carries.
type KilllistEvent struct {
	Event    string      `json:"event"`
	Killmail KilllistRow `json:"killmail"`
}

func nonZero(ids []int32) []int32 {
	seen := map[int32]bool{}
	var out []int32
	for _, id := range ids {
		if id == 0 || seen[id] {
			continue
		}
		seen[id] = true
		out = append(out, id)
	}
	return out
}

func idOrNil(v int32) *int32 {
	if v == 0 {
		return nil
	}
	return &v
}

func strOrNil(s string) *string {
	if s == "" {
		return nil
	}
	return &s
}
