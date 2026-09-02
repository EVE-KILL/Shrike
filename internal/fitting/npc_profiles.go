package fitting

import (
	"fmt"
	"strings"
)

// NPCDamageProfile is an approximate incoming damage mix. Percentages are
// normalized from EVE University's listed primary damage types; omitted minor
// damage varies between individual NPCs and sites.
type NPCDamageProfile struct {
	ID, Name                        string
	EM, Thermal, Kinetic, Explosive float64
}

var NPCDamageProfiles = []NPCDamageProfile{
	{ID: "omni", Name: "Omnidamage", EM: .25, Thermal: .25, Kinetic: .25, Explosive: .25},
	normalizedProfile("angels", "Angel Cartel", 0, 0, .22, .62),
	normalizedProfile("blood-raiders", "Blood Raiders", .50, .48, 0, 0),
	normalizedProfile("guristas", "Guristas Pirates", 0, .18, .79, 0),
	normalizedProfile("mordus", "Mordu's Legion", 0, .30, .70, 0),
	normalizedProfile("sansha", "Sansha's Nation", .53, .47, 0, 0),
	normalizedProfile("serpentis", "Serpentis", 0, .55, .45, 0),
	normalizedProfile("triglavian", "Triglavian Collective", 0, .60, 0, .40),
	normalizedProfile("amarr", "Amarr Empire", .47, .42, 0, 0),
	normalizedProfile("caldari", "Caldari State", 0, .48, .51, 0),
	normalizedProfile("gallente", "Gallente Federation", 0, .39, .60, 0),
	normalizedProfile("minmatar", "Minmatar Republic", 0, 0, .31, .50),
}

func normalizedProfile(id, name string, em, thermal, kinetic, explosive float64) NPCDamageProfile {
	total := em + thermal + kinetic + explosive
	return NPCDamageProfile{ID: id, Name: name, EM: em / total, Thermal: thermal / total, Kinetic: kinetic / total, Explosive: explosive / total}
}

func NPCProfile(id string) (NPCDamageProfile, bool) {
	for _, profile := range NPCDamageProfiles {
		if profile.ID == id {
			return profile, true
		}
	}
	return NPCDamageProfile{}, false
}

// NPCDamageEHPExpression returns a SQL expression for layered EHP against a
// whitelisted profile. The alias must be a trusted identifier supplied by the
// caller, never request input.
func NPCDamageEHPExpression(alias string, profile NPCDamageProfile) string {
	layer := func(name string) string {
		return fmt.Sprintf("COALESCE(%[1]s.%[2]s_hp / NULLIF(%[3]g*(1-COALESCE(%[1]s.%[2]s_em_resist,0))+%[4]g*(1-COALESCE(%[1]s.%[2]s_thermal_resist,0))+%[5]g*(1-COALESCE(%[1]s.%[2]s_kinetic_resist,0))+%[6]g*(1-COALESCE(%[1]s.%[2]s_explosive_resist,0)),0),0)", alias, name, profile.EM, profile.Thermal, profile.Kinetic, profile.Explosive)
	}
	return strings.Join([]string{layer("shield"), layer("armor"), layer("hull")}, "+")
}
