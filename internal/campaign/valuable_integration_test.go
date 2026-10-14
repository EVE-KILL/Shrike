package campaign

import (
	"context"
	"testing"
)

func TestCampaignMostValuableRanksBeforeNamesAndPreservesScope(t *testing.T) {
	pool := campaignTestPool(t)
	ctx := context.Background()
	tx, err := pool.Begin(ctx)
	if err != nil {
		t.Fatal(err)
	}
	defer tx.Rollback(ctx) //nolint:errcheck
	_, err = tx.Exec(ctx, `
        CREATE TEMP TABLE campaign_scratch_killmails (
            campaign_id text, killmail_id bigint, adj_value float8,
            victim_ship_type_id int, victim_character_id int,
            victim_corporation_id int, victim_side smallint,
            killmail_time timestamptz
        ) ON COMMIT DROP;
        CREATE TEMP TABLE inv_types(type_id int PRIMARY KEY,name text) ON COMMIT DROP;
        CREATE TEMP TABLE characters(character_id int PRIMARY KEY,name text) ON COMMIT DROP;
        CREATE TEMP TABLE corporations(corporation_id int PRIMARY KEY,name text) ON COMMIT DROP;
        INSERT INTO inv_types VALUES(587,'Rifter');
        INSERT INTO characters VALUES(7011,'Named Pilot');
        INSERT INTO corporations VALUES(8011,'Named Corporation');
        INSERT INTO campaign_scratch_killmails
        SELECT 'target',100+n,CASE WHEN n>=10 THEN 100 ELSE n END,
               CASE WHEN n=12 THEN 999999 ELSE 587 END,7000+n,8000+n,
               CASE WHEN n=12 THEN NULL ELSE 1::smallint END,
               TIMESTAMPTZ '2026-10-01 12:34:56.789+00'
        FROM generate_series(1,12)n;
        INSERT INTO campaign_scratch_killmails
        VALUES('other',999,1000000,587,7011,8011,1,'2026-10-01');`)
	if err != nil {
		t.Fatal(err)
	}
	blob := campaignStatsBlob{}
	if err := loadMostValuable(ctx, tx, "target", &blob); err != nil {
		t.Fatal(err)
	}
	if len(blob.MostValuable) != 10 {
		t.Fatalf("got %d valuable kills, want 10", len(blob.MostValuable))
	}
	for i, row := range blob.MostValuable {
		id := int64(112 - i)
		if row.KillmailID != id || row.KillmailTime != "2026-10-01T12:34:56.789Z" {
			t.Fatalf("valuable[%d]=%+v, want id=%d and unchanged UTC timestamp", i, row, id)
		}
		if id == 112 && (row.ShipName != nil || row.VictimCharacterName != nil || row.VictimCorporationName != nil || row.VictimSide != nil) {
			t.Fatalf("missing names or side consumed a rank: %+v", row)
		}
		if id == 111 && (row.ShipName == nil || *row.ShipName != "Rifter" || row.VictimCharacterName == nil || *row.VictimCharacterName != "Named Pilot" || row.VictimCorporationName == nil || *row.VictimCorporationName != "Named Corporation") {
			t.Fatalf("name enrichment changed: %+v", row)
		}
	}
	if err := loadMostValuable(ctx, tx, "empty", &campaignStatsBlob{}); err != nil {
		t.Fatalf("empty campaign: %v", err)
	}
}
