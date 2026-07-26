// Package stats maintains the killboard's aggregate counters.
//
// Three tables, all keyed the same way — (entity_type, entity_id, period_type,
// period_start):
//
//   - stats               one row per entity per period, the headline counters
//   - stats_breakdowns    the same, split by a dimension (ship flown, region,
//     who they died to) — the large one, hundreds of
//     millions of rows
//   - stats_leaderboards  the top N entities per metric per period
//
// Every killmail touches a lot of this. A single kill with fifty attackers
// updates the victim and fifty attackers, each across three period types, each
// with several breakdown dimensions — which is why the writer accumulates in
// memory and merges once rather than issuing an upsert per counter.
//
// The enum values are stored as smallints and are a wire contract with the
// frontend and the API. They are never renumbered; the reserved gaps below are
// gaps on purpose.
package stats

// EntityType is what a stats row is about.
type EntityType int16

const (
	EntityCharacter     EntityType = 0
	EntityCorporation   EntityType = 1
	EntityAlliance      EntityType = 2
	EntityShip          EntityType = 3
	EntitySystem        EntityType = 4
	EntityConstellation EntityType = 5
	EntityRegion        EntityType = 6
)

// PeriodType is the granularity of a row.
//
// Daily rows are the source of truth and everything else is rebuilt from them:
// monthly from daily, yearly from monthly. That is why a purge of old daily
// rows has to run after the rebuild rather than before it.
type PeriodType int16

const (
	PeriodDaily   PeriodType = 0
	PeriodMonthly PeriodType = 1
	PeriodYearly  PeriodType = 2
)

// DimCategory is what a breakdown row splits by.
//
// The gaps are reserved rather than free. 20, 30 and 40–42 were character-level
// relationship dimensions that moved to Memgraph, where the graph shape fits
// better; rows may still exist for them, so the ids must not be reused for
// something else or old rows would be read as the new meaning.
type DimCategory int16

const (
	DimShipFlown DimCategory = 0
	DimShipLost  DimCategory = 1

	DimSystem        DimCategory = 10
	DimConstellation DimCategory = 11
	DimRegion        DimCategory = 12

	DimDiesToCorporation DimCategory = 21
	DimDiesToAlliance    DimCategory = 22

	DimKilledCorporation DimCategory = 31
	DimKilledAlliance    DimCategory = 32
)

// LeaderboardMetric is what a leaderboard ranks by.
type LeaderboardMetric int16

const (
	MetricKills        LeaderboardMetric = 0
	MetricLosses       LeaderboardMetric = 1
	MetricIskDestroyed LeaderboardMetric = 2
	MetricIskLost      LeaderboardMetric = 3
	MetricSoloKills    LeaderboardMetric = 4
	MetricPoints       LeaderboardMetric = 5
	MetricFinalBlows   LeaderboardMetric = 6
)

// Row is the headline counters for one entity in one period.
type Row struct {
	Kills            int64   `json:"kills"`
	Losses           int64   `json:"losses"`
	SoloKills        int64   `json:"solo_kills"`
	SoloLosses       int64   `json:"solo_losses"`
	NPCLosses        int64   `json:"npc_losses"`
	FinalBlows       int64   `json:"final_blows"`
	Points           int64   `json:"points"`
	IskDestroyed     float64 `json:"isk_destroyed"`
	IskLost          float64 `json:"isk_lost"`
	DamageDealt      int64   `json:"damage_dealt"`
	DamageTaken      int64   `json:"damage_taken"`
	SumAttackerCount int64   `json:"sum_attacker_count"`
}

// Add accumulates another row into this one.
func (r *Row) Add(o Row) {
	r.Kills += o.Kills
	r.Losses += o.Losses
	r.SoloKills += o.SoloKills
	r.SoloLosses += o.SoloLosses
	r.NPCLosses += o.NPCLosses
	r.FinalBlows += o.FinalBlows
	r.Points += o.Points
	r.IskDestroyed += o.IskDestroyed
	r.IskLost += o.IskLost
	r.DamageDealt += o.DamageDealt
	r.DamageTaken += o.DamageTaken
	r.SumAttackerCount += o.SumAttackerCount
}

// Breakdown is one dimension's counters.
type Breakdown struct {
	Kills        int64   `json:"kills"`
	Losses       int64   `json:"losses"`
	IskDestroyed float64 `json:"isk_destroyed"`
	IskLost      float64 `json:"isk_lost"`

	// LastKillmail records the most recent kill contributing to this
	// breakdown, which is what "last seen flying" is read from.
	LastKillmailID   int64 `json:"last_killmail_id"`
	LastKillmailTime int64 `json:"last_killmail_time"`
}

// Add accumulates another breakdown, keeping the newer last-killmail marker.
func (b *Breakdown) Add(o Breakdown) {
	b.Kills += o.Kills
	b.Losses += o.Losses
	b.IskDestroyed += o.IskDestroyed
	b.IskLost += o.IskLost
	if markerAfter(o.LastKillmailTime, o.LastKillmailID, b.LastKillmailTime, b.LastKillmailID) {
		b.LastKillmailTime = o.LastKillmailTime
		b.LastKillmailID = o.LastKillmailID
	}
}

// markerAfter compares the stored "last seen" marker as one value.
//
// ESI timestamps have one-second precision, so equal timestamps are routine.
// The killmail id is the deterministic tie-breaker used by the SQL rollups and
// war interactions; applying the same ordering here keeps live ingest,
// catchups, and rollups convergent regardless of processing order.
func markerAfter(at, id, currentAt, currentID int64) bool {
	return at > currentAt || (at == currentAt && id > currentID)
}

// Derived are the metrics computed from a Row rather than stored.
//
// Kept here so every consumer renders the same efficiency and the same blob
// factor; two endpoints computing "efficiency" slightly differently is exactly
// the sort of thing nobody notices until the numbers are compared.
type Derived struct {
	Efficiency       int     `json:"efficiency"`
	IskEfficiency    int     `json:"isk_efficiency"`
	SoloRatio        int     `json:"solo_ratio"`
	SoloLossRatio    int     `json:"solo_loss_ratio"`
	NPCLossRatio     int     `json:"npc_loss_ratio"`
	BlobFactor       float64 `json:"blob_factor"`
	AvgDamagePerKill int64   `json:"avg_damage_per_kill"`

	// DangerRatio is kills per loss. Zero losses with kills is reported as -1
	// rather than an infinity, because JSON has no infinity and encoding one
	// produces a document the frontend cannot parse.
	DangerRatio float64 `json:"danger_ratio"`
}

// Derive computes the metrics that are not stored.
func Derive(r Row) Derived {
	out := Derived{}

	if total := r.Kills + r.Losses; total > 0 {
		out.Efficiency = pct(float64(r.Kills) / float64(total))
	}
	if total := r.IskDestroyed + r.IskLost; total > 0 {
		out.IskEfficiency = pct(r.IskDestroyed / total)
	}
	if r.Kills > 0 {
		out.SoloRatio = pct(float64(r.SoloKills) / float64(r.Kills))
		out.BlobFactor = round2(float64(r.SumAttackerCount) / float64(r.Kills))
		out.AvgDamagePerKill = int64(float64(r.DamageDealt)/float64(r.Kills) + 0.5)
	}
	if r.Losses > 0 {
		out.SoloLossRatio = pct(float64(r.SoloLosses) / float64(r.Losses))
		out.NPCLossRatio = pct(float64(r.NPCLosses) / float64(r.Losses))
		out.DangerRatio = round2(float64(r.Kills) / float64(r.Losses))
	} else if r.Kills > 0 {
		out.DangerRatio = -1
	}

	return out
}

func pct(v float64) int { return int(v*100 + 0.5) }

func round2(v float64) float64 { return float64(int64(v*100+0.5)) / 100 }
