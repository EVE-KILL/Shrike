package api

import (
	"context"
	"encoding/json"
	"fmt"
	"time"
)

// KillmailPageData is the public killmail detail read model used by both the
// JSON API and the Go-rendered page. Keep the field names aligned with the
// established /api/killmail/{id} response.
type KillmailPageData struct {
	KillmailID          int64              `json:"killmail_id"`
	KillmailHash        string             `json:"killmail_hash"`
	KillmailTime        time.Time          `json:"killmail_time"`
	Victim              KillmailVictim     `json:"victim"`
	SolarSystemID       int64              `json:"solar_system_id"`
	SolarSystemName     string             `json:"solar_system_name"`
	SolarSystemSecurity *float64           `json:"solar_system_security"`
	ConstellationID     *int64             `json:"constellation_id"`
	ConstellationName   string             `json:"constellation_name"`
	RegionID            *int64             `json:"region_id"`
	RegionName          string             `json:"region_name"`
	Location            *KillmailLocation  `json:"location"`
	TotalValue          float64            `json:"total_value"`
	FittedValue         float64            `json:"fitted_value"`
	DroppedValue        float64            `json:"dropped_value"`
	DestroyedValue      float64            `json:"destroyed_value"`
	Points              int64              `json:"points"`
	AttackerCount       int                `json:"attacker_count"`
	IsNPC               bool               `json:"is_npc"`
	IsSolo              bool               `json:"is_solo"`
	TotalDamage         int64              `json:"total_damage"`
	Attackers           []KillmailAttacker `json:"attackers"`
	Items               []KillmailItem     `json:"items"`
	Siblings            []KillmailSibling  `json:"siblings"`
}

type KillmailVictim struct {
	CharacterID        *int64          `json:"character_id"`
	CharacterName      string          `json:"character_name"`
	CorporationID      *int64          `json:"corporation_id"`
	CorporationName    string          `json:"corporation_name"`
	CorporationPalette json.RawMessage `json:"corporation_palette"`
	AllianceID         *int64          `json:"alliance_id"`
	AllianceName       string          `json:"alliance_name"`
	ShipTypeID         *int64          `json:"ship_type_id"`
	ShipName           string          `json:"ship_name"`
	ShipGroupID        *int64          `json:"ship_group_id"`
	ShipGroupName      string          `json:"ship_group_name"`
	ShipMarketPath     string          `json:"ship_market_path"`
	DamageTaken        int64           `json:"damage_taken"`
	ShipPrice          float64         `json:"ship_price"`
}

type KillmailLocation struct {
	ItemID   int64   `json:"item_id"`
	ItemName string  `json:"item_name"`
	TypeID   *int64  `json:"type_id"`
	GroupID  *int64  `json:"group_id"`
	Distance float64 `json:"distance"`
}

type KillmailAttacker struct {
	CharacterID     *int64   `json:"character_id"`
	CharacterName   string   `json:"character_name"`
	CorporationID   *int64   `json:"corporation_id"`
	CorporationName string   `json:"corporation_name"`
	AllianceID      *int64   `json:"alliance_id"`
	AllianceName    string   `json:"alliance_name"`
	ShipTypeID      *int64   `json:"ship_type_id"`
	ShipCategoryID  *int64   `json:"ship_category_id"`
	ShipName        string   `json:"ship_name"`
	WeaponTypeID    *int64   `json:"weapon_type_id"`
	WeaponName      string   `json:"weapon_name"`
	DamageDone      int64    `json:"damage_done"`
	FinalBlow       bool     `json:"final_blow"`
	SecurityStatus  *float64 `json:"security_status"`
}

type KillmailItem struct {
	ItemIndex         int     `json:"item_index"`
	TypeID            int64   `json:"type_id"`
	TypeName          string  `json:"type_name"`
	GroupID           *int64  `json:"group_id"`
	CategoryID        *int64  `json:"category_id"`
	FlagID            int     `json:"flag_id"`
	FlagName          string  `json:"flag_name"`
	QuantityDropped   int64   `json:"quantity_dropped"`
	QuantityDestroyed int64   `json:"quantity_destroyed"`
	ParentIndex       *int    `json:"parent_index"`
	IsContainer       bool    `json:"is_container"`
	Slot              string  `json:"slot"`
	Price             float64 `json:"price"`
	TotalValue        float64 `json:"total_value"`
}

type KillmailSibling struct {
	KillmailID   int64     `json:"killmail_id"`
	ShipTypeID   *int64    `json:"ship_type_id"`
	ShipGroupID  *int64    `json:"ship_group_id"`
	ShipName     string    `json:"ship_name"`
	TotalValue   float64   `json:"total_value"`
	KillmailTime time.Time `json:"killmail_time"`
}

func (r *HomepageReader) KillmailPage(ctx context.Context, id int64) (KillmailPageData, bool, error) {
	if r == nil || r.db == nil {
		return KillmailPageData{}, false, fmt.Errorf("killmail database is not configured")
	}
	if id <= 0 {
		return KillmailPageData{}, false, nil
	}
	body, found, err := loadKillmailDetail(ctx, r.db, id)
	if err != nil || !found {
		return KillmailPageData{}, found, err
	}
	raw, err := json.Marshal(body)
	if err != nil {
		return KillmailPageData{}, false, err
	}
	var data KillmailPageData
	if err := json.Unmarshal(raw, &data); err != nil {
		return KillmailPageData{}, false, err
	}
	return data, true, nil
}
