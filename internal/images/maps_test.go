package images

import (
	"bytes"
	"image/png"
	"testing"
)

func TestRenderEmptySystemMap(t *testing.T) {
	body, err := encodeMap(renderSystem(nil, 128))
	if err != nil {
		t.Fatal(err)
	}
	img, err := png.Decode(bytes.NewReader(body))
	if err != nil {
		t.Fatal(err)
	}
	if img.Bounds().Dx() != 128 || img.Bounds().Dy() != 128 {
		t.Fatalf("bounds = %v", img.Bounds())
	}
}
func TestMapObjectKey(t *testing.T) {
	if got := mapObjectKey(MapConstellation, 20000001, 32); got != "static/constellations/20000001_32.png" {
		t.Fatalf("key = %q", got)
	}
}

func TestLimitMapIDs(t *testing.T) {
	ids := []int64{1, 2, 3}
	for _, test := range []struct {
		limit int
		want  int
	}{
		{limit: 0, want: 3},
		{limit: -1, want: 3},
		{limit: 1, want: 1},
		{limit: 3, want: 3},
		{limit: 4, want: 3},
	} {
		got := limitMapIDs(ids, test.limit)
		if len(got) != test.want {
			t.Fatalf("limit %d: got %d IDs, want %d", test.limit, len(got), test.want)
		}
	}
}

func TestMapImageSizes(t *testing.T) {
	for _, size := range []int{32, 64, 128, 256, 512, 1024} {
		if err := validateSize(size, true); err != nil {
			t.Fatalf("size %d: %v", size, err)
		}
	}
}

func TestPlanetTypeColorsAreStableAndDistinct(t *testing.T) {
	typeIDs := []int64{11, 12, 13, 2014, 2015, 2016, 2017, 2063, 30889, 73911}
	colors := make(map[[4]uint8]int64, len(typeIDs))
	for _, typeID := range typeIDs {
		planetColor := planetTypeColor(typeID)
		key := [4]uint8{planetColor.R, planetColor.G, planetColor.B, planetColor.A}
		if previous, exists := colors[key]; exists {
			t.Fatalf("planet types %d and %d share color %v", previous, typeID, key)
		}
		colors[key] = typeID
	}
}

func TestKnownStarTypesHaveSpectralColors(t *testing.T) {
	typeIDs := []int64{
		6, 7, 8, 9, 10,
		3796, 3797, 3798, 3799, 3800, 3801, 3802, 3803,
		34331,
		45030, 45031, 45032, 45033, 45034, 45035, 45036, 45037, 45038,
		45039, 45040, 45041, 45042, 45046, 45047,
		56082, 56083, 56084, 56085, 56086, 56097, 56098,
		73909, 78350,
	}
	_, fallback := starTypeColors(0)
	for _, typeID := range typeIDs {
		_, core := starTypeColors(typeID)
		if core == fallback {
			t.Fatalf("star type %d uses the fallback color", typeID)
		}
	}
}

func TestPastelStable(t *testing.T) {
	if got, want := pastelRGBA(30000142), pastelRGBA(30000142); got != want {
		t.Fatalf("colors differ")
	}
}
