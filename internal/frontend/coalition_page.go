package frontend

import (
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/eve-kill/shrike/internal/api"
)

func (s *site) coalitionDirectoryPage(r *http.Request) (Page, error) {
	data, err := s.options.Coalitions.CoalitionDirectoryOverview(r.Context())
	if err != nil {
		return Page{}, err
	}
	return Page{Metadata: Metadata{
		Title: "EVE Online Coalitions", OGTitle: "EVE Online Coalitions — EVE-KILL", CanonicalPath: "/coalitions", Indexable: true,
		Description:    "Browse community-maintained EVE Online coalitions, their member alliances, territory, membership, and recent combat activity.",
		StructuredData: map[string]any{"@context": "https://schema.org", "@type": "CollectionPage", "name": "EVE Online Coalitions", "url": "https://eve-kill.com/coalitions"},
		Stylesheets:    []string{"coalitions.css"}, ModuleScripts: []string{"coalitions.js"},
	}, Content: coalitionDirectory(data)}, nil
}

func (s *site) coalitionDirectoryFragment(w http.ResponseWriter, r *http.Request) {
	data, err := s.options.Coalitions.CoalitionDirectory(r.Context())
	if err != nil {
		http.Error(w, "Could not load coalitions", http.StatusServiceUnavailable)
		return
	}
	s.renderFragment(w, r, coalitionDirectoryActivity(data))
}

func (s *site) coalitionDetailPage(r *http.Request) (Page, error) {
	slug := r.PathValue("slug")
	data, found, err := s.options.Coalitions.CoalitionDetail(r.Context(), slug)
	if err != nil {
		return Page{}, err
	}
	if !found {
		return Page{}, ErrNotFound
	}
	name := data.Coalition.Name
	return Page{Metadata: Metadata{
		Title: name + " Coalition", OGTitle: name + " — EVE-KILL", CanonicalPath: "/coalitions/" + slug, Indexable: true,
		Description: fmt.Sprintf("%s: %d alliances, %s members, territory, and recent combat statistics.", name, data.Coalition.AllianceCount, coalitionNumber(data.Coalition.MemberCount)),
		Stylesheets: []string{"coalitions.css"}, ModuleScripts: []string{"coalitions.js"},
	}, Content: coalitionDetail(data)}, nil
}

func (s *site) coalitionDetailFragment(w http.ResponseWriter, r *http.Request) {
	data, found, err := s.options.Coalitions.CoalitionDetail(r.Context(), r.PathValue("slug"))
	if err != nil {
		http.Error(w, "Could not load coalition", http.StatusServiceUnavailable)
		return
	}
	if !found {
		http.NotFound(w, r)
		return
	}
	s.renderFragment(w, r, coalitionDetailContent(data))
}

func coalitionNumber(value int64) string { return fullNumber(float64(value)) }
func coalitionEfficiency(kills, losses int64) string {
	if kills+losses <= 0 {
		return "0%"
	}
	return fmt.Sprintf("%.1f%%", float64(kills)/float64(kills+losses)*100)
}
func coalitionISKEfficiency(destroyed, lost float64) string {
	if destroyed+lost <= 0 {
		return "0%"
	}
	return fmt.Sprintf("%.1f%%", destroyed/(destroyed+lost)*100)
}
func coalitionSafeURL(raw *string) string {
	if raw == nil {
		return ""
	}
	parsed, err := url.Parse(*raw)
	if err != nil || parsed.Host == "" || parsed.User != nil || (parsed.Scheme != "https" && parsed.Scheme != "http") {
		return ""
	}
	return parsed.String()
}
func coalitionEditorName(name *string) string {
	if name == nil || *name == "" {
		return "Initial verified import"
	}
	return "Last edited by " + *name
}
func coalitionEditVerb(action, name string) string {
	switch action {
	case "update":
		return " edited " + name
	case "create":
		return " created " + name
	default:
		return " imported " + name
	}
}
func coalitionUTCDate(raw string) string {
	when, err := time.Parse(time.RFC3339Nano, raw)
	if err != nil {
		return ""
	}
	return when.UTC().Format("2 Jan 2006, 15:04 UTC")
}
func coalitionID(id int64) string { return strconv.FormatInt(id, 10) }
func coalitionSummaryText(coalition api.CoalitionSummary) string {
	if strings.TrimSpace(coalition.Description) == "" {
		return "No description has been added yet."
	}
	return coalition.Description
}

func coalitionSearchText(coalition api.CoalitionSummary) string {
	return strings.ToLower(coalition.Name + " " + coalition.Description)
}

func coalitionAllianceTotal(coalitions []api.CoalitionSummary) int64 {
	var total int64
	for _, coalition := range coalitions {
		total += coalition.AllianceCount
	}
	return total
}
