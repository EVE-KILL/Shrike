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
	"reflect"
	"testing"
	"time"

	"github.com/eve-kill/shrike/internal/migrate"
	"github.com/eve-kill/shrike/internal/testfixtures"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

func TestBattleGeneratorSharedPreviewAndTransactionsAgainstPostgres(t *testing.T) {
	dsn := os.Getenv("TEST_DATABASE_URL")
	if dsn == "" {
		t.Skip("TEST_DATABASE_URL is not set")
	}
	cfg, err := pgxpool.ParseConfig(dsn)
	if err != nil {
		t.Fatal(err)
	}
	if cfg.ConnConfig.Host != "127.0.0.1" && cfg.ConnConfig.Host != "localhost" {
		t.Fatal("generator tests require an explicit local PostgreSQL server")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()
	admin, err := pgxpool.NewWithConfig(ctx, cfg)
	if err != nil {
		t.Fatal(err)
	}
	defer admin.Close()
	var random [6]byte
	if _, err = rand.Read(random[:]); err != nil {
		t.Fatal(err)
	}
	name := "shrike_generator_test_" + hex.EncodeToString(random[:])
	if _, err = admin.Exec(ctx, "CREATE DATABASE "+name); err != nil {
		t.Fatal(err)
	}
	cfg = cfg.Copy()
	cfg.ConnConfig.Database = name
	pool, err := pgxpool.NewWithConfig(ctx, cfg)
	if err != nil {
		t.Fatal(err)
	}
	defer func() {
		pool.Close()
		clean, stop := context.WithTimeout(context.Background(), 15*time.Second)
		defer stop()
		if _, err := admin.Exec(clean, "DROP DATABASE "+name+" WITH (FORCE)"); err != nil {
			t.Errorf("cleanup generator test database: %v", err)
		}
	}()
	if err = migrate.Apply(ctx, pool); err != nil {
		t.Fatal(err)
	}
	if err = testfixtures.SeedBattleGenerator(ctx, pool); err != nil {
		t.Fatal(err)
	}
	sessions := []string{uuid.NewString(), uuid.NewString()}
	for i, session := range sessions {
		if _, err = pool.Exec(ctx, `INSERT INTO users (character_id,character_name,character_owner_hash,session_id) VALUES ($1,'Generator test','fixture',$2)`, i+1, session); err != nil {
			t.Fatal(err)
		}
		if _, err = pool.Exec(ctx, `INSERT INTO user_sessions (session_id,character_id,expires_at) VALUES ($1,$2,now()+interval '1 hour')`, session, i+1); err != nil {
			t.Fatal(err)
		}
	}
	service := New(Options{DB: &campaignCreatorTestDB{pool}})
	call := func(path string, body any, session string) *httptest.ResponseRecorder {
		encoded, _ := json.Marshal(body)
		r := httptest.NewRequest("POST", "https://eve-kill.com"+path, bytes.NewReader(encoded))
		r.Header.Set("Content-Type", "application/json")
		r.Header.Set("Origin", "https://eve-kill.com")
		if session != "" {
			r.AddCookie(&http.Cookie{Name: "evelogin", Value: session})
		}
		w := httptest.NewRecorder()
		service.Site().ServeHTTP(w, r)
		return w
	}
	window := map[string]any{"systemIds": []int{30000142}, "startTime": testfixtures.GeneratorStart, "endTime": testfixtures.GeneratorEnd}
	entities := call("/api/battle/generator/entities", window, "")
	if entities.Code != 200 {
		t.Fatalf("entities: %d %s", entities.Code, entities.Body.String())
	}
	var listed struct {
		KillCount int `json:"killCount"`
		Alliances []struct {
			ID int64 `json:"id"`
		}
		Corporations []struct {
			ID int64 `json:"id"`
		}
		Factions []struct {
			ID int64 `json:"id"`
		}
	}
	if err = json.Unmarshal(entities.Body.Bytes(), &listed); err != nil {
		t.Fatal(err)
	}
	if listed.KillCount != 2 || len(listed.Alliances) != 2 || len(listed.Corporations) != 3 || len(listed.Factions) != 2 {
		t.Fatalf("window entities = %+v", listed)
	}
	draft := map[string]any{"systemIds": []int{30000142}, "startTime": testfixtures.GeneratorStart, "endTime": testfixtures.GeneratorEnd, "sides": []any{map[string]any{"name": "Red", "entities": []any{map[string]any{"id": 70121, "type": "alliance"}}}, map[string]any{"name": "Blue", "entities": []any{map[string]any{"id": 70122, "type": "alliance"}}}}}
	preview := call("/api/battle/generator/preview", draft, "")
	if preview.Code != 200 {
		t.Fatalf("preview: %d %s", preview.Code, preview.Body.String())
	}
	var data map[string]any
	if err = json.Unmarshal(preview.Body.Bytes(), &data); err != nil {
		t.Fatal(err)
	}
	if data["kill_count"] != float64(2) || data["total_isk_destroyed"] != float64(350000000) {
		t.Fatalf("preview totals: %v", data)
	}
	encoded, _ := json.Marshal(draft)
	shared, err := service.Homepage().BattleGeneratorPreview(ctx, encoded)
	if err != nil {
		t.Fatal(err)
	}
	var direct map[string]any
	if err = json.Unmarshal(shared, &direct); err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(data, direct) {
		t.Fatalf("HTML reader and public JSON preview diverged: API=%s reader=%s", preview.Body.String(), shared)
	}
	if denied := call("/api/battle/generator/save", data, ""); denied.Code != 401 {
		t.Fatalf("anonymous save: %d", denied.Code)
	}
	saved := call("/api/battle/generator/save", data, sessions[0])
	if saved.Code != 200 {
		t.Fatalf("save: %d %s", saved.Code, saved.Body.String())
	}
	var id struct {
		ID int32 `json:"battle_id"`
	}
	_ = json.Unmarshal(saved.Body.Bytes(), &id)
	var creator, teamCount, memberCount int
	if err = pool.QueryRow(ctx, `SELECT created_by_character_id,(SELECT count(*) FROM battle_teams WHERE battle_id=$1),(SELECT count(*) FROM battle_team_members m JOIN battle_teams t ON t.id=m.battle_team_id WHERE t.battle_id=$1) FROM battles WHERE battle_id=$1`, id.ID).Scan(&creator, &teamCount, &memberCount); err != nil {
		t.Fatal(err)
	}
	if creator != 1 || teamCount != 2 || memberCount != 2 {
		t.Fatalf("saved rows: creator=%d teams=%d members=%d", creator, teamCount, memberCount)
	}
	data["battle_id"] = id.ID
	teams := data["teams"].([]any)
	teams[0], teams[1] = teams[1], teams[0]
	updated := call("/api/battle/generator/save", data, sessions[1])
	if updated.Code != 200 {
		t.Fatalf("cross-user correction: %d %s", updated.Code, updated.Body.String())
	}
	var returned struct {
		ID int32 `json:"battle_id"`
	}
	_ = json.Unmarshal(updated.Body.Bytes(), &returned)
	if returned.ID != id.ID {
		t.Fatal("correction created a second battle")
	}
	if err = pool.QueryRow(ctx, `SELECT created_by_character_id,(SELECT count(*) FROM battle_teams WHERE battle_id=$1) FROM battles WHERE battle_id=$1`, id.ID).Scan(&creator, &teamCount); err != nil {
		t.Fatal(err)
	}
	if creator != 1 || teamCount != 2 {
		t.Fatal("correction changed ownership or appended teams")
	}
	bad := map[string]any{}
	roundtrip, _ := json.Marshal(data)
	_ = json.Unmarshal(roundtrip, &bad)
	badTeams := bad["teams"].([]any)
	corp := badTeams[0].(map[string]any)["alliances"].([]any)[0].(map[string]any)["corporations"].([]any)[0]
	badTeams[1].(map[string]any)["alliances"].([]any)[0].(map[string]any)["corporations"] = []any{corp}
	if response := call("/api/battle/generator/save", bad, sessions[0]); response.Code != 400 {
		t.Fatalf("duplicate corp: %d %s", response.Code, response.Body.String())
	}
	if err = pool.QueryRow(ctx, `SELECT count(*) FROM battle_team_members m JOIN battle_teams t ON t.id=m.battle_team_id WHERE t.battle_id=$1`, id.ID).Scan(&memberCount); err != nil {
		t.Fatal(err)
	}
	if memberCount != 2 {
		t.Fatal("rejected correction damaged stored participants")
	}
	if _, err = pool.Exec(ctx, `UPDATE user_sessions SET expires_at=now()-interval '1 minute' WHERE session_id=$1`, sessions[0]); err != nil {
		t.Fatal(err)
	}
	if response := call("/api/battle/generator/save", data, sessions[0]); response.Code != 401 {
		t.Fatalf("expired session: %d", response.Code)
	}
}
