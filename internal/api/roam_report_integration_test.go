package api

import (
	"bytes"
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"time"

	"github.com/eve-kill/shrike/internal/migrate"
	"github.com/jackc/pgx/v5/pgxpool"
)

func TestRoamReportCreateAndLoadAgainstPostgres(t *testing.T) {
	dsn := os.Getenv("TEST_DATABASE_URL")
	if dsn == "" {
		t.Skip("TEST_DATABASE_URL is not set")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()
	admin, err := pgxpool.New(ctx, dsn)
	if err != nil {
		t.Fatal(err)
	}
	defer admin.Close()
	if err := admin.Ping(ctx); err != nil {
		t.Fatal(err)
	}
	var random [6]byte
	if _, err := rand.Read(random[:]); err != nil {
		t.Fatal(err)
	}
	dbName := "shrike_roamtest_" + hex.EncodeToString(random[:])
	if _, err := admin.Exec(ctx, `CREATE DATABASE `+dbName); err != nil {
		t.Fatal(err)
	}
	config, err := pgxpool.ParseConfig(dsn)
	if err != nil {
		t.Fatal(err)
	}
	config.ConnConfig.Database = dbName
	pool, err := pgxpool.NewWithConfig(ctx, config)
	if err != nil {
		t.Fatal(err)
	}
	defer func() {
		pool.Close()
		cleanupCtx, cleanupCancel := context.WithTimeout(context.Background(), 15*time.Second)
		defer cleanupCancel()
		if _, err := admin.Exec(cleanupCtx, `DROP DATABASE `+dbName+` WITH (FORCE)`); err != nil {
			t.Errorf("drop isolated report database: %v", err)
		}
	}()
	if err := migrate.Apply(ctx, pool); err != nil {
		t.Fatalf("apply schema: %v", err)
	}
	for _, statement := range []string{
		`INSERT INTO alliances (alliance_id, name, ticker) VALUES (20, 'Fleet Alliance', 'FLEET')`,
		`INSERT INTO corporations (corporation_id, name, ticker, alliance_id) VALUES
		    (10, 'Fleet Corp', 'FC', 20), (11, 'Target Corp', 'TC', NULL)`,
		`INSERT INTO characters (character_id, name, corporation_id, alliance_id) VALUES
		    (1, 'Pilot One', 10, 20), (2, 'Pilot Two', 10, 20), (3, 'Target Pilot', 11, NULL)`,
		`INSERT INTO regions (region_id, name) VALUES (100, 'Test Region')`,
		`INSERT INTO solar_systems (solar_system_id, system_name, region_id) VALUES (30000142, 'Test System', 100)`,
		`INSERT INTO inv_types (type_id, name) VALUES (587, 'Rifter')`,
	} {
		if _, err := pool.Exec(ctx, statement); err != nil {
			t.Fatalf("seed report fixture: %v", err)
		}
	}
	end := time.Now().UTC().Truncate(time.Second)
	start := end.Add(-3 * time.Hour)
	killTime := end.Add(-time.Hour)
	if _, err := pool.Exec(ctx, `
		INSERT INTO killmails (killmail_id, killmail_time, killmail_hash, solar_system_id,
		    region_id, victim_character_id, victim_corporation_id, victim_ship_type_id,
		    total_value, attacker_count)
		VALUES (123456, $1, 'test-hash', 30000142, 100, 3, 11, 587, 125000000, 1)`, killTime); err != nil {
		t.Fatal(err)
	}
	if _, err := pool.Exec(ctx, `
		INSERT INTO killmail_attackers (killmail_id, attacker_index, character_id,
		    corporation_id, alliance_id, damage_done, final_blow, killmail_time)
		VALUES (123456, 0, 1, 10, 20, 5000, true, $1)`, killTime); err != nil {
		t.Fatal(err)
	}

	handler := apiPathHandler(Options{Version: "test", DB: pool, Primary: pool, PrimaryPool: pool})
	body, err := json.Marshal(roamReportCreateBody{
		NamesText: "Pilot One\nPilot Two\nMissing Pilot",
		StartTime: start, EndTime: end,
	})
	if err != nil {
		t.Fatal(err)
	}
	create := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodPost, "http://example.test/tools/roam-report", bytes.NewReader(body))
	request.Header.Set("Content-Type", "application/json")
	handler.ServeHTTP(create, request)
	if create.Code != http.StatusOK {
		t.Fatalf("create report: status %d: %s", create.Code, create.Body.String())
	}
	var created struct {
		ID string `json:"id"`
	}
	if err := json.Unmarshal(create.Body.Bytes(), &created); err != nil {
		t.Fatal(err)
	}
	if len(created.ID) != 32 {
		t.Fatalf("report id = %q", created.ID)
	}
	get := httptest.NewRecorder()
	handler.ServeHTTP(get, httptest.NewRequest(http.MethodGet,
		"http://example.test/tools/roam-report/"+created.ID, nil))
	if get.Code != http.StatusOK {
		t.Fatalf("get report: status %d: %s", get.Code, get.Body.String())
	}
	var report roamReport
	if err := json.Unmarshal(get.Body.Bytes(), &report); err != nil {
		t.Fatal(err)
	}
	if report.ID != created.ID || report.Summary.Kills != 1 || report.Summary.Losses != 0 ||
		len(report.Engagements) != 1 || len(report.Unresolved) != 1 ||
		report.Engagements[0].Killmails[0].VictimShipName != "Rifter" {
		t.Fatalf("saved report = %+v", report)
	}
}
