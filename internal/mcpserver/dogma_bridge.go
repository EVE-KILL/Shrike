package mcpserver

import (
	"context"

	"github.com/eve-kill/shrike/internal/dogma"
)

type EsfSlot = dogma.Slot
type EsfCharge = dogma.Charge
type EsfModule = dogma.Module
type EsfDrone = dogma.Drone
type EsfFit = dogma.Fit
type HullStats dogma.Stats

func evaluateDogma(ctx context.Context, fit EsfFit, noSkills bool) (HullStats, error) {
	if noSkills {
		empty := map[string]int64{}
		stats, err := dogma.Evaluate(ctx, fit, &empty)
		return HullStats(stats), err
	}
	stats, err := dogma.Evaluate(ctx, fit, nil)
	return HullStats(stats), err
}
