// Package entities turns ESI responses into the character, corporation and
// alliance rows the killboard stores.
//
// A killmail names entities by id and nothing else. Everything a page shows —
// names, tickers, corp membership, alliance history — comes from fetching those
// ids separately and keeping the result. This package is that half: fetch,
// persist, and report what else needs fetching as a result.
//
// It deliberately does not enqueue anything. A refresh returns the follow-up
// work it implies and leaves dispatching to the caller, so the same code serves
// a CLI walking the cascade synchronously and, later, a queue driving it.
package entities

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/eve-kill/shrike/internal/esi"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// PlayerCorporationIDMin is where player corporations start. Below it are NPC
// corporations, which every new character belongs to and which no amount of
// fetching will make interesting — refreshing them would spend the entire
// corporation budget on the same few hundred ids.
const PlayerCorporationIDMin int32 = 10_000_000

// IsPlayerCorporation reports whether an id is worth fetching.
func IsPlayerCorporation(id int32) bool { return id >= PlayerCorporationIDMin }

// StaleAfter is how long an entity stays usable before it is refetched. Names
// and tickers change rarely; a week is a compromise between a stale killboard
// and spending the whole ESI budget re-reading things that did not change.
const StaleAfter = 7 * 24 * time.Hour

// Cascade is the follow-up work a refresh implies.
//
// Returned rather than enqueued so this package has no opinion about how work is
// scheduled. Everything here is already deduplicated and filtered — an id in
// this struct is one the caller should actually fetch.
type Cascade struct {
	Characters           []int32 `json:"characters,omitempty"`
	Corporations         []int32 `json:"corporations,omitempty"`
	Alliances            []int32 `json:"alliances,omitempty"`
	CharacterHistories   []int32 `json:"character_histories,omitempty"`
	CorporationHistories []int32 `json:"corporation_histories,omitempty"`
}

// Empty reports whether there is nothing left to do.
func (c Cascade) Empty() bool {
	return len(c.Characters) == 0 && len(c.Corporations) == 0 && len(c.Alliances) == 0 &&
		len(c.CharacterHistories) == 0 && len(c.CorporationHistories) == 0
}

// Result reports one refresh.
type Result struct {
	ID      int32   `json:"id"`
	Kind    string  `json:"kind"`
	Name    string  `json:"name,omitempty"`
	Status  int     `json:"status"`
	Deleted bool    `json:"deleted,omitempty"`
	Rows    int64   `json:"rows,omitempty"`
	Cascade Cascade `json:"cascade,omitzero"`
}

// ErrTransient marks a failure worth retrying — ESI was unreachable or unwell,
// as opposed to answering that the entity does not exist.
var ErrTransient = errors.New("ESI request failed")

// Refresher fetches entities and writes them.
type Refresher struct {
	Pool *pgxpool.Pool
	ESI  *esi.Client
}

// Character fetches a character and stores it.
//
// A 404 or 400 means biomassed or never existed. That is recorded as deleted
// rather than left absent, so the id is not retried forever — a killboard
// accumulates references to dead characters indefinitely.
func (r *Refresher) Character(ctx context.Context, id int32) (Result, error) {
	out := Result{ID: id, Kind: "character"}

	res, err := esi.FetchCharacter(ctx, r.ESI, id)
	if err != nil {
		return out, err
	}
	out.Status = res.Status

	if res.Status == 404 || res.Status == 400 {
		out.Deleted = true
		return out, r.markCharacterDeleted(ctx, id)
	}
	if !res.OK() {
		return out, fmt.Errorf("%w: character %d returned %d", ErrTransient, id, res.Status)
	}

	c := *res.Data
	out.Name = c.Name
	now := time.Now().UTC()

	// The two history markers come back from the write so the decision to
	// refetch history is made against what was stored a moment ago, not against
	// a separate read that could race with another worker.
	var fetchedAt *time.Time
	var syncedCorp *int32
	err = r.Pool.QueryRow(ctx, `
        INSERT INTO characters (
            character_id, name, description, birthday, gender, race_id,
            bloodline_id, security_status, title, corporation_id, alliance_id,
            faction_id, updated_at
        ) VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13)
        ON CONFLICT (character_id) DO UPDATE SET
            name = EXCLUDED.name,
            description = EXCLUDED.description,
            birthday = EXCLUDED.birthday,
            gender = EXCLUDED.gender,
            race_id = EXCLUDED.race_id,
            bloodline_id = EXCLUDED.bloodline_id,
            security_status = EXCLUDED.security_status,
            title = EXCLUDED.title,
            corporation_id = EXCLUDED.corporation_id,
            alliance_id = EXCLUDED.alliance_id,
            faction_id = EXCLUDED.faction_id,
            updated_at = EXCLUDED.updated_at
        RETURNING corporation_history_fetched_at, corporation_history_synced_corporation_id`,
		id, c.Name, nullText(c.Description), esiTime(c.Birthday), c.Gender, nullID(c.RaceID),
		nullID(c.BloodlineID), c.SecurityStatus, nullText(c.Title), nullID(c.CorporationID),
		nullID(c.AllianceID), nullID(c.FactionID), now,
	).Scan(&fetchedAt, &syncedCorp)
	if err != nil {
		return out, fmt.Errorf("upsert character %d: %w", id, err)
	}
	out.Rows = 1

	// History is only worth refetching when it has never been fetched, or when
	// the character has moved since it was: history is append-only, so an
	// unchanged corporation means an unchanged history.
	if fetchedAt == nil || syncedCorp == nil || *syncedCorp != c.CorporationID {
		out.Cascade.CharacterHistories = []int32{id}
		if _, err := r.Pool.Exec(ctx,
			`UPDATE characters SET corporation_history_queued_at = $2 WHERE character_id = $1`,
			id, now); err != nil {
			return out, err
		}
	}

	if IsPlayerCorporation(c.CorporationID) {
		known, err := r.exists(ctx, "corporations", "corporation_id", c.CorporationID)
		if err != nil {
			return out, err
		}
		if !known {
			out.Cascade.Corporations = []int32{c.CorporationID}
		}
	}

	return out, nil
}

// Corporation fetches a corporation and stores it.
func (r *Refresher) Corporation(ctx context.Context, id int32) (Result, error) {
	out := Result{ID: id, Kind: "corporation"}

	res, err := esi.FetchCorporation(ctx, r.ESI, id)
	if err != nil {
		return out, err
	}
	out.Status = res.Status

	if res.Status == 404 || res.Status == 400 {
		out.Deleted = true
		_, err := r.Pool.Exec(ctx,
			`UPDATE corporations SET deleted = true, updated_at = now() WHERE corporation_id = $1`, id)
		return out, err
	}
	if !res.OK() {
		return out, fmt.Errorf("%w: corporation %d returned %d", ErrTransient, id, res.Status)
	}

	c := *res.Data
	out.Name = c.Name
	now := time.Now().UTC()

	var fetchedAt *time.Time
	var syncedAlliance *int32
	err = r.Pool.QueryRow(ctx, `
        INSERT INTO corporations (
            corporation_id, name, ticker, description, date_founded, url,
            ceo_id, creator_id, home_station_id, alliance_id, faction_id,
            member_count, shares, tax_rate, lp_tax_rate, war_eligible,
            friendly_fire, state, type, palette, updated_at
        ) VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14,$15,$16,$17,$18,$19,$20,$21)
        ON CONFLICT (corporation_id) DO UPDATE SET
            name = EXCLUDED.name,
            ticker = EXCLUDED.ticker,
            description = EXCLUDED.description,
            url = EXCLUDED.url,
            ceo_id = EXCLUDED.ceo_id,
            home_station_id = EXCLUDED.home_station_id,
            alliance_id = EXCLUDED.alliance_id,
            faction_id = EXCLUDED.faction_id,
            member_count = EXCLUDED.member_count,
            shares = EXCLUDED.shares,
            tax_rate = EXCLUDED.tax_rate,
            lp_tax_rate = EXCLUDED.lp_tax_rate,
            war_eligible = EXCLUDED.war_eligible,
            friendly_fire = EXCLUDED.friendly_fire,
            state = EXCLUDED.state,
            type = EXCLUDED.type,
            palette = EXCLUDED.palette,
            updated_at = EXCLUDED.updated_at
        RETURNING alliance_history_fetched_at, alliance_history_synced_alliance_id`,
		id, c.Name, c.Ticker, nullText(c.Description), esiTime(c.DateFounded), nullText(c.URL),
		nullID(c.CEOID), nullID(c.CreatorID), nullID(c.HomeStationID), nullID(c.AllianceID),
		nullID(c.Faction()), c.MemberCount, nullInt64(c.Shares), c.TaxRateFraction(),
		c.LoyaltyPointTaxFraction(), c.WarEligible, nullText(c.FriendlyFire),
		nullText(c.State), nullText(c.Type), jsonbOrNull(c.Palette), now,
	).Scan(&fetchedAt, &syncedAlliance)
	if err != nil {
		return out, fmt.Errorf("upsert corporation %d: %w", id, err)
	}
	out.Rows = 1

	// Note the comparison uses the raw value on both sides, including zero: a
	// corporation that left its alliance has alliance_id 0 here and NULL in the
	// column, and that transition must trigger a history refetch.
	if fetchedAt == nil || deref(syncedAlliance) != c.AllianceID {
		out.Cascade.CorporationHistories = []int32{id}
		if _, err := r.Pool.Exec(ctx,
			`UPDATE corporations SET alliance_history_queued_at = $2 WHERE corporation_id = $1`,
			id, now); err != nil {
			return out, err
		}
	}

	if c.AllianceID != 0 {
		known, err := r.exists(ctx, "alliances", "alliance_id", c.AllianceID)
		if err != nil {
			return out, err
		}
		if !known {
			out.Cascade.Alliances = []int32{c.AllianceID}
		}
	}

	return out, nil
}

// Alliance fetches an alliance and stores it.
func (r *Refresher) Alliance(ctx context.Context, id int32) (Result, error) {
	out := Result{ID: id, Kind: "alliance"}

	res, err := esi.FetchAlliance(ctx, r.ESI, id)
	if err != nil {
		return out, err
	}
	out.Status = res.Status

	if res.Status == 404 || res.Status == 400 {
		out.Deleted = true
		_, err := r.Pool.Exec(ctx,
			`UPDATE alliances SET deleted = true, updated_at = now() WHERE alliance_id = $1`, id)
		return out, err
	}
	if !res.OK() {
		return out, fmt.Errorf("%w: alliance %d returned %d", ErrTransient, id, res.Status)
	}

	a := *res.Data
	out.Name = a.Name

	// Only the mutable fields are updated. creator_id, creator_corporation_id
	// and date_founded are historical facts that ESI has been known to return
	// as null on a bad day, and overwriting them would lose data permanently.
	_, err = r.Pool.Exec(ctx, `
        INSERT INTO alliances (
            alliance_id, name, ticker, creator_id, creator_corporation_id,
            executor_corporation_id, date_founded, faction_id, updated_at
        ) VALUES ($1,$2,$3,$4,$5,$6,$7,$8,now())
        ON CONFLICT (alliance_id) DO UPDATE SET
            name = EXCLUDED.name,
            ticker = EXCLUDED.ticker,
            executor_corporation_id = EXCLUDED.executor_corporation_id,
            faction_id = EXCLUDED.faction_id,
            updated_at = now()`,
		id, a.Name, a.Ticker, nullID(a.CreatorID), nullID(a.CreatorCorporationID),
		nullID(a.ExecutorCorporationID), esiTime(a.DateFounded), nullID(a.FactionID))
	if err != nil {
		return out, fmt.Errorf("upsert alliance %d: %w", id, err)
	}
	out.Rows = 1
	return out, nil
}

// CharacterHistory fetches and stores a character's corporation history.
//
// force overrides the sync marker, which otherwise suppresses a fetch whose
// answer is already known.
func (r *Refresher) CharacterHistory(ctx context.Context, id int32, force bool) (Result, error) {
	out := Result{ID: id, Kind: "character_history"}

	var currentCorp *int32
	var deleted bool
	var fetchedAt *time.Time
	var syncedCorp *int32
	err := r.Pool.QueryRow(ctx, `
        SELECT corporation_id, coalesce(deleted, false),
               corporation_history_fetched_at, corporation_history_synced_corporation_id
        FROM characters WHERE character_id = $1`, id).
		Scan(&currentCorp, &deleted, &fetchedAt, &syncedCorp)
	if errors.Is(err, pgx.ErrNoRows) {
		// Nothing to attach a history to yet; the character fetch comes first.
		return out, nil
	}
	if err != nil {
		return out, err
	}
	if deleted {
		return out, nil
	}
	// A stored history row is not proof that a complete response was saved, so
	// only the explicit marker may suppress a request. This is also what
	// distinguishes a genuinely empty history from one never fetched.
	if !force && fetchedAt != nil && deref(syncedCorp) == deref(currentCorp) {
		out.Status = 304
		return out, nil
	}

	res, err := esi.FetchCharacterCorporationHistory(ctx, r.ESI, id)
	if err != nil {
		return out, err
	}
	out.Status = res.Status

	if res.Status == 404 || res.Status == 400 {
		out.Deleted = true
		_, err := r.Pool.Exec(ctx, `
            UPDATE characters SET deleted = true, corporation_history_queued_at = NULL
            WHERE character_id = $1`, id)
		return out, err
	}
	if !res.OK() {
		return out, fmt.Errorf("%w: character history %d returned %d", ErrTransient, id, res.Status)
	}

	entries := *res.Data
	// The synced marker is the corporation of the newest record, which is what
	// a later fetch compares against to decide whether anything moved.
	synced := deref(currentCorp)
	var newest time.Time
	for _, e := range entries {
		t, ok := parseESITime(e.StartDate)
		if !ok {
			return out, fmt.Errorf("character history %d has invalid start date %q",
				id, e.StartDate)
		}
		if t.After(newest) {
			newest = t
			synced = e.CorporationID
		}
	}

	tx, err := r.Pool.Begin(ctx)
	if err != nil {
		return out, err
	}
	defer tx.Rollback(ctx) //nolint:errcheck // no-op after commit

	for _, e := range entries {
		start, _ := parseESITime(e.StartDate) // validated above
		if _, err := tx.Exec(ctx, `
            INSERT INTO character_corporation_history (character_id, record_id, corporation_id, start_date)
            VALUES ($1,$2,$3,$4)
            ON CONFLICT (character_id, record_id) DO UPDATE SET
                corporation_id = EXCLUDED.corporation_id,
                start_date = EXCLUDED.start_date`,
			id, e.RecordID, e.CorporationID, start); err != nil {
			return out, err
		}
		out.Rows++
	}

	if _, err := tx.Exec(ctx, `
        UPDATE characters SET
            corporation_history_queued_at = NULL,
            corporation_history_fetched_at = now(),
            corporation_history_synced_corporation_id = $2
        WHERE character_id = $1`, id, nullID(synced)); err != nil {
		return out, err
	}

	return out, tx.Commit(ctx)
}

// CorporationHistory fetches and stores a corporation's alliance history.
func (r *Refresher) CorporationHistory(ctx context.Context, id int32, force bool) (Result, error) {
	out := Result{ID: id, Kind: "corporation_history"}

	var currentAlliance *int32
	var deleted bool
	var fetchedAt *time.Time
	var syncedAlliance *int32
	err := r.Pool.QueryRow(ctx, `
        SELECT alliance_id, coalesce(deleted, false),
               alliance_history_fetched_at, alliance_history_synced_alliance_id
        FROM corporations WHERE corporation_id = $1`, id).
		Scan(&currentAlliance, &deleted, &fetchedAt, &syncedAlliance)
	if errors.Is(err, pgx.ErrNoRows) {
		return out, nil
	}
	if err != nil {
		return out, err
	}
	if deleted {
		return out, nil
	}
	if !force && fetchedAt != nil && deref(syncedAlliance) == deref(currentAlliance) {
		out.Status = 304
		return out, nil
	}

	res, err := esi.FetchCorporationAllianceHistory(ctx, r.ESI, id)
	if err != nil {
		return out, err
	}
	out.Status = res.Status

	if res.Status == 404 || res.Status == 400 {
		out.Deleted = true
		_, err := r.Pool.Exec(ctx, `
            UPDATE corporations SET deleted = true, alliance_history_queued_at = NULL
            WHERE corporation_id = $1`, id)
		return out, err
	}
	if !res.OK() {
		return out, fmt.Errorf("%w: corporation history %d returned %d", ErrTransient, id, res.Status)
	}

	entries := *res.Data
	synced := deref(currentAlliance)
	var newest time.Time
	for _, e := range entries {
		t, ok := parseESITime(e.StartDate)
		if !ok {
			return out, fmt.Errorf("corporation history %d has invalid start date %q",
				id, e.StartDate)
		}
		if t.After(newest) {
			newest = t
			synced = e.AllianceID
		}
	}

	tx, err := r.Pool.Begin(ctx)
	if err != nil {
		return out, err
	}
	defer tx.Rollback(ctx) //nolint:errcheck // no-op after commit

	for _, e := range entries {
		start, _ := parseESITime(e.StartDate) // validated above
		// alliance_id stays nullable here: the stretches a corporation spent
		// in no alliance are real history, not missing data.
		if _, err := tx.Exec(ctx, `
            INSERT INTO corporation_alliance_history (corporation_id, record_id, alliance_id, start_date)
            VALUES ($1,$2,$3,$4)
            ON CONFLICT (corporation_id, record_id) DO UPDATE SET
                alliance_id = EXCLUDED.alliance_id,
                start_date = EXCLUDED.start_date`,
			id, e.RecordID, nullID(e.AllianceID), start); err != nil {
			return out, err
		}
		out.Rows++
	}

	if _, err := tx.Exec(ctx, `
        UPDATE corporations SET
            alliance_history_queued_at = NULL,
            alliance_history_fetched_at = now(),
            alliance_history_synced_alliance_id = $2
        WHERE corporation_id = $1`, id, nullID(synced)); err != nil {
		return out, err
	}

	return out, tx.Commit(ctx)
}

func (r *Refresher) markCharacterDeleted(ctx context.Context, id int32) error {
	_, err := r.Pool.Exec(ctx, `
        INSERT INTO characters (character_id, name, deleted, updated_at)
        VALUES ($1, 'Deleted', true, now())
        ON CONFLICT (character_id) DO UPDATE SET deleted = true, updated_at = now()`, id)
	return err
}

func (r *Refresher) exists(ctx context.Context, table, column string, id int32) (bool, error) {
	var found bool
	err := r.Pool.QueryRow(ctx,
		fmt.Sprintf(`SELECT true FROM %s WHERE %s = $1`, table, column), id).Scan(&found)
	if errors.Is(err, pgx.ErrNoRows) {
		return false, nil
	}
	return found, err
}

// --- Value helpers ---

// nullID keeps the zero-means-absent convention used throughout the codebase.
func nullID(v int32) any {
	if v == 0 {
		return nil
	}
	return v
}

func nullInt64(v int64) any {
	if v == 0 {
		return nil
	}
	return v
}

func nullText(s string) any {
	if s == "" {
		return nil
	}
	return s
}

// jsonbOrNull encodes a value for a jsonb column.
//
// Encoded here rather than handed to pgx as a Go map: pgx picks a codec from the
// Go type, and a bare map has no unambiguous mapping, so the choice is made
// explicit instead of left to inference.
func jsonbOrNull(v map[string]any) any {
	if len(v) == 0 {
		return nil
	}
	encoded, err := json.Marshal(v)
	if err != nil {
		return nil
	}
	return encoded
}

// esiTime converts an ESI timestamp for storage, mapping an absent or
// unparseable one to NULL rather than to the zero time — which Postgres would
// happily store as the year 1.
func esiTime(s string) any {
	t, ok := parseESITime(s)
	if !ok {
		return nil
	}
	return t
}

func parseESITime(s string) (time.Time, bool) {
	if s == "" {
		return time.Time{}, false
	}
	for _, layout := range []string{time.RFC3339, "2006-01-02T15:04:05Z", "2006-01-02"} {
		if t, err := time.Parse(layout, s); err == nil {
			return t, true
		}
	}
	return time.Time{}, false
}

func deref(v *int32) int32 {
	if v == nil {
		return 0
	}
	return *v
}
