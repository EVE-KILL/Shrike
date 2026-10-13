package frontend

import (
	"fmt"
	"math"
	"strconv"
	"strings"
	"time"

	"github.com/eve-kill/shrike/internal/api"
)

type itemMarketBar struct {
	X, Y, Width, Height string
	Title               string
}

type itemMarketTick struct {
	Y, Label string
}

type itemMarketChart struct {
	Points, Area, Start, End string
	Bars                     []itemMarketBar
	Ticks                    []itemMarketTick
}

func itemMarketChartFor(rows []api.UniverseMarketHistoryPoint) itemMarketChart {
	const width, height, left, right, top, bottom = 900.0, 205.0, 66.0, 18.0, 14.0, 34.0
	points := make([]api.UniverseMarketHistoryPoint, 0, len(rows))
	for _, row := range rows {
		if row.Average != nil {
			points = append(points, row)
		}
	}
	if len(points) == 0 {
		return itemMarketChart{}
	}
	minimum, maximum, maxVolume := *points[0].Average, *points[0].Average, int64(1)
	for _, row := range points {
		minimum = math.Min(minimum, *row.Average)
		maximum = math.Max(maximum, *row.Average)
		if row.Volume != nil && *row.Volume > maxVolume {
			maxVolume = *row.Volume
		}
	}
	padding := math.Max(math.Max((maximum-minimum)*0.12, maximum*0.015), 0.01)
	minimum = math.Max(0, minimum-padding)
	maximum += padding
	plotWidth, plotHeight := width-left-right, height-top-bottom
	barWidth := math.Max(2, math.Min(16, plotWidth/float64(len(points))*0.62))
	chart := itemMarketChart{Start: points[0].Date.Format("Jan 2, 2006"), End: points[len(points)-1].Date.Format("Jan 2, 2006")}
	coords := make([]string, 0, len(points))
	for index, row := range points {
		x := left + ((float64(index)+.5)/float64(len(points)))*plotWidth
		y := top + (maximum-*row.Average)/(maximum-minimum)*plotHeight
		coords = append(coords, fmt.Sprintf("%.2f,%.2f", x, y))
		volume := int64(0)
		if row.Volume != nil {
			volume = *row.Volume
		}
		barHeight := float64(volume) / float64(maxVolume) * plotHeight * .42
		chart.Bars = append(chart.Bars, itemMarketBar{
			X: fmt.Sprintf("%.2f", x-barWidth/2), Y: fmt.Sprintf("%.2f", height-bottom-barHeight),
			Width: fmt.Sprintf("%.2f", barWidth), Height: fmt.Sprintf("%.2f", barHeight),
			Title: row.Date.Format("Jan 2, 2006") + " — " + fullNumber(float64(volume)) + " units traded",
		})
	}
	chart.Points = strings.Join(coords, " ")
	chart.Area = coords[0][:strings.IndexByte(coords[0], ',')] + ",171 " + chart.Points + " " +
		coords[len(coords)-1][:strings.IndexByte(coords[len(coords)-1], ',')] + ",171"
	for index := range 5 {
		ratio := float64(index) / 4
		chart.Ticks = append(chart.Ticks, itemMarketTick{Y: fmt.Sprintf("%.2f", top+ratio*plotHeight), Label: marketAxisLabel(maximum - ratio*(maximum-minimum))})
	}
	return chart
}

func marketAxisLabel(value float64) string {
	abs := math.Abs(value)
	var scale float64
	var suffix string
	switch {
	case abs >= 1e12:
		scale, suffix = 1e12, "t"
	case abs >= 1e9:
		scale, suffix = 1e9, "b"
	case abs >= 1e6:
		scale, suffix = 1e6, "m"
	case abs >= 1e3:
		scale, suffix = 1e3, "k"
	default:
		return marketPrice(value)
	}
	scaled := value / scale
	precision := 0
	if math.Abs(scaled) < 10 {
		precision = 2
	} else if math.Abs(scaled) < 100 {
		precision = 1
	}
	formatted := strconv.FormatFloat(scaled, 'f', precision, 64)
	if strings.Contains(formatted, ".") {
		formatted = strings.TrimRight(strings.TrimRight(formatted, "0"), ".")
	}
	return formatted + suffix
}

func marketPrice(value float64) string {
	value = math.Round(value*100) / 100
	whole := fullNumber(math.Trunc(value))
	fraction := math.Abs(value - math.Trunc(value))
	if fraction < .005 {
		return whole
	}
	return whole + strings.TrimRight(fmt.Sprintf("%.2f", fraction)[1:], "0")
}

func marketSelectedRegion(view itemMarketView) string {
	for _, region := range view.Orders.Regions {
		if region.ID == int64(view.RegionID) {
			if region.Name != "" {
				return region.Name
			}
			return strconv.FormatInt(region.ID, 10)
		}
	}
	return "All Regions"
}

func marketSecurityOn(view itemMarketView, value string) bool {
	return strings.Contains(","+view.Security+",", ","+value+",")
}

func marketSecurityClass(value *float64) string {
	if value == nil {
		return "unknown"
	}
	if *value >= .5 {
		return "high"
	}
	if *value > 0 {
		return "low"
	}
	return "null"
}

func marketSecurityLabel(value *float64) string {
	if value == nil {
		return "?"
	}
	return fmt.Sprintf("%.1f", math.Max(0, *value))
}

func marketExpiresIn(expires time.Time) string {
	remaining := time.Until(expires)
	if remaining < 0 {
		remaining = 0
	}
	days := int(remaining.Hours()) / 24
	hours := int(remaining.Hours()) % 24
	if days > 0 {
		return fmt.Sprintf("%dd %dh", days, hours)
	}
	return fmt.Sprintf("%dh %dm", hours, int(remaining.Minutes())%60)
}

func marketDayClass(active bool) string {
	if active {
		return "active"
	}
	return ""
}

func marketSecurityButtonClass(value string, active bool) string {
	if active {
		return value + " active"
	}
	return value
}

func marketInitialSort(kind string) string {
	if kind == "buyers" {
		return "descending"
	}
	return "ascending"
}
