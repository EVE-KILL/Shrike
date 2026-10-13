package main

import (
	"context"
	"errors"
	"net/http"
	"net/url"
	"strconv"

	"github.com/eve-kill/shrike/internal/api"
)

type remoteBlog struct{ origin string }

func (r remoteBlog) PublicBlogPosts(ctx context.Context, cursor *api.BlogCursor, tag string, limit int) (api.BlogListPage, error) {
	query := url.Values{"limit": {strconv.Itoa(limit)}}
	if cursor != nil {
		query.Set("cursor", cursor.String())
	}
	if tag != "" {
		query.Set("tag", tag)
	}
	data, err := fixtureRead[struct {
		Posts      []api.BlogListPost `json:"posts"`
		NextCursor *string            `json:"nextCursor"`
	}](ctx, r.origin, "/api/blog?"+query.Encode())
	result := api.BlogListPage{Posts: data.Posts}
	if data.NextCursor != nil {
		result.NextCursor = *data.NextCursor
	}
	return result, err
}
func (r remoteBlog) PublicBlogPost(ctx context.Context, slug string) (api.BlogPost, bool, error) {
	data, err := fixtureRead[struct {
		Post api.BlogPost `json:"post"`
	}](ctx, r.origin, "/api/blog/"+url.PathEscape(slug))
	var failure *fixtureHTTPError
	if errors.As(err, &failure) && failure.status == http.StatusNotFound {
		return data.Post, false, nil
	}
	data.Post.SanitizeBody()
	return data.Post, err == nil, err
}

func fixtureBlogCover(w http.ResponseWriter, r *http.Request) bool {
	if r.URL.Path != "/__fixture/blog-cover.svg" {
		return false
	}
	w.Header().Set("Content-Type", "image/svg+xml")
	w.Header().Set("Cache-Control", "public, max-age=3600")
	if r.Method != http.MethodHead {
		_, _ = w.Write([]byte(`<svg xmlns="http://www.w3.org/2000/svg" width="1200" height="480" viewBox="0 0 1200 480"><defs><linearGradient id="g"><stop stop-color="#121524"/><stop offset="1" stop-color="#383454"/></linearGradient></defs><rect width="1200" height="480" fill="url(#g)"/><g fill="none" stroke="#8b80d0" opacity=".3"><circle cx="970" cy="240" r="290"/><circle cx="970" cy="240" r="210"/><circle cx="970" cy="240" r="130"/></g><path d="M770 320 1000 105 1125 245 985 325 1015 260 918 246 862 308Z" fill="#a5a0c0" opacity=".5"/><text x="64" y="240" fill="#fff" font-family="system-ui,sans-serif" font-size="56" font-weight="600">News from New Eden</text><text x="68" y="290" fill="#b3afc9" font-family="system-ui,sans-serif" font-size="25">EVE-KILL · A local blog comparison</text></svg>`))
	}
	return true
}
