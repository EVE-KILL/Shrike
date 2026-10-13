package api

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"
)

type BlogListPost struct {
	ID            int64      `json:"id"`
	Slug          string     `json:"slug"`
	Title         string     `json:"title"`
	Excerpt       *string    `json:"excerpt"`
	CoverImageURL *string    `json:"cover_image_url"`
	Tags          []string   `json:"tags"`
	PublishedAt   *time.Time `json:"published_at"`
}

type BlogListPage struct {
	Posts      []BlogListPost
	NextCursor string
}

type BlogPost struct {
	BlogListPost
	BodyMD                string  `json:"body_md"`
	BodyHTML              string  `json:"body_html"`
	Status                int16   `json:"status"`
	AuthorID              int64   `json:"author_id"`
	AuthorName            string  `json:"author_name"`
	AuthorCorporationID   *int64  `json:"author_corporation_id"`
	AuthorCorporationName *string `json:"author_corporation_name"`
	AuthorAllianceID      *int64  `json:"author_alliance_id"`
	AuthorAllianceName    *string `json:"author_alliance_name"`
}

func (post *BlogPost) SanitizeBody() {
	post.BodyHTML = renderBlogMarkdown(post.BodyMD)
}

// PublicBlogPosts gives the server-rendered listing the same ordered rows and
// cursor semantics as GET /api/blog without routing an HTTP request internally.
func (r *HomepageReader) PublicBlogPosts(ctx context.Context, cursor *BlogCursor, tag string, limit int) (BlogListPage, error) {
	if r == nil || r.db == nil {
		return BlogListPage{}, fmt.Errorf("blog database is not configured")
	}
	if limit < 1 || limit > 50 {
		return BlogListPage{}, fmt.Errorf("invalid blog page size")
	}
	var normalizedTag *string
	if value := normalizeBlogTag(tag); value != "" {
		normalizedTag = &value
	}
	rows, err := publicBlogList(ctx, r.db, cursor, normalizedTag, limit+1)
	if err != nil {
		return BlogListPage{}, err
	}
	hasMore := len(rows) > limit
	if hasMore {
		rows = rows[:limit]
	}
	result := BlogListPage{Posts: make([]BlogListPost, 0, len(rows))}
	for _, row := range rows {
		encoded, err := json.Marshal(row)
		if err != nil {
			return BlogListPage{}, err
		}
		var post BlogListPost
		if err := json.Unmarshal(encoded, &post); err != nil {
			return BlogListPage{}, err
		}
		result.Posts = append(result.Posts, post)
	}
	if hasMore && len(rows) > 0 {
		result.NextCursor = blogRowCursor(rows[len(rows)-1])
	}
	return result, nil
}

// PublicBlogPost shares the API's published-or-archived visibility rule.
// Regenerating HTML from Markdown also keeps older stored HTML under today's
// allowlist before a Go template deliberately emits it as markup.
func (r *HomepageReader) PublicBlogPost(ctx context.Context, slug string) (BlogPost, bool, error) {
	if r == nil || r.db == nil {
		return BlogPost{}, false, fmt.Errorf("blog database is not configured")
	}
	slug = strings.TrimSpace(slug)
	if slug == "" || len(slug) > 200 {
		return BlogPost{}, false, nil
	}
	row, err := publicBlogPostBySlug(ctx, r.db, slug)
	if err != nil || row == nil {
		return BlogPost{}, false, err
	}
	encoded, err := json.Marshal(row)
	if err != nil {
		return BlogPost{}, false, err
	}
	var post BlogPost
	if err := json.Unmarshal(encoded, &post); err != nil {
		return BlogPost{}, false, err
	}
	post.SanitizeBody()
	return post, true, nil
}
