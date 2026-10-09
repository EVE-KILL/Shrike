package main

import (
	"context"
	"errors"
	"net/url"
	"testing"

	"github.com/eve-kill/shrike/internal/api"
	"github.com/eve-kill/shrike/internal/frontend"
)

type fittingFixtureSource struct {
	frontend.UniverseReader
	data api.UniverseFitFamilies
	err  error
}

func (s fittingFixtureSource) UniverseFitFamilies(context.Context, int64, url.Values) (api.UniverseFitFamilies, error) {
	return s.data, s.err
}

func TestFittingFixtureOnlyFillsMissingRifterFamilies(t *testing.T) {
	for _, tc := range []struct {
		name    string
		id      int64
		data    api.UniverseFitFamilies
		err     error
		fixture bool
	}{
		{"missing Rifter", 587, api.UniverseFitFamilies{ShipTypeID: 587, WindowDays: 90}, nil, true},
		{"different hull", 626, api.UniverseFitFamilies{ShipTypeID: 626}, nil, false},
		{"existing family", 587, api.UniverseFitFamilies{Families: []api.UniverseFitFamily{{FamilyHash: "existing"}}}, nil, false},
		{"read failure", 587, api.UniverseFitFamilies{}, errors.New("fixture read failed"), false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			data, err := (fixtureFittingReader{fittingFixtureSource{data: tc.data, err: tc.err}}).UniverseFitFamilies(context.Background(), tc.id, nil)
			if !errors.Is(err, tc.err) {
				t.Fatal("read error changed", err)
			}
			isFixture := len(data.Families) > 0 && data.Families[0].FamilyHash == "fixture-rifter-autocannon"
			if isFixture != tc.fixture {
				t.Fatal("unrelated read replaced", data)
			}
			if tc.fixture && (data.ShipTypeID != 587 || data.WindowDays != 90 || len(data.Families[0].Modules) != 3) {
				t.Fatal("invalid fixture family", data)
			}
		})
	}
}
