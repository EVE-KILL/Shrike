package frontend

import (
	"reflect"
	"testing"

	"github.com/eve-kill/shrike/internal/api"
)

func TestCampaignSideColorsUseNuxtAllOrNothingPalette(t *testing.T) {
	red := &api.TopPalette{MainColor: "#960000"}
	blue := &api.TopPalette{MainColor: "#000096"}
	for _, tt := range []struct {
		name     string
		palettes []*api.TopPalette
		want     []string
	}{
		{"missing palette", []*api.TopPalette{red, nil}, []string{"#ef4444", "#3b82f6"}},
		{"similar hues", []*api.TopPalette{red, {MainColor: "#aa1800"}}, []string{"#ef4444", "#3b82f6"}},
		{"distinct hues", []*api.TopPalette{red, blue}, []string{paletteAccent(red), paletteAccent(blue)}},
	} {
		t.Run(tt.name, func(t *testing.T) {
			var campaign api.EntityCampaignEntry
			for index, palette := range tt.palettes {
				campaign.Sides = append(campaign.Sides, api.EntityCampaignSide{SideIndex: int64(index), Palette: palette})
			}
			if got := campaignSideColors(campaign); !reflect.DeepEqual(got, tt.want) {
				t.Fatalf("campaignSideColors() = %v, want %v", got, tt.want)
			}
		})
	}
}
