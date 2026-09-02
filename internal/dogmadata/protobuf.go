package dogmadata

import (
	"encoding/binary"
	"math"
	"sort"

	"google.golang.org/protobuf/encoding/protowire"
)

func encode(d *Data) map[string][]byte {
	return map[string][]byte{
		"categories.pb2":      encodeMap(d.Categories, encodeCategory),
		"groups.pb2":          encodeMap(d.Groups, encodeGroup),
		"types.pb2":           encodeMap(d.Types, func(v map[string]any) []byte { return encodeType(v, d) }),
		"marketGroups.pb2":    encodeMap(d.MarketGroups, encodeMarketGroup),
		"dogmaAttributes.pb2": encodeMap(d.DogmaAttributes, encodeAttribute),
		"dogmaEffects.pb2":    encodeMap(d.DogmaEffects, encodeEffect),
		"typeDogma.pb2":       encodeMap(d.TypeDogma, encodeTypeDogma),
	}
}

func encodeMap(entries map[int32]map[string]any, value func(map[string]any) []byte) []byte {
	keys := make([]int, 0, len(entries))
	for key := range entries {
		keys = append(keys, int(key))
	}
	sort.Ints(keys)
	var out []byte
	for _, raw := range keys {
		key := int32(raw)
		var item []byte
		item = i32(item, 1, key)
		item = message(item, 2, value(entries[key]))
		out = message(out, 1, item)
	}
	return out
}

func encodeCategory(v map[string]any) []byte {
	var b []byte
	b = str(b, 1, text(v["name"]))
	b = boolean(b, 2, boolValue(v["published"]))
	return b
}
func encodeGroup(v map[string]any) []byte {
	var b []byte
	b = str(b, 1, text(v["name"]))
	b = i32(b, 2, int32num(v["categoryID"]))
	b = boolean(b, 3, boolValue(v["published"]))
	return b
}
func encodeMarketGroup(v map[string]any) []byte {
	var b []byte
	b = str(b, 1, text(v["name"]))
	if present(v, "parentGroupID") {
		b = i32(b, 2, int32num(v["parentGroupID"]))
	}
	if present(v, "iconID") {
		b = i32(b, 3, int32num(v["iconID"]))
	}
	return b
}
func encodeType(v map[string]any, d *Data) []byte {
	var b []byte
	b = str(b, 1, text(v["name"]))
	group := int32num(v["groupID"])
	b = i32(b, 2, group)
	b = i32(b, 3, int32num(d.Groups[group]["categoryID"]))
	b = boolean(b, 4, boolValue(v["published"]))
	for _, f := range []struct {
		name string
		num  protowire.Number
	}{{"factionID", 5}, {"marketGroupID", 6}, {"metaGroupID", 7}} {
		if present(v, f.name) {
			b = i32(b, f.num, int32num(v[f.name]))
		}
	}
	for _, f := range []struct {
		name string
		num  protowire.Number
	}{{"capacity", 8}, {"mass", 9}, {"radius", 10}, {"volume", 11}} {
		if value, ok := floatValue(v[f.name]); ok && value != 0 {
			b = f32(b, f.num, float32(value))
		}
	}
	return b
}
func encodeAttribute(v map[string]any) []byte {
	var b []byte
	b = str(b, 1, text(v["name"]))
	b = boolean(b, 2, boolValue(v["published"]))
	value, _ := floatValue(v["defaultValue"])
	b = f32(b, 3, float32(value))
	b = boolean(b, 4, boolValue(v["highIsGood"]))
	b = boolean(b, 5, boolValue(v["stackable"]))
	return b
}
func encodeEffect(v map[string]any) []byte {
	var b []byte
	b = str(b, 1, text(v["effectName"]))
	b = i32(b, 2, int32num(v["effectCategory"]))
	for _, f := range []struct {
		name string
		num  protowire.Number
	}{{"electronicChance", 3}, {"isAssistance", 4}, {"isOffensive", 5}, {"isWarpSafe", 6}, {"propulsionChance", 7}, {"rangeChance", 8}} {
		b = boolean(b, f.num, boolValue(v[f.name]))
	}
	for _, f := range []struct {
		name string
		num  protowire.Number
	}{{"dischargeAttributeID", 9}, {"durationAttributeID", 10}, {"rangeAttributeID", 11}, {"falloffAttributeID", 12}, {"trackingSpeedAttributeID", 13}, {"fittingUsageChanceAttributeID", 14}, {"resistanceAttributeID", 15}} {
		if present(v, f.name) {
			b = i32(b, f.num, int32num(v[f.name]))
		}
	}
	for _, m := range objects(v["modifierInfo"]) {
		b = message(b, 16, encodeModifier(m))
	}
	return b
}
func encodeModifier(v map[string]any) []byte {
	var b []byte
	b = i32(b, 1, domainIDs[text(v["domain"])])
	b = i32(b, 2, funcIDs[text(v["func"])])
	for _, f := range []struct {
		name string
		num  protowire.Number
	}{{"modifiedAttributeID", 3}, {"modifyingAttributeID", 4}, {"operation", 5}, {"groupID", 6}, {"skillTypeID", 7}} {
		if present(v, f.name) {
			b = i32(b, f.num, int32num(v[f.name]))
		}
	}
	return b
}
func encodeTypeDogma(v map[string]any) []byte {
	var b []byte
	for _, a := range objects(v["dogmaAttributes"]) {
		var item []byte
		item = i32(item, 1, int32num(a["attributeID"]))
		value, _ := floatValue(a["value"])
		item = f32(item, 2, float32(value))
		b = message(b, 1, item)
	}
	for _, e := range objects(v["dogmaEffects"]) {
		var item []byte
		item = i32(item, 1, int32num(e["effectID"]))
		item = boolean(item, 2, boolValue(e["isDefault"]))
		b = message(b, 2, item)
	}
	return b
}

var domainIDs = map[string]int32{"itemID": 0, "shipID": 1, "charID": 2, "otherID": 3, "structureID": 4, "target": 5, "targetID": 6}
var funcIDs = map[string]int32{"ItemModifier": 0, "LocationGroupModifier": 1, "LocationModifier": 2, "LocationRequiredSkillModifier": 3, "OwnerRequiredSkillModifier": 4, "EffectStopper": 5}

func tag(b []byte, n protowire.Number, t protowire.Type) []byte { return protowire.AppendTag(b, n, t) }
func i32(b []byte, n protowire.Number, v int32) []byte {
	b = tag(b, n, protowire.VarintType)
	return protowire.AppendVarint(b, uint64(int64(v)))
}
func boolean(b []byte, n protowire.Number, v bool) []byte {
	if v {
		return i32(b, n, 1)
	}
	return i32(b, n, 0)
}
func str(b []byte, n protowire.Number, v string) []byte {
	b = tag(b, n, protowire.BytesType)
	return protowire.AppendString(b, v)
}
func message(b []byte, n protowire.Number, v []byte) []byte {
	b = tag(b, n, protowire.BytesType)
	return protowire.AppendBytes(b, v)
}
func f32(b []byte, n protowire.Number, v float32) []byte {
	b = tag(b, n, protowire.Fixed32Type)
	var raw [4]byte
	binary.LittleEndian.PutUint32(raw[:], math.Float32bits(v))
	return append(b, raw[:]...)
}
func present(m map[string]any, k string) bool { v, ok := m[k]; return ok && v != nil }
func boolValue(v any) bool {
	switch value := v.(type) {
	case bool:
		return value
	case int:
		return value != 0
	case float64:
		return value != 0
	}
	return false
}
func floatValue(v any) (float64, bool) {
	switch value := v.(type) {
	case float64:
		return value, true
	case float32:
		return float64(value), true
	case int:
		return float64(value), true
	case int64:
		return float64(value), true
	case uint64:
		return float64(value), true
	}
	return 0, false
}
