package api

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestAdminDomainsReviewsPreviewPrivacyAndActivation(t *testing.T) {
	ctx, pool := localAPITestPool(t, "shrike_admin_domains_")
	for _, sql := range []string{
		`INSERT INTO users(character_id,character_name,character_owner_hash,session_id,is_admin) VALUES(42,'Fixture admin','fixture','00000000-0000-4000-8000-000000000042',true),(43,'Fixture owner','fixture','00000000-0000-4000-8000-000000000043',false)`,
		`INSERT INTO user_sessions(session_id,character_id,expires_at) SELECT session_id,character_id,now()+interval '1 hour' FROM users`,
		`INSERT INTO custom_domains(id,subdomain,user_id,site_name,active,entities,navbar_links) VALUES(7,'fixture-review',43,'Fixture board',true,'[{"type":"character","id":43,"name":"Fixture owner"}]','[{"label":"FAQ","url":"/faq"}]'),(8,'fixture-empty',42,NULL,false,'[]','[]')`,
	} {
		if _, err := pool.Exec(ctx, sql); err != nil {
			t.Fatal(err)
		}
	}
	body := make([]byte, 12)
	copy(body, []byte{0x89, 0x50, 0x4e, 0x47})
	hash := sha256.Sum256(body)
	storage := &domainMemoryAssetStorage{objects: map[string][]byte{}}
	for _, asset := range []struct {
		ID   int
		Type string
	}{{11, "logo"}, {12, "banner"}, {13, "background"}} {
		key := domainStorageKeyForTest(7, asset.Type, asset.ID)
		storage.objects[key] = body
		if _, err := pool.Exec(ctx, `INSERT INTO domain_assets(id,domain_id,type,status,storage_key,content_type,file_size,uploaded_by,file_hash) VALUES($1,7,$2,'pending',$3,'image/png',12,43,$4)`, asset.ID, asset.Type, key, hex.EncodeToString(hash[:])); err != nil {
			t.Fatal(err)
		}
	}
	service := New(Options{DB: &campaignCreatorTestDB{pool}, DomainAssets: storage})
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
		for _, path := range []string{"/api/admin/domains", "/api/admin/domains/7", "/api/admin/domains/7/assets/11/preview"} {
			w := call("GET", path, "", identity)
			if w.Code != want || !strings.Contains(w.Header().Get("Cache-Control"), "no-store") {
				t.Fatal("private admin domain read allowed")
			}
		}
		for _, path := range []string{"/api/admin/domains/7/toggle-active", "/api/admin/domains/7/assets/11/review"} {
			if w := call("POST", path, `{"action":"approve"}`, identity); w.Code != want {
				t.Fatal("non-admin domain mutation accepted")
			}
		}
	}
	var directory struct {
		Domains []struct {
			ID      int
			Owner   string `json:"owner_name"`
			Pending int    `json:"pending_assets"`
		}
	}
	w := call("GET", "/api/admin/domains", "", "42")
	if w.Code != 200 || json.Unmarshal(w.Body.Bytes(), &directory) != nil || len(directory.Domains) != 2 {
		t.Fatal("admin directory missing rows")
	}
	found := false
	for _, d := range directory.Domains {
		if d.ID == 7 {
			found = d.Pending == 3 && d.Owner == "Fixture owner"
		}
	}
	if !found {
		t.Fatal("pending count or owner projection diverged")
	}
	preview := call("GET", "/api/admin/domains/7/assets/11/preview", "", "42")
	if preview.Code != 200 || !bytes.Equal(preview.Body.Bytes(), body) || preview.Header().Get("Content-Type") != "image/png" || !strings.Contains(preview.Header().Get("Cache-Control"), "no-store") {
		t.Fatal("private preview metadata/bytes diverged")
	}
	if w := call("GET", "/api/admin/domains/8/assets/11/preview", "", "42"); w.Code != 404 {
		t.Fatal("asset preview ignored domain scope")
	}
	if w := call("GET", "/api/admin/domains/999", "", "42"); w.Code != 404 {
		t.Fatal("missing domain not 404")
	}
	if w := call("POST", "/api/admin/domains/7/assets/11/review", `{"action":"invalid"}`, "42"); w.Code != 400 {
		t.Fatal("invalid review accepted")
	}
	if w := call("POST", "/api/admin/domains/7/assets/11/review", `{"action":"approve","reason":"ignored"}`, "42"); w.Code != 200 {
		t.Fatal("approval failed")
	}
	if w := call("POST", "/api/admin/domains/7/assets/11/review", `{"action":"reject"}`, "42"); w.Code != 409 {
		t.Fatal("already reviewed asset changed")
	}
	if w := call("POST", "/api/admin/domains/7/assets/12/review", `{"action":"reject","reason":" <script>plain reason</script> "}`, "42"); w.Code != 200 {
		t.Fatal("rejection failed")
	}
	var status, reason, logo string
	var reviewer int
	if err := pool.QueryRow(ctx, `SELECT status,reject_reason,reviewed_by FROM domain_assets WHERE id=12`).Scan(&status, &reason, &reviewer); err != nil || status != "rejected" || reason != "<script>plain reason</script>" || reviewer != 42 {
		t.Fatal("review decision did not persist")
	}
	if err := pool.QueryRow(ctx, `SELECT theme->>'logoUrl' FROM custom_domains WHERE id=7`).Scan(&logo); err != nil || logo != "/api/domains/asset/7/logo" {
		t.Fatal("approved logo did not update domain theme")
	}
	if w := call("GET", "/api/admin/domains/7/assets/12/preview", "", "42"); w.Code != 200 {
		t.Fatal("admin lost rejected preview")
	}
	for i := 0; i < 2; i++ {
		if w := call("POST", "/api/admin/domains/7/toggle-active", "", "42"); w.Code != 200 {
			t.Fatal("activation toggle failed")
		}
		var active bool
		if err := pool.QueryRow(ctx, `SELECT active FROM custom_domains WHERE id=7`).Scan(&active); err != nil || active != (i == 1) {
			t.Fatal("domain activation did not persist")
		}
	}
	if _, err := pool.Exec(ctx, `UPDATE custom_domains SET subdomain='fixture-review' WHERE id=8`); err != nil {
		t.Fatal(err)
	}
	if w := call("POST", "/api/admin/domains/8/toggle-active", "", "42"); w.Code != 409 {
		t.Fatal("colliding domain activated")
	}
	var active bool
	if err := pool.QueryRow(ctx, `SELECT active FROM custom_domains WHERE id=8`).Scan(&active); err != nil || active {
		t.Fatal("failed activation damaged state")
	}
	if w := call("GET", "/api/admin/domains/7", "", "42"); w.Code != 200 || !strings.Contains(w.Body.String(), `"reviewed_by":42`) {
		t.Fatal("review metadata missing from detail")
	}
}
func domainStorageKeyForTest(domain int, kind string, id int) string {
	key, _ := domainAssetStorageKey(int32(domain), kind, int32(id))
	return key
}
