package frontend

import (
	"context"
	"encoding/json"
	"net/url"
	"sort"
	"strings"

	"github.com/eve-kill/shrike/internal/api"
)

type categoryGroup struct {
	Name, Icon, Image string
	Labels            []api.LabelEntry
}

type categoriesView struct {
	Groups     []categoryGroup
	LabelCount int
	MaxCount   string
}

var categoryOrder = []string{
	"Space", "Timezone", "Engagement", "Killmail Type", "Involvement",
	"Faction Warfare", "Value", "Value Bands", "Victim Category", "Victim Hull", "Technology",
}

var categoryVisuals = map[string]struct{ icon, image string }{
	"Space":           {"map", "/assets/category-space.webp"},
	"Timezone":        {"activity", "/images/types/19720/icon?size=64"},
	"Engagement":      {"swords", "/images/types/638/icon?size=64"},
	"Killmail Type":   {"shield", "/images/types/670/icon?size=64"},
	"Involvement":     {"users", "/images/types/11567/icon?size=64"},
	"Faction Warfare": {"shield", eveEntityImage("corporation", 1000180, 64)},
	"Value":           {"gem", "/images/types/34/icon?size=64"},
	"Value Bands":     {"gem", "/images/types/34/icon?size=64"},
	"Victim Category": {"tags", "/images/types/35832/icon?size=64"},
	"Victim Hull":     {"wrench", "/images/types/24694/icon?size=64"},
	"Technology":      {"settings", "/images/types/12005/icon?size=64"},
}

func loadCategories(ctx context.Context, reader HomepageReader) (categoriesView, error) {
	labels, err := reader.Labels(ctx)
	if err != nil {
		return categoriesView{}, err
	}
	groupsByName := make(map[string]*categoryGroup)
	var maxCount int64
	for _, label := range labels {
		group := groupsByName[label.Category]
		if group == nil {
			visual, exists := categoryVisuals[label.Category]
			if !exists {
				visual = struct{ icon, image string }{"tags", "/images/types/670/icon?size=64"}
			}
			group = &categoryGroup{Name: label.Category, Icon: visual.icon, Image: visual.image}
			groupsByName[label.Category] = group
		}
		group.Labels = append(group.Labels, label)
		if label.Count > maxCount {
			maxCount = label.Count
		}
	}
	view := categoriesView{LabelCount: len(labels), MaxCount: fullNumber(float64(maxCount))}
	for _, name := range categoryOrder {
		if group := groupsByName[name]; group != nil {
			view.Groups = append(view.Groups, *group)
			delete(groupsByName, name)
		}
	}
	var extra []string
	for name := range groupsByName {
		extra = append(extra, name)
	}
	sort.Strings(extra)
	for _, name := range extra {
		view.Groups = append(view.Groups, *groupsByName[name])
	}
	return view, nil
}

func categorySearchURL(label api.LabelEntry) string {
	if len(label.SearchFilters) == 0 {
		return ""
	}
	data, err := json.Marshal(label.SearchFilters)
	if err != nil {
		return ""
	}
	return "/advancedsearch?q=" + url.QueryEscape(string(data))
}

func categorySearchText(label api.LabelEntry) string {
	return strings.ToLower(label.Name + " " + label.ID + " " + label.Description + " " + label.Category)
}
