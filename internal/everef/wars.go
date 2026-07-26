package everef

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"regexp"
	"strings"
	"time"

	"github.com/eve-kill/shrike/internal/configstore"
	"github.com/eve-kill/shrike/internal/eve"
	"github.com/eve-kill/shrike/internal/killmail"
	"github.com/eve-kill/shrike/internal/pgbulk"
	"github.com/jackc/pgx/v5/pgxpool"
)

// Wars, published as:
//
//	wars-current.json                     the active wars, metadata only
//	history/wars-<year>.tar.bz2           2003-2020, one archive per year
//	history/<year>/wars-<date>.tar.bz2    2021 on, one archive per day
//
// An archive mixes two kinds of document: the war itself, and every killmail
// fought under it. The killmails matter beyond the war record — a war kill is
// the only place war_id is ever stated, and without it the war never shows any
// activity.

const warsPath = "/wars"

// warsYearlyEnd is the last year published as a single archive. From 2021 the
// dataset is daily.
const warsYearlyEnd = 2020

// warsFirstYear is when EVE's war system began.
const warsFirstYear = 2003

type warDocument struct {
	WarID int32 `json:"war_id"`
	// ID appears instead of war_id on some older documents.
	ID            int32   `json:"id"`
	Declared      string  `json:"declared"`
	Started       string  `json:"started"`
	Finished      string  `json:"finished"`
	Retracted     string  `json:"retracted"`
	Mutual        bool    `json:"mutual"`
	OpenForAllies bool    `json:"open_for_allies"`
	Aggressor     warSide `json:"aggressor"`
	Defender      warSide `json:"defender"`
	Allies        []struct {
		AllianceID    int32 `json:"alliance_id"`
		CorporationID int32 `json:"corporation_id"`
	} `json:"allies"`
}

type warSide struct {
	AllianceID    int32   `json:"alliance_id"`
	CorporationID int32   `json:"corporation_id"`
	IskDestroyed  float64 `json:"isk_destroyed"`
	ShipsKilled   int32   `json:"ships_killed"`
}

func (w warDocument) id() int32 {
	if w.WarID != 0 {
		return w.WarID
	}
	return w.ID
}

var warColumns = []string{
	"war_id", "declared", "started", "finished", "retracted",
	"mutual", "open_for_allies",
	"aggressor_alliance_id", "aggressor_corporation_id",
	"aggressor_isk_destroyed", "aggressor_ships_killed",
	"defender_alliance_id", "defender_corporation_id",
	"defender_isk_destroyed", "defender_ships_killed",
	"updated_at",
}

// upsertWars writes war metadata and replaces each war's ally list.
//
// Allies are replaced rather than merged because the source is a full statement
// of who is on the defence: an ally that withdrew has to disappear, and merging
// would keep them forever.
func upsertWars(ctx context.Context, pool *pgxpool.Pool, docs []warDocument) (int64, error) {
	if len(docs) == 0 {
		return 0, nil
	}

	conn, err := pool.Acquire(ctx)
	if err != nil {
		return 0, err
	}
	defer conn.Release()

	tx, err := conn.Begin(ctx)
	if err != nil {
		return 0, err
	}
	defer tx.Rollback(ctx) //nolint:errcheck // no-op after commit

	const staging = "everef_staging_wars"
	if err := pgbulk.StagingTx(ctx, tx, staging, "wars"); err != nil {
		return 0, err
	}

	now := time.Now().UTC()
	w := pgbulk.NewCopier(ctx, tx, staging, warColumns)

	allyOf := make(map[int32][][2]any, len(docs))
	for _, d := range docs {
		id := d.id()
		if id == 0 {
			continue
		}
		if err := w.Add([]any{
			id,
			warTime(d.Declared), warTime(d.Started), warTime(d.Finished), warTime(d.Retracted),
			d.Mutual, d.OpenForAllies,
			nullID(d.Aggressor.AllianceID), nullID(d.Aggressor.CorporationID),
			d.Aggressor.IskDestroyed, d.Aggressor.ShipsKilled,
			nullID(d.Defender.AllianceID), nullID(d.Defender.CorporationID),
			d.Defender.IskDestroyed, d.Defender.ShipsKilled,
			now,
		}); err != nil {
			return 0, err
		}

		// Recorded even when empty, so that a war whose allies all left has
		// its list cleared rather than left stale.
		rows := make([][2]any, 0, len(d.Allies))
		for _, a := range d.Allies {
			if a.AllianceID == 0 && a.CorporationID == 0 {
				continue
			}
			rows = append(rows, [2]any{nullID(a.AllianceID), nullID(a.CorporationID)})
		}
		allyOf[id] = rows
	}

	if err := w.Flush(); err != nil {
		return 0, err
	}
	if _, err := tx.Exec(ctx, pgbulk.MergeSQL("wars", staging, warColumns,
		[]string{"war_id"}, pgbulk.DoUpdate)); err != nil {
		return 0, fmt.Errorf("merge wars: %w", err)
	}

	warIDs := make([]int32, 0, len(allyOf))
	for id := range allyOf {
		warIDs = append(warIDs, id)
	}
	if _, err := tx.Exec(ctx, `DELETE FROM war_allies WHERE war_id = ANY($1::int[])`, warIDs); err != nil {
		return 0, fmt.Errorf("clear war_allies: %w", err)
	}

	const allyStaging = "everef_staging_war_allies"
	if err := pgbulk.StagingTx(ctx, tx, allyStaging, "war_allies"); err != nil {
		return 0, err
	}

	aw := pgbulk.NewCopier(ctx, tx, allyStaging, []string{"war_id", "alliance_id", "corporation_id"})
	for id, allies := range allyOf {
		for _, a := range allies {
			if err := aw.Add([]any{id, a[0], a[1]}); err != nil {
				return 0, err
			}
		}
	}
	if err := aw.Flush(); err != nil {
		return 0, err
	}
	if aw.Written() > 0 {
		// war_allies has a surrogate id from a sequence and no natural key, so
		// this is a plain append onto the rows just deleted.
		if _, err := tx.Exec(ctx, fmt.Sprintf(
			`INSERT INTO public.war_allies (war_id, alliance_id, corporation_id)
             SELECT war_id, alliance_id, corporation_id FROM %s`, allyStaging)); err != nil {
			return 0, fmt.Errorf("insert war_allies: %w", err)
		}
	}

	if err := tx.Commit(ctx); err != nil {
		return 0, err
	}
	return w.Written(), nil
}

// warTime parses an ESI timestamp, mapping an absent one to NULL. A war with no
// finished date is still running, which is not the same as finishing at the
// zero time.
func warTime(s string) any {
	if s == "" {
		return nil
	}
	t, err := time.Parse(time.RFC3339, s)
	if err != nil {
		return nil
	}
	return t
}

// WarImport carries what a war import needs.
type WarImport struct {
	Pool   *pgxpool.Pool
	Client *Client
	Cache  *eve.Cache
	Prices *eve.Prices

	// SkipKillmails imports war metadata only. The killmails in these archives
	// are largely the same ones the daily killmail archives carry, so skipping
	// them is a reasonable way to refresh war records quickly.
	SkipKillmails bool
}

// ImportCurrentWars applies the active-war snapshot. It carries no killmails.
func (w *WarImport) ImportCurrentWars(ctx context.Context) (Result, error) {
	start := time.Now()
	res := Result{Name: "wars (current)"}

	// Published as an object keyed by war id, not an array.
	var byID map[string]warDocument
	if err := w.Client.JSON(ctx, w.Client.url(warsPath+"/wars-current.json"), &byID); err != nil {
		return res, err
	}

	docs := make([]warDocument, 0, len(byID))
	for _, d := range byID {
		docs = append(docs, d)
	}
	res.Seen = int64(len(docs))

	rows, err := upsertWars(ctx, w.Pool, docs)
	if err != nil {
		return res, err
	}
	res.Rows = rows
	res.Elapsed = time.Since(start).Round(time.Millisecond).String()
	return res, nil
}

// ImportWarArchive imports one archive, whether yearly or daily.
func (w *WarImport) ImportWarArchive(ctx context.Context, url, name string) (Result, error) {
	start := time.Now()
	res := Result{Name: name}

	var (
		wars  []warDocument
		batch []*killmail.Parsed
	)

	// The killmails in a war archive span whatever period it covers, so unlike
	// the daily killmail archives there is no single date to snapshot prices
	// at. Each mail resolves its own, which the per-day memo keeps cheap as
	// long as the archive is roughly in date order — and they are.
	flush := func() error {
		if len(batch) == 0 {
			return nil
		}
		out, err := killmail.InsertBatch(ctx, w.Pool, batch)
		if err != nil {
			return err
		}
		res.Related += out.Killmails

		// A war kill the live queue already stored kept a null war_id, because
		// the public killmail endpoint never states one. This is the only place
		// that association can be made.
		assigned, err := killmail.AssignWars(ctx, w.Pool, batch)
		if err != nil {
			return err
		}
		res.Adjusted += assigned

		batch = batch[:0]
		return nil
	}

	err := w.Client.WalkArchive(ctx, url, func(name string, data []byte) error {
		// One archive holds both kinds of document. A killmail is recognised by
		// having a victim; anything else with a war id is the war record.
		var probe struct {
			KillmailID int64            `json:"killmail_id"`
			Victim     *json.RawMessage `json:"victim"`
			WarID      int32            `json:"war_id"`
			ID         int32            `json:"id"`
		}
		if !decodeMember(data, &probe) {
			res.Failed++
			return nil
		}

		if probe.KillmailID != 0 && probe.Victim != nil {
			if w.SkipKillmails {
				res.Skipped++
				return nil
			}
			var km killmail.ESIKillmail
			if !decodeMember(data, &km) || km.KillmailID == 0 {
				res.Failed++
				return nil
			}
			warID := km.WarID
			if warID == 0 {
				// Older archives leave it off the document and state it only in
				// the path: wars/<war_id>/killmails/<killmail_id>.json.
				warID = warIDFromPath(name)
			}
			parsed, err := killmail.Parse(ctx, w.Cache, w.Prices, &km, km.KillmailHash, warID)
			if err != nil {
				// A malformed or unresolvable killmail is isolated to that
				// member. War metadata and the other killmails in the archive
				// are still authoritative and should be retained.
				res.Failed++
				return nil
			}
			batch = append(batch, parsed)
			res.Seen++
			if len(batch) >= killmailBatch {
				return flush()
			}
			return nil
		}

		if probe.WarID != 0 || probe.ID != 0 {
			var doc warDocument
			if !decodeMember(data, &doc) {
				res.Failed++
				return nil
			}
			wars = append(wars, doc)
		}
		return nil
	})
	if errors.Is(err, ErrNotPublished) {
		res.Missing = true
		res.Elapsed = time.Since(start).Round(time.Millisecond).String()
		return res, nil
	}
	if err != nil {
		return res, err
	}

	// Wars before killmails: a killmail's war_id is only meaningful once the
	// war it points at exists.
	const warChunk = 500
	for i := 0; i < len(wars); i += warChunk {
		end := min(i+warChunk, len(wars))
		written, err := upsertWars(ctx, w.Pool, wars[i:end])
		if err != nil {
			return res, err
		}
		res.Rows += written
	}
	res.Seen += int64(len(wars))

	if err := flush(); err != nil {
		return res, err
	}

	res.Elapsed = time.Since(start).Round(time.Millisecond).String()
	return res, nil
}

var warPath = regexp.MustCompile(`wars/(\d+)/killmails/`)

func warIDFromPath(name string) int32 {
	m := warPath.FindStringSubmatch(name)
	if m == nil {
		return 0
	}
	var id int32
	if _, err := fmt.Sscanf(m[1], "%d", &id); err != nil {
		return 0
	}
	return id
}

var warDailyArchive = regexp.MustCompile(`(wars-\d{4}-\d{2}-\d{2}[^"/]*\.tar\.bz2)$`)

// ImportWarYears imports every archive covering the given years.
func (w *WarImport) ImportWarYears(ctx context.Context, fromYear, toYear int, reprocess bool, progress func(Result)) (Result, error) {
	start := time.Now()
	total := Result{Name: "wars"}

	if fromYear < warsFirstYear {
		fromYear = warsFirstYear
	}

	for year := fromYear; year <= toYear && year <= warsYearlyEnd; year++ {
		if err := ctx.Err(); err != nil {
			return total, err
		}
		url := w.Client.url(fmt.Sprintf("%s/history/wars-%d.tar.bz2", warsPath, year))
		r, err := w.ImportWarArchive(ctx, url, fmt.Sprintf("%d", year))
		if err != nil {
			return total, err
		}
		accumulate(&total, r)
		if progress != nil {
			progress(r)
		}
		if err := configstore.Set(ctx, w.Pool, configstore.KeyWarsLastYear, fmt.Sprintf("%d", year+1)); err != nil {
			return total, err
		}
	}

	for year := max(fromYear, warsYearlyEnd+1); year <= toYear; year++ {
		if err := ctx.Err(); err != nil {
			return total, err
		}
		dir := w.Client.url(fmt.Sprintf("%s/history/%d/", warsPath, year))
		files, err := w.Client.List(ctx, dir, warDailyArchive)
		if err != nil {
			if errors.Is(err, ErrNotPublished) {
				continue
			}
			return total, err
		}

		if !reprocess {
			// The bookmark is a filename, and the listing is sorted, so
			// "everything after it" is the set still to do.
			last, err := configstore.Get(ctx, w.Pool, configstore.KeyWarsLastDaily)
			if err != nil {
				return total, err
			}
			if last != "" {
				kept := files[:0]
				for _, f := range files {
					if f > last {
						kept = append(kept, f)
					}
				}
				files = kept
			}
		}

		for _, f := range files {
			if err := ctx.Err(); err != nil {
				return total, err
			}
			r, err := w.ImportWarArchive(ctx, dir+f, strings.TrimSuffix(f, ".tar.bz2"))
			if err != nil {
				return total, err
			}
			accumulate(&total, r)
			if progress != nil {
				progress(r)
			}
			if err := configstore.Set(ctx, w.Pool, configstore.KeyWarsLastDaily, f); err != nil {
				return total, err
			}
		}
		if err := configstore.Set(ctx, w.Pool, configstore.KeyWarsLastYear, fmt.Sprintf("%d", year+1)); err != nil {
			return total, err
		}
	}

	total.Elapsed = time.Since(start).Round(time.Millisecond).String()
	return total, nil
}

func accumulate(total *Result, r Result) {
	total.Seen += r.Seen
	total.Rows += r.Rows
	total.Related += r.Related
	total.Adjusted += r.Adjusted
	total.Skipped += r.Skipped
	total.Failed += r.Failed
}
