package killmail

import "testing"

func TestDeriveStableFacts(t *testing.T) {
	p := &Parsed{
		Killmail: Killmail{
			VictimCorporationID: 2_000_123,
			VictimFactionID:     FactionCaldari,
			VictimShipGroupID:   25,
		},
		Attackers: []Attacker{
			{CorporationID: 2_000_123, FinalBlow: true, FactionID: FactionGallente, ShipGroupID: 30},
			{ShipTypeID: 2836, ShipGroupID: 659},
		},
	}

	DeriveStableFacts(p)
	km := p.Killmail
	if !km.IsAwox || !km.IsCapitalInvolved || !km.IsSuperInvolved || !km.IsTitanInvolved || !km.IsATShipInvolved {
		t.Fatalf("stable flags = %+v", km)
	}
	if km.FWWinnerFactionID != FactionGallente {
		t.Fatalf("FW winner = %d, want %d", km.FWWinnerFactionID, FactionGallente)
	}
}

func TestDeriveStableFactsAwoxExclusions(t *testing.T) {
	for _, tc := range []struct {
		name string
		km   Killmail
		att  Attacker
	}{
		{"npc", Killmail{IsNPC: true, VictimCorporationID: 2_000_001, VictimShipGroupID: 25}, Attacker{CorporationID: 2_000_001, FinalBlow: true}},
		{"capsule", Killmail{VictimCorporationID: 2_000_001, VictimShipGroupID: 29}, Attacker{CorporationID: 2_000_001, FinalBlow: true}},
		{"rookie ship", Killmail{VictimCorporationID: 2_000_001, VictimShipGroupID: 237}, Attacker{CorporationID: 2_000_001, FinalBlow: true}},
		{"npc corporation", Killmail{VictimCorporationID: 1_000_001, VictimShipGroupID: 25}, Attacker{CorporationID: 1_000_001, FinalBlow: true}},
		{"not final blow", Killmail{VictimCorporationID: 2_000_001, VictimShipGroupID: 25}, Attacker{CorporationID: 2_000_001}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			p := &Parsed{Killmail: tc.km, Attackers: []Attacker{tc.att}}
			DeriveStableFacts(p)
			if p.Killmail.IsAwox {
				t.Fatal("classified as awox")
			}
		})
	}
}

func TestDeriveStableFactsFWRequiresOpposingMilitias(t *testing.T) {
	p := &Parsed{Killmail: Killmail{VictimFactionID: FactionAmarr}, Attackers: []Attacker{{FactionID: FactionCaldari}}}
	DeriveStableFacts(p)
	if p.Killmail.FWWinnerFactionID != 0 {
		t.Fatalf("FW winner = %d", p.Killmail.FWWinnerFactionID)
	}
}

func TestDeriveStableFactsAmarrMinmatarWinner(t *testing.T) {
	p := &Parsed{
		Killmail:  Killmail{VictimFactionID: FactionMinmatar},
		Attackers: []Attacker{{FactionID: FactionAmarr}},
	}
	DeriveStableFacts(p)
	if p.Killmail.FWWinnerFactionID != FactionAmarr {
		t.Fatalf("FW winner = %d, want %d", p.Killmail.FWWinnerFactionID, FactionAmarr)
	}
}
