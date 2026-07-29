package mcpserver

import (
	"context"
	"fmt"
	"math"
	"regexp"
	"strings"
)

type DogmaModuleInput struct {
	TypeID       int64  `json:"type_id"`
	Slot         string `json:"slot" enum:"high,med,low,rig,subsystem"`
	Index        *int   `json:"index,omitempty" minimum:"0"`
	ChargeTypeID *int64 `json:"charge_type_id,omitempty"`
}

type DogmaDroneInput struct {
	TypeID   int64 `json:"type_id"`
	Quantity int   `json:"quantity,omitempty" default:"1" minimum:"1"`
}

type DogmaFitInput struct {
	KillmailID *int64             `json:"killmail_id,omitempty"`
	FitHash    *string            `json:"fit_hash,omitempty"`
	EFT        *string            `json:"eft,omitempty"`
	ShipTypeID *int64             `json:"ship_type_id,omitempty"`
	Modules    []DogmaModuleInput `json:"modules,omitempty"`
	Drones     []DogmaDroneInput  `json:"drones,omitempty"`
}

type DogmaEvalInput struct {
	DogmaFitInput
	Skills string `json:"skills,omitempty" enum:"all_v,none" default:"all_v"`
}

type DogmaShip struct {
	TypeID int64   `json:"type_id"`
	Name   *string `json:"name"`
}

type HullDisplay struct {
	DPSWithReload    *float64 `json:"dps_with_reload"`
	DPSWithoutReload *float64 `json:"dps_without_reload"`
	Alpha            *float64 `json:"alpha"`
	EHP              *float64 `json:"ehp"`
	EHPShield        *float64 `json:"ehp_shield"`
	EHPArmor         *float64 `json:"ehp_armor"`
	EHPHull          *float64 `json:"ehp_hull"`
	CapStable        bool     `json:"cap_stable"`
	CapTimeSeconds   *float64 `json:"cap_time_s"`
	CapCapacityGJ    *float64 `json:"cap_capacity_gj"`
	CapPeakDeltaGJS  *float64 `json:"cap_peak_delta_gj_s"`
	AlignTimeSeconds *float64 `json:"align_time_s"`
	MaxVelocityMS    *float64 `json:"max_velocity_ms"`
	SignatureRadiusM *float64 `json:"signature_radius_m"`
	MaxTargetRangeKM *float64 `json:"max_target_range_km"`
	ScanResolutionMM *float64 `json:"scan_resolution_mm"`
	MaxLockedTargets *int64   `json:"max_locked_targets"`
}

type FittingDisplay struct {
	Powergrid   *float64 `json:"powergrid"`
	CPU         *float64 `json:"cpu"`
	Calibration *float64 `json:"calibration"`
}

type DogmaEvalOutput struct {
	Ship        DogmaShip      `json:"ship"`
	ModuleCount int            `json:"module_count"`
	DroneCount  int            `json:"drone_count"`
	Skills      string         `json:"skills"`
	Stats       HullDisplay    `json:"stats"`
	Fitting     FittingDisplay `json:"fitting"`
	DPSNote     string         `json:"dps_note,omitempty"`
	Source      string         `json:"source"`
	KillmailID  *int64         `json:"killmail_id,omitempty"`
	FitHash     *string        `json:"fit_hash,omitempty"`
}

type FitCompareInput struct {
	A      DogmaFitInput `json:"a"`
	B      DogmaFitInput `json:"b"`
	Skills string        `json:"skills,omitempty" enum:"all_v,none" default:"all_v"`
}

type FitComparisonSide struct {
	Ship    DogmaShip   `json:"ship"`
	Stats   HullDisplay `json:"stats"`
	DPSNote string      `json:"dps_note,omitempty"`
}

type NumericDiff struct {
	A        *float64 `json:"a"`
	B        *float64 `json:"b"`
	Delta    *float64 `json:"delta"`
	DeltaPct *float64 `json:"delta_pct"`
}

type CapDiff struct {
	A *string `json:"a"`
	B *string `json:"b"`
}

type FitDiff struct {
	DPS              *NumericDiff `json:"dps"`
	Alpha            *NumericDiff `json:"alpha"`
	EHP              *NumericDiff `json:"ehp"`
	EHPShield        *NumericDiff `json:"ehp_shield"`
	EHPArmor         *NumericDiff `json:"ehp_armor"`
	EHPHull          *NumericDiff `json:"ehp_hull"`
	AlignTimeSeconds *NumericDiff `json:"align_time_s"`
	MaxVelocityMS    *NumericDiff `json:"max_velocity_ms"`
	SignatureRadiusM *NumericDiff `json:"signature_radius_m"`
	Cap              CapDiff      `json:"cap"`
}

type FitCompareOutput struct {
	A      FitComparisonSide `json:"a"`
	B      FitComparisonSide `json:"b"`
	Diff   FitDiff           `json:"diff"`
	Skills string            `json:"skills"`
}

type builtDogmaFit struct {
	fit        EsfFit
	shipName   *string
	killmailID *int64
	fitHash    *string
	source     string
}

type fittingItem struct {
	slotGroup, ordinal int
	typeID             int64
	chargeTypeID       *int64
	quantity           int
}

func registerDogmaTools(registry *Registry) error {
	if err := addTool(registry, ToolDefinition{
		Name: "dogma_eval", Title: "Evaluate a ship fit",
		Description: "Compute DPS, alpha, EHP, capacitor, navigation, targeting, and fitting stats for a stored, EFT, or structured fit.",
	}, func(ctx context.Context, input DogmaEvalInput) (DogmaEvalOutput, error) {
		return dogmaEval(ctx, registry.deps, input)
	}); err != nil {
		return err
	}
	return addTool(registry, ToolDefinition{
		Name: "fit_compare", Title: "Compare two ship fits",
		Description: "Evaluate and diff two stored, EFT, or structured fits across damage, tank, capacitor, navigation, and signature stats.",
	}, func(ctx context.Context, input FitCompareInput) (FitCompareOutput, error) {
		return fitCompare(ctx, registry.deps, input)
	})
}

func dogmaEval(ctx context.Context, deps Dependencies, input DogmaEvalInput) (DogmaEvalOutput, error) {
	built, err := buildDogmaFit(ctx, deps, input.DogmaFitInput)
	if err != nil {
		return DogmaEvalOutput{}, err
	}
	skills := input.Skills
	if skills == "" {
		skills = "all_v"
	}
	hull, err := evaluateDogma(ctx, built.fit, skills == "none")
	if err != nil {
		return DogmaEvalOutput{}, err
	}
	return DogmaEvalOutput{
		Ship:        DogmaShip{TypeID: built.fit.ShipTypeID, Name: built.shipName},
		ModuleCount: len(built.fit.Modules), DroneCount: len(built.fit.Drones), Skills: skills,
		Stats: hullDisplay(hull), Fitting: fittingDisplay(hull), DPSNote: dogmaDPSNote(built.fit, hull.DPSWithReload),
		Source: built.source, KillmailID: built.killmailID, FitHash: built.fitHash,
	}, nil
}

func fitCompare(ctx context.Context, deps Dependencies, input FitCompareInput) (FitCompareOutput, error) {
	a, err := buildDogmaFit(ctx, deps, input.A)
	if err != nil {
		return FitCompareOutput{}, fmt.Errorf("a: %w", err)
	}
	b, err := buildDogmaFit(ctx, deps, input.B)
	if err != nil {
		return FitCompareOutput{}, fmt.Errorf("b: %w", err)
	}
	skills := input.Skills
	if skills == "" {
		skills = "all_v"
	}
	aHull, err := evaluateDogma(ctx, a.fit, skills == "none")
	if err != nil {
		return FitCompareOutput{}, err
	}
	bHull, err := evaluateDogma(ctx, b.fit, skills == "none")
	if err != nil {
		return FitCompareOutput{}, err
	}
	return FitCompareOutput{
		A:    FitComparisonSide{Ship: DogmaShip{TypeID: a.fit.ShipTypeID, Name: a.shipName}, Stats: hullDisplay(aHull), DPSNote: dogmaDPSNote(a.fit, aHull.DPSWithReload)},
		B:    FitComparisonSide{Ship: DogmaShip{TypeID: b.fit.ShipTypeID, Name: b.shipName}, Stats: hullDisplay(bHull), DPSNote: dogmaDPSNote(b.fit, bHull.DPSWithReload)},
		Diff: diffHullStats(aHull, bHull), Skills: skills,
	}, nil
}

func buildDogmaFit(ctx context.Context, deps Dependencies, input DogmaFitInput) (builtDogmaFit, error) {
	if input.KillmailID != nil {
		rows, err := queryMaps(ctx, deps.DB, `
			SELECT fitting.fit_hash, fitting.ship_type_id, type.name AS ship_name
			FROM killmail_fittings fitting LEFT JOIN inv_types type ON type.type_id = fitting.ship_type_id
			WHERE fitting.killmail_id = $1 LIMIT 1`, *input.KillmailID)
		if err != nil {
			return builtDogmaFit{}, err
		}
		row := firstMap(rows)
		if row == nil {
			return builtDogmaFit{}, fmt.Errorf("no stored fit for killmail %d", *input.KillmailID)
		}
		hash := valueString(row["fit_hash"])
		items, err := loadFittingItems(ctx, deps, hash)
		return builtDogmaFit{fit: fittingItemsToEsf(valueInt64(row["ship_type_id"]), items), shipName: nullableString(row["ship_name"]), killmailID: input.KillmailID, fitHash: &hash, source: "killmail"}, err
	}
	if input.FitHash != nil && *input.FitHash != "" {
		rows, err := queryMaps(ctx, deps.DB, `
			SELECT fitting.fit_hash, fitting.ship_type_id, type.name AS ship_name
			FROM fittings fitting LEFT JOIN inv_types type ON type.type_id = fitting.ship_type_id
			WHERE fitting.fit_hash = $1 LIMIT 1`, *input.FitHash)
		if err != nil {
			return builtDogmaFit{}, err
		}
		row := firstMap(rows)
		if row == nil {
			return builtDogmaFit{}, fmt.Errorf("no fit found for fit_hash %s", *input.FitHash)
		}
		items, err := loadFittingItems(ctx, deps, *input.FitHash)
		return builtDogmaFit{fit: fittingItemsToEsf(valueInt64(row["ship_type_id"]), items), shipName: nullableString(row["ship_name"]), fitHash: input.FitHash, source: "fit_hash"}, err
	}
	if input.EFT != nil && strings.TrimSpace(*input.EFT) != "" {
		return buildEFTFit(ctx, deps, *input.EFT)
	}
	if input.ShipTypeID != nil && len(input.Modules) > 0 {
		fit := EsfFit{ShipTypeID: *input.ShipTypeID, Modules: []EsfModule{}, Drones: []EsfDrone{}}
		next := map[string]int{}
		for _, module := range input.Modules {
			slot, state := dogmaSlot(module.Slot)
			if slot == "" {
				return builtDogmaFit{}, fmt.Errorf("unknown slot %s", module.Slot)
			}
			index := next[module.Slot]
			if module.Index != nil {
				index = *module.Index
			}
			next[module.Slot] = index + 1
			item := EsfModule{TypeID: module.TypeID, Slot: EsfSlot{Type: slot, Index: index}, State: state}
			if module.ChargeTypeID != nil {
				item.Charge = &EsfCharge{TypeID: *module.ChargeTypeID}
			}
			fit.Modules = append(fit.Modules, item)
		}
		for _, drone := range input.Drones {
			quantity := drone.Quantity
			if quantity == 0 {
				quantity = 1
			}
			for range quantity {
				fit.Drones = append(fit.Drones, EsfDrone{TypeID: drone.TypeID, State: "Active"})
			}
		}
		name, err := typeName(ctx, deps, *input.ShipTypeID)
		return builtDogmaFit{fit: fit, shipName: name, source: "structured"}, err
	}
	return builtDogmaFit{}, fmt.Errorf("provide killmail_id, fit_hash, eft, or ship_type_id with modules")
}

func loadFittingItems(ctx context.Context, deps Dependencies, hash string) ([]fittingItem, error) {
	rows, err := queryMaps(ctx, deps.DB, `
		SELECT slot_group, ordinal, type_id, charge_type_id, quantity
		FROM fitting_items WHERE fit_hash = $1 ORDER BY slot_group, ordinal`, hash)
	if err != nil {
		return nil, err
	}
	output := make([]fittingItem, 0, len(rows))
	for _, row := range rows {
		output = append(output, fittingItem{slotGroup: int(valueInt64(row["slot_group"])), ordinal: int(valueInt64(row["ordinal"])), typeID: valueInt64(row["type_id"]), chargeTypeID: nullableInt64(row["charge_type_id"]), quantity: int(valueInt64(row["quantity"]))})
	}
	return output, nil
}

func fittingItemsToEsf(shipTypeID int64, items []fittingItem) EsfFit {
	fit := EsfFit{ShipTypeID: shipTypeID, Modules: []EsfModule{}, Drones: []EsfDrone{}}
	for _, item := range items {
		if item.slotGroup == 6 {
			for range max(1, item.quantity) {
				fit.Drones = append(fit.Drones, EsfDrone{TypeID: item.typeID, State: "Active"})
			}
			continue
		}
		slot, state := dogmaSlot(map[int]string{1: "high", 2: "med", 3: "low", 4: "rig", 5: "subsystem"}[item.slotGroup])
		if slot == "" {
			continue
		}
		module := EsfModule{TypeID: item.typeID, Slot: EsfSlot{Type: slot, Index: item.ordinal}, State: state}
		if item.chargeTypeID != nil {
			module.Charge = &EsfCharge{TypeID: *item.chargeTypeID}
		}
		fit.Modules = append(fit.Modules, module)
	}
	return fit
}

func dogmaSlot(slot string) (string, string) {
	switch slot {
	case "high":
		return "High", "Active"
	case "med":
		return "Medium", "Active"
	case "low":
		return "Low", "Online"
	case "rig":
		return "Rig", "Passive"
	case "subsystem":
		return "SubSystem", "Online"
	default:
		return "", ""
	}
}

type parsedEFT struct {
	shipName string
	blocks   [][]string
	drones   []struct {
		name     string
		quantity int
	}
}

var eftDronePattern = regexp.MustCompile(`^(.+)\s+x(\d+)$`)

func buildEFTFit(ctx context.Context, deps Dependencies, text string) (builtDogmaFit, error) {
	parsed, err := parseEFT(text)
	if err != nil {
		return builtDogmaFit{}, err
	}
	names := []string{parsed.shipName}
	for _, block := range parsed.blocks {
		for _, line := range block {
			module, charge := splitEFTCharge(line)
			names = append(names, module)
			if charge != "" {
				names = append(names, charge)
			}
		}
	}
	for _, drone := range parsed.drones {
		names = append(names, drone.name)
	}
	typeIDs, err := resolveTypeNames(ctx, deps, names)
	if err != nil {
		return builtDogmaFit{}, err
	}
	shipID, ok := typeIDs[strings.ToLower(parsed.shipName)]
	if !ok {
		return builtDogmaFit{}, fmt.Errorf("unknown ship %s", parsed.shipName)
	}
	fit := EsfFit{ShipTypeID: shipID, Modules: []EsfModule{}, Drones: []EsfDrone{}}
	slots := []string{"low", "med", "high", "rig", "subsystem"}
	for index, block := range parsed.blocks {
		if index >= len(slots) {
			break
		}
		slot, state := dogmaSlot(slots[index])
		for ordinal, line := range block {
			moduleName, chargeName := splitEFTCharge(line)
			moduleID, exists := typeIDs[strings.ToLower(moduleName)]
			if !exists {
				continue
			}
			module := EsfModule{TypeID: moduleID, Slot: EsfSlot{Type: slot, Index: ordinal}, State: state}
			if chargeID, exists := typeIDs[strings.ToLower(chargeName)]; exists && chargeName != "" {
				module.Charge = &EsfCharge{TypeID: chargeID}
			}
			fit.Modules = append(fit.Modules, module)
		}
	}
	for _, drone := range parsed.drones {
		if id, exists := typeIDs[strings.ToLower(drone.name)]; exists {
			for range drone.quantity {
				fit.Drones = append(fit.Drones, EsfDrone{TypeID: id, State: "Active"})
			}
		}
	}
	name := parsed.shipName
	return builtDogmaFit{fit: fit, shipName: &name, source: "eft"}, nil
}

func parseEFT(text string) (parsedEFT, error) {
	lines := strings.Split(strings.ReplaceAll(text, "\r\n", "\n"), "\n")
	if len(lines) == 0 {
		return parsedEFT{}, fmt.Errorf("empty EFT")
	}
	header := strings.TrimSpace(lines[0])
	if !strings.HasPrefix(header, "[") || !strings.HasSuffix(header, "]") {
		return parsedEFT{}, fmt.Errorf("could not parse EFT header")
	}
	inner := strings.TrimSuffix(strings.TrimPrefix(header, "["), "]")
	shipName := strings.TrimSpace(strings.SplitN(inner, ",", 2)[0])
	blocks, current := [][]string{}, []string{}
	for _, raw := range lines[1:] {
		line := strings.TrimSpace(raw)
		if line == "" {
			if len(current) > 0 {
				blocks, current = append(blocks, current), nil
			}
			continue
		}
		current = append(current, line)
	}
	if len(current) > 0 {
		blocks = append(blocks, current)
	}
	output := parsedEFT{shipName: shipName}
	for len(blocks) < 4 {
		blocks = append(blocks, nil)
	}
	output.blocks = append(output.blocks, blocks[:4]...)
	subsystems := []string{}
	for _, block := range blocks[4:] {
		for _, line := range block {
			match := eftDronePattern.FindStringSubmatch(line)
			if len(match) == 3 {
				quantity := 1
				_, _ = fmt.Sscan(match[2], &quantity)
				output.drones = append(output.drones, struct {
					name     string
					quantity int
				}{strings.TrimSpace(match[1]), quantity})
			} else {
				subsystems = append(subsystems, line)
			}
		}
	}
	output.blocks = append(output.blocks, subsystems)
	return output, nil
}

func splitEFTCharge(line string) (string, string) {
	parts := strings.SplitN(line, ",", 2)
	if len(parts) == 1 {
		return strings.TrimSpace(parts[0]), ""
	}
	return strings.TrimSpace(parts[0]), strings.TrimSpace(parts[1])
}

func resolveTypeNames(ctx context.Context, deps Dependencies, names []string) (map[string]int64, error) {
	lower := make([]string, 0, len(names))
	for _, name := range names {
		if value := strings.ToLower(strings.TrimSpace(name)); value != "" {
			lower = append(lower, value)
		}
	}
	rows, err := queryMaps(ctx, deps.DB, `
		SELECT type_id, LOWER(name) AS lower_name FROM inv_types
		WHERE LOWER(name) = ANY($1) AND published = true`, lower)
	if err != nil {
		return nil, err
	}
	output := map[string]int64{}
	for _, row := range rows {
		output[valueString(row["lower_name"])] = valueInt64(row["type_id"])
	}
	return output, nil
}

func typeName(ctx context.Context, deps Dependencies, id int64) (*string, error) {
	rows, err := queryMaps(ctx, deps.DB, `SELECT name FROM inv_types WHERE type_id = $1 LIMIT 1`, id)
	if err != nil {
		return nil, err
	}
	return nullableString(firstMap(rows)["name"]), nil
}

func hullDisplay(hull HullStats) HullDisplay {
	var locked *int64
	if hull.MaxLockedTargets != nil {
		value := int64(math.Floor(*hull.MaxLockedTargets))
		locked = &value
	}
	var rangeKM *float64
	if hull.MaxTargetRange != nil {
		value := *hull.MaxTargetRange / 1000
		rangeKM = rounded(&value, 1)
	}
	var capTime *float64
	if hull.CapDepletesIn != nil && *hull.CapDepletesIn >= 0 {
		capTime = rounded(hull.CapDepletesIn, 1)
	}
	return HullDisplay{
		DPSWithReload: rounded(hull.DPSWithReload, 1), DPSWithoutReload: rounded(hull.DPSWithoutReload, 1),
		Alpha: rounded(hull.Alpha, 0), EHP: rounded(hull.EHP, 0), EHPShield: rounded(hull.ShieldEHP, 0),
		EHPArmor: rounded(hull.ArmorEHP, 0), EHPHull: rounded(hull.HullEHP, 0),
		CapStable: hull.CapDepletesIn != nil && *hull.CapDepletesIn < 0, CapTimeSeconds: capTime,
		CapCapacityGJ: rounded(hull.CapCapacity, 1), CapPeakDeltaGJS: rounded(hull.CapPeakDelta, 2),
		AlignTimeSeconds: rounded(hull.AlignTime, 2), MaxVelocityMS: rounded(hull.MaxVelocity, 0),
		SignatureRadiusM: rounded(hull.SignatureRadius, 1), MaxTargetRangeKM: rangeKM,
		ScanResolutionMM: rounded(hull.ScanResolution, 0), MaxLockedTargets: locked,
	}
}

func fittingDisplay(hull HullStats) FittingDisplay {
	return FittingDisplay{Powergrid: rounded(hull.PGOutput, 1), CPU: rounded(hull.CPUOutput, 1), Calibration: rounded(hull.Calibration, 0)}
}

func dogmaDPSNote(fit EsfFit, dps *float64) string {
	if dps != nil && *dps > 0 {
		return ""
	}
	highSlots, highWithCharge := 0, 0
	for _, module := range fit.Modules {
		if module.Slot.Type == "High" {
			highSlots++
			if module.Charge != nil {
				highWithCharge++
			}
		}
	}
	if highSlots == 0 && len(fit.Drones) == 0 {
		return "non_combat_fit: no high-slot modules and no drones — 0 DPS is by design"
	}
	if highSlots > 0 && highWithCharge == 0 {
		return "no_charges_loaded: high-slot modules fitted but no ammo/charges — DPS cannot be computed"
	}
	return "zero_dps_unexpected: high-slot modules and charges present but engine returned 0 DPS"
}

func rounded(value *float64, digits int) *float64 {
	if value == nil || math.IsNaN(*value) || math.IsInf(*value, 0) {
		return nil
	}
	multiplier := math.Pow10(digits)
	result := math.Round(*value*multiplier) / multiplier
	return &result
}

func diffHullStats(a, b HullStats) FitDiff {
	return FitDiff{
		DPS: numericDiff(a.DPSWithReload, b.DPSWithReload, 1), Alpha: numericDiff(a.Alpha, b.Alpha, 0),
		EHP: numericDiff(a.EHP, b.EHP, 0), EHPShield: numericDiff(a.ShieldEHP, b.ShieldEHP, 0),
		EHPArmor: numericDiff(a.ArmorEHP, b.ArmorEHP, 0), EHPHull: numericDiff(a.HullEHP, b.HullEHP, 0),
		AlignTimeSeconds: numericDiff(a.AlignTime, b.AlignTime, 2), MaxVelocityMS: numericDiff(a.MaxVelocity, b.MaxVelocity, 0),
		SignatureRadiusM: numericDiff(a.SignatureRadius, b.SignatureRadius, 1),
		Cap:              CapDiff{A: capDescription(a.CapDepletesIn), B: capDescription(b.CapDepletesIn)},
	}
}

func numericDiff(a, b *float64, digits int) *NumericDiff {
	if a == nil || b == nil {
		return nil
	}
	delta := *b - *a
	var percentage *float64
	if *a != 0 {
		value := delta / *a * 100
		percentage = rounded(&value, 1)
	}
	return &NumericDiff{A: rounded(a, digits), B: rounded(b, digits), Delta: rounded(&delta, digits), DeltaPct: percentage}
}

func capDescription(value *float64) *string {
	if value == nil {
		return nil
	}
	description := "stable"
	if *value >= 0 {
		description = fmt.Sprintf("%.0fs", *value)
	}
	return &description
}
