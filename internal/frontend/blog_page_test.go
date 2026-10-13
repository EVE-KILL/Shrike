package frontend

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"

	"github.com/eve-kill/shrike/internal/api"
)

type fixtureBlogReader struct {
	cursors []api.BlogCursor
	tags    []string
}

func (r *fixtureBlogReader) PublicBlogPosts(_ context.Context, cursor *api.BlogCursor, tag string, limit int) (api.BlogListPage, error) {
	r.tags = append(r.tags, tag)
	if limit != 20 {
		panic("unexpected blog page size")
	}
	first := time.Date(2026, 9, 24, 12, 0, 0, 0, time.UTC)
	if cursor == nil {
		excerpt, cover := "An observed doctrine from New Eden.", "https://images.evetech.net/types/11186/render?size=256"
		return api.BlogListPage{Posts: []api.BlogListPost{{ID: 1, Slug: "doctrine-notes", Title: "Doctrine Notes", Excerpt: &excerpt, CoverImageURL: &cover, Tags: []string{"ships", "doctrines", "pvp", "weekly"}, PublishedAt: &first}}, NextCursor: first.Format("2006-01-02T15:04:05.000Z")}, nil
	}
	r.cursors = append(r.cursors, *cursor)
	return api.BlogListPage{Posts: []api.BlogListPost{{ID: 2, Slug: "second-post", Title: "Second Post", Tags: []string{"ships"}}}}, nil
}

func (r *fixtureBlogReader) PublicBlogPost(_ context.Context, slug string) (api.BlogPost, bool, error) {
	if slug != "doctrine-notes" {
		return api.BlogPost{}, false, nil
	}
	date := time.Date(2026, 9, 24, 12, 0, 0, 0, time.UTC)
	excerpt, cover, corporation := "An observed doctrine from New Eden.", "https://images.evetech.net/types/11186/render?size=256", "Example Corp"
	corporationID := int64(1234)
	return api.BlogPost{BlogListPost: api.BlogListPost{
		ID: 1, Slug: slug, Title: "Doctrine Notes", Excerpt: &excerpt,
		CoverImageURL: &cover, Tags: []string{"ships", "doctrines"}, PublishedAt: &date,
	}, BodyMD: "A doctrine.", BodyHTML: "<p>A doctrine.</p>", Status: 2,
		AuthorID: 42, AuthorName: "Example Pilot", AuthorCorporationID: &corporationID,
		AuthorCorporationName: &corporation}, true, nil
}

func TestBlogListingAndCursorFragment(t *testing.T) {
	reader := &fixtureBlogReader{}
	handler := New(Options{Blog: reader})
	page := httptest.NewRecorder()
	handler.ServeHTTP(page, httptest.NewRequest("GET", "/blog?tag=SHIPS", nil))
	if page.Code != 200 {
		t.Fatalf("blog status = %d", page.Code)
	}
	for _, marker := range []string{
		"Doctrine Notes", `href="/blog/doctrine-notes"`, "An observed doctrine from New Eden.",
		"#ships", "+1", `data-blog-more`, "blog.css", "blog.js",
		`https://eve-kill.com/blog`,
	} {
		if !strings.Contains(page.Body.String(), marker) {
			t.Errorf("blog listing missing %q", marker)
		}
	}
	if len(reader.tags) != 1 || reader.tags[0] != "ships" {
		t.Fatalf("listing tags = %v", reader.tags)
	}

	fragment := httptest.NewRecorder()
	handler.ServeHTTP(fragment, httptest.NewRequest("GET", "/_fragments/blog?cursor=2026-09-24T12%3A00%3A00.000Z&tag=ships", nil))
	if fragment.Code != 200 || !strings.Contains(fragment.Body.String(), "Second Post") || strings.Contains(fragment.Body.String(), "Doctrine Notes") {
		t.Fatalf("blog fragment status = %d, body = %q", fragment.Code, fragment.Body.String())
	}
	if fragment.Header().Get("X-Next-Cursor") != "" || len(reader.cursors) != 1 || !reader.cursors[0].PublishedAt.Equal(time.Date(2026, 9, 24, 12, 0, 0, 0, time.UTC)) {
		t.Fatalf("blog cursor not passed through: %v", reader.cursors)
	}
	if len(reader.tags) != 2 || reader.tags[1] != "ships" {
		t.Fatalf("fragment tags = %v", reader.tags)
	}
	anchor := api.BlogCursor{PublishedAt: reader.cursors[0].PublishedAt, ID: 21}
	opaque := httptest.NewRecorder()
	handler.ServeHTTP(opaque, httptest.NewRequest("GET", "/_fragments/blog?cursor="+url.QueryEscape(anchor.String())+"&tag=ships", nil))
	if opaque.Code != 200 || len(reader.cursors) != 2 || reader.cursors[1].ID != anchor.ID || !reader.cursors[1].PublishedAt.Equal(anchor.PublishedAt) {
		t.Fatalf("opaque fragment cursor lost its anchor: %v", reader.cursors)
	}

	bad := httptest.NewRecorder()
	handler.ServeHTTP(bad, httptest.NewRequest("GET", "/_fragments/blog?cursor=not-a-date", nil))
	if bad.Code != 400 {
		t.Fatalf("invalid cursor status = %d", bad.Code)
	}
}

func TestBlogPostPage(t *testing.T) {
	handler := New(Options{Blog: &fixtureBlogReader{}})
	post := httptest.NewRecorder()
	handler.ServeHTTP(post, httptest.NewRequest("GET", "/blog/doctrine-notes", nil))
	if post.Code != 200 {
		t.Fatalf("blog post status = %d", post.Code)
	}
	for _, marker := range []string{
		"Doctrine Notes", "An observed doctrine from New Eden.", "<p>A doctrine.</p>",
		`href="/blog?tag=ships"`, "Archived", `href="/character/42"`, "Example Pilot",
		`href="/corporation/1234"`, `data-target-type="9"`, `data-target-id="1"`,
		`property="og:type" content="article"`, `property="og:title" content="Doctrine Notes"`,
	} {
		if !strings.Contains(post.Body.String(), marker) {
			t.Errorf("blog post missing %q", marker)
		}
	}
	missing := httptest.NewRecorder()
	handler.ServeHTTP(missing, httptest.NewRequest("GET", "/blog/no-such-post", nil))
	if missing.Code != 404 {
		t.Fatalf("missing post status = %d", missing.Code)
	}
}

func TestBlogCoverImageRejectsUnsafeSchemes(t *testing.T) {
	for _, tc := range []struct{ raw, want string }{
		{"https://images.evetech.net/types/11186/render?size=256", "https://images.evetech.net/types/11186/render?size=256"},
		{"/images/types/11186/render", "/images/types/11186/render"},
		{"javascript:alert(1)", ""},
		{"//example.test/cover.png", ""},
		{"http://example.test/cover.png", ""},
	} {
		if got := blogCoverImage(&tc.raw); got != tc.want {
			t.Errorf("blogCoverImage(%q) = %q, want %q", tc.raw, got, tc.want)
		}
	}
}

func TestBlogEmbeddedImageOriginsAreScopedToPage(t *testing.T) {
	sources := blogBodyImageSources(`<p><img src="https://cdn.example.test/a.png"><img src="javascript:alert(1)"></p>`)
	if len(sources) != 2 {
		t.Fatalf("embedded image sources = %v", sources)
	}
	meta, err := resolveMetadata(httptest.NewRequest("GET", "http://localhost/blog/example", nil), false, 200, Metadata{ImageSources: sources})
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(meta.CSP, "img-src 'self' https://images.evetech.net https://cdn.example.test;") || strings.Contains(meta.CSP, "javascript:") {
		t.Fatalf("blog CSP image origins = %q", meta.CSP)
	}
}

func TestBlogPreviewRequiresAdminAndStaysPrivate(t *testing.T) {
	var apiPath string
	previewAPI := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		apiPath = r.URL.Path
		cookie, _ := r.Cookie("evelogin")
		if cookie == nil || cookie.Value != "admin" {
			http.Error(w, "admin required", http.StatusForbidden)
			return
		}
		_ = json.NewEncoder(w).Encode(map[string]any{"post": api.BlogPost{
			BlogListPost: api.BlogListPost{ID: 7, Slug: "draft-post", Title: "Draft Post", Tags: []string{"ships"}},
			BodyMD:       "Draft **body**", BodyHTML: "<script>alert(1)</script>", Status: 0,
			AuthorID: 42, AuthorName: "Example Pilot",
		}})
	})
	handler := New(Options{Blog: &fixtureBlogReader{}, BlogPreviewAPI: previewAPI})
	unauthorized := httptest.NewRecorder()
	handler.ServeHTTP(unauthorized, httptest.NewRequest("GET", "/blog/preview/draft-post", nil))
	if unauthorized.Code != http.StatusNotFound || strings.Contains(unauthorized.Body.String(), "Draft Post") {
		t.Fatalf("unauthorized blog preview: status %d", unauthorized.Code)
	}
	if apiPath != "/api/admin/blog/preview/draft-post" {
		t.Fatalf("preview delegated to %q", apiPath)
	}
	head := httptest.NewRecorder()
	handler.ServeHTTP(head, httptest.NewRequest("HEAD", "/blog/preview/draft-post", nil))
	if head.Code != http.StatusNotFound || head.Body.Len() != 0 {
		t.Fatalf("unauthorized blog preview HEAD: status %d, body length %d", head.Code, head.Body.Len())
	}
	request := httptest.NewRequest("GET", "/blog/preview/draft-post", nil)
	request.AddCookie(&http.Cookie{Name: "evelogin", Value: "admin"})
	preview := httptest.NewRecorder()
	handler.ServeHTTP(preview, request)
	if preview.Code != 200 {
		t.Fatalf("admin blog preview status = %d", preview.Code)
	}
	for _, marker := range []string{"Preview mode", "Draft Post", "Draft <strong>body</strong>", "unpublished", "Edit in admin", `href="/admin/blog"`} {
		if !strings.Contains(preview.Body.String(), marker) {
			t.Errorf("blog preview missing %q", marker)
		}
	}
	if strings.Contains(preview.Body.String(), "<script>alert(1)</script>") || !strings.Contains(preview.Header().Get("Cache-Control"), "no-store") || !strings.Contains(preview.Header().Get("X-Robots-Tag"), "noindex") || strings.Contains(preview.Body.String(), `rel="canonical"`) {
		t.Fatal("blog preview leaked unsafe HTML, caching, or indexing")
	}
}
