package images

import "testing"

func TestAcceptsWebPRespectsQuality(t *testing.T) {
	tests := []struct {
		header string
		want   bool
	}{
		{"image/avif,image/webp,image/png", true},
		{"image/webp; q=0.8, image/png", true},
		{"image/webp;q=0, image/png", false},
		{"image/webp;q=0, image/webp;q=0.5", true},
		{"image/png,*/*;q=0.8", false},
		{"", false},
	}
	for _, test := range tests {
		if got := acceptsWebP(test.header); got != test.want {
			t.Errorf("acceptsWebP(%q) = %t, want %t", test.header, got, test.want)
		}
	}
}
