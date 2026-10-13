package frontend

import (
	"net/http"
	"net/url"
	"strings"
	"time"
)

type settingsCommentSpec struct {
	Label, Href, Icon, ChipBg, ChipText, EmptyLabel string
	NeedsSlug                                       bool
}
type settingsComment struct {
	feedComment
	DomainID   *int64     `json:"domain_id"`
	ParentID   *int64     `json:"parent_id"`
	EditedAt   *time.Time `json:"edited_at"`
	Moderation int        `json:"moderation_status"`
}
type settingsCommentsData struct {
	Comments   []settingsComment `json:"comments"`
	NextCursor *int64            `json:"nextCursor"`
}

func settingsCommentMeta(comment settingsComment) settingsCommentSpec {
	spec, ok := settingsData.CommentTargets[comment.TargetType]
	if !ok {
		spec = settingsData.CommentTargets[0]
	}
	if spec.NeedsSlug && comment.TargetSlug == "" {
		spec.Href = "#"
		if spec.EmptyLabel != "" {
			spec.Label = spec.EmptyLabel
		}
		return spec
	}
	raw := strings.NewReplacer("{id}", warID(comment.TargetID), "{comment}", warID(comment.ID), "{slug}", comment.TargetSlug, "{battle}", killBoolClass(comment.TargetSlug != "", comment.TargetSlug, warID(comment.TargetID)))
	spec.Label = raw.Replace(spec.Label)
	segments := strings.Split(strings.TrimLeft(comment.TargetSlug, "/"), "/")
	for i := range segments {
		segments[i] = url.PathEscape(segments[i])
	}
	path := strings.NewReplacer("{id}", warID(comment.TargetID), "{comment}", warID(comment.ID), "{slug}", strings.Join(segments, "/"), "{battle}", killBoolClass(comment.TargetSlug != "", url.PathEscape(comment.TargetSlug), warID(comment.TargetID)))
	spec.Href = path.Replace(spec.Href)
	return spec
}

func (s *site) settingsCommentsFragment(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Cache-Control", "private, no-store")
	user, err := s.settingsIdentity(r)
	if err != nil {
		http.Error(w, "Could not load your comments", http.StatusServiceUnavailable)
		return
	}
	if user == nil {
		http.Error(w, "Your session expired. Log in again to continue.", http.StatusUnauthorized)
		return
	}
	values := url.Values{"limit": {"25"}}
	if cursor := r.URL.Query().Get("cursor"); cursor != "" {
		values.Set("cursor", cursor)
	}
	var data settingsCommentsData
	if err := s.accountRead(r, "/api/user/comments?"+values.Encode(), &data); err != nil {
		http.Error(w, "Could not load your comments", http.StatusServiceUnavailable)
		return
	}
	w.Header().Set("X-Next-Cursor", cursorString(data.NextCursor))
	s.renderFragment(w, r, settingsCommentCards(data.Comments, time.Now()))
}
