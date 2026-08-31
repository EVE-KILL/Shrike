package killmail_test

import (
	"context"
	"fmt"
	"os"
	"testing"

	"github.com/eve-kill/shrike/internal/eve"
	"github.com/eve-kill/shrike/internal/killmail"
	"github.com/eve-kill/shrike/internal/killtype"
	"github.com/jackc/pgx/v5/pgxpool"
)

// The one test that matters for the daily rollup.
//
// Two independent implementations decide which subsets a killmail belongs to:
// Classify, which runs per killmail as it arrives, and the SQL predicates,
// which run over a date range during reconciliation. They must agree exactly.
// When they do not, the incremental count and the nightly rebuild disagree and
// the navigation number silently changes at midnight.
//
// Nothing but real killmails will find that. A hand-built case exercises the
// branches its author thought of; the corpus contains the ones they did not —
// null regions, hulls missing from the SDE, structures, pods, abyssal deaths.
func TestClassifyAgreesWithPredicates(t *testing.T) {
	dsn := os.Getenv("TEST_DATABASE_URL")
	if dsn == "" {
		t.Skip("TEST_DATABASE_URL is not set")
	}

	ctx := context.Background()
	pool, err := pgxpool.New(ctx, dsn)
	if err != nil {
		t.Fatalf("connect: %v", err)
	}
	defer pool.Close()
	if err := pool.Ping(ctx); err != nil {
		t.Skipf("database unreachable: %v", err)
	}

	cache, err := eve.Load(ctx, pool)
	if err != nil {
		t.Fatalf("load SDE cache: %v", err)
	}

	// Ordered by id descending rather than sampled: the interesting
	// classifications — titans, structures, abyssal deaths, hulls the SDE does
	// not know — are rare, and a 1% sample of a small corpus misses all of
	// them. Whatever is here, this walks it.
	const sampleSize = 50_000
	rows, err := pool.Query(ctx, `
        SELECT killmail_id, killmail_time, solar_system_id, coalesce(region_id, 0),
		       coalesce(total_value, 0), coalesce(is_solo, false), coalesce(is_npc, false),
		       coalesce(attacker_count, 0), coalesce(victim_ship_type_id, 0), coalesce(victim_ship_group_id, 0)
        FROM killmails
        ORDER BY killmail_id DESC
        LIMIT $1`, sampleSize)
	if err != nil {
		t.Fatalf("sample killmails: %v", err)
	}

	type sample struct {
		id  int64
		km  killmail.Killmail
		got map[string]bool
	}
	var samples []sample

	for rows.Next() {
		var s sample
		if err := rows.Scan(&s.id, &s.km.KillmailTime, &s.km.SolarSystemID, &s.km.RegionID,
			&s.km.TotalValue, &s.km.IsSolo, &s.km.IsNPC,
			&s.km.AttackerCount,
			&s.km.VictimShipTypeID, &s.km.VictimShipGroupID); err != nil {
			rows.Close()
			t.Fatalf("scan: %v", err)
		}
		s.km.KillmailID = s.id
		s.got = map[string]bool{}
		for _, name := range killtype.Classify(killmail.Subject(cache, s.km)) {
			s.got[name] = true
		}
		samples = append(samples, s)
	}
	rows.Close()
	if err := rows.Err(); err != nil {
		t.Fatalf("read sample: %v", err)
	}
	if len(samples) == 0 {
		t.Skip("no killmails in the database to compare against")
	}

	byID := make(map[int64]*sample, len(samples))
	ids := make([]int64, 0, len(samples))
	for i := range samples {
		byID[samples[i].id] = &samples[i]
		ids = append(ids, samples[i].id)
	}

	// Ask the database, using the same predicates the reconcile cron uses,
	// which of the sampled killmails belong to each subset.
	predicates := killtype.Predicates()
	var disagreements int

	for _, kind := range killtype.Types {
		predicate, ok := predicates[kind]
		if !ok {
			t.Errorf("type %q has no predicate — the reconcile would never count it", kind)
			continue
		}

		// The predicate is a constant fragment from the registry, never input.
		rows, err := pool.Query(ctx, fmt.Sprintf(`
            SELECT k.killmail_id FROM killmails k
            WHERE k.killmail_id = ANY($1::bigint[]) AND %s`, predicate), ids)
		if err != nil {
			t.Fatalf("select %s: %v", kind, err)
		}

		wanted := map[int64]bool{}
		for rows.Next() {
			var id int64
			if err := rows.Scan(&id); err != nil {
				rows.Close()
				t.Fatalf("scan %s: %v", kind, err)
			}
			wanted[id] = true
		}
		rows.Close()
		if err := rows.Err(); err != nil {
			t.Fatalf("read %s: %v", kind, err)
		}

		for _, s := range samples {
			inSQL, inGo := wanted[s.id], s.got[kind]
			if inSQL == inGo {
				continue
			}
			disagreements++
			if disagreements <= 20 {
				t.Errorf("killmail %d and subset %q: SQL says %v, Classify says %v "+
					"(system %d, region %d, value %.0f, ship %d, group %d)",
					s.id, kind, inSQL, inGo,
					s.km.SolarSystemID, s.km.RegionID, s.km.TotalValue,
					s.km.VictimShipTypeID, s.km.VictimShipGroupID)
			}
		}
	}

	if disagreements > 0 {
		t.Fatalf("%d disagreements across %d killmails — the incremental rollup "+
			"and the nightly reconcile would produce different numbers",
			disagreements, len(samples))
	}
	t.Logf("Classify and the SQL predicates agree on %d killmails across %d subsets",
		len(samples), len(killtype.Types))
}
