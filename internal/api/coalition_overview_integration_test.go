package api

import (
	"reflect"
	"strings"
	"testing"
)

func TestCoalitionOverviewPreservesDirectoryWithoutCombat(t *testing.T) {
	ctx, pool := localAPITestPool(t, "shrike_coalition_overview_")
	for _, query := range []string{
		`INSERT INTO alliances(alliance_id,name,ticker) VALUES(11,'Alpha','A'),(12,'Beta','B'),(13,'Enemy','E')`,
		`INSERT INTO coalitions(slug,name,description) VALUES('overview-fixture','Overview Fixture','Public description')`,
		`INSERT INTO coalition_memberships(coalition_id,alliance_id) SELECT coalition_id,alliance_id FROM coalitions CROSS JOIN (VALUES(11),(12)) a(alliance_id) WHERE slug='overview-fixture'`,
		`INSERT INTO characters(character_id,name,alliance_id,deleted) VALUES(101,'Alpha Pilot',11,false),(102,'Beta Pilot',12,false),(103,'Deleted Pilot',12,true),(104,'Unknown deletion flag',11,NULL)`,
		`INSERT INTO characters(character_id,name) SELECT 1000+n,'Unaffiliated '||n FROM generate_series(1,5000)n`,
		`INSERT INTO sovereignty(system_id,alliance_id,date_added) VALUES(30000001,11,now()),(30000002,12,now())`,
		`INSERT INTO killmails(killmail_id,killmail_hash,killmail_time,solar_system_id,victim_ship_type_id,victim_alliance_id,total_value) VALUES(1,'overview',now(),30000001,587,13,100),(2,'overview',now(),30000001,587,11,40),(3,'overview',now()-interval '40 days',30000001,587,11,80)`,
		`INSERT INTO killmail_attackers(killmail_id,attacker_index,alliance_id,killmail_time) VALUES(1,0,11,now()),(1,1,12,now()),(2,0,13,now()),(3,0,11,now()-interval '40 days')`,
	} {
		if _, err := pool.Exec(ctx, query); err != nil {
			t.Fatal(err)
		}
	}
	if _, err := pool.Exec(ctx, `VACUUM ANALYZE characters`); err != nil {
		t.Fatal(err)
	}
	var plan []byte
	if err := pool.QueryRow(ctx, `EXPLAIN (ANALYZE, FORMAT JSON) `+coalitionDirectoryOverviewSQL).Scan(&plan); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(plan), `"Index Only Scan"`) || !strings.Contains(string(plan), `characters_live_alliance_members_idx`) {
		t.Fatalf("member totals did not use the covering index: %s", plan)
	}
	reader := NewHomepageReader(pool)
	overview, err := reader.CoalitionDirectoryOverview(ctx)
	if err != nil {
		t.Fatal(err)
	}
	full, err := reader.CoalitionDirectory(ctx)
	if err != nil {
		t.Fatal(err)
	}
	var fixture *CoalitionSummary
	for index := range full.Coalitions {
		coalition := &full.Coalitions[index]
		if coalition.Slug == "overview-fixture" {
			copy := *coalition
			fixture = &copy
		}
		coalition.Kills, coalition.Losses, coalition.ISKDestroyed, coalition.ISKLost = 0, 0, 0, 0
	}
	if fixture == nil || fixture.MemberCount != 3 || fixture.AllianceCount != 2 || fixture.SystemCount != 2 || fixture.Kills != 1 || fixture.Losses != 1 || fixture.ISKDestroyed != 100 || fixture.ISKLost != 40 {
		t.Fatalf("unexpected directory/unique-kill totals: %#v", fixture)
	}
	if !reflect.DeepEqual(overview, full) {
		t.Fatalf("overview differs from non-combat API directory: %#v / %#v", overview, full)
	}
}
