package api

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"net/http"
	"net/url"
	"sort"
	"strconv"
	"strings"
	"time"
	"unicode"

	"github.com/danielgtaylor/huma/v2"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/russross/blackfriday/v2"
	"golang.org/x/net/html"
	"golang.org/x/net/html/atom"
	"golang.org/x/text/unicode/norm"
)

const (
	contentBodyLimit  = 2 << 20
	blogStatusDraft   = 0
	blogStatusLive    = 1
	blogStatusArchive = 2
)

const blogColumns = `
	id,
	slug,
	title,
	excerpt,
	body_md,
	body_html,
	cover_image_url,
	status,
	author_id,
	author_name,
	published_at,
	created_at,
	updated_at,
	tags,
	author_corporation_id,
	author_corporation_name,
	author_alliance_id,
	author_alliance_name`

type blogAuthor struct {
	CorporationID   *int32
	CorporationName *string
	AllianceID      *int32
	AllianceName    *string
}

type blogCreate struct {
	Slug          string
	Title         string
	Excerpt       *string
	BodyMD        string
	BodyHTML      string
	CoverImageURL *string
	Status        int16
	AuthorID      int32
	AuthorName    string
	Author        blogAuthor
	Tags          []string
	PublishedAt   *time.Time
}

type blogStore interface {
	PublicList(
		context.Context,
		*time.Time,
		*string,
		int,
	) ([]map[string]any, error)
	PublicBySlug(context.Context, string) (map[string]any, error)
	AdminList(context.Context, string, int) ([]map[string]any, error)
	ByID(context.Context, int64) (map[string]any, error)
	BySlug(context.Context, string) (map[string]any, error)
	SlugExists(context.Context, string, int64) (bool, error)
	ResolveAuthor(context.Context, int32) (blogAuthor, error)
	Create(context.Context, blogCreate) (map[string]any, error)
	Update(
		context.Context,
		int64,
		map[string]any,
	) (map[string]any, error)
	Delete(context.Context, int64) (bool, error)
}

type postgresBlogStore struct {
	db MutationDatabase
}

type blogService struct {
	auth     *authService
	store    blogStore
	storeErr error
	now      func() time.Time
}

func newBlogService(opts Options) *blogService {
	service := &blogService{
		auth: newAuthService(opts),
		now:  time.Now,
	}
	db, err := mutationDatabase(opts)
	if err != nil {
		service.storeErr = err
	} else {
		service.store = &postgresBlogStore{db: db}
	}
	return service
}

func (s *blogService) requireStore() error {
	if s.storeErr != nil {
		return s.storeErr
	}
	if s.store == nil {
		return apiError(
			http.StatusServiceUnavailable,
			"Blog storage is not configured",
		)
	}
	return nil
}

func requireContentPrincipal(
	ctx context.Context,
	req *legacyRequest,
	auth *authService,
) (*Principal, error) {
	principal, err := auth.requirePrincipal(ctx, req)
	if err != nil {
		return nil, err
	}
	return principal, nil
}

func requireContentAdmin(
	ctx context.Context,
	req *legacyRequest,
	auth *authService,
) (*Principal, error) {
	principal, err := requireContentPrincipal(ctx, req, auth)
	if err != nil {
		return nil, err
	}
	if !principal.IsAdmin {
		return nil, apiError(
			http.StatusForbidden,
			"Admin access required",
		)
	}
	return principal, nil
}

func boundedContentInt(raw string, fallback, minimum, maximum int) int {
	value, err := strconv.ParseFloat(strings.TrimSpace(raw), 64)
	if err != nil || value == 0 || math.IsNaN(value) ||
		math.IsInf(value, 0) {
		return fallback
	}
	result := int(value)
	if result < minimum {
		return minimum
	}
	if result > maximum {
		return maximum
	}
	return result
}

func contentID(raw, message string) (int64, error) {
	value, err := strconv.ParseInt(strings.TrimSpace(raw), 10, 64)
	if err != nil || value <= 0 {
		return 0, apiError(http.StatusBadRequest, message)
	}
	return value, nil
}

func stringField(value any) string {
	if value == nil {
		return ""
	}
	if result, ok := value.(string); ok {
		return result
	}
	return stringifyJSONValue(value)
}

func optionalTrimmedString(value any, maximum int) *string {
	raw := strings.TrimSpace(stringField(value))
	if raw == "" {
		return nil
	}
	raw = truncateRunes(raw, maximum)
	return &raw
}

func parseContentTime(value any) (time.Time, bool) {
	switch typed := value.(type) {
	case string:
		parsed, err := time.Parse(time.RFC3339Nano, strings.TrimSpace(typed))
		return parsed, err == nil
	case json.Number:
		number, err := typed.Float64()
		if err != nil || math.IsNaN(number) || math.IsInf(number, 0) {
			return time.Time{}, false
		}
		return time.UnixMilli(int64(number)).UTC(), true
	case float64:
		if math.IsNaN(typed) || math.IsInf(typed, 0) {
			return time.Time{}, false
		}
		return time.UnixMilli(int64(typed)).UTC(), true
	default:
		return time.Time{}, false
	}
}

func registerBlogRoutes(a huma.API, opts Options) {
	registerBlogServiceRoutes(a, opts, newBlogService(opts))
}

func registerBlogServiceRoutes(
	a huma.API,
	opts Options,
	service *blogService,
) {
	requiredSession := []map[string][]string{{"eveSession": {}}}
	publicCache := func(next legacyHandler) legacyHandler {
		return routeJSONCache(
			opts,
			5*time.Minute,
			"public, s-maxage=300, stale-while-revalidate=900",
			next,
		)
	}
	registerLegacy(a, huma.Operation{
		OperationID: "blog-posts",
		Method:      http.MethodGet,
		Path:        "/blog",
		Summary:     "Published blog posts",
		Tags:        []string{"blog"},
	}, publicCache(service.publicListHandler()))
	registerLegacy(a, huma.Operation{
		OperationID: "blog-post",
		Method:      http.MethodGet,
		Path:        "/blog/{slug}",
		Summary:     "Published or archived blog post",
		Tags:        []string{"blog"},
	}, publicCache(service.publicPostHandler()))

	registerLegacy(a, huma.Operation{
		OperationID: "blog-admin-list",
		Method:      http.MethodGet,
		Path:        "/admin/blog",
		Summary:     "Blog administration",
		Tags:        []string{"admin", "blog"},
		Security:    requiredSession,
	}, service.adminListHandler())
	registerLegacy(a, documentJSONBody[blogCreateDocument](a, huma.Operation{
		OperationID: "blog-admin-create",
		Method:      http.MethodPost,
		Path:        "/admin/blog",
		Summary:     "Create a blog post",
		Tags:        []string{"admin", "blog"},
		Security:    requiredSession,
	}), service.adminCreateHandler())
	registerLegacy(a, huma.Operation{
		OperationID: "blog-admin-detail",
		Method:      http.MethodGet,
		Path:        "/admin/blog/{id}",
		Summary:     "Blog post administration detail",
		Tags:        []string{"admin", "blog"},
		Security:    requiredSession,
	}, service.adminDetailHandler())
	registerLegacy(a, documentJSONBody[blogUpdateDocument](a, huma.Operation{
		OperationID: "blog-admin-update",
		Method:      http.MethodPatch,
		Path:        "/admin/blog/{id}",
		Summary:     "Update a blog post",
		Tags:        []string{"admin", "blog"},
		Security:    requiredSession,
	}), service.adminUpdateHandler())
	registerLegacy(a, huma.Operation{
		OperationID: "blog-admin-delete",
		Method:      http.MethodDelete,
		Path:        "/admin/blog/{id}",
		Summary:     "Permanently delete a blog post",
		Tags:        []string{"admin", "blog"},
		Security:    requiredSession,
	}, service.adminDeleteHandler())
	registerLegacy(a, huma.Operation{
		OperationID: "blog-admin-preview",
		Method:      http.MethodGet,
		Path:        "/admin/blog/preview/{slug}",
		Summary:     "Preview a blog post in any state",
		Tags:        []string{"admin", "blog"},
		Security:    requiredSession,
	}, service.adminPreviewHandler())
}

func (s *blogService) publicListHandler() legacyHandler {
	return func(ctx context.Context, req *legacyRequest) (legacyPayload, error) {
		if err := s.requireStore(); err != nil {
			return legacyPayload{}, err
		}
		limit := boundedContentInt(req.Query.Get("limit"), 20, 1, 50)
		var cursor *time.Time
		if raw := req.Query.Get("cursor"); raw != "" {
			value, ok := parseContentTime(raw)
			if !ok {
				return legacyPayload{}, apiError(
					http.StatusBadRequest,
					"invalid cursor",
				)
			}
			cursor = &value
		}
		var tag *string
		if value := normalizeBlogTag(req.Query.Get("tag")); value != "" {
			tag = &value
		}
		rows, err := s.store.PublicList(
			ctx,
			cursor,
			tag,
			limit+1,
		)
		if err != nil {
			return legacyPayload{}, err
		}
		hasMore := len(rows) > limit
		if hasMore {
			rows = rows[:limit]
		}
		var next any
		if hasMore && len(rows) > 0 {
			if value, ok := timeFrom(rows[len(rows)-1]["published_at"]); ok {
				next = javascriptTimestamp(value)
			}
		}
		return jsonPayload(map[string]any{
			"posts": rows, "nextCursor": next,
		}), nil
	}
}

func (s *blogService) publicPostHandler() legacyHandler {
	return func(ctx context.Context, req *legacyRequest) (legacyPayload, error) {
		if err := s.requireStore(); err != nil {
			return legacyPayload{}, err
		}
		slug := strings.TrimSpace(req.Param("slug"))
		if slug == "" {
			return legacyPayload{}, apiError(
				http.StatusBadRequest,
				"slug required",
			)
		}
		row, err := s.store.PublicBySlug(ctx, slug)
		if err != nil {
			return legacyPayload{}, err
		}
		if row == nil {
			return legacyPayload{}, apiError(
				http.StatusNotFound,
				"post not found",
			)
		}
		return jsonPayload(map[string]any{"post": row}), nil
	}
}

func (s *blogService) adminListHandler() legacyHandler {
	return func(ctx context.Context, req *legacyRequest) (legacyPayload, error) {
		setAccountNoStore(req.Huma)
		if err := s.requireStore(); err != nil {
			return legacyPayload{}, err
		}
		if _, err := requireContentAdmin(ctx, req, s.auth); err != nil {
			return legacyPayload{}, err
		}
		rows, err := s.store.AdminList(
			ctx,
			req.Query.Get("status"),
			boundedContentInt(req.Query.Get("limit"), 50, 1, 200),
		)
		if err != nil {
			return legacyPayload{}, err
		}
		return accountNoStorePayload(map[string]any{"posts": rows}), nil
	}
}

func (s *blogService) adminDetailHandler() legacyHandler {
	return func(ctx context.Context, req *legacyRequest) (legacyPayload, error) {
		setAccountNoStore(req.Huma)
		if err := s.requireStore(); err != nil {
			return legacyPayload{}, err
		}
		if _, err := requireContentAdmin(ctx, req, s.auth); err != nil {
			return legacyPayload{}, err
		}
		id, err := contentID(req.Param("id"), "invalid id")
		if err != nil {
			return legacyPayload{}, err
		}
		row, err := s.store.ByID(ctx, id)
		if err != nil {
			return legacyPayload{}, err
		}
		if row == nil {
			return legacyPayload{}, apiError(
				http.StatusNotFound,
				"post not found",
			)
		}
		return accountNoStorePayload(map[string]any{"post": row}), nil
	}
}

func (s *blogService) adminPreviewHandler() legacyHandler {
	return func(ctx context.Context, req *legacyRequest) (legacyPayload, error) {
		setAccountNoStore(req.Huma)
		if err := s.requireStore(); err != nil {
			return legacyPayload{}, err
		}
		if _, err := requireContentAdmin(ctx, req, s.auth); err != nil {
			return legacyPayload{}, err
		}
		slug := strings.TrimSpace(req.Param("slug"))
		if slug == "" {
			return legacyPayload{}, apiError(
				http.StatusBadRequest,
				"slug required",
			)
		}
		row, err := s.store.BySlug(ctx, slug)
		if err != nil {
			return legacyPayload{}, err
		}
		if row == nil {
			return legacyPayload{}, apiError(
				http.StatusNotFound,
				"post not found",
			)
		}
		payload := accountNoStorePayload(map[string]any{"post": row})
		payload.Headers.Set(
			"Cache-Control",
			"private, no-store, no-cache, must-revalidate",
		)
		return payload, nil
	}
}

// Runtime decode types for the blog write routes.
//
// Update branches on key presence for every field, so raw messages preserve
// absent versus null. openapi_body_types.go documents the concrete wire
// shapes independently.
type blogCreateBody struct {
	Title         string          `json:"title" doc:"Post title."`
	Slug          string          `json:"slug,omitempty" doc:"URL slug. Derived from the title when omitted."`
	BodyMD        string          `json:"body_md" doc:"Post body, in Markdown."`
	Excerpt       json.RawMessage `json:"excerpt,omitempty" doc:"Short summary, at most 500 characters."`
	CoverImageURL json.RawMessage `json:"cover_image_url,omitempty" doc:"Cover image URL, at most 4096 characters."`
	Status        json.RawMessage `json:"status,omitempty" doc:"Publication status."`
	PublishedAt   json.RawMessage `json:"published_at,omitempty" doc:"Publication timestamp."`
	Tags          json.RawMessage `json:"tags,omitempty" doc:"Tag list."`
}

// blogUpdateBody patches a post. Every field is optional and an absent field
// is left unchanged.
type blogUpdateBody struct {
	Title         json.RawMessage `json:"title,omitempty" doc:"New title."`
	Slug          json.RawMessage `json:"slug,omitempty" doc:"New slug."`
	BodyMD        json.RawMessage `json:"body_md,omitempty" doc:"New body, in Markdown."`
	Excerpt       json.RawMessage `json:"excerpt,omitempty" doc:"New excerpt."`
	CoverImageURL json.RawMessage `json:"cover_image_url,omitempty" doc:"New cover image URL."`
	Status        json.RawMessage `json:"status,omitempty" doc:"New publication status."`
	PublishedAt   json.RawMessage `json:"published_at,omitempty" doc:"New publication timestamp."`
	Tags          json.RawMessage `json:"tags,omitempty" doc:"Replacement tag list."`
}

func (s *blogService) adminCreateHandler() legacyHandler {
	return func(ctx context.Context, req *legacyRequest) (legacyPayload, error) {
		setAccountNoStore(req.Huma)
		if err := requireSameOriginMutation(req.Huma); err != nil {
			return legacyPayload{}, err
		}
		if err := s.requireStore(); err != nil {
			return legacyPayload{}, err
		}
		principal, err := requireContentAdmin(ctx, req, s.auth)
		if err != nil {
			return legacyPayload{}, err
		}
		body, err := decodeJSONBody[blogCreateBody](req, contentBodyLimit)
		if err != nil {
			return legacyPayload{}, err
		}
		title := strings.TrimSpace(body.Title)
		if title == "" {
			return legacyPayload{}, apiError(
				http.StatusBadRequest,
				"title is required",
			)
		}
		slug := slugifyBlogTitle(body.Slug)
		if slug == "" {
			slug = slugifyBlogTitle(title)
		}
		if slug == "" {
			return legacyPayload{}, apiError(
				http.StatusBadRequest,
				"slug could not be generated from title",
			)
		}
		status := int16(blogStatusDraft)
		if value, ok := contentInteger(rawJSONValue(body.Status)); ok &&
			(value == blogStatusDraft || value == blogStatusLive ||
				value == blogStatusArchive) {
			status = int16(value)
		}
		var publishedAt *time.Time
		if status == blogStatusLive {
			value := s.now().UTC()
			if raw, found := rawJSONField(body.PublishedAt); found && raw != nil {
				parsed, ok := parseContentTime(raw)
				if !ok {
					return legacyPayload{}, apiError(
						http.StatusBadRequest,
						"invalid published_at",
					)
				}
				value = parsed
			}
			publishedAt = &value
		}
		exists, err := s.store.SlugExists(ctx, slug, 0)
		if err != nil {
			return legacyPayload{}, err
		}
		if exists {
			return legacyPayload{}, apiError(
				http.StatusConflict,
				fmt.Sprintf("slug %q already exists", slug),
			)
		}
		author, err := s.store.ResolveAuthor(ctx, principal.CharacterID)
		if err != nil {
			return legacyPayload{}, err
		}
		bodyMD := body.BodyMD
		row, err := s.store.Create(ctx, blogCreate{
			Slug: slug, Title: title,
			Excerpt: optionalTrimmedString(rawJSONValue(body.Excerpt), 500),
			BodyMD:  bodyMD, BodyHTML: renderBlogMarkdown(bodyMD),
			CoverImageURL: optionalTrimmedString(
				rawJSONValue(body.CoverImageURL), 4096,
			),
			Status: status, AuthorID: principal.CharacterID,
			AuthorName: principal.CharacterName, Author: author,
			Tags:        normalizeBlogTags(rawJSONValue(body.Tags)),
			PublishedAt: publishedAt,
		})
		if isUniqueViolation(err) {
			return legacyPayload{}, apiError(
				http.StatusConflict,
				fmt.Sprintf("slug %q already exists", slug),
			)
		}
		if err != nil {
			return legacyPayload{}, err
		}
		return accountNoStorePayload(map[string]any{"post": row}), nil
	}
}

func (s *blogService) adminUpdateHandler() legacyHandler {
	return func(ctx context.Context, req *legacyRequest) (legacyPayload, error) {
		setAccountNoStore(req.Huma)
		if err := requireSameOriginMutation(req.Huma); err != nil {
			return legacyPayload{}, err
		}
		if err := s.requireStore(); err != nil {
			return legacyPayload{}, err
		}
		if _, err := requireContentAdmin(ctx, req, s.auth); err != nil {
			return legacyPayload{}, err
		}
		id, err := contentID(req.Param("id"), "invalid id")
		if err != nil {
			return legacyPayload{}, err
		}
		body, err := decodeJSONBody[blogUpdateBody](req, contentBodyLimit)
		if err != nil {
			return legacyPayload{}, err
		}
		current, err := s.store.ByID(ctx, id)
		if err != nil {
			return legacyPayload{}, err
		}
		if current == nil {
			return legacyPayload{}, apiError(
				http.StatusNotFound,
				"post not found",
			)
		}
		update := map[string]any{"updated_at": s.now().UTC()}
		if raw, found := rawJSONField(body.Title); found {
			title := strings.TrimSpace(stringField(raw))
			if title == "" {
				return legacyPayload{}, apiError(
					http.StatusBadRequest,
					"title cannot be empty",
				)
			}
			update["title"] = title
		}
		if raw, found := rawJSONField(body.Slug); found {
			slug := slugifyBlogTitle(stringField(raw))
			if slug == "" {
				return legacyPayload{}, apiError(
					http.StatusBadRequest,
					"invalid slug",
				)
			}
			if slug != mapString(current, "slug", "") {
				exists, err := s.store.SlugExists(ctx, slug, id)
				if err != nil {
					return legacyPayload{}, err
				}
				if exists {
					return legacyPayload{}, apiError(
						http.StatusConflict,
						fmt.Sprintf("slug %q already exists", slug),
					)
				}
				update["slug"] = slug
			}
		}
		if raw, found := rawJSONField(body.Excerpt); found {
			update["excerpt"] = optionalTrimmedString(raw, 500)
		}
		if raw, found := rawJSONField(body.BodyMD); found {
			bodyMD := stringField(raw)
			update["body_md"] = bodyMD
			update["body_html"] = renderBlogMarkdown(bodyMD)
		}
		if raw, found := rawJSONField(body.CoverImageURL); found {
			update["cover_image_url"] = optionalTrimmedString(raw, 4096)
		}
		if raw, found := rawJSONField(body.Status); found {
			value, ok := contentInteger(raw)
			if !ok || (value != blogStatusDraft &&
				value != blogStatusLive && value != blogStatusArchive) {
				return legacyPayload{}, apiError(
					http.StatusBadRequest,
					"status must be 0, 1, or 2",
				)
			}
			update["status"] = int16(value)
			currentStatus, _ := int64Value(current["status"])
			if value == blogStatusLive &&
				currentStatus != blogStatusLive {
				if _, explicit := rawJSONField(body.PublishedAt); !explicit &&
					current["published_at"] == nil {
					update["published_at"] = s.now().UTC()
				}
			}
		}
		if raw, found := rawJSONField(body.Tags); found {
			update["tags"] = normalizeBlogTags(raw)
		}
		if raw, found := rawJSONField(body.PublishedAt); found {
			if raw == nil {
				update["published_at"] = nil
			} else {
				value, ok := parseContentTime(raw)
				if !ok {
					return legacyPayload{}, apiError(
						http.StatusBadRequest,
						"invalid published_at",
					)
				}
				update["published_at"] = value
			}
		}
		row, err := s.store.Update(ctx, id, update)
		if isUniqueViolation(err) {
			slug := stringField(update["slug"])
			return legacyPayload{}, apiError(
				http.StatusConflict,
				fmt.Sprintf("slug %q already exists", slug),
			)
		}
		if err != nil {
			return legacyPayload{}, err
		}
		return accountNoStorePayload(map[string]any{"post": row}), nil
	}
}

func (s *blogService) adminDeleteHandler() legacyHandler {
	return func(ctx context.Context, req *legacyRequest) (legacyPayload, error) {
		setAccountNoStore(req.Huma)
		if err := requireSameOriginMutation(req.Huma); err != nil {
			return legacyPayload{}, err
		}
		if err := s.requireStore(); err != nil {
			return legacyPayload{}, err
		}
		if _, err := requireContentAdmin(ctx, req, s.auth); err != nil {
			return legacyPayload{}, err
		}
		id, err := contentID(req.Param("id"), "invalid id")
		if err != nil {
			return legacyPayload{}, err
		}
		found, err := s.store.Delete(ctx, id)
		if err != nil {
			return legacyPayload{}, err
		}
		if !found {
			return legacyPayload{}, apiError(
				http.StatusNotFound,
				"post not found",
			)
		}
		return accountNoStorePayload(
			map[string]any{"ok": true, "id": id},
		), nil
	}
}

func contentInteger(value any) (int64, bool) {
	switch typed := value.(type) {
	case json.Number:
		number, err := typed.Float64()
		if err != nil || math.Trunc(number) != number {
			return 0, false
		}
		return int64(number), true
	case float64:
		if math.Trunc(typed) != typed {
			return 0, false
		}
		return int64(typed), true
	case string:
		number, err := strconv.ParseFloat(strings.TrimSpace(typed), 64)
		if err != nil || math.Trunc(number) != number {
			return 0, false
		}
		return int64(number), true
	default:
		return 0, false
	}
}

func isUniqueViolation(err error) bool {
	var pgErr *pgconn.PgError
	return errors.As(err, &pgErr) && pgErr.Code == "23505"
}

func (s *postgresBlogStore) PublicList(
	ctx context.Context,
	cursor *time.Time,
	tag *string,
	limit int,
) ([]map[string]any, error) {
	filters := []string{"status = 1"}
	args := []any{}
	add := func(value any) string {
		args = append(args, value)
		return fmt.Sprintf("$%d", len(args))
	}
	if cursor != nil {
		filters = append(filters, "published_at < "+add(*cursor))
	}
	if tag != nil {
		filters = append(filters, "tags @> ARRAY["+add(*tag)+"]::text[]")
	}
	limitArg := add(limit)
	return queryMaps(ctx, s.db, `
		SELECT
			id,
			slug,
			title,
			excerpt,
			cover_image_url,
			tags,
			author_id,
			author_name,
			author_corporation_id,
			author_corporation_name,
			author_alliance_id,
			author_alliance_name,
			published_at
		FROM blog_posts
		WHERE `+strings.Join(filters, " AND ")+`
		ORDER BY published_at DESC
		LIMIT `+limitArg,
		args...,
	)
}

func (s *postgresBlogStore) PublicBySlug(
	ctx context.Context,
	slug string,
) (map[string]any, error) {
	return queryMap(ctx, s.db, `
		SELECT `+blogColumns+`
		FROM blog_posts
		WHERE slug = $1
		  AND status IN (1, 2)
		LIMIT 1`,
		slug,
	)
}

func (s *postgresBlogStore) AdminList(
	ctx context.Context,
	status string,
	limit int,
) ([]map[string]any, error) {
	query := `
		SELECT ` + blogColumns + `
		FROM blog_posts`
	args := []any{}
	switch status {
	case "draft":
		query += ` WHERE status = 0`
	case "published":
		query += ` WHERE status = 1`
	case "archived":
		query += ` WHERE status = 2`
	}
	query += ` ORDER BY created_at DESC LIMIT $1`
	args = append(args, limit)
	return queryMaps(ctx, s.db, query, args...)
}

func (s *postgresBlogStore) ByID(
	ctx context.Context,
	id int64,
) (map[string]any, error) {
	return queryMap(ctx, s.db, `
		SELECT `+blogColumns+`
		FROM blog_posts
		WHERE id = $1
		LIMIT 1`,
		id,
	)
}

func (s *postgresBlogStore) BySlug(
	ctx context.Context,
	slug string,
) (map[string]any, error) {
	return queryMap(ctx, s.db, `
		SELECT `+blogColumns+`
		FROM blog_posts
		WHERE slug = $1
		LIMIT 1`,
		slug,
	)
}

func (s *postgresBlogStore) SlugExists(
	ctx context.Context,
	slug string,
	exceptID int64,
) (bool, error) {
	var exists bool
	err := s.db.QueryRow(ctx, `
		SELECT EXISTS (
			SELECT 1
			FROM blog_posts
			WHERE slug = $1
			  AND ($2::bigint = 0 OR id <> $2)
		)`,
		slug,
		exceptID,
	).Scan(&exists)
	return exists, err
}

func (s *postgresBlogStore) ResolveAuthor(
	ctx context.Context,
	characterID int32,
) (blogAuthor, error) {
	var result blogAuthor
	err := s.db.QueryRow(ctx, `
		SELECT
			c.corporation_id,
			corp.name,
			c.alliance_id,
			ally.name
		FROM characters c
		LEFT JOIN corporations corp
		  ON corp.corporation_id = c.corporation_id
		LEFT JOIN alliances ally
		  ON ally.alliance_id = c.alliance_id
		WHERE c.character_id = $1
		LIMIT 1`,
		characterID,
	).Scan(
		&result.CorporationID,
		&result.CorporationName,
		&result.AllianceID,
		&result.AllianceName,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return blogAuthor{}, nil
	}
	return result, err
}

func (s *postgresBlogStore) Create(
	ctx context.Context,
	input blogCreate,
) (map[string]any, error) {
	return queryMap(ctx, s.db, `
		INSERT INTO blog_posts (
			slug,
			title,
			excerpt,
			body_md,
			body_html,
			cover_image_url,
			status,
			author_id,
			author_name,
			author_corporation_id,
			author_corporation_name,
			author_alliance_id,
			author_alliance_name,
			tags,
			published_at
		)
		VALUES (
			$1, $2, $3, $4, $5, $6, $7, $8,
			$9, $10, $11, $12, $13, $14, $15
		)
		RETURNING `+blogColumns,
		input.Slug,
		input.Title,
		input.Excerpt,
		input.BodyMD,
		input.BodyHTML,
		input.CoverImageURL,
		input.Status,
		input.AuthorID,
		input.AuthorName,
		input.Author.CorporationID,
		input.Author.CorporationName,
		input.Author.AllianceID,
		input.Author.AllianceName,
		input.Tags,
		input.PublishedAt,
	)
}

func (s *postgresBlogStore) Update(
	ctx context.Context,
	id int64,
	update map[string]any,
) (map[string]any, error) {
	keys := make([]string, 0, len(update))
	for key := range update {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	assignments := make([]string, 0, len(keys))
	args := make([]any, 0, len(keys)+1)
	for _, key := range keys {
		args = append(args, update[key])
		assignments = append(
			assignments,
			fmt.Sprintf("%s = $%d", key, len(args)),
		)
	}
	args = append(args, id)
	return queryMap(ctx, s.db, `
		UPDATE blog_posts
		SET `+strings.Join(assignments, ", ")+`
		WHERE id = $`+strconv.Itoa(len(args))+`
		RETURNING `+blogColumns,
		args...,
	)
}

func (s *postgresBlogStore) Delete(
	ctx context.Context,
	id int64,
) (bool, error) {
	tag, err := s.db.Exec(ctx, `
		DELETE FROM blog_posts
		WHERE id = $1`,
		id,
	)
	return tag.RowsAffected() > 0, err
}

func normalizeBlogTags(value any) []string {
	items, ok := value.([]any)
	if !ok {
		if stringsValue, ok := value.([]string); ok {
			items = make([]any, len(stringsValue))
			for index := range stringsValue {
				items[index] = stringsValue[index]
			}
		} else {
			return []string{}
		}
	}
	result := make([]string, 0, min(len(items), 10))
	seen := make(map[string]bool)
	for _, item := range items {
		raw, ok := item.(string)
		if !ok {
			continue
		}
		tag := normalizeBlogTag(raw)
		if tag == "" || seen[tag] {
			continue
		}
		seen[tag] = true
		result = append(result, tag)
		if len(result) == 10 {
			break
		}
	}
	return result
}

func normalizeBlogTag(value string) string {
	value = strings.ToLower(norm.NFKD.String(strings.TrimSpace(value)))
	var output strings.Builder
	previousWhitespace := false
	for _, r := range value {
		switch {
		case unicode.Is(unicode.Mn, r):
			continue
		case r >= 'a' && r <= 'z', r >= '0' && r <= '9':
			output.WriteRune(r)
			previousWhitespace = false
		case unicode.IsSpace(r):
			if output.Len() > 0 && !previousWhitespace {
				output.WriteByte('-')
				previousWhitespace = true
			}
		case r == '-':
			output.WriteByte('-')
			previousWhitespace = false
		default:
			// Invalid characters are stripped after whitespace runs are
			// replaced in the TypeScript pipeline, so they still separate
			// two runs ("a  !  b" becomes "a--b").
			previousWhitespace = false
		}
	}
	return truncateRunes(strings.Trim(output.String(), "-"), 40)
}

func slugifyBlogTitle(value string) string {
	value = strings.ToLower(norm.NFKD.String(value))
	var output strings.Builder
	separator := false
	for _, r := range value {
		switch {
		case unicode.Is(unicode.Mn, r):
			continue
		case r >= 'a' && r <= 'z', r >= '0' && r <= '9':
			if separator && output.Len() > 0 {
				output.WriteByte('-')
			}
			output.WriteRune(r)
			separator = false
		default:
			separator = true
		}
	}
	return truncateRunes(strings.Trim(output.String(), "-"), 80)
}

type contentMarkdownProfile struct {
	allowedTags  map[string]bool
	allowedAttrs map[string]bool
}

var blogMarkdownProfile = contentMarkdownProfile{
	allowedTags: valuesSet(
		"p", "br", "strong", "em", "b", "i", "u", "s", "del",
		"a", "code", "pre", "blockquote", "ul", "ol", "li",
		"h1", "h2", "h3", "h4", "h5", "h6", "hr", "details",
		"summary", "table", "thead", "tbody", "tr", "th", "td",
		"img", "figure", "figcaption",
	),
	allowedAttrs: valuesSet(
		"href", "target", "rel", "class", "src", "alt", "title", "loading",
		"colspan", "rowspan",
	),
}

var announcementMarkdownProfile = contentMarkdownProfile{
	allowedTags: valuesSet(
		"p", "br", "strong", "em", "b", "i", "u", "s", "del",
		"a", "code", "pre", "blockquote", "ul", "ol", "li",
		"h1", "h2", "h3", "h4", "hr", "details", "summary",
		"table", "thead", "tbody", "tr", "th", "td",
	),
	allowedAttrs: valuesSet(
		"href", "target", "rel", "class", "colspan", "rowspan",
	),
}

func renderBlogMarkdown(source string) string {
	return renderContentMarkdown(source, blogMarkdownProfile)
}

func renderAnnouncementMarkdown(source string) string {
	return renderContentMarkdown(source, announcementMarkdownProfile)
}

func renderContentMarkdown(
	source string,
	profile contentMarkdownProfile,
) string {
	if strings.TrimSpace(source) == "" {
		return ""
	}
	renderer := blackfriday.NewHTMLRenderer(
		blackfriday.HTMLRendererParameters{
			Flags: blackfriday.SkipHTML,
		},
	)
	raw := blackfriday.Run(
		[]byte(source),
		blackfriday.WithRenderer(renderer),
		blackfriday.WithExtensions(
			blackfriday.CommonExtensions,
		),
	)
	contextNode := &html.Node{
		Type: html.ElementNode, Data: "div", DataAtom: atom.Div,
	}
	nodes, err := html.ParseFragment(bytes.NewReader(raw), contextNode)
	if err != nil {
		return ""
	}
	var output bytes.Buffer
	for _, node := range nodes {
		appendContentNode(&output, node, profile)
	}
	return output.String()
}

func appendContentNode(
	output *bytes.Buffer,
	node *html.Node,
	profile contentMarkdownProfile,
) {
	for _, clean := range sanitizeContentNode(node, profile) {
		_ = html.Render(output, clean)
	}
}

func sanitizeContentNode(
	node *html.Node,
	profile contentMarkdownProfile,
) []*html.Node {
	if node.Type == html.TextNode {
		return []*html.Node{{
			Type: html.TextNode, Data: node.Data,
		}}
	}
	if node.Type != html.ElementNode {
		return nil
	}
	tag := strings.ToLower(node.Data)
	if !profile.allowedTags[tag] {
		result := []*html.Node{}
		for child := node.FirstChild; child != nil; child = child.NextSibling {
			result = append(
				result,
				sanitizeContentNode(child, profile)...,
			)
		}
		return result
	}
	clean := &html.Node{
		Type: html.ElementNode, Data: tag,
		DataAtom: atom.Lookup([]byte(tag)),
	}
	for _, attr := range node.Attr {
		name := strings.ToLower(attr.Key)
		value := strings.TrimSpace(attr.Val)
		if !profile.allowedAttrs[name] || strings.HasPrefix(name, "on") {
			continue
		}
		switch {
		case name == "href" && !safeContentURL(value, true):
			continue
		case name == "src" && !safeContentURL(value, false):
			continue
		case (name == "colspan" || name == "rowspan"):
			number, err := strconv.Atoi(value)
			if err != nil || number < 1 || number > 100 {
				continue
			}
		case len(value) > 4096:
			continue
		}
		clean.Attr = append(clean.Attr, html.Attribute{
			Key: name, Val: value,
		})
	}
	for child := node.FirstChild; child != nil; child = child.NextSibling {
		for _, childNode := range sanitizeContentNode(child, profile) {
			clean.AppendChild(childNode)
		}
	}
	return []*html.Node{clean}
}

func safeContentURL(value string, mailto bool) bool {
	if strings.HasPrefix(value, "/") && !strings.HasPrefix(value, "//") {
		return true
	}
	if strings.HasPrefix(value, "#") {
		return true
	}
	parsed, err := url.Parse(value)
	if err != nil {
		return false
	}
	if mailto && parsed.Scheme == "mailto" {
		return parsed.Opaque != "" || parsed.Path != ""
	}
	return parsed.Host != "" &&
		(parsed.Scheme == "http" || parsed.Scheme == "https")
}
