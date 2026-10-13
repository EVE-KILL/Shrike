package frontend

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"time"

	"github.com/eve-kill/shrike/internal/api"
	"golang.org/x/net/html"
)

type blogListView struct {
	Tag  string
	Page api.BlogListPage
}

type blogDetailView struct {
	Post    api.BlogPost
	Cover   string
	Preview bool
}

func blogTag(raw string) string {
	value := []rune(strings.ToLower(strings.TrimSpace(raw)))
	if len(value) > 40 {
		value = value[:40]
	}
	return string(value)
}

func (s *site) blogListingPage(r *http.Request) (Page, error) {
	tag := blogTag(r.URL.Query().Get("tag"))
	posts, err := s.options.Blog.PublicBlogPosts(r.Context(), nil, tag, 20)
	if err != nil {
		return Page{}, err
	}
	meta := Metadata{
		Title: "Blog — EVE-KILL", CanonicalPath: "/blog", Indexable: true,
		Description:  "News, updates, and long-form posts from EVE-KILL.",
		ImageSources: blogListImageSources(posts.Posts),
		Stylesheets:  []string{"utility-page.css", "blog.css"}, ModuleScripts: []string{"blog.js"},
	}
	if tag != "" {
		meta.Title = "#" + tag + " — EVE-KILL Blog"
		meta.Description = "Posts tagged #" + tag + " on EVE-KILL."
	}
	return Page{Metadata: meta, Content: blogListing(blogListView{Tag: tag, Page: posts}), SharedCacheTTL: time.Minute}, nil
}

func (s *site) blogPostPage(r *http.Request) (Page, error) {
	post, found, err := s.options.Blog.PublicBlogPost(r.Context(), r.PathValue("slug"))
	if err != nil {
		return Page{}, err
	}
	if !found {
		return Page{}, ErrNotFound
	}
	description := post.Title
	if post.Excerpt != nil && strings.TrimSpace(*post.Excerpt) != "" {
		description = *post.Excerpt
	}
	cover := blogCoverImage(post.CoverImageURL)
	return Page{Metadata: Metadata{
		Title: post.Title + " — EVE-KILL Blog", OGTitle: post.Title,
		Description: description, CanonicalPath: blogPostURL(post.Slug),
		Image: cover, ImageAlt: post.Title, Kind: "article", Indexable: true, CommentMedia: true, CommentEditor: true,
		ImageSources: append([]string{cover}, blogBodyImageSources(post.BodyHTML)...),
		Stylesheets:  []string{"utility-page.css", "blog.css", "comment-thread.css"}, ModuleScripts: []string{"blog.js", "comment-thread.js"},
	}, Content: blogDetail(blogDetailView{Post: post, Cover: cover}), SharedCacheTTL: 5 * time.Minute}, nil
}

func (s *site) blogPreviewPage(r *http.Request) (Page, error) {
	slug := strings.TrimSpace(r.PathValue("slug"))
	if slug == "" || len(slug) > 200 {
		return Page{}, ErrNotFound
	}
	// Reuse the API's session resolution, admin check and unpublished-post read.
	// This does not make a network request or duplicate auth rules in frontend.
	apiRequest := r.Clone(r.Context())
	requestURL := *r.URL
	requestURL.Path = "/api/admin/blog/preview/" + url.PathEscape(slug)
	requestURL.RawPath = ""
	requestURL.RawQuery = ""
	apiRequest.URL = &requestURL
	apiRequest.Method = http.MethodGet
	apiRequest.RequestURI = ""
	response := httptest.NewRecorder()
	s.options.BlogPreviewAPI.ServeHTTP(response, apiRequest)
	if response.Code == http.StatusUnauthorized || response.Code == http.StatusForbidden || response.Code == http.StatusNotFound {
		return Page{}, ErrNotFound
	}
	if response.Code != http.StatusOK {
		return Page{}, fmt.Errorf("blog preview API returned %d", response.Code)
	}
	var payload struct {
		Post api.BlogPost `json:"post"`
	}
	if err := json.Unmarshal(response.Body.Bytes(), &payload); err != nil {
		return Page{}, fmt.Errorf("decode blog preview: %w", err)
	}
	if payload.Post.ID <= 0 {
		return Page{}, ErrNotFound
	}
	payload.Post.SanitizeBody()
	cover := blogCoverImage(payload.Post.CoverImageURL)
	return Page{Metadata: Metadata{
		Title:       "[Preview] " + payload.Post.Title + " — EVE-KILL Blog",
		Description: payload.Post.Title, CommentMedia: true, CommentEditor: true,
		ImageSources: append([]string{cover}, blogBodyImageSources(payload.Post.BodyHTML)...),
		Stylesheets:  []string{"utility-page.css", "blog.css", "comment-thread.css"}, ModuleScripts: []string{"blog.js", "comment-thread.js"},
	}, Content: blogDetail(blogDetailView{Post: payload.Post, Cover: cover, Preview: true})}, nil
}

func (s *site) blogListFragment(w http.ResponseWriter, r *http.Request) {
	raw := r.URL.Query().Get("cursor")
	cursor, err := api.ParseBlogCursor(raw)
	if raw == "" || err != nil {
		http.Error(w, "Invalid blog cursor", http.StatusBadRequest)
		return
	}
	posts, err := s.options.Blog.PublicBlogPosts(r.Context(), &cursor, blogTag(r.URL.Query().Get("tag")), 20)
	if err != nil {
		http.Error(w, "Could not load blog posts", http.StatusServiceUnavailable)
		return
	}
	w.Header().Set("X-Next-Cursor", posts.NextCursor)
	s.renderFragment(w, r, blogCards(posts.Posts))
}

func blogPostURL(slug string) string { return "/blog/" + url.PathEscape(slug) }

func blogTagURL(tag string) string { return "/blog?tag=" + url.QueryEscape(tag) }

func blogCoverImage(raw *string) string {
	if raw == nil {
		return ""
	}
	value := strings.TrimSpace(*raw)
	if strings.HasPrefix(value, "/") && !strings.HasPrefix(value, "//") {
		return value
	}
	parsed, err := url.Parse(value)
	if err == nil && parsed.Scheme == "https" && parsed.Host != "" && parsed.User == nil {
		return value
	}
	return ""
}

func blogListImageSources(posts []api.BlogListPost) []string {
	result := make([]string, 0, len(posts))
	for _, post := range posts {
		if source := blogCoverImage(post.CoverImageURL); source != "" {
			result = append(result, source)
		}
	}
	return result
}

func blogBodyImageSources(body string) []string {
	result := []string{}
	stream := html.NewTokenizer(strings.NewReader(body))
	for {
		kind := stream.Next()
		if kind == html.ErrorToken {
			return result
		}
		if kind != html.StartTagToken && kind != html.SelfClosingTagToken {
			continue
		}
		token := stream.Token()
		if token.Data != "img" {
			continue
		}
		for _, attr := range token.Attr {
			if attr.Key == "src" {
				result = append(result, attr.Val)
			}
		}
	}
}

func blogPostDate(post api.BlogListPost) string {
	if post.PublishedAt == nil {
		return ""
	}
	return post.PublishedAt.Format("Jan 2, 2006")
}

func blogPostTime(post api.BlogListPost) string {
	if post.PublishedAt == nil {
		return ""
	}
	return post.PublishedAt.UTC().Format(time.RFC3339)
}

func blogPostLongDate(post api.BlogListPost) string {
	if post.PublishedAt == nil {
		return ""
	}
	return post.PublishedAt.Format("January 2, 2006")
}

func blogCorporationName(post api.BlogPost) string {
	if post.AuthorCorporationName != nil && *post.AuthorCorporationName != "" {
		return *post.AuthorCorporationName
	}
	if post.AuthorCorporationID != nil {
		return fmt.Sprintf("Corporation %d", *post.AuthorCorporationID)
	}
	return "Corporation"
}

func blogAllianceName(post api.BlogPost) string {
	if post.AuthorAllianceName != nil && *post.AuthorAllianceName != "" {
		return *post.AuthorAllianceName
	}
	if post.AuthorAllianceID != nil {
		return fmt.Sprintf("Alliance %d", *post.AuthorAllianceID)
	}
	return "Alliance"
}

func blogPreviewStatusLabel(status int16) string {
	switch status {
	case 0:
		return "Draft"
	case 1:
		return "Published"
	case 2:
		return "Archived"
	default:
		return "Unknown"
	}
}

func blogPreviewStatusClass(status int16) string {
	switch status {
	case 1:
		return "published"
	case 2:
		return "archived"
	default:
		return "draft"
	}
}

func blogDescription(post api.BlogPost) string {
	if post.Excerpt != nil {
		return *post.Excerpt
	}
	return ""
}
func blogHasAffiliation(id *int64) bool { return id != nil && *id > 0 }
