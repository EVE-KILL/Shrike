package frontend

import (
	"context"
	"strings"
	"testing"

	"github.com/eve-kill/shrike/internal/api"
)

func TestCorporationOwnedNPCFinalBlowRetainsTypePortraitAndCorporationName(t *testing.T) {
	row := api.KilllistRow{
		FinalBlowCorporationID: 1000043, FinalBlowCorporationName: "Amarr Navy",
		FinalBlowShipTypeID: 9965, FinalBlowShipName: "Imperial Navy Colonel",
	}
	var html strings.Builder
	if err := killIdentity(finalBlowIdentity(row)).Render(context.Background(), &html); err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{
		`class="kill-portrait" href="/item/9965"`,
		`src="/images/types/9965/icon?size=64"`,
		`href="/corporation/1000043" class="identity-primary">Amarr Navy`,
	} {
		if !strings.Contains(html.String(), want) {
			t.Fatalf("missing source final-blow identity behavior %q: %s", want, html.String())
		}
	}
	row.FinalBlowCharacterID, row.FinalBlowCharacterName = 42, "Player"
	player := finalBlowIdentity(row)
	if player.Portrait.Type != "character" || player.Portrait.ID != 42 || player.PrimaryID != 42 {
		t.Fatalf("player identity no longer takes priority: %+v", player)
	}
	row.FinalBlowCharacterID, row.FinalBlowShipTypeID = 0, 0
	unknown := finalBlowIdentity(row)
	if unknown.Portrait.ID != 0 || unknown.Portrait.Placeholder != "crosshair" || unknown.PrimaryName != "Amarr Navy" {
		t.Fatalf("missing NPC type must retain the name and empty portrait: %+v", unknown)
	}
	victim := victimIdentity(api.KilllistRow{VictimCorporationID: 1000043, VictimCorporationName: "Amarr Navy"})
	if victim.Portrait.Type != "corporation" || victim.Portrait.ID != 1000043 {
		t.Fatalf("victim corporation portrait changed: %+v", victim)
	}
}
