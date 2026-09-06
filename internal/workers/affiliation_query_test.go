package workers

import (
	"context"
	"os"
	"reflect"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"
)

func TestAffiliationCandidateQueryPreservesPriorityAndBoundaries(t *testing.T) {
	if os.Getenv("TEST_DATABASE_URL") == "" {
		t.Skip("requires explicit test database")
	}
	ctx := context.Background()
	pool := testPool(t)
	tx, err := pool.Begin(ctx)
	if err != nil {
		t.Fatal(err)
	}
	defer tx.Rollback(ctx) //nolint:errcheck
	_, err = tx.Exec(ctx, `CREATE TEMP TABLE characters(character_id int,updated_at timestamptz,last_active timestamptz,deleted bool) ON COMMIT DROP;
        INSERT INTO characters VALUES
        (1,NULL,NULL,false), (2,NULL,'2026-09-01',false),
        (3,'2026-08-01',NULL,false),(4,'2026-08-01','2026-09-01',false),
        (5,'2026-09-01',NULL,false),(6,'2026-08-01',NULL,true),
        (7,'2026-08-01','2025-09-06',false),(8,'2026-08-01','2025-09-05',NULL),
        (0,NULL,NULL,false),(-1,NULL,NULL,false),(9,'2026-08-01',NULL,false);`)
	if err != nil {
		t.Fatal(err)
	}
	activeSince := time.Date(2025, 9, 6, 0, 0, 0, 0, time.UTC)
	staleBefore := time.Date(2026, 9, 1, 0, 0, 0, 0, time.UTC)
	for _, predicate := range []string{"last_active IS NULL OR last_active < $1", "last_active IS NOT NULL AND last_active >= $1"} {
		for _, limit := range []int{1, 2, 3, 100} {
			read := func(query string) []int32 {
				rows, err := tx.Query(ctx, query, activeSince, staleBefore, limit)
				if err != nil {
					t.Fatal(err)
				}
				ids, err := pgx.CollectRows(rows, pgx.RowTo[int32])
				if err != nil {
					t.Fatal(err)
				}
				return ids
			}
			old := read("SELECT character_id FROM characters WHERE deleted IS NOT TRUE AND character_id>0 AND (" + predicate + ") AND (updated_at IS NULL OR updated_at<$2) ORDER BY updated_at ASC NULLS FIRST,character_id LIMIT $3")
			got := read(affiliationCandidateQuery(predicate))
			if !reflect.DeepEqual(old, got) {
				t.Fatalf("%s limit %d: old=%v new=%v", predicate, limit, old, got)
			}
		}
	}
}
