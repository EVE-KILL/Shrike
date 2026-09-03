package everef

import (
	"testing"
	"time"
)

func TestMarketOrderRowMatchesPublishedV3Shape(t *testing.T) {
	header := []string{
		"duration", "is_buy_order", "issued", "location_id", "min_volume",
		"order_id", "price", "range", "system_id", "type_id",
		"volume_remain", "volume_total", "http_last_modified", "station_id",
		"region_id", "constellation_id",
	}
	record := []string{
		"90", "false", "2026-06-28T15:13:51Z", "60014437", "1",
		"7367318263", "600.0", "region", "30000001", "20",
		"16500", "24500", "2026-09-03T13:41:11Z", "60014437",
		"10000001", "20000001",
	}
	snapshot := time.Date(2026, 9, 3, 13, 49, 53, 0, time.UTC)
	row, ok := marketOrderRow(record, headerIndex(header), snapshot)
	if !ok {
		t.Fatal("valid published row was rejected")
	}
	if len(row) != len(marketOrderColumns) {
		t.Fatalf("row width = %d, want %d", len(row), len(marketOrderColumns))
	}
	if row[1] != false || row[5] != int64(7367318263) || row[6] != 600.0 {
		t.Errorf("parsed row = %#v", row)
	}
	if row[16] != snapshot {
		t.Errorf("snapshot = %v, want %v", row[16], snapshot)
	}
}

func TestMarketOrderRowRejectsInvalidIdentityAndPrice(t *testing.T) {
	header := []string{
		"duration", "is_buy_order", "issued", "location_id", "min_volume",
		"order_id", "price", "range", "system_id", "type_id",
		"volume_remain", "volume_total", "region_id",
	}
	base := []string{
		"90", "true", "2026-09-03T12:00:00Z", "60000001", "1",
		"123", "10.5", "station", "30000001", "34", "5", "10", "10000001",
	}
	for _, index := range []int{5, 6, 8, 9, 12} {
		record := append([]string{}, base...)
		record[index] = "invalid"
		if _, ok := marketOrderRow(record, headerIndex(header), time.Now()); ok {
			t.Errorf("invalid field %s was accepted", header[index])
		}
	}
}

func TestSourceIdentityComparisonIncludesFallbackMetadata(t *testing.T) {
	stamp := time.Date(2026, 9, 3, 13, 49, 53, 0, time.UTC)
	if !sameTime(&stamp, &stamp) || sameTime(&stamp, nil) || sameTime(nil, &stamp) {
		t.Error("sameTime does not preserve nil/equality semantics")
	}
}
