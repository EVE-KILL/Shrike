package frontend

import (
	"fmt"
	"net/http"
	"strconv"
	"time"

	"github.com/eve-kill/shrike/internal/api"
)

func (s *site) entityIntelFragment(w http.ResponseWriter, r *http.Request) {
	kind := r.PathValue("kind")
	if kind != "character" && kind != "corporation" && kind != "alliance" {
		http.NotFound(w, r)
		return
	}
	id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil || id < 1 || id > 2_147_483_647 {
		http.NotFound(w, r)
		return
	}
	days, err := fragmentInt(r.URL.Query().Get("days"), 90, 0, 365)
	if err != nil || days != 0 && days != 14 && days != 30 && days != 90 && days != 180 && days != 365 {
		http.Error(w, "Invalid period", http.StatusBadRequest)
		return
	}
	data, err := s.options.Entities.EntityIntel(r.Context(), kind, id, days)
	if err != nil {
		http.Error(w, "Could not load intel", http.StatusServiceUnavailable)
		return
	}
	s.renderFragment(w, r, entityIntel(kind, data))
}

type entityPlaystyleBar struct {
	Label, Class string
	Percent      int64
}

func entityPlaystyleBars(data api.EntityIntelData) []entityPlaystyleBar {
	choices := []entityPlaystyleBar{
		{"Solo", "solo", data.Playstyle.Solo},
		{"Small", "small", data.Playstyle.SmallGang},
		{"Mid", "mid", data.Playstyle.MidGang},
		{"Fleet", "fleet", data.Playstyle.Fleet},
		{"Blob", "blob", data.Playstyle.Blob},
	}
	result := make([]entityPlaystyleBar, 0, len(choices))
	for _, choice := range choices {
		if choice.Percent > 0 {
			result = append(result, choice)
		}
	}
	return result
}

func entityPlaystyleWidth(percent int64) string {
	return fmt.Sprintf("width:%d%%", min(max(percent, 0), 100))
}

func entityIntelPeers(peers []api.EntityIntelPeer, field string) []api.TopEntry {
	result := make([]api.TopEntry, 0, len(peers))
	for _, peer := range peers {
		var count float64
		switch field {
		case "allies":
			count = float64(peer.SharedEnemyKills)
		case "enemies":
			count = float64(peer.Total)
		case "hunting":
			count = float64(peer.ActiveCharacters)
		case "departures":
			value, _ := time.Parse(time.RFC3339, peer.LeftAt)
			count = float64(value.Unix())
		case "joins":
			value, _ := time.Parse(time.RFC3339, peer.JoinedAt)
			count = float64(value.Unix())
		default:
			count = float64(peer.Count)
		}
		result = append(result, api.TopEntry{ID: peer.ID, Name: peer.Name, Count: count})
	}
	return result
}

func entityIntelFleetSize(value float64) string {
	if value <= 0 {
		return "-"
	}
	return fmt.Sprintf("%.1f", value)
}
