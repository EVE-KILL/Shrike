package killmail

import (
	"context"
	"errors"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// Delayed killmails.
//
// Some ESI keys deliver killmails on a delay — a corporation can configure how
// long its losses stay private. A kill observed through such a key must not be
// published until the delay expires, so it is parked in esi_killmail_delayed
// and dispatched later by the killmail_delayed cron.
//
// The table is a queue in its own right, deliberately: the delay can be hours
// or days, which is far longer than anything worth holding in the job queue,
// and it has to survive restarts and redeploys.

// Ref is the id and hash pair needed to fetch a killmail.
type Ref struct {
	KillmailID   int64  `json:"killmail_id"`
	KillmailHash string `json:"killmail_hash"`
}

// Delay parks a killmail until now + hours.
//
// When a row already exists the earlier of the two deadlines wins. Two sources
// can observe the same kill with different delays, and the shorter one is the
// one we are entitled to publish at — taking the later would hold back a kill
// somebody is already allowed to see.
func Delay(ctx context.Context, pool *pgxpool.Pool, km Ref, hours int) error {
	if hours <= 0 {
		return nil
	}
	_, err := pool.Exec(ctx, `
        INSERT INTO esi_killmail_delayed (killmail_id, killmail_hash, delayed_until)
        VALUES ($1, $2, now() + make_interval(hours => $3))
        ON CONFLICT (killmail_id) DO UPDATE SET
            killmail_hash = EXCLUDED.killmail_hash,
            delayed_until = LEAST(esi_killmail_delayed.delayed_until, EXCLUDED.delayed_until)`,
		km.KillmailID, km.KillmailHash, hours)
	return err
}

// Undelay removes killmails from the delay table.
//
// Called whenever a kill arrives from a source with no delay — the zKillboard
// feed, a manual post, an unrestricted key. Seeing it publicly means the delay
// no longer applies, whatever some other observation said.
func Undelay(ctx context.Context, pool *pgxpool.Pool, killmailIDs []int64) error {
	if len(killmailIDs) == 0 {
		return nil
	}
	_, err := pool.Exec(ctx,
		`DELETE FROM esi_killmail_delayed WHERE killmail_id = ANY($1::bigint[])`, killmailIDs)
	return err
}

// IsDelayed reports whether a killmail is currently held back.
func IsDelayed(ctx context.Context, pool *pgxpool.Pool, killmailID int64) (bool, error) {
	var found bool
	err := pool.QueryRow(ctx,
		`SELECT true FROM esi_killmail_delayed WHERE killmail_id = $1`, killmailID).Scan(&found)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return false, nil
		}
		return false, err
	}
	return found, nil
}

// ClaimExpired atomically takes up to limit killmails whose delay has passed.
//
// One DELETE ... RETURNING rather than a SELECT followed by a DELETE, so two
// concurrent runs cannot both claim the same rows and dispatch the killmail
// twice. The ordering is oldest first, so a backlog drains in the order the
// delays actually expired.
func ClaimExpired(ctx context.Context, pool *pgxpool.Pool, limit int) ([]Ref, error) {
	rows, err := pool.Query(ctx, `
        DELETE FROM esi_killmail_delayed
        WHERE killmail_id IN (
            SELECT killmail_id FROM esi_killmail_delayed
            WHERE delayed_until <= now()
            ORDER BY delayed_until ASC
            LIMIT $1
        )
        RETURNING killmail_id, killmail_hash`, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []Ref
	for rows.Next() {
		var r Ref
		if err := rows.Scan(&r.KillmailID, &r.KillmailHash); err != nil {
			return nil, err
		}
		out = append(out, r)
	}
	return out, rows.Err()
}
