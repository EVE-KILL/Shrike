package api

import (
	"encoding/json"
	"strings"
	"testing"
)

func TestDomainKilllistPreservesScopeCursorAndBoundsEnrichment(t *testing.T) {
	ctx, pool := localAPITestPool(t, "shrike_domain_query_")
	_, err := pool.Exec(ctx, `
 INSERT INTO solar_systems(solar_system_id,system_name) VALUES(30002813,'Hek'),(30000142,'Jita');
 INSERT INTO inv_groups(group_id,name) VALUES(25,'Frigate');
 INSERT INTO inv_types(type_id,group_id,name) VALUES(587,25,'Rifter');
 INSERT INTO corporations(corporation_id,name,ticker) VALUES(42,'Scoped Corp','SC'),(43,'Other Corp','OC');
 INSERT INTO characters(character_id,name) VALUES(91,'Final Blower');
 INSERT INTO killmails(killmail_id,killmail_hash,killmail_time,solar_system_id,victim_corporation_id,victim_ship_type_id,victim_ship_group_id,is_solo)
 SELECT n,'hash','2026-10-14'::timestamptz + n*interval '1 second',30002813,43,587,25,false FROM generate_series(1,30000) n;
 UPDATE killmails SET victim_corporation_id=42,is_solo=true WHERE killmail_id BETWEEN 20001 AND 20020;
 UPDATE killmails SET solar_system_id=30000142 WHERE killmail_id=20020;
 INSERT INTO killmail_attackers(killmail_id,attacker_index,killmail_time,corporation_id,character_id,ship_type_id,final_blow)
 SELECT killmail_id,0,killmail_time,CASE WHEN killmail_id=20021 THEN 42 ELSE 43 END,91,587,true FROM killmails;
 ANALYZE killmails; ANALYZE killmail_attackers;`)
	if err != nil {
		t.Fatal(err)
	}
	scope := &domainEntityScope{Corporations: []int32{42}}
	location := int64(30002813)
	for _, tt := range []struct {
		kind  string
		after *int64
		want  []int64
	}{
		{"latest", nil, []int64{20021, 20019, 20018}},
		{"solo", nil, []int64{20019, 20018, 20017}},
		{"latest", func() *int64 { n := int64(20018); return &n }(), []int64{20017, 20016, 20015}},
	} {
		query, args := buildDomainKilllistQuery(scope, tt.kind, "solar_system_id", &location, tt.after, 2)
		rows, err := queryMaps(ctx, pool, query, args...)
		if err != nil {
			t.Fatal(err)
		}
		if len(rows) != len(tt.want) {
			t.Fatalf("wrong page length: %d", len(rows))
		}
		for index, row := range rows {
			if int64From(row["killmail_id"]) != tt.want[index] || row["final_blow_character_name"] != "Final Blower" || row["ship_name"] != "Rifter" {
				t.Fatalf("lost scope/cursor/enrichment: %v", row)
			}
		}
		var raw []byte
		if err := pool.QueryRow(ctx, "EXPLAIN (ANALYZE,FORMAT JSON) "+query, args...).Scan(&raw); err != nil {
			t.Fatal(err)
		}
		var plans []struct{ Plan domainQueryPlan }
		if err := json.Unmarshal(raw, &plans); err != nil {
			t.Fatal(err)
		}
		found := false
		var visit func(domainQueryPlan)
		visit = func(plan domainQueryPlan) {
			if plan.Relation == "killmail_attackers" && strings.Contains(plan.Filter, "final_blow") {
				found = true
				if plan.Loops > float64(len(tt.want)) {
					t.Errorf("decorated %g rows for a %d-row cursor page", plan.Loops, len(tt.want))
				}
			}
			for _, child := range plan.Children {
				visit(child)
			}
		}
		visit(plans[0].Plan)
		if !found {
			t.Fatal("did not inspect final-blow enrichment")
		}
	}
	query, args := buildDomainKilllistQuery(&domainEntityScope{}, "latest", "solar_system_id", &location, nil, 10)
	rows, err := queryMaps(ctx, pool, query, args...)
	if err != nil || len(rows) != 0 {
		t.Fatalf("empty scope widened to global data: %v %v", rows, err)
	}
}

type domainQueryPlan struct {
	Relation string            `json:"Relation Name"`
	Filter   string            `json:"Filter"`
	Loops    float64           `json:"Actual Loops"`
	Children []domainQueryPlan `json:"Plans"`
}
