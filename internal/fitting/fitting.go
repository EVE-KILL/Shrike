// Package fitting turns a killmail's item list into a stable fit identity.
//
// Three things come out of one killmail:
//
//   - fit_hash     sha256 over the ship and its fitted modules
//   - family_hash  the same, with each module replaced by its T1 root, so
//     meta and faction variants of one doctrine cluster together
//   - items        the render payload — modules with their loaded charge, plus
//     drone rows carrying a bay total
//
// The hash format is a compatibility contract, not an implementation detail.
// Roughly 400,000 fits are stored under hashes produced by the TypeScript, and
// a hash that differs by so much as a separator orphans all of them. So the
// serialisation below is reproduced exactly — including the decisions that look
// arbitrary.
//
// The one worth calling out: charges and drones are deliberately NOT part of
// the hash. Two Rifters with identical modules and different ammo are the same
// fit. That was chosen so the enrichment columns could be added later without
// invalidating what was already on disk, and it stays that way.
package fitting

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"sort"
	"strings"

	"github.com/eve-kill/shrike/internal/eve"
)

// Slot groups, numbered to match the in-game and EFT layout — high, med, low,
// rig, subsystem, drone — so the frontend renders a fit without remapping.
const (
	SlotHigh      = 1
	SlotMed       = 2
	SlotLow       = 3
	SlotRig       = 4
	SlotSubsystem = 5
	SlotDrone     = 6
)

// DroneBayFlag is where drones sit on a killmail.
const DroneBayFlag = 87

// SDE categories. Modules and subsystems make up a fit; charges and drones are
// enrichment hanging off it.
const (
	categoryShip      = 6
	categoryModule    = 7
	categoryCharge    = 8
	categoryDrone     = 18
	categorySubsystem = 32
)

// Item is one killmail item, reduced to what the extractor needs.
type Item struct {
	TypeID int32
	FlagID int32

	// ParentIndex is >= 0 for an item inside a container. Only top-level items
	// are fitted hardware; anything nested is cargo.
	ParentIndex int32

	// Quantity is dropped plus destroyed. Only drones use it — a bay holds
	// several of one type and the count has to round-trip.
	Quantity int32
}

// TopLevel reports whether the item is fitted rather than in a container.
func (i Item) TopLevel() bool { return i.ParentIndex < 0 }

// ExtractedItem is one row of the render payload.
type ExtractedItem struct {
	SlotGroup    int32
	Ordinal      int32
	TypeID       int32
	ChargeTypeID int32
	Quantity     int32
}

// Fitting is the extracted identity and payload.
type Fitting struct {
	FitHash    string
	FamilyHash string
	Items      []ExtractedItem
}

// SlotGroupForFlag maps an SDE inventory flag to a slot group.
//
// The ranges are verified against inv_flags: 11-18 low, 19-26 med, 27-34 high,
// 92-99 rig (eight exist even though most hulls use three), 125-132 subsystem.
//
// Fighter tubes and bays are deliberately absent. They are not covered by the
// partial index on killmail_items, so including them would turn every
// carrier and supercarrier fit lookup into a full scan — rare enough to wait
// for a dedicated index.
func SlotGroupForFlag(flag int32) int32 {
	switch {
	case flag >= 27 && flag <= 34:
		return SlotHigh
	case flag >= 19 && flag <= 26:
		return SlotMed
	case flag >= 11 && flag <= 18:
		return SlotLow
	case flag >= 92 && flag <= 99:
		return SlotRig
	case flag >= 125 && flag <= 132:
		return SlotSubsystem
	case flag == DroneBayFlag:
		return SlotDrone
	}
	return 0
}

// IsShipType reports whether a hull is a ship.
//
// Resolved through group to category because CCP's SDE does not populate a
// category on types, only on groups. Callers use it to skip pods, structures
// and deployables, where a fit means nothing.
func IsShipType(cache *eve.Cache, typeID int32) bool {
	return categoryOf(cache, typeID) == categoryShip
}

func categoryOf(cache *eve.Cache, typeID int32) int32 {
	if cache == nil || typeID == 0 {
		return 0
	}
	t, ok := cache.Type(typeID)
	if !ok || t.GroupID == 0 {
		return 0
	}
	g, ok := cache.Group(t.GroupID)
	if !ok {
		return 0
	}
	return g.CategoryID
}

func isModuleCategory(cache *eve.Cache, typeID int32) bool {
	c := categoryOf(cache, typeID)
	return c == categoryModule || c == categorySubsystem
}

// module is one fitted item with whatever was loaded in it.
type module struct {
	typeID       int32
	chargeTypeID int32
}

// Extract builds the fit identity for a hull and its items.
//
// Returns nil when there is no fit: an empty hull, or one carrying only drones.
// A fit identity needs at least one fitted module to mean anything, so a pod
// with a drone bay does not get one.
func Extract(cache *eve.Cache, shipTypeID int32, items []Item) *Fitting {
	// Charges share their module's flag, so the loaded ammo is found by flag
	// rather than by nesting. Last one wins if two somehow share a flag, which
	// a real killmail does not produce.
	chargeByFlag := map[int32]int32{}
	for _, it := range items {
		if !it.TopLevel() || categoryOf(cache, it.TypeID) != categoryCharge {
			continue
		}
		chargeByFlag[it.FlagID] = it.TypeID
	}

	buckets := map[int32][]module{}
	for _, it := range items {
		if !it.TopLevel() {
			continue
		}
		slot := SlotGroupForFlag(it.FlagID)
		if slot == 0 || slot == SlotDrone {
			continue
		}
		if !isModuleCategory(cache, it.TypeID) {
			continue
		}
		buckets[slot] = append(buckets[slot], module{
			typeID:       it.TypeID,
			chargeTypeID: chargeByFlag[it.FlagID],
		})
	}

	// Drones are summed per type across rows, in case CCP ever emits more than
	// one row for the same drone at the same flag.
	droneByType := map[int32]int32{}
	for _, it := range items {
		if !it.TopLevel() || it.FlagID != DroneBayFlag {
			continue
		}
		if categoryOf(cache, it.TypeID) != categoryDrone {
			continue
		}
		qty := it.Quantity
		if qty <= 0 {
			qty = 1
		}
		droneByType[it.TypeID] += qty
	}

	if len(buckets) == 0 {
		return nil
	}

	slots := make([]int32, 0, len(buckets))
	for s := range buckets {
		slots = append(slots, s)
	}
	sort.Slice(slots, func(i, j int) bool { return slots[i] < slots[j] })

	exact := []string{fmt.Sprint(shipTypeID)}
	family := []string{fmt.Sprint(shipTypeID)}
	var out []ExtractedItem

	for _, slot := range slots {
		bucket := buckets[slot]

		// Sorted by type then charge. The charge is in the sort key but not in
		// the hash: two identical modules with different ammo must land in the
		// same ordinal positions on every re-extract, or the stored item rows
		// churn for a fit whose identity has not changed.
		sort.Slice(bucket, func(i, j int) bool {
			if bucket[i].typeID != bucket[j].typeID {
				return bucket[i].typeID < bucket[j].typeID
			}
			return bucket[i].chargeTypeID < bucket[j].chargeTypeID
		})

		typeIDs := make([]string, len(bucket))
		familyIDs := make([]string, len(bucket))
		for i, m := range bucket {
			typeIDs[i] = fmt.Sprint(m.typeID)
			familyIDs[i] = fmt.Sprint(variationRoot(cache, m.typeID))

			out = append(out, ExtractedItem{
				SlotGroup:    slot,
				Ordinal:      int32(i),
				TypeID:       m.typeID,
				ChargeTypeID: m.chargeTypeID,
				Quantity:     1,
			})
		}

		exact = append(exact, fmt.Sprintf("%d:%s", slot, strings.Join(typeIDs, ",")))
		family = append(family, fmt.Sprintf("%d:%s", slot, strings.Join(familyIDs, ",")))
	}

	// Drones come last, in their own slot group, sorted for a stable ordinal.
	droneTypes := make([]int32, 0, len(droneByType))
	for t := range droneByType {
		droneTypes = append(droneTypes, t)
	}
	sort.Slice(droneTypes, func(i, j int) bool { return droneTypes[i] < droneTypes[j] })

	for i, typeID := range droneTypes {
		out = append(out, ExtractedItem{
			SlotGroup: SlotDrone,
			Ordinal:   int32(i),
			TypeID:    typeID,
			Quantity:  droneByType[typeID],
		})
	}

	return &Fitting{
		FitHash:    hash(exact),
		FamilyHash: hash(family),
		Items:      out,
	}
}

// variationRoot resolves a module to the T1 hull of its meta family, so
// variants cluster. Types with no parent are their own root.
func variationRoot(cache *eve.Cache, typeID int32) int32 {
	if t, ok := cache.Type(typeID); ok && t.VariationParentTypeID != 0 {
		return t.VariationParentTypeID
	}
	return typeID
}

// hash joins the segments with a pipe and digests them.
//
// The separator and the digest are part of the stored contract — see the
// package comment.
func hash(segments []string) string {
	sum := sha256.Sum256([]byte(strings.Join(segments, "|")))
	return hex.EncodeToString(sum[:])
}
