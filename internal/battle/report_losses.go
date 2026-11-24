package battle

import (
	"fmt"
	"math"
	"sort"
	"time"
)

type ReportSide struct {
	Corps     []int64 `json:"corps"`
	Alliances []int64 `json:"alliances"`
}
type ReportLoss struct {
	ID         int64    `json:"killmail_id"`
	Time       string   `json:"killmail_time"`
	SystemID   int64    `json:"solar_system_id"`
	SystemName string   `json:"solar_system_name"`
	X          *float64 `json:"position_x"`
	Y          *float64 `json:"position_y"`
	Z          *float64 `json:"position_z"`
	VictimName string   `json:"victim_character_name"`
	CorpID     int64    `json:"victim_corporation_id"`
	AllianceID int64    `json:"victim_alliance_id"`
	ShipName   string   `json:"ship_name"`
	ShipID     int64    `json:"ship_type_id"`
	GroupID    int64    `json:"ship_group_id"`
	GroupName  string   `json:"ship_group_name"`
	Value      float64  `json:"total_value"`
}
type LossFilter struct {
	Side   *int64
	Group  *int64
	MinISK float64
	From   *int64
	To     *int64
}
type LossTotal struct {
	Count int
	ISK   float64
}
type LossBin struct {
	Start, End int64
	Count      int
	ISK        float64
	Big        int
	Sides      []LossTotal
}
type LossShipGroup struct {
	ID   int64
	Name string
}

// LossSummary is the compact loss explorer data for a generated report.
// The replay and filtered lists retain their individual events at runtime.
type LossSummary struct {
	Filtered           []ReportLoss `json:"-"`
	Biggest            []ReportLoss
	Supers             []ReportLoss
	Groups             []LossShipGroup
	Bins               []LossBin
	PeakCount, PeakISK float64
	Busiest            LossBin
	LossISK            float64
	LossCount          int
}

func VictimSide(k ReportLoss, sides []ReportSide) int {
	for i, side := range sides {
		for _, id := range side.Corps {
			if id > 0 && id == k.CorpID {
				return i
			}
		}
	}
	for i, side := range sides {
		for _, id := range side.Alliances {
			if id > 0 && id == k.AllianceID {
				return i
			}
		}
	}
	return -1
}
func reportTimeMS(raw string) int64 {
	t, e := time.Parse(time.RFC3339Nano, raw)
	if e != nil {
		return 0
	}
	return t.UnixMilli()
}
func SummarizeLosses(startTime, endTime string, sides []ReportSide, kills []ReportLoss, filter LossFilter) LossSummary {
	var v LossSummary
	start, end := reportTimeMS(startTime), reportTimeMS(endTime)
	step := int64(180 * 60000)
	for _, minutes := range []int64{1, 5, 15, 30, 60, 180} {
		if float64(end-start)/60000/float64(minutes) <= 60 {
			step = minutes * 60000
			break
		}
	}
	count := max(1, int(math.Ceil(float64(end-start)/float64(step))))
	v.Bins = make([]LossBin, count)
	for i := range v.Bins {
		binEnd := start + int64(i+1)*step
		if i == count-1 {
			binEnd = end + 1
		}
		v.Bins[i] = LossBin{Start: start + int64(i)*step, End: binEnd, Sides: make([]LossTotal, len(sides)+1)}
	}
	groups := map[int64]string{}
	for _, k := range kills {
		if k.GroupID > 0 {
			name := k.GroupName
			if name == "" {
				name = fmt.Sprintf("Class %d", k.GroupID)
			}
			groups[k.GroupID] = name
		}
		t := reportTimeMS(k.Time)
		side := VictimSide(k, sides)
		f := filter
		if t == 0 || f.Side != nil && int64(side) != *f.Side || f.Group != nil && k.GroupID != *f.Group || k.Value < f.MinISK || f.From != nil && t < *f.From || f.To != nil && t >= *f.To {
			continue
		}
		v.Filtered = append(v.Filtered, k)
		v.LossISK += k.Value
		if k.GroupID == 30 || k.GroupID == 659 {
			v.Supers = append(v.Supers, k)
		}
		if t < start || t > end {
			continue
		}
		index := min(count-1, int((t-start)/step))
		bin := &v.Bins[index]
		bin.Count++
		bin.ISK += k.Value
		if side < 0 {
			side = len(sides)
		}
		bin.Sides[side].Count++
		bin.Sides[side].ISK += k.Value
		if k.GroupID == 30 || k.GroupID == 659 {
			bin.Big++
		}
	}
	for id, name := range groups {
		v.Groups = append(v.Groups, LossShipGroup{id, name})
	}
	sort.Slice(v.Groups, func(i, j int) bool { return v.Groups[i].Name < v.Groups[j].Name })
	v.LossCount = len(v.Filtered)
	v.Biggest = append([]ReportLoss{}, v.Filtered...)
	sort.SliceStable(v.Biggest, func(i, j int) bool { return v.Biggest[i].Value > v.Biggest[j].Value })
	v.Biggest = v.Biggest[:min(6, len(v.Biggest))]
	v.Busiest = v.Bins[0]
	v.PeakCount, v.PeakISK = 1, 1
	for _, bin := range v.Bins {
		v.PeakCount = max(v.PeakCount, float64(bin.Count))
		v.PeakISK = max(v.PeakISK, bin.ISK)
		if bin.Count > v.Busiest.Count {
			v.Busiest = bin
		}
	}
	return v
}
