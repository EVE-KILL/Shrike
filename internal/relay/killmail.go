package relay

import (
	"time"

	"github.com/eve-kill/shrike/internal/eve"
	"github.com/eve-kill/shrike/internal/killmail"
)

// KillmailEvent is the full hydrated event consumed by the /killmails relay
// channel. Keep this distinct from KilllistEvent: the latter is a compact,
// flattened row for list pages, while this mirrors the ESI-shaped detail
// payload emitted by the TypeScript worker.
type KillmailEvent struct {
	Event        string `json:"event"`
	KillmailID   int64  `json:"killmail_id"`
	KillmailHash string `json:"killmail_hash"`
	KillmailTime string `json:"killmail_time"`

	SolarSystemID       int32    `json:"solar_system_id"`
	SolarSystemName     *string  `json:"solar_system_name"`
	SolarSystemSecurity *float64 `json:"solar_system_security"`
	ConstellationID     *int32   `json:"constellation_id"`
	ConstellationName   *string  `json:"constellation_name"`
	RegionID            *int32   `json:"region_id"`
	RegionName          *string  `json:"region_name"`

	TotalValue     float64 `json:"total_value"`
	FittedValue    float64 `json:"fitted_value"`
	DroppedValue   float64 `json:"dropped_value"`
	DestroyedValue float64 `json:"destroyed_value"`
	Points         int32   `json:"points"`
	IsSolo         bool    `json:"is_solo"`
	IsNPC          bool    `json:"is_npc"`
	WarID          int32   `json:"war_id"`

	Victim    KillmailVictim     `json:"victim"`
	Attackers []KillmailAttacker `json:"attackers"`
}

type KillmailVictim struct {
	CharacterID   int32 `json:"character_id"`
	CorporationID int32 `json:"corporation_id"`
	AllianceID    int32 `json:"alliance_id"`
	FactionID     int32 `json:"faction_id"`
	ShipTypeID    int32 `json:"ship_type_id"`
	DamageTaken   int32 `json:"damage_taken"`

	CharacterName   *string `json:"character_name"`
	CorporationName *string `json:"corporation_name"`
	AllianceName    *string `json:"alliance_name"`
	ShipTypeName    *string `json:"ship_type_name"`
	ShipGroupID     *int32  `json:"ship_group_id"`
	ShipGroupName   *string `json:"ship_group_name"`

	Position *killmail.ESIPosition `json:"position,omitempty"`
	Items    []KillmailItem        `json:"items"`
}

type KillmailItem struct {
	ItemTypeID        int32   `json:"item_type_id"`
	Flag              int32   `json:"flag"`
	QuantityDropped   int64   `json:"quantity_dropped"`
	QuantityDestroyed int64   `json:"quantity_destroyed"`
	Singleton         int16   `json:"singleton"`
	ItemTypeName      *string `json:"item_type_name"`
	ItemGroupID       *int32  `json:"item_group_id"`
	ItemGroupName     *string `json:"item_group_name"`
	ItemCategoryID    *int32  `json:"item_category_id"`
	Value             float64 `json:"value"`
}

type KillmailAttacker struct {
	CharacterID    int32   `json:"character_id"`
	CorporationID  int32   `json:"corporation_id"`
	AllianceID     int32   `json:"alliance_id"`
	FactionID      int32   `json:"faction_id"`
	ShipTypeID     int32   `json:"ship_type_id"`
	WeaponTypeID   int32   `json:"weapon_type_id"`
	DamageDone     int32   `json:"damage_done"`
	FinalBlow      bool    `json:"final_blow"`
	SecurityStatus float64 `json:"security_status"`

	CharacterName   *string `json:"character_name"`
	CorporationName *string `json:"corporation_name"`
	AllianceName    *string `json:"alliance_name"`
	ShipTypeName    *string `json:"ship_type_name"`
	ShipGroupID     *int32  `json:"ship_group_id"`
	ShipGroupName   *string `json:"ship_group_name"`
	WeaponTypeName  *string `json:"weapon_type_name"`
}

// BuildKillmailEvent hydrates the stored rows with the same cache, name, and
// price information as backend/src/queues/killmails/queue.ts.
func BuildKillmailEvent(
	p *killmail.Parsed,
	cache *eve.Cache,
	names EntityNames,
	price func(int32) float64,
) KillmailEvent {
	km := p.Killmail
	out := KillmailEvent{
		Event:           "killmail",
		KillmailID:      km.KillmailID,
		KillmailHash:    km.KillmailHash,
		KillmailTime:    isoMillis(km.KillmailTime),
		SolarSystemID:   km.SolarSystemID,
		ConstellationID: idOrNil(km.ConstellationID),
		RegionID:        idOrNil(km.RegionID),
		TotalValue:      km.TotalValue,
		FittedValue:     km.FittedValue,
		DroppedValue:    km.DroppedValue,
		DestroyedValue:  km.DestroyedValue,
		Points:          km.Points,
		IsSolo:          km.IsSolo,
		IsNPC:           km.IsNPC,
		WarID:           km.WarID,
		Attackers:       make([]KillmailAttacker, 0, len(p.Attackers)),
	}

	if s, ok := cache.System(km.SolarSystemID); ok {
		out.SolarSystemName = strOrNil(s.Name)
		security := s.Security
		out.SolarSystemSecurity = &security
	}
	if c, ok := cache.Constellation(km.ConstellationID); ok {
		out.ConstellationName = strOrNil(c.Name)
	}
	if r, ok := cache.Region(km.RegionID); ok {
		out.RegionName = strOrNil(r.Name)
	}

	out.Victim = KillmailVictim{
		CharacterID:     km.VictimCharacterID,
		CorporationID:   km.VictimCorporationID,
		AllianceID:      km.VictimAllianceID,
		FactionID:       km.VictimFactionID,
		ShipTypeID:      km.VictimShipTypeID,
		DamageTaken:     km.VictimDamageTaken,
		CharacterName:   strOrNil(names.Characters[km.VictimCharacterID]),
		CorporationName: strOrNil(names.Corporations[km.VictimCorporationID]),
		AllianceName:    strOrNil(names.Alliances[km.VictimAllianceID]),
		Position:        km.Position,
		Items:           make([]KillmailItem, 0, len(p.Items)),
	}
	if t, ok := cache.Type(km.VictimShipTypeID); ok {
		out.Victim.ShipTypeName = strOrNil(t.Name)
		out.Victim.ShipGroupID = idOrNil(t.GroupID)
		if g, ok := cache.Group(t.GroupID); ok {
			out.Victim.ShipGroupName = strOrNil(g.Name)
		}
	}

	for _, item := range p.Items {
		hydrated := KillmailItem{
			ItemTypeID:        item.TypeID,
			Flag:              item.FlagID,
			QuantityDropped:   item.QuantityDropped,
			QuantityDestroyed: item.QuantityDestroyed,
			Singleton:         item.Singleton,
		}
		if item.Singleton == 2 {
			hydrated.Value = 0.01
		} else if price != nil {
			hydrated.Value = price(item.TypeID)
		}
		if t, ok := cache.Type(item.TypeID); ok {
			hydrated.ItemTypeName = strOrNil(t.Name)
			hydrated.ItemGroupID = idOrNil(t.GroupID)
			hydrated.ItemCategoryID = idOrNil(t.CategoryID)
			if g, ok := cache.Group(t.GroupID); ok {
				hydrated.ItemGroupName = strOrNil(g.Name)
			}
		}
		out.Victim.Items = append(out.Victim.Items, hydrated)
	}

	for _, attacker := range p.Attackers {
		hydrated := KillmailAttacker{
			CharacterID:     attacker.CharacterID,
			CorporationID:   attacker.CorporationID,
			AllianceID:      attacker.AllianceID,
			FactionID:       attacker.FactionID,
			ShipTypeID:      attacker.ShipTypeID,
			WeaponTypeID:    attacker.WeaponTypeID,
			DamageDone:      attacker.DamageDone,
			FinalBlow:       attacker.FinalBlow,
			CharacterName:   strOrNil(names.Characters[attacker.CharacterID]),
			CorporationName: strOrNil(names.Corporations[attacker.CorporationID]),
			AllianceName:    strOrNil(names.Alliances[attacker.AllianceID]),
		}
		if attacker.SecurityStatus != nil {
			hydrated.SecurityStatus = *attacker.SecurityStatus
		}
		if t, ok := cache.Type(attacker.ShipTypeID); ok {
			hydrated.ShipTypeName = strOrNil(t.Name)
			hydrated.ShipGroupID = idOrNil(t.GroupID)
			if g, ok := cache.Group(t.GroupID); ok {
				hydrated.ShipGroupName = strOrNil(g.Name)
			}
		}
		if t, ok := cache.Type(attacker.WeaponTypeID); ok {
			hydrated.WeaponTypeName = strOrNil(t.Name)
		}
		out.Attackers = append(out.Attackers, hydrated)
	}

	return out
}

func isoMillis(t time.Time) string {
	return t.UTC().Format("2006-01-02T15:04:05.000Z")
}
