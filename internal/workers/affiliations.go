package workers

import (
	"context"
	"fmt"
	"time"

	"github.com/eve-kill/shrike/internal/entities"
	"github.com/eve-kill/shrike/internal/esi"
	"github.com/eve-kill/shrike/internal/queue"
)

const (
	affiliationBatchSize  = 1_000
	affiliationMaxPerTick = 10_000
)

// runAffiliationUpdate keeps the inexpensive bulk affiliation check on the
// same cadence as the TypeScript worker: characters seen in the last year are
// checked daily, and dormant characters every fourteen days.
//
// Touching unchanged rows is important. Without it, ORDER BY updated_at keeps
// selecting the same unchanged characters and everybody after the first batch
// starves forever.
func (d *Deps) runAffiliationUpdate(ctx context.Context) (string, error) {
	if d.Queue == nil {
		return "", fmt.Errorf("affiliation_update needs a queue to dispatch into")
	}
	if d.ESI == nil {
		return "", fmt.Errorf("affiliation_update needs an ESI client")
	}

	now := time.Now()
	active, err := affiliationCandidates(
		ctx,
		d,
		now.Add(-365*24*time.Hour),
		now.Add(-24*time.Hour),
		true,
		affiliationMaxPerTick,
	)
	if err != nil {
		return "", err
	}

	activeChanged, err := d.processAffiliationBatch(ctx, active)
	if err != nil {
		return "", err
	}

	remaining := affiliationMaxPerTick - len(active)
	var inactive []int32
	if remaining > 0 {
		inactive, err = affiliationCandidates(
			ctx,
			d,
			now.Add(-365*24*time.Hour),
			now.Add(-14*24*time.Hour),
			false,
			remaining,
		)
		if err != nil {
			return "", err
		}
	}

	inactiveChanged, err := d.processAffiliationBatch(ctx, inactive)
	if err != nil {
		return "", err
	}

	checked := len(active) + len(inactive)
	if checked == 0 {
		return "no characters due for an affiliation check", nil
	}
	return fmt.Sprintf(
		"%d checked (%d active, %d inactive), %d changed",
		checked,
		len(active),
		len(inactive),
		activeChanged+inactiveChanged,
	), nil
}

func affiliationCandidates(
	ctx context.Context,
	d *Deps,
	activeSince time.Time,
	staleBefore time.Time,
	active bool,
	limit int,
) ([]int32, error) {
	if limit <= 0 {
		return nil, nil
	}

	activityPredicate := `(last_active IS NULL OR last_active < $1)`
	if active {
		activityPredicate = `last_active IS NOT NULL AND last_active >= $1`
	}

	rows, err := d.Pool.Query(ctx, `
        SELECT character_id
        FROM characters
        WHERE deleted IS NOT TRUE
          AND character_id > 0
          AND (`+activityPredicate+`)
          AND (updated_at IS NULL OR updated_at < $2)
        ORDER BY updated_at ASC NULLS FIRST, character_id
        LIMIT $3`,
		activeSince,
		staleBefore,
		limit,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	out := make([]int32, 0, limit)
	for rows.Next() {
		var id int32
		if err := rows.Scan(&id); err != nil {
			return nil, err
		}
		out = append(out, id)
	}
	return out, rows.Err()
}

func (d *Deps) processAffiliationBatch(ctx context.Context, ids []int32) (int, error) {
	changed := 0
	for start := 0; start < len(ids); start += affiliationBatchSize {
		end := min(start+affiliationBatchSize, len(ids))
		n, err := d.checkAffiliationChunk(ctx, ids[start:end])
		if err != nil {
			return changed, err
		}
		changed += n
	}
	return changed, nil
}

func (d *Deps) checkAffiliationChunk(ctx context.Context, ids []int32) (int, error) {
	affiliations := d.fetchAffiliationsWithSplit(ctx, ids, 0)
	if len(affiliations) == 0 {
		if err := touchAffiliationCheckedAt(ctx, d, ids); err != nil {
			return 0, err
		}
		return 0, nil
	}

	cascade, unchanged, changed, err := classifyAffiliations(ctx, d, affiliations)
	if err != nil {
		return 0, err
	}
	if err := touchAffiliationCheckedAt(ctx, d, unchanged); err != nil {
		return 0, err
	}
	if _, err := d.dispatchCascade(ctx, cascade, queue.Live); err != nil {
		return 0, err
	}
	return changed, nil
}

// fetchAffiliationsWithSplit isolates invalid character ids without throwing
// away the other 999 useful results. ESI rejects the whole POST when one id is
// bad, so a bounded binary split matches the old worker's recovery behavior.
func (d *Deps) fetchAffiliationsWithSplit(
	ctx context.Context,
	ids []int32,
	attempt int,
) []esi.Affiliation {
	res, err := esi.FetchAffiliations(ctx, d.ESI, ids)
	if err == nil && res.OK() {
		return *res.Data
	}
	if attempt >= 3 || len(ids) <= 1 {
		return nil
	}

	mid := len(ids) / 2
	left := d.fetchAffiliationsWithSplit(ctx, ids[:mid], attempt+1)
	right := d.fetchAffiliationsWithSplit(ctx, ids[mid:], attempt+1)
	if len(left) == 0 && len(right) == 0 {
		return nil
	}
	return append(left, right...)
}

func classifyAffiliations(
	ctx context.Context,
	d *Deps,
	current []esi.Affiliation,
) (entities.Cascade, []int32, int, error) {
	if len(current) == 0 {
		return entities.Cascade{}, nil, 0, nil
	}

	ids := make([]int32, 0, len(current))
	for _, affiliation := range current {
		ids = append(ids, affiliation.CharacterID)
	}

	rows, err := d.Pool.Query(ctx, `
        SELECT character_id, coalesce(corporation_id, 0), coalesce(alliance_id, 0)
        FROM characters
        WHERE character_id = ANY($1::int[])`,
		ids,
	)
	if err != nil {
		return entities.Cascade{}, nil, 0, err
	}
	defer rows.Close()

	type stored struct {
		corporation int32
		alliance    int32
	}
	known := make(map[int32]stored, len(ids))
	for rows.Next() {
		var id int32
		var value stored
		if err := rows.Scan(&id, &value.corporation, &value.alliance); err != nil {
			return entities.Cascade{}, nil, 0, err
		}
		known[id] = value
	}
	if err := rows.Err(); err != nil {
		return entities.Cascade{}, nil, 0, err
	}

	var cascade entities.Cascade
	var unchanged []int32
	corporations := make(map[int32]struct{})
	alliances := make(map[int32]struct{})
	changed := 0

	for _, affiliation := range current {
		value, exists := known[affiliation.CharacterID]
		if exists &&
			value.corporation == affiliation.CorporationID &&
			value.alliance == affiliation.AllianceID {
			unchanged = append(unchanged, affiliation.CharacterID)
			continue
		}

		changed++
		cascade.Characters = append(cascade.Characters, affiliation.CharacterID)
		if entities.IsPlayerCorporation(affiliation.CorporationID) {
			corporations[affiliation.CorporationID] = struct{}{}
		}
		if affiliation.AllianceID > 0 {
			alliances[affiliation.AllianceID] = struct{}{}
		}
	}

	for id := range corporations {
		cascade.Corporations = append(cascade.Corporations, id)
	}
	for id := range alliances {
		cascade.Alliances = append(cascade.Alliances, id)
	}
	return cascade, unchanged, changed, nil
}

func touchAffiliationCheckedAt(ctx context.Context, d *Deps, ids []int32) error {
	if len(ids) == 0 {
		return nil
	}
	_, err := d.Pool.Exec(ctx, `
        UPDATE characters
        SET updated_at = now()
        WHERE character_id = ANY($1::int[])`,
		ids,
	)
	return err
}
