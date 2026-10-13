package api

import (
	"bytes"
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/eve-kill/shrike/internal/migrate"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

// Wrapping the pool prevents optional River dispatch in this isolated fixture;
// all HTTP authentication, validation and transaction code remains real.
type campaignCreatorTestDB struct{ *pgxpool.Pool }

func TestCampaignCreatorTransactionsAndEditingAgainstPostgres(t *testing.T) {
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
	var random [6]byte
	if _, err = rand.Read(random[:]); err != nil {
		t.Fatal(err)
	}
	name := "shrike_campaign_test_" + hex.EncodeToString(random[:])
	if _, err = admin.Exec(ctx, "CREATE DATABASE "+name); err != nil {
		t.Fatal(err)
	}
	config, err := pgxpool.ParseConfig(dsn)
	if err != nil {
		t.Fatal(err)
	}
	config.ConnConfig.Database = name
	pool, err := pgxpool.NewWithConfig(ctx, config)
	if err != nil {
		t.Fatal(err)
	}
	defer func() {
		pool.Close()
		cleanup, stop := context.WithTimeout(context.Background(), 15*time.Second)
		defer stop()
		if _, err := admin.Exec(cleanup, "DROP DATABASE "+name+" WITH (FORCE)"); err != nil {
			t.Errorf("clean campaign test database: %v", err)
		}
	}()
	if err = migrate.Apply(ctx, pool); err != nil {
		t.Fatal(err)
	}
	session1, session2 := uuid.NewString(), uuid.NewString()
	for _, sql := range []string{
		`INSERT INTO corporations (corporation_id,name,ticker) VALUES (10,'Fleet','FLT'),(11,'Opponent','OPP')`,
		`INSERT INTO characters (character_id,name,corporation_id) VALUES (1,'Creator',10),(2,'Other',11)`,
		`INSERT INTO regions (region_id,name) VALUES (100,'Battlefield')`,
		`INSERT INTO solar_systems (solar_system_id,system_name,region_id,security) VALUES (30000142,'Jita',100,0.9)`,
	} {
		if _, err = pool.Exec(ctx, sql); err != nil {
			t.Fatal(err)
		}
	}
	for index, session := range []string{session1, session2} {
		if _, err = pool.Exec(ctx, `INSERT INTO users (character_id,character_name,character_owner_hash,session_id) VALUES ($1,'Campaign test','test',$2);`, index+1, session); err != nil {
			t.Fatal(err)
		}
		if _, err = pool.Exec(ctx, `INSERT INTO user_sessions (session_id,character_id,expires_at) VALUES ($1,$2,now()+interval '1 hour')`, session, index+1); err != nil {
			t.Fatal(err)
		}
	}
	if _, err = pool.Exec(ctx, `INSERT INTO ek_wallet_accounts (character_id,balance,reserved_balance) VALUES (1,1000,200)`); err != nil {
		t.Fatal(err)
	}
	service := New(Options{DB: &campaignCreatorTestDB{pool}})
	call := func(method, path string, body any, session string) *httptest.ResponseRecorder {
		t.Helper()
		encoded, _ := json.Marshal(body)
		request := httptest.NewRequest(method, "https://eve-kill.com"+path, bytes.NewReader(encoded))
		request.Header.Set("Content-Type", "application/json")
		request.Header.Set("Origin", "https://eve-kill.com")
		if session != "" {
			request.AddCookie(&http.Cookie{Name: authSessionCookie, Value: session})
		}
		response := httptest.NewRecorder()
		service.Site().ServeHTTP(response, request)
		return response
	}
	start := time.Now().UTC().Add(-time.Hour).Truncate(time.Second)
	base := func() map[string]any {
		return map[string]any{"name": "Test campaign", "startTime": start.Format(time.RFC3339), "visibility": 1, "sides": []map[string]any{{"name": "Fleet", "entities": []map[string]any{{"type": "corporation", "id": 10}}}, {"name": "Opponent", "entities": []map[string]any{{"type": "corporation", "id": 11}}}}, "prizePool": map[string]any{"enabled": false}}
	}
	create := func(body map[string]any) string {
		t.Helper()
		response := call(http.MethodPost, "/api/campaign/create", body, session1)
		if response.Code != 200 && response.Code != 201 {
			t.Fatalf("create: %d %s", response.Code, response.Body.String())
		}
		if response.Header().Get("Cache-Control") != "private, no-store" {
			t.Fatal("account write was cacheable")
		}
		var result struct {
			ID string `json:"campaign_id"`
		}
		if err = json.Unmarshal(response.Body.Bytes(), &result); err != nil {
			t.Fatal(err)
		}
		if len(result.ID) != 14 {
			t.Fatal("invalid campaign ID")
		}
		return result.ID
	}
	if response := call(http.MethodPost, "/api/campaign/create", base(), ""); response.Code != 401 {
		t.Fatalf("anonymous write: %d", response.Code)
	}
	id := create(base())
	if response := call(http.MethodGet, "/api/campaign/"+id, nil, session2); response.Code != 404 {
		t.Fatalf("private campaign leaked to other account: %d", response.Code)
	}
	edit := map[string]any{"name": "Renamed campaign", "description": "Updated", "visibility": 1, "sides": []map[string]any{{"name": "One side", "entities": []map[string]any{{"type": "character", "id": 1}}}}}
	if response := call(http.MethodPost, "/api/campaign/"+id+"/update", edit, session2); response.Code != 403 {
		t.Fatalf("other account edited campaign: %d", response.Code)
	}
	if response := call(http.MethodPost, "/api/campaign/"+id+"/update", edit, session1); response.Code != 200 {
		t.Fatalf("owner edit: %d %s", response.Code, response.Body.String())
	}
	var sideCount int
	var renamed string
	if err = pool.QueryRow(ctx, `SELECT name,(SELECT count(*) FROM campaign_sides WHERE campaign_id=$1) FROM campaigns WHERE campaign_id=$1`, id).Scan(&renamed, &sideCount); err != nil || renamed != "Renamed campaign" || sideCount != 1 {
		t.Fatalf("edit was not persisted: %q %d %v", renamed, sideCount, err)
	}
	area := base()
	area["name"] = "Area campaign"
	area["visibility"] = 2
	area["sides"] = []any{}
	area["location"] = map[string]any{"systemIds": []int{30000142}}
	area["allowedEntities"] = []map[string]any{{"type": "corporation", "id": 11, "name": "Opponent"}}
	areaID := create(area)
	var areas, grants int
	if err = pool.QueryRow(ctx, `SELECT jsonb_array_length(location->'systemIds'),jsonb_array_length(allowed_entities) FROM campaigns WHERE campaign_id=$1`, areaID).Scan(&areas, &grants); err != nil || areas != 1 || grants != 1 {
		t.Fatalf("area/grants: %d %d %v", areas, grants, err)
	}
	prize := base()
	prize["name"] = "Funded campaign"
	prize["endTime"] = time.Now().UTC().Add(time.Hour).Format(time.RFC3339)
	requestID := uuid.NewString()
	prize["prizePool"] = map[string]any{"enabled": true, "metric": 2, "winnerCount": 3, "payoutPercentages": []int{70, 20, 10}, "initialContribution": "100.00", "fundingRequestId": requestID}
	prizeID := create(prize)
	if replayID := create(prize); replayID != prizeID {
		t.Fatalf("funded retry created another campaign: %q != %q", replayID, prizeID)
	}
	var balance string
	var locked bool
	if err = pool.QueryRow(ctx, `SELECT balance::text FROM ek_wallet_accounts WHERE character_id=1`).Scan(&balance); err != nil || balance != "900.00" {
		t.Fatalf("funded retry charged more than once: %s %v", balance, err)
	}
	if err = pool.QueryRow(ctx, `SELECT rules_locked_at IS NOT NULL FROM campaign_prize_pools WHERE campaign_id=$1`, prizeID).Scan(&locked); err != nil || !locked {
		t.Fatal("initial funding did not lock rules")
	}
	if response := call(http.MethodPost, "/api/campaign/"+prizeID+"/update", map[string]any{"endTime": time.Now().UTC().Add(2 * time.Hour).Format(time.RFC3339)}, session1); response.Code != 400 {
		t.Fatalf("funded end was editable: %d", response.Code)
	}
	if response := call(http.MethodPost, "/api/campaign/"+prizeID+"/update", edit, session1); response.Code != 400 {
		t.Fatalf("funded participants were editable: %d", response.Code)
	}
	failed := base()
	failed["name"] = "Unaffordable campaign"
	failed["endTime"] = prize["endTime"]
	failed["prizePool"] = map[string]any{"enabled": true, "metric": 0, "winnerCount": 3, "payoutPercentages": []int{70, 20, 10}, "initialContribution": "3000.00", "fundingRequestId": uuid.NewString()}
	response := call(http.MethodPost, "/api/campaign/create", failed, session1)
	if response.Code != 400 || !strings.Contains(strings.ToLower(response.Body.String()), "available funds") {
		t.Fatalf("unaffordable funding: %d %s", response.Code, response.Body.String())
	}
	var failedCount int
	if err = pool.QueryRow(ctx, `SELECT count(*) FROM campaigns WHERE name='Unaffordable campaign'`).Scan(&failedCount); err != nil || failedCount != 0 {
		t.Fatal("failed funding left a campaign behind")
	}
	if err = pool.QueryRow(ctx, `SELECT balance::text FROM ek_wallet_accounts WHERE character_id=1`).Scan(&balance); err != nil || balance != "900.00" {
		t.Fatal("failed funding charged the wallet")
	}
	contribution := map[string]any{"amount": "10.25", "requestId": uuid.NewString()}
	for attempt := range 2 {
		response := call(http.MethodPost, "/api/campaign/"+prizeID+"/prize/contribute", contribution, session1)
		if response.Code != 200 {
			t.Fatalf("contribution attempt %d: %d %s", attempt, response.Code, response.Body.String())
		}
		var result struct {
			Replayed bool `json:"replayed"`
		}
		if err = json.Unmarshal(response.Body.Bytes(), &result); err != nil || result.Replayed != (attempt == 1) {
			t.Fatalf("contribution retry was not idempotent: %+v %v", result, err)
		}
	}
	var funded string
	if err = pool.QueryRow(ctx, `SELECT funded_total::text FROM campaign_prize_pools WHERE campaign_id=$1`, prizeID).Scan(&funded); err != nil || funded != "110.25" {
		t.Fatalf("contribution charged more than once: %s %v", funded, err)
	}
	if err = pool.QueryRow(ctx, `SELECT balance::text FROM ek_wallet_accounts WHERE character_id=1`).Scan(&balance); err != nil || balance != "889.75" {
		t.Fatalf("contribution lost decimal precision: %s %v", balance, err)
	}
	if response := call(http.MethodPost, "/api/campaign/"+prizeID+"/prize/claim", nil, session1); response.Code != 400 {
		t.Fatalf("unfinalized prize was claimable: %d", response.Code)
	}
	if _, err = pool.Exec(ctx, `UPDATE campaign_prize_pools SET status=1,finalized_at=now() WHERE campaign_id=$1`, prizeID); err != nil {
		t.Fatal(err)
	}
	if _, err = pool.Exec(ctx, `INSERT INTO campaign_prize_results (campaign_id,rank,character_id,character_name,metric_value,secondary_value,payout_percentage,payout_amount) VALUES ($1,1,1,'Creator',10,1000,70,77.18)`, prizeID); err != nil {
		t.Fatal(err)
	}
	if response := call(http.MethodPost, "/api/campaign/"+prizeID+"/prize/claim", nil, session2); response.Code != 404 {
		t.Fatalf("private prize claim leaked to another user: %d", response.Code)
	}
	if response := call(http.MethodPost, "/api/campaign/"+prizeID+"/prize/claim", nil, session1); response.Code != 200 {
		t.Fatalf("winning character could not claim: %d %s", response.Code, response.Body.String())
	}
	var claimedBy int
	if err = pool.QueryRow(ctx, `SELECT claimed_by_character_id FROM campaign_prize_results WHERE campaign_id=$1 AND rank=1`, prizeID).Scan(&claimedBy); err != nil || claimedBy != 1 {
		t.Fatal("prize claim was not persisted")
	}
	if response := call(http.MethodPost, "/api/campaign/"+prizeID+"/prize/claim", nil, session1); response.Code != 400 {
		t.Fatalf("a claimed prize was claimable again: %d", response.Code)
	}
	// Exercise shared campaign comments against PostgreSQL, including the
	// report recount query. Mock database tests cannot catch parameter type
	// inference conflicts between SET and the flag comparison.
	commentResponse := call(http.MethodPost, "/api/comments", map[string]any{"target_type": 10, "target_id": 0, "target_slug": areaID, "body_md": "Comment **parity**"}, session1)
	if commentResponse.Code != 200 && commentResponse.Code != 201 {
		t.Fatalf("campaign comment: %d %s", commentResponse.Code, commentResponse.Body.String())
	}
	var commentResult struct {
		Comment map[string]any `json:"comment"`
	}
	if err = json.Unmarshal(commentResponse.Body.Bytes(), &commentResult); err != nil {
		t.Fatal(err)
	}
	commentID := mapInt64(commentResult.Comment, "id", 0)
	detailPath := fmt.Sprintf("/api/comments/%d", commentID)
	if response := call(http.MethodGet, detailPath, nil, session1); response.Code != 200 || response.Header().Get("Cache-Control") != "private, no-store" {
		t.Fatalf("authenticated Markdown detail was not fresh: %d %v", response.Code, response.Header())
	}
	if response := call(http.MethodPatch, detailPath, map[string]any{"body_md": "Updated **parity**"}, session1); response.Code != 200 {
		t.Fatalf("campaign comment edit: %d %s", response.Code, response.Body.String())
	}
	if response := call(http.MethodGet, detailPath, nil, session1); response.Code != 200 || !strings.Contains(response.Body.String(), "Updated **parity**") || response.Header().Get("Cache-Control") != "private, no-store" {
		t.Fatalf("editor read stale Markdown after an edit: %d %s", response.Code, response.Body.String())
	}
	reportPath := fmt.Sprintf("/api/comments/%d/report", commentID)
	report := map[string]any{"reason": "offtopic", "message": "Disposable integration report"}
	if response := call(http.MethodPost, reportPath, report, session1); response.Code != 400 {
		t.Fatalf("author reported own comment: %d", response.Code)
	}
	if response := call(http.MethodPost, reportPath, report, session2); response.Code != 200 {
		t.Fatalf("campaign comment report: %d %s", response.Code, response.Body.String())
	}
	if response := call(http.MethodPost, reportPath, report, session2); response.Code != 409 {
		t.Fatalf("duplicate report accepted: %d", response.Code)
	}
	commentService := newCommentService(Options{DB: &campaignCreatorTestDB{pool}})
	reportErrors := make(chan error, 2)
	for _, reporterID := range []int32{3, 4} {
		go func(id int32) {
			_, reportErr := commentService.insertCommentReport(ctx, commentResult.Comment, &Principal{CharacterID: id, CharacterName: "Report test"}, "spam", nil, time.Now().UTC())
			reportErrors <- reportErr
		}(reporterID)
	}
	for range 2 {
		if reportErr := <-reportErrors; reportErr != nil {
			t.Fatal(reportErr)
		}
	}
	var reports int
	var flagged bool
	if err = pool.QueryRow(ctx, `SELECT reports_count,flagged FROM comments WHERE id=$1`, commentID).Scan(&reports, &flagged); err != nil || reports != 3 || !flagged {
		t.Fatalf("concurrent reports lost their recount or threshold: %d %v %v", reports, flagged, err)
	}
	principal, err := service.Session().BrowserSession(ctx, session1)
	if err != nil || principal == nil || principal.CharacterID != 1 {
		t.Fatalf("SSR session: %+v %v", principal, err)
	}
	if _, err = pool.Exec(ctx, `UPDATE user_sessions SET expires_at=now()-interval '1 minute' WHERE session_id=$1`, session1); err != nil {
		t.Fatal(err)
	}
	principal, err = service.Session().BrowserSession(ctx, session1)
	if err != nil || principal != nil {
		t.Fatal("SSR accepted an expired session")
	}
}
