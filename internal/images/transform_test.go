package images

import (
	"bytes"
	"image"
	"image/color"
	"image/png"
	"testing"
)

func TestTransformImageResizesInsideAndEncodesWebP(t *testing.T) {
	source := image.NewNRGBA(image.Rect(0, 0, 64, 32))
	for y := range 32 {
		for x := range 64 {
			source.SetNRGBA(x, y, color.NRGBA{R: 220, G: 30, B: 20, A: 255})
		}
	}
	var input bytes.Buffer
	if err := png.Encode(&input, source); err != nil {
		t.Fatal(err)
	}

	body, contentType, err := transformImage(
		input.Bytes(),
		"png",
		transformSpec{Size: 32, Format: "webp"},
	)
	if err != nil {
		t.Fatal(err)
	}
	if contentType != "image/webp" {
		t.Fatalf("content type = %q", contentType)
	}
	config, format, err := image.DecodeConfig(bytes.NewReader(body))
	if err != nil {
		t.Fatal(err)
	}
	if format != "webp" || config.Width != 32 || config.Height != 16 {
		t.Fatalf("encoded image = %s %dx%d", format, config.Width, config.Height)
	}
}

func TestTransformImageOverlaysTopLeft(t *testing.T) {
	base := solidPNG(t, 32, 32, color.NRGBA{R: 20, G: 20, B: 20, A: 255})
	overlay := solidPNG(t, 8, 8, color.NRGBA{R: 255, A: 255})
	body, _, err := transformImage(
		base,
		"png",
		transformSpec{Format: "png", Overlay: overlay},
	)
	if err != nil {
		t.Fatal(err)
	}
	decoded, _, err := image.Decode(bytes.NewReader(body))
	if err != nil {
		t.Fatal(err)
	}
	r, g, b, _ := decoded.At(1, 1).RGBA()
	if r <= g || r <= b {
		t.Fatalf("overlay pixel = %d,%d,%d", r, g, b)
	}
	r, g, b, _ = decoded.At(31, 31).RGBA()
	if r != g || g != b {
		t.Fatalf("base pixel changed = %d,%d,%d", r, g, b)
	}
}

func solidPNG(
	t *testing.T,
	width, height int,
	value color.NRGBA,
) []byte {
	t.Helper()
	img := image.NewNRGBA(image.Rect(0, 0, width, height))
	for y := range height {
		for x := range width {
			img.SetNRGBA(x, y, value)
		}
	}
	var body bytes.Buffer
	if err := png.Encode(&body, img); err != nil {
		t.Fatal(err)
	}
	return body.Bytes()
}
