package dogmadata

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"sort"

	"github.com/eve-kill/shrike/internal/sde"
	"go.yaml.in/yaml/v3"
)

type patchDocument struct {
	Attributes []map[string]any `yaml:"attributes"`
	Effects    []map[string]any `yaml:"effects"`
	TypeDogma  []map[string]any `yaml:"typeDogma"`
}

type Data struct {
	Categories      map[int32]map[string]any
	Groups          map[int32]map[string]any
	Types           map[int32]map[string]any
	MarketGroups    map[int32]map[string]any
	DogmaAttributes map[int32]map[string]any
	DogmaEffects    map[int32]map[string]any
	TypeDogma       map[int32]map[string]any
	attributeByName map[string]int32
	effectByName    map[string]int32
	typeByName      map[string]int32
	categoryByName  map[string]int32
}

type Result struct {
	Types, Attributes, Effects, TypeDogma int
	SyntheticAttributes, SyntheticEffects int
}

func Build(ctx context.Context, source *sde.Source, patches [][]byte, destination string) (Result, error) {
	data := &Data{}
	var err error
	loads := []struct {
		name string
		dest *map[int32]map[string]any
	}{
		{"categories", &data.Categories}, {"groups", &data.Groups},
		{"types", &data.Types}, {"marketGroups", &data.MarketGroups},
		{"dogmaAttributes", &data.DogmaAttributes}, {"dogmaEffects", &data.DogmaEffects},
		{"typeDogma", &data.TypeDogma},
	}
	for _, load := range loads {
		*load.dest, err = source.Collect(ctx, load.name)
		if err != nil {
			return Result{}, fmt.Errorf("load %s: %w", load.name, err)
		}
	}
	// CCP's official JSONL calls these fields `name` and
	// `effectCategoryID`; EVEShipFit's client-FSD converter expects
	// `effectName` and `effectCategory`.
	for _, effect := range data.DogmaEffects {
		if effect["effectName"] == nil {
			effect["effectName"] = effect["name"]
		}
		if effect["effectCategory"] == nil {
			effect["effectCategory"] = effect["effectCategoryID"]
		}
	}
	data.reindex()

	var documents []patchDocument
	for _, raw := range patches {
		var document patchDocument
		if err := yaml.Unmarshal(raw, &document); err != nil {
			return Result{}, fmt.Errorf("parse EVEShipFit patch: %w", err)
		}
		documents = append(documents, document)
	}
	if err := data.patchAttributes(documents); err != nil {
		return Result{}, err
	}
	if err := data.patchEffects(documents); err != nil {
		return Result{}, err
	}
	if err := data.patchTypeDogma(documents); err != nil {
		return Result{}, err
	}
	if err := data.validate(); err != nil {
		return Result{}, err
	}

	files := encode(data)
	staging := destination + ".new"
	if err := os.RemoveAll(staging); err != nil {
		return Result{}, err
	}
	if err := os.MkdirAll(staging, 0o750); err != nil {
		return Result{}, err
	}
	for name, payload := range files {
		if err := os.WriteFile(filepath.Join(staging, name), payload, 0o600); err != nil {
			return Result{}, err
		}
	}
	if err := os.RemoveAll(destination); err != nil {
		return Result{}, err
	}
	if err := os.Rename(staging, destination); err != nil {
		return Result{}, err
	}

	result := Result{Types: len(data.Types), Attributes: len(data.DogmaAttributes), Effects: len(data.DogmaEffects), TypeDogma: len(data.TypeDogma)}
	for id := range data.DogmaAttributes {
		if id < 0 {
			result.SyntheticAttributes++
		}
	}
	for id := range data.DogmaEffects {
		if id < 0 {
			result.SyntheticEffects++
		}
	}
	return result, nil
}

func (d *Data) reindex() {
	d.attributeByName = nameIndex(d.DogmaAttributes, "name")
	d.effectByName = nameIndex(d.DogmaEffects, "effectName")
	d.typeByName = nameIndex(d.Types, "name")
	d.categoryByName = nameIndex(d.Categories, "name")
}

func nameIndex(entries map[int32]map[string]any, field string) map[string]int32 {
	out := make(map[string]int32, len(entries))
	for id, entry := range entries {
		if name := text(entry[field]); name != "" {
			out[name] = id
		}
	}
	return out
}

func (d *Data) patchAttributes(docs []patchDocument) error {
	next := int32(-1)
	for _, doc := range docs {
		for _, patch := range doc.Attributes {
			created := object(patch["new"])
			if created == nil {
				continue
			}
			name := text(created["name"])
			if _, exists := d.attributeByName[name]; exists {
				return fmt.Errorf("duplicate synthetic attribute %q", name)
			}
			id := next
			if value, ok := number(created["id"]); ok {
				id = int32(value)
			}
			entry := cloneMap(patch)
			delete(entry, "new")
			entry["name"] = name
			d.DogmaAttributes[id] = entry
			d.attributeByName[name] = id
			next--
		}
	}
	return nil
}

var effectCategories = map[string]int{"passive": 0, "active": 1, "target": 2, "area": 3, "online": 4, "overload": 5, "dungeon": 6, "system": 7}
var operations = map[string]int{"preAssign": -1, "preMul": 0, "preDiv": 1, "modAdd": 2, "modSub": 3, "postMul": 4, "postDiv": 5, "postPercent": 6, "postAssign": 7}

func (d *Data) fixModifier(mod map[string]any) error {
	for source, target := range map[string]string{"modifiedAttribute": "modifiedAttributeID", "modifyingAttribute": "modifyingAttributeID"} {
		if name := text(mod[source]); name != "" {
			id, ok := d.attributeByName[name]
			if !ok {
				return fmt.Errorf("unknown attribute %q", name)
			}
			mod[target] = int(id)
			delete(mod, source)
		}
	}
	if skill := text(mod["skillType"]); skill != "" {
		if skill == "IfSkillRequired" {
			mod["skillTypeID"] = -1
		} else if id, ok := d.typeByName[skill]; ok {
			mod["skillTypeID"] = int(id)
		} else {
			return fmt.Errorf("unknown skill %q", skill)
		}
		delete(mod, "skillType")
	}
	if operation := text(mod["operation"]); operation != "" {
		value, ok := operations[operation]
		if !ok {
			return fmt.Errorf("unknown operation %q", operation)
		}
		mod["operation"] = value
	}
	return nil
}

func (d *Data) normalizeEffect(patch map[string]any) error {
	if category := text(patch["effectCategory"]); category != "" {
		value, ok := effectCategories[category]
		if !ok {
			return fmt.Errorf("unknown effect category %q", category)
		}
		patch["effectCategory"] = value
	}
	for _, modifier := range objects(patch["modifierInfo"]) {
		if err := d.fixModifier(modifier); err != nil {
			return err
		}
	}
	return nil
}

func (d *Data) patchEffects(docs []patchDocument) error {
	next := int32(-1)
	for _, doc := range docs {
		for _, original := range doc.Effects {
			patch := cloneMap(original)
			if err := d.normalizeEffect(patch); err != nil {
				return err
			}
			if created := object(patch["new"]); created != nil {
				name := text(created["name"])
				if _, exists := d.effectByName[name]; exists {
					return fmt.Errorf("duplicate synthetic effect %q", name)
				}
				delete(patch, "new")
				patch["effectName"] = name
				d.DogmaEffects[next] = patch
				d.effectByName[name] = next
				next--
				continue
			}
			targets := objects(patch["patch"])
			if len(targets) == 0 {
				continue
			}
			modifiers := objects(patch["modifierInfo"])
			delete(patch, "patch")
			delete(patch, "modifierInfo")
			for _, target := range targets {
				id, ok := d.effectByName[text(target["name"])]
				if !ok {
					continue
				}
				entry := d.DogmaEffects[id]
				if len(modifiers) > 0 {
					entry["modifierInfo"] = append(objects(entry["modifierInfo"]), cloneObjects(modifiers)...)
				}
				for key, value := range patch {
					entry[key] = value
				}
			}
		}
	}
	return nil
}

func (d *Data) patchTypeDogma(docs []patchDocument) error {
	for _, doc := range docs {
		for _, patch := range doc.TypeDogma {
			attrs := cloneObjects(objects(patch["dogmaAttributes"]))
			effects := cloneObjects(objects(patch["dogmaEffects"]))
			for _, attr := range attrs {
				name := text(attr["attribute"])
				id, ok := d.attributeByName[name]
				if !ok {
					return fmt.Errorf("unknown attribute %q", name)
				}
				attr["attributeID"] = int(id)
				delete(attr, "attribute")
			}
			for _, effect := range effects {
				name := text(effect["effect"])
				id, ok := d.effectByName[name]
				if !ok {
					return fmt.Errorf("unknown effect %q", name)
				}
				effect["effectID"] = int(id)
				delete(effect, "effect")
			}
			applied := map[int32]bool{}
			for _, target := range objects(patch["patch"]) {
				ids, err := d.matchTypes(target)
				if err != nil {
					return err
				}
				for _, id := range ids {
					if applied[id] {
						continue
					}
					applied[id] = true
					entry := d.TypeDogma[id]
					if entry == nil {
						entry = map[string]any{"dogmaAttributes": []any{}, "dogmaEffects": []any{}}
						d.TypeDogma[id] = entry
					}
					entry["dogmaAttributes"] = append(anySlice(entry["dogmaAttributes"]), mapsToAny(cloneObjects(attrs))...)
					entry["dogmaEffects"] = append(anySlice(entry["dogmaEffects"]), mapsToAny(cloneObjects(effects))...)
				}
			}
		}
	}
	return nil
}

func (d *Data) matchTypes(target map[string]any) ([]int32, error) {
	var ids []int32
	if category := text(target["category"]); category != "" {
		categoryID, ok := d.categoryByName[category]
		if !ok {
			return nil, fmt.Errorf("unknown category %q", category)
		}
		groups := map[int32]bool{}
		for id, g := range d.Groups {
			if int32num(g["categoryID"]) == categoryID {
				groups[id] = true
			}
		}
		for id, t := range d.Types {
			if groups[int32num(t["groupID"])] {
				ids = append(ids, id)
			}
		}
	} else if name := text(target["type"]); name != "" {
		if id, ok := d.typeByName[name]; ok {
			ids = []int32{id}
		}
	} else {
		return nil, fmt.Errorf("unknown type patch target")
	}
	filtered := ids[:0]
	for _, id := range ids {
		entry := d.TypeDogma[id]
		attrs := objects(entry["dogmaAttributes"])
		effects := objects(entry["dogmaEffects"])
		if !hasAll(attrs, objects(target["hasAllAttributes"]), d.attributeByName, "attributeID") {
			continue
		}
		if !hasAny(attrs, objects(target["hasAnyAttributes"]), d.attributeByName, "attributeID") {
			continue
		}
		if !hasAny(effects, objects(target["hasAnyEffects"]), d.effectByName, "effectID") {
			continue
		}
		filtered = append(filtered, id)
	}
	sort.Slice(filtered, func(i, j int) bool { return filtered[i] < filtered[j] })
	return filtered, nil
}

func hasAll(entries, required []map[string]any, index map[string]int32, field string) bool {
	if len(required) == 0 {
		return true
	}
	present := ids(entries, field)
	for _, r := range required {
		if !present[index[text(r["name"])]] {
			return false
		}
	}
	return true
}
func hasAny(entries, required []map[string]any, index map[string]int32, field string) bool {
	if len(required) == 0 {
		return true
	}
	present := ids(entries, field)
	for _, r := range required {
		if present[index[text(r["name"])]] {
			return true
		}
	}
	return false
}
func ids(entries []map[string]any, field string) map[int32]bool {
	out := map[int32]bool{}
	for _, e := range entries {
		out[int32num(e[field])] = true
	}
	return out
}

func (d *Data) validate() error {
	if len(d.DogmaAttributes) == 0 || len(d.DogmaEffects) == 0 {
		return fmt.Errorf("empty Dogma definitions")
	}
	negativeA, negativeE := 0, 0
	for id := range d.DogmaAttributes {
		if id < 0 {
			negativeA++
		}
	}
	for id := range d.DogmaEffects {
		if id < 0 {
			negativeE++
		}
	}
	if negativeA == 0 || negativeE == 0 {
		return fmt.Errorf("EVEShipFit synthetic patches were not applied")
	}
	for id, t := range d.Types {
		if _, ok := d.Groups[int32num(t["groupID"])]; !ok {
			return fmt.Errorf("type %d references missing group", id)
		}
	}
	for id, entry := range d.TypeDogma {
		if _, ok := d.Types[id]; !ok {
			return fmt.Errorf("dogma references missing type %d", id)
		}
		for _, a := range objects(entry["dogmaAttributes"]) {
			if _, ok := d.DogmaAttributes[int32num(a["attributeID"])]; !ok {
				return fmt.Errorf("type %d references missing attribute", id)
			}
		}
		for _, e := range objects(entry["dogmaEffects"]) {
			if _, ok := d.DogmaEffects[int32num(e["effectID"])]; !ok {
				return fmt.Errorf("type %d references missing effect", id)
			}
		}
	}
	return nil
}

func text(v any) string {
	if s, ok := v.(string); ok {
		return s
	}
	if m := object(v); m != nil {
		if s := text(m["en"]); s != "" {
			return s
		}
		for _, v := range m {
			if s := text(v); s != "" {
				return s
			}
		}
	}
	return ""
}
func number(v any) (int64, bool) {
	switch n := v.(type) {
	case int:
		return int64(n), true
	case int64:
		return n, true
	case uint64:
		return int64(n), true
	case float64:
		return int64(n), true
	case float32:
		return int64(n), true
	}
	return 0, false
}
func int32num(v any) int32        { n, _ := number(v); return int32(n) }
func object(v any) map[string]any { m, _ := v.(map[string]any); return m }
func objects(v any) []map[string]any {
	switch list := v.(type) {
	case []any:
		out := make([]map[string]any, 0, len(list))
		for _, v := range list {
			if m := object(v); m != nil {
				out = append(out, m)
			}
		}
		return out
	case []map[string]any:
		return list
	}
	return nil
}
func anySlice(v any) []any {
	if s, ok := v.([]any); ok {
		return s
	}
	if m, ok := v.([]map[string]any); ok {
		return mapsToAny(m)
	}
	return nil
}
func mapsToAny(v []map[string]any) []any {
	out := make([]any, len(v))
	for i := range v {
		out[i] = v[i]
	}
	return out
}
func cloneMap(in map[string]any) map[string]any {
	out := make(map[string]any, len(in))
	for k, v := range in {
		out[k] = v
	}
	return out
}
func cloneObjects(in []map[string]any) []map[string]any {
	out := make([]map[string]any, len(in))
	for i := range in {
		out[i] = cloneMap(in[i])
	}
	return out
}
