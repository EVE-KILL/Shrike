package api

import (
	"testing"
	"time"
)

func TestKillmailAttackerCategoryDistinguishesCorporationOwnedModules(t *testing.T) {
	ctx, pool := localAPITestPool(t, "shrike_killmail_read_")
	if _, err := pool.Exec(ctx, `INSERT INTO inv_groups(group_id,name,category_id) VALUES(25,'Frigate',6),(1656,'Structure weapon',66);
        INSERT INTO inv_types(type_id,name,group_id) VALUES(587,'Rifter',25),(35928,'Standup Arcing Vorton Projector I',1656)`); err != nil {
		t.Fatal(err)
	}
	killTime := time.Now().UTC()
	if _, err := pool.Exec(ctx, `INSERT INTO killmails(killmail_id,killmail_hash,killmail_time,solar_system_id,victim_ship_type_id) VALUES(200000099,'test',$1,30000142,587)`, killTime); err != nil {
		t.Fatal(err)
	}
	if _, err := pool.Exec(ctx, `INSERT INTO killmail_attackers(killmail_id,killmail_time,attacker_index,corporation_id,ship_type_id,damage_done,final_blow) VALUES(200000099,$1,0,77,35928,100,false),(200000099,$1,1,78,587,50,true)`, killTime); err != nil {
		t.Fatal(err)
	}
	data, found, err := NewHomepageReader(&campaignCreatorTestDB{pool}).KillmailPage(ctx, 200000099)
	if err != nil || !found {
		t.Fatalf("killmail read: found=%v error=%v", found, err)
	}
	if len(data.Attackers) != 2 || data.Attackers[0].CharacterID != nil || data.Attackers[0].ShipCategoryID == nil || *data.Attackers[0].ShipCategoryID != 66 || data.Attackers[1].ShipCategoryID == nil || *data.Attackers[1].ShipCategoryID != 6 {
		t.Fatalf("attacker categories or ordering lost: %+v", data.Attackers)
	}
}
