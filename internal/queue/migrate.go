package queue

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/riverqueue/river/riverdriver/riverpgxv5"
	"github.com/riverqueue/river/rivermigrate"
)

// River owns its own schema.
//
// This is a second migration ledger alongside goose's, and that is the right
// arrangement rather than a compromise. River's tables are River's, versioned
// against the library rather than against our data model: copying its DDL into
// a goose migration would freeze it at whatever the library shipped that week,
// and every upgrade would become a hand-transcription of someone else's schema
// change. The two ledgers never touch the same tables, so there is nothing for
// them to disagree about.
//
// The safety posture is the same as goose's: this is a separate, explicit
// command, never something a worker runs on startup. A process that migrates
// its own database on boot will eventually migrate one you did not mean.

// MigrationStatus reports where the River schema stands.
type MigrationStatus struct {
	Applied   []int `json:"applied"`
	Available []int `json:"available"`
	Pending   []int `json:"pending"`
}

// UpToDate reports whether there is nothing left to apply.
func (s MigrationStatus) UpToDate() bool { return len(s.Pending) == 0 }

func migrator(pool *pgxpool.Pool) (*rivermigrate.Migrator[pgx.Tx], error) {
	return rivermigrate.New(riverpgxv5.New(pool), nil)
}

// MigrationState reads the applied and available versions.
func MigrationState(ctx context.Context, pool *pgxpool.Pool) (MigrationStatus, error) {
	var out MigrationStatus

	m, err := migrator(pool)
	if err != nil {
		return out, err
	}

	for _, v := range m.AllVersions() {
		out.Available = append(out.Available, v.Version)
	}

	// An unmigrated database has no river_migration table at all, which surfaces
	// as an error rather than an empty list. That is the expected state on a
	// fresh install, not a failure, so it reports as "everything is pending".
	existing, err := m.ExistingVersions(ctx)
	if err != nil {
		out.Pending = out.Available
		return out, nil
	}

	applied := map[int]bool{}
	for _, v := range existing {
		out.Applied = append(out.Applied, v.Version)
		applied[v.Version] = true
	}
	for _, v := range out.Available {
		if !applied[v] {
			out.Pending = append(out.Pending, v)
		}
	}
	return out, nil
}

// Migrate applies every pending River migration.
func Migrate(ctx context.Context, pool *pgxpool.Pool) ([]string, error) {
	m, err := migrator(pool)
	if err != nil {
		return nil, err
	}

	res, err := m.Migrate(ctx, rivermigrate.DirectionUp, nil)
	if err != nil {
		return nil, fmt.Errorf("apply River migrations: %w", err)
	}

	applied := make([]string, 0, len(res.Versions))
	for _, v := range res.Versions {
		applied = append(applied, fmt.Sprintf("%03d %s", v.Version, v.Name))
	}
	return applied, nil
}
