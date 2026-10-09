package frontend

import (
	"fmt"
	"math"
	"strconv"
	"strings"

	"github.com/eve-kill/shrike/internal/api"
)

// paletteAccent matches the Nuxt entityAccent color selection. In-game logo
// colors are often too dark to read as a three-pixel strip on this background.
func paletteAccent(palette *api.TopPalette) string {
	if palette == nil {
		return ""
	}
	var best oklch
	for _, color := range []string{palette.MainColor, palette.SecondaryColor, palette.TertiaryColor} {
		rgb, ok := parsePaletteColor(color)
		if !ok {
			continue
		}
		candidate := rgbToOKLCH(rgb)
		if candidate.c >= 0.04 && candidate.c > best.c {
			best = candidate
		}
	}
	if best.c == 0 {
		return ""
	}
	best.l = math.Min(math.Max(best.l, 0.62), 0.78)
	rgb := gamutMapAccent(best)
	return fmt.Sprintf("#%02x%02x%02x", accentByte(rgb[0]), accentByte(rgb[1]), accentByte(rgb[2]))
}

func entityPaletteSwatches(palette *api.TopPalette) []string {
	if palette == nil {
		return nil
	}
	var colors []string
	for _, color := range []string{palette.MainColor, palette.SecondaryColor, palette.TertiaryColor} {
		rgb, ok := parsePaletteColor(color)
		if ok {
			colors = append(colors, fmt.Sprintf("#%02x%02x%02x", accentByte(rgb[0]), accentByte(rgb[1]), accentByte(rgb[2])))
		}
	}
	return colors
}

type oklch struct{ l, c, h float64 }
type accentRGB [3]float64

func parsePaletteColor(color string) (accentRGB, bool) {
	color = strings.TrimSpace(strings.TrimPrefix(strings.TrimSpace(color), "#"))
	if len(color) != 6 {
		return accentRGB{}, false
	}
	value, err := strconv.ParseUint(color, 16, 24)
	if err != nil {
		return accentRGB{}, false
	}
	return accentRGB{float64((value>>16)&255) / 255, float64((value>>8)&255) / 255, float64(value&255) / 255}, true
}

func linearAccent(value float64) float64 {
	if value <= 0.04045 {
		return value / 12.92
	}
	return math.Pow((value+0.055)/1.055, 2.4)
}

func delinearAccent(value float64) float64 {
	if value <= 0.0031308 {
		return value * 12.92
	}
	return 1.055*math.Pow(value, 1/2.4) - 0.055
}

func rgbToOKLCH(rgb accentRGB) oklch {
	r, g, b := linearAccent(rgb[0]), linearAccent(rgb[1]), linearAccent(rgb[2])
	l := math.Cbrt(0.4122214708*r + 0.5363325363*g + 0.0514459929*b)
	m := math.Cbrt(0.2119034982*r + 0.6806995451*g + 0.1073969566*b)
	s := math.Cbrt(0.0883024619*r + 0.2817188376*g + 0.6299787005*b)
	light := 0.2104542553*l + 0.793617785*m - 0.0040720468*s
	a := 1.9779984951*l - 2.428592205*m + 0.4505937099*s
	blue := 0.0259040371*l + 0.7827717662*m - 0.808675766*s
	return oklch{light, math.Hypot(a, blue), math.Atan2(blue, a)}
}

func oklchToRGB(color oklch) accentRGB {
	a := color.c * math.Cos(color.h)
	b := color.c * math.Sin(color.h)
	l := color.l + 0.3963377774*a + 0.2158037573*b
	m := color.l - 0.1055613458*a - 0.0638541728*b
	s := color.l - 0.0894841775*a - 1.2914855480*b
	l, m, s = l*l*l, m*m*m, s*s*s
	return accentRGB{
		delinearAccent(4.0767416621*l - 3.3077115913*m + 0.2309699292*s),
		delinearAccent(-1.2684380046*l + 2.6097574011*m - 0.3413193965*s),
		delinearAccent(-0.0041960863*l - 0.7034186147*m + 1.7076147010*s),
	}
}

func gamutMapAccent(color oklch) accentRGB {
	rgb := oklchToRGB(color)
	if accentInGamut(rgb) {
		return rgb
	}
	low, high := 0.0, color.c
	for range 20 {
		mid := (low + high) / 2
		candidate := oklchToRGB(oklch{color.l, mid, color.h})
		if accentInGamut(candidate) {
			low = mid
		} else {
			high = mid
		}
	}
	return oklchToRGB(oklch{color.l, low, color.h})
}

func accentInGamut(rgb accentRGB) bool {
	for _, value := range rgb {
		if value < -0.0001 || value > 1.0001 {
			return false
		}
	}
	return true
}

func accentByte(value float64) uint8 {
	return uint8(math.Round(math.Min(1, math.Max(0, value)) * 255))
}
