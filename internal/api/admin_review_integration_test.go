package api

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"testing"
)

func TestAdminReviewQueuesPrivacyTargetsPublicationAndVisibility(t *testing.T) {
	ctx, pool := localAPITestPool(t, "shrike_admin_review_")
	for _, sql := range []string{
		`INSERT INTO users(character_id,character_name,character_owner_hash,session_id,is_admin) VALUES(42,'Fixture admin','fixture','00000000-0000-4000-8000-000000000042',true),(43,'Fixture reader','fixture','00000000-0000-4000-8000-000000000043',false)`,
		`INSERT INTO user_sessions(session_id,character_id,expires_at) SELECT session_id,character_id,now()+interval '1 hour' FROM users`,
		`INSERT INTO characters(character_id,name) VALUES(43,'Fixture reader'); INSERT INTO corporations(corporation_id,name,ticker) VALUES(11,'Fixture corporation','FIX'); INSERT INTO alliances(alliance_id,name,ticker) VALUES(22,'Fixture alliance','FIX')`,
		`INSERT INTO custom_domains(id,subdomain,user_id,active) VALUES(7,'review-fixture',43,true)`,
		`INSERT INTO blog_posts(id,slug,title,status,author_id,author_name) VALUES(91,'review-article','Review article',1,42,'Fixture admin')`,
		`INSERT INTO comments(id,target_type,target_id,target_slug,domain_id,body_md,body_html,character_id,character_name,corporation_id,corporation_name,flagged,reports_count,moderation_status,visibility,deleted_at) VALUES
		 (1,1,99,NULL,NULL,'Pending kill','<p>Pending kill</p><script>alert(1)</script>',43,'Fixture reader',11,'Fixture corporation',true,3,1,1,NULL),
		 (2,9,91,NULL,NULL,'Legacy blog','<p>Legacy blog</p>',43,'Fixture reader',11,'Fixture corporation',true,1,2,2,NULL),
		 (3,3,11,NULL,7,'Domain comment','<p>Domain comment</p>',44,'Fixture other',11,'Fixture corporation',false,2,0,0,NULL),
		 (4,2,43,NULL,NULL,'Deleted','<p>Deleted</p>',43,'Fixture reader',11,'Fixture corporation',true,9,0,0,now()),
		 (5,2,43,NULL,NULL,'Published','<p>Published</p>',43,'Fixture reader',11,'Fixture corporation',false,0,0,0,NULL)`,
		`INSERT INTO moderation_queue(id,target_kind,target_id,body,body_format,rendered_html,character_id,character_name,status) VALUES
		 (11,0,1,'Pending kill','markdown','<p>Pending kill</p><script>alert(1)</script>',43,'Fixture reader',0),
		 (12,1,43,'New character bio','eve_html','<span style="color:#ff00ff;font-size:18px">New character bio</span><script>alert(1)</script>',43,'Fixture reader',0),
		 (13,2,11,'New corporation bio','markdown',NULL,43,'Fixture reader',0),
		 (14,3,22,'New alliance bio','markdown','<p>New alliance bio</p>',43,'Fixture reader',0),
		 (15,0,2,'Legacy blog','markdown','<p>Legacy blog</p>',43,'Fixture reader',4),
		 (16,0,5,'Published','markdown','<p>Published</p>',43,'Fixture reader',1)`,
	} {
		if _, err := pool.Exec(ctx, sql); err != nil {
			t.Fatal(err)
		}
	}
	service := New(Options{DB: &campaignCreatorTestDB{pool}})
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
		for _, path := range []string{"/api/admin/moderation/queue", "/api/admin/comments/queue"} {
			w := call("GET", path, "", identity)
			if w.Code != want || !strings.Contains(w.Header().Get("Cache-Control"), "no-store") {
				t.Fatal("private queue exposed")
			}
		}
		for _, path := range []string{"/api/admin/moderation/11/approve", "/api/admin/comments/1/hide", "/api/admin/comments/1/restore"} {
			if w := call("POST", path, "", identity); w.Code != want {
				t.Fatal("unauthorized moderation write")
			}
		}
	}
	comments := func(filter string) []map[string]any {
		w := call("GET", "/api/admin/comments/queue?filter="+filter+"&limit=100", "", "42")
		var data struct {
			Rows []map[string]any `json:"comments"`
		}
		if w.Code != 200 || json.Unmarshal(w.Body.Bytes(), &data) != nil || !strings.Contains(w.Header().Get("Cache-Control"), "no-store") {
			t.Fatal("comment queue failed")
		}
		return data.Rows
	}
	if len(comments("flagged")) != 2 || len(comments("reported")) != 3 || len(comments("all")) != 4 {
		t.Fatal("deleted rows or comment filters diverged")
	}
	for _, row := range comments("all") {
		if strings.Contains(stringField(row["body_html"]), "<script") {
			t.Fatal("stored script survived comment queue read")
		}
		if mapInt64(row, "id", 0) == 2 && stringField(row["target_slug"]) != "review-article" {
			t.Fatal("legacy blog target not projected")
		}
	}
	queue := func(query string) []map[string]any {
		w := call("GET", "/api/admin/moderation/queue?"+query, "", "42")
		var data struct {
			Items  []map[string]any `json:"items"`
			Counts map[string]int   `json:"counts"`
		}
		if w.Code != 200 || json.Unmarshal(w.Body.Bytes(), &data) != nil {
			t.Fatal("moderation queue failed")
		}
		for _, row := range data.Items {
			if strings.Contains(stringField(row["rendered_html"]), "<script") {
				t.Fatal("stored script survived moderation read")
			}
			if mapInt64(row, "id", 0) == 12 && !strings.Contains(stringField(row["rendered_html"]), "font-size") {
				t.Fatal("safe EVE styling lost")
			}
			if mapInt64(row, "id", 0) == 15 {
				context := row["comment_context"].(map[string]any)
				if stringField(context["target_slug"]) != "review-article" {
					t.Fatal("legacy moderation context not projected")
				}
			}
		}
		return data.Items
	}
	if len(queue("kind=all&status=pending")) != 4 || len(queue("kind=bios&status=pending")) != 3 || len(queue("kind=comments&status=all")) != 3 || len(queue("status=approved")) != 1 || len(queue("status=rejected")) != 0 {
		t.Fatal("kind/status filters changed")
	}
	for _, decision := range []struct {
		ID, Action string
		Status     int
	}{{"11", "approve", 1}, {"12", "approve", 1}, {"13", "reject", 2}, {"14", "approve", 1}} {
		if w := call("POST", "/api/admin/moderation/"+decision.ID+"/"+decision.Action, "", "42"); w.Code != 200 {
			t.Fatal("review failed")
		}
		var status, reviewer int
		id, _ := strconv.ParseInt(decision.ID, 10, 64)
		if err := pool.QueryRow(ctx, `SELECT status,reviewed_by FROM moderation_queue WHERE id=$1`, id).Scan(&status, &reviewer); err != nil || status != decision.Status || reviewer != 42 {
			t.Fatal("review metadata did not persist")
		}
	}
	var visibility, moderation int
	if err := pool.QueryRow(ctx, `SELECT visibility,moderation_status FROM comments WHERE id=1`).Scan(&visibility, &moderation); err != nil || visibility != 0 || moderation != 0 {
		t.Fatal("approved comment did not publish")
	}
	for _, entity := range []struct{ Table, ID, Body string }{{"characters", "character_id", "New character bio"}, {"alliances", "alliance_id", "New alliance bio"}} {
		var body string
		if err := pool.QueryRow(ctx, "SELECT custom_description FROM "+entity.Table+" WHERE "+entity.ID+"=$1", map[string]int{"characters": 43, "alliances": 22}[entity.Table]).Scan(&body); err != nil || body != entity.Body {
			t.Fatal("approved bio not published")
		}
	}
	if w := call("POST", "/api/admin/moderation/11/approve", "", "42"); w.Code != 409 {
		t.Fatal("already handled review not protected")
	}
	for _, action := range []string{"hide", "restore"} {
		if w := call("POST", "/api/admin/comments/3/"+action, "", "42"); w.Code != 200 {
			t.Fatal("cross-domain visibility change failed")
		}
		if err := pool.QueryRow(ctx, `SELECT visibility,moderation_status FROM comments WHERE id=3`).Scan(&visibility, &moderation); err != nil {
			t.Fatal(err)
		}
		want := 3
		if action == "restore" {
			want = 0
		}
		if visibility != want || moderation != want {
			t.Fatal("public visibility did not follow moderation state")
		}
	}
	if w := call("DELETE", "/api/comments/3", "", "42"); w.Code != 200 {
		t.Fatal("global admin could not delete domain comment")
	}
	if len(comments("all")) != 3 {
		t.Fatal("deleted domain comment retained in queue")
	}
	if w := call("POST", "/api/admin/comments/3/restore", "", "42"); w.Code != 200 || len(comments("all")) != 4 {
		t.Fatal("soft-deleted comment could not be restored")
	}
	if len(queue("status=pending")) != 0 {
		t.Fatal("reviewed rows remained pending")
	}
}
