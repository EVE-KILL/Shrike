package frontend

import (
	"context"
	"fmt"
	"slices"
	"strings"

	"github.com/eve-kill/shrike/internal/api"
	"github.com/eve-kill/shrike/internal/killtype"
	"golang.org/x/sync/errgroup"
)

type killsPageView struct {
	Kind, Title string
	Valuable    []api.ValuableKill
	Killlist    api.KilllistPage
	TopBoxes    []topBoxView
}

var killsPageTitles = map[string]string{
	"latest": "Latest Kills", "highsec": "High Sec Kills", "lowsec": "Low Sec Kills",
	"nullsec": "Null Sec Kills", "wspace": "Wormhole Kills", "abyssal": "Abyssal Kills",
	"pochven": "Pochven Kills", "jove": "Jove Space Kills",
	"timezone-au": "AU Timezone Kills", "timezone-ru": "RU Timezone Kills",
	"timezone-eu": "EU Timezone Kills", "timezone-us-east": "US East Timezone Kills",
	"timezone-us-west": "US West Timezone Kills",
	"big":              "Big Kills", "solo": "Solo Kills", "npc": "NPC Kills",
	"attackers-1": "1-Attacker Non-solo Kills", "attackers-2-4": "2–4 Attacker Kills",
	"attackers-5-9": "5–9 Attacker Kills", "attackers-10-24": "10–24 Attacker Kills",
	"attackers-25-49": "25–49 Attacker Kills", "attackers-50-99": "50–99 Attacker Kills",
	"attackers-100-999": "100–999 Attacker Kills", "attackers-1000-plus": "1,000+ Attacker Kills",
	"pvp": "PvP Kills", "ganked": "Highsec Ganks", "awox": "Awox Kills",
	"capital-involved": "Capital-Involved Kills", "supercarrier-involved": "Supercarrier-Involved Kills",
	"titan-involved": "Titan-Involved Kills", "at-ship-involved": "Alliance Tournament Ship-Involved Kills",
	"fw-caldari-winner": "Caldari FW Victories", "fw-gallente-winner": "Gallente FW Victories",
	"fw-amarr-winner": "Amarr FW Victories", "fw-minmatar-winner": "Minmatar FW Victories",
	"fw-caldari-gallente": "Caldari–Gallente FW Kills", "fw-amarr-minmatar": "Amarr–Minmatar FW Kills",
	"5b": "5B+ Kills", "10b": "10B+ Kills", "under-1b": "Under 1B ISK Kills",
	"1b-5b": "1B–5B ISK Kills", "5b-10b": "5B–10B ISK Kills",
	"10b-100b": "10B–100B ISK Kills", "100b-1t": "100B–1T ISK Kills",
	"1t-plus":             "1T+ ISK Kills",
	"category-deployable": "Deployable Kills", "category-drone": "Drone Kills",
	"category-fighter": "Fighter Kills", "category-orbital": "Orbital Kills",
	"category-starbase": "Starbase Kills", "category-ship": "Ship Kills",
	"category-sovereignty": "Sovereignty Structure Kills",
	"category-structure":   "Structure Category Kills", "category-infantry": "Infantry Kills",
	"frigates": "Frigate Kills", "destroyers": "Destroyer Kills", "cruisers": "Cruiser Kills",
	"battlecruisers": "Battlecruiser Kills", "battleships": "Battleship Kills",
	"capitals": "Capital Kills", "freighters": "Freighter Kills",
	"supercarriers": "Supercarrier Kills", "titans": "Titan Kills",
	"citadels": "Structure Kills", "t1": "T1 Kills", "t2": "T2 Kills",
	"t3": "T3 Kills", "faction": "Faction Kills",
}

func validKillsPageType(kind string) bool { return slices.Contains(killtype.Types, kind) }

func killsPageTitle(kind string) string {
	if title := killsPageTitles[kind]; title != "" {
		return title
	}
	return "Kills"
}

func topKillsTitle(kind, dataType string) string {
	prefix := "Top"
	if kind == "npc" {
		prefix = "Top Victim"
	}
	switch dataType {
	case "characters":
		return prefix + " Characters"
	case "corporations":
		return prefix + " Corporations"
	case "alliances":
		return prefix + " Alliances"
	default:
		return ""
	}
}

func killsPageMetadata(kind string) Metadata {
	title := killsPageTitle(kind)
	return Metadata{
		Title: title, CanonicalPath: "/kills/" + kind, Indexable: true,
		Description: fmt.Sprintf("Browse %s in EVE Online — a live killmail feed with ship fits, ISK values, and combat details on EVE-KILL.", strings.ToLower(title)),
		StructuredData: map[string]any{
			"@context": "https://schema.org", "@type": "CollectionPage", "name": title,
			"url": canonicalOrigin + "/kills/" + kind,
		},
	}
}

func loadKillsPage(ctx context.Context, reader HomepageReader, kind string) (killsPageView, error) {
	view := killsPageView{
		Kind: kind, Title: killsPageTitle(kind),
		TopBoxes: []topBoxView{
			{Title: topKillsTitle(kind, "characters"), DataType: "characters", Limit: 10, ShowFooter: true},
			{Title: topKillsTitle(kind, "corporations"), DataType: "corporations", Limit: 10, ShowFooter: true},
			{Title: topKillsTitle(kind, "alliances"), DataType: "alliances", Limit: 10, ShowFooter: true},
		},
	}
	group, ctx := errgroup.WithContext(ctx)
	group.SetLimit(4)
	group.Go(func() error {
		var err error
		view.Valuable, err = reader.MostValuableKills(ctx, kind, 7, 8)
		if err != nil {
			return fmt.Errorf("most valuable %s: %w", kind, err)
		}
		return nil
	})
	group.Go(func() error {
		var err error
		view.Killlist, err = reader.Killlist(ctx, kind, 1, 50)
		if err != nil {
			return fmt.Errorf("%s killlist: %w", kind, err)
		}
		return nil
	})
	for index := range view.TopBoxes {
		group.Go(func() error {
			box := &view.TopBoxes[index]
			var cached bool
			box.Entries, cached = reader.TopKillsCached(ctx, kind, box.DataType, 7, box.Limit)
			box.Pending = !cached
			return nil
		})
	}
	if err := group.Wait(); err != nil {
		return killsPageView{}, err
	}
	return view, nil
}
