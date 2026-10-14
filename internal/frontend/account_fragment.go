package frontend

import (
	"net/http"
	"strconv"
)

// Only browser display/preferences data crosses this boundary. Session IDs,
// ESI tokens, owner hashes and account metadata never enter the fragment.
type navbarUser struct {
	CharacterID            int64          `json:"characterId"`
	CharacterName          string         `json:"characterName"`
	IsAdmin                bool           `json:"isAdmin"`
	CorporationID          *int64         `json:"corporationId"`
	CorporationName        *string        `json:"corporationName"`
	AllianceID             *int64         `json:"allianceId"`
	AllianceName           *string        `json:"allianceName"`
	LastSeenNotificationID int64          `json:"lastSeenNotificationId"`
	Settings               navbarSettings `json:"settings"`
}

type navbarSettings struct {
	Theme       map[string]any `json:"theme,omitempty"`
	DefaultTabs map[string]any `json:"defaultTabs,omitempty"`
	Boards      *struct {
		Dismissed []string `json:"dismissed"`
		Pinned    []string `json:"pinned"`
	} `json:"boards,omitempty"`
}

func (s *site) accountFragment(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Cache-Control", "private, no-store")
	w.Header().Set("X-Robots-Tag", "noindex, nofollow")
	var data struct{ User *navbarUser }
	headers, err := s.accountReadHeaders(r, "/auth/me", &data)
	for _, cookie := range headers.Values("Set-Cookie") {
		w.Header().Add("Set-Cookie", cookie)
	}
	if err != nil {
		http.Error(w, "Unable to load account. Please try again.", http.StatusServiceUnavailable)
		return
	}
	s.renderFragment(w, r, accountFragmentContent(data.User))
}

func accountCharacterPath(user *navbarUser) string {
	return "/character/" + strconv.FormatInt(user.CharacterID, 10)
}

func accountPortrait(user *navbarUser, size int) string {
	return "https://images.evetech.net/characters/" + strconv.FormatInt(user.CharacterID, 10) + "/portrait?size=" + strconv.Itoa(size)
}

type navbarEntity struct{ Kind, Path, Image, Name string }

func accountEntities(user *navbarUser) []navbarEntity {
	var entities []navbarEntity
	for _, entity := range []struct {
		kind string
		id   *int64
		name *string
	}{{"corporation", user.CorporationID, user.CorporationName}, {"alliance", user.AllianceID, user.AllianceName}} {
		if entity.id == nil || *entity.id <= 0 {
			continue
		}
		name := ""
		if entity.name != nil {
			name = *entity.name
		}
		id := strconv.FormatInt(*entity.id, 10)
		entities = append(entities, navbarEntity{entity.kind, "/" + entity.kind + "/" + id, "https://images.evetech.net/" + entity.kind + "s/" + id + "/logo?size=32", name})
	}
	return entities
}

func accountTriggerLabel(user *navbarUser) string {
	if user != nil {
		return "Account"
	}
	return "Login with EVE Online"
}
