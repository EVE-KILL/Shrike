package sde

import (
	"context"
	"testing"
	"time"
)

func TestLoadHTTPKeys(t *testing.T) {
	rows := map[string][]string{
		"categories":            {"{\"categoryID\":0}"},
		"groups":                {"{\"groupID\":0,\"categoryID\":0}"},
		"types":                 {"{\"typeID\":0,\"groupID\":0}", "{\"typeID\":587,\"groupID\":25}"},
		"typeDogma":             {"{\"typeID\":587,\"dogmaAttributes\":[{\"attributeID\":6,\"value\":4}]}"},
		"dynamicItemAttributes": {"{\"typeID\":47297,\"attributeIDs\":[{\"attributeID\":6,\"min\":0.6,\"max\":1.4}]}"},
		"typeBonus":             {"{\"typeID\":587,\"types\":[{\"typeID\":3329,\"bonuses\":[{\"bonusText\":{\"en\":\"test bonus\"},\"importance\":1}]}]}"},
	}
	date := time.Date(2026, 11, 20, 16, 58, 26, 0, time.UTC)
	data, err := Load(t.Context(), func(_ context.Context, table string, visit func([]byte) error) error {
		for _, raw := range rows[table] {
			if err := visit([]byte(raw)); err != nil {
				return err
			}
		}
		return nil
	}, 3724492, date)
	if err != nil {
		t.Fatal(err)
	}
	if data.BuildNumber != 3724492 || data.ReleaseDate != date {
		t.Fatalf("incorrect build metadata: %d %s", data.BuildNumber, data.ReleaseDate)
	}
	if data.Categories[0] == nil || data.Groups[0] == nil || data.Types[0] == nil {
		t.Fatal("bundle lost valid ID-zero placeholders")
	}
	rifter := data.Types[587]
	if rifter == nil || len(rifter.DogmaAttributes) != 1 || rifter.DogmaAttributes[0].AttributeID != 6 {
		t.Fatalf("Rifter dogma data not loaded: %+v", rifter)
	}
	mutaplasmid := data.Mutaplasmids[47297]
	if mutaplasmid == nil || len(mutaplasmid.Attributes) != 1 || mutaplasmid.Attributes[0].AttributeID != 6 {
		t.Fatalf("mutaplasmid attributes not loaded: %+v", mutaplasmid)
	}
	bonuses := data.TypeBonuses[587]
	if bonuses == nil || len(bonuses.Skills) != 1 || bonuses.Skills[0].SkillTypeID != 3329 || len(bonuses.Skills[0].Bonuses) != 1 {
		t.Fatalf("type skill bonuses not loaded: %+v", bonuses)
	}
}
