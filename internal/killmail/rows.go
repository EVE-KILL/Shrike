package killmail

import "time"

// The stored form.
//
// Nullable ID columns are plain int32 here and become NULL on insert when they
// are zero, because EVE issues no entity with ID 0 — the two representations
// have never been distinguishable in this schema. Columns where zero is a real
// value say so and use a pointer or a sentinel.

// Killmail is one row of the killmails table.
//
// The JSON names are the column names, not Go-idiomatic ones: this shape is
// diffed against the TypeScript parser's output during the port, and later
// serves the API, where the column name is what a consumer expects.
type Killmail struct {
	KillmailID   int64     `json:"killmail_id"`
	KillmailTime time.Time `json:"killmail_time"`
	KillmailHash string    `json:"killmail_hash"`

	SolarSystemID   int32 `json:"solar_system_id"`
	ConstellationID int32 `json:"constellation_id"`
	RegionID        int32 `json:"region_id"`

	// Position is stored as three columns but is present or absent as a unit.
	Position *ESIPosition `json:"position"`

	VictimCharacterID   int32 `json:"victim_character_id"`
	VictimCorporationID int32 `json:"victim_corporation_id"`
	VictimAllianceID    int32 `json:"victim_alliance_id"`
	VictimFactionID     int32 `json:"victim_faction_id"`
	VictimShipTypeID    int32 `json:"victim_ship_type_id"`
	VictimShipGroupID   int32 `json:"victim_ship_group_id"`
	VictimDamageTaken   int32 `json:"victim_damage_taken"`

	TotalValue     float64 `json:"total_value"`
	FittedValue    float64 `json:"fitted_value"`
	DroppedValue   float64 `json:"dropped_value"`
	DestroyedValue float64 `json:"destroyed_value"`

	Points            int32 `json:"points"`
	AttackerCount     int32 `json:"attacker_count"`
	IsNPC             bool  `json:"is_npc"`
	IsSolo            bool  `json:"is_solo"`
	IsAwox            bool  `json:"is_awox"`
	IsCapitalInvolved bool  `json:"is_capital_involved"`
	IsSuperInvolved   bool  `json:"is_super_involved"`
	IsTitanInvolved   bool  `json:"is_titan_involved"`
	IsATShipInvolved  bool  `json:"is_at_ship_involved"`
	FWWinnerFactionID int32 `json:"fw_winner_faction_id"`

	WarID int32 `json:"war_id"`

	// Blob marks a mail whose item list was too large to store inline. Nothing
	// sets it yet; it exists so the row matches the table.
	Blob bool `json:"blob"`
}

// Attacker is one row of the killmail_attackers table.
type Attacker struct {
	KillmailID int64 `json:"killmail_id"`

	// AttackerIndex is the position in the ESI attackers array. It is the only
	// stable identity an attacker has — mails carry no per-attacker ID — so it
	// doubles as half the primary key and must preserve ESI's ordering.
	AttackerIndex int32 `json:"attacker_index"`

	CharacterID   int32 `json:"character_id"`
	CorporationID int32 `json:"corporation_id"`
	AllianceID    int32 `json:"alliance_id"`
	FactionID     int32 `json:"faction_id"`
	ShipTypeID    int32 `json:"ship_type_id"`
	ShipGroupID   int32 `json:"ship_group_id"`
	WeaponTypeID  int32 `json:"weapon_type_id"`
	DamageDone    int32 `json:"damage_done"`
	Points        int64 `json:"points"`
	FinalBlow     bool  `json:"final_blow"`

	// SecurityStatus is nullable and legitimately zero. See ESIAttacker.
	SecurityStatus *float64 `json:"security_status"`

	// KillmailTime is denormalised onto every attacker row so that "what did
	// this character kill in March" needs no join back to killmails.
	KillmailTime time.Time `json:"killmail_time"`
}

// Item is one row of the killmail_items table.
type Item struct {
	KillmailID int64 `json:"killmail_id"`

	// ItemIndex is assigned by the flattening walk, not by ESI. It is depth-first
	// in ESI order, which is what makes ParentIndex meaningful.
	ItemIndex int32 `json:"item_index"`

	TypeID            int32 `json:"type_id"`
	FlagID            int32 `json:"flag_id"`
	QuantityDropped   int64 `json:"quantity_dropped"`
	QuantityDestroyed int64 `json:"quantity_destroyed"`
	Singleton         int16 `json:"singleton"`

	// ParentIndex is the ItemIndex of the container or fitted module holding
	// this item, and NoParent for anything at the top level. Zero cannot mean
	// "none" here: item 0 is a real item and is frequently a parent.
	ParentIndex int32 `json:"parent_index"`
}

// NoParent marks a top-level item. It is stored as NULL.
const NoParent int32 = -1

// Parsed is the complete set of rows one killmail produces.
type Parsed struct {
	Killmail  Killmail
	Attackers []Attacker
	Items     []Item
}
