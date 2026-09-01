package cli

import (
	"fmt"
	"sort"
	"time"

	"github.com/eve-kill/shrike/internal/db"
	"github.com/eve-kill/shrike/internal/killmail"
	"github.com/eve-kill/shrike/internal/ui"
	"github.com/spf13/cobra"
)

var flagPointsSimulationDays int

type pointSimulation struct {
	ReservePercent   int                   `json:"reserve_percent"`
	Killmails        int64                 `json:"killmails"`
	Players          int64                 `json:"players"`
	PointPool        int64                 `json:"point_pool"`
	ZeroDamagePoints int64                 `json:"zero_damage_points"`
	ByFleetSize      map[string]*pointBand `json:"by_fleet_size"`
}

type pointBand struct {
	Killmails        int64 `json:"killmails"`
	Players          int64 `json:"players"`
	PointPool        int64 `json:"point_pool"`
	ZeroDamagePoints int64 `json:"zero_damage_points"`
}

var killmailPointsSimulateCmd = &cobra.Command{
	Use:   "points-simulate",
	Short: "Compare candidate attacker point allocations without writing data",
	RunE: func(cmd *cobra.Command, _ []string) error {
		if err := requireConfig(); err != nil {
			return err
		}
		pool, err := db.NewRead(cmd.Context(), cfg)
		if err != nil {
			return err
		}
		defer pool.Close()

		rows, err := pool.Query(cmd.Context(), `
			SELECT k.killmail_id, coalesce(k.points, 0),
			       coalesce(a.character_id, 0), coalesce(a.damage_done, 0),
			       coalesce(a.final_blow, false)
			FROM killmails k
			JOIN killmail_attackers a ON a.killmail_id = k.killmail_id
			WHERE k.killmail_time >= now() - make_interval(days => $1)
			ORDER BY k.killmail_id, a.attacker_index`, flagPointsSimulationDays)
		if err != nil {
			return err
		}
		defer rows.Close()

		reserves := []int64{500, 1_000, 1_500}
		results := make([]pointSimulation, len(reserves))
		for i, reserve := range reserves {
			results[i] = pointSimulation{ReservePercent: int(reserve / 100), ByFleetSize: map[string]*pointBand{}}
		}

		var currentID, currentPool int64
		var participants []killmail.PointParticipant
		flush := func() {
			if currentID == 0 {
				return
			}
			players := distinctPlayers(participants)
			if len(players) == 0 {
				return
			}
			bandName := fleetBand(len(players))
			for i, reserve := range reserves {
				shares := killmail.AllocatePoints(currentPool, reserve, participants)
				result := &results[i]
				result.Killmails++
				result.Players += int64(len(players))
				result.PointPool += currentPool
				band := result.ByFleetSize[bandName]
				if band == nil {
					band = &pointBand{}
					result.ByFleetSize[bandName] = band
				}
				band.Killmails++
				band.Players += int64(len(players))
				band.PointPool += currentPool
				for id, participant := range players {
					if participant.DamageDone == 0 {
						result.ZeroDamagePoints += shares[id]
						band.ZeroDamagePoints += shares[id]
					}
				}
			}
		}

		for rows.Next() {
			var killmailID, poolPoints int64
			var participant killmail.PointParticipant
			if err := rows.Scan(&killmailID, &poolPoints, &participant.CharacterID, &participant.DamageDone, &participant.FinalBlow); err != nil {
				return err
			}
			if currentID != 0 && killmailID != currentID {
				flush()
				participants = participants[:0]
			}
			currentID, currentPool = killmailID, poolPoints
			participants = append(participants, participant)
		}
		if err := rows.Err(); err != nil {
			return err
		}
		flush()

		if flagJSON {
			return ui.JSON(results)
		}
		for _, result := range results {
			ui.Section(fmt.Sprintf("%d%% participation reserve", result.ReservePercent))
			ui.KV("Killmails", fmt.Sprintf("%d", result.Killmails))
			ui.KV("Players", fmt.Sprintf("%d", result.Players))
			ui.KV("Point pool", fmt.Sprintf("%d", result.PointPool))
			pct := float64(0)
			if result.PointPool > 0 {
				pct = 100 * float64(result.ZeroDamagePoints) / float64(result.PointPool)
			}
			ui.KV("Zero-damage share", fmt.Sprintf("%d (%.2f%%)", result.ZeroDamagePoints, pct))
			bands := make([]string, 0, len(result.ByFleetSize))
			for band := range result.ByFleetSize {
				bands = append(bands, band)
			}
			sort.Strings(bands)
			for _, bandName := range bands {
				band := result.ByFleetSize[bandName]
				ui.KV("Fleet "+bandName, fmt.Sprintf("%d mails, %d points to zero damage", band.Killmails, band.ZeroDamagePoints))
			}
		}
		ui.KV("Window", fmt.Sprintf("%s to now", time.Now().UTC().AddDate(0, 0, -flagPointsSimulationDays).Format("2006-01-02")))
		return nil
	},
}

func distinctPlayers(participants []killmail.PointParticipant) map[int32]killmail.PointParticipant {
	players := make(map[int32]killmail.PointParticipant)
	for _, participant := range participants {
		if participant.CharacterID == 0 {
			continue
		}
		merged := players[participant.CharacterID]
		merged.CharacterID = participant.CharacterID
		if participant.DamageDone > 0 {
			merged.DamageDone += participant.DamageDone
		}
		merged.FinalBlow = merged.FinalBlow || participant.FinalBlow
		players[participant.CharacterID] = merged
	}
	return players
}

func fleetBand(players int) string {
	switch {
	case players == 1:
		return "01 solo"
	case players <= 5:
		return "02 2-5"
	case players <= 20:
		return "03 6-20"
	case players <= 100:
		return "04 21-100"
	default:
		return "05 101+"
	}
}

func init() {
	killmailPointsSimulateCmd.Flags().IntVar(&flagPointsSimulationDays, "days", 7, "Recent UTC days to sample")
	killmailCmd.AddCommand(killmailPointsSimulateCmd)
}
