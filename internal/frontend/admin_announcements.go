package frontend

import (
	"net/url"
	"time"
)

type adminAnnouncement struct {
	ID        int64      `json:"id"`
	Tier      int        `json:"tier"`
	Title     string     `json:"title"`
	Body      string     `json:"body_md"`
	Color     string     `json:"color"`
	Icon      *string    `json:"icon"`
	Link      *string    `json:"link_url"`
	LinkLabel *string    `json:"link_label"`
	Starts    time.Time  `json:"starts_at"`
	Expires   time.Time  `json:"expires_at"`
	Archived  *time.Time `json:"archived_at"`
}
type adminAnnouncementData struct {
	Rows   []adminAnnouncement `json:"announcements"`
	Status string              `json:"-"`
	Failed bool                `json:"-"`
}

func adminAnnouncementTierLabel(tier int) string {
	return map[int]string{1: "Ticker", 2: "Banner", 3: "Modal"}[tier]
}
func adminAnnouncementTierTone(tier int) string {
	return map[int]string{1: "text-blue-400", 2: "text-yellow-400", 3: "text-red-400"}[tier]
}

func adminAnnouncementStatus(query url.Values) string {
	switch value := query.Get("status"); value {
	case "all", "scheduled", "expired", "archived":
		return value
	default:
		return "active"
	}
}
func (a adminAnnouncement) status(now time.Time) (string, string) {
	if a.Archived != nil {
		return "Archived", "bg-gray-500/20 text-gray-400 border-gray-500/30"
	}
	if now.Before(a.Starts) {
		return "Scheduled", "bg-blue-500/20 text-blue-300 border-blue-500/30"
	}
	if now.After(a.Expires) {
		return "Expired", "bg-yellow-500/20 text-yellow-400 border-yellow-500/30"
	}
	return "Active", "bg-green-500/20 text-green-300 border-green-500/30"
}
