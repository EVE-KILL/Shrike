package frontend

import (
	"reflect"
	"testing"
)

func TestPageSlotsMatchNuxtNumberedPagination(t *testing.T) {
	for _, tt := range []struct {
		current, total int
		want           []int
	}{
		{1, 8, []int{1, 2, 0, 8}},
		{2, 8, []int{1, 2, 3, 0, 8}},
		{5, 10, []int{1, 0, 4, 5, 6, 0, 10}},
		{8, 8, []int{1, 0, 7, 8}},
		{3, 6, []int{1, 2, 3, 4, 5, 6}},
	} {
		if got := pageSlots(tt.current, tt.total); !reflect.DeepEqual(got, tt.want) {
			t.Errorf("pageSlots(%d, %d) = %v, want %v", tt.current, tt.total, got, tt.want)
		}
	}
}

func TestTopBoxMapThumbnailsMatchNuxt(t *testing.T) {
	if got := topEntryImage("regions", 10000058); got != "/images/regions/10000058?size=32" {
		t.Errorf("region thumbnail = %q", got)
	}
	if got := topEntryImage("constellations", 20000001); got != "" {
		t.Errorf("constellation should use the map-pin placeholder, got %q", got)
	}
}
