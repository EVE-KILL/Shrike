package frontend

import (
	"net/http"
	"net/url"
	"strconv"
	"time"

	"github.com/eve-kill/shrike/internal/api"
)

type entityTracker struct {
	ID            int64      `json:"id"`
	Type          string     `json:"target_type"`
	TargetID      int64      `json:"target_id"`
	Name          string     `json:"target_name"`
	Ticker        *string    `json:"target_ticker"`
	Enabled       bool       `json:"enabled"`
	Notifications bool       `json:"notifications_enabled"`
	CreatedAt     time.Time  `json:"created_at"`
	UpdatedAt     time.Time  `json:"updated_at"`
	EventCount    int64      `json:"event_count"`
	LastEvent     *time.Time `json:"last_event_at"`
	Unread        int64      `json:"unread_notifications"`
}
type trackersView struct {
	User     *settingsUser
	Trackers []entityTracker
	Limit    int
	Error    bool
}

func trackerIcon(kind string) string {
	switch kind {
	case "character":
		return "user"
	case "corporation":
		return "building-2"
	case "alliance":
		return "flag"
	case "region":
		return "map"
	case "constellation":
		return "orbit"
	default:
		return "map-pin"
	}
}
func trackerPicture(tracker entityTracker) string {
	switch tracker.Type {
	case "character", "corporation", "alliance":
		return killImage(tracker.Type, tracker.TargetID)
	}
	return ""
}
func (s *site) trackersPage(r *http.Request) (Page, error) {
	user, err := s.settingsIdentity(r)
	if err != nil {
		return Page{}, err
	}
	view := trackersView{User: user, Limit: 50}
	if user != nil {
		var data struct {
			Trackers []entityTracker `json:"trackers"`
			Limit    int             `json:"limit"`
		}
		if err := s.accountRead(r, "/api/me/trackers", &data); err != nil {
			view.Error = true
		} else {
			view.Trackers = data.Trackers
			view.Limit = data.Limit
		}
	}
	return Page{Metadata: Metadata{Title: "Entity Trackers", CanonicalPath: "/trackers", Stylesheets: []string{"utility-page.css", "trackers.css"}, ModuleScripts: []string{"trackers.js"}}, Content: trackersPageContent(view)}, nil
}

// Private feeds render the same rows as every other killlist, with the API's
// account ownership checks and cursor pagination preserved in-process.
func (s *site) trackerKilllistFragment(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Cache-Control", "private, no-store")
	id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil || id <= 0 {
		http.NotFound(w, r)
		return
	}
	q := r.URL.Query()
	limit, err := fragmentInt(q.Get("limit"), 50, 10, 100)
	if err != nil {
		http.Error(w, "Invalid limit", 400)
		return
	}
	page, err := fragmentInt(q.Get("page"), 1, 1, 100000)
	if err != nil {
		http.Error(w, "Invalid page", 400)
		return
	}
	params := url.Values{"limit": {strconv.Itoa(limit)}}
	if after := q.Get("after"); after != "" {
		params.Set("after", after)
	}
	var kills api.KilllistPage
	if err := s.accountRead(r, "/api/me/trackers/"+strconv.FormatInt(id, 10)+"/killmails?"+params.Encode(), &kills); err != nil {
		s.trackerFragmentError(w, err)
		return
	}
	options := killListOptions{Kind: "tracker", Page: page, Limit: limit, Live: q.Get("live") == "1"}
	targetID, _ := strconv.ParseInt(q.Get("lossId"), 10, 64)
	if targetID > 0 {
		switch q.Get("lossType") {
		case "character":
			options.LossCharacters = map[int64]bool{targetID: true}
		case "corporation":
			options.LossCorporations = map[int64]bool{targetID: true}
		case "alliance":
			options.LossAlliances = map[int64]bool{targetID: true}
		}
	}
	s.renderFragment(w, r, killList(kills, options))
}
func (s *site) trackerFragmentError(w http.ResponseWriter, err error) {
	status := http.StatusServiceUnavailable
	if failure, ok := err.(accountPageError); ok {
		status = failure.Status
	}
	http.Error(w, "Could not load recorded activity", status)
}

func trackerUnreadLabel(value int64) string {
	if value > 99 {
		return "99+"
	}
	return strconv.FormatInt(value, 10)
}
