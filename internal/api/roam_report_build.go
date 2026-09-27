package api

import (
	"fmt"
	"math"
	"sort"
	"time"
)

// buildRoamReport turns observed killmail participation into a combat trail.
// It does not infer jumps or membership in an EVE fleet: neither is present in
// public killmails. Separate runs in the same system become separate
// engagements when the gap between recorded kills exceeds 30 minutes.
func buildRoamReport(
	id string, createdAt, start, end time.Time,
	names []string, roster []roamPilot, unresolved []string,
	killmails []roamKillmail, attackers []roamAttacker, finalBlows []roamFinalBlow,
) roamReport {
	report := roamReport{
		ID: id, CreatedAt: createdAt, StartTime: start, EndTime: end,
		InputCount: len(names), Roster: append([]roamPilot{}, roster...),
		Unresolved: append([]string{}, unresolved...),
		Systems:    []roamSystem{}, Targets: []roamTarget{}, Engagements: []roamEngagement{},
	}
	pilotIndex := make(map[int32]int, len(roster))
	for index, pilot := range roster {
		pilotIndex[pilot.CharacterID] = index
	}
	attackersByKill := make(map[int32][]roamAttacker)
	for _, attacker := range attackers {
		if _, ok := pilotIndex[attacker.CharacterID]; ok {
			attackersByKill[attacker.KillmailID] = append(attackersByKill[attacker.KillmailID], attacker)
		}
	}
	blowByKill := make(map[int32]roamFinalBlow, len(finalBlows))
	for _, blow := range finalBlows {
		blowByKill[blow.KillmailID] = blow
	}
	ordered := append([]roamKillmail{}, killmails...)
	sort.Slice(ordered, func(i, j int) bool {
		if ordered[i].KillmailTime.Equal(ordered[j].KillmailTime) {
			return ordered[i].KillmailID < ordered[j].KillmailID
		}
		return ordered[i].KillmailTime.Before(ordered[j].KillmailTime)
	})

	systemIndex := make(map[int32]int)
	regionIDs := make(map[int32]struct{})
	targets := make(map[int32]*roamTarget)
	uniqueVictims := make(map[int32]struct{})
	pilotEngagements := make(map[int32]map[int]struct{})
	engagementPilots := make(map[int]map[int32]struct{})
	currentEngagement := -1
	var lastKillTime time.Time
	var lastSystemID int32

	for _, km := range ordered {
		fleetAttackers := attackersByKill[km.KillmailID]
		_, fleetVictim := pilotIndex[km.VictimCharacterID]
		if !fleetVictim && len(fleetAttackers) == 0 {
			continue
		}
		if km.SolarSystemName == "" {
			km.SolarSystemName = fmt.Sprintf("System %d", km.SolarSystemID)
		}
		if km.RegionName == "" {
			km.RegionName = "Unknown region"
		}
		if math.IsNaN(km.TotalValue) || math.IsInf(km.TotalValue, 0) || km.TotalValue < 0 {
			km.TotalValue = 0
		}
		km.FleetAttackerIDs = []int32{}
		seenAttackers := make(map[int32]struct{})
		for _, attacker := range fleetAttackers {
			if _, seen := seenAttackers[attacker.CharacterID]; seen {
				continue
			}
			seenAttackers[attacker.CharacterID] = struct{}{}
			km.FleetAttackerIDs = append(km.FleetAttackerIDs, attacker.CharacterID)
			if attacker.FinalBlow {
				km.FleetFinalBlow = true
			}
		}
		if blow, ok := blowByKill[km.KillmailID]; ok {
			km.FinalBlowCharacterName = blow.CharacterName
			km.FinalBlowCorpName = blow.CorporationName
		}
		if fleetVictim {
			km.Role = "loss"
			km.FriendlyFire = len(km.FleetAttackerIDs) > 0
			report.Summary.Losses++
			report.Summary.IskLost += km.TotalValue
			if km.FriendlyFire {
				report.Summary.FriendlyFireLosses++
			}
			report.Roster[pilotIndex[km.VictimCharacterID]].Losses++
		} else {
			km.Role = "kill"
			report.Summary.Kills++
			report.Summary.IskDestroyed += km.TotalValue
			if len(km.FleetAttackerIDs) > 1 {
				report.Summary.CoordinatedKills++
			}
			if km.VictimCharacterID != 0 {
				uniqueVictims[km.VictimCharacterID] = struct{}{}
			}
			if km.VictimCorporationID != 0 {
				target := targets[km.VictimCorporationID]
				if target == nil {
					target = &roamTarget{
						CorporationID:   km.VictimCorporationID,
						CorporationName: km.VictimCorporationName,
						AllianceID:      km.VictimAllianceID,
						AllianceName:    km.VictimAllianceName,
					}
					targets[km.VictimCorporationID] = target
				}
				target.Kills++
				target.IskDestroyed += km.TotalValue
			}
			for _, attacker := range fleetAttackers {
				pilot := &report.Roster[pilotIndex[attacker.CharacterID]]
				pilot.KillParticipations++
				pilot.DamageDone += attacker.DamageDone
				if attacker.FinalBlow {
					pilot.FinalBlows++
				}
			}
		}

		if currentEngagement < 0 || lastSystemID != km.SolarSystemID ||
			km.KillmailTime.Sub(lastKillTime) > roamEngagementGap {
			report.Engagements = append(report.Engagements, roamEngagement{
				Number:        len(report.Engagements) + 1,
				SolarSystemID: km.SolarSystemID, SolarSystemName: km.SolarSystemName,
				RegionID: km.RegionID, RegionName: km.RegionName,
				StartTime: km.KillmailTime, EndTime: km.KillmailTime,
				PilotIDs: []int32{}, Killmails: []roamKillmail{},
			})
			currentEngagement = len(report.Engagements) - 1
			if index, ok := systemIndex[km.SolarSystemID]; ok {
				report.Systems[index].Engagements++
			}
		}
		lastKillTime, lastSystemID = km.KillmailTime, km.SolarSystemID
		engagement := &report.Engagements[currentEngagement]
		engagement.EndTime = km.KillmailTime
		engagement.Killmails = append(engagement.Killmails, km)
		if km.Role == "kill" {
			engagement.Kills++
			engagement.IskDestroyed += km.TotalValue
		} else {
			engagement.Losses++
			engagement.IskLost += km.TotalValue
		}

		index, ok := systemIndex[km.SolarSystemID]
		if !ok {
			index = len(report.Systems)
			systemIndex[km.SolarSystemID] = index
			report.Systems = append(report.Systems, roamSystem{
				SolarSystemID: km.SolarSystemID, SolarSystemName: km.SolarSystemName,
				RegionID: km.RegionID, RegionName: km.RegionName,
				FirstSeen: km.KillmailTime, Engagements: 1,
			})
		}
		system := &report.Systems[index]
		system.LastSeen = km.KillmailTime
		if km.Role == "kill" {
			system.Kills++
			system.IskDestroyed += km.TotalValue
		} else {
			system.Losses++
			system.IskLost += km.TotalValue
		}
		if km.RegionID != 0 {
			regionIDs[km.RegionID] = struct{}{}
		}

		members := engagementPilots[currentEngagement]
		if members == nil {
			members = make(map[int32]struct{})
			engagementPilots[currentEngagement] = members
		}
		for _, pilotID := range km.FleetAttackerIDs {
			members[pilotID] = struct{}{}
		}
		if fleetVictim {
			members[km.VictimCharacterID] = struct{}{}
		}
		for pilotID := range members {
			if pilotEngagements[pilotID] == nil {
				pilotEngagements[pilotID] = make(map[int]struct{})
			}
			pilotEngagements[pilotID][currentEngagement] = struct{}{}
		}
	}

	for index := range report.Engagements {
		for _, pilot := range report.Roster {
			if _, ok := engagementPilots[index][pilot.CharacterID]; ok {
				report.Engagements[index].PilotIDs = append(report.Engagements[index].PilotIDs, pilot.CharacterID)
			}
		}
	}
	for index := range report.Roster {
		pilot := &report.Roster[index]
		pilot.Engagements = len(pilotEngagements[pilot.CharacterID])
		if pilot.Engagements > 0 {
			report.Summary.ActivePilots++
		}
	}
	report.Summary.Engagements = len(report.Engagements)
	report.Summary.Systems = len(report.Systems)
	report.Summary.Regions = len(regionIDs)
	report.Summary.UniqueCharacterTargets = len(uniqueVictims)
	if total := report.Summary.IskDestroyed + report.Summary.IskLost; total > 0 {
		report.Summary.Efficiency = math.Round(report.Summary.IskDestroyed/total*10000) / 100
	}
	for _, target := range targets {
		report.Targets = append(report.Targets, *target)
	}
	sort.Slice(report.Targets, func(i, j int) bool {
		if report.Targets[i].Kills != report.Targets[j].Kills {
			return report.Targets[i].Kills > report.Targets[j].Kills
		}
		if report.Targets[i].IskDestroyed != report.Targets[j].IskDestroyed {
			return report.Targets[i].IskDestroyed > report.Targets[j].IskDestroyed
		}
		return report.Targets[i].CorporationID < report.Targets[j].CorporationID
	})
	if len(report.Targets) > 10 {
		report.Targets = report.Targets[:10]
	}
	return report
}

// addRoamShips counts each pilot/ship/killmail once, including a ship lost on
// a killmail where the pilot also appeared as an attacker. A pilot may have
// flown several hulls during the report window.
func addRoamShips(report *roamReport, attackers []roamAttacker) {
	type observation struct{ pilotID, shipTypeID, killmailID int32 }
	pilotIndex := make(map[int32]int, len(report.Roster))
	ships := make(map[int32]map[int32]*roamShip, len(report.Roster))
	seen := make(map[observation]struct{})
	for index := range report.Roster {
		pilot := &report.Roster[index]
		pilot.Ships = []roamShip{}
		pilotIndex[pilot.CharacterID] = index
		ships[pilot.CharacterID] = make(map[int32]*roamShip)
	}
	add := func(pilotID, shipTypeID, groupID, killmailID int32, shipName, groupName string, damage int64, lost bool) {
		if shipTypeID == 0 {
			return
		}
		if _, ok := pilotIndex[pilotID]; !ok {
			return
		}
		ship := ships[pilotID][shipTypeID]
		if ship == nil {
			if shipName == "" {
				shipName = fmt.Sprintf("Ship %d", shipTypeID)
			}
			ship = &roamShip{ShipTypeID: shipTypeID, ShipName: shipName, ShipGroupID: groupID, ShipGroupName: groupName}
			ships[pilotID][shipTypeID] = ship
		}
		if ship.ShipGroupID == 0 && groupID != 0 {
			ship.ShipGroupID, ship.ShipGroupName = groupID, groupName
		}
		key := observation{pilotID, shipTypeID, killmailID}
		if _, ok := seen[key]; !ok {
			ship.Killmails++
			seen[key] = struct{}{}
		}
		if damage > 0 {
			ship.DamageDone += damage
		}
		if lost {
			ship.Losses++
		}
	}
	for _, attacker := range attackers {
		add(attacker.CharacterID, attacker.ShipTypeID, attacker.ShipGroupID, attacker.KillmailID,
			attacker.ShipName, attacker.ShipGroupName, attacker.DamageDone, false)
	}
	for _, engagement := range report.Engagements {
		for _, kill := range engagement.Killmails {
			if kill.Role == "loss" {
				add(kill.VictimCharacterID, kill.VictimShipTypeID, kill.VictimShipGroupID, kill.KillmailID,
					kill.VictimShipName, kill.VictimShipGroupName, 0, true)
			}
		}
	}
	for index := range report.Roster {
		pilot := &report.Roster[index]
		for _, ship := range ships[pilot.CharacterID] {
			pilot.Ships = append(pilot.Ships, *ship)
		}
		sort.Slice(pilot.Ships, func(i, j int) bool {
			if pilot.Ships[i].Killmails != pilot.Ships[j].Killmails {
				return pilot.Ships[i].Killmails > pilot.Ships[j].Killmails
			}
			return pilot.Ships[i].ShipTypeID < pilot.Ships[j].ShipTypeID
		})
	}
}
