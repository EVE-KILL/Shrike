// Package ticker emits the ephemeral announcements that scroll across the site.
//
// These are not the editorial announcements stored in the announcements table.
// They are transient: a titan died, a battle is happening, a war was declared.
// Nothing keeps them beyond their expiry, and nothing is lost if one is missed
// — the killmail is still on the site either way.
//
// Each is published twice for that reason. The relay delivers it to whoever is
// already connected, and a short-lived Redis key holds it so a page loaded a
// minute later still sees it. Neither is authoritative and both are allowed to
// fail.
package ticker

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/eve-kill/shrike/internal/eve"
	"github.com/eve-kill/shrike/internal/killmail"
	"github.com/eve-kill/shrike/internal/relay"
	"github.com/redis/go-redis/v9"
)

// Hull groups worth announcing on their own.
const (
	GroupTitan        int32 = 30
	GroupSupercarrier int32 = 659
	GroupCapsule      int32 = 29
)

// MetaOfficer marks the rarest module tier. See inv_meta_groups.
const MetaOfficer int32 = 5

// Value thresholds, in ISK.
//
// Set where they are because the ticker is a scarcity signal: lower them and it
// becomes a second killfeed that nobody reads.
const (
	ThresholdHighValue    = 25_000_000_000
	ThresholdOfficerFit   = 3_000_000_000
	ThresholdExpensivePod = 10_000_000_000
)

// Ephemeral ticker ids are negative so they can never collide with a real
// announcement row, and each category has its own band so the client can cap
// them independently:
//
//	kills    -1             .. -1_999_999_999   (-killmail_id)
//	battles  -2_000_000_000 .. -2_999_999_999   (BattleIDBase - battle_id)
//	wars     -3_000_000_000 .. -3_999_999_999   (WarIDBase - war_id)
//
// TQ status sits at -999_999_00X: inside the kill band, but far above any
// killmail id that will be issued in the game's lifetime.
const (
	BattleIDBase int64 = -2_000_000_000
	WarIDBase    int64 = -3_000_000_000

	tqOfflineID int64 = -999_999_001
	tqOnlineID  int64 = -999_999_002

	// WarEndedOffset separates a war's end announcement from its start, which
	// would otherwise share an id and have the second overwrite the first.
	WarEndedOffset int64 = -500_000_000
)

const (
	redisKey = "announcements:ephemeral"
	redisTTL = 10 * time.Minute
)

// Colors and icons, matching what the frontend knows how to render.
const (
	ColorInfo    = "info"
	ColorWarning = "warning"
	ColorDanger  = "danger"
	ColorSuccess = "success"
)

// Emitter publishes ticker announcements.
//
// Both dependencies are optional. A CLI path that detects a battle without a
// relay should still detect the battle.
type Emitter struct {
	Relay *relay.Publisher
	Redis *redis.Client
	Cache *eve.Cache
}

// Announcement is the payload the frontend consumes.
//
// The field names are a contract with the client, and BodyHTML duplicating
// BodyMD is deliberate: ticker text is plain, and the client renders whichever
// field its announcement component expects.
type Announcement struct {
	ID        int64   `json:"id"`
	Tier      int     `json:"tier"`
	Title     string  `json:"title"`
	BodyMD    string  `json:"body_md"`
	BodyHTML  string  `json:"body_html"`
	Color     string  `json:"color"`
	Icon      string  `json:"icon"`
	LinkURL   *string `json:"link_url"`
	LinkLabel *string `json:"link_label"`
	StartsAt  string  `json:"starts_at"`
	ExpiresAt string  `json:"expires_at"`
	CreatedAt string  `json:"created_at"`
}

// Spec describes one announcement to emit.
type Spec struct {
	ID      int64
	Title   string
	Body    string
	Color   string
	Icon    string
	LinkURL string
	Expires time.Duration
	Now     time.Time // zero means time.Now
}

// Emit publishes one announcement to the relay and to Redis.
//
// Errors are swallowed on purpose. The caller has already stored a killmail or
// saved a battle; failing that work because a decoration did not go out would
// be the wrong trade.
func (e *Emitter) Emit(ctx context.Context, s Spec) {
	if e == nil {
		return
	}

	now := s.Now
	if now.IsZero() {
		now = time.Now()
	}
	now = now.UTC()

	a := Announcement{
		ID:        s.ID,
		Tier:      1,
		Title:     s.Title,
		BodyMD:    s.Body,
		BodyHTML:  s.Body,
		Color:     s.Color,
		Icon:      s.Icon,
		StartsAt:  now.Format(time.RFC3339),
		ExpiresAt: now.Add(s.Expires).Format(time.RFC3339),
		CreatedAt: now.Format(time.RFC3339),
	}
	if s.LinkURL != "" {
		link, label := s.LinkURL, "View"
		a.LinkURL, a.LinkLabel = &link, &label
	}

	e.publish(ctx, "new", a)

	if e.Redis != nil {
		if body, err := json.Marshal(a); err == nil {
			// Best-effort: a page load missing one ticker entry is not worth
			// surfacing, let alone retrying.
			_ = e.Redis.Set(ctx, fmt.Sprintf("%s:%d", redisKey, s.ID), body, redisTTL).Err()
		}
	}
}

// Expire retracts an announcement before its natural expiry.
func (e *Emitter) Expire(ctx context.Context, id int64) {
	if e == nil {
		return
	}
	e.publish(ctx, "expired", Announcement{ID: id})
	if e.Redis != nil {
		_ = e.Redis.Del(ctx, fmt.Sprintf("%s:%d", redisKey, id)).Err()
	}
}

func (e *Emitter) publish(ctx context.Context, eventType string, a Announcement) {
	if e.Relay == nil {
		return
	}
	e.Relay.Publish(ctx, relay.ChannelAnnouncements, []string{"all"}, map[string]any{
		"event_type":   eventType,
		"announcement": a,
	})
}

// EvaluateKillmail emits at most one announcement for a killmail.
func (e *Emitter) EvaluateKillmail(ctx context.Context, p killmail.Parsed) {
	if e == nil {
		return
	}
	if spec, ok := e.killmailSpec(p); ok {
		e.Emit(ctx, spec)
	}
}

// killmailSpec decides what, if anything, to announce for a killmail.
//
// Separate from the emitting so the decision can be tested as a decision. It
// returns at most one announcement, chosen by descending significance: a titan
// loss is a titan loss whatever else is true of it, and announcing the same
// kill three times for three reasons would read as three kills.
func (e *Emitter) killmailSpec(p killmail.Parsed) (Spec, bool) {
	km := p.Killmail
	shipName := "Unknown Ship"
	if t, ok := e.typeOf(km.VictimShipTypeID); ok && t.Name != "" {
		shipName = t.Name
	}

	location := e.locationText(km.SolarSystemID)
	value := FormatISK(km.TotalValue)
	link := fmt.Sprintf("/kill/%d", km.KillmailID)
	id := -km.KillmailID

	switch {
	case km.VictimShipGroupID == GroupTitan:
		return Spec{
			ID: id, Title: "Titan down: " + shipName,
			Body:  fmt.Sprintf("%s ISK · %s", value, location),
			Color: ColorDanger, Icon: "lucide:skull", LinkURL: link,
			Expires: 90 * time.Minute,
		}, true

	case km.VictimShipGroupID == GroupSupercarrier:
		return Spec{
			ID: id, Title: "Supercarrier down: " + shipName,
			Body:  fmt.Sprintf("%s ISK · %s", value, location),
			Color: ColorDanger, Icon: "lucide:flame", LinkURL: link,
			Expires: 90 * time.Minute,
		}, true

	case km.TotalValue >= ThresholdHighValue:
		return Spec{
			ID: id, Title: fmt.Sprintf("%s destroyed — %s ISK", shipName, value),
			Body:  location,
			Color: ColorWarning, Icon: "lucide:trending-up", LinkURL: link,
			Expires: 45 * time.Minute,
		}, true

	case km.TotalValue >= ThresholdOfficerFit && e.hasOfficerModules(p):
		return Spec{
			ID: id, Title: "Officer-fit " + shipName + " destroyed",
			Body:  fmt.Sprintf("%s ISK · %s", value, location),
			Color: ColorWarning, Icon: "lucide:gem", LinkURL: link,
			Expires: 45 * time.Minute,
		}, true

	case km.VictimShipGroupID == GroupCapsule && km.TotalValue >= ThresholdExpensivePod:
		return Spec{
			ID: id, Title: fmt.Sprintf("Pod worth %s ISK popped", value),
			Body:  "Implants lost in " + location,
			Color: ColorWarning, Icon: "lucide:brain", LinkURL: link,
			Expires: 45 * time.Minute,
		}, true
	}

	return Spec{}, false
}

// hasOfficerModules reports an officer module in a fitted slot.
//
// Fitted slots only. Officer modules in the cargo hold are freight, not a fit,
// and a hauler carrying one is not the story the ticker is trying to tell.
func (e *Emitter) hasOfficerModules(p killmail.Parsed) bool {
	for _, item := range p.Items {
		if item.ParentIndex != killmail.NoParent {
			continue
		}
		if slot := SlotGroup(item.FlagID); slot == 0 || slot > 3 {
			continue
		}
		if t, ok := e.typeOf(item.TypeID); ok && t.MetaGroupID == MetaOfficer {
			return true
		}
	}
	return false
}

// SlotGroup maps an inventory flag to its slot group: 1 high, 2 mid, 3 low,
// 4 rig, 5 subsystem, 6 drone bay. Zero for anything else.
//
// Shared with the fittings extractor, which uses the same numbering.
func SlotGroup(flag int32) int32 {
	switch {
	case flag >= 27 && flag <= 34:
		return 1
	case flag >= 19 && flag <= 26:
		return 2
	case flag >= 11 && flag <= 18:
		return 3
	case flag >= 92 && flag <= 99:
		return 4
	case flag >= 125 && flag <= 132:
		return 5
	case flag == 87:
		return 6
	}
	return 0
}

// BattleStarted announces a detected battle.
func (e *Emitter) BattleStarted(ctx context.Context, battleID int64, systemName, regionName string, kills int, isk float64) {
	// The title already names the system, so the body adds the region rather
	// than repeating it.
	parts := []string{
		fmt.Sprintf("%s ships destroyed", formatCount(kills)),
		FormatISK(isk) + " ISK",
	}
	if regionName != "" {
		parts = append(parts, regionName)
	}

	e.Emit(ctx, Spec{
		ID: BattleIDBase - battleID, Title: "Battle in " + systemName,
		Body:  strings.Join(parts, " · "),
		Color: ColorDanger, Icon: "lucide:swords",
		LinkURL: fmt.Sprintf("/battle/%d", battleID),
		Expires: 90 * time.Minute,
	})
}

// BattleExpired retracts a battle announcement.
//
// The hourly re-detection deletes and re-inserts auto-detected battles, giving
// the same real fight a new id each time. Without this the ticker accumulates
// one entry per re-detection of a battle that has not changed.
func (e *Emitter) BattleExpired(ctx context.Context, battleID int64) {
	e.Expire(ctx, BattleIDBase-battleID)
}

// WarStarted announces a war going live.
//
// Fires on the transition to live rather than on declaration: EVE wars have a
// 24-hour cooldown before anyone may shoot, and that moment is the one that
// matters to the people in it.
func (e *Emitter) WarStarted(ctx context.Context, warID int32, aggressor, defender string, mutual, openForAllies bool) {
	var notes []string
	if mutual {
		notes = append(notes, "Mutual — both sides may engage")
	} else {
		notes = append(notes, fmt.Sprintf("%s may now engage %s", aggressor, defender))
	}
	if openForAllies {
		notes = append(notes, "open for allies")
	}

	e.Emit(ctx, Spec{
		ID:    WarIDBase - int64(warID),
		Title: fmt.Sprintf("War declared: %s vs %s", aggressor, defender),
		Body:  strings.Join(notes, " · "),
		Color: ColorWarning, Icon: "lucide:swords",
		LinkURL: fmt.Sprintf("/war/%d", warID),
		Expires: 180 * time.Minute,
	})
}

// WarEndedBody summarises how a war went.
//
// EVE declares no winner, so neither does this. The honest summary is who
// destroyed more ISK, and the two degenerate cases — a war with no fighting,
// and one too close to call — get their own wording rather than being forced
// into a victory narrative.
func WarEndedBody(aggressor, defender string, aggressorISK, defenderISK float64, retracted bool) string {
	total := aggressorISK + defenderISK
	if total <= 0 {
		if retracted {
			return "Retracted without a shot fired"
		}
		return "Ended without a shot fired"
	}

	aggAhead := aggressorISK >= defenderISK
	leader, lead, trail := defender, defenderISK, aggressorISK
	if aggAhead {
		leader, lead, trail = aggressor, aggressorISK, defenderISK
	}

	// Within ten percent is a draw in everything but name; calling it a win
	// would be editorialising over noise.
	var body string
	if (lead-trail)/total > 0.1 {
		body = fmt.Sprintf("%s came out ahead — %s ISK destroyed to %s",
			leader, FormatISK(lead), FormatISK(trail))
	} else {
		body = fmt.Sprintf("Evenly matched — %s ISK to %s",
			FormatISK(aggressorISK), FormatISK(defenderISK))
	}
	if retracted {
		return "Retracted · " + body
	}
	return body
}

// WarEnded announces a war ending.
func (e *Emitter) WarEnded(ctx context.Context, warID int32, aggressor, defender string, aggressorISK, defenderISK float64, retracted bool) {
	e.Emit(ctx, Spec{
		ID:    WarIDBase - int64(warID) + WarEndedOffset,
		Title: fmt.Sprintf("War over: %s vs %s", aggressor, defender),
		Body:  WarEndedBody(aggressor, defender, aggressorISK, defenderISK, retracted),
		Color: ColorInfo, Icon: "lucide:flag",
		LinkURL: fmt.Sprintf("/war/%d", warID),
		Expires: 180 * time.Minute,
	})
}

// TQOffline announces Tranquility going down.
func (e *Emitter) TQOffline(ctx context.Context, detail string) {
	e.Emit(ctx, Spec{
		ID: tqOfflineID, Title: "Tranquility Offline", Body: detail,
		Color: ColorDanger, Icon: "lucide:server-off", Expires: 30 * time.Minute,
	})
}

// TQOnline announces Tranquility coming back.
func (e *Emitter) TQOnline(ctx context.Context, detail string) {
	// Retract the offline notice first, or both sit in the ticker at once
	// saying opposite things.
	e.Expire(ctx, tqOfflineID)
	e.Emit(ctx, Spec{
		ID: tqOnlineID, Title: "Tranquility Online", Body: detail,
		Color: ColorSuccess, Icon: "lucide:server", Expires: 15 * time.Minute,
	})
}

func (e *Emitter) typeOf(id int32) (eve.Type, bool) {
	if e.Cache == nil || id == 0 {
		return eve.Type{}, false
	}
	return e.Cache.Type(id)
}

// locationText renders "System, Region", or just the system when the region is
// unknown.
func (e *Emitter) locationText(systemID int32) string {
	if e.Cache == nil {
		return "Unknown"
	}
	sys, ok := e.Cache.System(systemID)
	if !ok || sys.Name == "" {
		return "Unknown"
	}
	if region, ok := e.Cache.Region(sys.RegionID); ok && region.Name != "" {
		return sys.Name + ", " + region.Name
	}
	return sys.Name
}

// FormatISK renders a value the way the ticker shows it: 1.2T, 340.5B, 900M.
//
// Anything under a million keeps its digits — at that scale the exact number is
// short enough to read and rounding it to "0M" would say nothing.
func FormatISK(v float64) string {
	switch {
	case v >= 1_000_000_000_000:
		return fmt.Sprintf("%.1fT", v/1_000_000_000_000)
	case v >= 1_000_000_000:
		return fmt.Sprintf("%.1fB", v/1_000_000_000)
	case v >= 1_000_000:
		return fmt.Sprintf("%.0fM", v/1_000_000)
	}
	return formatCount(int(v))
}

// FormatCount renders an integer with thousands separators, matching the
// client's toLocaleString output.
func FormatCount(n int) string { return formatCount(n) }

func formatCount(n int) string {
	s := fmt.Sprintf("%d", n)
	neg := strings.HasPrefix(s, "-")
	s = strings.TrimPrefix(s, "-")

	var b strings.Builder
	for i, c := range s {
		if i > 0 && (len(s)-i)%3 == 0 {
			b.WriteByte(',')
		}
		b.WriteRune(c)
	}
	if neg {
		return "-" + b.String()
	}
	return b.String()
}
