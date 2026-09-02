package cli

import (
	"bufio"
	"context"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/eve-kill/shrike/internal/ui"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/spf13/cobra"
)

// Exporting the entity tables as JSON.
//
// For moving entities somewhere that is not this database — a fresh instance,
// another tool, an archive. Written as a streaming array rather than built in
// memory: the character table is millions of rows and holding the encoded form
// of all of them before writing the first byte would need more memory than the
// machine has.

var (
	flagExportOut   string
	flagExportBatch int
)

var exportCmd = &cobra.Command{
	Use:   "export",
	Short: "Write database contents out as files",
}

var exportEntitiesCmd = &cobra.Command{
	Use:   "entities [out]",
	Short: "Export characters, corporations and alliances as JSON",
	Long: `Streams the three entity tables into JSON arrays.

Each character carries its corporation history and each corporation its
alliance history, so a row is self-contained rather than needing a join against
a second file. Alliances have no history and are written flat.

Written incrementally, so the memory cost is one batch rather than one table.

Examples:
  shrike export:entities
  shrike export:entities ./dump
  shrike export:entities ../entity_export --batch 5000`,
	Args: cobra.MaximumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		pool, err := openPool(cmd)
		if err != nil {
			return err
		}
		defer pool.Close()

		out := flagExportOut
		if len(args) == 1 {
			out = args[0]
		}
		outDir, err := filepath.Abs(out)
		if err != nil {
			return err
		}
		if err := os.MkdirAll(outDir, 0o750); err != nil {
			return err
		}

		ui.Section("Entity export")
		ui.KV("Directory", outDir)
		ui.Newline()

		start := time.Now()
		for _, ex := range entityExports() {
			n, err := ex.run(cmd.Context(), pool, outDir, flagExportBatch)
			if err != nil {
				return fmt.Errorf("export %s: %w", ex.name, err)
			}
			ui.KV(ex.name, fmt.Sprintf("%s rows → %s.json", fmtCount(n), ex.name))
		}

		ui.Newline()
		ui.Success("Export finished in %s", time.Since(start).Round(time.Millisecond))
		return nil
	},
}

// entityExport is one table's export.
type entityExport struct {
	name string

	// query selects the rows as a single JSON object per row, keyed by id.
	//
	// Built in Postgres rather than in Go: the history arrays are a correlated
	// subquery per row, and having the database assemble the whole document
	// means one round trip per batch instead of one per entity for its history.
	query string
	idCol string
}

func entityExports() []entityExport {
	return []entityExport{
		{
			name:  "characters",
			idCol: "c.character_id",
			query: `
                SELECT c.character_id, jsonb_build_object(
                    'character_id', c.character_id,
                    'name', c.name,
                    'description', c.description,
                    'birthday', CASE WHEN c.birthday IS NULL THEN NULL ELSE
                        (c.birthday AT TIME ZONE 'UTC')::text || '+00' END,
                    'gender', c.gender,
                    'race_id', c.race_id,
                    'security_status', c.security_status,
                    'bloodline_id', c.bloodline_id,
                    'corporation_id', c.corporation_id,
                    'alliance_id', c.alliance_id,
                    'faction_id', c.faction_id,
                    'deleted', c.deleted,
                    'last_active', CASE WHEN c.last_active IS NULL THEN NULL ELSE
                        (c.last_active AT TIME ZONE 'UTC')::text || '+00' END,
                    'createdAt', CASE WHEN c.created_at IS NULL THEN NULL ELSE
                        (c.created_at AT TIME ZONE 'UTC')::text || '+00' END,
                    'updatedAt', CASE WHEN c.updated_at IS NULL THEN NULL ELSE
                        (c.updated_at AT TIME ZONE 'UTC')::text || '+00' END,
                    'history', coalesce((
                        SELECT jsonb_agg(jsonb_build_object(
                            'record_id', h.record_id,
                            'corporation_id', h.corporation_id,
                            'start_date', to_char(
                                h.start_date AT TIME ZONE 'UTC',
                                'YYYY-MM-DD"T"HH24:MI:SS.MS"Z"'
                            )
                        ) ORDER BY h.record_id)
                        FROM character_corporation_history h
                        WHERE h.character_id = c.character_id
                    ), '[]'::jsonb)
                )
                FROM characters c
                WHERE c.character_id > $1
                ORDER BY c.character_id
                LIMIT $2`,
		},
		{
			name:  "corporations",
			idCol: "c.corporation_id",
			query: `
                SELECT c.corporation_id, jsonb_build_object(
                    'corporation_id', c.corporation_id,
                    'name', c.name,
                    'ticker', c.ticker,
                    'description', c.description,
                    'date_founded', CASE WHEN c.date_founded IS NULL THEN NULL ELSE
                        (c.date_founded AT TIME ZONE 'UTC')::text || '+00' END,
                    'alliance_id', c.alliance_id,
                    'faction_id', c.faction_id,
                    'ceo_id', c.ceo_id,
                    'creator_id', c.creator_id,
                    'home_station_id', c.home_station_id,
                    'member_count', c.member_count,
                    'shares', c.shares::text,
                    'tax_rate', c.tax_rate,
                    'url', c.url,
                    'war_eligible', c.war_eligible,
                    'deleted', c.deleted,
                    'createdAt', CASE WHEN c.created_at IS NULL THEN NULL ELSE
                        (c.created_at AT TIME ZONE 'UTC')::text || '+00' END,
                    'updatedAt', CASE WHEN c.updated_at IS NULL THEN NULL ELSE
                        (c.updated_at AT TIME ZONE 'UTC')::text || '+00' END,
                    'history', coalesce((
                        SELECT jsonb_agg(jsonb_build_object(
                            'record_id', h.record_id,
                            'alliance_id', h.alliance_id,
                            'start_date', to_char(
                                h.start_date AT TIME ZONE 'UTC',
                                'YYYY-MM-DD"T"HH24:MI:SS.MS"Z"'
                            )
                        ) ORDER BY h.record_id)
                        FROM corporation_alliance_history h
                        WHERE h.corporation_id = c.corporation_id
                    ), '[]'::jsonb)
                )
                FROM corporations c
                WHERE c.corporation_id > $1
                ORDER BY c.corporation_id
                LIMIT $2`,
		},
		{
			name:  "alliances",
			idCol: "a.alliance_id",
			query: `
                SELECT a.alliance_id, jsonb_build_object(
                    'alliance_id', a.alliance_id,
                    'name', a.name,
                    'ticker', a.ticker,
                    'creator_id', a.creator_id,
                    'creator_corporation_id', a.creator_corporation_id,
                    'executor_corporation_id', a.executor_corporation_id,
                    'date_founded', CASE WHEN a.date_founded IS NULL THEN NULL ELSE
                        (a.date_founded AT TIME ZONE 'UTC')::text || '+00' END,
                    'faction_id', a.faction_id,
                    'corporation_count', a.corporation_count,
                    'member_count', a.member_count,
                    'deleted', a.deleted,
                    'createdAt', CASE WHEN a.created_at IS NULL THEN NULL ELSE
                        (a.created_at AT TIME ZONE 'UTC')::text || '+00' END,
                    'updatedAt', CASE WHEN a.updated_at IS NULL THEN NULL ELSE
                        (a.updated_at AT TIME ZONE 'UTC')::text || '+00' END
                )
                FROM alliances a
                WHERE a.alliance_id > $1
                ORDER BY a.alliance_id
                LIMIT $2`,
		},
	}
}

// run streams one table to a file.
//
// Paged by id rather than OFFSET, so the cost of reaching page N does not grow
// with N. The file is written as it goes and the array brackets are emitted by
// hand, which is what keeps this to one batch of memory rather than one table.
func (e entityExport) run(ctx context.Context, pool *pgxpool.Pool, dir string, batch int) (int64, error) {
	if batch < 1 {
		batch = 1
	}
	path := filepath.Join(dir, e.name+".json")
	// #nosec G304 -- path is the explicit operator-selected export destination.
	f, err := os.Create(path)
	if err != nil {
		return 0, err
	}
	defer f.Close() //nolint:errcheck // the explicit Close below is the one that matters

	w := bufio.NewWriterSize(f, 1<<20)
	if _, err := w.WriteString("["); err != nil {
		return 0, err
	}

	var cursor int64
	var written int64

	for {
		rows, err := pool.Query(ctx, e.query, cursor, batch)
		if err != nil {
			return written, err
		}

		var inBatch int
		for rows.Next() {
			var id int64
			var doc []byte
			if err := rows.Scan(&id, &doc); err != nil {
				rows.Close()
				return written, err
			}
			if written > 0 {
				if _, err := w.WriteString(","); err != nil {
					rows.Close()
					return written, err
				}
			}
			if _, err := w.Write(doc); err != nil {
				rows.Close()
				return written, err
			}
			cursor = id
			written++
			inBatch++
		}
		rows.Close()
		if err := rows.Err(); err != nil {
			return written, err
		}

		if inBatch < batch {
			break
		}
	}

	if _, err := w.WriteString("]"); err != nil {
		return written, err
	}
	if err := w.Flush(); err != nil {
		return written, err
	}
	// Closed explicitly so a write error surfaces here rather than being
	// swallowed by the deferred close — a truncated export that reported
	// success would be worse than one that failed.
	if err := f.Close(); err != nil {
		return written, err
	}
	return written, nil
}

func init() {
	exportEntitiesCmd.Flags().StringVar(&flagExportOut, "out", "entity_export", "Output directory")
	exportEntitiesCmd.Flags().IntVarP(&flagExportBatch, "batch", "b", 1000, "Rows per query")
	exportCmd.AddCommand(exportEntitiesCmd)
}
