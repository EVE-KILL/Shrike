package api

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"testing"

	"github.com/eve-kill/shrike/internal/queue"
)

func TestAdminRiverPrivacyPagingInspectionAndLocalControls(t *testing.T) {
	ctx, pool := localAPITestPool(t, "shrike_admin_river_")
	if _, err := queue.Migrate(ctx, pool); err != nil {
		t.Fatal(err)
	}
	for _, sql := range []string{
		`INSERT INTO users(character_id,character_name,character_owner_hash,session_id,is_admin) VALUES(42,'Fixture admin','fixture','00000000-0000-4000-8000-000000000042',true),(43,'Fixture reader','fixture','00000000-0000-4000-8000-000000000043',false)`,
		`INSERT INTO user_sessions(session_id,character_id,expires_at) SELECT session_id,character_id,now()+interval '1 hour' FROM users`,
		`INSERT INTO river_queue(name,updated_at) VALUES('fixture',now()),('other',now()-interval '1 hour')`,
		`INSERT INTO river_job(id,queue,kind,state,max_attempts,attempt,finalized_at,args,metadata,errors) SELECT n,CASE WHEN n%2=0 THEN 'fixture' ELSE 'other' END,'fixture_job',CASE WHEN n=115 THEN 'running'::river_job_state ELSE 'completed'::river_job_state END,3,1,CASE WHEN n=115 THEN NULL ELSE now() END,'{"nested":{"pilots":[42,43]},"label":"<script>plain input</script>"}','{"output":{"processed":17}}',ARRAY['{"error":"<script>plain error</script>","attempt":1}'::jsonb] FROM generate_series(1,115)n`,
		`INSERT INTO river_job(id,queue,kind,max_attempts,args) VALUES(116,'fixture','fixture_job',3,'{}')`,
	} {
		if _, err := pool.Exec(ctx, sql); err != nil {
			t.Fatal(err)
		}
	}
	service := New(Options{DB: &campaignCreatorTestDB{pool}, PrimaryPool: pool})
	call := func(method, path, body, identity string) *httptest.ResponseRecorder {
		r := httptest.NewRequest(method, "http://localhost"+path, strings.NewReader(body))
		r.Header.Set("Origin", "http://localhost")
		r.Header.Set("Content-Type", "application/json")
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
		for _, path := range []string{"/api/admin/river", "/api/admin/river/jobs", "/api/admin/river/jobs/116"} {
			if w := call("GET", path, "", identity); w.Code != want || !strings.Contains(w.Header().Get("Cache-Control"), "no-store") {
				t.Fatal("River read exposed")
			}
		}
		for _, entry := range []struct{ Path, Body string }{{"/api/admin/river/jobs/116/action", `{"action":"cancel"}`}, {"/api/admin/river/queues/fixture/action", `{"action":"pause"}`}, {"/api/admin/river/queues/fixture/clear", `{"states":["completed"]}`}} {
			if w := call("POST", entry.Path, entry.Body, identity); w.Code != want {
				t.Fatal("unauthorized River control")
			}
		}
	}
	readJobs := func(query string) ([]map[string]any, int64) {
		w := call("GET", "/api/admin/river/jobs?"+query, "", "42")
		var data struct {
			Jobs []map[string]any `json:"jobs"`
			Next int64            `json:"next_before_id"`
		}
		if w.Code != 200 || json.Unmarshal(w.Body.Bytes(), &data) != nil || !strings.Contains(w.Header().Get("Cache-Control"), "no-store") {
			t.Fatal("River jobs failed")
		}
		return data.Jobs, data.Next
	}
	first, cursor := readJobs("limit=100")
	second, last := readJobs("limit=100&before_id=" + strconv.FormatInt(cursor, 10))
	if len(first) != 100 || len(second) != 16 || last != 0 {
		t.Fatalf("River paging skipped a row: %d + %d", len(first), len(second))
	}
	for i, job := range append(first, second...) {
		if mapInt64(job, "id", 0) != int64(116-i) {
			t.Fatal("exclusive cursor omitted or duplicated a job")
		}
	}
	filtered, _ := readJobs("queue=fixture&state=completed&limit=200")
	if len(filtered) != 57 {
		t.Fatal("River queue/state filter changed")
	}
	if w := call("GET", "/api/admin/river/jobs?state=invalid", "", "42"); w.Code != 400 {
		t.Fatal("invalid state accepted")
	}
	if w := call("GET", "/api/admin/river/jobs/114", "", "42"); w.Code != 200 || !strings.Contains(w.Body.String(), `"processed":17`) {
		t.Fatal("durable output missing from inspection")
	}
	for _, action := range []string{"pause", "resume"} {
		if w := call("POST", "/api/admin/river/queues/fixture/action", fmt.Sprintf(`{"action":%q}`, action), "42"); w.Code != 200 {
			t.Fatalf("queue %s failed: %d", action, w.Code)
		}
		var paused bool
		if err := pool.QueryRow(ctx, `SELECT paused_at IS NOT NULL FROM river_queue WHERE name='fixture'`).Scan(&paused); err != nil || paused != (action == "pause") {
			t.Fatal("pause/resume not persisted")
		}
	}
	for _, action := range []struct{ ID, Operation, State string }{{"116", "cancel", "cancelled"}, {"114", "retry", "available"}} {
		if w := call("POST", "/api/admin/river/jobs/"+action.ID+"/action", fmt.Sprintf(`{"action":%q}`, action.Operation), "42"); w.Code != 200 {
			t.Fatalf("job action failed: %d", w.Code)
		}
		var state string
		if err := pool.QueryRow(ctx, `SELECT state FROM river_job WHERE id=$1`, map[string]int{"116": 116, "114": 114}[action.ID]).Scan(&state); err != nil || state != action.State {
			t.Fatal("cancel/retry not persisted")
		}
	}
	if w := call("POST", "/api/admin/river/jobs/115/action", `{"action":"delete"}`, "42"); w.Code != 409 {
		t.Fatal("running job deletion allowed")
	}
	if w := call("POST", "/api/admin/river/jobs/112/action", `{"action":"delete"}`, "42"); w.Code != 200 {
		t.Fatal("finalized job deletion failed")
	}
	for _, body := range []string{`{"states":["running"]}`, `{"states":[]}`, `{"states":["completed"],"limit":10001}`} {
		if w := call("POST", "/api/admin/river/queues/fixture/clear", body, "42"); w.Code != 400 && w.Code != 422 {
			t.Fatal("unsafe clear accepted")
		}
	}
	if w := call("POST", "/api/admin/river/queues/*/clear", `{"states":["completed"]}`, "42"); w.Code != 400 {
		t.Fatal("wildcard clear accepted")
	}
	if w := call("POST", "/api/admin/river/queues/fixture/clear", `{"states":["completed","cancelled","discarded"],"limit":10000}`, "42"); w.Code != 200 {
		t.Fatal("scoped finalized-history clear failed")
	}
	var other, active int
	if err := pool.QueryRow(ctx, `SELECT count(*) FILTER(WHERE queue='other'),count(*) FILTER(WHERE queue='fixture' AND state='available') FROM river_job`).Scan(&other, &active); err != nil || other != 58 || active != 1 {
		t.Fatal("clear removed another queue or active work")
	}
}
