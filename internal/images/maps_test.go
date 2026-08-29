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
func TestPastelStable(t *testing.T) {
	if got, want := pastelRGBA(30000142), pastelRGBA(30000142); got != want {
		t.Fatalf("colors differ")
	}
}
