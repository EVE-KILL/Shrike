package killmail

import (
	"fmt"
	"math"
	"time"
)

// KillmailColumns is the width of the killmails table, quoted in the compare
// output so "identical" states what was actually checked.
const KillmailColumns = 26

// Difference is one column that disagrees between two copies of a killmail.
type Difference struct {
	Field  string `json:"field"`
	Mine   string `json:"mine"`
	Theirs string `json:"theirs"`
}

// Diff compares two copies of a killmail column by column.
//
// tolerance is relative and applies only to the float columns — the ISK values.
// It exists for one specific case: hulls priced from custom_prices, which are
// regenerated from current market data and therefore move by a fraction of a
// percent between imports. Everything else compares exactly, because everything
// else is either an ID or a computation over IDs, and "close" would be wrong.
func Diff(mine, theirs *Parsed, tolerance float64) []Difference {
	var out []Difference
	add := func(field string, a, b any) {
		out = append(out, Difference{Field: field, Mine: fmt.Sprint(a), Theirs: fmt.Sprint(b)})
	}
	eqInt := func(field string, a, b int64) {
		if a != b {
			add(field, a, b)
		}
	}
	eqFloat := func(field string, a, b float64) {
		if !floatsEqual(a, b, tolerance) {
			out = append(out, Difference{
				Field:  field,
				Mine:   fmt.Sprintf("%.2f", a),
				Theirs: fmt.Sprintf("%.2f", b),
			})
		}
	}
	eqBool := func(field string, a, b bool) {
		if a != b {
			add(field, a, b)
		}
	}

	a, b := mine.Killmail, theirs.Killmail

	eqInt("killmail_id", a.KillmailID, b.KillmailID)
	if !a.KillmailTime.Equal(b.KillmailTime) {
		add("killmail_time", fmtTime(a.KillmailTime), fmtTime(b.KillmailTime))
	}
	if a.KillmailHash != b.KillmailHash {
		add("killmail_hash", a.KillmailHash, b.KillmailHash)
	}
	eqInt("solar_system_id", int64(a.SolarSystemID), int64(b.SolarSystemID))
	eqInt("constellation_id", int64(a.ConstellationID), int64(b.ConstellationID))
	eqInt("region_id", int64(a.RegionID), int64(b.RegionID))

	switch {
	case a.Position == nil && b.Position == nil:
	case a.Position == nil || b.Position == nil:
		add("position", fmtPos(a.Position), fmtPos(b.Position))
	default:
		eqFloat("position_x", a.Position.X, b.Position.X)
		eqFloat("position_y", a.Position.Y, b.Position.Y)
		eqFloat("position_z", a.Position.Z, b.Position.Z)
	}

	eqInt("victim_character_id", int64(a.VictimCharacterID), int64(b.VictimCharacterID))
	eqInt("victim_corporation_id", int64(a.VictimCorporationID), int64(b.VictimCorporationID))
	eqInt("victim_alliance_id", int64(a.VictimAllianceID), int64(b.VictimAllianceID))
	eqInt("victim_faction_id", int64(a.VictimFactionID), int64(b.VictimFactionID))
	eqInt("victim_ship_type_id", int64(a.VictimShipTypeID), int64(b.VictimShipTypeID))
	eqInt("victim_ship_group_id", int64(a.VictimShipGroupID), int64(b.VictimShipGroupID))
	eqInt("victim_damage_taken", int64(a.VictimDamageTaken), int64(b.VictimDamageTaken))

	eqFloat("total_value", a.TotalValue, b.TotalValue)
	eqFloat("fitted_value", a.FittedValue, b.FittedValue)
	eqFloat("dropped_value", a.DroppedValue, b.DroppedValue)
	eqFloat("destroyed_value", a.DestroyedValue, b.DestroyedValue)

	eqInt("points", int64(a.Points), int64(b.Points))
	eqInt("attacker_count", int64(a.AttackerCount), int64(b.AttackerCount))
	eqBool("is_npc", a.IsNPC, b.IsNPC)
	eqBool("is_solo", a.IsSolo, b.IsSolo)
	eqInt("war_id", int64(a.WarID), int64(b.WarID))
	eqBool("blob", a.Blob, b.Blob)

	if len(mine.Attackers) != len(theirs.Attackers) {
		add("attackers", len(mine.Attackers), len(theirs.Attackers))
	} else {
		for i := range mine.Attackers {
			x, y := mine.Attackers[i], theirs.Attackers[i]
			f := func(name string) string { return fmt.Sprintf("attacker[%d].%s", i, name) }
			eqInt(f("attacker_index"), int64(x.AttackerIndex), int64(y.AttackerIndex))
			eqInt(f("character_id"), int64(x.CharacterID), int64(y.CharacterID))
			eqInt(f("corporation_id"), int64(x.CorporationID), int64(y.CorporationID))
			eqInt(f("alliance_id"), int64(x.AllianceID), int64(y.AllianceID))
			eqInt(f("faction_id"), int64(x.FactionID), int64(y.FactionID))
			eqInt(f("ship_type_id"), int64(x.ShipTypeID), int64(y.ShipTypeID))
			eqInt(f("ship_group_id"), int64(x.ShipGroupID), int64(y.ShipGroupID))
			eqInt(f("weapon_type_id"), int64(x.WeaponTypeID), int64(y.WeaponTypeID))
			eqInt(f("damage_done"), int64(x.DamageDone), int64(y.DamageDone))
			eqBool(f("final_blow"), x.FinalBlow, y.FinalBlow)
			if !floatPtrEqual(x.SecurityStatus, y.SecurityStatus, tolerance) {
				add(f("security_status"), fmtFloatPtr(x.SecurityStatus), fmtFloatPtr(y.SecurityStatus))
			}
			if !x.KillmailTime.Equal(y.KillmailTime) {
				add(f("killmail_time"), fmtTime(x.KillmailTime), fmtTime(y.KillmailTime))
			}
		}
	}

	if len(mine.Items) != len(theirs.Items) {
		add("items", len(mine.Items), len(theirs.Items))
	} else {
		for i := range mine.Items {
			x, y := mine.Items[i], theirs.Items[i]
			f := func(name string) string { return fmt.Sprintf("item[%d].%s", i, name) }
			eqInt(f("item_index"), int64(x.ItemIndex), int64(y.ItemIndex))
			eqInt(f("type_id"), int64(x.TypeID), int64(y.TypeID))
			eqInt(f("flag_id"), int64(x.FlagID), int64(y.FlagID))
			eqInt(f("quantity_dropped"), x.QuantityDropped, y.QuantityDropped)
			eqInt(f("quantity_destroyed"), x.QuantityDestroyed, y.QuantityDestroyed)
			eqInt(f("singleton"), int64(x.Singleton), int64(y.Singleton))
			eqInt(f("parent_index"), int64(x.ParentIndex), int64(y.ParentIndex))
		}
	}

	return out
}

// floatsEqual compares relatively, so a tolerance means the same thing on a
// frigate and on a titan.
func floatsEqual(a, b, tolerance float64) bool {
	if a == b {
		return true
	}
	if tolerance <= 0 {
		return false
	}
	scale := math.Max(math.Abs(a), math.Abs(b))
	if scale == 0 {
		return true
	}
	return math.Abs(a-b)/scale <= tolerance
}

// floatPtrEqual compares two nullable floats.
//
// security_status is stored in a `real` column, so a value that went in as a
// float64 comes back rounded to float32 — -2.7 returns as -2.700000047683716.
// Comparing a parsed killmail against its stored form therefore needs the
// tolerance; comparing two stored copies does not, and passing zero still
// demands exactness.
func floatPtrEqual(a, b *float64, tolerance float64) bool {
	if a == nil || b == nil {
		return a == b
	}
	return floatsEqual(*a, *b, tolerance)
}

func fmtFloatPtr(v *float64) string {
	if v == nil {
		return "null"
	}
	return fmt.Sprintf("%g", *v)
}

func fmtPos(p *ESIPosition) string {
	if p == nil {
		return "null"
	}
	return fmt.Sprintf("(%.1f, %.1f, %.1f)", p.X, p.Y, p.Z)
}

func fmtTime(t time.Time) string {
	return t.UTC().Format(time.RFC3339)
}
