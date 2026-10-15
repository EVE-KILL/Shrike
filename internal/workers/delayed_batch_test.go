package workers

import (
	"context"
	"testing"
	"time"

	"github.com/eve-kill/shrike/internal/killmail"
)

func TestAuthenticatedKillmailBatchPreservesEarliestDelayAndLastHash(t *testing.T) {
	pool := testPool(t)
	ctx := t.Context()
	ids := []int64{2100002101, 2100002102, 2100002103}
	t.Cleanup(func() {
		_, _ = pool.Exec(context.Background(), "DELETE FROM esi_killmail_delayed WHERE killmail_id=ANY($1::bigint[])", ids)
	})
	if err := killmail.Delay(ctx, pool, killmail.Ref{KillmailID: ids[0], KillmailHash: "old"}, 1); err != nil {
		t.Fatal(err)
	}
	var earliest time.Time
	if err := pool.QueryRow(ctx, "SELECT delayed_until FROM esi_killmail_delayed WHERE killmail_id=$1", ids[0]).Scan(&earliest); err != nil {
		t.Fatal(err)
	}
	d := &Deps{Pool: pool}
	refs := []killmail.Ref{
		{KillmailID: ids[2], KillmailHash: "third"},
		{KillmailID: ids[0], KillmailHash: "updated"},
		{KillmailID: ids[1], KillmailHash: "first"},
		{KillmailID: ids[1], KillmailHash: "last"},
	}
	if err := d.enqueueKillmails(ctx, refs, 4); err != nil {
		t.Fatal(err)
	}
	for _, id := range ids {
		var hash string
		var until time.Time
		if err := pool.QueryRow(ctx, "SELECT killmail_hash,delayed_until FROM esi_killmail_delayed WHERE killmail_id=$1", id).Scan(&hash, &until); err != nil {
			t.Fatal(err)
		}
		want := map[int64]string{ids[0]: "updated", ids[1]: "last", ids[2]: "third"}[id]
		if hash != want {
			t.Fatalf("id %d hash=%q, want %q", id, hash, want)
		}
		if id == ids[0] {
			if !until.Equal(earliest) {
				t.Fatal("a later observation extended an existing delay")
			}
		} else if remaining := time.Until(until); remaining < 4*time.Hour-time.Minute || remaining > 4*time.Hour+time.Minute {
			t.Fatalf("new delay has wrong duration: %s", remaining)
		}
	}
	if err := d.enqueueKillmails(ctx, refs, 0); err != nil {
		t.Fatal(err)
	}
	var remaining int
	if err := pool.QueryRow(ctx, "SELECT count(*) FROM esi_killmail_delayed WHERE killmail_id=ANY($1::bigint[])", ids).Scan(&remaining); err != nil || remaining != 0 {
		t.Fatalf("public observations did not remove the delays: remaining=%d error=%v", remaining, err)
	}
}

func TestEmptyOrNonpositiveDelayedBatchDoesNotTouchDatabase(t *testing.T) {
	for _, tc := range []struct {
		refs  []killmail.Ref
		hours int
	}{{nil, 3}, {[]killmail.Ref{{KillmailID: 1}}, 0}, {[]killmail.Ref{{KillmailID: 1}}, -1}} {
		if err := killmail.DelayMany(t.Context(), nil, tc.refs, tc.hours); err != nil {
			t.Fatal(err)
		}
	}
}
