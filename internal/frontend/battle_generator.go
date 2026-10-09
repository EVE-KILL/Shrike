package frontend

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"
)

type generatorSystem struct {
	ID   int64  `json:"id"`
	Name string `json:"name"`
}
type generatorPrefill struct {
	Systems  []generatorSystem `json:"systems"`
	Start    string            `json:"startTime"`
	End      string            `json:"endTime"`
	BattleID int64             `json:"battleId"`
	Auto     bool              `json:"autoAdvance"`
}
type wizardStep struct {
	N           int
	Label, Hint string
}

var generatorSteps = []wizardStep{{1, "Location & time", "Where and when"}, {2, "Assign sides", "Who fought whom"}, {3, "Preview & save", "Check, then publish"}}

func generatorInputTime(raw string) string {
	date, err := time.Parse(time.RFC3339Nano, raw)
	if err != nil {
		return ""
	}
	return date.UTC().Format("2006-01-02T15:04")
}
func generatorQuery(r *http.Request) generatorPrefill {
	q := r.URL.Query()
	prefill := generatorPrefill{Systems: []generatorSystem{}}
	id, _ := strconv.ParseInt(q.Get("system"), 10, 32)
	start, end := generatorInputTime(q.Get("start")), generatorInputTime(q.Get("end"))
	if id > 0 && start != "" && end != "" {
		name := string([]rune(q.Get("systemName"))[:min(60, len([]rune(q.Get("systemName"))))])
		if name == "" {
			name = "System " + strconv.FormatInt(id, 10)
		}
		prefill.Systems = []generatorSystem{{id, name}}
		prefill.Start = start
		prefill.End = end
		prefill.Auto = q.Get("step") == "sides"
	}
	if battle, err := strconv.ParseInt(q.Get("battle"), 10, 32); err == nil && battle > 0 {
		prefill.BattleID = battle
	}
	return prefill
}
func (s *site) battleGeneratorPage(r *http.Request) (Page, error) {
	user, err := s.browserSession(r)
	if err != nil {
		return Page{}, err
	}
	return Page{Metadata: Metadata{Title: "Battle Generator", OGTitle: "Battle Generator — EVE-KILL", Description: "Create custom battle reports by selecting systems, time ranges, and assigning alliances and corporations to sides.", CanonicalPath: "/battlegenerator", Indexable: true, Stylesheets: []string{"utilities.css", "utility-page.css", "battle-detail.css", "battle-generator.css"}, ModuleScripts: []string{"battle-generator.js"}, StructuredData: map[string]any{"@context": "https://schema.org", "@type": "BreadcrumbList", "itemListElement": []map[string]any{{"@type": "ListItem", "position": 1, "name": "Home", "item": canonicalOrigin + "/"}, {"@type": "ListItem", "position": 2, "name": "Battle Generator", "item": canonicalOrigin + "/battlegenerator"}}}}, Content: battleGeneratorContent(generatorQuery(r), user != nil)}, nil
}
func generatorError(w http.ResponseWriter, status int, message string) {
	w.Header().Set("Cache-Control", "private, no-store")
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(map[string]string{"message": message})
}
func (s *site) battleGeneratorPreviewFragment(w http.ResponseWriter, r *http.Request) {
	user, err := s.browserSession(r)
	if err != nil {
		generatorError(w, 503, "Could not verify your session")
		return
	}
	if user == nil {
		generatorError(w, 401, "Your session expired. Log in again to continue.")
		return
	}
	body, err := io.ReadAll(http.MaxBytesReader(w, r.Body, 4<<20))
	if err != nil {
		generatorError(w, 413, "Preview request is too large")
		return
	}
	data, err := s.options.BattleGenerator.BattleGeneratorPreview(r.Context(), body)
	if err != nil {
		var public interface{ GetStatus() int }
		if errors.As(err, &public) && public.GetStatus() >= 400 && public.GetStatus() < 500 {
			generatorError(w, public.GetStatus(), err.Error())
		} else {
			generatorError(w, 503, "Could not generate preview. Try again.")
		}
		return
	}
	var view battleDetailData
	if err = json.Unmarshal(data, &view); err != nil || len(view.Teams) != 2 {
		generatorError(w, 503, "Could not render battle preview")
		return
	}
	w.Header().Set("Cache-Control", "private, no-store")
	s.renderFragment(w, r, battleGeneratorPreview(view, data))
}
func generatorTeamName(team battleTeam, index int) string {
	if strings.TrimSpace(team.Name) != "" {
		return team.Name
	}
	return "Team " + strconv.Itoa(index+1)
}
