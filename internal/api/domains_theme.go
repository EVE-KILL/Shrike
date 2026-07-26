package api

import (
	"fmt"
	"math"
	"strconv"
	"strings"
)

const (
	domainAccentMinimumChroma = 0.04
	domainAccentMinimumLight  = 0.62
	domainAccentMaximumLight  = 0.78
)

type domainRGB struct {
	R, G, B float64
}

type domainOKLCH struct {
	L, C, H float64
}

func hasDomainThemeChoice(theme map[string]any) bool {
	return domainTruthy(theme["defaultThemePreset"]) ||
		domainTruthy(theme["defaultThemeOverrides"])
}

func domainTruthy(value any) bool {
	switch typed := value.(type) {
	case nil:
		return false
	case bool:
		return typed
	case string:
		return typed != ""
	case float64:
		return typed != 0
	default:
		return true
	}
}

func seedDomainTheme(theme, palette map[string]any) {
	accent, hover, ok := domainPaletteAccent(palette)
	if !ok {
		return
	}
	theme["defaultThemeOverrides"] = map[string]any{
		"brandPrimary":      accent,
		"brandPrimaryHover": hover,
		"brandAccent":       accent,
		"borderFocus":       accent,
		"selectionBg":       accent,
		"scrollbarThumb":    accent,
	}
}

func domainPaletteAccent(
	palette map[string]any,
) (accent string, hover string, ok bool) {
	var best *domainOKLCH
	for _, key := range []string{
		"main_color", "secondary_color", "tertiary_color",
	} {
		raw, _ := palette[key].(string)
		rgb, valid := parseDomainHex(raw)
		if !valid {
			continue
		}
		lch := domainSRGBToOKLCH(rgb)
		if lch.C >= domainAccentMinimumChroma &&
			(best == nil || lch.C > best.C) {
			copy := lch
			best = &copy
		}
	}
	if best == nil {
		return "", "", false
	}
	light := math.Min(
		math.Max(best.L, domainAccentMinimumLight),
		domainAccentMaximumLight,
	)
	return domainRGBHex(domainGamutMap(light, best.C, best.H)),
		domainRGBHex(domainGamutMap(
			math.Max(light-0.08, 0.4), best.C, best.H,
		)), true
}

func parseDomainHex(raw string) (domainRGB, bool) {
	raw = strings.TrimSpace(strings.TrimPrefix(raw, "#"))
	if len(raw) != 6 {
		return domainRGB{}, false
	}
	value, err := strconv.ParseUint(raw, 16, 24)
	if err != nil {
		return domainRGB{}, false
	}
	return domainRGB{
		R: float64((value>>16)&0xff) / 255,
		G: float64((value>>8)&0xff) / 255,
		B: float64(value&0xff) / 255,
	}, true
}

func domainLinearize(value float64) float64 {
	if value <= 0.04045 {
		return value / 12.92
	}
	return math.Pow((value+0.055)/1.055, 2.4)
}

func domainDelinearize(value float64) float64 {
	if value <= 0.0031308 {
		return value * 12.92
	}
	return 1.055*math.Pow(value, 1.0/2.4) - 0.055
}

func domainSRGBToOKLCH(rgb domainRGB) domainOKLCH {
	lr, lg, lb := domainLinearize(rgb.R),
		domainLinearize(rgb.G), domainLinearize(rgb.B)
	l := math.Cbrt(
		0.4122214708*lr + 0.5363325363*lg + 0.0514459929*lb,
	)
	m := math.Cbrt(
		0.2119034982*lr + 0.6806995451*lg + 0.1073969566*lb,
	)
	s := math.Cbrt(
		0.0883024619*lr + 0.2817188376*lg + 0.6299787005*lb,
	)
	light := 0.2104542553*l + 0.793617785*m - 0.0040720468*s
	a := 1.9779984951*l - 2.428592205*m + 0.4505937099*s
	b := 0.0259040371*l + 0.7827717662*m - 0.808675766*s
	return domainOKLCH{
		L: light, C: math.Hypot(a, b), H: math.Atan2(b, a),
	}
}

func domainOKLCHToSRGB(light, chroma, hue float64) domainRGB {
	a, b := chroma*math.Cos(hue), chroma*math.Sin(hue)
	l := math.Pow(light+0.3963377774*a+0.2158037573*b, 3)
	m := math.Pow(light-0.1055613458*a-0.0638541728*b, 3)
	s := math.Pow(light-0.0894841775*a-1.291485548*b, 3)
	return domainRGB{
		R: domainDelinearize(
			4.0767416621*l - 3.3077115913*m + 0.2309699292*s,
		),
		G: domainDelinearize(
			-1.2684380046*l + 2.6097574011*m - 0.3413193965*s,
		),
		B: domainDelinearize(
			-0.0041960863*l - 0.7034186147*m + 1.707614701*s,
		),
	}
}

func domainRGBInGamut(rgb domainRGB) bool {
	return rgb.R >= -0.0001 && rgb.R <= 1.0001 &&
		rgb.G >= -0.0001 && rgb.G <= 1.0001 &&
		rgb.B >= -0.0001 && rgb.B <= 1.0001
}

func domainGamutMap(light, chroma, hue float64) domainRGB {
	rgb := domainOKLCHToSRGB(light, chroma, hue)
	if domainRGBInGamut(rgb) {
		return rgb
	}
	low, high := 0.0, chroma
	for range 20 {
		middle := (low + high) / 2
		rgb = domainOKLCHToSRGB(light, middle, hue)
		if domainRGBInGamut(rgb) {
			low = middle
		} else {
			high = middle
		}
	}
	return domainOKLCHToSRGB(light, low, hue)
}

func domainRGBHex(rgb domainRGB) string {
	clamp := func(value float64) int {
		return int(math.Round(math.Min(1, math.Max(0, value)) * 255))
	}
	return fmt.Sprintf(
		"#%02x%02x%02x", clamp(rgb.R), clamp(rgb.G), clamp(rgb.B),
	)
}
