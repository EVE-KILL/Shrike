package relay

import (
	"context"
	"encoding/json"
	"errors"
	"testing"
	"time"

	"github.com/eve-kill/shrike/internal/eve"
	"github.com/eve-kill/shrike/internal/killmail"
	"github.com/redis/go-redis/v9"
)

// Routing keys decide what a subscriber sees. Getting one wrong is not an
// error anybody notices — the kill simply never arrives for the people
// following that alliance, and the site looks quiet rather than broken.

func testKillmail() *killmail.Parsed {
	return &killmail.Parsed{
		Killmail: killmail.Killmail{
			KillmailID:          137000001,
			KillmailHash:        "abc",
			KillmailTime:        time.Date(2026, 7, 20, 12, 0, 0, 0, time.UTC),
			SolarSystemID:       30000142,
			ConstellationID:     20000020,
			RegionID:            10000002,
			VictimCharacterID:   90000001,
			VictimCorporationID: 98000001,
			VictimAllianceID:    99000001,
			VictimFactionID:     500001,
			VictimShipTypeID:    17738,
			VictimShipGroupID:   27, // battleship
			TotalValue:          6_000_000_000,
			AttackerCount:       2,
		},
		Attackers: []killmail.Attacker{
			{CharacterID: 90000002, CorporationID: 98000002, AllianceID: 99000002, FinalBlow: true},
			{CharacterID: 90000003, CorporationID: 98000002, FactionID: 500002},
		},
	}
}

func has(keys []string, want string) bool {
	for _, k := range keys {
		if k == want {
			return true
		}
	}
	return false
}

// Every kill carries "all", which is what an unfiltered live feed subscribes to.
func TestEveryKillmailIsRoutedToAll(t *testing.T) {
	keys := RoutingKeys(testKillmail(), 0.9, true)
	if !has(keys, "all") {
		t.Error("no \"all\" key — the unfiltered live feed would receive nothing")
	}
}

// Entity keys are what "follow this alliance" is built on, and victim and
// attacker are separate namespaces so a subscriber can ask for losses only.
func TestEntityKeysCoverVictimAndAttackers(t *testing.T) {
	keys := RoutingKeys(testKillmail(), 0.9, true)

	for _, want := range []string{
		"victim.90000001", "victim.98000001", "victim.99000001", "victim.500001",
		"attacker.90000002", "attacker.98000002", "attacker.99000002",
		"attacker.90000003", "attacker.500002",
	} {
		if !has(keys, want) {
			t.Errorf("missing %q — anyone following that entity would not see this kill", want)
		}
	}

	// An attacker must not appear under the victim namespace, or "show me my
	// losses" would deliver kills.
	if has(keys, "victim.90000002") {
		t.Error("an attacker was routed as a victim")
	}
}

// A zero id is absent, not entity zero.
func TestZeroEntityIDsAreNotRouted(t *testing.T) {
	p := testKillmail()
	p.Killmail.VictimAllianceID = 0
	p.Attackers[1].AllianceID = 0

	keys := RoutingKeys(p, 0.9, true)
	for _, bad := range []string{"victim.0", "attacker.0"} {
		if has(keys, bad) {
			t.Errorf("a zero id produced the key %q", bad)
		}
	}
}

// Value brackets are cumulative: a 12b kill is both 10b and 5b.
func TestValueBracketsAreCumulative(t *testing.T) {
	cases := []struct {
		value   float64
		want10b bool
		want5b  bool
	}{
		{12_000_000_000, true, true},
		{10_000_000_000, true, true}, // the boundary is inclusive
		{6_000_000_000, false, true},
		{5_000_000_000, false, true},
		{4_999_999_999, false, false},
		{0, false, false},
	}

	for _, c := range cases {
		p := testKillmail()
		p.Killmail.TotalValue = c.value
		keys := RoutingKeys(p, 0.9, true)

		if got := has(keys, "10b"); got != c.want10b {
			t.Errorf("value %.0f: 10b = %v, want %v", c.value, got, c.want10b)
		}
		if got := has(keys, "5b"); got != c.want5b {
			t.Errorf("value %.0f: 5b = %v, want %v", c.value, got, c.want5b)
		}
	}
}

// Security banding must match what the killlist pages use, or a subscriber's
// "highsec only" feed disagrees with the highsec page.
func TestSecurityBanding(t *testing.T) {
	cases := []struct {
		security float64
		want     string
	}{
		{1.0, "highsec"},
		{0.45, "highsec"},
		{0.449, "lowsec"},
		{0.1, "lowsec"},
		{0.0, "lowsec"}, // 0.0 bands as lowsec here, matching the TypeScript
		{-0.1, "nullsec"},
		{-1.0, "nullsec"},
	}

	for _, c := range cases {
		keys := RoutingKeys(testKillmail(), c.security, true)
		if !has(keys, c.want) {
			t.Errorf("security %.3f did not produce %q (keys: %v)", c.security, c.want, keys)
		}
		for _, other := range []string{"highsec", "lowsec", "nullsec"} {
			if other != c.want && has(keys, other) {
				t.Errorf("security %.3f produced both %q and %q", c.security, c.want, other)
			}
		}
	}
}

// A system we do not know is not banded at all, rather than defaulting to
// nullsec — which would put every unknown system in the nullsec feed.
func TestUnknownSecurityIsNotBanded(t *testing.T) {
	keys := RoutingKeys(testKillmail(), 0, false)
	for _, band := range []string{"highsec", "lowsec", "nullsec"} {
		if has(keys, band) {
			t.Errorf("an unknown system was banded as %q", band)
		}
	}
}

// Ship classes overlap by design and a kill must carry every class it belongs
// to, or a subscriber filtering on one of them misses it.
func TestShipClassKeysOverlap(t *testing.T) {
	cases := []struct {
		group  int32
		want   []string
		absent []string
	}{
		// A freighter is big, but is tech I — 513 is not in the T2 set.
		{513, []string{"big", "freighters"}, []string{"t2", "t1"}},
		// A jump freighter is big, a freighter, and tech II.
		{902, []string{"big", "freighters", "t2"}, nil},
		// A titan is big and a capital-adjacent class of its own.
		{30, []string{"big", "titans"}, []string{"freighters"}},
		// An ordinary battleship is none of the special classes.
		{27, []string{"battleships", "t1"}, []string{"big", "freighters", "titans"}},
	}

	for _, c := range cases {
		p := testKillmail()
		p.Killmail.VictimShipGroupID = c.group
		keys := RoutingKeys(p, 0.9, true)

		for _, want := range c.want {
			if !has(keys, want) {
				t.Errorf("group %d did not produce %q", c.group, want)
			}
		}
		for _, absent := range c.absent {
			if has(keys, absent) {
				t.Errorf("group %d produced %q, which it does not belong to", c.group, absent)
			}
		}
	}
}

// Location keys let a subscriber watch one system or one region.
func TestLocationKeys(t *testing.T) {
	keys := RoutingKeys(testKillmail(), 0.9, true)
	for _, want := range []string{"system.30000142", "region.10000002", "constellation.20000020"} {
		if !has(keys, want) {
			t.Errorf("missing %q", want)
		}
	}
}

// The key list must be stable between runs. Two identical killmails producing
// different JSON makes the payload impossible to compare or cache by content.
func TestRoutingKeysAreSorted(t *testing.T) {
	for range 5 {
		keys := RoutingKeys(testKillmail(), 0.9, true)
		for i := 1; i < len(keys); i++ {
			if keys[i-1] > keys[i] {
				t.Fatalf("keys are not sorted: %q before %q", keys[i-1], keys[i])
			}
		}
	}
}

func TestBuildKilllistRowUsesISOTimeAndCanonicalNPCFinalBlowNulls(t *testing.T) {
	p := testKillmail()
	p.Attackers = []killmail.Attacker{{FinalBlow: true}}

	row := BuildKilllistRow(
		p,
		eve.NewCache(eve.CacheData{}),
		nil,
		EntityNames{
			Characters:   map[int32]string{},
			Corporations: map[int32]string{},
			Alliances:    map[int32]string{},
		},
	)

	if row.KillmailTime != "2026-07-20T12:00:00.000Z" {
		t.Errorf("killmail_time = %q, want TypeScript toISOString shape", row.KillmailTime)
	}
	if row.VictimFactionID == nil || *row.VictimFactionID != 500001 {
		t.Errorf("victim_faction_id = %v, want 500001", row.VictimFactionID)
	}
	for name, id := range map[string]*int32{
		"character":   row.FinalBlowCharacterID,
		"corporation": row.FinalBlowCorporationID,
		"alliance":    row.FinalBlowAllianceID,
		"ship":        row.FinalBlowShipTypeID,
	} {
		if id != nil {
			t.Errorf("NPC final-blow %s id = %v, want canonical REST null", name, *id)
		}
	}
}

func TestBuildKillmailEventMatchesTheHydratedWireShape(t *testing.T) {
	p := testKillmail()
	p.Killmail.VictimDamageTaken = 1234
	p.Killmail.Position = &killmail.ESIPosition{X: 1, Y: 2, Z: 3}
	security := -1.25
	p.Attackers[0].ShipTypeID = 24688
	p.Attackers[0].WeaponTypeID = 2048
	p.Attackers[0].SecurityStatus = &security
	p.Items = []killmail.Item{
		{TypeID: 34, FlagID: 5, QuantityDropped: 7, Singleton: 0},
		{TypeID: 35, FlagID: 87, QuantityDestroyed: 1, Singleton: 2},
	}

	cache := eve.NewCache(eve.CacheData{
		Types: map[int32]eve.Type{
			17738: {Name: "Machariel", GroupID: 27, CategoryID: 6},
			24688: {Name: "Rokh", GroupID: 27, CategoryID: 6},
			2048:  {Name: "Blaster", GroupID: 53, CategoryID: 7},
			34:    {Name: "Tritanium", GroupID: 18, CategoryID: 4},
			35:    {Name: "Pyerite", GroupID: 18, CategoryID: 4},
		},
		Groups: map[int32]eve.Group{
			27: {Name: "Battleship"},
			53: {Name: "Energy Weapon"},
			18: {Name: "Mineral"},
		},
		Systems: map[int32]eve.System{
			30000142: {Name: "Jita", Security: 0.9},
		},
		Constellations: map[int32]eve.Constellation{
			20000020: {Name: "Kimotoro"},
		},
		Regions: map[int32]eve.Region{
			10000002: {Name: "The Forge"},
		},
	})
	names := EntityNames{
		Characters: map[int32]string{
			90000001: "Victim", 90000002: "Final Blow", 90000003: "Wingmate",
		},
		Corporations: map[int32]string{
			98000001: "Victim Corp", 98000002: "Attacker Corp",
		},
		Alliances: map[int32]string{
			99000001: "Victim Alliance", 99000002: "Attacker Alliance",
		},
	}

	event := BuildKillmailEvent(p, cache, names, func(id int32) float64 {
		return float64(id) * 10
	})

	if event.Event != "killmail" || event.KillmailID != p.Killmail.KillmailID {
		t.Fatalf("event identity = %q/%d, want killmail/%d",
			event.Event, event.KillmailID, p.Killmail.KillmailID)
	}
	if event.SolarSystemName == nil || *event.SolarSystemName != "Jita" ||
		event.ConstellationName == nil || *event.ConstellationName != "Kimotoro" ||
		event.RegionName == nil || *event.RegionName != "The Forge" {
		t.Errorf("location was not fully hydrated: %+v", event)
	}
	if event.Victim.ShipTypeName == nil || *event.Victim.ShipTypeName != "Machariel" ||
		event.Victim.CharacterName == nil || *event.Victim.CharacterName != "Victim" {
		t.Errorf("victim was not fully hydrated: %+v", event.Victim)
	}
	if len(event.Victim.Items) != 2 || event.Victim.Items[0].Value != 340 ||
		event.Victim.Items[1].Value != 0.01 {
		t.Errorf("items = %+v, want market price and singleton-copy price", event.Victim.Items)
	}
	if len(event.Attackers) != 2 ||
		event.Attackers[0].CharacterName == nil ||
		*event.Attackers[0].CharacterName != "Final Blow" ||
		event.Attackers[0].SecurityStatus != security ||
		event.Attackers[0].WeaponTypeName == nil ||
		*event.Attackers[0].WeaponTypeName != "Blaster" {
		t.Errorf("attackers were not fully hydrated: %+v", event.Attackers)
	}

	body, err := json.Marshal(event)
	if err != nil {
		t.Fatal(err)
	}
	var wire map[string]any
	if err := json.Unmarshal(body, &wire); err != nil {
		t.Fatal(err)
	}
	if _, ok := wire["victim"]; !ok {
		t.Error("wire payload has no victim object")
	}
	if got := wire["killmail_time"]; got != "2026-07-20T12:00:00.000Z" {
		t.Errorf("killmail_time = %v, want TypeScript toISOString shape", got)
	}
}

// --- Publishing ---

// The relay subscribes to channels prefixed ws:, so a publish without it goes
// nowhere and silently.
func TestPublishPrefixesTheChannel(t *testing.T) {
	rdb, mock := newMockRedis(t)
	p := &Publisher{Redis: rdb}

	p.Publish(context.Background(), ChannelKilllist, []string{"all"}, map[string]int{"x": 1})

	if len(mock.published) != 1 {
		t.Fatalf("published %d messages, want 1", len(mock.published))
	}
	if got := mock.published[0].channel; got != "ws:killlist" {
		t.Errorf("published to %q, want %q — the relay subscribes to the ws: prefix "+
			"and would never see this", got, "ws:killlist")
	}
}

func TestFullKillmailChannelUsesTheTypeScriptPluralName(t *testing.T) {
	if ChannelKillmail != "killmails" {
		t.Errorf("ChannelKillmail = %q, want %q", ChannelKillmail, "killmails")
	}
}

// A publish failure must never surface. The caller has already stored the
// killmail; failing the job to re-send a notification would redo all of it.
func TestPublishSwallowsFailures(t *testing.T) {
	rdb, mock := newMockRedis(t)
	mock.err = errors.New("redis is down")

	var reported int
	p := &Publisher{Redis: rdb, OnError: func(string, error) { reported++ }}

	// The absence of a panic or a return value is the assertion.
	p.Publish(context.Background(), ChannelKilllist, nil, map[string]int{"x": 1})

	if reported != 1 {
		t.Errorf("the failure was reported %d times, want 1 — it must be visible in "+
			"the log even though it is not returned", reported)
	}
}

// A nil publisher and a nil client are both valid: the CLI runs jobs with no
// relay, and every call site would otherwise need a guard.
func TestNilPublisherIsSafe(t *testing.T) {
	var p *Publisher
	p.Publish(context.Background(), ChannelKilllist, nil, nil)
	p.PublishRaw(context.Background(), ChannelFeed, nil)

	empty := &Publisher{}
	empty.Publish(context.Background(), ChannelKilllist, nil, nil)
}

// Routing keys must serialise as [] rather than null, so a subscriber decoding
// into a typed array does not have to special-case it.
func TestEmptyRoutingKeysSerialiseAsAnArray(t *testing.T) {
	rdb, mock := newMockRedis(t)
	p := &Publisher{Redis: rdb}

	p.Publish(context.Background(), ChannelStatus, nil, map[string]bool{"ok": true})

	var decoded map[string]json.RawMessage
	if err := json.Unmarshal([]byte(mock.published[0].payload), &decoded); err != nil {
		t.Fatal(err)
	}
	if string(decoded["routing_keys"]) != "[]" {
		t.Errorf("routing_keys serialised as %s, want []", decoded["routing_keys"])
	}
}

// The feed payload is flat rather than wrapped, because its subscribers parse
// it at the top level and predate the envelope.
func TestFeedPayloadIsNotWrapped(t *testing.T) {
	rdb, mock := newMockRedis(t)
	p := &Publisher{Redis: rdb}

	p.PublishRaw(context.Background(), ChannelFeed, FeedNotice{
		Seq: 42, KillmailID: 137000001, RoutingKeys: []string{"all"},
	})

	var decoded map[string]any
	if err := json.Unmarshal([]byte(mock.published[0].payload), &decoded); err != nil {
		t.Fatal(err)
	}
	if _, wrapped := decoded["data"]; wrapped {
		t.Error("the feed payload was wrapped in an envelope — its subscribers read " +
			"seq and killmail_id at the top level")
	}
	if decoded["seq"] != float64(42) {
		t.Errorf("seq = %v, want 42 at the top level", decoded["seq"])
	}
}

// --- A Redis stand-in ---

type publishedMessage struct{ channel, payload string }

type mockRedis struct {
	published []publishedMessage
	err       error
}

// newMockRedis returns a client whose PUBLISH is intercepted.
//
// go-redis has no injection point for this, so the hook interface is used —
// which exercises the real command encoding rather than stubbing the client
// out entirely.
func newMockRedis(t *testing.T) (*redis.Client, *mockRedis) {
	t.Helper()

	m := &mockRedis{}
	rdb := redis.NewClient(&redis.Options{Addr: "127.0.0.1:1", MaxRetries: -1})
	rdb.AddHook(&captureHook{m: m})
	t.Cleanup(func() { _ = rdb.Close() })
	return rdb, m
}

type captureHook struct{ m *mockRedis }

func (h *captureHook) DialHook(next redis.DialHook) redis.DialHook { return next }

func (h *captureHook) ProcessHook(next redis.ProcessHook) redis.ProcessHook {
	return func(ctx context.Context, cmd redis.Cmder) error {
		if cmd.Name() == "publish" {
			args := cmd.Args()
			if len(args) >= 3 {
				channel, _ := args[1].(string)
				payload := ""
				switch v := args[2].(type) {
				case string:
					payload = v
				case []byte:
					payload = string(v)
				}
				h.m.published = append(h.m.published, publishedMessage{channel, payload})
			}
		}
		if h.m.err != nil {
			cmd.SetErr(h.m.err)
			return h.m.err
		}
		return nil
	}
}

func (h *captureHook) ProcessPipelineHook(next redis.ProcessPipelineHook) redis.ProcessPipelineHook {
	return next
}
