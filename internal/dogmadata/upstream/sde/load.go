package sde

import (
	"context"
	"encoding/json"
	"time"
)

// Tables are the datasets consumed by Load.
var Tables = []string{
	"categories", "groups", "marketGroups", "metaGroups", "types",
	"dogmaAttributes", "dogmaUnits", "dogmaAttributeCategories", "dogmaEffects",
	"dbuffCollections", "typeDogma", "fighterAbilities", "fighterAbilitiesByType",
	"dynamicItemAttributes", "typeBonus",
}

type scanTable func(context.Context, string, func([]byte) error) error

// Load reads the selected JSONL tables in dependency order.
func Load(ctx context.Context, scan scanTable, build int32, releaseDate time.Time) (*Data, error) {
	data := &Data{
		BuildNumber:      build,
		ReleaseDate:      releaseDate,
		Types:            map[int32]*Type{},
		Groups:           map[int32]*Group{},
		Categories:       map[int32]*Category{},
		MarketGroups:     map[int32]*MarketGroup{},
		MetaGroups:       map[int32]*MetaGroup{},
		FighterAbilities: map[int32]*FighterAbility{},
		DogmaAttributes:  map[int32]*DogmaAttribute{},
		DogmaUnits:       map[int32]*DogmaUnit{},
		DogmaCategories:  map[int32]*DogmaAttributeCategory{},
		DogmaEffects:     map[int32]*DogmaEffect{},
		DbuffCollections: map[int32]*DbuffCollection{},
		Mutaplasmids:     map[int32]*Mutaplasmid{},
		TypeBonuses:      map[int32]*TypeBonuses{},
	}

	if err := decode(ctx, scan, "categories", func(entry *Category) {
		data.Categories[entry.Key] = entry
	}); err != nil {
		return nil, err
	}
	if err := decode(ctx, scan, "groups", func(entry *Group) {
		data.Groups[entry.Key] = entry
	}); err != nil {
		return nil, err
	}
	if err := decode(ctx, scan, "marketGroups", func(entry *MarketGroup) {
		data.MarketGroups[entry.Key] = entry
	}); err != nil {
		return nil, err
	}
	if err := decode(ctx, scan, "metaGroups", func(entry *MetaGroup) {
		data.MetaGroups[entry.Key] = entry
	}); err != nil {
		return nil, err
	}
	if err := decode(ctx, scan, "types", func(entry *Type) {
		data.Types[entry.Key] = entry
	}); err != nil {
		return nil, err
	}
	if err := decode(ctx, scan, "dogmaAttributes", func(entry *DogmaAttribute) {
		data.DogmaAttributes[entry.Key] = entry
	}); err != nil {
		return nil, err
	}
	if err := decode(ctx, scan, "dogmaUnits", func(entry *DogmaUnit) {
		data.DogmaUnits[entry.Key] = entry
	}); err != nil {
		return nil, err
	}
	if err := decode(ctx, scan, "dogmaAttributeCategories", func(entry *DogmaAttributeCategory) {
		data.DogmaCategories[entry.Key] = entry
	}); err != nil {
		return nil, err
	}
	if err := decode(ctx, scan, "dogmaEffects", func(entry *DogmaEffect) {
		data.DogmaEffects[entry.Key] = entry
	}); err != nil {
		return nil, err
	}

	if err := decode(ctx, scan, "dbuffCollections", func(entry *DbuffCollection) {
		data.DbuffCollections[entry.Key] = entry
	}); err != nil {
		return nil, err
	}

	if err := decode(ctx, scan, "typeDogma", func(entry *typeDogmaEntry) {
		item, ok := data.Types[entry.Key]
		if !ok {
			return
		}
		item.DogmaAttributes = entry.DogmaAttributes
		item.DogmaEffects = entry.DogmaEffects
	}); err != nil {
		return nil, err
	}

	if err := decode(ctx, scan, "fighterAbilities", func(entry *FighterAbility) {
		data.FighterAbilities[entry.Key] = entry
	}); err != nil {
		return nil, err
	}

	if err := decode(ctx, scan, "fighterAbilitiesByType", func(entry *fighterAbilitiesEntry) {
		item, ok := data.Types[entry.Key]
		if !ok {
			return
		}
		for slot, ability := range []*fighterAbilitySlot{entry.Slot0, entry.Slot1, entry.Slot2} {
			if ability == nil {
				continue
			}
			item.FighterAbilities = append(item.FighterAbilities, TypeFighterAbility{
				Slot:             int8(slot),
				AbilityID:        ability.AbilityID,
				CooldownSeconds:  ability.CooldownSeconds,
				ChargeCount:      ability.Charges.ChargeCount,
				RearmTimeSeconds: ability.Charges.RearmTimeSeconds,
			})
		}
	}); err != nil {
		return nil, err
	}

	if err := decode(ctx, scan, "dynamicItemAttributes", func(entry *Mutaplasmid) {
		data.Mutaplasmids[entry.Key] = entry
	}); err != nil {
		return nil, err
	}

	if err := decode(ctx, scan, "typeBonus", func(entry *TypeBonuses) {
		data.TypeBonuses[entry.Key] = entry
	}); err != nil {
		return nil, err
	}

	for _, item := range data.Types {
		if group, ok := data.Groups[item.GroupID]; ok {
			item.CategoryID = group.CategoryID
		}
	}

	return data, nil
}

type typeDogmaEntry struct {
	Key             int32                `json:"typeID"`
	DogmaAttributes []TypeDogmaAttribute `json:"dogmaAttributes"`
	DogmaEffects    []TypeDogmaEffect    `json:"dogmaEffects"`
}

// The SDE gives each ability slot its own key rather than a list.
type fighterAbilitiesEntry struct {
	Key   int32               `json:"typeID"`
	Slot0 *fighterAbilitySlot `json:"abilitySlot0"`
	Slot1 *fighterAbilitySlot `json:"abilitySlot1"`
	Slot2 *fighterAbilitySlot `json:"abilitySlot2"`
}

type fighterAbilitySlot struct {
	AbilityID       int32   `json:"abilityID"`
	CooldownSeconds float64 `json:"cooldownSeconds"`
	Charges         struct {
		ChargeCount      int32   `json:"chargeCount"`
		RearmTimeSeconds float64 `json:"rearmTimeSeconds"`
	} `json:"charges"`
}

func decode[T any](ctx context.Context, scan scanTable, name string, add func(*T)) error {
	return scan(ctx, name, func(raw []byte) error {
		var entry T
		if err := json.Unmarshal(raw, &entry); err != nil {
			return err
		}
		add(&entry)
		return nil
	})
}
