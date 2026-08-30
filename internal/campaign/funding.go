package campaign

import (
	"context"
	"errors"
	"fmt"
	"regexp"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// Turning wallet donations into prize pool funding.
//
// Someone sends ISK to EVE-KILL with "campaign:<id>" in the reason, and that
// becomes money in a campaign's prize pool. There is no confirmation step and
// no way to take it back, so the classification has to be right the first time
// and has to happen exactly once per journal entry.
//
// Every entry is recorded, including the ones that fund nothing. A donation
// with a typo in the reason is money someone actually sent, and the reference
// row with its status and note is the only way to answer them when they ask
// where it went.

// EveKillCorporationID is the only corporation whose donations fund campaigns.
const EveKillCorporationID = 98779905

// Reference statuses.
const (
	ReferenceMatched   = 0
	ReferenceUnmatched = 1
	ReferenceLate      = 2
	ReferenceInvalid   = 3
)

// fundingReason matches a well-formed campaign funding reason.
//
// Exactly fourteen alphanumerics, because that is what a campaign id is.
// Anything looser would let "campaign:my-fleet-fund" resolve to a prefix of a
// real id and pay a stranger's pool.
var fundingReason = regexp.MustCompile(`(?i)^campaign:([0-9A-Za-z]{14})$`)

// ParseFundingReason extracts the campaign id from a donation reason.
func ParseFundingReason(reason string) (string, bool) {
	m := fundingReason.FindStringSubmatch(strings.TrimSpace(reason))
	if m == nil {
		return "", false
	}
	return m[1], true
}

// JournalEntry is one wallet journal row being considered as funding.
type JournalEntry struct {
	CorporationID int32
	Division      int16
	JournalID     int64
	Date          time.Time
	RefType       string
	Amount        float64
	Reason        string
}

// maxReferenceID bounds what is stored for an unparseable reason, so a garbage
// reason cannot overflow the column.
const maxReferenceID = 114

// ProcessJournalEntries classifies donations and credits the ones that qualify.
//
// Each entry is its own transaction. One malformed reason must not roll back
// the twenty good donations processed alongside it, and the journal entry's
// identity is the idempotency key — the reference row's primary key is
// (corporation, division, journal id), so a second pass over the same entry
// inserts nothing and credits nothing.
func ProcessJournalEntries(ctx context.Context, pool *pgxpool.Pool, entries []JournalEntry) (int, error) {
	var funded int
	for _, e := range entries {
		if !strings.HasPrefix(strings.ToLower(strings.TrimSpace(e.Reason)), "campaign:") {
			continue
		}
		ok, err := processEntry(ctx, pool, e)
		if err != nil {
			return funded, err
		}
		if ok {
			funded++
		}
	}
	return funded, nil
}

func processEntry(ctx context.Context, pool *pgxpool.Pool, e JournalEntry) (bool, error) {
	tx, err := pool.Begin(ctx)
	if err != nil {
		return false, err
	}
	defer tx.Rollback(ctx) //nolint:errcheck // no-op after commit

	reason := strings.TrimSpace(e.Reason)
	parsedID, parsed := ParseFundingReason(reason)

	referenceID := parsedID
	if !parsed {
		referenceID = reason[len("campaign:"):]
		if len(referenceID) > maxReferenceID {
			referenceID = referenceID[:maxReferenceID]
		}
	}

	status := ReferenceInvalid
	var note *string
	var matchedID string

	switch {
	case !parsed:
		note = new("Campaign references must be exactly campaign:<14-character-id>")

	case e.CorporationID != EveKillCorporationID || e.RefType != "player_donation" || e.Amount <= 0:
		// A corporation transfer, a market escrow release or a negative
		// adjustment is not a donation, whatever its reason says.
		note = new("Only positive player donations to EVE-KILL.com can fund campaigns")

	default:
		matched, poolStatus, endTime, err := lockPool(ctx, tx, parsedID)
		switch {
		case errors.Is(err, errAmbiguousPool):
			note = new("Campaign reference is ambiguous")
		case errors.Is(err, pgx.ErrNoRows):
			status = ReferenceUnmatched
			note = new("Campaign does not exist or does not have prizes enabled")
		case err != nil:
			return false, err
		default:
			matchedID = matched
			referenceID = matched
			// Funding closes when the campaign ends. Money arriving after that
			// is recorded and refused rather than silently changing a result
			// the participants have already seen.
			if poolStatus != PrizePoolFunding || endTime == nil || e.Date.After(*endTime) {
				status = ReferenceLate
				note = new("Campaign funding had already closed")
			} else {
				status = ReferenceMatched
			}
		}
	}

	// ON CONFLICT DO NOTHING plus RETURNING is what makes the credit below
	// exactly-once: only the insert that actually happened returns a row, so a
	// replay adds nothing to the pool.
	var inserted int64
	err = tx.QueryRow(ctx, `
        INSERT INTO wallet_journal_references (
            corporation_id, division, journal_id,
            reference_type, reference_id, status, amount, note)
        VALUES ($1, $2, $3, 'campaign', $4, $5, $6, $7)
        ON CONFLICT DO NOTHING
        RETURNING journal_id`,
		e.CorporationID, e.Division, e.JournalID,
		referenceID, status, e.Amount, note).Scan(&inserted)
	if errors.Is(err, pgx.ErrNoRows) {
		// Already classified on an earlier pass.
		return false, nil
	}
	if err != nil {
		return false, fmt.Errorf("record wallet reference %d: %w", e.JournalID, err)
	}

	if status != ReferenceMatched || matchedID == "" {
		return false, tx.Commit(ctx)
	}

	// The status guard in the WHERE is not redundant with the lock above: it
	// is the last check before real ISK is added, and it costs nothing.
	if _, err := tx.Exec(ctx, `
        UPDATE campaign_prize_pools
        SET funded_total = funded_total + $2::numeric,
            rules_locked_at = COALESCE(rules_locked_at, now()),
            updated_at = now()
        WHERE campaign_id = $1 AND status = $3`,
		matchedID, e.Amount, PrizePoolFunding); err != nil {
		return false, fmt.Errorf("credit prize pool %s: %w", matchedID, err)
	}

	return true, tx.Commit(ctx)
}

var errAmbiguousPool = errors.New("campaign reference matches more than one pool")

// lockPool finds and locks the prize pool a reference names.
//
// The lookup is case-insensitive because a donation reason is typed by hand,
// but that makes two ids differing only in case a possible collision. Reading
// two rows and refusing is the honest answer — crediting either one would be a
// guess about whose money it is.
func lockPool(ctx context.Context, tx pgx.Tx, campaignID string) (string, int16, *time.Time, error) {
	rows, err := tx.Query(ctx, `
        SELECT pool.campaign_id, pool.status, campaign.end_time
        FROM campaign_prize_pools pool
        JOIN campaigns campaign ON campaign.campaign_id = pool.campaign_id
        WHERE lower(pool.campaign_id) = lower($1)
        ORDER BY pool.campaign_id
        LIMIT 2
        FOR UPDATE OF pool`, campaignID)
	if err != nil {
		return "", 0, nil, err
	}
	defer rows.Close()

	type match struct {
		id      string
		status  int16
		endTime *time.Time
	}
	var found []match
	for rows.Next() {
		var m match
		if err := rows.Scan(&m.id, &m.status, &m.endTime); err != nil {
			return "", 0, nil, err
		}
		found = append(found, m)
	}
	if err := rows.Err(); err != nil {
		return "", 0, nil, err
	}

	switch len(found) {
	case 0:
		return "", 0, nil, pgx.ErrNoRows
	case 1:
		return found[0].id, found[0].status, found[0].endTime, nil
	default:
		return "", 0, nil, errAmbiguousPool
	}
}

// PendingJournalEntries finds campaign donations not yet classified.
//
// Bounded because it also serves as the catch-up for entries imported before
// this existed: without a limit the first run over that backlog would be one
// enormous query.
func PendingJournalEntries(ctx context.Context, pool *pgxpool.Pool, corporationID int32, limit int) ([]JournalEntry, error) {
	rows, err := pool.Query(ctx, `
        SELECT journal.corporation_id, journal.division, journal.journal_id,
               journal.date, journal.ref_type, coalesce(journal.amount, 0), coalesce(journal.reason, '')
        FROM corporation_wallet_journal journal
        LEFT JOIN wallet_journal_references reference
          ON reference.corporation_id = journal.corporation_id
         AND reference.division = journal.division
         AND reference.journal_id = journal.journal_id
        WHERE journal.corporation_id = $1
          AND reference.journal_id IS NULL
          AND lower(trim(coalesce(journal.reason, ''))) LIKE 'campaign:%'
        ORDER BY journal.date, journal.journal_id
        LIMIT $2`, corporationID, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []JournalEntry
	for rows.Next() {
		var e JournalEntry
		if err := rows.Scan(&e.CorporationID, &e.Division, &e.JournalID,
			&e.Date, &e.RefType, &e.Amount, &e.Reason); err != nil {
			return nil, err
		}
		out = append(out, e)
	}
	return out, rows.Err()
}

// PendingJournalLimit bounds one sweep.
const PendingJournalLimit = 5000
