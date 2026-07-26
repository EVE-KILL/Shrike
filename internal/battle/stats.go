package battle

import (
	"sort"
	"time"
)

// TeamEntry is one corporation's contribution to a side.
type TeamEntry struct {
	CorporationID int32   `json:"corporation_id"`
	AllianceID    int32   `json:"alliance_id"`
	Kills         int64   `json:"kills"`
	Losses        int64   `json:"losses"`
	IskDestroyed  float64 `json:"isk_destroyed"`
	IskLost       float64 `json:"isk_lost"`
}

// Team is one side of a battle.
type Team struct {
	Entries      []TeamEntry `json:"entries"`
	Kills        int64       `json:"total_kills"`
	Losses       int64       `json:"total_losses"`
	IskDestroyed float64     `json:"total_isk_destroyed"`
	IskLost      float64     `json:"total_isk_lost"`
}

// Detected is a battle and its two sides.
type Detected struct {
	SolarSystemID   int32
	RegionID        int32
	Start           time.Time
	End             time.Time
	DurationMinutes int
	KillCount       int
	IskDestroyed    float64
	Teams           [2]Team
	MultiParty      bool
	KillmailIDs     []int64
}

// ComputeTeamStats tallies each side from the killmails and the assignment.
//
// Only corporations the assignment placed are counted, and that exclusion is
// load-bearing rather than tidy. Killwhore-filtered alts, NPC structures
// landing a final blow, and third parties whose victims were filtered out all
// have no side — defaulting them onto team zero silently inflates one side's
// kills with no matching loss on the other, which is what makes a battle's
// numbers stop adding up.
func ComputeTeamStats(killmails []Killmail, attackersByKill map[int64][]Attacker, a TeamAssignment) [2]Team {
	type corpStats struct {
		team         int
		allianceID   int32
		kills        int64
		losses       int64
		iskDestroyed float64
		iskLost      float64
	}
	stats := map[int32]*corpStats{}

	// Returns nil for a corporation with no side — see the note above.
	get := func(corpID int32) *corpStats {
		team, ok := a.CorpTeam[corpID]
		if !ok {
			return nil
		}
		s, exists := stats[corpID]
		if !exists {
			s = &corpStats{team: team, allianceID: a.CorpAlliance[corpID]}
			stats[corpID] = s
		}
		return s
	}

	for _, km := range killmails {
		if s := get(km.VictimCorporationID); s != nil {
			s.losses++
			s.iskLost += km.TotalValue
		}

		// The kill is credited to the final blow, falling back to the top
		// damage dealer when no attacker is flagged — which happens on older
		// mails and on some structure kills.
		atts := attackersByKill[km.KillmailID]
		var killer *Attacker
		for i := range atts {
			if atts[i].FinalBlow {
				killer = &atts[i]
				break
			}
		}
		if killer == nil {
			for i := range atts {
				if killer == nil || atts[i].DamageDone > killer.DamageDone {
					killer = &atts[i]
				}
			}
		}
		if killer == nil {
			continue
		}
		if s := get(killer.CorporationID); s != nil {
			s.kills++
			s.iskDestroyed += km.TotalValue
		}
	}

	var teams [2]Team
	corps := make([]int32, 0, len(stats))
	for id := range stats {
		corps = append(corps, id)
	}
	// Sorted so the output is stable between runs, which matters because these
	// rows are stored and compared.
	sort.Slice(corps, func(i, j int) bool { return corps[i] < corps[j] })

	for _, id := range corps {
		s := stats[id]
		t := &teams[s.team]
		t.Entries = append(t.Entries, TeamEntry{
			CorporationID: id,
			AllianceID:    s.allianceID,
			Kills:         s.kills,
			Losses:        s.losses,
			IskDestroyed:  s.iskDestroyed,
			IskLost:       s.iskLost,
		})
		t.Kills += s.kills
		t.Losses += s.losses
		t.IskDestroyed += s.iskDestroyed
		t.IskLost += s.iskLost
	}

	return teams
}

// Detect runs the whole pipeline over one system's killmails.
//
// Returns nil when the killmails are not a battle — too sparse to have an
// active segment, or with no belligerents left after filtering.
func Detect(killmails []Killmail, attackersByKill map[int64][]Attacker, required *time.Time) *Detected {
	if len(killmails) == 0 {
		return nil
	}

	sort.Slice(killmails, func(i, j int) bool {
		return killmails[i].KillmailTime.Before(killmails[j].KillmailTime)
	})

	window := RefineBoundaries(killmails, MinKillsPerSegment, required)
	if window == nil {
		return nil
	}

	// Only the kills inside the refined window are part of the battle. The
	// input deliberately spans more, so the boundaries can be found.
	var inWindow []Killmail
	var ids []int64
	var isk float64
	for _, km := range killmails {
		if km.KillmailTime.Before(window.Start) || !km.KillmailTime.Before(window.End) {
			continue
		}
		inWindow = append(inWindow, km)
		ids = append(ids, km.KillmailID)
		isk += km.TotalValue
	}
	if len(inWindow) == 0 {
		return nil
	}

	assignment := AssignTeams(inWindow, attackersByKill)
	if len(assignment.CorpTeam) == 0 {
		return nil
	}

	return &Detected{
		SolarSystemID:   inWindow[0].SolarSystemID,
		RegionID:        inWindow[0].RegionID,
		Start:           window.Start,
		End:             window.End,
		DurationMinutes: int(window.End.Sub(window.Start).Minutes()),
		KillCount:       len(inWindow),
		IskDestroyed:    isk,
		Teams:           ComputeTeamStats(inWindow, attackersByKill, assignment),
		MultiParty:      assignment.MultiParty,
		KillmailIDs:     ids,
	}
}

// DetectAll finds every distinct fight in one system window.
//
// Detect returns the first active burst when no required timestamp is supplied.
// A daily re-scan can contain several fights in the same system, so stopping
// after that first result silently drops the rest. Advancing to the end of each
// detected window preserves the same boundary logic while exhausting the
// input.
func DetectAll(killmails []Killmail, attackersByKill map[int64][]Attacker) []*Detected {
	if len(killmails) == 0 {
		return nil
	}

	sorted := append([]Killmail(nil), killmails...)
	sort.Slice(sorted, func(i, j int) bool {
		return sorted[i].KillmailTime.Before(sorted[j].KillmailTime)
	})

	var out []*Detected
	for len(sorted) > 0 {
		detected := Detect(sorted, attackersByKill, nil)
		if detected == nil {
			break
		}
		out = append(out, detected)

		next := 0
		for next < len(sorted) && sorted[next].KillmailTime.Before(detected.End) {
			next++
		}
		if next == 0 {
			// Defensive progress guard; a valid detected window always consumes
			// at least one killmail.
			next = 1
		}
		sorted = sorted[next:]
	}
	return out
}
