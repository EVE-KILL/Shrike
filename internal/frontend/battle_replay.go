package frontend

import (
	"strconv"
	"time"
)

type battleReplayState struct {
	Start, End, Time int64
	SystemID         int64
	SystemName       string
	Selected         int64
	Systems          []battleShipGroup
	Totals           []battleLossTotal
	Unsided          battleLossTotal
	Occurred         int
	ISK              float64
	Positions        int
	SystemKills      int
	Celestials       int
}

func battleInitialReplay(v battleDetailView) battleReplayState {
	start, end := battleTimeMS(v.Data.Start), battleTimeMS(v.Data.End)
	start = max(start, min(battleFilterTime(v.Filter.From, start), end))
	end = max(start+1, min(end, battleFilterTime(v.Filter.To, end)))
	s := battleReplayState{Start: start, End: end, Time: start, Totals: make([]battleLossTotal, len(v.Data.Teams))}
	if at, e := strconv.ParseInt(v.Query.Get("at"), 10, 64); e == nil && at > 0 {
		s.Time = max(start, min(end, at))
	}
	s.Selected, _ = strconv.ParseInt(v.Query.Get("kill"), 10, 64)
	seen := map[int64]bool{}
	for _, k := range v.Filtered {
		if !seen[k.SystemID] {
			seen[k.SystemID] = true
			s.Systems = append(s.Systems, battleShipGroup{k.SystemID, k.SystemName})
		}
	}
	if len(s.Systems) > 0 {
		s.SystemID = s.Systems[0].ID
		s.SystemName = s.Systems[0].Name
	}
	for _, k := range v.Filtered {
		if k.SystemID != s.SystemID {
			continue
		}
		s.SystemKills++
		if k.X != nil && k.Y != nil && k.Z != nil {
			s.Positions++
		}
		if battleTimeMS(k.Time) > s.Time {
			continue
		}
		s.Occurred++
		s.ISK += k.Value
		side := battleVictimSide(k, v.Data.Sides)
		if side < 0 {
			s.Unsided.Count++
			s.Unsided.ISK += k.Value
		} else if side < len(s.Totals) {
			s.Totals[side].Count++
			s.Totals[side].ISK += k.Value
		}
	}
	for _, l := range v.Replay.Landmarks {
		if l.SystemID == s.SystemID {
			s.Celestials++
		}
	}
	return s
}
func battleReplayPayload(v battleDetailView) map[string]any {
	s := battleInitialReplay(v)
	return map[string]any{"kills": v.Filtered, "landmarks": v.Replay.Landmarks, "teams": v.Data.Teams, "team_entities": v.Data.Sides, "start": s.Start, "end": s.End, "time": s.Time, "selected": s.Selected}
}
func battleReplayClock(ms int64) string { return time.UnixMilli(ms).UTC().Format("15:04:05") }
