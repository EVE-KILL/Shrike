package cli

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/eve-kill/shrike/internal/ui"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/spf13/cobra"
)

var (
	flagDBStatusLive     bool
	flagDBStatusKillPID  int32
	flagDBStatusFull     bool
	flagDBStatusInterval float64
)

var dbStatusCmd = &cobra.Command{
	Use:   "status",
	Short: "Show database activity, table sizes, progress, and lock waits",
	RunE: func(cmd *cobra.Command, _ []string) error {
		pool, err := openPool(cmd)
		if err != nil {
			return err
		}
		defer pool.Close()

		if flagDBStatusKillPID != 0 {
			var terminated bool
			if err := pool.QueryRow(cmd.Context(),
				`SELECT pg_terminate_backend($1)`, flagDBStatusKillPID).Scan(&terminated); err != nil {
				return err
			}
			if !terminated {
				return fmt.Errorf("postgres did not terminate PID %d", flagDBStatusKillPID)
			}
			ui.Success("Sent terminate signal to PID %d.", flagDBStatusKillPID)
			return nil
		}

		if !flagDBStatusLive {
			return renderDBStatus(cmd.Context(), pool, flagDBStatusFull)
		}
		if flagDBStatusInterval <= 0 {
			return fmt.Errorf("--interval must be positive")
		}

		return RunService(cmd, "db status", func(ctx context.Context) error {
			interval := time.Duration(flagDBStatusInterval * float64(time.Second))
			ticker := time.NewTicker(interval)
			defer ticker.Stop()
			for {
				if !ui.JSONMode {
					fmt.Print("\x1b[2J\x1b[H")
				}
				if err := renderDBStatus(ctx, pool, flagDBStatusFull); err != nil {
					return err
				}
				select {
				case <-ctx.Done():
					return ctx.Err()
				case <-ticker.C:
				}
			}
		})
	},
}

type dbOverview struct {
	Size                      string
	Connections, Active, Idle int64
	Inserts, Updates, Deletes int64
	Commits, Rollbacks        int64
}

func renderDBStatus(ctx context.Context, pool *pgxpool.Pool, full bool) error {
	var overview dbOverview
	if err := pool.QueryRow(ctx, `
		SELECT pg_size_pretty(pg_database_size(current_database())),
		       count(*)::bigint,
		       count(*) FILTER (WHERE activity.state = 'active')::bigint,
		       count(*) FILTER (WHERE activity.state = 'idle')::bigint,
		       stats.tup_inserted, stats.tup_updated, stats.tup_deleted,
		       stats.xact_commit, stats.xact_rollback
		FROM pg_stat_activity activity
		CROSS JOIN pg_stat_database stats
		WHERE activity.datname = current_database()
		  AND stats.datname = current_database()
		GROUP BY stats.tup_inserted, stats.tup_updated, stats.tup_deleted,
		         stats.xact_commit, stats.xact_rollback`).
		Scan(&overview.Size, &overview.Connections, &overview.Active, &overview.Idle,
			&overview.Inserts, &overview.Updates, &overview.Deletes,
			&overview.Commits, &overview.Rollbacks); err != nil {
		return err
	}

	ui.Section("Database status — " + time.Now().UTC().Format("15:04:05 UTC"))
	ui.KV("Size", overview.Size)
	ui.KV("Connections", fmt.Sprintf("%d (%d active, %d idle)",
		overview.Connections, overview.Active, overview.Idle))
	ui.KV("Rows changed", fmt.Sprintf("%d inserted, %d updated, %d deleted",
		overview.Inserts, overview.Updates, overview.Deletes))
	ui.KV("Transactions", fmt.Sprintf("%d committed, %d rolled back",
		overview.Commits, overview.Rollbacks))

	if err := renderDBTables(ctx, pool); err != nil {
		return err
	}
	if err := renderDBProgress(ctx, pool, full); err != nil {
		return err
	}
	if err := renderDBQueries(ctx, pool, full); err != nil {
		return err
	}
	return renderDBLocks(ctx, pool, full)
}

func renderDBTables(ctx context.Context, pool *pgxpool.Pool) error {
	rows, err := pool.Query(ctx, `
		WITH table_stats AS (
			SELECT CASE
			         WHEN c.relname ~ '_\d{4}(_\d{2})*$'
			         THEN regexp_replace(c.relname, '_\d{4}(_\d{2})*$', '')
			         ELSE c.relname
			       END AS name,
			       pg_total_relation_size(c.oid) AS total_bytes,
			       pg_relation_size(c.oid) AS data_bytes,
			       greatest(c.reltuples::bigint, coalesce(s.n_live_tup, 0)) AS rows,
			       coalesce(s.n_dead_tup, 0) AS dead
			FROM pg_class c
			JOIN pg_namespace n ON n.oid = c.relnamespace
			LEFT JOIN pg_stat_user_tables s ON s.relid = c.oid
			WHERE n.nspname = 'public' AND c.relkind = 'r'
		)
		SELECT name, pg_size_pretty(sum(total_bytes)::bigint),
		       pg_size_pretty(sum(data_bytes)::bigint),
		       sum(rows)::bigint, sum(dead)::bigint
		FROM table_stats
		GROUP BY name
		HAVING sum(total_bytes) > 1048576
		ORDER BY sum(total_bytes) DESC
		LIMIT 15`)
	if err != nil {
		return err
	}
	defer rows.Close()

	table := ui.NewTable("TABLE", "TOTAL", "DATA", "ROWS", "DEAD")
	var count int
	for rows.Next() {
		var name, total, data string
		var live, dead int64
		if err := rows.Scan(&name, &total, &data, &live, &dead); err != nil {
			return err
		}
		table.Row(name, total, data, fmtCount(live), fmtCount(dead))
		count++
	}
	if err := rows.Err(); err != nil {
		return err
	}
	if count > 0 {
		ui.Section("Top tables")
		fmt.Println(table.Render())
	}
	return nil
}

func renderDBProgress(ctx context.Context, pool *pgxpool.Pool, full bool) error {
	rows, err := pool.Query(ctx, `
		SELECT 'INDEX', p.pid, p.phase,
		       coalesce(a.query, '')
		FROM pg_stat_progress_create_index p
		LEFT JOIN pg_stat_activity a ON a.pid = p.pid
		UNION ALL
		SELECT 'VACUUM', p.pid, p.phase, ''
		FROM pg_stat_progress_vacuum p
		UNION ALL
		SELECT 'ANALYZE', p.pid, p.phase,
		       coalesce(a.query, '')
		FROM pg_stat_progress_analyze p
		LEFT JOIN pg_stat_activity a ON a.pid = p.pid`)
	if err != nil {
		// Older Postgres versions may not expose every progress view. This
		// section is diagnostic, so absence should not hide the rest.
		return nil
	}
	defer rows.Close()

	table := ui.NewTable("OPERATION", "PID", "PHASE", "QUERY")
	var count int
	for rows.Next() {
		var operation, phase, query string
		var pid int32
		if err := rows.Scan(&operation, &pid, &phase, &query); err != nil {
			return err
		}
		table.Row(operation, fmt.Sprint(pid), phase, displayQuery(query, full))
		count++
	}
	if err := rows.Err(); err != nil {
		return err
	}
	if count > 0 {
		ui.Section("Operations in progress")
		fmt.Println(table.Render())
	}
	return nil
}

func renderDBQueries(ctx context.Context, pool *pgxpool.Pool, full bool) error {
	rows, err := pool.Query(ctx, `
		SELECT pid, state,
		       extract(epoch FROM (now() - query_start))::int,
		       coalesce(wait_event_type, ''), coalesce(wait_event, ''), query
		FROM pg_stat_activity
		WHERE datname = current_database()
		  AND pid <> pg_backend_pid()
		  AND state <> 'idle'
		  AND query NOT LIKE '%pg_stat_activity%'
		ORDER BY query_start`)
	if err != nil {
		return err
	}
	defer rows.Close()

	table := ui.NewTable("PID", "STATE", "AGE", "WAIT", "QUERY")
	var count int
	for rows.Next() {
		var pid, seconds int32
		var state, waitType, wait, query string
		if err := rows.Scan(&pid, &state, &seconds, &waitType, &wait, &query); err != nil {
			return err
		}
		waiting := ""
		if wait != "" {
			waiting = waitType + "/" + wait
		}
		table.Row(fmt.Sprint(pid), state, formatSeconds(seconds), waiting, displayQuery(query, full))
		count++
	}
	if err := rows.Err(); err != nil {
		return err
	}
	if count > 0 {
		ui.Section("Active queries")
		fmt.Println(table.Render())
	}
	return nil
}

func renderDBLocks(ctx context.Context, pool *pgxpool.Pool, full bool) error {
	rows, err := pool.Query(ctx, `
		SELECT blocked.pid, blocker.pid,
		       coalesce(blocked.query, ''), coalesce(blocker.query, '')
		FROM pg_stat_activity blocked
		CROSS JOIN LATERAL unnest(pg_blocking_pids(blocked.pid)) AS blocker_pid
		JOIN pg_stat_activity blocker ON blocker.pid = blocker_pid
		LIMIT 5`)
	if err != nil {
		return err
	}
	defer rows.Close()

	table := ui.NewTable("BLOCKED", "BLOCKER", "BLOCKED QUERY", "BLOCKER QUERY")
	var count int
	for rows.Next() {
		var blocked, blocker int32
		var blockedQuery, blockerQuery string
		if err := rows.Scan(&blocked, &blocker, &blockedQuery, &blockerQuery); err != nil {
			return err
		}
		table.Row(fmt.Sprint(blocked), fmt.Sprint(blocker),
			displayQuery(blockedQuery, full), displayQuery(blockerQuery, full))
		count++
	}
	if err := rows.Err(); err != nil {
		return err
	}
	if count > 0 {
		ui.Section("Lock waits")
		fmt.Println(table.Render())
	}
	return nil
}

func displayQuery(query string, full bool) string {
	query = strings.Join(strings.Fields(query), " ")
	if full || len(query) <= 100 {
		return query
	}
	return query[:97] + "..."
}

func formatSeconds(seconds int32) string {
	switch {
	case seconds < 60:
		return fmt.Sprintf("%ds", seconds)
	case seconds < 3600:
		return fmt.Sprintf("%dm%ds", seconds/60, seconds%60)
	default:
		return fmt.Sprintf("%dh%dm", seconds/3600, seconds%3600/60)
	}
}

func init() {
	dbStatusCmd.Flags().BoolVarP(&flagDBStatusLive, "live", "l", false, "Refresh continuously")
	dbStatusCmd.Flags().Int32VarP(&flagDBStatusKillPID, "kill", "k", 0, "Terminate a backend PID")
	dbStatusCmd.Flags().BoolVar(&flagDBStatusFull, "full", false, "Show full query text")
	dbStatusCmd.Flags().Float64VarP(&flagDBStatusInterval, "interval", "i", 2, "Refresh interval in seconds")
}
