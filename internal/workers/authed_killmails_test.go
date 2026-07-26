package workers

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/eve-kill/shrike/internal/esi"
	"github.com/eve-kill/shrike/internal/killmail"
)

func TestAuthenticatedKillmailFetchWritesOperationalLog(t *testing.T) {
	pool := testPool(t)
	ctx := context.Background()

	const (
		characterID int32 = 2_140_001_000
		endpoint          = "/characters/2140001000/killmails/recent/"
	)

	if _, err := pool.Exec(ctx, `
        INSERT INTO users (
            character_id, character_name, character_owner_hash, session_id,
            created_at, updated_at
        ) VALUES (
            $1, 'Shrike Auth Fetch Test', 'shrike-auth-fetch-test',
            gen_random_uuid(), now(), now()
        )
        ON CONFLICT (character_id) DO NOTHING`, characterID); err != nil {
		t.Fatal(err)
	}
	_, _ = pool.Exec(ctx,
		`DELETE FROM esi_request_logs WHERE character_id = $1 AND endpoint = $2`,
		characterID, endpoint)
	t.Cleanup(func() {
		_, _ = pool.Exec(context.Background(),
			`DELETE FROM esi_request_logs WHERE character_id = $1 AND endpoint = $2`,
			characterID, endpoint)
		_, _ = pool.Exec(context.Background(),
			`DELETE FROM users WHERE character_id = $1`, characterID)
	})

	d := &Deps{Pool: pool}
	d.logKillmailFetch(ctx, characterID, endpoint, "character_killmail_fetch",
		killmailWalk{
			Status: 200,
			Refs: []esi.WarKillmailRef{
				{KillmailID: 900_000_001},
				{KillmailID: 900_000_002},
				{KillmailID: 900_000_003},
			},
		},
		[]killmail.Ref{
			{KillmailID: 900_000_001},
			{KillmailID: 900_000_003},
		},
		1250*time.Millisecond,
	)

	var (
		status      *int16
		success     bool
		items, new  int
		newIDsBody  []byte
		source      string
		durationMS  *int32
		errorDetail *string
	)
	err := pool.QueryRow(ctx, `
        SELECT status_code, success, items_returned, new_items,
               new_item_ids, source, request_duration_ms, error_message
        FROM esi_request_logs
        WHERE character_id = $1 AND endpoint = $2
        ORDER BY id DESC LIMIT 1`, characterID, endpoint).
		Scan(&status, &success, &items, &new, &newIDsBody, &source, &durationMS, &errorDetail)
	if err != nil {
		t.Fatal(err)
	}

	if status == nil || *status != 200 || !success {
		t.Errorf("status/success = %v/%v, want 200/true", status, success)
	}
	if items != 3 || new != 2 {
		t.Errorf("items/new = %d/%d, want 3/2", items, new)
	}
	if source != "character_killmail_fetch" {
		t.Errorf("source = %q", source)
	}
	if durationMS == nil || *durationMS != 1250 {
		t.Errorf("duration = %v ms, want 1250", durationMS)
	}
	if errorDetail != nil {
		t.Errorf("successful fetch recorded error %q", *errorDetail)
	}

	var newIDs []int64
	if err := json.Unmarshal(newIDsBody, &newIDs); err != nil {
		t.Fatal(err)
	}
	if len(newIDs) != 2 || newIDs[0] != 900_000_001 || newIDs[1] != 900_000_003 {
		t.Errorf("new_item_ids = %v", newIDs)
	}
}
