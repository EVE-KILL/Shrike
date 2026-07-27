package esi

import (
	"context"
	"fmt"
	"net/url"
)

// The typed endpoints.
//
// Each is a thin wrapper naming a path and a shape. They live here rather than
// at the call sites so that a schema change is one edit, and so the pipeline's
// group classification stays in step with the paths actually used.
//
// Killmails are deliberately absent: their response type is shared with the
// parser and lives in internal/killmail, so callers use
// Get[killmail.ESIKillmail] with KillmailPath rather than have this package
// depend on the domain it serves.

// Character is /characters/{id}/.
type Character struct {
	Name           string  `json:"name"`
	Description    string  `json:"description"`
	Birthday       string  `json:"birthday"`
	Gender         string  `json:"gender"`
	RaceID         int32   `json:"race_id"`
	BloodlineID    int32   `json:"bloodline_id"`
	SecurityStatus float64 `json:"security_status"`
	Title          string  `json:"title"`
	CorporationID  int32   `json:"corporation_id"`
	AllianceID     int32   `json:"alliance_id"`
	FactionID      int32   `json:"faction_id"`
}

// Corporation is /corporations/{id}/.
//
// The shape straddles a schema change CCP made on 2026-07-21, and both forms
// have to be understood because the compatibility date rolls forward on its own.
// Corporation() pins the newer one; the older fields stay decodable so a cached
// body written before the change still parses.
type Corporation struct {
	Name          string `json:"name"`
	Ticker        string `json:"ticker"`
	Description   string `json:"description"`
	DateFounded   string `json:"date_founded"`
	URL           string `json:"url"`
	CEOID         int32  `json:"ceo_id"` // omitted for closed corporations on 2026-07-21+
	CreatorID     int32  `json:"creator_id"`
	HomeStationID int32  `json:"home_station_id"`
	AllianceID    int32  `json:"alliance_id"`
	MemberCount   int32  `json:"member_count"`
	Shares        int64  `json:"shares"`
	WarEligible   *bool  `json:"war_eligible"`
	State         string `json:"state"`
	Type          string `json:"type"`
	FriendlyFire  string `json:"friendly_fire"`

	// FactionID is the pre-2026-07-21 name; EnlistedFactionID replaced it.
	FactionID         int32 `json:"faction_id"`
	EnlistedFactionID int32 `json:"enlisted_faction_id"`

	// TaxRate is a fraction 0–1; TaxRates are percentages 0–100. Read them
	// through TaxRateFraction rather than directly.
	TaxRate  *float64 `json:"tax_rate"`
	TaxRates *struct {
		ISK          float64 `json:"isk"`
		LoyaltyPoint float64 `json:"loyalty_point"`
	} `json:"tax_rates"`

	Palette map[string]any `json:"palette"`
}

// TaxRateFraction normalises both spellings onto the 0–1 fraction the database
// stores. Mixing the two would show a 10% corporation as taxing 1000%.
func (c Corporation) TaxRateFraction() float64 {
	if c.TaxRates != nil {
		return c.TaxRates.ISK / 100
	}
	if c.TaxRate != nil {
		return *c.TaxRate
	}
	return 0
}

// LoyaltyPointTaxFraction is nil before the 2026-07-21 shape, which had no such
// field — distinct from a corporation that taxes loyalty points at zero.
func (c Corporation) LoyaltyPointTaxFraction() *float64 {
	if c.TaxRates == nil {
		return nil
	}
	v := c.TaxRates.LoyaltyPoint / 100
	return &v
}

// Faction resolves whichever of the two faction fields is populated.
func (c Corporation) Faction() int32 {
	if c.EnlistedFactionID != 0 {
		return c.EnlistedFactionID
	}
	return c.FactionID
}

// Alliance is /alliances/{id}/.
type Alliance struct {
	Name                  string `json:"name"`
	Ticker                string `json:"ticker"`
	CreatorID             int32  `json:"creator_id"`
	CreatorCorporationID  int32  `json:"creator_corporation_id"`
	ExecutorCorporationID int32  `json:"executor_corporation_id"`
	DateFounded           string `json:"date_founded"`
	FactionID             int32  `json:"faction_id"`
}

// CorporationHistoryEntry is one row of a character's corporation history.
type CorporationHistoryEntry struct {
	RecordID      int32  `json:"record_id"`
	CorporationID int32  `json:"corporation_id"`
	StartDate     string `json:"start_date"`
	IsDeleted     bool   `json:"is_deleted"`
}

// AllianceHistoryEntry is one row of a corporation's alliance history.
//
// AllianceID is zero for the stretches a corporation was in no alliance, which
// are real history rather than gaps.
type AllianceHistoryEntry struct {
	RecordID   int32  `json:"record_id"`
	AllianceID int32  `json:"alliance_id"`
	StartDate  string `json:"start_date"`
	IsDeleted  bool   `json:"is_deleted"`
}

// Affiliation is one entry from the bulk affiliation lookup.
type Affiliation struct {
	CharacterID   int32 `json:"character_id"`
	CorporationID int32 `json:"corporation_id"`
	AllianceID    int32 `json:"alliance_id"`
	FactionID     int32 `json:"faction_id"`
}

// War is /wars/{id}/.
type War struct {
	ID            int32   `json:"id"`
	Declared      string  `json:"declared"`
	Started       string  `json:"started"`
	Finished      string  `json:"finished"`
	Retracted     string  `json:"retracted"`
	Mutual        bool    `json:"mutual"`
	OpenForAllies bool    `json:"open_for_allies"`
	Aggressor     WarSide `json:"aggressor"`
	Defender      WarSide `json:"defender"`
	Allies        []struct {
		AllianceID    int32 `json:"alliance_id"`
		CorporationID int32 `json:"corporation_id"`
	} `json:"allies"`
}

// WarSide is one belligerent.
type WarSide struct {
	AllianceID    int32   `json:"alliance_id"`
	CorporationID int32   `json:"corporation_id"`
	IskDestroyed  float64 `json:"isk_destroyed"`
	ShipsKilled   int32   `json:"ships_killed"`
}

// WarKillmailRef is an id/hash pair from a war's killmail list.
type WarKillmailRef struct {
	KillmailID   int64  `json:"killmail_id"`
	KillmailHash string `json:"killmail_hash"`
}

// --- Faction warfare ---

// FwWar is one faction pairing.
type FwWar struct {
	FactionID int32 `json:"faction_id"`
	AgainstID int32 `json:"against_id"`
}

// FwSystem is one contested system.
type FwSystem struct {
	SolarSystemID          int32  `json:"solar_system_id"`
	OwnerFactionID         int32  `json:"owner_faction_id"`
	OccupierFactionID      int32  `json:"occupier_faction_id"`
	Contested              string `json:"contested"`
	VictoryPoints          int32  `json:"victory_points"`
	VictoryPointsThreshold int32  `json:"victory_points_threshold"`
}

// FwStats is one faction's standing.
type FwStats struct {
	FactionID         int32 `json:"faction_id"`
	Pilots            int32 `json:"pilots"`
	SystemsControlled int32 `json:"systems_controlled"`
	Kills             struct {
		Yesterday int32 `json:"yesterday"`
		LastWeek  int32 `json:"last_week"`
		Total     int32 `json:"total"`
	} `json:"kills"`
	VictoryPoints struct {
		Yesterday int32 `json:"yesterday"`
		LastWeek  int32 `json:"last_week"`
		Total     int32 `json:"total"`
	} `json:"victory_points"`
}

// FwLeaderboardEntry is one ranked row.
type FwLeaderboardEntry struct {
	Amount        int32 `json:"amount"`
	FactionID     int32 `json:"faction_id"`
	CharacterID   int32 `json:"character_id"`
	CorporationID int32 `json:"corporation_id"`
}

// FwLeaderboard is the kills/victory-points ranking triplet.
type FwLeaderboard struct {
	Kills struct {
		Yesterday   []FwLeaderboardEntry `json:"yesterday"`
		LastWeek    []FwLeaderboardEntry `json:"last_week"`
		ActiveTotal []FwLeaderboardEntry `json:"active_total"`
	} `json:"kills"`
	VictoryPoints struct {
		Yesterday   []FwLeaderboardEntry `json:"yesterday"`
		LastWeek    []FwLeaderboardEntry `json:"last_week"`
		ActiveTotal []FwLeaderboardEntry `json:"active_total"`
	} `json:"victory_points"`
}

// Status is /status/, the state of Tranquility itself.
//
// It is what tells a worker fleet whether there is any point running: during a
// downtime or an unplanned outage every other endpoint returns errors, and
// spending the global error budget discovering that repeatedly is how a
// deployment gets itself 420'd for the length of the outage.
type Status struct {
	Players       int32  `json:"players"`
	ServerVersion string `json:"server_version"`
	StartTime     string `json:"start_time"`
	Error         string `json:"error"`
	// VIP marks a restricted login window after a patch — the server is up but
	// only staff can play, so no killmails will arrive.
	VIP bool `json:"vip"`
}

// --- Fetchers ---

// FetchStatus reads the state of Tranquility.
//
// It deliberately bypasses the ordinary ESI pause, cache, and limiter. This is
// the probe that decides whether those mechanisms should be paused, so making
// it obey their current state would prevent it from ever observing recovery.
func FetchStatus(ctx context.Context, c *Client) (Response[Status], error) {
	r, err := c.doProbe(ctx, "/latest/status/?datasource=tranquility")
	return typed[Status](r, err)
}

// FetchCharacter reads one character's public profile.
func FetchCharacter(ctx context.Context, c *Client, id int32) (Response[Character], error) {
	return Get[Character](ctx, c, fmt.Sprintf("/latest/characters/%d/", id))
}

// FetchCorporation reads one corporation.
//
// The compatibility date is pinned per-request rather than left to the client's
// rolling header, because this endpoint's newer shape carries fields the older
// one does not — palette, tax_rates, state, type, friendly_fire — and the
// database has columns for them. Removable once the rolling date passes
// 2026-07-21 on its own.
func FetchCorporation(ctx context.Context, c *Client, id int32) (Response[Corporation], error) {
	return Get[Corporation](ctx, c, fmt.Sprintf("/latest/corporations/%d/?compatibility_date=2026-07-21", id))
}

// FetchAlliance reads one alliance.
func FetchAlliance(ctx context.Context, c *Client, id int32) (Response[Alliance], error) {
	return Get[Alliance](ctx, c, fmt.Sprintf("/latest/alliances/%d/", id))
}

// FetchCharacterCorporationHistory reads every corporation a character has been in.
func FetchCharacterCorporationHistory(ctx context.Context, c *Client, id int32) (Response[[]CorporationHistoryEntry], error) {
	return Get[[]CorporationHistoryEntry](ctx, c, fmt.Sprintf("/latest/characters/%d/corporationhistory/", id))
}

// FetchCorporationAllianceHistory reads every alliance a corporation has been in.
func FetchCorporationAllianceHistory(ctx context.Context, c *Client, id int32) (Response[[]AllianceHistoryEntry], error) {
	return Get[[]AllianceHistoryEntry](ctx, c, fmt.Sprintf("/latest/corporations/%d/alliancehistory/", id))
}

// FetchAllianceList reads every alliance id in the game.
func FetchAllianceList(ctx context.Context, c *Client) (Response[[]int32], error) {
	return Get[[]int32](ctx, c, "/latest/alliances/")
}

// FetchAllianceCorporations reads an alliance's member corporations.
func FetchAllianceCorporations(ctx context.Context, c *Client, id int32) (Response[[]int32], error) {
	return Get[[]int32](ctx, c, fmt.Sprintf("/latest/alliances/%d/corporations/", id))
}

// FetchAffiliations resolves many characters' corporation and alliance at once.
//
// A POST because the id list would not fit in a URL, but a read: it is the
// cheapest way to notice that a thousand characters changed corp, at one request
// instead of a thousand.
func FetchAffiliations(ctx context.Context, c *Client, characterIDs []int32) (Response[[]Affiliation], error) {
	return Post[[]Affiliation](ctx, c, "/latest/characters/affiliation/", characterIDs)
}

// FetchWarList reads war ids, newest first. maxWarID pages backwards through
// history; zero starts at the most recent.
func FetchWarList(ctx context.Context, c *Client, maxWarID int32) (Response[[]int32], error) {
	path := "/latest/wars/"
	if maxWarID > 0 {
		path += "?max_war_id=" + url.QueryEscape(fmt.Sprint(maxWarID))
	}
	return Get[[]int32](ctx, c, path)
}

// FetchWar reads one war.
func FetchWar(ctx context.Context, c *Client, id int32) (Response[War], error) {
	return Get[War](ctx, c, fmt.Sprintf("/latest/wars/%d/", id))
}

// FetchWarKillmails reads one page of a war's killmail references.
func FetchWarKillmails(ctx context.Context, c *Client, warID int32, page int) (Response[[]WarKillmailRef], error) {
	return Get[[]WarKillmailRef](ctx, c, fmt.Sprintf("/latest/wars/%d/killmails/?page=%d", warID, page))
}

// KillmailPath builds the public killmail route.
//
// The trailing slash is not decoration: ESI redirects without it, and a redirect
// on every killmail doubles the request count against the error limit.
func KillmailPath(killmailID int64, hash string) string {
	return fmt.Sprintf("/latest/killmails/%d/%s/", killmailID, url.PathEscape(hash))
}

// FetchFwWars reads the faction pairings.
func FetchFwWars(ctx context.Context, c *Client) (Response[[]FwWar], error) {
	return Get[[]FwWar](ctx, c, "/latest/fw/wars/")
}

// FetchFwSystems reads every contested system.
func FetchFwSystems(ctx context.Context, c *Client) (Response[[]FwSystem], error) {
	return Get[[]FwSystem](ctx, c, "/latest/fw/systems/")
}

// FetchFwStats reads per-faction standings.
func FetchFwStats(ctx context.Context, c *Client) (Response[[]FwStats], error) {
	return Get[[]FwStats](ctx, c, "/latest/fw/stats/")
}

// FetchFwLeaderboards reads the faction leaderboard.
func FetchFwLeaderboards(ctx context.Context, c *Client) (Response[FwLeaderboard], error) {
	return Get[FwLeaderboard](ctx, c, "/latest/fw/leaderboards/")
}

// FetchFwLeaderboardsCharacters reads the character leaderboard.
func FetchFwLeaderboardsCharacters(ctx context.Context, c *Client) (Response[FwLeaderboard], error) {
	return Get[FwLeaderboard](ctx, c, "/latest/fw/leaderboards/characters/")
}

// FetchFwLeaderboardsCorporations reads the corporation leaderboard.
func FetchFwLeaderboardsCorporations(ctx context.Context, c *Client) (Response[FwLeaderboard], error) {
	return Get[FwLeaderboard](ctx, c, "/latest/fw/leaderboards/corporations/")
}
