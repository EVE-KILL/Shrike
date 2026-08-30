package api

import (
	"context"
	"testing"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
)

type stubMutationDatabase struct{ stubDatabase }

func (stubMutationDatabase) Exec(context.Context, string, ...any) (pgconn.CommandTag, error) {
	return pgconn.CommandTag{}, nil
}

func (stubMutationDatabase) Begin(context.Context) (pgx.Tx, error) { return nil, nil }

func TestDatabaseRolesUseConfiguredPrimary(t *testing.T) {
	read := stubDatabase{}
	primary := stubMutationDatabase{}
	opts := Options{DB: read, Primary: primary}

	if got := primaryDatabase(opts); got != primary {
		t.Fatalf("primaryDatabase() = %#v, want configured primary", got)
	}
	got, err := mutationDatabase(opts)
	if err != nil {
		t.Fatalf("mutationDatabase: %v", err)
	}
	if got != primary {
		t.Fatalf("mutationDatabase() = %#v, want configured primary", got)
	}
}

func TestDatabaseRolesPreserveSinglePoolCompatibility(t *testing.T) {
	primary := stubMutationDatabase{}
	opts := Options{DB: primary}

	if got := primaryDatabase(opts); got != primary {
		t.Fatalf("primaryDatabase() = %#v, want DB fallback", got)
	}
	if _, err := mutationDatabase(opts); err != nil {
		t.Fatalf("mutationDatabase fallback: %v", err)
	}
}
