package queue

// Priority is a semantic tier, not a number to be invented at each call site.
//
// The four tiers exist so that bulk repair work can never crowd out an explicit
// request or live entity maintenance. A backfill walking a million characters
// and the fetch for a character who just appeared on a killmail are both
// "esi_character" jobs; without tiers the backfill wins by sheer volume and the
// live killboard shows "Unknown" for minutes.
type Priority int

const (
	// Immediate is work someone is waiting on — an explicit request from the
	// site or an operator.
	Immediate Priority = iota + 1

	// Live is ordinary maintenance driven by things happening in game now.
	Live

	// RecentBackfill is repair work over the recent past.
	RecentBackfill

	// DormantBackfill is bulk work over everything else, and must never
	// interfere with any of the above.
	DormantBackfill
)

// The BullMQ spellings of the same four tiers, as they appear in
// backend/src/queue/priorities.ts. Lower is more urgent there too, but the
// numbering is 0/1/5/10 rather than 1/2/3/4, and 0 has the additional meaning
// of BullMQ's un-prioritised FIFO lane.
//
// These constants exist to make the translation checkable rather than implied:
// jobs enqueued by the TypeScript dispatcher and by this one have to land in
// the same relative order during the changeover, when both are running.
const (
	bullImmediate       = 0
	bullLive            = 1
	bullRecentBackfill  = 5
	bullDormantBackfill = 10
)

// FromBullMQ converts a BullMQ priority to a tier.
//
// Values between the defined tiers round towards the more urgent one, matching
// historyPriorityFromParent(): a job asking for 3 is treated as recent
// backfill, not dormant. An absent priority is Live rather than Immediate,
// because BullMQ's implicit 0 was never a considered choice — it is what a
// caller that said nothing got, and treating silence as maximum urgency is how
// the immediate lane fills with routine work.
func FromBullMQ(priority int) Priority {
	switch {
	case priority <= bullImmediate:
		return Immediate
	case priority <= bullLive:
		return Live
	case priority <= bullRecentBackfill:
		return RecentBackfill
	default:
		return DormantBackfill
	}
}

// CascadePriority is the tier for follow-up work discovered while running a
// job, mirroring historyPriorityFromParent().
//
// Follow-up work inherits its parent's urgency rather than resetting to the
// top. A character discovered during a dormant backfill leads to a corporation
// fetch that is also dormant backfill; without this, one job at the bottom tier
// spawns children at the top and the tiers stop meaning anything.
func CascadePriority(parent Priority) Priority {
	if parent < Immediate || parent > DormantBackfill {
		return Live
	}
	return parent
}

// River accepts priorities 1 through 4, which the tiers are numbered to match
// directly. The assertion that they do is in the tests rather than here.
const (
	riverHighest = 1
	riverLowest  = 4
)

// Valid reports whether a tier is one River will accept.
func (p Priority) Valid() bool { return p >= riverHighest && p <= riverLowest }

// String names the tier for display.
func (p Priority) String() string {
	switch p {
	case Immediate:
		return "immediate"
	case Live:
		return "live"
	case RecentBackfill:
		return "recent-backfill"
	case DormantBackfill:
		return "dormant-backfill"
	default:
		return "unknown"
	}
}
