package testfixtures

import (
	"context"
	"fmt"
)

const LocalScanNames = "Fixture Red Pilot\nFixture Second Pilot\nFixture Support Pilot\nFixture Independent Pilot\nUnresolved Fixture Pilot\nFixture Red Pilot"
const DirectionalScan = "587\tScout\tRifter\t10 km\n587\tScout Two\tRifter\t20 km\n16242\tSupport\tThrasher\t40 km\n2454\tDrone\tHobgoblin I\t5 km\n0\tUnknown\tFixture Unknown Cloud\t1 AU"

func SeedScans(ctx context.Context, db SQLWriter) error {
	if err := SeedBattleGenerator(ctx, db); err != nil {
		return err
	}
	statements := []string{
		`INSERT INTO corporations (corporation_id,name,ticker,alliance_id) VALUES (70114,'Fixture Support','SUP',70121) ON CONFLICT DO NOTHING`,
		`INSERT INTO characters (character_id,name,corporation_id,alliance_id) VALUES (70134,'Fixture Second Pilot',70111,70121),(70135,'Fixture Support Pilot',70114,70121) ON CONFLICT DO NOTHING`,
		`INSERT INTO inv_categories (category_id,name) VALUES (6,'Ship'),(18,'Drone') ON CONFLICT DO NOTHING`,
		`INSERT INTO inv_groups (group_id,name,category_id) VALUES (100,'Combat Drone',18) ON CONFLICT DO NOTHING`,
		`INSERT INTO inv_types (type_id,name,group_id,category_id) VALUES (2454,'Hobgoblin I',100,18) ON CONFLICT DO NOTHING`,
		`INSERT INTO killmails (killmail_id,killmail_hash,killmail_time,solar_system_id,region_id,victim_character_id,victim_corporation_id,victim_alliance_id,victim_ship_type_id,total_value,attacker_count)
		 SELECT 2000000200+n,'fixture-scan-'||n,now()-interval '1 hour',30000142,10000002,70132,70112,70122,587,1000,1 FROM generate_series(1,7) n ON CONFLICT DO NOTHING`,
		`INSERT INTO killmail_attackers (killmail_id,attacker_index,character_id,corporation_id,alliance_id,ship_type_id,damage_done,final_blow,killmail_time)
		 SELECT 2000000200+n,0,70131,70111,70121,587,100,true,now()-interval '1 hour' FROM generate_series(1,7) n ON CONFLICT DO NOTHING`,
	}
	for i, statement := range statements {
		if _, err := db.Exec(ctx, statement); err != nil {
			return fmt.Errorf("seed scan statement %d: %w", i+1, err)
		}
	}
	return nil
}
