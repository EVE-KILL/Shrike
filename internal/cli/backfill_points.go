package cli

import (
	"fmt"
	"sync/atomic"

	"github.com/eve-kill/shrike/internal/killmail"
	"github.com/eve-kill/shrike/internal/ui"
	"github.com/spf13/cobra"
	"golang.org/x/sync/errgroup"
)

var (
	flagPointSharesFromID  int64
	flagPointSharesToID    int64
	flagPointSharesChunk   int
	flagPointSharesWorkers int
)

var backfillPointSharesCmd = &cobra.Command{
	Use:   "attacker-points",
	Short: "Allocate every killmail point pool between its player attackers",
	RunE: func(cmd *cobra.Command, _ []string) error {
		pool, err := openPool(cmd)
		if err != nil {
			return err
		}
		defer pool.Close()
		var minID, maxID int64
		if err := pool.QueryRow(cmd.Context(), `
			SELECT coalesce(min(killmail_id), 0), coalesce(max(killmail_id), 0)
			FROM killmails
			WHERE ($1::bigint = 0 OR killmail_id >= $1)
			  AND ($2::bigint = 0 OR killmail_id <= $2)`,
			flagPointSharesFromID, flagPointSharesToID).Scan(&minID, &maxID); err != nil {
			return err
		}
		if minID == 0 || maxID == 0 {
			return fmt.Errorf("no killmails in requested ID range")
		}
		chunkSize := int64(max(1, flagPointSharesChunk))
		workers := max(1, flagPointSharesWorkers)
		ui.Section("Backfill attacker point shares")
		ui.KV("Killmail IDs", fmt.Sprintf("%d–%d", minID, maxID))
		ui.KV("Chunk size", fmtCount(chunkSize))
		ui.KV("Workers", fmt.Sprint(workers))
		if !flagBackfillApply {
			ui.Warn("Dry run. Re-run with --apply to write point shares.")
			return nil
		}

		type idRange struct{ from, to int64 }
		jobs := make(chan idRange)
		group, groupCtx := errgroup.WithContext(cmd.Context())
		var mails, attackers atomic.Int64
		for range workers {
			group.Go(func() error {
				for job := range jobs {
					mailCount, attackerCount, err := killmail.BackfillPointShares(groupCtx, pool, job.from, job.to)
					if err != nil {
						return fmt.Errorf("attacker points IDs %d-%d: %w", job.from, job.to, err)
					}
					mails.Add(mailCount)
					attackers.Add(attackerCount)
				}
				return nil
			})
		}
		group.Go(func() error {
			defer close(jobs)
			for from := minID; from <= maxID; from += chunkSize {
				job := idRange{from: from, to: min(from+chunkSize-1, maxID)}
				select {
				case jobs <- job:
				case <-groupCtx.Done():
					return groupCtx.Err()
				}
			}
			return nil
		})
		if err := group.Wait(); err != nil {
			return err
		}
		ui.Success("Allocated %s killmails; changed %s attacker rows.", fmtCount(mails.Load()), fmtCount(attackers.Load()))
		return nil
	},
}

func init() {
	backfillPointSharesCmd.Flags().Int64Var(&flagPointSharesFromID, "from-id", 0, "First killmail ID, inclusive")
	backfillPointSharesCmd.Flags().Int64Var(&flagPointSharesToID, "to-id", 0, "Last killmail ID, inclusive")
	backfillPointSharesCmd.Flags().IntVar(&flagPointSharesChunk, "chunk", 25_000, "Killmail ID span per transaction")
	backfillPointSharesCmd.Flags().IntVar(&flagPointSharesWorkers, "workers", 4, "Concurrent database workers")
	backfillPointSharesCmd.Flags().BoolVar(&flagBackfillApply, "apply", false, "Write calculated point shares")
	backfillCmd.AddCommand(backfillPointSharesCmd)
}
