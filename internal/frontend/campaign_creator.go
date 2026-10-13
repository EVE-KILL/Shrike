package frontend

import (
	"net/http"
	"time"

	"github.com/eve-kill/shrike/internal/api"
)

type campaignCreatorOption struct{ Key, Label, Description, Icon string }

var campaignCreatorModes = []campaignCreatorOption{
	{"conflict", "Conflict", "Track named sides and compare their kills, losses and efficiency.", "swords"},
	{"single", "One-sided", "Follow a single group — every kill and loss for the entities you pick.", "crosshair"},
	{"area", "Area activity", "Track every killmail in selected regions, constellations or systems, regardless of participants.", "map-pinned"},
}
var campaignCreatorVisibilities = []campaignCreatorOption{
	{"0", "Public", "Listed on EVE-KILL for everyone.", "globe"},
	{"1", "Private", "Only you, admins, and signed-in members of a custom domain you add it to can view it.", "lock"},
	{"2", "Killboard only", "Unlisted on EVE-KILL; shown on the custom killboards of the entities involved.", "shield"},
}
var campaignCreatorSideBorders = []string{"border-red-500/20", "border-blue-500/20", "border-emerald-500/20", "border-orange-500/20"}
var campaignCreatorSideTexts = []string{"text-red-400", "text-blue-400", "text-emerald-400", "text-orange-400"}
var campaignCreatorSideColors = []string{"#ef4444", "#3b82f6", "#059669", "#ea580c"}

func campaignCreatorIdentity(user *api.Principal) map[string]any {
	if user == nil {
		return nil
	}
	return map[string]any{"characterId": user.CharacterID, "characterName": user.CharacterName, "isAdmin": user.IsAdmin}
}

func (s *site) campaignCreatorPage(r *http.Request) (Page, error) {
	start := time.Now().UTC().Truncate(24 * time.Hour).Format("2006-01-02T15:04")
	user, err := s.browserSession(r)
	if err != nil {
		return Page{}, err
	}
	return Page{Metadata: Metadata{Title: "Campaign Creator", OGTitle: "Campaign Creator — EVE-KILL", Description: "Create a campaign on EVE-KILL: track a war between alliances, corporations or characters with live scoreboards, ISK stats and killmail feeds.", CanonicalPath: "/campaigncreator", Indexable: true, Stylesheets: []string{"entity.css", "utilities.css", "utility-page.css", "campaign-creator.css"}, ModuleScripts: []string{"campaign-creator.js"}, StructuredData: map[string]any{"@context": "https://schema.org", "@type": "BreadcrumbList", "itemListElement": []map[string]any{{"@type": "ListItem", "position": 1, "name": "Home", "item": canonicalOrigin + "/"}, {"@type": "ListItem", "position": 2, "name": "Campaign Creator", "item": canonicalOrigin + "/campaigncreator"}}}}, Content: campaignCreatorPage(start, user)}, nil
}

func campaignCreatorPreview(start string) api.EntityCampaignEntry {
	return api.EntityCampaignEntry{CampaignID: "preview", Name: "Untitled campaign", Status: 1, Mode: "conflict", StartTime: start + ":00Z", Sides: []api.EntityCampaignSide{{Name: "Side 1", SideIndex: 0}, {Name: "Side 2", SideIndex: 1}}}
}
