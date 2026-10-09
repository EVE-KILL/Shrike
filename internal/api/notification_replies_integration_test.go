package api

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"testing"
)

func TestNotificationReplyTargetsAndReadCursorIsolation(t *testing.T) {
	ctx, pool := localAPITestPool(t, "shrike_notification_replies_")
	const session = "00000000-0000-4000-8000-000000000042"
	for _, sql := range []string{
		`INSERT INTO users(character_id,character_name,character_owner_hash,session_id) VALUES(42,'Fixture Owner','fixture','00000000-0000-4000-8000-000000000042'),(43,'Fixture Reader','fixture','00000000-0000-4000-8000-000000000043')`,
		`INSERT INTO user_sessions(session_id,character_id,expires_at) VALUES('00000000-0000-4000-8000-000000000042',42,now()+interval '1 hour')`,
		`INSERT INTO blog_posts(id,slug,title,status,author_id,author_name) VALUES(91,'fixture-article','Fixture article',1,42,'Fixture Owner')`,
		`INSERT INTO custom_domains(id,subdomain,user_id) VALUES(1,'fixture-comments',42)`,
		`INSERT INTO comments(id,target_type,target_id,target_slug,character_id,character_name,corporation_id,corporation_name,body_md,body_html,domain_id) VALUES(1,1,100,NULL,42,'Fixture Owner',0,'','Root kill','<p>Root kill</p>',NULL),(2,9,91,NULL,42,'Fixture Owner',0,'','Root article','<p>Root article</p>',NULL),(3,10,0,'LocalCampaign',42,'Fixture Owner',0,'','Root campaign','<p>Root campaign</p>',NULL),(4,8,0,'LocalFit',42,'Fixture Owner',0,'','Root fitting','<p>Root fitting</p>',NULL),(5,6,0,'about',42,'Fixture Owner',0,'','Root page','<p>Root page</p>',NULL),(20,2,43,NULL,43,'Fixture Reader',0,'','Other owner','<p>Other owner</p>',NULL),(30,1,100,NULL,42,'Fixture Owner',0,'','Domain root','<p>Domain root</p>',1)`,
		`INSERT INTO comments(id,target_type,target_id,parent_id,root_id,depth,character_id,character_name,corporation_id,corporation_name,body_md,body_html) SELECT 5+id,target_type,target_id,id,id,1,43,'Fixture Reader',0,'','Reply '||id,'<p>Reply '||id||'</p>' FROM comments WHERE id BETWEEN 1 AND 5`,
	} {

		if _, err := pool.Exec(ctx, sql); err != nil {
			t.Fatal(err)
		}
	}
	for _, sql := range []string{
		`INSERT INTO comments(id,target_type,target_id,parent_id,root_id,depth,character_id,character_name,corporation_id,corporation_name,body_md,body_html,domain_id) VALUES(11,1,100,1,1,1,42,'Fixture Owner',0,'','Self reply','<p>Self reply</p>',NULL),(12,2,43,20,20,1,42,'Fixture Owner',0,'','Different recipient','<p>Different recipient</p>',NULL),(31,1,100,30,30,1,43,'Fixture Reader',0,'','Private board','<p>Private board</p>',1)`,
		`INSERT INTO comments(id,target_type,target_id,parent_id,root_id,depth,character_id,character_name,corporation_id,corporation_name,body_md,body_html,deleted_at,visibility) VALUES(13,1,100,1,1,1,43,'Fixture Reader',0,'','Deleted','<p>Deleted</p>',now(),0),(14,1,100,1,1,1,43,'Fixture Reader',0,'','Private','<p>Private</p>',NULL,1)`,
	} {
		if _, err := pool.Exec(ctx, sql); err != nil {
			t.Fatal(err)
		}
	}
	service := New(Options{DB: &campaignCreatorTestDB{pool}})
	call := func(method, path, body string, authenticated bool) *httptest.ResponseRecorder {
		t.Helper()
		r := httptest.NewRequest(method, "http://localhost"+path, strings.NewReader(body))
		r.Header.Set("Origin", "http://localhost")
		if body != "" {
			r.Header.Set("Content-Type", "application/json")
		}
		if authenticated {
			r.AddCookie(&http.Cookie{Name: "evelogin", Value: session})
		}
		w := httptest.NewRecorder()
		service.Site().ServeHTTP(w, r)
		return w
	}
	var result struct {
		Replies []accountNotificationReply `json:"replies"`
	}
	response := call("GET", "/api/notifications/replies?limit=50", "", true)
	if err := json.Unmarshal(response.Body.Bytes(), &result); err != nil {
		t.Fatal(err)
	}
	if response.Code != 200 || len(result.Replies) != 5 || response.Header().Get("Cache-Control") != "private, no-store" {
		t.Fatalf("private reply list=%d rows=%d", response.Code, len(result.Replies))
	}
	expected := map[int64]string{7: "fixture-article", 8: "LocalCampaign", 9: "LocalFit", 10: "about"}
	for _, reply := range result.Replies {
		if want, ok := expected[reply.ID]; ok && (reply.TargetSlug == nil || *reply.TargetSlug != want) {
			t.Errorf("reply %d lost its target slug", reply.ID)
		}
	}
	for _, id := range []int{10, 6} {
		w := call("POST", "/api/notifications/mark-read", `{"id":`+strconv.Itoa(id)+`}`, true)
		if w.Code != 200 {
			t.Fatalf("mark-read=%d", w.Code)
		}
	}
	var cursor int64
	if err := pool.QueryRow(ctx, "SELECT last_seen_notification_id FROM users WHERE character_id=42").Scan(&cursor); err != nil || cursor != 10 {
		t.Fatalf("read cursor regressed: %d %v", cursor, err)
	}
	response = call("GET", "/api/notifications/replies?since=8&limit=50", "", true)
	result.Replies = nil
	_ = json.Unmarshal(response.Body.Bytes(), &result)
	if response.Code != 200 || len(result.Replies) != 2 {
		t.Fatalf("since cursor=%d rows=%d", response.Code, len(result.Replies))
	}
	if w := call("GET", "/api/notifications/replies", "", false); w.Code != 401 {
		t.Fatalf("anonymous reply read=%d", w.Code)
	}
}
