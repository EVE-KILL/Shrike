// Package killmail turns an ESI killmail into the rows the killboard stores.
package killmail

import "time"

// The ESI wire format.
//
// Optional numeric fields decode to zero when absent, which is the same thing:
// EVE issues no entity with ID 0, and the TypeScript parser coerces with `||`
// for exactly this reason. The two exceptions are called out where they appear.

// ESIKillmail is the response body of /killmails/{id}/{hash}/.
type ESIKillmail struct {
	KillmailID    int64         `json:"killmail_id"`
	KillmailTime  time.Time     `json:"killmail_time"`
	SolarSystemID int32         `json:"solar_system_id"`
	Victim        ESIVictim     `json:"victim"`
	Attackers     []ESIAttacker `json:"attackers"`

	// WarID is absent from the public endpoint. The war queue supplies it out
	// of band, so the parser takes it as an argument rather than reading it
	// here. EVE Ref's war archives do carry it on the document, which is why
	// the field is decoded at all.
	WarID int32 `json:"war_id"`

	// KillmailHash is likewise absent from ESI's response — there the hash is
	// half the request — but present on every document in EVE Ref's archives,
	// which is the only way an archive import can store one.
	KillmailHash string `json:"killmail_hash"`
}

// ESIVictim is the loss half of a killmail.
type ESIVictim struct {
	CharacterID   int32        `json:"character_id"`
	CorporationID int32        `json:"corporation_id"`
	AllianceID    int32        `json:"alliance_id"`
	FactionID     int32        `json:"faction_id"`
	ShipTypeID    int32        `json:"ship_type_id"`
	DamageTaken   int32        `json:"damage_taken"`
	Position      *ESIPosition `json:"position"`
	Items         []ESIItem    `json:"items"`
}

// ESIPosition is where in the system the ship died. Absent on structure kills
// and on some older mails, so it is a pointer: the three coordinates are stored
// together or not at all.
type ESIPosition struct {
	X float64 `json:"x"`
	Y float64 `json:"y"`
	Z float64 `json:"z"`
}

// ESIAttacker is one participant on the kill.
type ESIAttacker struct {
	CharacterID   int32 `json:"character_id"`
	CorporationID int32 `json:"corporation_id"`
	AllianceID    int32 `json:"alliance_id"`
	FactionID     int32 `json:"faction_id"`
	ShipTypeID    int32 `json:"ship_type_id"`
	WeaponTypeID  int32 `json:"weapon_type_id"`
	DamageDone    int32 `json:"damage_done"`
	FinalBlow     bool  `json:"final_blow"`

	// SecurityStatus is a pointer because 0 is a real security status — a
	// freshly created character has exactly 0.0 — and only NPC attackers omit
	// the field entirely.
	SecurityStatus *float64 `json:"security_status"`
}

// ESIItem is a fitted module, a cargo item, or the contents of either. Items
// nest one level: a container or a fitted launcher carries its own items array.
type ESIItem struct {
	ItemTypeID        int32     `json:"item_type_id"`
	Flag              int32     `json:"flag"`
	Singleton         int16     `json:"singleton"`
	QuantityDropped   int64     `json:"quantity_dropped"`
	QuantityDestroyed int64     `json:"quantity_destroyed"`
	Items             []ESIItem `json:"items"`
}

// SingletonBPC marks a blueprint copy. Copies carry no market value regardless
// of what the original sells for, so they are priced at the floor rather than
// looked up — otherwise a hauler full of copied battleship blueprints would
// read as a fortune.
const SingletonBPC int16 = 2
