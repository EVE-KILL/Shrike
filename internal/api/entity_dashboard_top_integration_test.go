package api

import (
	"testing"
)

func TestDashboardTopListsPreserveRankingAggregationAndMissingNames(t *testing.T) {
	ctx, pool := localAPITestPool(t, "shrike_dashboard_top_")
	_, err := pool.Exec(ctx, `
		INSERT INTO inv_types(type_id,name) SELECT 1000+n,'Type '||n FROM generate_series(1,12)n WHERE n NOT IN (1,11);
		INSERT INTO corporations(corporation_id,name,ticker) SELECT 1000+n,'Corp '||n,'C'||n FROM generate_series(1,12)n WHERE n NOT IN (1,11);
		INSERT INTO alliances(alliance_id,name,ticker) SELECT 1000+n,'Alliance '||n,'A'||n FROM generate_series(1,12)n WHERE n NOT IN (1,11);
		INSERT INTO stats_breakdowns(entity_type,entity_id,dim_category,dim_id,period_type,period_start,kills,losses,isk_destroyed,isk_lost)
		SELECT entity_type,9999,category,1000+n,period_type,day,
		       CASE WHEN n<=3 THEN 100 ELSE 100-n END,
		       CASE WHEN n>=10 THEN 12 ELSE n END,100,200
		FROM generate_series(1,12)n
		CROSS JOIN (VALUES (1::smallint),(2::smallint)) entities(entity_type)
		CROSS JOIN (VALUES (0::smallint),(1::smallint),(21::smallint),(22::smallint)) categories(category)
		CROSS JOIN (VALUES (2::smallint,DATE '2020-01-01'),(2::smallint,DATE '2021-01-01'),
		                   (0::smallint,CURRENT_DATE),(0::smallint,DATE '2000-01-01')) periods(period_type,day);`)
	if err != nil {
		t.Fatal(err)
	}
	for _, kind := range []string{entityPageCorporation, entityPageAlliance} {
		entityType, _ := entityPageStatsType(kind)
		for _, window := range []string{entityPageDaysWindow(0), "30d"} {
			t.Run(kind+"/"+window, func(t *testing.T) {
				result, err := loadEntityDashboardStats(ctx, pool, kind, entityType, 9999, 0, window)
				if err != nil {
					t.Fatal(err)
				}
				factor := int64(1)
				if window == entityPageDaysWindow(0) {
					factor = 2
				}
				for _, list := range []string{"topShipsUsed", "topShipsLost", "diesToCorporations", "diesToAlliances"} {
					rows := result[list].([]map[string]any)
					if len(rows) != 10 {
						t.Fatalf("%s returned %d rows", list, len(rows))
					}
					for index, row := range rows {
						idKey, nameKey := "id", "name"
						if list == "topShipsUsed" || list == "topShipsLost" {
							idKey, nameKey = "ship_type_id", "ship_name"
						}
						n := int64(index + 1)
						count := int64(100) - n
						if list == "topShipsUsed" {
							if n <= 3 {
								count = 100
							}
						} else {
							if index < 3 {
								n, count = int64(10+index), 12
							} else {
								n, count = int64(12-index), int64(12-index)
							}
						}
						if int64OrZero(row[idKey]) != 1000+n || int64OrZero(row["count"]) != factor*count {
							t.Fatalf("%s[%d]=%v, want id=%d count=%d", list, index, row, 1000+n, factor*count)
						}
						if (n == 1 || n == 11) && row[nameKey] != "Unknown" {
							t.Fatalf("missing name changed ranking or fallback: %v", row)
						}
					}
				}
			})
		}
	}
}
