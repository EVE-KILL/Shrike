package api

import "testing"

func TestParseCampaignSidesAcceptsParticipantReplacement(t *testing.T) {
	sides, err := parseCampaignSides([]any{
		map[string]any{
			"name": "Attackers",
			"entities": []any{
				map[string]any{"type": "alliance", "id": float64(99000001)},
			},
		},
		map[string]any{
			"name": "Defenders",
			"entities": []any{
				map[string]any{"type": "corporation", "id": float64(98000001)},
			},
		},
	}, false)
	if err != nil {
		t.Fatal(err)
	}
	if len(sides) != 2 || sides[0].Name != "Attackers" ||
		len(sides[1].Entities) != 1 || sides[1].Entities[0].ID != 98000001 {
		t.Fatalf("parsed sides = %#v", sides)
	}
}

func TestParseCampaignSidesRejectsParticipantOnMultipleSides(t *testing.T) {
	_, err := parseCampaignSides([]any{
		map[string]any{"entities": []any{
			map[string]any{"type": "corporation", "id": float64(98000001)},
		}},
		map[string]any{"entities": []any{
			map[string]any{"type": "corporation", "id": float64(98000001)},
		}},
	}, false)
	if err == nil {
		t.Fatal("expected duplicate participant error")
	}
}
