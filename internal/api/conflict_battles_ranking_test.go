package api

import (
	"context"
	"reflect"
	"testing"

	"github.com/jackc/pgx/v5"
)

type battleRankingDB struct{ stubDatabase }

func (battleRankingDB) Query(context.Context, string, ...any) (pgx.Rows, error) {
	return &searchResponseRows{
		columns: []string{"battle_id", "team_index", "corporation_id", "corporation_name", "alliance_id", "alliance_name", "kills", "isk_destroyed"},
		rows: [][]any{
			{int64(42), int64(0), int64(101), "c", int64(1), "ab", int64(1), float64(100)},
			{int64(42), int64(0), int64(102), "bc", int64(2), "a", int64(1), float64(100)},
			{int64(42), int64(0), int64(103), "z", int64(3), "a", int64(1), float64(100)},
			{int64(42), int64(0), int64(104), "corp", int64(4), "c", int64(1), float64(100)},
			{int64(42), int64(0), int64(105), "corp", int64(5), "d", int64(1), float64(100)},
			{int64(42), int64(0), int64(106), "corp", int64(6), "e", int64(1), float64(100)},
		},
	}, nil
}

func TestConflictBattleTopFiveUsesSeparateNameTieBreakers(t *testing.T) {
	// These names collide when concatenated; a corporation name must also
	// never move an alliance ahead of an alphabetically earlier alliance.
	battles := []map[string]any{{"battle_id": int64(42)}}
	if err := attachConflictBattleListTeams(context.Background(), battleRankingDB{}, battles); err != nil {
		t.Fatal(err)
	}
	teams := battles[0]["teams"].([]map[string]any)
	if len(teams) != 1 || teams[0]["alliance_count"] != 6 {
		t.Fatalf("unexpected teams: %+v", teams)
	}
	top := teams[0]["top_alliances"].([]map[string]any)
	ids := make([]int64, 0, len(top))
	for _, alliance := range top {
		ids = append(ids, alliance["alliance_id"].(int64))
	}
	if want := []int64{2, 3, 1, 4, 5}; !reflect.DeepEqual(ids, want) {
		t.Fatalf("top alliances = %v, want %v", ids, want)
	}
}
