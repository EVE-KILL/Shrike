// Package testfixtures holds disposable local data shared by API and browser tests.
package testfixtures

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5/pgconn"
)

type SQLWriter interface {
	Exec(context.Context, string, ...any) (pgconn.CommandTag, error)
}

const GeneratorStart = "2026-09-24T12:00:00Z"
const GeneratorEnd = "2026-09-24T13:00:00Z"

func SeedBattleGenerator(ctx context.Context, db SQLWriter) error {
	statements := []string{
		`INSERT INTO alliances (alliance_id,name,ticker) VALUES (70121,'Fixture Red Alliance','RALL'),(70122,'Fixture Blue Alliance','BALL') ON CONFLICT DO NOTHING`,
		`INSERT INTO corporations (corporation_id,name,ticker,alliance_id,palette) VALUES (70111,'Fixture Red Fleet','RFLT',70121,'{"main_color":"#ef4444","secondary_color":"#b91c1c","tertiary_color":"#fca5a5"}'),(70112,'Fixture Blue Fleet','BFLT',70122,NULL),(70113,'Fixture Independent','IND',NULL,NULL) ON CONFLICT DO NOTHING`,
		`INSERT INTO characters (character_id,name,corporation_id,alliance_id,faction_id) VALUES (70131,'Fixture Red Pilot',70111,70121,500001),(70132,'Fixture Blue Pilot',70112,70122,500004),(70133,'Fixture Independent Pilot',70113,NULL,NULL) ON CONFLICT DO NOTHING`,
		`INSERT INTO factions (faction_id,name) VALUES (500001,'Caldari State'),(500004,'Gallente Federation') ON CONFLICT DO NOTHING`,
		`INSERT INTO regions (region_id,name) VALUES (10000002,'The Forge'),(10000033,'The Citadel') ON CONFLICT DO NOTHING`,
		`INSERT INTO solar_systems (solar_system_id,system_name,region_id,security,x2d,z2d) VALUES (30000142,'Jita',10000002,0.9,0,0),(30002813,'Tama',10000033,0.3,1e15,1e15) ON CONFLICT DO NOTHING`,
		`INSERT INTO inv_groups (group_id,name) VALUES (25,'Frigate'),(420,'Destroyer') ON CONFLICT DO NOTHING`,
		`INSERT INTO inv_types (type_id,name,group_id,category_id) VALUES (587,'Rifter',25,6),(16242,'Thrasher',420,6) ON CONFLICT DO NOTHING`,
		`INSERT INTO killmails (killmail_id,killmail_hash,killmail_time,solar_system_id,region_id,victim_character_id,victim_corporation_id,victim_alliance_id,victim_faction_id,victim_ship_type_id,victim_ship_group_id,victim_damage_taken,total_value,attacker_count) VALUES
		(2000000001,'fixture-generator-1','2026-09-24T12:10:00Z',30000142,10000002,70132,70112,70122,500004,587,25,9000,150000000,3),
		(2000000002,'fixture-generator-2','2026-09-24T12:20:00Z',30000142,10000002,70131,70111,70121,500001,16242,420,10000,200000000,1),
		(2000000003,'fixture-generator-3','2026-09-24T12:40:00Z',30002813,10000033,70133,70113,NULL,NULL,587,25,8000,75000000,1),
		(2000000004,'fixture-generator-4','2026-09-24T13:10:00Z',30000142,10000002,70131,70111,70121,500001,587,25,9000,300000000,1) ON CONFLICT DO NOTHING`,
		`INSERT INTO killmail_attackers (killmail_id,attacker_index,character_id,corporation_id,alliance_id,faction_id,ship_type_id,ship_group_id,damage_done,final_blow,killmail_time) VALUES
		(2000000001,0,70131,70111,70121,500001,16242,420,8000,true,'2026-09-24T12:10:00Z'),
		(2000000001,1,70133,70113,NULL,NULL,587,25,1000,false,'2026-09-24T12:10:00Z'),
		(2000000001,2,NULL,1000125,NULL,500004,587,25,200,false,'2026-09-24T12:10:00Z'),
		(2000000002,0,70132,70112,70122,500004,587,25,10000,true,'2026-09-24T12:20:00Z'),
		(2000000003,0,70132,70112,70122,500004,587,25,8000,true,'2026-09-24T12:40:00Z'),
		(2000000004,0,70132,70112,70122,500004,587,25,9000,true,'2026-09-24T13:10:00Z') ON CONFLICT DO NOTHING`,
	}
	for i, sql := range statements {
		if _, err := db.Exec(ctx, sql); err != nil {
			return fmt.Errorf("seed generator statement %d: %w", i+1, err)
		}
	}
	return nil
}
