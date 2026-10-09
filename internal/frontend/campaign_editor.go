package frontend

import (
	"net/http"
	"strconv"
	"time"
)

type campaignEditEntity struct {
	Type string `json:"type"`
	ID   int64  `json:"id"`
	Name string `json:"name"`
}
type campaignEditSide struct {
	Name     string               `json:"name"`
	Entities []campaignEditEntity `json:"entities"`
}
type campaignEditView struct {
	Name, Description, EndTime, Mode string
	Visibility                       int
	Ongoing, Locked, HasPrize        bool
	Sides                            []campaignEditSide
	Allowed                          []campaignEditEntity
}

func campaignEditViewOf(data campaignDetailData) campaignEditView {
	view := campaignEditView{Name: data.Name, Description: data.Description, Mode: data.Mode, Visibility: data.Visibility, Ongoing: data.EndTime == "", HasPrize: data.PrizePool != nil, Allowed: data.AllowedEntities}
	if data.PrizePool != nil {
		view.Locked = data.PrizePool.RulesLockedAt != ""
	}
	if end, err := time.Parse(time.RFC3339, data.EndTime); err == nil {
		view.EndTime = end.UTC().Format("2006-01-02T15:04")
	}
	for _, side := range data.Sides {
		draft := campaignEditSide{Name: side.Name, Entities: []campaignEditEntity{}}
		for _, entity := range side.Entities {
			kind := "character"
			switch entity.EntityType {
			case 1:
				kind = "corporation"
			case 2:
				kind = "alliance"
			}
			draft.Entities = append(draft.Entities, campaignEditEntity{Type: kind, ID: entity.EntityID, Name: campaignDetailMemberName(entity)})
		}
		view.Sides = append(view.Sides, draft)
	}
	if view.Sides == nil {
		view.Sides = []campaignEditSide{}
	}
	if view.Allowed == nil {
		view.Allowed = []campaignEditEntity{}
	}
	for index := range view.Allowed {
		if view.Allowed[index].Name == "" {
			view.Allowed[index].Name = "#" + strconv.FormatInt(view.Allowed[index].ID, 10)
		}
	}
	return view
}

func (s *site) campaignEditFragment(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Cache-Control", "private, no-store")
	w.Header().Set("X-Robots-Tag", "noindex, nofollow")
	user, err := s.browserSession(r)
	if err != nil {
		http.Error(w, "Could not read the session", http.StatusServiceUnavailable)
		return
	}
	if user == nil {
		http.Error(w, "Authentication required", http.StatusUnauthorized)
		return
	}
	id := r.PathValue("id")
	if !validCampaignID(id) {
		http.NotFound(w, r)
		return
	}
	data, found, err := s.readCampaignDetail(r, id)
	if err != nil {
		http.Error(w, "Could not load campaign", http.StatusServiceUnavailable)
		return
	}
	if !found {
		http.NotFound(w, r)
		return
	}
	if !user.IsAdmin && int64(user.CharacterID) != data.Creator.CharacterID {
		http.Error(w, "Only the owner can edit this campaign", http.StatusForbidden)
		return
	}
	s.renderFragment(w, r, campaignEditorForm(campaignEditViewOf(data)))
}
