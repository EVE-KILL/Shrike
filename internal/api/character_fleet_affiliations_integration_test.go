package api

import (
	"fmt"
	"reflect"
	"testing"
	"time"

	"github.com/eve-kill/shrike/internal/killmail"
)

func TestCharacterFleetAffiliationsBackfillAndIngestion(t *testing.T) {
	ctx, pool := localAPITestPool(t, "shrike_fleet_affiliations_")
	since := time.Now().UTC().Truncate(time.Second).AddDate(0, 0, -90)
	at := since.Add(24 * time.Hour)
	const pilot int32 = 90000001
	// An older writer stored this kill before fleet projections existed.
	if _, err := pool.Exec(ctx, `INSERT INTO killmails
		(killmail_id, killmail_time, killmail_hash, solar_system_id) VALUES (100, $1, 'historical-fleet', 30000142)`, at); err != nil {
		t.Fatal(err)
	}
	if _, err := pool.Exec(ctx, `INSERT INTO killmail_attackers
		(killmail_id, killmail_time, attacker_index, character_id, corporation_id, alliance_id) VALUES
		(100, $1, 0, 90000001, 11, 21),
		(100, $1, 1, 90000001, 11, 21),
		(100, $1, 2, 90000002, 12, 22),
		(100, $1, 3, 90000003, 12, 22),
		(100, $1, 4, NULL, 13, 23)`, at); err != nil {
		t.Fatal(err)
	}
	assertAffiliations := func(want ...string) {
		t.Helper()
		rows, err := queryMaps(ctx, pool, characterFleetAffiliationsSQL, pilot, since)
		if err != nil {
			t.Fatal(err)
		}
		got := make([]string, 0, len(rows))
		for _, r := range rows {
			got = append(got, fmt.Sprintf("%d:%d:%d", r["target_type"], r["id"], r["count"]))
		}
		if !reflect.DeepEqual(got, want) {
			t.Fatalf("affiliations = %v, want %v", got, want)
		}
	}
	assertAffiliations("0:12:1", "1:22:1")
	if n, err := killmail.BackfillFleetAffiliations(ctx, pool, at, at.Add(time.Hour)); n != 1 || err != nil {
		t.Fatalf("prepared %d historical projections: %v", n, err)
	}
	if n, err := killmail.BackfillFleetAffiliations(ctx, pool, at, at.Add(time.Hour)); n != 0 || err != nil {
		t.Fatalf("retry prepared %d historical projections: %v", n, err)
	}
	assertAffiliations("0:12:1", "1:22:1")

	// Ingest a late kill with a wingmate from the viewer's own affiliation.
	other := at.Add(2 * time.Hour)
	parsed := &killmail.Parsed{
		Killmail: killmail.Killmail{KillmailID: 101, KillmailTime: other, KillmailHash: "late-fleet", SolarSystemID: 30000142},
		Attackers: []killmail.Attacker{
			{KillmailID: 101, KillmailTime: other, AttackerIndex: 0, CharacterID: pilot, CorporationID: 11, AllianceID: 21},
			{KillmailID: 101, KillmailTime: other, AttackerIndex: 1, CharacterID: 90000002, CorporationID: 11, AllianceID: 21},
			{KillmailID: 101, KillmailTime: other, AttackerIndex: 2, CharacterID: 90000003, CorporationID: 12, AllianceID: 22},
		},
	}
	if inserted, err := killmail.Insert(ctx, pool, parsed); !inserted || err != nil {
		t.Fatalf("insert late killmail: %t, %v", inserted, err)
	}
	var entries int
	if err := pool.QueryRow(ctx, `SELECT cardinality(kinds) FROM killmail_fleet_affiliations WHERE killmail_id = 101`).Scan(&entries); err != nil || entries != 4 {
		t.Fatalf("live affiliation entries = %d, err = %v", entries, err)
	}
	assertAffiliations("0:12:2", "0:11:1", "1:22:2", "1:21:1")
	if err := killmail.Delete(ctx, pool, 101); err != nil {
		t.Fatal(err)
	}
	var projectionExists bool
	if err := pool.QueryRow(ctx, `SELECT EXISTS (SELECT 1 FROM killmail_fleet_affiliations WHERE killmail_id = 101)`).Scan(&projectionExists); err != nil || projectionExists {
		t.Fatalf("deleted kill projection exists = %t, err = %v", projectionExists, err)
	}
	assertAffiliations("0:12:1", "1:22:1")
}
