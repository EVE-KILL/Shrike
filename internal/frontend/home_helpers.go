package frontend

import (
	"fmt"
	"math"
	"strconv"
	"strings"
	"time"

	"github.com/eve-kill/shrike/internal/api"
)

func compactNumber(value float64) string {
	switch {
	case value >= 1e12:
		return fmt.Sprintf("%.2ft", value/1e12)
	case value >= 1e9:
		return fmt.Sprintf("%.2fb", value/1e9)
	case value >= 1e6:
		return fmt.Sprintf("%.2fm", value/1e6)
	case value >= 1e3:
		return fmt.Sprintf("%.1fk", value/1e3)
	default:
		return fmt.Sprintf("%.0f", value)
	}
}

func fullNumber(value float64) string {
	whole := strconv.FormatInt(int64(math.Round(value)), 10)
	start := 0
	if strings.HasPrefix(whole, "-") {
		start = 1
	}
	for i := len(whole) - 3; i > start; i -= 3 {
		whole = whole[:i] + "," + whole[i:]
	}
	return whole
}

func formatTopCount(format string, value float64) string {
	switch format {
	case "isk":
		return compactNumber(value)
	case "age":
		seconds := int64(time.Since(time.Unix(int64(value), 0)).Seconds())
		if seconds < 0 {
			seconds = 0
		}
		switch {
		case seconds < 60:
			return fmt.Sprintf("%ds", seconds)
		case seconds < 3600:
			return fmt.Sprintf("%dm", seconds/60)
		case seconds < 86400:
			return fmt.Sprintf("%dh", seconds/3600)
		case seconds < 30*86400:
			return fmt.Sprintf("%dd", seconds/86400)
		case seconds < 365*86400:
			return fmt.Sprintf("%dmo", seconds/(30*86400))
		default:
			return fmt.Sprintf("%dy", seconds/(365*86400))
		}
	default:
		return fullNumber(value)
	}
}

func topEntryPath(kind string, id int64) string {
	prefix := map[string]string{
		"characters": "character", "corporations": "corporation", "alliances": "alliance",
		"factions": "faction", "ships": "item", "systems": "system", "regions": "region", "constellations": "constellation",
	}[kind]
	if prefix == "" || id <= 0 {
		return "#"
	}
	return fmt.Sprintf("/%s/%d", prefix, id)
}

func topEntryImage(kind string, id int64) string {
	if id <= 0 {
		return ""
	}
	switch kind {
	case "characters":
		return eveEntityImage("character", id, 64)
	case "corporations", "factions":
		return eveEntityImage("corporation", id, 64)
	case "alliances":
		return eveEntityImage("alliance", id, 64)
	case "ships":
		return fmt.Sprintf("/images/types/%d/icon?size=64", id)
	case "systems":
		return fmt.Sprintf("/images/systems/%d?size=32", id)
	case "regions":
		return fmt.Sprintf("/images/regions/%d?size=32", id)
	default:
		return ""
	}
}

func topAccentColor(entry api.TopEntry) string {
	return paletteAccent(entry.Palette)
}

func valuableImage(id int64) string {
	return fmt.Sprintf("/images/types/%d/overlayrender?size=256", id)
}

func idPath(kind string, id int64) string {
	if id <= 0 {
		return "#"
	}
	return fmt.Sprintf("/%s/%d", kind, id)
}

func killDetailPath(kind string, id int64) string {
	if kind == "legacy" {
		return idPath("legacy/kill", id)
	}
	return idPath("kill", id)
}

func killImage(kind string, id int64) string {
	if id <= 0 {
		return ""
	}
	switch kind {
	case "item":
		return fmt.Sprintf("/images/types/%d/icon?size=64", id)
	case "character":
		return eveEntityImage(kind, id, 64)
	case "corporation":
		return eveEntityImage(kind, id, 32)
	case "alliance":
		return eveEntityImage(kind, id, 32)
	case "system":
		return fmt.Sprintf("/images/systems/%d?size=32", id)
	}
	return ""
}

func eveEntityImage(kind string, id int64, size int) string {
	if id <= 0 {
		return ""
	}
	switch kind {
	case "character":
		return fmt.Sprintf("https://images.evetech.net/characters/%d/portrait?size=%d", id, size)
	case "corporation":
		return fmt.Sprintf("https://images.evetech.net/corporations/%d/logo?size=%d", id, size)
	case "alliance":
		return fmt.Sprintf("https://images.evetech.net/alliances/%d/logo?size=%d", id, size)
	default:
		return ""
	}
}

func securityClass(value *float64) string {
	if value == nil {
		return "security-unknown"
	}
	switch {
	case *value >= 0.5:
		return "security-high"
	case *value > 0:
		return "security-low"
	default:
		return "security-null"
	}
}

func securityLabel(value *float64) string {
	if value == nil {
		return "?"
	}
	return fmt.Sprintf("%.1f", *value)
}

func sameUTCDay(left, right time.Time) bool {
	return left.UTC().Format("2006-01-02") == right.UTC().Format("2006-01-02")
}

func dayLabel(value time.Time) string {
	value = value.UTC()
	now := time.Now().UTC()
	day := value.Day()
	suffix := "th"
	if day%100 < 11 || day%100 > 13 {
		switch day % 10 {
		case 1:
			suffix = "st"
		case 2:
			suffix = "nd"
		case 3:
			suffix = "rd"
		}
	}
	date := fmt.Sprintf("%d%s %s", day, suffix, value.Format("January"))
	if value.Year() != now.Year() {
		date += " " + strconv.Itoa(value.Year())
	}
	if sameUTCDay(value, now) {
		return "Today (" + date + ")"
	}
	if sameUTCDay(value, now.AddDate(0, 0, -1)) {
		return "Yesterday (" + date + ")"
	}
	return date
}

func killTime(value time.Time) string {
	minutes := int(time.Since(value).Minutes())
	if minutes < 1 {
		return "just now"
	}
	if minutes < 30 {
		return fmt.Sprintf("%dm ago", minutes)
	}
	return value.UTC().Format("15:04")
}

type killIdentityView struct {
	PrimaryType, PrimaryName string
	PrimaryID                int64
	Portrait                 killPortraitView
	CorporationID            int64
	CorporationName          string
	AllianceID               int64
	AllianceName             string
	NPC                      bool
}

// eveDate matches the shared Nuxt formatEveDate helper, including UTC dates.
func eveDate(raw string) string {
	value, err := time.Parse(time.RFC3339, raw)
	if err != nil {
		value, err = time.Parse(time.DateOnly, raw)
	}
	if err != nil {
		return ""
	}
	return value.UTC().Format("2 Jan 2006")
}

type killPortraitView struct {
	Type, Name, Placeholder string
	ID                      int64
}

func victimIdentity(row api.KilllistRow) killIdentityView {
	identity := killIdentityView{
		CorporationID: row.VictimCorporationID, CorporationName: row.VictimCorporationName,
		AllianceID: row.VictimAllianceID, AllianceName: row.VictimAllianceName,
	}
	if row.VictimCharacterID > 0 {
		identity.PrimaryType, identity.PrimaryID, identity.PrimaryName = "character", row.VictimCharacterID, row.VictimCharacterName
	} else {
		identity.PrimaryType, identity.PrimaryID, identity.PrimaryName = "corporation", row.VictimCorporationID, row.VictimCorporationName
	}
	identity.Portrait = killPortraitView{Type: identity.PrimaryType, Name: identity.PrimaryName, Placeholder: "image", ID: identity.PrimaryID}
	return identity
}

func finalBlowIdentity(row api.KilllistRow) killIdentityView {
	identity := killIdentityView{
		CorporationID: row.FinalBlowCorporationID, CorporationName: row.FinalBlowCorporationName,
		AllianceID: row.FinalBlowAllianceID, AllianceName: row.FinalBlowAllianceName,
	}
	switch {
	case row.FinalBlowCharacterID > 0:
		identity.PrimaryType, identity.PrimaryID, identity.PrimaryName = "character", row.FinalBlowCharacterID, row.FinalBlowCharacterName
	case row.FinalBlowCorporationID > 0:
		identity.PrimaryType, identity.PrimaryID, identity.PrimaryName = "corporation", row.FinalBlowCorporationID, row.FinalBlowCorporationName
	case row.FinalBlowShipTypeID > 0:
		identity.PrimaryType, identity.PrimaryID, identity.PrimaryName = "item", row.FinalBlowShipTypeID, row.FinalBlowShipName
		identity.NPC = true
	default:
		identity.PrimaryName = "NPC"
		identity.NPC = true
	}
	// Nuxt chooses the final-blow portrait independently of its display name:
	// an NPC type icon can accompany a corporation's name and entity badges.
	identity.Portrait.Placeholder = "crosshair"
	if row.FinalBlowCharacterID > 0 {
		identity.Portrait.Type, identity.Portrait.ID, identity.Portrait.Name = "character", row.FinalBlowCharacterID, row.FinalBlowCharacterName
	} else if row.FinalBlowShipTypeID > 0 {
		identity.Portrait.Type, identity.Portrait.ID, identity.Portrait.Name = "item", row.FinalBlowShipTypeID, row.FinalBlowShipName
	}
	return identity
}

type killListOptions struct {
	Kind              string
	Page, Limit       int
	Live, QuickFilter bool
	Filtered          bool
	WarSides          *warSideIDs
	LossCharacters    map[int64]bool
	LossCorporations  map[int64]bool
	LossAlliances     map[int64]bool
}

func killListRowClass(kill api.KilllistRow, index int, options killListOptions) string {
	class := warKillRowClass(kill, index, options.WarSides)
	if options.LossCharacters[kill.VictimCharacterID] || options.LossCorporations[kill.VictimCorporationID] || options.LossAlliances[kill.VictimAllianceID] {
		class += " loss-row loss-row-hover"
	}
	return class
}

func cursorString(value *int64) string {
	if value == nil {
		return ""
	}
	return strconv.FormatInt(*value, 10)
}

func pageSlots(current, total int) []int {
	if total <= 0 {
		return nil
	}
	if total <= 7 {
		pages := make([]int, total)
		for index := range pages {
			pages[index] = index + 1
		}
		return pages
	}
	pages := []int{1}
	if current > 4 {
		pages = append(pages, 0)
	}
	for page := max(2, current-1); page <= min(total-1, current+1); page++ {
		pages = append(pages, page)
	}
	if current < total-3 {
		pages = append(pages, 0)
	}
	pages = append(pages, total)
	return pages
}

func killPageClass(value, active int) string {
	if value == active {
		return "kill-page-number active"
	}
	return "kill-page-number"
}

func formatPageNumber(value, active int) string {
	if value == active {
		return fullNumber(float64(value))
	}
	if value < 1000 {
		return strconv.Itoa(value)
	}
	if value < 1_000_000 {
		return strings.TrimSuffix(fmt.Sprintf("%.1f", float64(value)/1000), ".0") + "K"
	}
	return strings.TrimSuffix(fmt.Sprintf("%.1f", float64(value)/1_000_000), ".0") + "M"
}

type filterOption struct{ Key, Label string }

func filterLabel(kind string) string {
	switch kind {
	case "region":
		return "Region"
	case "system":
		return "System"
	case "ship":
		return "Ship"
	default:
		return kind
	}
}

func valuableTabClass(active, tab string) string {
	if active == tab {
		return "valuable-tab active"
	}
	return "valuable-tab"
}

func valuableGridClass(compact bool) string {
	if compact {
		return "valuable-grid compact"
	}
	return "valuable-grid"
}

func valuableCardClass(index int, compact bool) string {
	if !compact && index >= 6 {
		return "valuable-card valuable-card-secondary"
	}
	return "valuable-card"
}

func valuableLoading(index int) string {
	if index < 6 {
		return "eager"
	}
	return "lazy"
}

func valuablePriority(index int) string {
	if index == 0 {
		return "high"
	}
	return "auto"
}

func killLimitClass(value, active int) string {
	if value == active {
		return "kill-limit active"
	}
	return "kill-limit"
}

func killRowClass(index int) string {
	if index%2 == 0 {
		return "kill-row alternate"
	}
	return "kill-row"
}

func identityPrimaryClass(npc bool) string {
	if npc {
		return "identity-primary npc"
	}
	return "identity-primary"
}

func fallbackName(names ...string) string {
	for _, name := range names {
		if name != "" {
			return name
		}
	}
	return "Unknown"
}
