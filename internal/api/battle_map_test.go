package api

import (
	"context"
	"errors"
	"net/url"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"
)

type battleMapCaptureDB struct {
	stubDatabase
	query string
	args  []any
}

var errBattleMapCaptured = errors.New("query captured")

func (db *battleMapCaptureDB) Query(_ context.Context, query string, args ...any) (pgx.Rows, error) {
	db.query, db.args = query, args
	return nil, errBattleMapCaptured
}

func TestBattleMapSharesListFiltersBeforeAggregation(t *testing.T) {
	filters := url.Values{"year": {"2026"}, "hours": {"48"}, "minKills": {"25"}, "minIsk": {"10000000000"}, "regionId": {"10000003"}, "systemId": {"30000225"}, "allianceId": {"99003581"}, "corporationId": {"98540583"}, "custom": {"true"}, "page": {"3"}}
	list, mapped := &battleMapCaptureDB{}, &battleMapCaptureDB{}
	for _, db := range []*battleMapCaptureDB{list, mapped} {
		if db == mapped {
			filters.Set("map", "true")
		}
		_, err := conflictBattlesHandler(Options{DB: db})(context.Background(), &legacyRequest{Query: filters})
		if !errors.Is(err, errBattleMapCaptured) {
			t.Fatalf("handler: %v", err)
		}
	}
	predicate := func(query string, end string) string {
		start := strings.Index(query, "WHERE TRUE")
		if start < 0 {
			t.Fatalf("missing filter predicates: %s", query)
		}
		return strings.TrimSpace(strings.Split(query[start:], end)[0])
	}
	if predicate(list.query, "ORDER BY b.start_time") != predicate(mapped.query, "GROUP BY b.solar_system_id") {
		t.Fatal("list and map filters differ")
	}
	if strings.Contains(mapped.query, "LIMIT") || strings.Contains(mapped.query, "OFFSET") {
		t.Fatal("map aggregates only a page")
	}
	if len(list.args) != len(mapped.args)+2 {
		t.Fatal("map/list argument mismatch")
	}
	for i, value := range mapped.args {
		if cutoff, ok := value.(time.Time); ok {
			if cutoff.Sub(list.args[i].(time.Time)).Abs() > time.Second {
				t.Fatal("time filter differs")
			}
		} else if !reflect.DeepEqual(value, list.args[i]) {
			t.Fatalf("argument %d differs", i)
		}
	}
}

func TestBattleMapRejectsInvalidFiltersBeforeQuery(t *testing.T) {
	for _, query := range []string{"hours=0", "hours=-1", "hours=8761", "hours=two", "minIsk=NaN", "minIsk=Inf", "minIsk=-1"} {
		values, _ := url.ParseQuery(query + "&map=true")
		_, err := conflictBattlesHandler(Options{})(context.Background(), &legacyRequest{Query: values})
		var apiErr *legacyAPIError
		if !errors.As(err, &apiErr) || apiErr.Status != 400 {
			t.Fatalf("%s: %v", query, err)
		}
	}
}
