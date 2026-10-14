package api

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"reflect"
	"testing"

	"github.com/danielgtaylor/huma/v2"
	"github.com/danielgtaylor/huma/v2/adapters/humachi"
	"github.com/go-chi/chi/v5"
	"github.com/jackc/pgx/v5"
)

func TestScopedWarPagesMatchFullSetBeforePagination(t *testing.T) {
	ctx, pool := localAPITestPool(t, "shrike_scoped_wars_")
	_, err := pool.Exec(ctx, `
        INSERT INTO corporations(corporation_id,name,ticker) VALUES(101,'Known Corporation','KNOWN');
        INSERT INTO alliances(alliance_id,name,ticker) VALUES(901,'Known Alliance','KNOWN');
        INSERT INTO users(character_id,character_name,character_owner_hash,session_id)
        VALUES(1,'Fixture owner','fixture','11111111-1111-1111-1111-111111111111');
        INSERT INTO custom_domains(subdomain,user_id,entities)
        VALUES('war-fixture',1,'[{"type":"alliance","id":901}]'),('war-empty',1,'[]');
        INSERT INTO wars(war_id,started,finished,mutual,aggressor_corporation_id,defender_corporation_id,
                         aggressor_alliance_id,defender_alliance_id,aggressor_isk_destroyed,
                         defender_isk_destroyed,aggressor_ships_killed,defender_ships_killed)
        SELECT n,
               CASE WHEN n%11=0 THEN NULL WHEN n%5=0 THEN now()+interval '2 days'+(n%2)*interval '1 hour' ELSE now()-interval '1 day' END,
               CASE WHEN n%4=0 THEN now()-interval '1 hour' ELSE NULL END,n%3=0,
               CASE n%3 WHEN 0 THEN 101 WHEN 1 THEN 102 ELSE 201 END,
               CASE n%4 WHEN 0 THEN 101 WHEN 1 THEN 103 ELSE 202 END,
               CASE WHEN n%6=0 THEN 901 ELSE NULL END,CASE WHEN n%7=0 THEN 902 ELSE NULL END,
               (n%2)*100,(n%3)*100,n%4,n%5
        FROM generate_series(1,60)n;
        INSERT INTO war_allies(war_id,corporation_id) SELECT n,101 FROM generate_series(1,60)n WHERE n%2=0;
        INSERT INTO war_allies(war_id,alliance_id) SELECT n,901 FROM generate_series(1,60)n WHERE n%3=0;
        INSERT INTO war_allies(war_id,corporation_id) SELECT n,101 FROM generate_series(1,60)n WHERE n%4=0;`)
	if err != nil {
		t.Fatal(err)
	}
	router := chi.NewRouter()
	a := humachi.New(router, huma.DefaultConfig("scope-fixture", "test"))
	registerLegacy(a, huma.Operation{OperationID: "fixture-wars", Method: http.MethodGet, Path: "/wars"}, conflictWarsHandler(Options{DB: pool}))
	// Independent reference: the original full-set OR/EXISTS definition, with
	// pagination only after every scope/filter. Do not reuse the new builder.
	corp := `(w.aggressor_corporation_id=101 OR w.defender_corporation_id=101 OR EXISTS(SELECT 1 FROM war_allies a WHERE a.war_id=w.war_id AND a.corporation_id=101))`
	alliance := func(id int) string {
		return fmt.Sprintf(`(w.aggressor_alliance_id=%d OR w.defender_alliance_id=%d OR EXISTS(SELECT 1 FROM war_allies a WHERE a.war_id=w.war_id AND a.alliance_id=%d))`, id, id, id)
	}
	unknownNames := false
	for _, tenant := range []struct{ host, predicate string }{
		{"eve-kill.com", "TRUE"}, {"war-fixture.eve-kill.com", alliance(901)}, {"war-empty.eve-kill.com", "FALSE"},
	} {
		for _, scope := range []struct{ query, predicate string }{
			{"", "TRUE"}, {"corporationId=101", corp}, {"allianceId=902", alliance(902)},
			{"corporationId=101&allianceId=902", "(" + corp + " OR " + alliance(902) + ")"},
		} {
			for _, filter := range []struct{ query, predicate, order string }{
				{"", "(w.started IS NULL OR w.started<=NOW())", "w.war_id DESC"},
				{"ongoing=true&mutual=true", "(w.started IS NULL OR w.started<=NOW()) AND w.finished IS NULL AND w.mutual IS TRUE", "w.war_id DESC"},
				{"hasAllies=true&hasKills=true", "(w.started IS NULL OR w.started<=NOW()) AND EXISTS(SELECT 1 FROM war_allies a WHERE a.war_id=w.war_id) AND (COALESCE(w.aggressor_ships_killed,0)>0 OR COALESCE(w.defender_ships_killed,0)>0)", "w.war_id DESC"},
				{"finished=true", "w.finished IS NOT NULL", "w.finished DESC NULLS LAST,w.war_id DESC"},
				{"upcoming=true", "w.started>NOW()", "w.started ASC,w.war_id DESC"},
				{"sort=isk&hasActivity=true", "(w.started IS NULL OR w.started<=NOW()) AND (COALESCE(w.aggressor_isk_destroyed,0)>0 OR COALESCE(w.defender_isk_destroyed,0)>0)", "(COALESCE(w.aggressor_isk_destroyed,0)+COALESCE(w.defender_isk_destroyed,0)) DESC,w.war_id DESC"},
				{"sort=kills", "(w.started IS NULL OR w.started<=NOW())", "(COALESCE(w.aggressor_ships_killed,0)+COALESCE(w.defender_ships_killed,0)) DESC,w.war_id DESC"},
			} {
				for _, page := range []int{1, 2, 4, 40} {
					const limit = 5
					query, _ := url.ParseQuery(scope.query + "&" + filter.query)
					query.Set("page", fmt.Sprint(page))
					query.Set("limit", fmt.Sprint(limit))
					response := httptest.NewRecorder()
					router.ServeHTTP(response, httptest.NewRequest(http.MethodGet, "http://"+tenant.host+"/wars?"+query.Encode(), nil))
					if response.Code != http.StatusOK {
						t.Fatalf("%s %s returned %d: %s", tenant.host, query.Encode(), response.Code, response.Body.String())
					}
					var body struct{ Wars []map[string]any }
					if err := json.Unmarshal(response.Body.Bytes(), &body); err != nil {
						t.Fatal(err)
					}
					got := make([]int32, 0, len(body.Wars))
					for _, war := range body.Wars {
						got = append(got, int32(int64OrZero(war["war_id"])))
						if war["aggressor"].(map[string]any)["name"] == "Unknown" {
							unknownNames = true
						}
					}
					rows, err := pool.Query(ctx, "SELECT w.war_id FROM wars w WHERE "+tenant.predicate+" AND "+scope.predicate+" AND "+filter.predicate+" ORDER BY "+filter.order+" LIMIT $1 OFFSET $2", limit, (page-1)*limit)
					if err != nil {
						t.Fatal(err)
					}
					want, err := pgx.CollectRows(rows, pgx.RowTo[int32])
					if err != nil {
						t.Fatal(err)
					}
					if !reflect.DeepEqual(got, want) {
						t.Fatalf("%s %s: got=%v want=%v", tenant.host, query.Encode(), got, want)
					}
				}
			}
		}
	}
	if !unknownNames {
		t.Fatal("missing entity names were not retained in the fixtures")
	}
}
