package sde

// Tables declares the straightforward one-member-to-one-table imports,
// transcribed from the importTable() calls in
// backend/src/commands/UpdateSdeCommand.ts.
//
// Column types come from internal/dbgen (generated from the schema), which is
// what decides the accessor used for each field: Int for integer columns, Float
// for double precision, Lang for localised text, Str for bare strings.
//
// Not covered here, because they are not one-to-one and get their own passes:
// typeDogma (two tables), typeMaterials, blueprints (five tables), the map
// celestial members (five members into one table), stargate-derived system
// jumps, inv_flags (a hardcoded list, not in the archive), and the EVE Ref
// feeds.
var Tables = []Table{
	{
		Member: "types",
		Name:   "inv_types",
		PK:     []string{"type_id"},
		// category_id is deliberately absent: it is denormalised from
		// inv_groups by a post-import fill, not present on the SDE record.
		Columns: []string{
			"type_id", "group_id", "name", "description", "mass", "volume",
			"capacity", "portion_size", "packaged_volume", "radius", "published",
			"market_group_id", "icon_id", "sound_id", "graphic_id", "meta_group_id",
			"race_id", "faction_id", "base_price", "variation_parent_type_id",
		},
		Values: func(r Row) ([]any, bool) {
			id, ok := r.Key()
			if !ok {
				return nil, false
			}
			return []any{
				int32(id), r.Int("groupID"), r.Lang("name"), r.Lang("description"),
				r.Float("mass"), r.Float("volume"), r.Float("capacity"),
				r.Int("portionSize"), r.Float("packagedVolume"), r.Float("radius"),
				r.Bool("published"), r.Int("marketGroupID"), r.Int("iconID"),
				r.Int("soundID"), r.Int("graphicID"), r.Int("metaGroupID"),
				r.Int("raceID"), r.Int("factionID"), r.Float("basePrice"),
				r.Int("variationParentTypeID"),
			}, true
		},
	},
	{
		Member:  "groups",
		Name:    "inv_groups",
		PK:      []string{"group_id"},
		Columns: []string{"group_id", "category_id", "name", "published", "icon_id"},
		Values: func(r Row) ([]any, bool) {
			id, ok := r.Key()
			if !ok {
				return nil, false
			}
			return []any{
				int32(id), r.Int("categoryID"), r.Lang("name"),
				r.Bool("published"), r.Int("iconID"),
			}, true
		},
	},
	{
		Member:  "categories",
		Name:    "inv_categories",
		PK:      []string{"category_id"},
		Columns: []string{"category_id", "name", "published", "icon_id"},
		Values: func(r Row) ([]any, bool) {
			id, ok := r.Key()
			if !ok {
				return nil, false
			}
			return []any{int32(id), r.Lang("name"), r.Bool("published"), r.Int("iconID")}, true
		},
	},
	{
		Member:      "metaGroups",
		Name:        "inv_meta_groups",
		PK:          []string{"meta_group_id"},
		PruneAbsent: true,
		Columns:     []string{"meta_group_id", "name", "icon_id", "icon_suffix"},
		Values: func(r Row) ([]any, bool) {
			id, ok := r.Key()
			if !ok {
				return nil, false
			}
			return []any{int32(id), r.Lang("name"), r.Int("iconID"), r.Str("iconSuffix")}, true
		},
	},
	{
		Member:  "marketGroups",
		Name:    "inv_market_groups",
		PK:      []string{"market_group_id"},
		Columns: []string{"market_group_id", "parent_group_id", "name", "description", "icon_id", "has_types"},
		Values: func(r Row) ([]any, bool) {
			id, ok := r.Key()
			if !ok {
				return nil, false
			}
			return []any{
				int32(id), r.Int("parentGroupID"), r.Lang("name"),
				r.Lang("description"), r.Int("iconID"), r.Bool("hasTypes"),
			}, true
		},
	},
	{
		Member:      "mapRegions",
		Name:        "regions",
		PK:          []string{"region_id"},
		PruneAbsent: true,
		// faction_id, nebula_id and wormhole_class_id are present in the archive
		// and have columns here, but the TypeScript importer never wrote them
		// (0 of 114 rows populated in production). Filling them is a deliberate
		// improvement, and additive — readers that expected NULL still work.
		Columns: []string{
			"region_id", "name", "description", "center_x", "center_y", "center_z",
			"faction_id", "nebula_id", "wormhole_class_id",
		},
		Values: func(r Row) ([]any, bool) {
			id, ok := r.Key()
			if !ok {
				return nil, false
			}
			x, y, z := xyz(r, "position")
			return []any{
				int32(id), r.Lang("name"), r.Lang("description"), x, y, z,
				r.Int("factionID"), r.Int("nebulaID"), r.Int("wormholeClassID"),
			}, true
		},
	},
	{
		Member:      "mapConstellations",
		Name:        "constellations",
		PK:          []string{"constellation_id"},
		PruneAbsent: true,
		Columns:     []string{"constellation_id", "constellation_name", "region_id", "x", "y", "z", "faction_id", "radius"},
		Values: func(r Row) ([]any, bool) {
			id, ok := r.Key()
			if !ok {
				return nil, false
			}
			x, y, z := xyz(r, "position")
			return []any{
				int32(id), r.Lang("name"), r.Int("regionID"),
				x, y, z, r.Int("factionID"), r.Float("radius"),
			}, true
		},
	},
	{
		Member:      "mapSolarSystems",
		Name:        "solar_systems",
		PK:          []string{"solar_system_id"},
		PruneAbsent: true,
		Columns: []string{
			"solar_system_id", "system_name", "constellation_id", "region_id",
			"x", "y", "z", "x2d", "z2d", "security", "security_class", "faction_id",
			"border", "fringe", "corridor", "hub", "international", "regional",
			"luminosity", "radius", "sun_type_id",
		},
		Values: func(r Row) ([]any, bool) {
			id, ok := r.Key()
			if !ok {
				return nil, false
			}
			x, y, z := xyz(r, "position")
			// The 2D map projection stores the source y as z2d — it is a
			// top-down projection, so the vertical axis is dropped and what the
			// SDE calls y is the second screen axis.
			var x2d, z2d *float64
			if p := r.Map("position2D"); p != nil {
				x2d = Row(p).Float("x")
				z2d = Row(p).Float("y")
			}
			return []any{
				int32(id), r.Lang("name"), r.Int("constellationID"), r.Int("regionID"),
				x, y, z, x2d, z2d, r.Float("securityStatus"), r.Str("securityClass"),
				r.Int("factionID"), r.Bool("border"), r.Bool("fringe"), r.Bool("corridor"),
				r.Bool("hub"), r.Bool("international"), r.Bool("regional"),
				r.Float("luminosity"), r.Float("radius"), r.Int("sunTypeID"),
			}, true
		},
	},
	{
		Member:      "factions",
		Name:        "factions",
		PK:          []string{"faction_id"},
		PruneAbsent: true,
		Columns: []string{
			"faction_id", "name", "description", "corporation_id",
			"militia_corporation_id", "solar_system_id", "icon_id", "size_factor",
			// station_count and station_system_count exist on the table but not
			// in the archive (verified: no occurrences in factions.jsonl), so
			// they are left alone rather than written as NULL.
			"race_ids",
		},
		Values: func(r Row) ([]any, bool) {
			id, ok := r.Key()
			if !ok {
				return nil, false
			}
			return []any{
				int32(id), r.Lang("name"), r.Lang("description"),
				r.Int("corporationID"), r.Int("militiaCorporationID"),
				r.Int("solarSystemID"), r.Int("iconID"), r.Float("sizeFactor"),
				int32Slice(r.List("memberRaces")),
			}, true
		},
	},
	{
		Member:      "bloodlines",
		Name:        "bloodlines",
		PK:          []string{"bloodline_id"},
		PruneAbsent: true,
		Columns: []string{
			"bloodline_id", "bloodline_name", "race_id", "description",
			"male_description", "female_description", "short_description",
			"short_male_description", "short_female_description", "ship_type_id",
			"corporation_id", "perception", "willpower", "charisma", "memory",
			"intelligence", "icon_id",
		},
		Values: func(r Row) ([]any, bool) {
			id, ok := r.Key()
			if !ok {
				return nil, false
			}
			return []any{
				int32(id), r.Lang("name"), r.Int("raceID"), r.Lang("description"),
				r.Lang("maleDescription"), r.Lang("femaleDescription"),
				r.Lang("shortDescription"), r.Lang("shortMaleDescription"),
				r.Lang("shortFemaleDescription"), r.Int("shipTypeID"),
				r.Int("corporationID"), r.Int("perception"), r.Int("willpower"),
				r.Int("charisma"), r.Int("memory"), r.Int("intelligence"), r.Int("iconID"),
			}, true
		},
	},
	{
		Member:      "races",
		Name:        "races",
		PK:          []string{"race_id"},
		PruneAbsent: true,
		Columns:     []string{"race_id", "race_name", "description", "short_description", "icon_id"},
		Values: func(r Row) ([]any, bool) {
			id, ok := r.Key()
			if !ok {
				return nil, false
			}
			return []any{
				int32(id), r.Lang("name"), r.Lang("description"),
				r.Lang("shortDescription"), r.Int("iconID"),
			}, true
		},
	},
	{
		Member:      "npcCorporations",
		Name:        "npc_corporations",
		PK:          []string{"corporation_id"},
		PruneAbsent: true,
		Columns: []string{
			"corporation_id", "name", "description", "ticker_name", "ceo_id",
			"station_id", "size", "extent", "tax_rate", "deleted",
		},
		Values: func(r Row) ([]any, bool) {
			id, ok := r.Key()
			if !ok {
				return nil, false
			}
			return []any{
				int32(id), r.Lang("name"), r.Lang("description"), r.Str("tickerName"),
				r.Int("ceoID"), r.Int("stationID"), r.Str("size"), r.Str("extent"),
				r.Float("taxRate"), r.Bool("deleted"),
			}, true
		},
	},
	{
		Member:      "stationOperations",
		Name:        "station_operations",
		PK:          []string{"operation_id"},
		PruneAbsent: true,
		Columns:     []string{"operation_id", "operation_name", "description", "activity_id", "services"},
		Values: func(r Row) ([]any, bool) {
			id, ok := r.Key()
			if !ok {
				return nil, false
			}
			return []any{
				int32(id), r.Lang("operationName"), r.Lang("description"),
				r.Int("activityID"), int32Slice(r.List("services")),
			}, true
		},
	},
	{
		Member:      "npcStations",
		Name:        "stations",
		PK:          []string{"station_id"},
		PruneAbsent: true,
		// station_name is omitted: NPC station names are not in the archive.
		// They are composed later from the owning corporation plus the operation
		// and celestial index, which is a separate derived pass.
		// constellation_id and region_id are absent from the archive and get
		// filled from solar_systems by a derived pass. security is never
		// populated (0 of 5210 rows in production) and is not attempted.
		Columns: []string{
			"station_id", "type_id", "corporation_id", "solar_system_id",
			"operation_id", "celestial_index", "orbit_id", "orbit_index",
			"x", "y", "z", "reprocessing_efficiency", "reprocessing_stations_take",
		},
		Values: func(r Row) ([]any, bool) {
			id, ok := r.Key()
			if !ok {
				return nil, false
			}
			x, y, z := xyz(r, "position")
			return []any{
				int32(id), r.Int("typeID"), r.Int("ownerID"), r.Int("solarSystemID"),
				r.Int("operationID"), r.Int("celestialIndex"),
				r.Int("orbitID"), r.Int("orbitIndex"), x, y, z,
				r.Float("reprocessingEfficiency"), r.Float("reprocessingStationsTake"),
			}, true
		},
	},
}

// xyz pulls a coordinate triple from a nested position object. Absent positions
// stay NULL rather than collapsing to the origin, which is a real place in EVE.
func xyz(r Row, field string) (x, y, z *float64) {
	p := r.Map(field)
	if p == nil {
		return nil, nil, nil
	}
	pos := Row(p)
	return pos.Float("x"), pos.Float("y"), pos.Float("z")
}

// int32Slice converts a JSON array to []int32 for integer[] columns. Returns nil
// for an absent or empty array so the column is NULL rather than '{}' — the
// TypeScript importer omitted the field entirely in that case.
func int32Slice(list []any) []int32 {
	if len(list) == 0 {
		return nil
	}
	out := make([]int32, 0, len(list))
	for _, v := range list {
		if n := toInt64(v); n != nil && *n <= 2147483647 && *n >= -2147483648 {
			out = append(out, int32(*n))
		}
	}
	if len(out) == 0 {
		return nil
	}
	return out
}

// AllTables is every declarative import, simple and nested.
func AllTables() []Table {
	out := make([]Table, 0, len(Tables)+len(NestedTables))
	out = append(out, Tables...)
	out = append(out, NestedTables...)
	return out
}

// TableByName finds a declaration, for --only. Matching on member name is
// ambiguous for nested members (typeDogma feeds two tables, blueprints five),
// so table name wins and a member match returns the first declaration using it.
func TableByName(name string) *Table {
	all := AllTables()
	for i := range all {
		if all[i].Name == name {
			return &all[i]
		}
	}
	for i := range all {
		if all[i].Member == name {
			return &all[i]
		}
	}
	return nil
}
