package wars

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
)

// War interaction aggregation.
//
// A war's page answers "who did this war actually hurt" — how many kills each
// side landed, on whom, and for how much ISK. That is a rollup over every
// killmail fought under the war, maintained incrementally as the kills arrive
// because recomputing it per page view would scan years of mails.
//
// The subtlety is what counts as one interaction. A fleet of forty puts forty
// attacker rows on one killmail, and every one of them belongs to the same
// side of the same war. Counting per attacker would report forty kills for one
// dead ship and multiply the ISK by forty. So contributions are deduplicated
// down to the distinct facts the mail establishes, and each of those counts
// exactly once.

// Sides of a war. COMBINED is the war's own total, tracked alongside the two
// sides so a war page does not have to add them up — and because a kill by
// someone on neither side still happened in the war.
const (
	SideCombined  int16 = 0
	SideAggressor int16 = 1
	SideDefender  int16 = 2
)

// Interaction categories.
const (
	CategoryKilled   int16 = 0
	CategoryKilledBy int16 = 1
)

// Target types.
const (
	TargetCharacter   int16 = 0
	TargetCorporation int16 = 1
	TargetAlliance    int16 = 2
)

// Advisory lock coordinating incremental writers against a full rebuild.
//
// Live writers take it shared; the rebuild takes it exclusive. Without that,
// a rebuild that swaps the table contents could lose a kill ingested while it
// was computing — the increment would land on rows about to be replaced.
const (
	LockNamespace = 20_260_721
	LockKey       = 1
)

// Membership is who is on which side of a war.
//
// Defenders are sets because a war accumulates allies; aggressors are single
// ids because EVE allows exactly one.
type Membership struct {
	AggressorAllianceID    int32
	AggressorCorporationID int32
	DefenderAllianceIDs    map[int32]bool
	DefenderCorporationIDs map[int32]bool
}

// Side reports which side of the war an entity is on, or false for neither.
//
// Neither is a normal answer, not an error: third parties shoot into wars all
// the time, and a neutral who lands a kill belongs to the war's combined total
// without belonging to a side.
func (m Membership) Side(corporationID, allianceID int32) (int16, bool) {
	if (allianceID != 0 && allianceID == m.AggressorAllianceID) ||
		(corporationID != 0 && corporationID == m.AggressorCorporationID) {
		return SideAggressor, true
	}
	if (allianceID != 0 && m.DefenderAllianceIDs[allianceID]) ||
		(corporationID != 0 && m.DefenderCorporationIDs[corporationID]) {
		return SideDefender, true
	}
	return 0, false
}

// Contribution is one row to increment.
type Contribution struct {
	Side       int16
	Category   int16
	TargetType int16
	TargetID   int32
}

// Victim is the loss side of a killmail.
type Victim struct {
	CharacterID   int32
	CorporationID int32
	AllianceID    int32
}

// Attacker is one attacker row, reduced to what the aggregation reads.
type Attacker struct {
	CharacterID   int32
	CorporationID int32
	AllianceID    int32
	FinalBlow     bool
}

// Contributions produces the distinct interactions one killmail establishes.
//
// One per fact, not one per attacker. Forty attackers from one alliance produce
// a single aggressor-killed-victim contribution, so the count stays "ships
// destroyed" and the ISK stays "ISK destroyed" — which is what the war page
// claims they are. Counting per attacker would report forty kills for one dead
// ship and multiply the ISK by forty.
//
// That collapsing happens structurally rather than by filtering: the attacker
// sides are gathered into a set before anything is emitted, and every add below
// uses a distinct (side, category, target type) combination. The seen map is a
// guard against a future edit breaking that, not a working part of it — as
// written, it never fires.
func Contributions(v Victim, attackers []Attacker, m Membership) []Contribution {
	seen := make(map[Contribution]bool)
	var out []Contribution

	add := func(side, category, targetType int16, targetID int32) {
		if targetID == 0 {
			return
		}
		c := Contribution{Side: side, Category: category, TargetType: targetType, TargetID: targetID}
		if seen[c] {
			return
		}
		seen[c] = true
		out = append(out, c)
	}

	victimTargets := []struct {
		kind int16
		id   int32
	}{
		{TargetCharacter, v.CharacterID},
		{TargetCorporation, v.CorporationID},
		{TargetAlliance, v.AllianceID},
	}

	// The war as a whole killed this victim, whoever pulled the trigger.
	for _, t := range victimTargets {
		add(SideCombined, CategoryKilled, t.kind, t.id)
	}

	// Then once per side actually represented among the attackers. A mail with
	// attackers from both sides — which happens, when a third party's victim
	// dies to crossfire — credits both, once each.
	sides := make(map[int16]bool, 2)
	for _, a := range attackers {
		if side, ok := m.Side(a.CorporationID, a.AllianceID); ok {
			sides[side] = true
		}
	}
	for _, side := range []int16{SideAggressor, SideDefender} {
		if !sides[side] {
			continue
		}
		for _, t := range victimTargets {
			add(side, CategoryKilled, t.kind, t.id)
		}
	}

	// The killed-by direction is attributed to the final blow alone. Spreading
	// it over every attacker would make "killed by" mean "shot at by", and the
	// war page would credit a kill to everyone who scratched the paint.
	var finalBlow *Attacker
	for i := range attackers {
		if attackers[i].FinalBlow {
			finalBlow = &attackers[i]
			break
		}
	}
	if finalBlow != nil {
		finalBlowTargets := []struct {
			kind int16
			id   int32
		}{
			{TargetCharacter, finalBlow.CharacterID},
			{TargetCorporation, finalBlow.CorporationID},
			{TargetAlliance, finalBlow.AllianceID},
		}
		for _, t := range finalBlowTargets {
			add(SideCombined, CategoryKilledBy, t.kind, t.id)
		}
		// Attributed to the victim's side, not the killer's: this row answers
		// "what did this side lose ships to", so it hangs off the side that
		// took the loss.
		if side, ok := m.Side(v.CorporationID, v.AllianceID); ok {
			for _, t := range finalBlowTargets {
				add(side, CategoryKilledBy, t.kind, t.id)
			}
		}
	}

	return out
}

// ErrWarUnknown means the killmail names a war we have no row for.
var ErrWarUnknown = errors.New("war not found")

// AggregateKillmail folds one killmail into war_interactions.
//
// Returns false when the war itself is missing. The war list has historical
// gaps, and a killmail that names a war we have never imported is not a
// failure — it is work that has to wait for the war. Reporting that as "not
// done" leaves the effect pending so the repair path replays it once the war
// row exists, rather than marking it complete over a war that was never read.
func AggregateKillmail(ctx context.Context, tx pgx.Tx, killmailID int64) (bool, error) {
	var (
		warID        *int32
		totalValue   *float64
		killmailTime any
		v            Victim
	)
	err := tx.QueryRow(ctx, `
        SELECT war_id, total_value, killmail_time,
               coalesce(victim_character_id, 0), coalesce(victim_corporation_id, 0),
               coalesce(victim_alliance_id, 0)
        FROM killmails WHERE killmail_id = $1`, killmailID).
		Scan(&warID, &totalValue, &killmailTime, &v.CharacterID, &v.CorporationID, &v.AllianceID)
	if errors.Is(err, pgx.ErrNoRows) {
		return false, nil
	}
	if err != nil {
		return false, err
	}

	// Not a war kill. Nothing to aggregate, and the effect is genuinely done.
	if warID == nil || *warID == 0 {
		return true, nil
	}

	m, err := LoadMembership(ctx, tx, *warID)
	if err != nil {
		if errors.Is(err, ErrWarUnknown) {
			return false, nil
		}
		return false, err
	}

	attackers, err := loadAttackers(ctx, tx, killmailID)
	if err != nil {
		return false, err
	}

	isk := 0.0
	if totalValue != nil {
		isk = *totalValue
	}

	for _, c := range Contributions(v, attackers, m) {
		if _, err := tx.Exec(ctx, `
            INSERT INTO war_interactions (
                war_id, side, category, target_type, target_id,
                count, isk_value, last_killmail_id, last_killmail_time)
            VALUES ($1, $2, $3, $4, $5, 1, $6, $7, $8)
            ON CONFLICT (war_id, side, category, target_type, target_id) DO UPDATE SET
                count = war_interactions.count + 1,
                isk_value = war_interactions.isk_value + $6,
                last_killmail_id = CASE
                    WHEN war_interactions.last_killmail_time IS NULL
                         OR $8 > war_interactions.last_killmail_time THEN $7
                    WHEN $8 = war_interactions.last_killmail_time
                         AND (war_interactions.last_killmail_id IS NULL
                              OR $7 > war_interactions.last_killmail_id) THEN $7
                    ELSE war_interactions.last_killmail_id
                END,
                last_killmail_time = GREATEST(war_interactions.last_killmail_time, $8)`,
			*warID, c.Side, c.Category, c.TargetType, c.TargetID,
			isk, killmailID, killmailTime); err != nil {
			return false, fmt.Errorf("upsert war interaction: %w", err)
		}
	}

	return true, nil
}

// LoadMembership reads a war's sides, including accumulated allies.
func LoadMembership(ctx context.Context, tx pgx.Tx, warID int32) (Membership, error) {
	var m Membership
	var defAlliance, defCorp int32

	err := tx.QueryRow(ctx, `
        SELECT coalesce(aggressor_alliance_id, 0), coalesce(aggressor_corporation_id, 0),
               coalesce(defender_alliance_id, 0), coalesce(defender_corporation_id, 0)
        FROM wars WHERE war_id = $1`, warID).
		Scan(&m.AggressorAllianceID, &m.AggressorCorporationID, &defAlliance, &defCorp)
	if errors.Is(err, pgx.ErrNoRows) {
		return m, ErrWarUnknown
	}
	if err != nil {
		return m, err
	}

	m.DefenderAllianceIDs = map[int32]bool{}
	m.DefenderCorporationIDs = map[int32]bool{}
	if defAlliance != 0 {
		m.DefenderAllianceIDs[defAlliance] = true
	}
	if defCorp != 0 {
		m.DefenderCorporationIDs[defCorp] = true
	}

	rows, err := tx.Query(ctx,
		`SELECT coalesce(alliance_id, 0), coalesce(corporation_id, 0)
         FROM war_allies WHERE war_id = $1`, warID)
	if err != nil {
		return m, err
	}
	defer rows.Close()

	for rows.Next() {
		var alliance, corp int32
		if err := rows.Scan(&alliance, &corp); err != nil {
			return m, err
		}
		if alliance != 0 {
			m.DefenderAllianceIDs[alliance] = true
		}
		if corp != 0 {
			m.DefenderCorporationIDs[corp] = true
		}
	}
	return m, rows.Err()
}

func loadAttackers(ctx context.Context, tx pgx.Tx, killmailID int64) ([]Attacker, error) {
	rows, err := tx.Query(ctx, `
        SELECT coalesce(character_id, 0), coalesce(corporation_id, 0),
               coalesce(alliance_id, 0), final_blow
        FROM killmail_attackers WHERE killmail_id = $1
        ORDER BY attacker_index`, killmailID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []Attacker
	for rows.Next() {
		var a Attacker
		if err := rows.Scan(&a.CharacterID, &a.CorporationID, &a.AllianceID, &a.FinalBlow); err != nil {
			return nil, err
		}
		out = append(out, a)
	}
	return out, rows.Err()
}
