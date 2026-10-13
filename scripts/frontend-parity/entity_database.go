package main

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/eve-kill/shrike/internal/frontend"

	"github.com/jackc/pgx/v5/pgxpool"
)

// The disposable seed has combat records but no NPC faction catalogue. Supply
// explicitly named fixture identities only; all combat/warzone reads stay real
// HomepageReader queries in the read-only database.
type fixtureEntityReader struct{ frontend.EntityReader }

func (r fixtureEntityReader) EntityDetail(ctx context.Context, kind string, id int64) (map[string]any, error) {
	data, err := r.EntityReader.EntityDetail(ctx, kind, id)
	if err == nil && data == nil && kind == "faction" && (id == 500001 || id == 500003) {
		return map[string]any{"faction": map[string]any{"name": fmt.Sprintf("Fixture Faction %d", id)}}, nil
	}
	return data, err
}

func entityDatabaseConfig(dsn string) (*pgxpool.Config, error) {
	config, err := pgxpool.ParseConfig(dsn)
	if dsn == "" || err != nil || (config.ConnConfig.Host != "127.0.0.1" && config.ConnConfig.Host != "localhost") || !strings.HasPrefix(config.ConnConfig.Database, "shrike_frontend_parity_") {
		return nil, errors.New("entity-database requires TEST_DATABASE_URL for the existing disposable loopback fixture")
	}
	config.ConnConfig.RuntimeParams["default_transaction_read_only"] = "on"
	return config, nil
}

func openEntityDatabase(ctx context.Context, dsn string) (*pgxpool.Pool, error) {
	config, err := entityDatabaseConfig(dsn)
	if err != nil {
		return nil, err
	}
	pool, err := pgxpool.NewWithConfig(ctx, config)
	if err != nil {
		return nil, err
	}
	var readonly string
	if err = pool.QueryRow(ctx, "SHOW transaction_read_only").Scan(&readonly); err != nil || readonly != "on" {
		pool.Close()
		return nil, errors.New("entity fixture could not establish read-only sessions")
	}
	return pool, nil
}
