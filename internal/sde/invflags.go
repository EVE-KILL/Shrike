package sde

import (
	"context"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// Inventory flags name the slot or bay an item occupies — "Low power slot 1",
// "Drone Bay", "Cargo". Killmails reference them constantly, since every
// destroyed and dropped item carries one.
//
// CCP does not ship them in the JSONL archive, so the list is bundled. These
// 78 entries were taken from production, which matches the hardcoded list in
// the TypeScript importer.
//
// order_id exists on the table but is NULL for every row in production, so it is
// not written here either.
type invFlag struct {
	ID   int32
	Name string
	Text string
}

var invFlags = []invFlag{
	{0, "None", "None"},
	{5, "Cargo", "Cargo"},
	{11, "LoSlot0", "Low power slot 1"},
	{12, "LoSlot1", "Low power slot 2"},
	{13, "LoSlot2", "Low power slot 3"},
	{14, "LoSlot3", "Low power slot 4"},
	{15, "LoSlot4", "Low power slot 5"},
	{16, "LoSlot5", "Low power slot 6"},
	{17, "LoSlot6", "Low power slot 7"},
	{18, "LoSlot7", "Low power slot 8"},
	{19, "MedSlot0", "Medium power slot 1"},
	{20, "MedSlot1", "Medium power slot 2"},
	{21, "MedSlot2", "Medium power slot 3"},
	{22, "MedSlot3", "Medium power slot 4"},
	{23, "MedSlot4", "Medium power slot 5"},
	{24, "MedSlot5", "Medium power slot 6"},
	{25, "MedSlot6", "Medium power slot 7"},
	{26, "MedSlot7", "Medium power slot 8"},
	{27, "HiSlot0", "High power slot 1"},
	{28, "HiSlot1", "High power slot 2"},
	{29, "HiSlot2", "High power slot 3"},
	{30, "HiSlot3", "High power slot 4"},
	{31, "HiSlot4", "High power slot 5"},
	{32, "HiSlot5", "High power slot 6"},
	{33, "HiSlot6", "High power slot 7"},
	{34, "HiSlot7", "High power slot 8"},
	{87, "DroneBay", "Drone Bay"},
	{88, "Booster", "Booster"},
	{89, "Implant", "Implant"},
	{92, "RigSlot0", "Rig power slot 1"},
	{93, "RigSlot1", "Rig power slot 2"},
	{94, "RigSlot2", "Rig power slot 3"},
	{95, "RigSlot3", "Rig power slot 4"},
	{96, "RigSlot4", "Rig power slot 5"},
	{97, "RigSlot5", "Rig power slot 6"},
	{98, "RigSlot6", "Rig power slot 7"},
	{99, "RigSlot7", "Rig power slot 8"},
	{125, "SubSystem0", "Sub system slot 0"},
	{126, "SubSystem1", "Sub system slot 1"},
	{127, "SubSystem2", "Sub system slot 2"},
	{128, "SubSystem3", "Sub system slot 3"},
	{129, "SubSystem4", "Sub system slot 4"},
	{130, "SubSystem5", "Sub system slot 5"},
	{131, "SubSystem6", "Sub system slot 6"},
	{132, "SubSystem7", "Sub system slot 7"},
	{133, "SpecializedFuelBay", "Specialized Fuel Bay"},
	{134, "SpecializedAsteroidHold", "Specialized Asteroid Hold"},
	{135, "SpecializedGasHold", "Specialized Gas Hold"},
	{136, "SpecializedMineralHold", "Specialized Mineral Hold"},
	{137, "SpecializedSalvageHold", "Specialized Salvage Hold"},
	{138, "SpecializedShipHold", "Specialized Ship Hold"},
	{143, "SpecializedAmmoHold", "Specialized Ammo Hold"},
	{148, "SpecializedCommandCenterHold", "Specialized Command Center Hold"},
	{149, "SpecializedPlanetaryCommoditiesHold", "Specialized Planetary Commodities Hold"},
	{151, "SpecializedMaterialBay", "Specialized Material Bay"},
	{155, "FleetHangar", "Fleet Hangar"},
	{158, "FighterBay", "Fighter Bay"},
	{159, "FighterTube0", "Fighter Tube 0"},
	{160, "FighterTube1", "Fighter Tube 1"},
	{161, "FighterTube2", "Fighter Tube 2"},
	{162, "FighterTube3", "Fighter Tube 3"},
	{163, "FighterTube4", "Fighter Tube 4"},
	{164, "StructureServiceSlot0", "Structure service slot 1"},
	{165, "StructureServiceSlot1", "Structure service slot 2"},
	{166, "StructureServiceSlot2", "Structure service slot 3"},
	{167, "StructureServiceSlot3", "Structure service slot 4"},
	{168, "StructureServiceSlot4", "Structure service slot 5"},
	{169, "StructureServiceSlot5", "Structure service slot 6"},
	{170, "StructureServiceSlot6", "Structure service slot 7"},
	{171, "StructureServiceSlot7", "Structure service slot 8"},
	{172, "StructureFuel", "Structure Fuel"},
	{176, "BoosterBay", "Booster Hold"},
	{177, "SubsystemBay", "Subsystem Hold"},
	{179, "FrigateEscapeBay", "Frigate escape bay Hangar"},
	{180, "StructureDeedBay", "Structure Deed Bay"},
	{181, "SpecializedIceHold", "Specialized Ice Hold"},
	{185, "ColonyResourcesHold", "Infrastructure Hold"},
	{186, "MoonMaterialBay", "Moon Material Bay"},
}

// SeedInvFlags upserts the bundled flag list.
func SeedInvFlags(ctx context.Context, pool *pgxpool.Pool) (int64, string, error) {
	start := time.Now()

	batch := &pgx.Batch{}
	for _, f := range invFlags {
		batch.Queue(`
            INSERT INTO inv_flags (flag_id, flag_name, flag_text, order_id)
            VALUES ($1, $2, $3, NULL)
            ON CONFLICT (flag_id) DO UPDATE
            SET flag_name = EXCLUDED.flag_name, flag_text = EXCLUDED.flag_text
        `, f.ID, f.Name, f.Text)
	}

	br := pool.SendBatch(ctx, batch)
	for range invFlags {
		if _, err := br.Exec(); err != nil {
			br.Close()
			return 0, "", err
		}
	}
	if err := br.Close(); err != nil {
		return 0, "", err
	}
	return int64(len(invFlags)), time.Since(start).Round(time.Millisecond).String(), nil
}
