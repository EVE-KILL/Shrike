package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"time"
)

const fixtureBlogMarkdown = "# News from New Eden\n\nA closer look at the killboard, **reusable components**, *personal themes*, and `inline code`.\n\n## Shared components\n\nVisit the [tracked dashboard](/tracker) or browse the [kill feed](/kills/latest).\n\n> A useful update should explain what changed and why it matters.\n\n### Features in this update\n\n- Server-rendered pages\n- Live killmails and reusable lists\n- Responsive layouts and personal themes\n\n1. Choose the entities you follow.\n2. Customize your dashboard.\n3. Review your activity.\n\n| Page | Purpose |\n| --- | --- |\n| Tracker | Your private killboard |\n| Blog | News and updates |\n\n```text\nA deliberately long line exercises the code block's own scrolling instead of stretching the page sideways at a narrow mobile width.\n```\n\n---\n\n![Rifter](/images/types/587/render?size=256)\n\n## More to come\n\nDisposable comparison content, with a complete author card and a real comment thread.\n"

// Optional, account-isolated records exercise the real blog writer and its
// Markdown renderer. This handler is the fixture's local API, never its public
// production proxy. No existing non-fixture posts are updated.
func seedBlogFixtures(ctx context.Context, handler http.Handler, owner account) error {
	now := time.Now().UTC().Truncate(time.Millisecond)
	send := func(method, path string, body any) error {
		raw, err := json.Marshal(body)
		if err != nil {
			return err
		}
		req := httptest.NewRequest(method, "http://localhost"+path, bytes.NewReader(raw)).WithContext(ctx)
		req.Header.Set("Origin", "http://localhost")
		req.Header.Set("Content-Type", "application/json")
		// #nosec G124 -- outgoing request cookie, not a browser Set-Cookie policy.
		req.AddCookie(&http.Cookie{Name: "evelogin", Value: owner.session})
		response := httptest.NewRecorder()
		handler.ServeHTTP(response, req)
		if response.Code < 200 || response.Code >= 300 {
			return fmt.Errorf("fixture blog writer %s returned %d", path, response.Code)
		}
		return nil
	}
	for i := 1; i <= 42; i++ {
		tags := []string{"fixture", "dispatch"}
		if i%2 == 0 {
			tags[1] = "updates"
		}
		var excerpt, cover any
		if i%3 != 0 {
			excerpt = "Updates from across New Eden, with reusable components and a consistent reading experience."
		}
		if i%4 == 0 {
			cover = "/__fixture/blog-cover.svg"
		}
		if err := send("POST", "/api/admin/blog", map[string]any{"slug": fmt.Sprintf("fixture-blog-%02d", i), "title": fmt.Sprintf("New Eden dispatch %02d", i), "body_md": fmt.Sprintf("## Dispatch %02d\n\nDisposable comparison content.", i), "excerpt": excerpt, "cover_image_url": cover, "status": 1, "published_at": now.Add(-48*time.Hour - time.Duration(i)*time.Hour), "tags": tags}); err != nil {
			return err
		}
	}
	if err := send("POST", "/api/admin/blog", map[string]any{"slug": "fixture-blog-rich", "title": "News from New Eden", "excerpt": "A closer look at the killboard, reusable components, and the activity you follow.", "body_md": fixtureBlogMarkdown, "cover_image_url": "/__fixture/blog-cover.svg", "status": 1, "published_at": now, "tags": []string{"fixture", "updates", "killboard", "components", "themes"}}); err != nil {
		return err
	}
	if err := send("POST", "/api/admin/blog", map[string]any{"slug": "fixture-blog-archive", "title": "An archived New Eden update", "body_md": "## Archived article\n\nThis article remains available at its original URL.", "status": 2, "published_at": now.Add(-30 * 24 * time.Hour), "tags": []string{"fixture", "archive"}}); err != nil {
		return err
	}
	return send("PATCH", "/api/admin/blog/91002", map[string]any{"title": "Preview: the next update", "excerpt": "A private draft for checking the blog before publication.", "body_md": fixtureBlogMarkdown, "cover_image_url": "/__fixture/blog-cover.svg", "tags": []string{"fixture", "draft", "updates"}})
}
