package sde

// Members whose payload is nested, so one archive record becomes many rows.
//
// These use Table.Expand rather than Table.Values. The row counts are wildly
// different from the record counts: typeDogma has one record per type but
// produces 645 k attribute rows, and one blueprint record produces rows across
// five tables.
//
// These are registered separately from Tables so the simple one-to-one
// mappings stay readable, but they load through exactly the same COPY-to-
// staging-then-merge path.
var NestedTables = []Table{
	{
		// {"_key": 18, "dogmaAttributes": [{"attributeID": 182, "value": 3386.0}, ...]}
		Member:  "typeDogma",
		Name:    "type_dogma_attributes",
		PK:      []string{"type_id", "attribute_id"},
		Columns: []string{"type_id", "attribute_id", "value"},
		Expand: func(r Row) [][]any {
			id, ok := r.Key()
			if !ok {
				return nil
			}
			list := r.List("dogmaAttributes")
			out := make([][]any, 0, len(list))
			for _, item := range list {
				m, ok := item.(map[string]any)
				if !ok {
					continue
				}
				a := Row(m)
				attrID := a.Int("attributeID")
				if attrID == nil {
					continue
				}
				out = append(out, []any{int32(id), *attrID, a.Float("value")})
			}
			return out
		},
	},
	{
		// Same member, different nested array.
		Member:  "typeDogma",
		Name:    "type_dogma_effects",
		PK:      []string{"type_id", "effect_id"},
		Columns: []string{"type_id", "effect_id", "is_default"},
		Expand: func(r Row) [][]any {
			id, ok := r.Key()
			if !ok {
				return nil
			}
			list := r.List("dogmaEffects")
			out := make([][]any, 0, len(list))
			for _, item := range list {
				m, ok := item.(map[string]any)
				if !ok {
					continue
				}
				e := Row(m)
				effectID := e.Int("effectID")
				if effectID == nil {
					continue
				}
				out = append(out, []any{int32(id), *effectID, e.Bool("isDefault")})
			}
			return out
		},
	},
	{
		// {"_key": 18, "materials": [{"materialTypeID": 34, "quantity": 175}, ...]}
		Member:  "typeMaterials",
		Name:    "type_materials",
		PK:      []string{"type_id", "material_type_id"},
		Columns: []string{"type_id", "material_type_id", "quantity"},
		Expand: func(r Row) [][]any {
			id, ok := r.Key()
			if !ok {
				return nil
			}
			list := r.List("materials")
			out := make([][]any, 0, len(list))
			for _, item := range list {
				m, ok := item.(map[string]any)
				if !ok {
					continue
				}
				mat := Row(m)
				matID := mat.Int("materialTypeID")
				if matID == nil {
					continue
				}
				out = append(out, []any{int32(id), *matID, mat.Int("quantity")})
			}
			return out
		},
	},
	{
		Member:  "blueprints",
		Name:    "blueprints",
		PK:      []string{"blueprint_type_id"},
		Columns: []string{"blueprint_type_id", "max_production_limit"},
		Expand: func(r Row) [][]any {
			id, ok := r.Key()
			if !ok {
				return nil
			}
			return [][]any{{int32(id), r.Int("maxProductionLimit")}}
		},
	},
	{
		// "activities" is an object keyed by activity name, not an array:
		// {"copying": {"time": 480}, "manufacturing": {"time": 600, "materials": [...]}}
		Member:  "blueprints",
		Name:    "blueprint_activities",
		PK:      []string{"blueprint_type_id", "activity"},
		Columns: []string{"blueprint_type_id", "activity", "time"},
		Expand: func(r Row) [][]any {
			id, ok := r.Key()
			if !ok {
				return nil
			}
			var out [][]any
			forEachActivity(r, func(name string, a Row) {
				out = append(out, []any{int32(id), name, a.Int("time")})
			})
			return out
		},
	},
	{
		Member:  "blueprints",
		Name:    "blueprint_activity_materials",
		PK:      []string{"blueprint_type_id", "activity", "material_type_id"},
		Columns: []string{"blueprint_type_id", "activity", "material_type_id", "quantity"},
		Expand: func(r Row) [][]any {
			id, ok := r.Key()
			if !ok {
				return nil
			}
			var out [][]any
			forEachActivity(r, func(name string, a Row) {
				for _, item := range a.List("materials") {
					m, ok := item.(map[string]any)
					if !ok {
						continue
					}
					mat := Row(m)
					// Nested entries use "typeID", not "materialTypeID" as
					// typeMaterials does for the same concept.
					typeID := mat.Int("typeID")
					if typeID == nil {
						continue
					}
					out = append(out, []any{int32(id), name, *typeID, mat.Int("quantity")})
				}
			})
			return out
		},
	},
	{
		Member:  "blueprints",
		Name:    "blueprint_activity_products",
		PK:      []string{"blueprint_type_id", "activity", "product_type_id"},
		Columns: []string{"blueprint_type_id", "activity", "product_type_id", "quantity", "probability"},
		Expand: func(r Row) [][]any {
			id, ok := r.Key()
			if !ok {
				return nil
			}
			var out [][]any
			forEachActivity(r, func(name string, a Row) {
				for _, item := range a.List("products") {
					m, ok := item.(map[string]any)
					if !ok {
						continue
					}
					p := Row(m)
					typeID := p.Int("typeID")
					if typeID == nil {
						continue
					}
					out = append(out, []any{
						int32(id), name, *typeID, p.Int("quantity"), p.Float("probability"),
					})
				}
			})
			return out
		},
	},
	{
		Member:  "blueprints",
		Name:    "blueprint_activity_skills",
		PK:      []string{"blueprint_type_id", "activity", "skill_type_id"},
		Columns: []string{"blueprint_type_id", "activity", "skill_type_id", "level"},
		Expand: func(r Row) [][]any {
			id, ok := r.Key()
			if !ok {
				return nil
			}
			var out [][]any
			forEachActivity(r, func(name string, a Row) {
				for _, item := range a.List("skills") {
					m, ok := item.(map[string]any)
					if !ok {
						continue
					}
					s := Row(m)
					typeID := s.Int("typeID")
					if typeID == nil {
						continue
					}
					out = append(out, []any{int32(id), name, *typeID, s.Int("level")})
				}
			})
			return out
		},
	},
}

// forEachActivity walks the "activities" object. Production carries six
// activity names — copying, invention, manufacturing, reaction,
// research_material, research_time — but the key is taken from the archive
// rather than validated against that list, so a new activity type imports
// rather than being silently dropped.
func forEachActivity(r Row, fn func(name string, activity Row)) {
	activities := r.Map("activities")
	if activities == nil {
		return
	}
	for name, v := range activities {
		m, ok := v.(map[string]any)
		if !ok {
			continue
		}
		fn(name, Row(m))
	}
}
