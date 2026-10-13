package api

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"reflect"
	"strings"
	"testing"

	"github.com/eve-kill/shrike/internal/queue"
)

func TestAdminWalletPrivacyPagingSyncAuthorizationAndPrizeRecords(t *testing.T) {
	ctx, pool := localAPITestPool(t, "shrike_admin_wallet_")
	if _, err := queue.Migrate(ctx, pool); err != nil {
		t.Fatal(err)
	}
	for _, sql := range []string{
		`INSERT INTO users(character_id,character_name,character_owner_hash,session_id,is_admin) VALUES(42,'Fixture admin','fixture','00000000-0000-4000-8000-000000000042',true),(43,'Fixture reader','fixture','00000000-0000-4000-8000-000000000043',false)`,
		`INSERT INTO user_sessions(session_id,character_id,expires_at) SELECT session_id,character_id,now()+interval '1 hour' FROM users`,
		`INSERT INTO corporations(corporation_id,name,ticker) VALUES(98779905,'Fixture corporation','FIX')`,
		`INSERT INTO corporation_wallet_tokens(corporation_id,authorized_character_id,authorized_character_name,authorized_character_owner_hash,authorized_by_admin_character_id,access_token,refresh_token,token_expiry,scopes) VALUES(98779905,42,'Fixture admin','fixture',42,'do-not-expose-fixture-access','do-not-expose-fixture-refresh',now()+interval '1 day',ARRAY['esi-wallet.read_corporation_wallets.v1'])`,
		`INSERT INTO corporation_wallet_balances(corporation_id,division,balance) VALUES(98779905,1,1234.50),(98779905,3,9876.50)`,
		`INSERT INTO corporation_wallet_journal(corporation_id,division,journal_id,date,ref_type,description,amount,balance) SELECT 98779905,n%7+1,1000+n,now(),'player_donation','Fixture journal '||n,CASE WHEN n%3=0 THEN NULL WHEN n%3=1 THEN -1.50 ELSE 2.75 END,100 FROM generate_series(1,115)n`,
		`INSERT INTO wallet_journal_references(corporation_id,division,journal_id,reference_type,reference_id,status) SELECT corporation_id,division,journal_id,'wallet','fixture',journal_id%4 FROM corporation_wallet_journal WHERE journal_id<=1004`,
		`INSERT INTO campaigns(campaign_id,name,created_by_character_id,start_time,end_time,location) VALUES('WalletPrize001','Fixture prize',42,now()-interval '10 days',now()-interval '2 days','{}')`,
		`INSERT INTO campaign_prize_pools(campaign_id,metric,winner_count,payout_percentages,status,funded_total,finalized_at) VALUES('WalletPrize001',0,3,'{70,20,10}',1,100,now())`,
		`INSERT INTO campaign_prize_results(campaign_id,rank,character_id,character_name,metric_value,secondary_value,payout_percentage,payout_amount,claimed_at) SELECT 'WalletPrize001',n,42+n,'Fixture winner '||n,100-n,10,(ARRAY[70,20,10])[n],(ARRAY[70,20,10])[n],CASE WHEN n=2 THEN NULL ELSE now() END FROM generate_series(1,3)n`,
	} {
		if _, err := pool.Exec(ctx, sql); err != nil {
			t.Fatal(err)
		}
	}
	oauth := &fakeOAuthCodeClient{}
	flows := &fakeOAuthFlowStore{items: map[string]fakeOAuthFlow{}}
	service := New(Options{DB: &campaignCreatorTestDB{pool}, PrimaryPool: pool, Auth: AuthOptions{StateSecret: "fixture-state-secret", oauth: oauth, flowStore: flows}})
	call := func(method, path, body, identity string) *httptest.ResponseRecorder {
		r := httptest.NewRequest(method, "http://localhost"+path, strings.NewReader(body))
		r.Header.Set("Content-Type", "application/json")
		r.Header.Set("Origin", "http://localhost")
		if identity != "" {
			r.AddCookie(&http.Cookie{Name: "evelogin", Value: "00000000-0000-4000-8000-0000000000" + identity})
		}
		w := httptest.NewRecorder()
		service.Site().ServeHTTP(w, r)
		return w
	}
	for _, identity := range []string{"", "43"} {
		want := 401
		if identity == "43" {
			want = 403
		}
		for _, entry := range []struct{ Method, Path string }{{"GET", "/api/admin/wallet"}, {"GET", "/api/admin/wallet/authorize"}, {"POST", "/api/admin/wallet/sync"}, {"POST", "/api/admin/campaign-prizes/WalletPrize001/1/paid"}} {
			if w := call(entry.Method, entry.Path, `{"note":"fixture"}`, identity); w.Code != want || !strings.Contains(w.Header().Get("Cache-Control"), "no-store") {
				t.Fatal("private wallet read/write exposed")
			}
		}
	}
	read := func(query string) map[string]any {
		w := call("GET", "/api/admin/wallet?"+query, "", "42")
		var data map[string]any
		if w.Code != 200 || json.Unmarshal(w.Body.Bytes(), &data) != nil || !strings.Contains(w.Header().Get("Cache-Control"), "no-store") {
			t.Fatal("wallet read failed")
		}
		if strings.Contains(w.Body.String(), "do-not-expose") {
			t.Fatal("ESI token leaked")
		}
		return data
	}
	var ids []int64
	for page := 1; page <= 3; page++ {
		data := read(fmt.Sprintf("page=%d", page))
		rows := data["journal"].([]any)
		want := 50
		if page == 3 {
			want = 15
		}
		if len(rows) != want || data["hasMore"] != (page < 3) {
			t.Fatal("wallet paging diverged")
		}
		for _, raw := range rows {
			ids = append(ids, mapInt64(raw.(map[string]any), "journal_id", 0))
		}
	}
	for index, id := range ids {
		if id != int64(1115-index) {
			t.Fatal("equal timestamp paging skipped or duplicated a journal row")
		}
	}
	data := read("division=3&page=1")
	for _, raw := range data["journal"].([]any) {
		if mapInt64(raw.(map[string]any), "division", 0) != 3 {
			t.Fatal("division filter ignored")
		}
	}
	if data["totalBalance"] != "11111.00" || len(data["walletReferences"].([]any)) != 4 || len(data["prizeSettlements"].([]any)) != 3 {
		t.Fatal("wallet metadata/reference/prize projection failed")
	}
	if w := call("GET", "/api/admin/wallet/authorize", "", "42"); w.Code != 200 || len(w.Result().Cookies()) != 1 || !reflect.DeepEqual(oauth.scopes, walletAuthorizationScopes) || len(flows.items) != 1 {
		t.Fatal("wallet authorization did not retain required scopes and bound state")
	}
	if w := call("POST", "/api/admin/wallet/sync", "", "42"); w.Code != 200 {
		t.Fatalf("sync failed: %d", w.Code)
	}
	var jobs int
	if err := pool.QueryRow(ctx, `SELECT count(*) FROM river_job WHERE queue='corporation_wallet'`).Scan(&jobs); err != nil || jobs != 1 {
		t.Fatal("sync did not queue exactly one local job")
	}
	if _, err := pool.Exec(ctx, `UPDATE corporation_wallet_tokens SET disabled=true`); err != nil {
		t.Fatal(err)
	}
	if w := call("POST", "/api/admin/wallet/sync", "", "42"); w.Code != 409 {
		t.Fatal("disabled wallet sync accepted")
	}
	if w := call("POST", "/api/admin/campaign-prizes/WalletPrize001/2/paid", `{"note":"fixture"}`, "42"); w.Code != 400 {
		t.Fatal("unclaimed prize marked paid")
	}
	if w := call("POST", "/api/admin/campaign-prizes/WalletPrize001/1/paid", `{"note":"  Fixture payment <script>plain note</script>  "}`, "42"); w.Code != 200 {
		t.Fatalf("claimed prize record failed: %d", w.Code)
	}
	var note string
	var reviewer int
	var paid bool
	if err := pool.QueryRow(ctx, `SELECT payment_note,paid_by_admin_character_id,paid_at IS NOT NULL FROM campaign_prize_results WHERE campaign_id='WalletPrize001' AND rank=1`).Scan(&note, &reviewer, &paid); err != nil || note != "Fixture payment <script>plain note</script>" || reviewer != 42 || !paid {
		t.Fatal("payment record/author/note not persisted")
	}
	if w := call("POST", "/api/admin/campaign-prizes/WalletPrize001/1/paid", `{"note":"duplicate"}`, "42"); w.Code != 400 {
		t.Fatal("paid prize could be paid again")
	}
}
