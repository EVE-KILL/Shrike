package frontend

import (
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/eve-kill/shrike/internal/api"
)

type entityMembersView struct {
	Kind, Sort string
	ID         int64
	Activity   int
	CorpID     int64
	Data       api.EntityMembersData
	Corps      api.EntityCorporationsData
}

type entityCorporationsView struct {
	ID        int64
	Sort, Dir string
	Data      api.EntityCorporationsData
}

func (s *site) entityMembersFragment(w http.ResponseWriter, r *http.Request) {
	kind := r.PathValue("kind")
	if kind != "corporation" && kind != "alliance" {
		http.NotFound(w, r)
		return
	}
	id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil || id < 1 || id > 2_147_483_647 {
		http.NotFound(w, r)
		return
	}
	page, err := fragmentInt(r.URL.Query().Get("page"), 1, 1, 1_000_000)
	if err != nil {
		http.Error(w, "Invalid page", http.StatusBadRequest)
		return
	}
	activity, err := fragmentInt(r.URL.Query().Get("activity"), 0, 0, 90)
	if err != nil || activity != 0 && activity != 7 && activity != 30 && activity != 90 {
		http.Error(w, "Invalid activity filter", http.StatusBadRequest)
		return
	}
	sortBy := r.URL.Query().Get("sort")
	if sortBy == "" {
		sortBy = "name"
	}
	if sortBy != "name" && sortBy != "last_active" && sortBy != "security_status" {
		http.Error(w, "Invalid sort", http.StatusBadRequest)
		return
	}
	var corpID int64
	if raw := r.URL.Query().Get("corporation_id"); raw != "" {
		corpID, err = strconv.ParseInt(raw, 10, 64)
		if err != nil || corpID < 1 || corpID > 2_147_483_647 || kind != "alliance" {
			http.Error(w, "Invalid corporation filter", http.StatusBadRequest)
			return
		}
	}
	data, err := s.options.Entities.EntityMembers(r.Context(), kind, id, page, sortBy, activity, corpID)
	if err != nil {
		http.Error(w, "Could not load members", http.StatusServiceUnavailable)
		return
	}
	view := entityMembersView{Kind: kind, ID: id, Sort: sortBy, Activity: activity, CorpID: corpID, Data: data}
	if kind == "alliance" {
		view.Corps, err = s.options.Entities.EntityCorporations(r.Context(), id, "members", "desc")
		if err != nil {
			http.Error(w, "Could not load corporations", http.StatusServiceUnavailable)
			return
		}
	}
	s.renderFragment(w, r, entityMembers(view))
}

func (s *site) entityCorporationsFragment(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil || id < 1 || id > 2_147_483_647 {
		http.NotFound(w, r)
		return
	}
	sortBy := r.URL.Query().Get("sort")
	if sortBy == "" {
		sortBy = "members"
	}
	dir := r.URL.Query().Get("dir")
	if dir == "" {
		dir = "desc"
	}
	if (sortBy != "name" && sortBy != "members") || (dir != "asc" && dir != "desc") {
		http.Error(w, "Invalid corporation sort", http.StatusBadRequest)
		return
	}
	data, err := s.options.Entities.EntityCorporations(r.Context(), id, sortBy, dir)
	if err != nil {
		http.Error(w, "Could not load corporations", http.StatusServiceUnavailable)
		return
	}
	s.renderFragment(w, r, entityCorporations(entityCorporationsView{ID: id, Sort: sortBy, Dir: dir, Data: data}))
}

func entityMembersPages(page int, total int64) []int {
	count := int((total + 99) / 100)
	if count <= 1 {
		return nil
	}
	pages := []int{1}
	if page > 3 {
		pages = append(pages, 0)
	}
	for p := max(2, page-1); p <= min(count-1, page+1); p++ {
		pages = append(pages, p)
	}
	if page < count-2 {
		pages = append(pages, 0)
	}
	return append(pages, count)
}

func entitySelectedCorp(corps []api.EntityCorporation, id int64) string {
	for _, corp := range corps {
		if corp.CorporationID == id {
			return corp.Name
		}
	}
	return "All Corporations"
}

func entitySelectedCorpTrigger(corps []api.EntityCorporation, id int64) string {
	if id == 0 {
		return "All Corps"
	}
	for _, corp := range corps {
		if corp.CorporationID == id {
			return corp.Name + " [" + corp.Ticker + "]"
		}
	}
	return "All Corps"
}

func entityMemberSecurityClass(value float64) string {
	switch {
	case value >= 5:
		return "sec-cyan"
	case value >= 3:
		return "sec-green"
	case value >= 1:
		return "sec-lime"
	case value >= 0:
		return "sec-yellow"
	case value >= -2:
		return "sec-orange"
	case value >= -5:
		return "sec-red"
	default:
		return "sec-dark-red"
	}
}

func entityMemberSecurity(value float64) string { return fmt.Sprintf("%.1f", value) }

func entityMemberTimeAgo(raw string) string {
	value, err := time.Parse(time.RFC3339, raw)
	if err != nil {
		return ""
	}
	minutes := max(int(time.Since(value).Minutes()), 0)
	hours, days := minutes/60, minutes/(60*24)
	switch {
	case minutes < 60:
		return strconv.Itoa(minutes) + "m ago"
	case hours < 24:
		return strconv.Itoa(hours) + "h ago"
	case days < 30:
		return strconv.Itoa(days) + "d ago"
	case days < 365:
		return strconv.Itoa(days/30) + "mo ago"
	default:
		return strconv.Itoa(days/365) + "y ago"
	}
}

func entityCorpAccent(palette *api.TopPalette) string {
	if accent := paletteAccent(palette); accent != "" {
		return "box-shadow:inset 3px 0 0 " + accent
	}
	return ""
}

func entityCorpSortIcon(sortBy, dir, column string) string {
	if sortBy != column {
		return "↕"
	}
	if strings.EqualFold(dir, "asc") {
		return "↑"
	}
	return "↓"
}
