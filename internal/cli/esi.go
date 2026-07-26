package cli

import (
	"context"
	"encoding/json"
	"fmt"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/eve-kill/shrike/internal/db"
	"github.com/eve-kill/shrike/internal/entities"
	"github.com/eve-kill/shrike/internal/esi"
	"github.com/eve-kill/shrike/internal/ui"
	"github.com/spf13/cobra"
)

var esiCmd = &cobra.Command{
	Use:   "esi",
	Short: "Talk to CCP's ESI and refresh entities from it",
	Long: `ESI polices two budgets, and the client respects both.

The rate limit is per endpoint family and is tracked in a Redis token bucket
shared by every worker, so the deployment behaves as one client. The error limit
is global: exceed it and every endpoint returns 420 for a minute, whichever one
you were abusing — so a 420 pauses the whole client, not just the group that
earned it.`,
}

var (
	flagESICascade bool
	flagESIForce   bool
	flagESIRaw     bool
)

var esiStatusCmd = &cobra.Command{
	Use:   "status",
	Short: "Show rate-limit budgets, the pause flag, and the error budget",
	Long: `Reads the live coordination state out of Redis.

A group showing "unseeded" has not been used since its window last expired,
which reads as a full budget rather than an empty one. PAUSED means someone in
the cluster took a 420 and every request is on hold until it clears.`,
	RunE: func(cmd *cobra.Command, _ []string) error {
		if err := requireConfig(); err != nil {
			return err
		}
		client := esi.New(cfg)
		defer client.Close()

		names := make([]string, 0, len(esi.Groups))
		for name := range esi.Groups {
			names = append(names, name)
		}
		sort.Strings(names)

		states := make([]esi.BucketState, 0, len(names)+1)
		for _, name := range append(names, esi.Probation.Name) {
			g, ok := esi.Groups[name]
			if !ok {
				g = esi.Probation
			}
			state, err := client.Limiter().Peek(cmd.Context(), g)
			if err != nil {
				return err
			}
			states = append(states, state)
		}

		if ui.JSONMode {
			return ui.JSON(states)
		}

		ui.Section("ESI rate limits")
		t := ui.NewTable("GROUP", "REMAINING", "LIMIT", "WINDOW", "RESETS IN")
		for _, s := range states {
			g, ok := esi.Groups[s.Group]
			if !ok {
				g = esi.Probation
			}
			resets := ui.Dim("unseeded")
			remaining := ui.Dim(fmtCount(int64(s.Remaining)))
			if s.Seeded {
				remaining = fmtCount(int64(s.Remaining))
				if d := time.Until(s.ResetAt); d > 0 {
					resets = d.Round(time.Second).String()
				} else {
					resets = ui.Dim("expired")
				}
			}
			flags := ""
			if g.Sequential {
				flags = ui.Dim(" seq")
			}
			t.Row(s.Group+flags, remaining, fmtCount(int64(s.Limit)),
				fmt.Sprintf("%ds", g.Window), resets)
		}
		fmt.Println(t.Render())
		ui.Newline()
		return nil
	},
}

var esiGroupsCmd = &cobra.Command{
	Use:   "groups [path]",
	Short: "Show the endpoint registry, or which group a path falls into",
	Long: `Each endpoint family has a preset budget and dispatch rules.

Where ESI emits x-ratelimit-* headers the learned values win; where it does not,
the preset here is the only ground truth and was hand-tuned against observed
420s. Those families are also marked sequential: they return 420 under a
concurrent burst even while the bucket still has capacity.

Anything unrecognised lands in "probation" — slow, serialised, header-ignoring —
because an unmeasured endpoint gets the treatment that cannot cause harm.

    shrike esi:groups
    shrike esi:groups /latest/characters/12345/corporationhistory/`,
	Args: cobra.MaximumNArgs(1),
	RunE: func(_ *cobra.Command, args []string) error {
		if len(args) == 1 {
			g := esi.ResolveGroup(args[0])
			if ui.JSONMode {
				return ui.JSON(map[string]any{
					"path":   args[0],
					"prefix": esi.PathPrefix(args[0]),
					"group":  g,
				})
			}
			ui.Section("Classification")
			ui.KV("Path", args[0])
			ui.KV("Normalised", esi.PathPrefix(args[0]))
			ui.KV("Group", g.Name)
			ui.KV("Budget", fmt.Sprintf("%d per %ds", g.Limit, g.Window))
			ui.KV("Headers authoritative", yesNo(g.HeaderAuthoritative))
			ui.KV("Sequential", yesNo(g.Sequential))
			ui.Newline()
			return nil
		}

		names := make([]string, 0, len(esi.Groups))
		for name := range esi.Groups {
			names = append(names, name)
		}
		sort.Strings(names)

		if ui.JSONMode {
			all := make([]esi.Group, 0, len(names)+1)
			for _, n := range names {
				all = append(all, esi.Groups[n])
			}
			return ui.JSON(append(all, esi.Probation))
		}

		ui.Section("ESI endpoint groups")
		t := ui.NewTable("GROUP", "BUDGET", "HEADERS", "DISPATCH")
		for _, name := range names {
			g := esi.Groups[name]
			headers := ui.Dim("preset only")
			if g.HeaderAuthoritative {
				headers = "authoritative"
			}
			dispatch := "concurrent"
			if g.Sequential {
				dispatch = ui.Warn2("sequential")
			}
			t.Row(g.Name, fmt.Sprintf("%d / %ds", g.Limit, g.Window), headers, dispatch)
		}
		g := esi.Probation
		t.Row(ui.Dim(g.Name), fmt.Sprintf("%d / %ds", g.Limit, g.Window),
			ui.Dim("preset only"), ui.Warn2("sequential"))
		fmt.Println(t.Render())
		ui.Newline()
		return nil
	},
}

var esiGetCmd = &cobra.Command{
	Use:   "get <path>",
	Short: "Fetch any ESI path through the full pipeline",
	Long: `Runs a request through cache, rate limit, singleflight and all.

Useful for seeing what an endpoint actually returns, and for confirming the
cache works: the second call to the same path answers instantly and reports
cached=true without touching ESI.

    shrike esi:get /latest/characters/2124561066/
    shrike esi:get /latest/alliances/ --raw`,
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		if err := requireConfig(); err != nil {
			return err
		}
		client := esi.New(cfg)
		defer client.Close()

		start := time.Now()
		res, err := esi.Get[json.RawMessage](cmd.Context(), client, args[0])
		if err != nil {
			return err
		}
		elapsed := time.Since(start).Round(time.Millisecond)

		if flagESIRaw || ui.JSONMode {
			if res.Data == nil {
				return ui.JSON(map[string]any{"status": res.Status, "group": res.Group})
			}
			fmt.Println(string(*res.Data))
			return nil
		}

		ui.Section(fmt.Sprintf("GET %s", args[0]))
		ui.KV("Status", strconv.Itoa(res.Status))
		ui.KV("Group", res.Group)
		ui.KV("Elapsed", elapsed.String())
		ui.KV("From cache", yesNo(res.Cached))
		if res.Pages > 1 {
			ui.KV("Pages", strconv.Itoa(res.Pages))
		}
		if res.RetryAfter > 0 {
			ui.KV("Retry after", fmt.Sprintf("%ds", res.RetryAfter))
		}
		ui.Newline()
		if res.Data != nil {
			var pretty any
			if err := json.Unmarshal(*res.Data, &pretty); err == nil {
				out, _ := json.MarshalIndent(pretty, "", "  ")
				fmt.Println(string(out))
			}
		}
		ui.Newline()
		return nil
	},
}

var esiRefreshCmd = &cobra.Command{
	Use:   "refresh <character|corporation|alliance> <id>...",
	Short: "Fetch entities from ESI and store them",
	Long: `Fetches one or more entities and writes them to the database.

A 404 records the entity as deleted rather than leaving it absent, so a
biomassed character is not retried forever — a killboard accumulates references
to dead characters indefinitely.

--cascade follows what each refresh implies: a character pulls in its
corporation, a corporation its alliance, and both pull their history when the
affiliation has moved since it was last synced. Without it the cascade is only
reported.

    shrike esi:refresh character 2124561066
    shrike esi:refresh character 2124561066 --cascade
    shrike esi:refresh corporation 98187159 --cascade`,
	Args: cobra.MinimumNArgs(2),
	RunE: func(cmd *cobra.Command, args []string) error {
		if err := requireConfig(); err != nil {
			return err
		}

		kind := strings.TrimSuffix(strings.ToLower(args[0]), "s")
		switch kind {
		case "character", "corporation", "alliance":
		default:
			return fmt.Errorf("unknown entity kind %q: expected character, corporation or alliance", args[0])
		}

		ids := make([]int32, 0, len(args)-1)
		for _, raw := range args[1:] {
			id, err := strconv.ParseInt(raw, 10, 32)
			if err != nil || id <= 0 {
				return fmt.Errorf("invalid id %q", raw)
			}
			ids = append(ids, int32(id))
		}

		pool, err := db.New(cmd.Context(), cfg)
		if err != nil {
			return err
		}
		defer pool.Close()

		client := esi.New(cfg)
		defer client.Close()

		r := &entities.Refresher{Pool: pool, ESI: client}
		results, err := runRefresh(cmd.Context(), r, kind, ids, flagESICascade, flagESIForce)

		if ui.JSONMode {
			if jsonErr := ui.JSON(results); jsonErr != nil {
				return jsonErr
			}
			return err
		}

		ui.Section("Entity refresh")
		t := ui.NewTable("KIND", "ID", "NAME", "STATUS", "ROWS")
		for _, res := range results {
			name := res.Name
			switch {
			case res.Deleted:
				name = ui.Warn2("deleted")
			case name == "":
				name = ui.Dim("—")
			}
			status := strconv.Itoa(res.Status)
			if res.Status == 304 {
				status = ui.Dim("unchanged")
			}
			t.Row(res.Kind, strconv.Itoa(int(res.ID)), name, status, fmtCount(res.Rows))
		}
		fmt.Println(t.Render())

		if !flagESICascade {
			pending := map[string][]int32{}
			for _, res := range results {
				collectCascade(pending, res.Cascade)
			}
			if len(pending) > 0 {
				ui.Newline()
				ui.KV("Pending cascade", ui.Dim("re-run with --cascade to follow"))
				for _, kind := range sortedKinds(pending) {
					ui.KV("  "+kind, joinIDs(pending[kind]))
				}
			}
		}
		ui.Newline()
		return err
	},
}

// runRefresh walks the entity graph breadth-first.
//
// A visited set is what keeps it terminating: corporations point at alliances
// which point back at corporations, and a naive walk would loop forever on any
// alliance whose executor is one of its own members.
func runRefresh(ctx context.Context, r *entities.Refresher, kind string, ids []int32, cascade, force bool) ([]entities.Result, error) {
	type item struct {
		kind string
		id   int32
	}

	queue := make([]item, 0, len(ids))
	for _, id := range ids {
		queue = append(queue, item{kind, id})
	}

	visited := map[item]bool{}
	var results []entities.Result

	for len(queue) > 0 {
		next := queue[0]
		queue = queue[1:]
		if visited[next] {
			continue
		}
		visited[next] = true

		var res entities.Result
		var err error
		switch next.kind {
		case "character":
			res, err = r.Character(ctx, next.id)
		case "corporation":
			res, err = r.Corporation(ctx, next.id)
		case "alliance":
			res, err = r.Alliance(ctx, next.id)
		case "character_history":
			res, err = r.CharacterHistory(ctx, next.id, force)
		case "corporation_history":
			res, err = r.CorporationHistory(ctx, next.id, force)
		}
		if err != nil {
			return results, err
		}
		results = append(results, res)

		if !cascade {
			continue
		}
		for _, id := range res.Cascade.Characters {
			queue = append(queue, item{"character", id})
		}
		for _, id := range res.Cascade.Corporations {
			queue = append(queue, item{"corporation", id})
		}
		for _, id := range res.Cascade.Alliances {
			queue = append(queue, item{"alliance", id})
		}
		for _, id := range res.Cascade.CharacterHistories {
			queue = append(queue, item{"character_history", id})
		}
		for _, id := range res.Cascade.CorporationHistories {
			queue = append(queue, item{"corporation_history", id})
		}
	}

	return results, nil
}

func collectCascade(into map[string][]int32, c entities.Cascade) {
	for kind, ids := range map[string][]int32{
		"characters":          c.Characters,
		"corporations":        c.Corporations,
		"alliances":           c.Alliances,
		"character history":   c.CharacterHistories,
		"corporation history": c.CorporationHistories,
	} {
		if len(ids) > 0 {
			into[kind] = append(into[kind], ids...)
		}
	}
}

func sortedKinds(m map[string][]int32) []string {
	out := make([]string, 0, len(m))
	for k := range m {
		out = append(out, k)
	}
	sort.Strings(out)
	return out
}

func joinIDs(ids []int32) string {
	parts := make([]string, 0, len(ids))
	for _, id := range ids {
		parts = append(parts, strconv.Itoa(int(id)))
	}
	return strings.Join(parts, ", ")
}

func init() {
	esiRefreshCmd.Flags().BoolVar(&flagESICascade, "cascade", false, "Follow the entities each refresh implies")
	esiRefreshCmd.Flags().BoolVar(&flagESIForce, "force", false, "Refetch history even when the sync marker says it is current")
	esiGetCmd.Flags().BoolVar(&flagESIRaw, "raw", false, "Print the response body only")

	esiCmd.AddCommand(esiStatusCmd, esiGroupsCmd, esiGetCmd, esiRefreshCmd)
}
