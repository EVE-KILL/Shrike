package frontend

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/eve-kill/shrike/internal/api"
)

const commentFeedPageSize = 30

type feedComment struct {
	ID              int64      `json:"id"`
	TargetType      int        `json:"target_type"`
	TargetID        int64      `json:"target_id"`
	TargetSlug      string     `json:"target_slug"`
	CharacterID     int64      `json:"character_id"`
	CharacterName   string     `json:"character_name"`
	CorporationID   int64      `json:"corporation_id"`
	CorporationName string     `json:"corporation_name"`
	AllianceID      *int64     `json:"alliance_id"`
	AllianceName    string     `json:"alliance_name"`
	BodyHTML        string     `json:"body_html"`
	CreatedAt       time.Time  `json:"created_at"`
	DeletedAt       *time.Time `json:"deleted_at"`
	ReplyCount      int        `json:"reply_count"`
}

type commentFeedData struct {
	Comments   []feedComment `json:"comments"`
	NextCursor *int64        `json:"nextCursor"`
}

type commentFeedView struct {
	Data  commentFeedData
	Error string
}

type commentTypeMeta struct {
	Key, Label, Plural, Icon string
	ID                       int
}

var commentTypes = []commentTypeMeta{
	{"all", "All", "All", "network", 0},
	{"killmail", "Killmail", "Killmails", "swords", 1},
	{"character", "Character", "Characters", "users", 2},
	{"corporation", "Corporation", "Corporations", "building-2", 3},
	{"alliance", "Alliance", "Alliances", "flag", 4},
	{"system", "System", "Systems", "globe", 5},
	{"battle", "Battle", "Battles", "shield", 7},
	{"fit", "Fit", "Fits", "wrench", 8},
}

func commentMeta(target int) commentTypeMeta {
	for _, meta := range commentTypes {
		if meta.ID == target {
			return meta
		}
	}
	return commentTypeMeta{Key: "unknown", Label: "Page", Plural: "Pages", Icon: "link"}
}

func commentTarget(comment feedComment) (string, string) {
	id := strconv.FormatInt(comment.TargetID, 10)
	switch comment.TargetType {
	case 1:
		return "Killmail #" + id, "/kill/" + id + "#comment-" + strconv.FormatInt(comment.ID, 10)
	case 2:
		return "Character profile", "/character/" + id
	case 3:
		return "Corporation profile", "/corporation/" + id
	case 4:
		return "Alliance profile", "/alliance/" + id
	case 5:
		return "Solar system", "/system/" + id
	case 7:
		return "Battle #" + id, "/battle/" + id + "/comments#comment-" + strconv.FormatInt(comment.ID, 10)
	case 8:
		if comment.TargetSlug != "" {
			return "Fit " + comment.TargetSlug, "/fit/" + url.PathEscape(comment.TargetSlug) + "#comment-" + strconv.FormatInt(comment.ID, 10)
		}
	}
	return "Page", "#"
}
func commentTargetLabel(comment feedComment) string { label, _ := commentTarget(comment); return label }
func commentTargetURL(comment feedComment) string   { _, target := commentTarget(comment); return target }

func commentRelative(t time.Time) string {
	d := time.Since(t)
	if d < time.Minute {
		return "just now"
	}
	if d < time.Hour {
		return fmt.Sprintf("%dm ago", int(d.Minutes()))
	}
	if d < 24*time.Hour {
		return fmt.Sprintf("%dh ago", int(d.Hours()))
	}
	if d < 30*24*time.Hour {
		return fmt.Sprintf("%dd ago", int(d.Hours()/24))
	}
	return t.UTC().Format("2 Jan 2006")
}
func commentTime(t time.Time) string    { return t.UTC().Format("2 Jan 2006, 15:04:05 EVE") }
func commentSafeBody(raw string) string { return api.SafeCommentHTML(raw) }

func (s *site) readCommentFeed(r *http.Request, values url.Values) (commentFeedData, error) {
	request := r.Clone(r.Context())
	address := *r.URL
	address.Path = "/api/comments"
	address.RawPath = ""
	address.RawQuery = values.Encode()
	request.URL = &address
	request.Method = http.MethodGet
	request.RequestURI = ""
	w := httptest.NewRecorder()
	s.options.CommentsAPI.ServeHTTP(w, request)
	if w.Code != http.StatusOK {
		return commentFeedData{}, fmt.Errorf("comments API returned %d", w.Code)
	}
	var data commentFeedData
	if err := json.Unmarshal(w.Body.Bytes(), &data); err != nil {
		return commentFeedData{}, fmt.Errorf("decode comment feed: %w", err)
	}
	if data.Comments == nil {
		data.Comments = []feedComment{}
	}
	return data, nil
}

func (s *site) commentsPage(r *http.Request) (Page, error) {
	data, err := s.readCommentFeed(r, url.Values{"limit": {strconv.Itoa(commentFeedPageSize)}})
	view := commentFeedView{Data: data}
	if err != nil {
		view.Error = "Failed to load comments"
	}
	return Page{Metadata: Metadata{
		Title: "Comments — EVE-KILL", CanonicalPath: "/comments", Indexable: true,
		Description:  "Site-wide comment feed across killmails, characters, corporations, alliances, systems, and battles.",
		CommentMedia: true,
		Stylesheets:  []string{"comments-feed.css"}, ModuleScripts: []string{"comments-feed.js"},
	}, Content: commentsFeedPage(view)}, nil
}

func commentFilterValues(raw url.Values) (url.Values, error) {
	values := url.Values{"limit": {strconv.Itoa(commentFeedPageSize)}}
	if q := strings.TrimSpace(raw.Get("q")); len(q) >= 2 {
		if runes := []rune(q); len(runes) > 100 {
			q = string(runes[:100])
		}
		values.Set("q", q)
	}
	if target := raw.Get("target_type"); target != "" {
		id, err := strconv.Atoi(target)
		if err != nil || id == 0 || commentMeta(id).Key == "unknown" {
			return nil, fmt.Errorf("invalid comment target type")
		}
		values.Set("target_type", target)
	}
	if cursor := raw.Get("cursor"); cursor != "" {
		id, err := strconv.ParseInt(cursor, 10, 64)
		if err != nil || id <= 0 {
			return nil, fmt.Errorf("invalid comment cursor")
		}
		values.Set("cursor", cursor)
	}
	return values, nil
}

func (s *site) commentsFragment(w http.ResponseWriter, r *http.Request) {
	values, err := commentFilterValues(r.URL.Query())
	if err != nil {
		http.Error(w, "Invalid comment filters", http.StatusBadRequest)
		return
	}
	data, err := s.readCommentFeed(r, values)
	if err != nil {
		http.Error(w, "Could not load comments", http.StatusServiceUnavailable)
		return
	}
	if data.NextCursor != nil {
		w.Header().Set("X-Next-Cursor", strconv.FormatInt(*data.NextCursor, 10))
	}
	w.Header().Set("Cache-Control", "private, no-store")
	s.renderFragment(w, r, commentFeedFragmentContent(data.Comments))
}
