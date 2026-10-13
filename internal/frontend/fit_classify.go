package frontend

import (
	"fmt"
	"math"
	"sort"
	"strings"

	"github.com/eve-kill/shrike/internal/api"
)

func fitFamilyName(f api.UniverseFitFamily, hullGroup string) string {
	names := make([]string, 0, len(f.Modules))
	for _, module := range f.Modules {
		names = append(names, strings.ToLower(module.Name))
	}
	has := func(terms ...string) bool {
		for _, name := range names {
			for _, term := range terms {
				if strings.Contains(name, term) {
					return true
				}
			}
		}
		return false
	}
	combat := func(terms ...string) bool {
		for _, name := range names {
			if strings.Contains(name, "civilian") {
				continue
			}
			for _, term := range terms {
				if strings.Contains(name, term) {
					return true
				}
			}
		}
		return false
	}
	drone := func(terms ...string) bool {
		for _, item := range f.Drones {
			name := strings.ToLower(item.Name)
			for _, term := range terms {
				if strings.Contains(name, term) {
					return true
				}
			}
		}
		return false
	}
	weapon := ""
	for _, choice := range []struct {
		Label string
		Terms []string
	}{
		{"Heavy Assault Missile", []string{"heavy assault missile launcher"}}, {"Rapid Light Missile", []string{"rapid light missile launcher"}},
		{"Rapid Heavy Missile", []string{"rapid heavy missile launcher"}}, {"Heavy Missile", []string{"heavy missile launcher"}},
		{"Rocket", []string{"rocket launcher"}}, {"Light Missile", []string{"light missile launcher"}},
		{"Torpedo", []string{"torpedo launcher"}}, {"Cruise Missile", []string{"cruise missile launcher"}},
		{"Pulse Laser", []string{"pulse laser"}}, {"Beam Laser", []string{"beam laser", "tachyon beam"}},
		{"Autocannon", []string{"autocannon"}}, {"Artillery", []string{"artillery"}},
		{"Blaster", []string{"blaster"}}, {"Railgun", []string{"railgun"}},
		{"Disintegrator", []string{"entropic disintegrator"}}, {"Vorton Projector", []string{"vorton projector"}},
		{"Smartbomb", []string{"smartbomb"}},
	} {
		if combat(choice.Terms...) {
			weapon = choice.Label
			break
		}
	}
	hasDroneDamage := has("drone damage amplifier", "drone link augmentor", "omnidirectional tracking link")
	hasCombatDrones := false
	for _, item := range f.Drones {
		name := strings.ToLower(item.Name)
		if name == "" {
			continue
		}
		utility := false
		for _, term := range []string{"maintenance bot", "salvage drone", "mining drone", "ec-", "ev-", "sw-", "sd-", "tp-"} {
			if strings.Contains(name, term) {
				utility = true
				break
			}
		}
		if !utility {
			hasCombatDrones = true
			break
		}
	}
	droneWeapon := ""
	switch {
	case has("fighter support unit"):
		droneWeapon = "Fighter"
	case drone("bouncer", "curator", "garde", "warden") && hasDroneDamage:
		droneWeapon = "Sentry Drone"
	case hasDroneDamage || hasCombatDrones:
		droneWeapon = "Drone"
	}
	offense := weapon
	if droneWeapon != "" {
		offense = droneWeapon
		if weapon != "" {
			offense += "/" + weapon
		}
	}
	mode := ""
	for _, choice := range []struct{ Label, Term string }{
		{"Industrial Cyno", "industrial cynosural field generator"}, {"Cyno", "cynosural field generator"},
		{"Rolling", "higgs anchor"}, {"Interdiction", "interdiction sphere launcher"}, {"Bomber", "bomb launcher"},
		{"Siege", "siege module"}, {"Bastion", "bastion module"}, {"Triage", "triage module"},
		{"Jump Field", "micro jump field generator"}, {"Industrial Command", "industrial core"},
	} {
		if has(choice.Term) {
			mode = choice.Label
			break
		}
	}
	hasWeb, hasPainter := has("stasis webifier", "stasis grappler"), has("target painter")
	ewar := ""
	switch {
	case has("ecm", "burst jammer"):
		ewar = "ECM"
	case has("remote sensor dampener"):
		ewar = "Sensor Damping"
	case has("weapon disruptor", "tracking disruptor", "guidance disruptor"):
		ewar = "Weapon Disruption"
	case hasWeb && hasPainter:
		ewar = "Web/Painter"
	case hasPainter:
		ewar = "Target Painting"
	case hasWeb && !has("warp scrambler"):
		ewar = "Web Support"
	}
	role := ""
	switch {
	case has("cynosural field generator", "industrial cynosural field generator"):
	case has("bastion module", "siege module", "interdiction sphere launcher", "bomb launcher") && offense != "":
		role = offense
	case has("remote shield booster"):
		role = "Remote Shield Boost"
	case has("remote armor repairer"):
		role = "Remote Armor Repair"
	case has("gas cloud scoop", "gas cloud harvester"):
		role = "Gas Harvesting"
	case has("mining laser", "strip miner"):
		role = "Mining"
	case has("command burst"):
		role = "Command"
	case has("data analyzer", "relic analyzer", "integrated analyzer"):
		role = "Exploration"
	case has("salvager"):
		role = "Salvaging"
	case has("probe launcher") && has("scan acquisition", "scan pinpointing", "scan rangefinding"):
		role = "Scanning"
	case offense != "":
		role = offense
	case has("energy neutralizer"):
		role = "Neut"
	case ewar != "":
		role = ewar
	case has("expanded cargohold", "cargohold optimization") || strings.Contains(strings.ToLower(hullGroup), "industrial") || strings.Contains(strings.ToLower(hullGroup), "transport ship") || strings.Contains(strings.ToLower(hullGroup), "freighter"):
		role = "Hauling"
	case has("interdiction nullifier"):
		role = "Travel"
	}
	modifier := ""
	if has("covert ops cloaking") {
		modifier = "Covert"
	}
	tank := ""
	switch {
	case has("shield booster", "shield extender", "shield hardener", "shield recharger"):
		tank = "Shield"
	case has("armor repairer", "armor plate", "steel plates", "restrained plates", "energized adaptive"):
		tank = "Armor"
	case has("reinforced bulkhead", "transverse bulkhead"):
		tank = "Hull"
	}
	style := ""
	switch {
	case has("warp scrambler", "stasis webifier") && has("heavy assault missile", "autocannon", "blaster", "pulse laser"):
		style = "Brawler"
	case has("warp disruptor") && has("artillery", "beam laser", "railgun", "heavy missile", "cruise missile"):
		style = "Kite"
	case fitLocalBoost(names, "shield booster", "remote shield booster") || fitLocalBoost(names, "armor repairer", "remote armor repairer"):
		style = "Active"
	case has("shield power relay", "shield recharger", "core defense field purger"):
		style = "Passive"
	case has("shield extender", "armor plate", "steel plates", "restrained plates", "reinforced bulkhead", "transverse bulkhead"):
		style = "Buffer"
	}
	identity := []string{mode, role}
	if mode == "" && role == "" {
		identity = []string{"General Purpose"}
	}
	parts := []string{modifier}
	parts = append(parts, identity...)
	if weapon != "" && has("energy neutralizer") {
		parts = append(parts, "Neut")
	}
	if strings.HasPrefix(role, "Remote ") && droneWeapon != "" {
		parts = append(parts, droneWeapon)
	}
	if ewar != "" && role != ewar {
		parts = append(parts, ewar)
	}
	parts = append(parts, tank, style)
	clean := make([]string, 0, len(parts))
	for _, part := range parts {
		if part != "" {
			clean = append(clean, part)
		}
	}
	return strings.Join(clean, " ")
}

func fitLocalBoost(names []string, wanted, remote string) bool {
	for _, name := range names {
		if strings.Contains(name, wanted) && !strings.Contains(name, remote) {
			return true
		}
	}
	return false
}

func fitContextParts(context *api.UniverseFitContext) []string {
	if context == nil {
		return nil
	}
	result := []string{}
	rows := append([]api.UniverseFitSecurityContext(nil), context.SecurityDistribution...)
	sort.SliceStable(rows, func(i, j int) bool { return rows[i].Pct > rows[j].Pct })
	primary := ""
	if len(rows) > 0 {
		primary = rows[0].Name
		part := fmt.Sprintf("%.0f%% %s", math.Round(rows[0].Pct), strings.ToLower(primary))
		if len(rows) > 1 && rows[0].Pct < 60 {
			part += fmt.Sprintf(" / %.0f%% %s", math.Round(rows[1].Pct), strings.ToLower(rows[1].Name))
		}
		result = append(result, part)
	}
	if region := context.TopRegion; region != nil && region.Name != "" && region.Pct >= 35 && !strings.EqualFold(region.Name, primary) {
		result = append(result, fmt.Sprintf("%s %.0f%%", region.Name, math.Round(region.Pct)))
	}
	if context.MedianAttackers > 0 {
		attackers := int(math.Round(context.MedianAttackers))
		if attackers == 1 {
			result = append(result, "median solo loss")
		} else {
			result = append(result, fmt.Sprintf("median %d attackers", attackers))
		}
	}
	if context.MedianLossValue > 0 {
		result = append(result, "median loss "+fitLossValue(context.MedianLossValue))
	}
	return result
}

func fitLossValue(value float64) string {
	if value >= 1e9 {
		if value >= 1e10 {
			return fmt.Sprintf("%.0fb", value/1e9)
		}
		return fmt.Sprintf("%.1fb", value/1e9)
	}
	if value >= 1e8 {
		return fmt.Sprintf("%.0fm", value/1e6)
	}
	return fmt.Sprintf("%.1fm", value/1e6)
}

func fitStatParts(stats *api.UniverseFitStats) []string {
	if stats == nil {
		return nil
	}
	result := []string{}
	if stats.EHP != nil {
		result = append(result, fitMetricValue("ehp", *stats.EHP)+" EHP")
	}
	if stats.DPS != nil {
		result = append(result, fitMetricValue("dps", *stats.DPS)+" DPS")
	}
	if stats.Repair != nil && *stats.Repair > 0 {
		result = append(result, fitMetricValue("repair", *stats.Repair)+" EHP/s")
	}
	if stats.Speed != nil {
		result = append(result, fitMetricValue("speed", *stats.Speed))
	}
	if stats.NPCEHP != nil && stats.NPCProfile != "" {
		result = append(result, fitMetricValue("ehp", *stats.NPCEHP)+" vs "+fitNPCLabel(stats.NPCProfile))
	}
	return result
}
