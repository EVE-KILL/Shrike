package cli

import (
	"fmt"

	"github.com/eve-kill/shrike/internal/queue"
	"github.com/eve-kill/shrike/internal/ui"
	"github.com/jackc/pgx/v5"
	"github.com/riverqueue/river"
	"github.com/spf13/cobra"
)

func init() {
	var apply bool
	var limit int
	command := &cobra.Command{
		Use:   "battle-reports",
		Short: "Queue the pending battle report archive",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			if limit < 0 {
				return fmt.Errorf("--limit-batches must be non-negative")
			}
			ctx := cmd.Context()
			pool, err := openPool(cmd)
			if err != nil {
				return err
			}
			defer pool.Close()

			// Use fixed ID boundaries so reruns deduplicate against active jobs
			// even if part of a range has already finished.
			rows, err := pool.Query(ctx, `
				SELECT DISTINCT (battle_id::bigint - 1) / $1::bigint * $1 + 1 AS first_id
				FROM battles WHERE report_pending
				ORDER BY first_id DESC LIMIT NULLIF($2::bigint, 0)`, queue.BattleReportBatchSize, limit)
			if err != nil {
				return err
			}
			starts, err := pgx.CollectRows(rows, pgx.RowTo[int64])
			if err != nil {
				return err
			}
			ui.KV("Pending archive batches", fmtCount(int64(len(starts))))
			if !apply || len(starts) == 0 {
				return nil
			}
			client, err := queue.New(queue.Options{Pool: pool})
			if err != nil {
				return err
			}
			var queued int
			for offset := 0; offset < len(starts); offset += 512 {
				batch := starts[offset:min(offset+512, len(starts))]
				args := make([]river.JobArgs, 0, len(batch))
				for _, first := range batch {
					args = append(args, queue.BattleReportArgs{FirstID: first, LastID: first + queue.BattleReportBatchSize - 1})
				}
				inserted, err := queue.DispatchMany(ctx, client, args, queue.DormantBackfill)
				if err != nil {
					return fmt.Errorf("queued %d archive batches before error: %w", queued, err)
				}
				queued += inserted
			}
			ui.Success("Queued %s battle report batches.", fmtCount(int64(queued)))
			return nil
		},
	}
	command.Flags().BoolVar(&apply, "apply", false, "Enqueue the archive batches")
	command.Flags().IntVar(&limit, "limit-batches", 0, "Maximum number of batches to enqueue, with zero for the full archive")
	backfillCmd.AddCommand(command)
}
