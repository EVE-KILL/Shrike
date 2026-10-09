package frontend

import (
	"math"
	"net/url"
	"strconv"
	"time"

	"github.com/eve-kill/shrike/internal/api"
)

type adminModerationItem struct {
	ID              int64        `json:"id"`
	Kind            int          `json:"target_kind"`
	TargetID        int64        `json:"target_id"`
	Body            string       `json:"body"`
	Format          *string      `json:"body_format"`
	HTML            *string      `json:"rendered_html"`
	CharacterID     int64        `json:"character_id"`
	CharacterName   string       `json:"character_name"`
	CorporationName string       `json:"corporation_name"`
	Status          int          `json:"status"`
	Submitted       *time.Time   `json:"submitted_at"`
	Reviewed        *time.Time   `json:"reviewed_at"`
	Reviewer        *int64       `json:"reviewed_by"`
	Notes           *string      `json:"review_notes"`
	AIAction        *string      `json:"ai_action"`
	AICategory      *string      `json:"ai_category"`
	AIScore         *float64     `json:"ai_max_score"`
	Context         *feedComment `json:"comment_context"`
}
type adminReviewQuery struct{ Kind, Status, Filter string }
type adminModerationData struct {
	Items  []adminModerationItem `json:"items"`
	Counts struct {
		Pending  int `json:"pending"`
		Comments int `json:"pending_comments"`
		Bios     int `json:"pending_bios"`
		Total    int `json:"total"`
	} `json:"counts"`
	NextCursor *int64           `json:"nextCursor"`
	Query      adminReviewQuery `json:"-"`
	Failed     bool             `json:"-"`
}
type adminComment struct {
	settingsComment
	Flagged bool `json:"flagged"`
	Reports int  `json:"reports_count"`
}
type adminCommentsData struct {
	Comments []adminComment   `json:"comments"`
	Query    adminReviewQuery `json:"-"`
	Failed   bool             `json:"-"`
}

func adminReviewSelection(query url.Values) adminReviewQuery {
	result := adminReviewQuery{Kind: "all", Status: "pending", Filter: "flagged"}
	if kind := query.Get("kind"); kind == "comments" || kind == "bios" {
		result.Kind = kind
	}
	if status := query.Get("status"); status == "approved" || status == "rejected" || status == "all" {
		result.Status = status
	}
	if filter := query.Get("filter"); filter == "reported" || filter == "all" {
		result.Filter = filter
	}
	return result
}
func (query adminReviewQuery) values(section string) url.Values {
	if section == "comments" {
		return url.Values{"filter": {query.Filter}, "limit": {"100"}}
	}
	return url.Values{"kind": {query.Kind}, "status": {query.Status}, "limit": {"100"}}
}
func adminCommentMeta(comment adminComment) settingsCommentSpec {
	meta := settingsCommentMeta(comment.settingsComment)
	id := warID(comment.TargetID)
	switch comment.TargetType {
	case 1:
		meta.Label = "Kill #" + id
	case 2:
		meta.Label = "Char " + id
	case 3:
		meta.Label = "Corp " + id
	case 4:
		meta.Label = "Alli " + id
	case 5:
		meta.Label = "Sys " + id
	case 7:
		meta.Label = "Battle " + killBoolClass(comment.TargetSlug != "", comment.TargetSlug, id)
	case 8:
		meta.Label = "Fit " + domainDefault(comment.TargetSlug, "?")
	case 9:
		meta.Label = "Blog post"
	case 10:
		meta.Label = "Campaign"
	default:
		meta.Label = "Page"
	}
	if meta.Href != "#" && (comment.TargetType == 6 || comment.TargetType == 8) {
		meta.Href += "#comment-" + warID(comment.ID)
	}
	return meta
}
func adminModerationKind(kind int) string {
	if kind >= 0 && kind < 4 {
		return []string{"Comment", "Character bio", "Corporation bio", "Alliance bio"}[kind]
	}
	return "Unknown"
}
func (item adminModerationItem) target() (string, string) {
	id := warID(item.TargetID)
	if item.Kind == 0 {
		if item.Context == nil {
			return id, "/comments"
		}
		comment := adminComment{settingsComment: settingsComment{feedComment: *item.Context}}
		comment.ID = item.TargetID
		meta := adminCommentMeta(comment)
		href := meta.Href
		if href == "#" {
			href = "/comments"
		}
		return killBoolClass(comment.TargetSlug != "", comment.TargetSlug, warID(comment.TargetID)), href
	}
	if item.Kind > 0 && item.Kind < 4 {
		return id, "/" + []string{"", "character", "corporation", "alliance"}[item.Kind] + "/" + id
	}
	return id, "#"
}
func (item adminModerationItem) body() string {
	if item.HTML == nil {
		return ""
	}
	if item.Kind == 0 {
		return commentSafeBody(*item.HTML)
	}
	return api.SafeAccountBioHTML(*item.HTML)
}
func (item adminModerationItem) ai() string {
	if item.AIAction == nil || *item.AIAction == "" {
		return ""
	}
	value := "AI: " + *item.AIAction
	if item.AICategory != nil && *item.AICategory != "" {
		value += " (" + *item.AICategory + ")"
	}
	if item.AIScore != nil {
		value += " · " + strconv.FormatFloat(math.Round(*item.AIScore*100), 'f', 0, 64) + "%"
	}
	return value
}
func adminModerationStatus(status int) (string, string) {
	labels := []string{"Pending", "Approved", "Rejected", "Auto-approved", "Auto-rejected"}
	tones := []string{"bg-yellow-500/20 text-yellow-400 border-yellow-500/30", "bg-green-500/20 text-green-300 border-green-500/30", "bg-red-500/20 text-red-300 border-red-500/30", "bg-green-500/10 text-green-400/70 border-green-500/20", "bg-red-500/10 text-red-400/70 border-red-500/20"}
	if status >= 0 && status < len(labels) {
		return labels[status], tones[status]
	}
	return strconv.Itoa(status), "bg-white/[0.04] text-gray-400"
}
func adminCommentStatus(status int) (string, string) {
	if status >= 0 && status < 4 {
		return []string{"OK", "Pending", "AI flagged", "Hidden"}[status], []string{"text-gray-400", "text-yellow-400", "text-yellow-400", "text-red-400"}[status]
	}
	return "?", "text-gray-500"
}
