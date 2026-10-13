package main

import (
	"context"
	"net/url"
	"time"

	"github.com/eve-kill/shrike/internal/api"
	"github.com/eve-kill/shrike/internal/frontend"
)

// The existing disposable database has no observed Rifter families. Add a
// clearly named in-memory family only in the opt-in fitting comparison mode.
// Item metadata uses the guarded read-only pool with the seed's missing ship
// category supplied below; every other universe read remains unchanged.
type fixtureFittingReader struct{ frontend.UniverseReader }

func (r fixtureFittingReader) UniverseItem(ctx context.Context, id int64) (api.UniverseItemData, bool, error) {
	data, found, err := r.UniverseReader.UniverseItem(ctx, id)
	if err == nil && found && id == 587 {
		// The tiny seed has Rifter's type ID but omits its inventory category.
		data.Item.IsShip = true
		data.Item.CategoryID, data.Item.CategoryName = 6, "Ship"
		data.Item.GroupName = "Frigate"
	}
	return data, found, err
}

func (r fixtureFittingReader) UniverseFitFamilies(ctx context.Context, id int64, query url.Values) (api.UniverseFitFamilies, error) {
	data, err := r.UniverseReader.UniverseFitFamilies(ctx, id, query)
	if err != nil || id != 587 || len(data.Families) != 0 {
		return data, err
	}
	data.Families = []api.UniverseFitFamily{{
		FamilyHash: "fixture-rifter-autocannon", CanonicalFitHash: "fixture-rifter-canonical",
		TotalUses: 12, CanonicalUses: 8, VariantCount: 3,
		LastUsed: time.Date(2026, 10, 8, 12, 0, 0, 0, time.UTC), FitCost: 1250000,
		Modules: []api.UniverseFitModule{
			{SlotGroup: 1, Ordinal: 0, TypeID: 484, Name: "125mm Gatling AutoCannon I", ChargeTypeID: 178, ChargeName: "Carbonized Lead S"},
			{SlotGroup: 2, Ordinal: 0, TypeID: 439, Name: "1MN Afterburner I"},
			{SlotGroup: 3, Ordinal: 0, TypeID: 2048, Name: "Damage Control II"},
		},
		Context: &api.UniverseFitContext{MedianAttackers: 3, SecurityDistribution: []api.UniverseFitSecurityContext{{Name: "low", Pct: 75}}, TopRegion: &api.UniverseFitRegionContext{ID: 10000002, Name: "Fixture fitting region", Pct: 60}},
	}}
	return data, nil
}
