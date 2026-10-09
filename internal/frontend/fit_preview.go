package frontend

import "github.com/eve-kill/shrike/internal/api"

type fitPreviewSlot struct {
	ID          int
	Label, Tone string
	Modules     []api.UniverseFitModule
}

func fitPreviewSlots(fit api.AdvancedSearchFit) []fitPreviewSlot {
	groups := []fitPreviewSlot{{ID: 1, Label: "High", Tone: "text-amber-400"}, {ID: 2, Label: "Med", Tone: "text-gray-300"}, {ID: 3, Label: "Low", Tone: "text-green-400"}, {ID: 4, Label: "Rig", Tone: "text-blue-400"}, {ID: 5, Label: "Sub", Tone: "text-yellow-400"}}
	for _, module := range fit.Modules {
		if module.SlotGroup >= 1 && module.SlotGroup <= 5 {
			groups[module.SlotGroup-1].Modules = append(groups[module.SlotGroup-1].Modules, module)
		}
	}
	out := []fitPreviewSlot{}
	for _, group := range groups {
		if len(group.Modules) > 0 {
			out = append(out, group)
		}
	}
	return out
}
func advancedDedupLabel(mode string) string {
	switch mode {
	case "family":
		return "T1/T2/meta/faction variants grouped"
	case "exact":
		return "Identical module sets grouped"
	default:
		return "Every kill shown individually"
	}
}
