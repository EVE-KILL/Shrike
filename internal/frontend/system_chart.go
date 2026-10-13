package frontend

import (
	"fmt"
	"time"

	"github.com/eve-kill/shrike/internal/api"
)

type systemChartHour struct {
	Time     time.Time
	Activity api.UniverseSystemActivity
	Recorded bool
}

type systemChartRow struct {
	Key    string
	Label  string
	Values []*int64
	Total  int64
	Peak   int64
}

type systemChartView struct {
	Hours   []systemChartHour
	Rows    []systemChartRow
	Missing int
}

func systemActivityChart(history []api.UniverseSystemActivity) systemChartView {
	if len(history) == 0 {
		return systemChartView{}
	}
	byHour := make(map[time.Time]api.UniverseSystemActivity, len(history))
	var latest time.Time
	for _, row := range history {
		if row.Timestamp.IsZero() {
			continue
		}
		hour := row.Timestamp.UTC().Truncate(time.Hour)
		byHour[hour] = row
		if hour.After(latest) {
			latest = hour
		}
	}
	if latest.IsZero() {
		return systemChartView{}
	}
	view := systemChartView{Hours: make([]systemChartHour, 24)}
	for index := range view.Hours {
		hour := latest.Add(time.Duration(index-23) * time.Hour)
		activity, recorded := byHour[hour]
		view.Hours[index] = systemChartHour{Time: hour, Activity: activity, Recorded: recorded}
		if !recorded {
			view.Missing++
		}
	}
	for _, series := range []struct {
		Key, Label string
		Value      func(api.UniverseSystemActivity) int64
	}{
		{"jumps", "Jumps", func(a api.UniverseSystemActivity) int64 { return a.ShipJumps }},
		{"ship-kills", "Ship kills", func(a api.UniverseSystemActivity) int64 { return a.ShipKills }},
		{"npc-kills", "NPC kills", func(a api.UniverseSystemActivity) int64 { return a.NPCKills }},
		{"pod-kills", "Pod kills", func(a api.UniverseSystemActivity) int64 { return a.PodKills }},
	} {
		row := systemChartRow{Key: series.Key, Label: series.Label, Values: make([]*int64, 24)}
		for index, hour := range view.Hours {
			if !hour.Recorded {
				continue
			}
			value := series.Value(hour.Activity)
			row.Values[index] = &value
			row.Total += value
			row.Peak = max(row.Peak, value)
		}
		if row.Total > 0 {
			view.Rows = append(view.Rows, row)
		}
	}
	return view
}

func systemChartHeight(value *int64, peak int64) string {
	if value == nil {
		return ""
	}
	if peak <= 0 {
		return "2px"
	}
	return fmt.Sprintf("%.2f%%", max(2, float64(*value)/float64(peak)*100))
}
