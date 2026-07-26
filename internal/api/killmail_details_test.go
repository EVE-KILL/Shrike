package api

import (
	"reflect"
	"testing"
)

func TestEditorSlotGroupMatchesFrontendExtractor(t *testing.T) {
	for flag, want := range map[int64]int{
		27: 1, 34: 1, 19: 2, 26: 2, 11: 3, 18: 3,
		92: 4, 99: 4, 125: 5, 132: 5, 87: 6, 5: 0,
	} {
		if got := editorSlotGroup(flag); got != want {
			t.Errorf("flag %d = slot %d, want %d", flag, got, want)
		}
	}
}

func TestBuildKillmailEditorFitPairsChargesAndSumsDrones(t *testing.T) {
	rows := []map[string]any{
		{
			"type_id": int32(200), "flag_id": int32(27),
			"category_id": int32(7), "type_name": "Blaster",
		},
		{
			"type_id": int32(300), "flag_id": int32(27),
			"category_id": int32(8), "type_name": "Ammo",
		},
		{
			"type_id": int32(400), "flag_id": int32(87),
			"category_id": int32(18), "type_name": "Drone",
			"quantity_dropped": int64(2), "quantity_destroyed": int64(3),
		},
		{
			"type_id": int32(400), "flag_id": int32(87),
			"category_id": int32(18), "type_name": "Drone",
			"quantity_dropped": int64(1), "quantity_destroyed": int64(0),
		},
	}

	fit := buildKillmailEditorFit(100, "Hull", rows)
	modules := fit["modules"].([]editorFitModule)
	charge := int64(300)
	wantModules := []editorFitModule{{
		SlotGroup: 1, Ordinal: 0, TypeID: 200,
		Name: "Blaster", ChargeTypeID: &charge,
	}}
	if !reflect.DeepEqual(modules, wantModules) {
		t.Errorf("modules = %#v, want %#v", modules, wantModules)
	}
	drones := fit["drones"].([]editorFitDrone)
	wantDrones := []editorFitDrone{{
		TypeID: 400, Name: "Drone", Quantity: 6,
	}}
	if !reflect.DeepEqual(drones, wantDrones) {
		t.Errorf("drones = %#v, want %#v", drones, wantDrones)
	}
}

func TestBuildKillmailEditorFitKeepsStableEmptyArrays(t *testing.T) {
	fit := buildKillmailEditorFit(100, "Hull", nil)
	if modules := fit["modules"].([]editorFitModule); modules == nil {
		t.Fatal("modules is nil")
	}
	if drones := fit["drones"].([]editorFitDrone); drones == nil {
		t.Fatal("drones is nil")
	}
}
