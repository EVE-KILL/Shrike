package migrate

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"testing"

	"github.com/eve-kill/shrike/migrations"
	"github.com/jackc/pgx/v5/pgconn"
)

func TestCustomDomainJSONRepair(t *testing.T) {
	ctx := context.Background()
	pool := scratchDB(t, "domain_json")
	_, err := pool.Exec(ctx, `
		CREATE TABLE custom_domains (
			id int PRIMARY KEY, entities jsonb NOT NULL, theme jsonb NOT NULL,
			navbar_links jsonb NOT NULL, widgets jsonb NOT NULL
		);
		INSERT INTO custom_domains VALUES (
			1, '[{"type":"corporation","id":98630834}]',
			'{"color":"purple"}', '[{"label":"Home"}]', '{"left":[]}'
		);
		INSERT INTO custom_domains
		SELECT 2, to_jsonb(entities::text), to_jsonb(theme::text),
		       to_jsonb(navbar_links::text), to_jsonb(widgets::text)
		FROM custom_domains WHERE id = 1;
	`)
	if err != nil {
		t.Fatal(err)
	}
	sql, err := migrations.FS.ReadFile("00027_custom_domain_json_shapes.sql")
	if err != nil {
		t.Fatal(err)
	}
	parts := strings.Split(string(sql), "-- +goose Down")
	for attempt := range 2 {
		if _, err := pool.Exec(ctx, parts[0]); err != nil {
			t.Fatalf("repair attempt %d: %v", attempt, err)
		}
	}
	assertDocumentsEqual := func() {
		t.Helper()
		var equal bool
		err := pool.QueryRow(ctx, `
			SELECT (to_jsonb(a) - 'id') = (to_jsonb(b) - 'id')
			FROM custom_domains a, custom_domains b
			WHERE a.id = 1 AND b.id = 2
		`).Scan(&equal)
		if err != nil || !equal {
			t.Fatalf("documents differ after repair: equal=%v, err=%v", equal, err)
		}
	}
	assertDocumentsEqual()
	for _, column := range []string{"entities", "theme", "navbar_links", "widgets"} {
		t.Run(column, func(t *testing.T) {
			_, err := pool.Exec(ctx, fmt.Sprintf(
				"UPDATE custom_domains SET %s = to_jsonb(%s::text) WHERE id = 1",
				column, column,
			))
			var pgErr *pgconn.PgError
			if !errors.As(err, &pgErr) || pgErr.Code != "23514" {
				t.Fatalf("expected check violation, got %v", err)
			}
		})
	}
	if _, err := pool.Exec(ctx, parts[1]); err != nil {
		t.Fatalf("down: %v", err)
	}
	assertDocumentsEqual()
}
