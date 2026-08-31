package graph

import (
	"context"
	"os"
	"testing"
	"time"
)

func TestISOTimestampMatchesJavaScriptShape(t *testing.T) {
	got := isoTimestamp(time.Date(2026, 7, 26, 12, 34, 56, 789_999_999, time.FixedZone("offset", 2*60*60)))
	if got != "2026-07-26T10:34:56.789Z" {
		t.Fatalf("isoTimestamp() = %q, want JavaScript Date.toISOString shape", got)
	}
}

func TestNullableTimeUsesCanonicalISOString(t *testing.T) {
	if got := nullableTime(time.Time{}); got != nil {
		t.Fatalf("nullableTime(zero) = %#v, want nil", got)
	}
	got := nullableTime(time.Date(2026, 7, 26, 12, 34, 56, 0, time.UTC))
	if got != "2026-07-26T12:34:56.000Z" {
		t.Fatalf("nullableTime() = %#v, want fixed millisecond precision", got)
	}
}

func TestIngestIsIdempotentAndPreservesNewestAffiliation(t *testing.T) {
	url := os.Getenv("TEST_MEMGRAPH_URL")
	if url == "" {
		t.Skip("TEST_MEMGRAPH_URL is not set")
	}
	ctx := context.Background()
	client, err := Connect(ctx, url)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		_ = client.Clear(ctx)
		_ = client.Close(ctx)
	})
	if err := client.Clear(ctx); err != nil {
		t.Fatal(err)
	}
	if err := client.EnsureSchema(ctx); err != nil {
		t.Fatal(err)
	}

	newer := Killmail{
		KillmailID:    200,
		KillmailTime:  time.Date(2026, 8, 30, 12, 0, 0, 0, time.UTC),
		SolarSystemID: 30_000_001,
		Characters: []Character{
			{ID: 10, CorporationID: 100, AllianceID: 1000},
			{ID: 20, CorporationID: 200, AllianceID: 2000},
		},
		FlewWith: []FlewWithEdge{{Lo: 10, Hi: 20}},
		Killed: []KilledEdge{{
			AttackerID: 10, VictimID: 20, IskDestroyed: 42, FinalBlows: 1,
		}},
	}
	if err := client.Ingest(ctx, newer); err != nil {
		t.Fatal(err)
	}
	if err := client.Ingest(ctx, newer); err != nil {
		t.Fatal(err)
	}

	older := newer
	older.KillmailID = 199
	// ESI timestamps only have second precision. The killmail id is the stable
	// tie-breaker when two observations at the same time disagree.
	older.KillmailTime = newer.KillmailTime
	older.Characters = []Character{
		{ID: 10, CorporationID: 999, AllianceID: 9999},
		{ID: 20, CorporationID: 200, AllianceID: 2000},
	}
	if err := client.Ingest(ctx, older); err != nil {
		t.Fatal(err)
	}
	leftAlliance := Killmail{
		KillmailID:   201,
		KillmailTime: newer.KillmailTime.Add(time.Second),
		Characters: []Character{{
			ID: 30, CorporationID: 100,
		}},
	}
	if err := client.Ingest(ctx, leftAlliance); err != nil {
		t.Fatal(err)
	}

	rows, err := client.Read(ctx, `
		MATCH (a:Character {id: 10})-[f:FLEW_WITH]->(b:Character {id: 20})
		MATCH (a)-[k:KILLED]->(b)
		MATCH (corp:Corporation {id: 100})
		RETURN a.corporation_id AS corp, a.alliance_id AS alliance,
		       corp.alliance_id AS corp_alliance,
		       f.weight AS flew, k.weight AS killed, k.isk_destroyed AS isk`, nil)
	if err != nil {
		t.Fatal(err)
	}
	if len(rows) != 1 {
		t.Fatalf("relationship rows = %d, want 1", len(rows))
	}
	row := rows[0]
	if row["corp"] != int64(100) || row["alliance"] != int64(1000) {
		t.Errorf("affiliation = (%v, %v), want newest (100, 1000)", row["corp"], row["alliance"])
	}
	if row["corp_alliance"] != nil {
		t.Errorf("corporation alliance = %v, want cleared affiliation", row["corp_alliance"])
	}
	if row["flew"] != int64(2) || row["killed"] != int64(2) || row["isk"] != float64(84) {
		t.Errorf("aggregates = flew %v killed %v isk %v, want 2, 2, 84", row["flew"], row["killed"], row["isk"])
	}

	markers, err := client.Read(ctx, `MATCH (k:Killmail) RETURN count(k) AS count`, nil)
	if err != nil {
		t.Fatal(err)
	}
	if len(markers) != 1 || markers[0]["count"] != int64(3) {
		t.Fatalf("killmail markers = %#v, want count 3", markers)
	}

	indexes, err := client.Read(ctx, `SHOW INDEX INFO`, nil)
	if err != nil {
		t.Fatal(err)
	}
	if len(indexes) < 9 {
		t.Errorf("indexes = %d, want at least 9 node and edge indexes", len(indexes))
	}
	constraints, err := client.Read(ctx, `SHOW CONSTRAINT INFO`, nil)
	if err != nil {
		t.Fatal(err)
	}
	if len(constraints) < 5 {
		t.Errorf("constraints = %d, want at least 5 id uniqueness constraints", len(constraints))
	}

	stale := Killmail{
		KillmailID:   100,
		KillmailTime: time.Now().UTC().AddDate(0, 0, -RetentionDays-1),
		Characters: []Character{
			{ID: 40, CorporationID: 400, AllianceID: 4000},
			{ID: 50, CorporationID: 500, AllianceID: 5000},
		},
		FlewWith: []FlewWithEdge{{Lo: 40, Hi: 50}},
	}
	if err := client.Ingest(ctx, stale); err != nil {
		t.Fatal(err)
	}
	purged, err := client.Purge(ctx, 10)
	if err != nil {
		t.Fatal(err)
	}
	if purged.ByType["FLEW_WITH"] != 1 || purged.Killmails != 1 {
		t.Errorf("purge = %#v, want one stale edge and marker", purged)
	}
	if purged.Orphans < 2 {
		t.Errorf("purged orphans = %d, want stale nodes removed", purged.Orphans)
	}
}
