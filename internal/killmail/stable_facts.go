package killmail

// Stable killmail facts are derived solely from the complete victim and
// attacker rows. They may therefore be calculated once at ingest and safely
// reconstructed by the historical backfill.

const (
	FactionCaldari  int32 = 500001
	FactionMinmatar int32 = 500002
	FactionAmarr    int32 = 500003
	FactionGallente int32 = 500004
)

var capitalInvolvedGroups = map[int32]struct{}{
	30: {}, 485: {}, 513: {}, 547: {}, 659: {}, 883: {}, 902: {}, 1538: {}, 4594: {}, 5120: {},
}

func CapitalInvolvedGroupIDs() []int32 {
	ids := make([]int32, 0, len(capitalInvolvedGroups))
	for id := range capitalInvolvedGroups {
		ids = append(ids, id)
	}
	return ids
}

func AllianceTournamentShipIDs() []int32 {
	ids := make([]int32, 0, len(AllianceTournamentShipTypeIDs))
	for id := range AllianceTournamentShipTypeIDs {
		ids = append(ids, id)
	}
	return ids
}

// AllianceTournamentShipTypeIDs mirrors the curated zKillboard list inspected
// at a2734504e8c1a0b14bbaa58dd2fe3b3c51b7546e. Keep additions explicit and
// fixture-tested: tournament hull membership is not represented by one SDE
// category or meta group.
var AllianceTournamentShipTypeIDs = map[int32]struct{}{
	2836: {}, 74316: {}, 42246: {}, 32788: {}, 33675: {}, 33397: {}, 32790: {},
	35781: {}, 32207: {}, 74141: {}, 35779: {}, 60764: {}, 3516: {}, 32209: {},
	33395: {}, 42245: {}, 60765: {}, 26842: {}, 2834: {}, 3518: {}, 33673: {},
}

func DeriveStableFacts(p *Parsed) {
	if p == nil {
		return
	}
	km := &p.Killmail
	km.IsAwox = false
	km.IsCapitalInvolved = false
	km.IsSuperInvolved = false
	km.IsTitanInvolved = false
	km.IsATShipInvolved = false
	km.FWWinnerFactionID = 0
	_, km.IsCapitalInvolved = capitalInvolvedGroups[km.VictimShipGroupID]
	km.IsSuperInvolved = km.VictimShipGroupID == 659
	km.IsTitanInvolved = km.VictimShipGroupID == 30
	_, km.IsATShipInvolved = AllianceTournamentShipTypeIDs[km.VictimShipTypeID]

	factions := map[int32]bool{km.VictimFactionID: km.VictimFactionID != 0}
	for _, attacker := range p.Attackers {
		if _, ok := capitalInvolvedGroups[attacker.ShipGroupID]; ok {
			km.IsCapitalInvolved = true
		}
		km.IsSuperInvolved = km.IsSuperInvolved || attacker.ShipGroupID == 659
		km.IsTitanInvolved = km.IsTitanInvolved || attacker.ShipGroupID == 30
		if _, ok := AllianceTournamentShipTypeIDs[attacker.ShipTypeID]; ok {
			km.IsATShipInvolved = true
		}
		if attacker.FactionID != 0 {
			factions[attacker.FactionID] = true
		}
		if !km.IsNPC && km.VictimShipGroupID != 29 && km.VictimShipGroupID != 237 &&
			attacker.FinalBlow && attacker.CorporationID > 1_999_999 &&
			attacker.CorporationID == km.VictimCorporationID {
			km.IsAwox = true
		}
	}

	switch km.VictimFactionID {
	case FactionCaldari:
		if factions[FactionGallente] {
			km.FWWinnerFactionID = FactionGallente
		}
	case FactionGallente:
		if factions[FactionCaldari] {
			km.FWWinnerFactionID = FactionCaldari
		}
	case FactionAmarr:
		if factions[FactionMinmatar] {
			km.FWWinnerFactionID = FactionMinmatar
		}
	case FactionMinmatar:
		if factions[FactionAmarr] {
			km.FWWinnerFactionID = FactionAmarr
		}
	}
}
