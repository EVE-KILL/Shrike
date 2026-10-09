package main

import (
	"context"
	"testing"

	"github.com/eve-kill/shrike/internal/frontend"
)

func TestEntityDatabaseRequiresDisposableLocalNameAndReadOnlySessions(t *testing.T) {
	for _, dsn := range []string{"", "postgres://test@100.111.3.8/shrike_frontend_parity_test", "postgres://test@127.0.0.1/shrike", "postgres://test@127.0.0.1/postgres"} {
		if _, err := entityDatabaseConfig(dsn); err == nil {
			t.Error("unsafe entity database config accepted")
		}
	}
	config, err := entityDatabaseConfig("postgres://test@127.0.0.1/shrike_frontend_parity_test?default_transaction_read_only=off")
	if err != nil || config.ConnConfig.RuntimeParams["default_transaction_read_only"] != "on" {
		t.Fatal("read-only session guard missing", err)
	}
}

type missingFixtureEntity struct{ frontend.EntityReader }

func (missingFixtureEntity) EntityDetail(context.Context, string, int64) (map[string]any, error) {
	return nil, nil
}
func TestFixtureFactionIdentityDoesNotReplaceOtherMissingEntities(t *testing.T) {
	reader := fixtureEntityReader{missingFixtureEntity{}}
	for _, item := range []struct {
		kind    string
		id      int64
		fixture bool
	}{{"faction", 500001, true}, {"faction", 500003, true}, {"faction", 999, false}, {"character", 70131, false}} {
		data, err := reader.EntityDetail(context.Background(), item.kind, item.id)
		if err != nil || (data != nil) != item.fixture {
			t.Fatal("unexpected fixture identity", item.kind, item.id, err)
		}
	}
}
