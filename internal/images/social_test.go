package images

import (
	"bytes"
	"context"
	"image"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
)

type socialLoaderStub struct {
	value SocialKillmail
	found bool
	calls int
}

func TestKillmailSocialFetchesEntityImagesWithoutStoringThem(t *testing.T) {
	var mu sync.Mutex
	paths := []string{}
	icon := solidPNG(t, 64, 64, colorValue(40, 80, 120))
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		mu.Lock()
		paths = append(paths, r.URL.RequestURI())
		mu.Unlock()
		w.Header().Set("Content-Type", "image/png")
		_, _ = w.Write(icon)
	}))
	defer upstream.Close()
	store := newMemoryStore()
	service := New(Options{
		Store: store, UpstreamURL: upstream.URL,
		Social: &socialLoaderStub{found: true, value: SocialKillmail{
			Victim: SocialParty{
				CharacterID: 1, CorporationID: 2, AllianceID: 3,
			},
		}},
	})
	if _, err := service.KillmailSocial(context.Background(), 987654321); err != nil {
		t.Fatal(err)
	}
	for _, path := range []string{
		"/characters/1/portrait?size=64",
		"/corporations/2/logo?size=64",
		"/alliances/3/logo?size=64",
	} {
		if !containsString(paths, path) {
			t.Errorf("missing EVE image request %q: %v", path, paths)
		}
	}
	for _, key := range store.puts {
		if strings.HasPrefix(key, "entities/") {
			t.Errorf("stored a dynamic entity image at %q", key)
		}
	}
}

func containsString(values []string, target string) bool {
	for _, value := range values {
		if value == target {
			return true
		}
	}
	return false
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
		t.Fatal("social image was not persisted to image storage")
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
