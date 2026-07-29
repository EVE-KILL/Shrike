package mcpserver

import (
	"context"
	"fmt"
	"html"
	"math"
	"regexp"
	"strings"
)

const (
	attrMass              = 4
	attrHullHP            = 9
	attrPowergrid         = 11
	attrSlotsLow          = 12
	attrSlotsMed          = 13
	attrSlotsHigh         = 14
	attrMaxVelocity       = 37
	attrCPU               = 48
	attrCapRecharge       = 55
	attrAgility           = 70
	attrMaxRange          = 76
	attrLauncherSlots     = 101
	attrTurretSlots       = 102
	attrMaxTargets        = 192
	attrScanRadar         = 208
	attrScanLadar         = 209
	attrScanMagnetometric = 210
	attrScanGravimetric   = 211
	attrShieldHP          = 263
	attrArmorHP           = 265
	attrArmorEM           = 267
	attrArmorExplosive    = 268
	attrArmorKinetic      = 269
	attrArmorThermal      = 270
	attrShieldEM          = 271
	attrShieldThermal     = 272
	attrShieldKinetic     = 273
	attrShieldExplosive   = 274
	attrDroneBay          = 283
	attrCapCapacity       = 482
	attrSignature         = 552
	attrScanResolution    = 564
	attrHullEM            = 974
	attrHullExplosive     = 975
	attrHullKinetic       = 976
	attrHullThermal       = 977
	attrSpecialBay        = 1055
	attrCalibration       = 1132
	attrSlotsRig          = 1137
	attrDroneBandwidth    = 1271
	attrSlotsSubsystem    = 1367
)

type ShipInfoInput struct {
	Ship StringOrInt64 `json:"ship" jsonschema:"Hull name or type identifier."`
}

type HullSlots struct {
	High               *int64 `json:"high"`
	Med                *int64 `json:"med"`
	Low                *int64 `json:"low"`
	Rig                *int64 `json:"rig"`
	Subsystem          *int64 `json:"subsystem"`
	TurretHardpoints   *int64 `json:"turret_hardpoints"`
	LauncherHardpoints *int64 `json:"launcher_hardpoints"`
}

type HullFitting struct {
	Powergrid   *float64 `json:"powergrid"`
	CPU         *float64 `json:"cpu"`
	Calibration *float64 `json:"calibration"`
}

type HullHP struct {
	Shield *float64 `json:"shield"`
	Armor  *float64 `json:"armor"`
	Hull   *float64 `json:"hull"`
}

type ResistLayer struct {
	EM        *float64 `json:"em"`
	Thermal   *float64 `json:"thermal"`
	Kinetic   *float64 `json:"kinetic"`
	Explosive *float64 `json:"explosive"`
}

type HullResists struct {
	Shield ResistLayer `json:"shield"`
	Armor  ResistLayer `json:"armor"`
	Hull   ResistLayer `json:"hull"`
}

type HullCapacitor struct {
	Capacity  *float64 `json:"capacity"`
	RechargeS *float64 `json:"recharge_s"`
}

type HullMobility struct {
	Mass             *float64 `json:"mass"`
	MaxVelocityMS    *float64 `json:"max_velocity_ms"`
	SignatureRadiusM *float64 `json:"signature_radius_m"`
	Agility          *float64 `json:"agility"`
}

type HullSensors struct {
	MaxTargetRangeM       *float64 `json:"max_target_range_m"`
	ScanResolutionMM      *float64 `json:"scan_resolution_mm"`
	MaxLockedTargets      *int64   `json:"max_locked_targets"`
	LadarStrength         *float64 `json:"ladar_strength"`
	RadarStrength         *float64 `json:"radar_strength"`
	MagnetometricStrength *float64 `json:"magnetometric_strength"`
	GravimetricStrength   *float64 `json:"gravimetric_strength"`
}

type HullDrones struct {
	Bay       *float64 `json:"bay"`
	Bandwidth *float64 `json:"bandwidth"`
}

type HullCargo struct {
	Standard   *float64 `json:"standard"`
	SpecialBay *float64 `json:"special_bay"`
}

type ShipInfoOutput struct {
	TypeID           int64         `json:"type_id"`
	Name             string        `json:"name"`
	URL              string        `json:"url"`
	Category         *string       `json:"category"`
	Group            *string       `json:"group"`
	Race             *string       `json:"race"`
	MetaGroup        *string       `json:"meta_group"`
	Slots            HullSlots     `json:"slots"`
	Fitting          HullFitting   `json:"fitting"`
	BaseHP           HullHP        `json:"base_hp"`
	ResistProfile    HullResists   `json:"resist_profile"`
	Capacitor        HullCapacitor `json:"capacitor"`
	Mobility         HullMobility  `json:"mobility"`
	Sensors          HullSensors   `json:"sensors"`
	Drones           HullDrones    `json:"drones"`
	Cargo            HullCargo     `json:"cargo"`
	Description      *string       `json:"description"`
	CurrentJitaPrice *float64      `json:"current_jita_price"`
	MarketNote       string        `json:"market_note,omitempty"`
}

type ItemInfoInput struct {
	Item StringOrInt64 `json:"item" jsonschema:"Item name or type identifier."`
}

type IDName struct {
	ID   *int64  `json:"id"`
	Name *string `json:"name"`
}

type ItemPhysical struct {
	Mass     *float64 `json:"mass"`
	Volume   *float64 `json:"volume"`
	Capacity *float64 `json:"capacity"`
}

type ItemFitting struct {
	CPU         *float64 `json:"cpu"`
	Powergrid   *float64 `json:"powergrid"`
	Calibration *float64 `json:"calibration"`
}

type ItemVariant struct {
	TypeID    int64   `json:"type_id"`
	Name      string  `json:"name"`
	MetaGroup *string `json:"meta_group"`
}

type ItemInfoOutput struct {
	TypeID                int64         `json:"type_id"`
	Name                  string        `json:"name"`
	URL                   string        `json:"url"`
	Category              IDName        `json:"category"`
	Group                 IDName        `json:"group"`
	MetaGroup             IDName        `json:"meta_group"`
	VariationParentTypeID *int64        `json:"variation_parent_type_id"`
	Description           *string       `json:"description"`
	Physical              ItemPhysical  `json:"physical"`
	Fitting               ItemFitting   `json:"fitting"`
	CurrentJitaPrice      *float64      `json:"current_jita_price"`
	MarketNote            string        `json:"market_note,omitempty"`
	Variants              []ItemVariant `json:"variants"`
}

type SystemInfoInput struct {
	System StringOrInt64 `json:"system" jsonschema:"System name or solar system identifier."`
}

type SystemNeighbor struct {
	SolarSystemID int64    `json:"solar_system_id"`
	Name          string   `json:"name"`
	Security      *float64 `json:"security"`
	RegionName    *string  `json:"region_name"`
}

type SystemStation struct {
	StationID int64  `json:"station_id"`
	Name      string `json:"name"`
}

type SystemFlags struct {
	Border        bool `json:"border"`
	Fringe        bool `json:"fringe"`
	Corridor      bool `json:"corridor"`
	Hub           bool `json:"hub"`
	International bool `json:"international"`
	Regional      bool `json:"regional"`
}

type RequiredIDName struct {
	ID   int64   `json:"id"`
	Name *string `json:"name"`
}

type SystemInfoOutput struct {
	SolarSystemID int64            `json:"solar_system_id"`
	Name          string           `json:"name"`
	URL           string           `json:"url"`
	Security      *float64         `json:"security"`
	SecurityBand  *string          `json:"security_band"`
	SecurityClass *string          `json:"security_class"`
	Region        RequiredIDName   `json:"region"`
	Constellation RequiredIDName   `json:"constellation"`
	Faction       *RequiredIDName  `json:"faction"`
	Flags         SystemFlags      `json:"flags"`
	StationCount  int              `json:"station_count"`
	Stations      []SystemStation  `json:"stations"`
	NeighborCount int              `json:"neighbor_count"`
	IsPipeTip     bool             `json:"is_pipe_tip"`
	Neighbors     []SystemNeighbor `json:"neighbors"`
}

type ShipCompareInput struct {
	A StringOrInt64 `json:"a" jsonschema:"First hull name or type identifier."`
	B StringOrInt64 `json:"b" jsonschema:"Second hull name or type identifier."`
}

type ValueDifference struct {
	A        float64  `json:"a"`
	B        float64  `json:"b"`
	Delta    float64  `json:"delta"`
	DeltaPct *float64 `json:"delta_pct"`
}

type HullDifference struct {
	ShieldHP        *ValueDifference `json:"shield_hp,omitempty"`
	ArmorHP         *ValueDifference `json:"armor_hp,omitempty"`
	HullHP          *ValueDifference `json:"hull_hp,omitempty"`
	MaxVelocityMS   *ValueDifference `json:"max_velocity_ms,omitempty"`
	SignatureRadius *ValueDifference `json:"signature_radius_m,omitempty"`
	Powergrid       *ValueDifference `json:"powergrid,omitempty"`
	CPU             *ValueDifference `json:"cpu,omitempty"`
	MaxTargetRangeM *ValueDifference `json:"max_target_range_m,omitempty"`
}

type ShipCompareOutput struct {
	A    ShipInfoOutput `json:"a"`
	B    ShipInfoOutput `json:"b"`
	Diff HullDifference `json:"diff"`
}

func registerSDEInfoTools(registry *Registry) error {
	if err := addTool(registry, ToolDefinition{
		Name:        "ship_info",
		Title:       "Get ship information",
		Description: "Return static hull, slot, fitting, tank, mobility, sensor, drone, cargo, and Jita price data.",
	}, func(ctx context.Context, input ShipInfoInput) (ShipInfoOutput, error) {
		resolved, err := resolveEntity(ctx, registry.deps, input.Ship, entityTypePointer(EntityShip))
		if err != nil {
			return ShipInfoOutput{}, err
		}
		if resolved == nil {
			return ShipInfoOutput{}, fmt.Errorf("could not resolve ship %q", input.Ship.String())
		}
		return loadShipInfo(ctx, registry.deps, resolved.ID)
	}); err != nil {
		return err
	}
	if err := addTool(registry, ToolDefinition{
		Name:        "item_info",
		Title:       "Get item information",
		Description: "Return static category, group, physical, fitting, variation, and Jita price data for an EVE item.",
	}, func(ctx context.Context, input ItemInfoInput) (ItemInfoOutput, error) {
		return loadItemInfo(ctx, registry.deps, input)
	}); err != nil {
		return err
	}
	if err := addTool(registry, ToolDefinition{
		Name:        "system_info",
		Title:       "Get system information",
		Description: "Return security, topology, region, constellation, faction, station, and stargate-neighbor data for a solar system.",
	}, func(ctx context.Context, input SystemInfoInput) (SystemInfoOutput, error) {
		return loadSystemInfo(ctx, registry.deps, input)
	}); err != nil {
		return err
	}
	return addTool(registry, ToolDefinition{
		Name:        "ship_compare",
		Title:       "Compare ships",
		Description: "Compare the static hull stats of two ships side by side.",
	}, func(ctx context.Context, input ShipCompareInput) (ShipCompareOutput, error) {
		return compareShips(ctx, registry.deps, input)
	})
}

func entityTypePointer(value EntityType) *EntityType {
	return &value
}

func loadShipInfo(
	ctx context.Context,
	deps Dependencies,
	typeID int64,
) (ShipInfoOutput, error) {
	typeRows, err := queryMaps(ctx, deps.DB, `
		SELECT item.type_id, item.name, item.description, item.mass,
		       item.volume, item.capacity, item.radius, item.group_id,
		       item_group.name AS group_name, item.category_id,
		       category.name AS category_name, item.meta_group_id,
		       meta_group.name AS meta_group_name, item.race_id, race.race_name
		FROM inv_types item
		LEFT JOIN inv_groups item_group ON item_group.group_id = item.group_id
		LEFT JOIN inv_categories category
		       ON category.category_id = item.category_id
		LEFT JOIN inv_meta_groups meta_group
		       ON meta_group.meta_group_id = item.meta_group_id
		LEFT JOIN races race ON race.race_id = item.race_id
		WHERE item.type_id = $1`, typeID)
	if err != nil {
		return ShipInfoOutput{}, fmt.Errorf("load ship type: %w", err)
	}
	row := firstMap(typeRows)
	if row == nil {
		return ShipInfoOutput{}, fmt.Errorf("ship %d not found", typeID)
	}
	attributes, err := loadDogmaAttributes(ctx, deps, typeID)
	if err != nil {
		return ShipInfoOutput{}, err
	}
	price, err := loadJitaPrice(ctx, deps, typeID)
	if err != nil {
		return ShipInfoOutput{}, err
	}
	resist := func(attribute int) *float64 {
		value, ok := attributes[attribute]
		if !ok {
			return nil
		}
		result := math.Round((1-value)*1000) / 10
		return &result
	}
	output := ShipInfoOutput{
		TypeID:    valueInt64(row["type_id"]),
		Name:      valueString(row["name"]),
		URL:       entityURL(deps.BaseURL, EntityShip, typeID),
		Category:  nullableString(row["category_name"]),
		Group:     nullableString(row["group_name"]),
		Race:      nullableString(row["race_name"]),
		MetaGroup: nullableString(row["meta_group_name"]),
		Slots: HullSlots{
			High:               roundedAttribute(attributes, attrSlotsHigh),
			Med:                roundedAttribute(attributes, attrSlotsMed),
			Low:                roundedAttribute(attributes, attrSlotsLow),
			Rig:                roundedAttribute(attributes, attrSlotsRig),
			Subsystem:          roundedAttribute(attributes, attrSlotsSubsystem),
			TurretHardpoints:   roundedAttribute(attributes, attrTurretSlots),
			LauncherHardpoints: roundedAttribute(attributes, attrLauncherSlots),
		},
		Fitting: HullFitting{
			Powergrid:   attributePointer(attributes, attrPowergrid),
			CPU:         attributePointer(attributes, attrCPU),
			Calibration: attributePointer(attributes, attrCalibration),
		},
		BaseHP: HullHP{
			Shield: attributePointer(attributes, attrShieldHP),
			Armor:  attributePointer(attributes, attrArmorHP),
			Hull:   attributePointer(attributes, attrHullHP),
		},
		ResistProfile: HullResists{
			Shield: ResistLayer{
				EM: resist(attrShieldEM), Thermal: resist(attrShieldThermal),
				Kinetic: resist(attrShieldKinetic), Explosive: resist(attrShieldExplosive),
			},
			Armor: ResistLayer{
				EM: resist(attrArmorEM), Thermal: resist(attrArmorThermal),
				Kinetic: resist(attrArmorKinetic), Explosive: resist(attrArmorExplosive),
			},
			Hull: ResistLayer{
				EM: resist(attrHullEM), Thermal: resist(attrHullThermal),
				Kinetic: resist(attrHullKinetic), Explosive: resist(attrHullExplosive),
			},
		},
		Capacitor: HullCapacitor{
			Capacity: attributePointer(attributes, attrCapCapacity),
		},
		Mobility: HullMobility{
			Mass:             nullableFloat64(row["mass"]),
			MaxVelocityMS:    attributePointer(attributes, attrMaxVelocity),
			SignatureRadiusM: attributePointer(attributes, attrSignature),
			Agility:          attributePointer(attributes, attrAgility),
		},
		Sensors: HullSensors{
			MaxTargetRangeM:       attributePointer(attributes, attrMaxRange),
			ScanResolutionMM:      attributePointer(attributes, attrScanResolution),
			MaxLockedTargets:      roundedAttribute(attributes, attrMaxTargets),
			LadarStrength:         attributePointer(attributes, attrScanLadar),
			RadarStrength:         attributePointer(attributes, attrScanRadar),
			MagnetometricStrength: attributePointer(attributes, attrScanMagnetometric),
			GravimetricStrength:   attributePointer(attributes, attrScanGravimetric),
		},
		Drones: HullDrones{
			Bay:       attributePointer(attributes, attrDroneBay),
			Bandwidth: attributePointer(attributes, attrDroneBandwidth),
		},
		Cargo: HullCargo{
			Standard:   nullableFloat64(row["capacity"]),
			SpecialBay: attributePointer(attributes, attrSpecialBay),
		},
		Description:      stripHTML(nullableString(row["description"])),
		CurrentJitaPrice: price,
	}
	if recharge := attributePointer(attributes, attrCapRecharge); recharge != nil {
		value := math.Round(*recharge/100) / 10
		output.Capacitor.RechargeS = &value
	}
	if price == nil {
		output.MarketNote = "Not actively traded on Jita (common for pirate supers, officer mods, and unpublished types)"
	}
	return output, nil
}

func loadItemInfo(
	ctx context.Context,
	deps Dependencies,
	input ItemInfoInput,
) (ItemInfoOutput, error) {
	resolved, err := resolveEntity(ctx, deps, input.Item, nil)
	if err != nil {
		return ItemInfoOutput{}, err
	}
	var typeID int64
	if resolved != nil && (resolved.Type == EntityItem || resolved.Type == EntityShip) {
		typeID = resolved.ID
	} else if input.Item.IsInt() {
		typeID = input.Item.Int64()
	}
	if typeID <= 0 {
		return ItemInfoOutput{}, fmt.Errorf("could not resolve item %q", input.Item.String())
	}
	rows, err := queryMaps(ctx, deps.DB, `
		SELECT item.type_id, item.name, item.description, item.mass,
		       item.volume, item.capacity, item.meta_group_id,
		       meta_group.name AS meta_group_name, item.group_id,
		       item_group.name AS group_name, item.category_id,
		       category.name AS category_name, item.variation_parent_type_id
		FROM inv_types item
		LEFT JOIN inv_groups item_group ON item_group.group_id = item.group_id
		LEFT JOIN inv_categories category
		       ON category.category_id = item.category_id
		LEFT JOIN inv_meta_groups meta_group
		       ON meta_group.meta_group_id = item.meta_group_id
		WHERE item.type_id = $1`, typeID)
	if err != nil {
		return ItemInfoOutput{}, fmt.Errorf("load item type: %w", err)
	}
	row := firstMap(rows)
	if row == nil {
		return ItemInfoOutput{}, fmt.Errorf("type %d not found", typeID)
	}
	attributes, err := loadDogmaAttributes(ctx, deps, typeID)
	if err != nil {
		return ItemInfoOutput{}, err
	}
	price, err := loadJitaPrice(ctx, deps, typeID)
	if err != nil {
		return ItemInfoOutput{}, err
	}
	variants, err := queryMaps(ctx, deps.DB, `
		SELECT item.type_id, item.name, item.meta_group_id,
		       meta_group.name AS meta_group_name
		FROM inv_types item
		LEFT JOIN inv_meta_groups meta_group
		       ON meta_group.meta_group_id = item.meta_group_id
		WHERE item.published IS TRUE
		  AND (
		    item.variation_parent_type_id = (
		      SELECT COALESCE(variation_parent_type_id, type_id)
		      FROM inv_types WHERE type_id = $1
		    )
		    OR item.type_id = (
		      SELECT COALESCE(variation_parent_type_id, type_id)
		      FROM inv_types WHERE type_id = $1
		    )
		  )
		ORDER BY item.meta_group_id, item.name
		LIMIT 25`, typeID)
	if err != nil {
		return ItemInfoOutput{}, fmt.Errorf("load item variants: %w", err)
	}
	output := ItemInfoOutput{
		TypeID: typeID,
		Name:   valueString(row["name"]),
		URL:    entityURL(deps.BaseURL, EntityItem, typeID),
		Category: IDName{
			ID:   nullableInt64(row["category_id"]),
			Name: nullableString(row["category_name"]),
		},
		Group: IDName{
			ID:   nullableInt64(row["group_id"]),
			Name: nullableString(row["group_name"]),
		},
		MetaGroup: IDName{
			ID:   nullableInt64(row["meta_group_id"]),
			Name: nullableString(row["meta_group_name"]),
		},
		VariationParentTypeID: nullableInt64(row["variation_parent_type_id"]),
		Description:           stripHTML(nullableString(row["description"])),
		Physical: ItemPhysical{
			Mass:     nullableFloat64(row["mass"]),
			Volume:   nullableFloat64(row["volume"]),
			Capacity: nullableFloat64(row["capacity"]),
		},
		Fitting: ItemFitting{
			CPU:         attributePointer(attributes, 50),
			Powergrid:   attributePointer(attributes, 30),
			Calibration: attributePointer(attributes, 1153),
		},
		CurrentJitaPrice: price,
		Variants:         []ItemVariant{},
	}
	if price == nil {
		output.MarketNote = "Not actively traded on Jita"
	}
	for _, variant := range variants {
		output.Variants = append(output.Variants, ItemVariant{
			TypeID:    valueInt64(variant["type_id"]),
			Name:      valueString(variant["name"]),
			MetaGroup: nullableString(variant["meta_group_name"]),
		})
	}
	return output, nil
}

func loadSystemInfo(
	ctx context.Context,
	deps Dependencies,
	input SystemInfoInput,
) (SystemInfoOutput, error) {
	resolved, err := resolveEntity(
		ctx, deps, input.System, entityTypePointer(EntitySystem),
	)
	if err != nil {
		return SystemInfoOutput{}, err
	}
	if resolved == nil {
		return SystemInfoOutput{}, fmt.Errorf("could not resolve system %q", input.System.String())
	}
	rows, err := queryMaps(ctx, deps.DB, `
		SELECT system.*, region.name AS region_name,
		       constellation.constellation_name, faction.name AS faction_name
		FROM solar_systems system
		LEFT JOIN regions region ON region.region_id = system.region_id
		LEFT JOIN constellations constellation
		       ON constellation.constellation_id = system.constellation_id
		LEFT JOIN factions faction ON faction.faction_id = system.faction_id
		WHERE system.solar_system_id = $1`, resolved.ID)
	if err != nil {
		return SystemInfoOutput{}, fmt.Errorf("load solar system: %w", err)
	}
	system := firstMap(rows)
	if system == nil {
		return SystemInfoOutput{}, fmt.Errorf("system %d not found", resolved.ID)
	}
	neighborRows, err := queryMaps(ctx, deps.DB, `
		SELECT DISTINCT neighbor.solar_system_id, neighbor.system_name,
		       neighbor.security, region.name AS region_name
		FROM solar_system_jumps jump
		JOIN solar_systems neighbor
		     ON neighbor.solar_system_id = jump.to_solar_system_id
		LEFT JOIN regions region ON region.region_id = neighbor.region_id
		WHERE jump.from_solar_system_id = $1
		UNION
		SELECT DISTINCT neighbor.solar_system_id, neighbor.system_name,
		       neighbor.security, region.name AS region_name
		FROM solar_system_jumps jump
		JOIN solar_systems neighbor
		     ON neighbor.solar_system_id = jump.from_solar_system_id
		LEFT JOIN regions region ON region.region_id = neighbor.region_id
		WHERE jump.to_solar_system_id = $1`, resolved.ID)
	if err != nil {
		return SystemInfoOutput{}, fmt.Errorf("load system neighbors: %w", err)
	}
	stationRows, err := queryMaps(ctx, deps.DB, `
		SELECT station_id, station_name AS name, type_id
		FROM stations
		WHERE solar_system_id = $1`, resolved.ID)
	if err != nil {
		return SystemInfoOutput{}, fmt.Errorf("load system stations: %w", err)
	}
	security := nullableFloat64(system["security"])
	var securityBand *string
	if security != nil {
		band := "nullsec"
		if *security >= 0.45 {
			band = "highsec"
		} else if *security > 0 {
			band = "lowsec"
		}
		securityBand = &band
		rounded := math.Round(*security*100) / 100
		security = &rounded
	}
	output := SystemInfoOutput{
		SolarSystemID: resolved.ID,
		Name:          valueString(system["system_name"]),
		URL:           entityURL(deps.BaseURL, EntitySystem, resolved.ID),
		Security:      security,
		SecurityBand:  securityBand,
		SecurityClass: nullableString(system["security_class"]),
		Region: RequiredIDName{
			ID:   valueInt64(system["region_id"]),
			Name: nullableString(system["region_name"]),
		},
		Constellation: RequiredIDName{
			ID:   valueInt64(system["constellation_id"]),
			Name: nullableString(system["constellation_name"]),
		},
		Flags: SystemFlags{
			Border:        valueBool(system["border"]),
			Fringe:        valueBool(system["fringe"]),
			Corridor:      valueBool(system["corridor"]),
			Hub:           valueBool(system["hub"]),
			International: valueBool(system["international"]),
			Regional:      valueBool(system["regional"]),
		},
		StationCount:  len(stationRows),
		Stations:      []SystemStation{},
		NeighborCount: len(neighborRows),
		IsPipeTip:     len(neighborRows) == 1,
		Neighbors:     []SystemNeighbor{},
	}
	if system["faction_id"] != nil {
		output.Faction = &RequiredIDName{
			ID:   valueInt64(system["faction_id"]),
			Name: nullableString(system["faction_name"]),
		}
	}
	for index, row := range stationRows {
		if index == 10 {
			break
		}
		output.Stations = append(output.Stations, SystemStation{
			StationID: valueInt64(row["station_id"]),
			Name:      valueString(row["name"]),
		})
	}
	for _, row := range neighborRows {
		neighborSecurity := nullableFloat64(row["security"])
		if neighborSecurity != nil {
			value := math.Round(*neighborSecurity*100) / 100
			neighborSecurity = &value
		}
		output.Neighbors = append(output.Neighbors, SystemNeighbor{
			SolarSystemID: valueInt64(row["solar_system_id"]),
			Name:          valueString(row["system_name"]),
			Security:      neighborSecurity,
			RegionName:    nullableString(row["region_name"]),
		})
	}
	return output, nil
}

func compareShips(
	ctx context.Context,
	deps Dependencies,
	input ShipCompareInput,
) (ShipCompareOutput, error) {
	first, err := resolveEntity(ctx, deps, input.A, entityTypePointer(EntityShip))
	if err != nil {
		return ShipCompareOutput{}, err
	}
	if first == nil {
		return ShipCompareOutput{}, fmt.Errorf("could not resolve ship a %q", input.A.String())
	}
	second, err := resolveEntity(ctx, deps, input.B, entityTypePointer(EntityShip))
	if err != nil {
		return ShipCompareOutput{}, err
	}
	if second == nil {
		return ShipCompareOutput{}, fmt.Errorf("could not resolve ship b %q", input.B.String())
	}
	a, err := loadShipInfo(ctx, deps, first.ID)
	if err != nil {
		return ShipCompareOutput{}, err
	}
	b, err := loadShipInfo(ctx, deps, second.ID)
	if err != nil {
		return ShipCompareOutput{}, err
	}
	return ShipCompareOutput{
		A: a,
		B: b,
		Diff: HullDifference{
			ShieldHP:        difference(a.BaseHP.Shield, b.BaseHP.Shield, 0),
			ArmorHP:         difference(a.BaseHP.Armor, b.BaseHP.Armor, 0),
			HullHP:          difference(a.BaseHP.Hull, b.BaseHP.Hull, 0),
			MaxVelocityMS:   difference(a.Mobility.MaxVelocityMS, b.Mobility.MaxVelocityMS, 0),
			SignatureRadius: difference(a.Mobility.SignatureRadiusM, b.Mobility.SignatureRadiusM, 1),
			Powergrid:       difference(a.Fitting.Powergrid, b.Fitting.Powergrid, 1),
			CPU:             difference(a.Fitting.CPU, b.Fitting.CPU, 1),
			MaxTargetRangeM: difference(a.Sensors.MaxTargetRangeM, b.Sensors.MaxTargetRangeM, 0),
		},
	}, nil
}

func difference(a, b *float64, digits int) *ValueDifference {
	if a == nil || b == nil {
		return nil
	}
	delta := *b - *a
	var deltaPct *float64
	if *a != 0 {
		value := math.Round(delta / *a * 1000) / 10
		deltaPct = &value
	}
	return &ValueDifference{
		A: roundTo(*a, digits), B: roundTo(*b, digits),
		Delta: roundTo(delta, digits), DeltaPct: deltaPct,
	}
}

func roundTo(value float64, digits int) float64 {
	multiplier := math.Pow10(digits)
	return math.Round(value*multiplier) / multiplier
}

func loadDogmaAttributes(
	ctx context.Context,
	deps Dependencies,
	typeID int64,
) (map[int]float64, error) {
	rows, err := queryMaps(ctx, deps.DB, `
		SELECT attribute_id, value
		FROM type_dogma_attributes
		WHERE type_id = $1`, typeID)
	if err != nil {
		return nil, fmt.Errorf("load dogma attributes: %w", err)
	}
	result := make(map[int]float64, len(rows))
	for _, row := range rows {
		result[int(valueInt64(row["attribute_id"]))] = valueFloat64(row["value"])
	}
	return result, nil
}

func loadJitaPrice(
	ctx context.Context,
	deps Dependencies,
	typeID int64,
) (*float64, error) {
	rows, err := queryMaps(ctx, deps.DB, `
		SELECT average
		FROM prices
		WHERE type_id = $1 AND region_id = 10000002
		ORDER BY date DESC
		LIMIT 1`, typeID)
	if err != nil {
		return nil, fmt.Errorf("load Jita price: %w", err)
	}
	if len(rows) == 0 {
		return nil, nil
	}
	price := valueFloat64(rows[0]["average"])
	if price == 0 {
		return nil, nil
	}
	return &price, nil
}

func attributePointer(attributes map[int]float64, id int) *float64 {
	value, ok := attributes[id]
	if !ok {
		return nil
	}
	return &value
}

func roundedAttribute(attributes map[int]float64, id int) *int64 {
	value, ok := attributes[id]
	if !ok {
		return nil
	}
	result := int64(math.Round(value))
	return &result
}

var (
	breakPattern = regexp.MustCompile(`(?i)<br\s*/?>`)
	tagPattern   = regexp.MustCompile(`<[^>]+>`)
)

func stripHTML(value *string) *string {
	if value == nil || *value == "" {
		return nil
	}
	text := breakPattern.ReplaceAllString(*value, "\n")
	text = tagPattern.ReplaceAllString(text, "")
	text = html.UnescapeString(text)
	lines := strings.Split(text, "\n")
	for index := range lines {
		lines[index] = strings.TrimRight(lines[index], " \t")
	}
	text = strings.TrimSpace(strings.Join(lines, "\n"))
	return &text
}
