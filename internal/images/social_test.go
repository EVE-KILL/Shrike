package images

import (
	"bytes"
	"context"
	"image"
	"testing"
)

type socialLoaderStub struct {
	value SocialKillmail
	found bool
	calls int
}

func (s *socialLoaderStub) LoadKillmailSocial(
	context.Context,
	int64,
) (SocialKillmail, bool, error) {
	s.calls++
	return s.value, s.found, nil
}

func TestKillmailSocialRendersAndPersistsImmutablePNG(t *testing.T) {
	store := newMemoryStore()
	loader := &socialLoaderStub{
		found: true,
		value: SocialKillmail{
			TotalValue:      12_345_678_901,
			SolarSystemName: "Jita",
			RegionName:      "The Forge",
			Victim: SocialParty{
				CharacterName:   "Victim Pilot",
				CorporationName: "Victim Corporation",
				AllianceName:    "Victim Alliance",
				ShipName:        "Revelation Navy Issue",
			},
			FinalBlow: &SocialParty{
				CharacterName:   "Final Blow Pilot",
				CorporationName: "Killer Corporation",
				AllianceName:    "Killer Alliance",
				ShipName:        "Ibis",
			},
		},
	}
	service := New(Options{
		Store: store, Social: loader, CacheBytes: 8 << 20,
	})
	first, err := service.KillmailSocial(context.Background(), 123456789)
	if err != nil {
		t.Fatal(err)
	}
	if first.ContentType != "image/png" ||
		first.CacheControl != immutableCacheControl {
		t.Fatalf("social response = %+v", first)
	}
	config, format, err := image.DecodeConfig(bytes.NewReader(first.Body))
	if err != nil {
		t.Fatal(err)
	}
	if format != "png" || config.Width != socialWidth ||
		config.Height != socialHeight {
		t.Fatalf("social image = %s %dx%d", format, config.Width, config.Height)
	}
	if store.objects["social/killmails/v1/123456789.png"] == nil {
		t.Fatal("social image was not persisted to B2")
	}
	second, err := service.KillmailSocial(context.Background(), 123456789)
	if err != nil {
		t.Fatal(err)
	}
	if loader.calls != 1 || first.ETag != second.ETag {
		t.Fatalf(
			"loader calls = %d, ETags = %q/%q",
			loader.calls,
			first.ETag,
			second.ETag,
		)
	}
}
