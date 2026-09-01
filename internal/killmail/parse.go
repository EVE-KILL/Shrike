package killmail

import (
	"context"

	"github.com/eve-kill/shrike/internal/eve"
)

// Slot ranges. EVE numbers inventory locations with a flat "flag" enum, and
// these are the stretches of it that mean "fitted to the hull":
//
//	 11– 34  low, mid and high slots
//	 87      drone bay
//	 92– 95  rig slots
//	125–132  subsystems and service slots
//	158–162  fighter tubes
//
// isModuleSlot is the narrower question the points algorithm asks — an actual
// module in an actual slot — and deliberately excludes the drone bay and
// fighter tubes, which hold ammunition rather than modules.
func isModuleSlot(flag int32) bool {
	return (flag >= 11 && flag <= 34) || (flag >= 92 && flag <= 95) || (flag >= 125 && flag <= 132)
}

// isFittedSlot is the wider question the valuation asks: does the loss of this
// item represent the loss of a fit, as opposed to cargo that happened to be
// aboard.
func isFittedSlot(flag int32) bool {
	return isModuleSlot(flag) || flag == 87 || (flag >= 158 && flag <= 162)
}

// categoryEntity is the SDE category NPCs belong to; categoryShip is hulls.
const (
	categoryShip      int32 = 6
	categoryModule    int32 = 7
	categoryEntity    int32 = 11
	categoryStructure int32 = 65
)

// Parse turns an ESI killmail into the rows the killboard stores.
//
// It performs exactly one database query — the batched price lookup — and does
// everything else from the in-memory cache.
func Parse(
	ctx context.Context,
	cache *eve.Cache,
	prices *eve.Prices,
	km *ESIKillmail,
	hash string,
	warID int32,
) (*Parsed, error) {
	// The kill date, not the timestamp: prices are daily, and a mail at 00:30
	// is valued with the same numbers as one at 23:30.
	killDate := km.KillmailTime.UTC().Format("2006-01-02")

	items := flattenItems(
		km.Victim.Items,
		km.KillmailID,
		make([]Item, 0, countItems(km.Victim.Items)),
		NoParent,
	)

	// Every type whose price has to be asked of the market, gathered before a
	// single query goes out. Blueprint copies are left out: they are priced by
	// rule, not by lookup.
	typeIDs := make([]int32, 0, len(items)+1)
	if km.Victim.ShipTypeID != 0 {
		typeIDs = append(typeIDs, km.Victim.ShipTypeID)
	}
	for _, it := range items {
		if it.Singleton != SingletonBPC {
			typeIDs = append(typeIDs, it.TypeID)
		}
	}

	day, err := prices.On(ctx, killDate, typeIDs)
	if err != nil {
		return nil, err
	}

	var shipPrice float64
	if km.Victim.ShipTypeID != 0 {
		shipPrice = day.Of(km.Victim.ShipTypeID)
	}

	var droppedValue, destroyedValue, fittedValue float64
	for _, it := range items {
		price := day.Of(it.TypeID)
		if it.Singleton == SingletonBPC {
			price = eve.FallbackPrice
		}
		dropped := float64(it.QuantityDropped) * price
		destroyed := float64(it.QuantityDestroyed) * price

		droppedValue += dropped
		destroyedValue += destroyed
		if isFittedSlot(it.FlagID) {
			fittedValue += dropped + destroyed
		}
	}

	// The hull counts as fitted value but is not an item, so it is added after
	// the loop rather than inside it.
	fittedValue += shipPrice
	totalValue := shipPrice + droppedValue + destroyedValue

	system, _ := cache.System(km.SolarSystemID)
	victimType, _ := cache.Type(km.Victim.ShipTypeID)

	row := Killmail{
		KillmailID:   km.KillmailID,
		KillmailTime: km.KillmailTime,
		KillmailHash: hash,

		SolarSystemID:   km.SolarSystemID,
		ConstellationID: system.ConstellationID,
		RegionID:        system.RegionID,
		Position:        km.Victim.Position,

		VictimCharacterID:   km.Victim.CharacterID,
		VictimCorporationID: km.Victim.CorporationID,
		VictimAllianceID:    km.Victim.AllianceID,
		VictimFactionID:     km.Victim.FactionID,
		VictimShipTypeID:    km.Victim.ShipTypeID,
		VictimShipGroupID:   victimType.GroupID,
		VictimDamageTaken:   km.Victim.DamageTaken,

		TotalValue:     totalValue,
		FittedValue:    fittedValue,
		DroppedValue:   droppedValue,
		DestroyedValue: destroyedValue,

		Points:        calculatePoints(cache, km),
		AttackerCount: int32(len(km.Attackers)),
		IsNPC:         detectNPC(cache, km.Attackers),
		IsSolo:        detectSolo(km.Attackers),

		WarID: warID,
		Blob:  false,
	}

	attackers := make([]Attacker, 0, len(km.Attackers))
	for i, att := range km.Attackers {
		shipTypeID := resolveAttackerShip(cache, att)
		shipType, _ := cache.Type(shipTypeID)

		attackers = append(attackers, Attacker{
			KillmailID:     km.KillmailID,
			AttackerIndex:  int32(i),
			CharacterID:    att.CharacterID,
			CorporationID:  att.CorporationID,
			AllianceID:     att.AllianceID,
			FactionID:      att.FactionID,
			ShipTypeID:     shipTypeID,
			ShipGroupID:    shipType.GroupID,
			WeaponTypeID:   att.WeaponTypeID,
			DamageDone:     att.DamageDone,
			FinalBlow:      att.FinalBlow,
			SecurityStatus: att.SecurityStatus,
			KillmailTime:   km.KillmailTime,
		})
	}
	participants := make([]PointParticipant, 0, len(attackers))
	for _, attacker := range attackers {
		participants = append(participants, PointParticipant{
			CharacterID: attacker.CharacterID,
			DamageDone:  int64(attacker.DamageDone),
			FinalBlow:   attacker.FinalBlow,
		})
	}
	shares := AllocatePoints(int64(row.Points), DefaultParticipationBasisPoints, participants)
	seenCharacters := make(map[int32]bool)
	for i := range attackers {
		characterID := attackers[i].CharacterID
		if characterID != 0 && !seenCharacters[characterID] {
			attackers[i].Points = shares[characterID]
			seenCharacters[characterID] = true
		}
	}

	parsed := &Parsed{Killmail: row, Attackers: attackers, Items: items}
	DeriveStableFacts(parsed)
	return parsed, nil
}

func countItems(items []ESIItem) int {
	count := len(items)
	for _, item := range items {
		count += countItems(item.Items)
	}
	return count
}

// resolveAttackerShip fills in a missing hull from the weapon.
//
// Some attackers arrive with no ship_type_id but a weapon_type_id that is
// itself a hull. That is standard for NPCs and for cases where the ship is the
// weapon — ramming, bombs, structures firing. Recording no ship there would
// lose the only information the mail has about what did the killing.
func resolveAttackerShip(cache *eve.Cache, att ESIAttacker) int32 {
	if att.ShipTypeID != 0 {
		return att.ShipTypeID
	}
	if att.WeaponTypeID == 0 {
		return 0
	}
	weapon, ok := cache.Type(att.WeaponTypeID)
	if !ok || weapon.GroupID == 0 {
		return 0
	}
	// Resolved through the group rather than the type's own category_id, which
	// is what the TypeScript does; the two agree because the SDE importer
	// derives inv_types.category_id from the group in the first place.
	if group, ok := cache.Group(weapon.GroupID); ok && group.CategoryID == categoryShip {
		return att.WeaponTypeID
	}
	return 0
}

// flattenItems walks the two-level ESI item tree into the flat rows the table
// holds, depth-first so that a container is always assigned a lower index than
// its contents and ParentIndex can refer backwards.
func flattenItems(src []ESIItem, killmailID int64, out []Item, parent int32) []Item {
	for _, it := range src {
		index := int32(len(out))
		out = append(out, Item{
			KillmailID:        killmailID,
			ItemIndex:         index,
			TypeID:            it.ItemTypeID,
			FlagID:            it.Flag,
			QuantityDropped:   it.QuantityDropped,
			QuantityDestroyed: it.QuantityDestroyed,
			Singleton:         it.Singleton,
			ParentIndex:       parent,
		})
		if len(it.Items) > 0 {
			out = flattenItems(it.Items, killmailID, out, index)
		}
	}
	return out
}

// detectNPC reports whether nothing but NPCs was involved.
//
// A single player attacker disqualifies the mail outright. Beyond that, every
// attacker with a hull must be flying an Entity — the SDE category NPCs belong
// to — because an empty attacker with a player hull is a disconnected capsuleer,
// not rats.
func detectNPC(cache *eve.Cache, attackers []ESIAttacker) bool {
	if len(attackers) == 0 {
		return false
	}
	for _, att := range attackers {
		if att.CharacterID != 0 {
			return false
		}
		if att.ShipTypeID == 0 {
			continue
		}
		t, ok := cache.Type(att.ShipTypeID)
		if !ok || t.GroupID == 0 {
			continue
		}
		group, ok := cache.Group(t.GroupID)
		if !ok || group.CategoryID != categoryEntity {
			return false
		}
	}
	return true
}

// detectSolo reports whether exactly one capsuleer is responsible.
//
// With no players at all it falls back to the attacker count, so a lone rat
// counts as a solo kill while a rat gang does not.
func detectSolo(attackers []ESIAttacker) bool {
	players := 0
	for _, att := range attackers {
		if att.CharacterID != 0 {
			players++
		}
	}
	switch {
	case players == 1:
		return true
	case players > 1:
		return false
	default:
		return len(attackers) == 1
	}
}
