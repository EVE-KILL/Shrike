package killmail

import (
	"math"

	"github.com/eve-kill/shrike/internal/eve"
)

// Points, the zKillboard scoring algorithm.
//
// The intent is to reward difficulty rather than ISK: a well-fitted frigate
// taken by one pilot scores better than a freighter blapped by fifty. It gets
// there by starting from hull size, adjusting for how dangerous the fit was,
// dividing by how many people showed up, and finally comparing the victim's
// size against the average attacker's.
//
// Ported verbatim, quirks included, because the number is stored per kill and
// any change to it would silently rescore eighteen years of history. Where a
// step looks wrong, the comment says so rather than the code fixing it.

const (
	groupStrategicCruiser int32 = 963 // T3 cruisers carry no rigSize attribute
	groupSmartbomb        int32 = 645 // area weapons count as extra danger
	groupMiningLaser      int32 = 54  // a mining fit is the opposite of danger
)

// rigSize is the hull-size exponent: 1 frigate, 2 cruiser, 3 battleship,
// 4 capital. Strategic cruisers have no such attribute and are pinned.
func rigSize(cache *eve.Cache, typeID int32) float64 {
	if t, ok := cache.Type(typeID); ok && t.GroupID == groupStrategicCruiser {
		return 2
	}
	if v, ok := cache.Dogma(typeID, eve.AttrRigSize); ok {
		return v
	}
	return 1
}

// PointsItem is one fitted item, as the scorer reads it.
type PointsItem struct {
	TypeID            int32
	Flag              int32
	QuantityDropped   int64
	QuantityDestroyed int64
}

// PointsAttacker is one attacker, as the scorer reads it.
type PointsAttacker struct {
	CharacterID int32
	ShipTypeID  int32
}

type pointsItem interface {
	pointsItemFields() (typeID, flag int32, dropped, destroyed int64)
}

func (item PointsItem) pointsItemFields() (int32, int32, int64, int64) {
	return item.TypeID, item.Flag, item.QuantityDropped, item.QuantityDestroyed
}

func (item ESIItem) pointsItemFields() (int32, int32, int64, int64) {
	return item.ItemTypeID, item.Flag, item.QuantityDropped, item.QuantityDestroyed
}

type pointsAttacker interface {
	pointsAttackerFields() (characterID, shipTypeID int32)
}

func (attacker PointsAttacker) pointsAttackerFields() (int32, int32) {
	return attacker.CharacterID, attacker.ShipTypeID
}

func (attacker ESIAttacker) pointsAttackerFields() (int32, int32) {
	return attacker.CharacterID, attacker.ShipTypeID
}

// PointsInput is everything the score depends on.
//
// Neutral between the two sources that produce it: a freshly fetched ESI
// document during ingest, and the stored rows during a backfill. They must
// score identically — a backfill that disagreed with ingest would rewrite
// history to a different number than the live path would have produced — so
// there is one implementation and two thin adapters rather than two copies.
type PointsInput struct {
	VictimShipTypeID int32

	// Items are the victim's top-level items only, never the flattened tree.
	// Modules inside containers are cargo, not a fit, and counting them would
	// let a hauler full of spare guns score as a brawler.
	Items     []PointsItem
	Attackers []PointsAttacker
}

func calculatePoints(cache *eve.Cache, km *ESIKillmail) int32 {
	return scorePoints(cache, km.Victim.ShipTypeID, km.Victim.Items, km.Attackers)
}

// Points is the zKillboard score for one killmail.
func Points(cache *eve.Cache, km PointsInput) int32 {
	return scorePoints(cache, km.VictimShipTypeID, km.Items, km.Attackers)
}

func scorePoints[I pointsItem, A pointsAttacker](
	cache *eve.Cache,
	victimShipTypeID int32,
	items []I,
	attackers []A,
) int32 {
	if victimShipTypeID == 0 {
		return 1
	}

	victimRigSize := rigSize(cache, victimShipTypeID)
	basePoints := math.Pow(5, victimRigSize)

	dangerFactor := 0.0
	for _, item := range items {
		typeID, flag, dropped, destroyed := item.pointsItemFields()
		t, ok := cache.Type(typeID)
		if !ok || t.CategoryID != categoryModule {
			continue
		}
		if !isModuleSlot(flag) {
			continue
		}

		qty := float64(destroyed + dropped)

		metaLevel, _ := cache.Dogma(typeID, eve.AttrMetaLevel)
		meta := 1 + math.Floor(metaLevel/2)

		// A module that generates heat damage is one that can be overheated,
		// which is the cheapest available proxy for "this was a combat module".
		if heat, ok := cache.Dogma(typeID, eve.AttrHeatDamage); ok && heat > 0 {
			dangerFactor += qty * meta
		}
		if t.GroupID == groupSmartbomb {
			dangerFactor += qty * meta
		}
		if t.GroupID == groupMiningLaser {
			dangerFactor -= qty * meta
		}
	}

	points := basePoints + dangerFactor
	// An unfitted hull scores at one percent. Four dangerous modules is enough
	// to score at full rate; the scale between is linear.
	points *= math.Max(0.01, math.Min(1, dangerFactor/4))

	playerAttackers := 0
	for _, att := range attackers {
		characterID, _ := att.pointsAttackerFields()
		if characterID != 0 {
			playerAttackers++
		}
	}
	numAttackers := math.Max(1, float64(playerAttackers))

	// Roughly quadratic in gang size, so a blob is worth very little per head.
	involvedPenalty := math.Max(1, numAttackers*math.Max(1, numAttackers/2))
	points /= involvedPenalty

	hasPlayer := false
	totalSize := 0.0
	for _, att := range attackers {
		characterID, shipTypeID := att.pointsAttackerFields()
		if characterID != 0 {
			hasPlayer = true
		}
		// The raw ship_type_id, not the weapon-inferred hull used for the
		// attacker rows: an attacker with no ship contributes no size.
		if shipTypeID == 0 {
			continue
		}
		attType, _ := cache.Type(shipTypeID)
		// A structure on the mail voids scoring entirely. Structures have
		// absurd effective hull sizes and would dominate the average.
		if attType.CategoryID == categoryStructure {
			return 1
		}
		if attType.GroupID != groupCapsule {
			totalSize += math.Pow(5, rigSize(cache, shipTypeID))
		} else {
			// A pod among the attackers is scored as one size class above the
			// victim, so podding does not deflate the average.
			totalSize += math.Pow(5, victimRigSize+1)
		}
	}

	if !hasPlayer {
		return 1
	}

	// Note the divisor: the player count, against a total that summed every
	// attacker including NPCs. On a mail with rats among the players this
	// inflates the average attacker size and suppresses the score. It is
	// zKillboard's behaviour and is kept deliberately.
	avgSize := math.Max(1, totalSize/numAttackers)
	modifier := math.Min(1.2, math.Max(0.5, basePoints/avgSize))
	points = math.Floor(points * modifier)

	if points < 1 {
		return 1
	}
	if points > math.MaxInt32 {
		return math.MaxInt32
	}
	return int32(points)
}

// groupCapsule is duplicated from the eve package's price rules because the two
// uses are unrelated: there it means "priced by convention", here it means "not
// a real ship for sizing purposes".
const groupCapsule int32 = 29
