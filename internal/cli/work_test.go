package cli

import (
	"context"
	"testing"

	"github.com/eve-kill/shrike/internal/cron"
	"github.com/eve-kill/shrike/internal/queue"
	"github.com/riverqueue/river"
)

func TestEveryRiverWorkerRoleCarriesThePeriodicSchedule(t *testing.T) {
	registry := cron.NewRegistry()
	registry.MustRegister("status_update", func(context.Context) (string, error) {
		return "", nil
	})
	riverWorkers := river.NewWorkers()

	for _, tc := range []struct {
		name   string
		queues []string
	}{
		{name: "general queues", queues: []string{"killmails"}},
		{name: "cron queue", queues: []string{queue.CronQueue}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			options, err := riverWorkerOptions(nil, riverWorkers, registry, tc.queues)
			if err != nil {
				t.Fatal(err)
			}
			if len(options.PeriodicJobs) != 1 {
				t.Fatalf(
					"worker has %d periodic jobs, want 1; a leader without the schedule "+
						"would leave the cron queue empty",
					len(options.PeriodicJobs),
				)
			}
			if len(options.Queues) != 1 || options.Queues[0] != tc.queues[0] {
				t.Errorf("queues = %v, want %v", options.Queues, tc.queues)
			}
		})
	}
}
