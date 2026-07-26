package cli

import (
	"context"
	"fmt"
	"strconv"
	"time"

	"github.com/eve-kill/shrike/internal/achievements"
	"github.com/eve-kill/shrike/internal/entities"
	"github.com/eve-kill/shrike/internal/esi"
	"github.com/eve-kill/shrike/internal/eve"
	"github.com/eve-kill/shrike/internal/killmail"
	"github.com/eve-kill/shrike/internal/queue"
	"github.com/eve-kill/shrike/internal/stats"
	"github.com/eve-kill/shrike/internal/ui"
	"github.com/jackc/pgx/v5"
	"github.com/spf13/cobra"
)

var debugCmd = &cobra.Command{
	Use:   "debug",
	Short: "Run backend paths interactively",
}

var (
	flagDebugKMRepeat           int
	flagDebugKMSkipInsert       bool
	flagDebugKMSkipAchievements bool
	flagDebugKMSkipStats        bool
	flagDebugKMSkipEntities     bool
)

var debugKillmailCmd = &cobra.Command{
	Use:   "killmail <killmail-id> <hash>",
	Short: "Process one killmail with a timing breakdown",
	Args:  cobra.ExactArgs(2),
	RunE: func(cmd *cobra.Command, args []string) error {
		id, err := strconv.ParseInt(args[0], 10, 64)
		if err != nil {
			return fmt.Errorf("invalid killmail id %q", args[0])
		}
		if flagDebugKMRepeat < 1 {
			return fmt.Errorf("--repeat must be positive")
		}

		pool, err := openPool(cmd)
		if err != nil {
			return err
		}
		defer pool.Close()

		cache, prices, err := loadLookups(cmd.Context(), pool)
		if err != nil {
			return err
		}
		client := esi.New(cfg)
		defer client.Close()

		fetchStart := time.Now()
		response, err := esi.Get[killmail.ESIKillmail](
			cmd.Context(), client, esi.KillmailPath(id, args[1]),
		)
		if err != nil {
			return err
		}
		if !response.OK() || response.Data == nil {
			return fmt.Errorf("ESI returned %d for killmail %d", response.Status, id)
		}

		var queueClient *queue.Client
		if !flagDebugKMSkipEntities {
			queueClient, err = queue.New(queue.Options{Pool: pool})
			if err != nil {
				return err
			}
		}

		ui.Section(fmt.Sprintf("Debug killmail %d", id))
		ui.KV("Fetch", time.Since(fetchStart).Round(time.Millisecond).String())

		for run := 1; run <= flagDebugKMRepeat; run++ {
			start := time.Now()
			parsed, err := killmail.Parse(
				cmd.Context(), cache, prices, response.Data, args[1], 0,
			)
			if err != nil {
				return err
			}
			parseElapsed := time.Since(start)

			var insertElapsed, statsElapsed, achievementElapsed, entitiesElapsed time.Duration
			if !flagDebugKMSkipInsert {
				step := time.Now()
				if _, err := killmail.InsertUntracked(cmd.Context(), pool, parsed); err != nil {
					return err
				}
				insertElapsed = time.Since(step)
			}

			if !flagDebugKMSkipStats {
				step := time.Now()
				km, attackers, err := stats.Load(cmd.Context(), pool, id)
				if err != nil {
					return err
				}
				acc := stats.NewAccumulator()
				acc.Add(km, attackers)
				if _, err := killmail.RunDBEffect(
					cmd.Context(),
					pool,
					id,
					killmail.EffectStatsWritten,
					func(ctx context.Context, tx pgx.Tx) (bool, error) {
						_, err := stats.WritePeriodTx(
							ctx, tx, acc, km.KillmailTime,
							stats.PeriodDaily, true, true,
						)
						return true, err
					},
					killmail.EffectOptions{AllowUntracked: true},
				); err != nil {
					return err
				}
				statsElapsed = time.Since(step)
			}

			if !flagDebugKMSkipAchievements {
				step := time.Now()
				achievement := debugAchievementInput(cache, parsed)
				if _, err := achievements.Process(cmd.Context(), pool, achievement); err != nil {
					return err
				}
				achievementElapsed = time.Since(step)
			}

			if !flagDebugKMSkipEntities {
				step := time.Now()
				cascade, err := entities.Stale(cmd.Context(), pool, debugEntityReferences(parsed))
				if err != nil {
					return err
				}
				if _, err := queue.DispatchCascade(
					cmd.Context(), queueClient, queue.Cascade{
						Characters:           cascade.Characters,
						Corporations:         cascade.Corporations,
						Alliances:            cascade.Alliances,
						CharacterHistories:   cascade.CharacterHistories,
						CorporationHistories: cascade.CorporationHistories,
					}, queue.RecentBackfill,
				); err != nil {
					return err
				}
				entitiesElapsed = time.Since(step)
			}

			ui.Section(fmt.Sprintf("Run %d/%d", run, flagDebugKMRepeat))
			ui.KV("Parse", parseElapsed.Round(time.Millisecond).String())
			debugTiming("Insert", insertElapsed, flagDebugKMSkipInsert)
			debugTiming("Stats", statsElapsed, flagDebugKMSkipStats)
			debugTiming("Achievements", achievementElapsed, flagDebugKMSkipAchievements)
			debugTiming("Entities", entitiesElapsed, flagDebugKMSkipEntities)
		}
		return nil
	},
}

func debugAchievementInput(cache *eve.Cache, parsed *killmail.Parsed) achievements.Killmail {
	km := parsed.Killmail
	out := achievements.Killmail{
		TotalValue:        km.TotalValue,
		IsNPC:             km.IsNPC,
		IsSolo:            km.IsSolo,
		VictimShipGroupID: km.VictimShipGroupID,
		VictimCharacterID: km.VictimCharacterID,
	}
	if system, ok := cache.System(km.SolarSystemID); ok {
		out.SystemSecurity = system.Security
		out.HasSecurity = true
	}
	for _, attacker := range parsed.Attackers {
		out.Attackers = append(out.Attackers, achievements.Attacker{
			CharacterID: attacker.CharacterID,
			ShipGroupID: attacker.ShipGroupID,
			FinalBlow:   attacker.FinalBlow,
		})
	}
	return out
}

func debugEntityReferences(parsed *killmail.Parsed) entities.Referenced {
	ref := entities.Referenced{KillmailTime: parsed.Killmail.KillmailTime}
	add := func(character, corporation, alliance int32) {
		if character != 0 {
			ref.Affiliations = append(ref.Affiliations, entities.Affiliation{
				CharacterID: character, CorporationID: corporation, AllianceID: alliance,
			})
		}
		ref.Corporations = append(ref.Corporations, corporation)
		ref.Alliances = append(ref.Alliances, alliance)
	}
	add(parsed.Killmail.VictimCharacterID, parsed.Killmail.VictimCorporationID,
		parsed.Killmail.VictimAllianceID)
	for _, attacker := range parsed.Attackers {
		add(attacker.CharacterID, attacker.CorporationID, attacker.AllianceID)
	}
	return ref
}

func debugTiming(label string, elapsed time.Duration, skipped bool) {
	if skipped {
		ui.KV(label, ui.Dim("skipped"))
		return
	}
	ui.KV(label, elapsed.Round(time.Millisecond).String())
}

func init() {
	debugKillmailCmd.Flags().IntVarP(&flagDebugKMRepeat, "repeat", "n", 1, "Run N times")
	debugKillmailCmd.Flags().BoolVar(&flagDebugKMSkipInsert, "skip-insert", false, "Skip killmail insert")
	debugKillmailCmd.Flags().BoolVar(&flagDebugKMSkipAchievements, "skip-achievements", false, "Skip achievements")
	debugKillmailCmd.Flags().BoolVar(&flagDebugKMSkipStats, "skip-stats", false, "Skip stats")
	debugKillmailCmd.Flags().BoolVar(&flagDebugKMSkipEntities, "skip-entities", false, "Skip entity refresh")
	debugCmd.AddCommand(debugKillmailCmd)
	rootCmd.AddCommand(debugCmd)
}
