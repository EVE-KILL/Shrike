package mcpserver

import (
	"context"
	"fmt"
	"strings"
	"time"
)

type KillmailFittingInput struct {
	KillmailID int64  `json:"killmail_id" minimum:"1" jsonschema:"Numeric killmail identifier."`
	Format     string `json:"format,omitempty" enum:"json,eft" default:"json" jsonschema:"Return structured JSON or an EFT text fit."`
}

type FittingType struct {
	TypeID int64   `json:"type_id"`
	Name   *string `json:"name"`
}

type FittingItem struct {
	TypeID   int64        `json:"type_id"`
	Name     *string      `json:"name"`
	Quantity int64        `json:"quantity"`
	Charge   *FittingType `json:"charge,omitempty"`
}

type FittingSlot struct {
	Slot  string        `json:"slot"`
	Items []FittingItem `json:"items"`
}

type FittingVictim struct {
	CharacterID   *int64  `json:"character_id"`
	CharacterName *string `json:"character_name"`
}

type KillmailFittingOutput struct {
	KillmailID  int64          `json:"killmail_id"`
	URL         string         `json:"url"`
	KillTime    *time.Time     `json:"kill_time,omitempty"`
	Ship        FittingType    `json:"ship"`
	Victim      *FittingVictim `json:"victim,omitempty"`
	FitHash     string         `json:"fit_hash"`
	FamilyHash  string         `json:"family_hash"`
	FirstSeenAt *time.Time     `json:"first_seen_at,omitempty"`
	Slots       []FittingSlot  `json:"slots,omitempty"`
	Drones      []FittingItem  `json:"drones,omitempty"`
	TotalValue  *float64       `json:"total_value,omitempty"`
	EFT         string         `json:"eft,omitempty"`
}

func registerKillmailFittingTool(registry *Registry) error {
	return addTool(registry, ToolDefinition{
		Name:  "killmail_fitting",
		Title: "Get killmail fitting",
		Description: "Return a killmail fit grouped by slots, including charges, " +
			"drones, fit hash, and family hash. It can also render EFT text.",
	}, func(
		ctx context.Context,
		input KillmailFittingInput,
	) (KillmailFittingOutput, error) {
		return loadKillmailFitting(ctx, registry.deps, input)
	})
}

func loadKillmailFitting(
	ctx context.Context,
	deps Dependencies,
	input KillmailFittingInput,
) (KillmailFittingOutput, error) {
	if input.KillmailID <= 0 {
		return KillmailFittingOutput{}, fmt.Errorf("invalid killmail_id")
	}
	fittingRows, err := queryMaps(ctx, deps.DB, `
		SELECT killmail_fitting.killmail_id, killmail_fitting.fit_hash,
		       killmail_fitting.ship_type_id, killmail_fitting.kill_time,
		       fitting.family_hash, fitting.first_seen_at
		FROM killmail_fittings killmail_fitting
		JOIN fittings fitting ON fitting.fit_hash = killmail_fitting.fit_hash
		WHERE killmail_fitting.killmail_id = $1
		LIMIT 1`, input.KillmailID)
	if err != nil {
		return KillmailFittingOutput{}, fmt.Errorf("load extracted fitting: %w", err)
	}
	killmailRows, err := queryMaps(ctx, deps.DB, `
		SELECT killmail.killmail_id, killmail.killmail_time, killmail.total_value,
		       killmail.victim_character_id,
		       character.name AS victim_character_name,
		       killmail.victim_ship_type_id, ship.name AS ship_name
		FROM killmails killmail
		LEFT JOIN characters character
		       ON character.character_id = killmail.victim_character_id
		LEFT JOIN inv_types ship ON ship.type_id = killmail.victim_ship_type_id
		WHERE killmail.killmail_id = $1
		LIMIT 1`, input.KillmailID)
	if err != nil {
		return KillmailFittingOutput{}, fmt.Errorf("load fitting killmail: %w", err)
	}
	killmail := firstMap(killmailRows)
	if killmail == nil {
		return KillmailFittingOutput{}, fmt.Errorf(
			"killmail %d not found",
			input.KillmailID,
		)
	}
	fitting := firstMap(fittingRows)
	if fitting == nil {
		return KillmailFittingOutput{}, fmt.Errorf(
			"no extracted fitting for killmail %d (not yet processed, or victim had no fitted modules)",
			input.KillmailID,
		)
	}
	itemRows, err := queryMaps(ctx, deps.DB, `
		SELECT item.slot_group, item.ordinal, item.type_id,
		       item.charge_type_id, item.quantity,
		       item_type.name AS type_name, charge_type.name AS charge_name
		FROM fitting_items item
		LEFT JOIN inv_types item_type ON item_type.type_id = item.type_id
		LEFT JOIN inv_types charge_type
		       ON charge_type.type_id = item.charge_type_id
		WHERE item.fit_hash = $1
		ORDER BY item.slot_group, item.ordinal`, fitting["fit_hash"])
	if err != nil {
		return KillmailFittingOutput{}, fmt.Errorf("load fitting items: %w", err)
	}
	bySlot := map[int64][]FittingItem{}
	for _, row := range itemRows {
		item := FittingItem{
			TypeID:   valueInt64(row["type_id"]),
			Name:     nullableString(row["type_name"]),
			Quantity: valueInt64(row["quantity"]),
		}
		if item.Quantity == 0 {
			item.Quantity = 1
		}
		if row["charge_type_id"] != nil {
			item.Charge = &FittingType{
				TypeID: valueInt64(row["charge_type_id"]),
				Name:   nullableString(row["charge_name"]),
			}
		}
		group := valueInt64(row["slot_group"])
		bySlot[group] = append(bySlot[group], item)
	}
	output := KillmailFittingOutput{
		KillmailID: input.KillmailID,
		URL:        killmailURL(deps.BaseURL, input.KillmailID),
		Ship: FittingType{
			TypeID: valueInt64(fitting["ship_type_id"]),
			Name:   nullableString(killmail["ship_name"]),
		},
		FitHash:    valueString(fitting["fit_hash"]),
		FamilyHash: valueString(fitting["family_hash"]),
	}
	if input.Format == "eft" {
		output.EFT = renderEFT(
			valueString(killmail["ship_name"]),
			nullableString(killmail["victim_character_name"]),
			bySlot,
		)
		return output, nil
	}
	output.KillTime = nullableTime(killmail["killmail_time"])
	output.Victim = &FittingVictim{
		CharacterID:   nullableInt64(killmail["victim_character_id"]),
		CharacterName: nullableString(killmail["victim_character_name"]),
	}
	output.FirstSeenAt = nullableTime(fitting["first_seen_at"])
	slotNames := map[int64]string{
		1: "high", 2: "med", 3: "low", 4: "rig", 5: "subsystem",
	}
	for _, group := range []int64{1, 2, 3, 4, 5} {
		items := bySlot[group]
		if items == nil {
			items = []FittingItem{}
		}
		output.Slots = append(output.Slots, FittingSlot{
			Slot:  slotNames[group],
			Items: items,
		})
	}
	output.Drones = bySlot[6]
	if output.Drones == nil {
		output.Drones = []FittingItem{}
	}
	total := valueFloat64(killmail["total_value"])
	output.TotalValue = &total
	return output, nil
}

func renderEFT(
	shipName string,
	victimName *string,
	bySlot map[int64][]FittingItem,
) string {
	if shipName == "" {
		shipName = "Unknown ship"
	}
	fitName := "Fit from killmail"
	if victimName != nil && *victimName != "" {
		fitName = *victimName
	}
	lines := []string{"[" + shipName + ", " + fitName + "]"}
	for _, group := range []int64{3, 2, 1, 4, 5} {
		lines = append(lines, "")
		for _, item := range bySlot[group] {
			name := fmt.Sprintf("TypeID_%d", item.TypeID)
			if item.Name != nil {
				name = *item.Name
			}
			if item.Charge != nil {
				charge := fmt.Sprintf("TypeID_%d", item.Charge.TypeID)
				if item.Charge.Name != nil {
					charge = *item.Charge.Name
				}
				name += ", " + charge
			}
			lines = append(lines, name)
		}
	}
	if len(bySlot[6]) > 0 {
		lines = append(lines, "")
		for _, item := range bySlot[6] {
			name := fmt.Sprintf("TypeID_%d", item.TypeID)
			if item.Name != nil {
				name = *item.Name
			}
			lines = append(lines, fmt.Sprintf("%s x%d", name, item.Quantity))
		}
	}
	return strings.Join(lines, "\n")
}
