package frontend

import (
	"context"
	"fmt"

	"github.com/eve-kill/shrike/internal/api"
	"golang.org/x/sync/errgroup"
)

type topBoxView struct {
	Title       string
	DataType    string
	Limit       int
	Entries     []api.TopEntry
	ShowFooter  bool
	Pending     bool
	CountFormat string
	CountClass  string
}

type homeView struct {
	Valuable []api.ValuableKill
	Killlist api.KilllistPage
	TopBoxes []topBoxView
}

var homeTopBoxes = []topBoxView{
	{Title: "Top Characters", DataType: "characters", Limit: 10},
	{Title: "Top Corporations", DataType: "corporations", Limit: 10},
	{Title: "Top Alliances", DataType: "alliances", Limit: 10},
	{Title: "Top Factions", DataType: "factions", Limit: 6},
	{Title: "Top Ships", DataType: "ships", Limit: 10},
	{Title: "Top Systems", DataType: "systems", Limit: 10},
	{Title: "Top Regions", DataType: "regions", Limit: 10},
}

func loadHomepage(ctx context.Context, reader HomepageReader) (homeView, error) {
	view := homeView{TopBoxes: append([]topBoxView(nil), homeTopBoxes...)}
	group, ctx := errgroup.WithContext(ctx)
	group.SetLimit(4)
	group.Go(func() error {
		var err error
		view.Valuable, err = reader.MostValuable(ctx, "kills", 7, 8)
		if err != nil {
			return fmt.Errorf("most valuable: %w", err)
		}
		return nil
	})
	group.Go(func() error {
		var err error
		view.Killlist, err = reader.LatestKills(ctx, 1, 50)
		if err != nil {
			return fmt.Errorf("latest kills: %w", err)
		}
		return nil
	})
	for index := range view.TopBoxes {
		group.Go(func() error {
			var err error
			box := &view.TopBoxes[index]
			box.Entries, err = reader.Top(ctx, box.DataType, 7, box.Limit)
			if err != nil {
				return fmt.Errorf("%s: %w", box.DataType, err)
			}
			return nil
		})
	}
	if err := group.Wait(); err != nil {
		return homeView{}, err
	}
	return view, nil
}
