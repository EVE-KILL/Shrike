package frontend

import (
	"fmt"
	"math"
	"net/http"
	"strconv"
	"time"

	"github.com/eve-kill/shrike/internal/api"
)

type legacyItemGroup struct {
	Label string
	Items []api.LegacyArchiveItem
}

type legacyKillView struct {
	Data      api.LegacyArchiveKillData
	FinalBlow *api.LegacyArchiveAttacker
	Groups    []legacyItemGroup
}

func (s *site) loadLegacyKillPage(r *http.Request) (Page, error) {
	id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil || id <= 0 {
		return Page{}, ErrNotFound
	}
	data, found, err := s.options.Legacy.LegacyArchiveKill(r.Context(), id)
	if err != nil {
		return Page{}, err
	}
	if !found {
		return Page{}, ErrNotFound
	}
	view := legacyKillView{Data: data}
	for i := range data.Attackers {
		if data.Attackers[i].FinalBlow {
			view.FinalBlow = &data.Attackers[i]
			break
		}
	}
	if view.FinalBlow == nil && len(data.Attackers) > 0 {
		view.FinalBlow = &data.Attackers[0]
	}
	groups := []legacyItemGroup{{Label: "Fitted"}, {Label: "Cargo"}, {Label: "Drone Bay"}}
	for _, item := range data.Items {
		idx := 0
		if item.Location != nil {
			switch *item.Location {
			case "Cargo":
				idx = 1
			case "Drone Bay":
				idx = 2
			}
		}
		groups[idx].Items = append(groups[idx].Items, item)
	}
	for _, group := range groups {
		if len(group.Items) > 0 {
			view.Groups = append(view.Groups, group)
		}
	}
	kill := data.Kill
	ship := fallbackName(kill.VictimShip)
	who := fallbackName(kill.VictimName)
	title := fmt.Sprintf("%s | %s | %s ISK", ship, who, compactNumber(kill.TotalValue))
	image := ""
	if kill.VictimShipTypeID != nil {
		image = fmt.Sprintf("https://eve-kill.com/images/types/%d/overlayrender?size=512", *kill.VictimShipTypeID)
	}
	return Page{Metadata: Metadata{
		Title: title, OGTitle: title + " — EVE-KILL Legacy", CanonicalPath: r.URL.Path, Indexable: true,
		Description: fmt.Sprintf("%s lost a %s in %s", who, ship, kill.SystemName), Image: image,
		Stylesheets: []string{"legacy-kill.css"}, ModuleScripts: []string{"legacy-kill.js"},
	}, Content: legacyKillPage(view), SharedCacheTTL: time.Hour}, nil
}

func legacyID(id *int64) int64 {
	if id == nil {
		return 0
	}
	return *id
}
func legacyTypeImage(id *int64, variant string, size int) string {
	value := legacyID(id)
	if value == 0 {
		value = 670
	}
	return fmt.Sprintf("/images/types/%d/%s?size=%d", value, variant, size)
}
func legacyEntityImage(kind string, id *int64, size int) string {
	value := legacyID(id)
	if kind == "character" && value == 0 {
		value = 1
	}
	return eveEntityImage(kind, value, size)
}
func legacyKillTime(value time.Time) string { return value.UTC().Format("2 Jan 2006, 15:04 EVE") }
func legacySecurity(value *float64) string {
	if value == nil {
		return ""
	}
	return fmt.Sprintf("%.1f", math.Round(*value*10)/10)
}
func legacyAttackerRemaining(total int) string { return strconv.Itoa(total - 10) }

func legacyWheelStyle(group string, index int) string {
	angle := float64(index) * 10.5
	switch group {
	case "top":
		angle -= 125
	case "right":
		angle -= 37
	case "bottom":
		angle += 54
	case "left":
		angle += 196
	}
	radians := angle * math.Pi / 180
	x := math.Round((50+42*math.Cos(radians))*1000) / 1000
	y := math.Round((50+42*math.Sin(radians))*1000) / 1000
	return fmt.Sprintf("left:%.3f%%;top:%.3f%%", x, y)
}

func legacyAttackerClass(attacker api.LegacyArchiveAttacker, index int) string {
	if attacker.FinalBlow {
		return "legacy-attacker-row final-blow"
	}
	return "legacy-attacker-row"
}

func legacyAttackerTypeImage(id *int64) string {
	if id == nil || *id == 0 {
		return ""
	}
	return legacyTypeImage(id, "icon", 128)
}
