package killmail

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// BackfillPointShares recalculates attacker point shares for an inclusive,
// bounded killmail-ID range. The temporary-table update keeps each chunk one
// transaction and avoids issuing one UPDATE per attacker over the network.
func BackfillPointShares(ctx context.Context, pool *pgxpool.Pool, fromID, toID int64) (int64, int64, error) {
	rows, err := pool.Query(ctx, `
		SELECT k.killmail_id, coalesce(k.points, 0), a.attacker_index,
		       coalesce(a.character_id, 0), coalesce(a.damage_done, 0),
		       coalesce(a.final_blow, false)
		FROM killmails k
		JOIN killmail_attackers a ON a.killmail_id = k.killmail_id
		WHERE k.killmail_id BETWEEN $1 AND $2
		ORDER BY k.killmail_id, a.attacker_index`, fromID, toID)
	if err != nil {
		return 0, 0, err
	}
	defer rows.Close()

	type storedParticipant struct {
		index int32
		PointParticipant
	}
	type updateRow struct {
		killmailID int64
		index      int32
		points     int64
	}
	var updates []updateRow
	var currentID, currentPool, killmails int64
	var participants []storedParticipant
	flush := func() {
		if currentID == 0 {
			return
		}
		input := make([]PointParticipant, len(participants))
		for i := range participants {
			input[i] = participants[i].PointParticipant
		}
		shares := AllocatePoints(currentPool, DefaultParticipationBasisPoints, input)
		seen := make(map[int32]bool)
		for _, participant := range participants {
			points := int64(0)
			if participant.CharacterID != 0 && !seen[participant.CharacterID] {
				points = shares[participant.CharacterID]
				seen[participant.CharacterID] = true
			}
			updates = append(updates, updateRow{currentID, participant.index, points})
		}
		killmails++
	}
	for rows.Next() {
		var killmailID, poolPoints int64
		var participant storedParticipant
		if err := rows.Scan(&killmailID, &poolPoints, &participant.index,
			&participant.CharacterID, &participant.DamageDone, &participant.FinalBlow); err != nil {
			return 0, 0, err
		}
		if currentID != 0 && killmailID != currentID {
			flush()
			participants = participants[:0]
		}
		currentID, currentPool = killmailID, poolPoints
		participants = append(participants, participant)
	}
	if err := rows.Err(); err != nil {
		return 0, 0, err
	}
	flush()
	rows.Close()
	if len(updates) == 0 {
		return killmails, 0, nil
	}

	conn, err := pool.Acquire(ctx)
	if err != nil {
		return 0, 0, err
	}
	defer conn.Release()
	tx, err := conn.Begin(ctx)
	if err != nil {
		return 0, 0, err
	}
	defer tx.Rollback(ctx) //nolint:errcheck
	if _, err := tx.Exec(ctx, `CREATE TEMP TABLE point_share_updates (
		killmail_id bigint NOT NULL, attacker_index integer NOT NULL, points bigint NOT NULL
	) ON COMMIT DROP`); err != nil {
		return 0, 0, err
	}
	source := pgx.CopyFromSlice(len(updates), func(i int) ([]any, error) {
		row := updates[i]
		return []any{row.killmailID, row.index, row.points}, nil
	})
	if _, err := tx.CopyFrom(ctx, pgx.Identifier{"point_share_updates"},
		[]string{"killmail_id", "attacker_index", "points"}, source); err != nil {
		return 0, 0, fmt.Errorf("copy point shares: %w", err)
	}
	tag, err := tx.Exec(ctx, `
		UPDATE killmail_attackers a SET points = u.points
		FROM point_share_updates u
		WHERE a.killmail_id = u.killmail_id
		  AND a.attacker_index = u.attacker_index
		  AND a.points IS DISTINCT FROM u.points`)
	if err != nil {
		return 0, 0, err
	}
	if err := tx.Commit(ctx); err != nil {
		return 0, 0, err
	}
	return killmails, tag.RowsAffected(), nil
}
