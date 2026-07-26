package cli

import (
	"fmt"
	"path/filepath"
	"strconv"

	"github.com/eve-kill/shrike/internal/db"
	"github.com/eve-kill/shrike/internal/migrate"
	"github.com/eve-kill/shrike/internal/ui"
	"github.com/spf13/cobra"
)

var dbCmd = &cobra.Command{
	Use:   "db",
	Short: "Database schema and migrations",
}

// flagApply gates every mutating db command. The existing Bun CLI requires
// `--apply` on db:migrate, so the muscle memory is preserved: without it these
// commands print the plan and change nothing.
var flagApply bool

var dbMigrationsCmd = &cobra.Command{
	Use:   "migrations",
	Short: "Show migration state and what would be applied",
	RunE: func(cmd *cobra.Command, _ []string) error {
		if err := requireConfig(); err != nil {
			return err
		}
		pool, err := db.New(cmd.Context(), cfg)
		if err != nil {
			return err
		}
		defer pool.Close()

		st, err := migrate.Inspect(cmd.Context(), pool)
		if err != nil {
			return err
		}

		if ui.JSONMode {
			return ui.JSON(st)
		}

		ui.Section("Database")
		ui.KV("Tables", strconv.Itoa(st.TableCount))
		ui.KV("Goose ledger", boolLabel(st.HasLedger, "present", "absent"))
		if st.HasLedger {
			ui.KV("Version", strconv.FormatInt(st.DBVersion, 10))
		}

		ui.Section("Migrations")
		table := ui.NewTable("VERSION", "STATUS", "APPLIED AT", "SOURCE")
		for _, m := range st.Migrations {
			status, at := ui.StatusBadge("ok"), ""
			if !m.Applied {
				status = ui.StatusBadge("pending")
			} else {
				at = m.At.Format("2006-01-02 15:04:05")
			}
			table.Row(strconv.FormatInt(m.Version, 10), status, at, filepath.Base(m.Source))
		}
		fmt.Println(table.Render())
		ui.Newline()

		// The production shape: schema present, goose unaware of it.
		if st.NeedsBaselin {
			ui.Warn("This database has %d tables but no goose ledger.", st.TableCount)
			ui.Newline()
			fmt.Printf("  %s\n", ui.Dim("Its schema was created outside goose (drizzle). Applying migration 1"))
			fmt.Printf("  %s\n", ui.Dim("here would try to recreate live tables, so db:migrate will refuse."))
			ui.Newline()
			fmt.Printf("  Stamp it as already-migrated instead: %s\n",
				ui.Command("shrike db:baseline --apply"))
			ui.Newline()
			return nil
		}

		if pending := st.Pending(); len(pending) > 0 {
			ui.Warn("%d migration(s) pending — run %s", len(pending), "shrike db:migrate --apply")
			ui.Newline()
		} else {
			ui.Success("Schema is up to date.")
			ui.Newline()
		}
		return nil
	},
}

var dbMigrateCmd = &cobra.Command{
	Use:   "migrate",
	Short: "Apply pending migrations",
	Long: `Applies every pending migration in order.

Refuses to run when the database already has a schema but no goose ledger —
that is production's shape, where the tables were created by drizzle, and
applying migration 1 there would attempt to recreate live tables. Use
db:baseline for those databases.

Requires --apply; without it the plan is printed and nothing changes.`,
	RunE: func(cmd *cobra.Command, _ []string) error {
		if err := requireConfig(); err != nil {
			return err
		}
		pool, err := db.New(cmd.Context(), cfg)
		if err != nil {
			return err
		}
		defer pool.Close()

		st, err := migrate.Inspect(cmd.Context(), pool)
		if err != nil {
			return err
		}

		pending := st.Pending()
		if len(pending) == 0 && !st.NeedsBaselin {
			ui.Success("Nothing to do — schema is up to date.")
			ui.Newline()
			return nil
		}

		if !flagApply {
			ui.Section("Would apply")
			table := ui.NewTable("VERSION", "SOURCE")
			for _, m := range pending {
				table.Row(strconv.FormatInt(m.Version, 10), filepath.Base(m.Source))
			}
			fmt.Println(table.Render())
			ui.Newline()
			if st.NeedsBaselin {
				ui.Error("Refusing: %d existing tables and no goose ledger.", st.TableCount)
				ui.Newline()
				fmt.Printf("  Run %s instead.\n", ui.Command("shrike db:baseline --apply"))
				ui.Newline()
				return fmt.Errorf("baseline required")
			}
			ui.Warn("Dry run. Re-run with --apply to execute.")
			ui.Newline()
			return nil
		}

		if err := migrate.Apply(cmd.Context(), pool); err != nil {
			return err
		}

		after, err := migrate.Inspect(cmd.Context(), pool)
		if err != nil {
			return err
		}
		ui.Success("Applied %d migration(s) — now at version %d.", len(pending), after.DBVersion)
		ui.Newline()
		return nil
	},
}

var dbBaselineCmd = &cobra.Command{
	Use:   "baseline",
	Short: "Record the baseline as applied without executing it",
	Long: `Stamps migration 1 as already-applied without running a single statement.

This is how a database whose schema predates goose joins its migration history
— production, whose 102 tables were created by drizzle. Nothing is created,
altered, or dropped; only the goose ledger is written.

Refuses on an empty database, where stamping would skip table creation
permanently and leave a schema that can never be migrated into place.

Requires --apply.`,
	RunE: func(cmd *cobra.Command, _ []string) error {
		if err := requireConfig(); err != nil {
			return err
		}
		pool, err := db.New(cmd.Context(), cfg)
		if err != nil {
			return err
		}
		defer pool.Close()

		st, err := migrate.Inspect(cmd.Context(), pool)
		if err != nil {
			return err
		}

		if !flagApply {
			ui.Section("Would baseline")
			ui.KV("Tables found", strconv.Itoa(st.TableCount))
			ui.KV("Stamp version", strconv.FormatInt(migrate.BaselineVersion, 10))
			ui.KV("Statements run", ui.Bold("none"))
			ui.Newline()
			ui.Warn("Dry run. Re-run with --apply to write the ledger.")
			ui.Newline()
			return nil
		}

		if err := migrate.Baseline(cmd.Context(), pool); err != nil {
			return err
		}
		ui.Success("Stamped version %d as applied. No schema changes were made.",
			migrate.BaselineVersion)
		ui.Newline()
		return nil
	},
}

func boolLabel(b bool, yes, no string) string {
	if b {
		return yes
	}
	return ui.Dim(no)
}

func init() {
	dbMigrateCmd.Flags().BoolVar(&flagApply, "apply", false, "Actually execute; without it this is a dry run")
	dbBaselineCmd.Flags().BoolVar(&flagApply, "apply", false, "Actually write the ledger; without it this is a dry run")
	dbCmd.AddCommand(dbStatusCmd, dbMigrationsCmd, dbMigrateCmd, dbBaselineCmd)
}
