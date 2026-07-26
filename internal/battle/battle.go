// Package battle detects fights from clusters of killmails.
//
// A "battle" is not a thing EVE records — it is inferred from the shape of the
// killmails: a burst of kills in one system, and two sides that can be told
// apart from who shot whom. Both halves are heuristics, and the interesting
// parts of this package are the filters that stop the heuristics being fooled.
package battle

import (
	"math"
	"sort"
	"time"
)

// Detection parameters.
const (
	// SegmentDuration is the granularity boundaries are refined at. Five
	// minutes is short enough to find the real edges of a fight and long
	// enough that a lull between waves does not split one battle into three.
	SegmentDuration = 5 * time.Minute

	// MinKillsPerSegment is what makes a segment "active". Below it, the
	// killmails are ambient — gate camps, ratting, unrelated skirmishes.
	MinKillsPerSegment = 5

	// MaxInactiveSegments is how much quiet ends a battle. Thirty minutes:
	// long enough to survive a reship, short enough that the next fight in the
	// same system is a separate battle.
	MaxInactiveSegments = 6

	// KillwhoreThreshold drops attackers who contributed almost nothing.
	// Someone who applied 4% of the damage was passing through, and counting
	// them as a belligerent puts unrelated corporations on a side.
	KillwhoreThreshold = 0.05

	// MultiPartyConflictRatio is when a two-sided model stops describing the
	// fight. Above it, too many pairs that shot each other ended up on the same
	// side for the split to mean anything.
	MultiPartyConflictRatio = 0.20
)

// Killmail is one kill in a candidate battle.
type Killmail struct {
	KillmailID          int64
	KillmailTime        time.Time
	SolarSystemID       int32
	RegionID            int32
	TotalValue          float64
	VictimCorporationID int32
	VictimAllianceID    int32
	VictimFactionID     int32
	VictimShipTypeID    int32
}

// Attacker is one participant on a kill.
type Attacker struct {
	KillmailID    int64
	CharacterID   int32
	CorporationID int32
	AllianceID    int32
	FactionID     int32
	DamageDone    int64
	FinalBlow     bool
}

// Window is a refined battle start and end.
type Window struct {
	Start time.Time
	End   time.Time
}

// RefineBoundaries finds the real extent of a fight within a set of killmails.
//
// The killmails must be sorted by time. Segments with enough kills are active;
// a run of active segments is a candidate battle, ended by enough consecutive
// quiet ones. A required time selects the candidate containing it, which is how
// a specific kill is resolved to the battle it belonged to rather than to the
// first burst in the window.
//
// Returns nil when nothing in the range is busy enough to be a battle.
func RefineBoundaries(killmails []Killmail, minKills int, required *time.Time) *Window {
	if len(killmails) == 0 {
		return nil
	}
	if minKills <= 0 {
		minKills = MinKillsPerSegment
	}

	seg := int64(SegmentDuration / time.Millisecond)
	earliest := killmails[0].KillmailTime.UnixMilli()
	latest := killmails[len(killmails)-1].KillmailTime.UnixMilli()

	segStart := (earliest / seg) * seg
	// Ceil, so the final partial segment is walked.
	segEnd := ((latest+1)/seg + 1) * seg

	counts := map[int64]int{}
	for _, km := range killmails {
		counts[(km.KillmailTime.UnixMilli()/seg)*seg]++
	}

	type candidate struct{ start, end int64 }
	var candidates []candidate

	var battleStart, battleEnd int64
	open := false
	inactive := 0

	for t := segStart; t < segEnd; t += seg {
		if counts[t] >= minKills {
			if !open {
				battleStart, open = t, true
			}
			battleEnd = t + seg
			inactive = 0
			continue
		}
		if !open {
			continue
		}
		inactive++
		if inactive >= MaxInactiveSegments {
			candidates = append(candidates, candidate{battleStart, battleEnd})
			open, inactive = false, 0
		}
	}
	if open {
		candidates = append(candidates, candidate{battleStart, battleEnd})
	}
	if len(candidates) == 0 {
		return nil
	}

	selected := candidates[0]
	if required != nil {
		found := false
		at := required.UnixMilli()
		for _, c := range candidates {
			if at >= c.start && at < c.end {
				selected, found = c, true
				break
			}
		}
		if !found {
			return nil
		}
	}

	return &Window{
		Start: time.UnixMilli(selected.start).UTC(),
		End:   time.UnixMilli(selected.end).UTC(),
	}
}

// TeamAssignment is the result of splitting belligerents into two sides.
type TeamAssignment struct {
	// CorpTeam maps a corporation to side 0 or 1. A corporation absent from
	// this map was filtered out and must not be counted anywhere.
	CorpTeam map[int32]int

	// CorpAlliance is each corporation's alliance at the time of the fight,
	// taken from the killmails rather than from current membership.
	CorpAlliance map[int32]int32

	// MultiParty marks a fight that two sides do not describe — a
	// free-for-all, or a third party that engaged both.
	MultiParty bool
}

// AssignTeams splits the belligerents into two opposing sides.
//
// The signal is who shot whom: every attacker-victim pair is an edge, weighted
// by the value of the kill on a log scale so a titan counts for more than a
// shuttle without swamping everything. Sides are then assigned greedily from
// the strongest edges outward.
//
// Two filters matter more than the algorithm:
//
//   - Killwhores. Someone who did 4% of the damage was passing through. Without
//     this filter, a single opportunistic attacker drags their whole
//     corporation onto a side of a fight they had nothing to do with.
//   - NPCs. An attacker with a faction and no character is an NPC, and NPCs
//     shoot everyone. Counting them creates edges between corporations that
//     never fought each other.
func AssignTeams(killmails []Killmail, attackersByKill map[int64][]Attacker) TeamAssignment {
	out := TeamAssignment{
		CorpTeam:     map[int32]int{},
		CorpAlliance: map[int32]int32{},
	}

	// Affiliation as recorded on the killmails — a corporation that changed
	// alliance since the fight belongs to the one it was in at the time.
	for _, km := range killmails {
		if km.VictimCorporationID != 0 && km.VictimAllianceID != 0 {
			out.CorpAlliance[km.VictimCorporationID] = km.VictimAllianceID
		}
	}
	for _, atts := range attackersByKill {
		for _, a := range atts {
			if a.CorporationID != 0 && a.AllianceID != 0 {
				out.CorpAlliance[a.CorporationID] = a.AllianceID
			}
		}
	}

	type edge struct {
		a, b  int32
		score float64
	}
	scores := map[[2]int32]float64{}

	for _, km := range killmails {
		victim := km.VictimCorporationID
		if victim == 0 {
			continue
		}
		atts := attackersByKill[km.KillmailID]

		var totalDamage int64
		for _, a := range atts {
			totalDamage += a.DamageDone
		}
		// Floored so a zero-value kill still contributes an edge rather than a
		// negative or infinite weight.
		weight := math.Log10(math.Max(km.TotalValue, 10_000))

		for _, a := range atts {
			if a.CorporationID == 0 || a.CorporationID == victim {
				continue
			}
			if totalDamage > 0 && float64(a.DamageDone)/float64(totalDamage) < KillwhoreThreshold {
				continue
			}
			if a.FactionID != 0 && a.CharacterID == 0 {
				continue
			}
			scores[canonical(a.CorporationID, victim)] += weight
		}
	}

	edges := make([]edge, 0, len(scores))
	for k, s := range scores {
		edges = append(edges, edge{k[0], k[1], s})
	}
	// Strongest first, with the pair as a tiebreak so the assignment is
	// deterministic — otherwise two runs over the same fight can produce
	// mirrored teams.
	sort.Slice(edges, func(i, j int) bool {
		if edges[i].score != edges[j].score {
			return edges[i].score > edges[j].score
		}
		if edges[i].a != edges[j].a {
			return edges[i].a < edges[j].a
		}
		return edges[i].b < edges[j].b
	})

	for _, e := range edges {
		ta, okA := out.CorpTeam[e.a]
		tb, okB := out.CorpTeam[e.b]
		switch {
		case !okA && !okB:
			out.CorpTeam[e.a], out.CorpTeam[e.b] = 0, 1
		case okA && !okB:
			out.CorpTeam[e.b] = 1 - ta
		case !okA && okB:
			out.CorpTeam[e.a] = 1 - tb
		}
	}

	// Alliance cohesion. Corporations in one alliance fought together, so a
	// greedy pass that split them got it wrong — a majority vote pulls them
	// back onto one side.
	votes := map[int32][2]int{}
	for corp, team := range out.CorpTeam {
		alliance := out.CorpAlliance[corp]
		if alliance == 0 {
			continue
		}
		v := votes[alliance]
		v[team]++
		votes[alliance] = v
	}
	allianceTeam := map[int32]int{}
	for alliance, v := range votes {
		if v[0] >= v[1] {
			allianceTeam[alliance] = 0
		} else {
			allianceTeam[alliance] = 1
		}
	}
	for corp := range out.CorpTeam {
		if alliance := out.CorpAlliance[corp]; alliance != 0 {
			if team, ok := allianceTeam[alliance]; ok {
				out.CorpTeam[corp] = team
			}
		}
	}

	// If enough pairs that shot each other ended up on the same side, two teams
	// is the wrong model for this fight.
	conflicts := 0
	for _, e := range edges {
		ta, okA := out.CorpTeam[e.a]
		tb, okB := out.CorpTeam[e.b]
		if okA && okB && ta == tb {
			conflicts++
		}
	}
	out.MultiParty = len(edges) > 0 &&
		float64(conflicts)/float64(len(edges)) > MultiPartyConflictRatio

	return out
}

func canonical(a, b int32) [2]int32 {
	if a < b {
		return [2]int32{a, b}
	}
	return [2]int32{b, a}
}
