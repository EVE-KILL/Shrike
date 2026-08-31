package images

import (
	"bytes"
	"context"
	"fmt"
	"image"
	"image/color"
	"image/png"
	"math"
	"sort"
	"strconv"
	"sync"
	"sync/atomic"

	"github.com/jackc/pgx/v5"
	xdraw "golang.org/x/image/draw"
	"golang.org/x/image/vector"
	"golang.org/x/sync/errgroup"
)

const (
	mapPadRatio       = 0.08
	mapLineWidthAt128 = 1.5
	mapDotRadiusAt128 = 2.7
)

type MapKind string

const (
	MapSystem        MapKind = "system"
	MapConstellation MapKind = "constellation"
	MapRegion        MapKind = "region"
)

type MapDatabase interface {
	Query(context.Context, string, ...any) (pgx.Rows, error)
}

type MapGenerateOptions struct {
	Kind        MapKind
	ID          int64
	Limit       int
	Size        int
	Sizes       []int
	Concurrency int
	Started     func(kind MapKind, id, current, total int64)
	Progress    func(done, total int64)
}

type mapPoint struct {
	id, typeID, group, constellation int64
	x, z, security                   float64
	hasPosition                      bool
	scoped                           bool
}

type mapJump struct{ from, to int64 }

func GenerateMapImages(ctx context.Context, db MapDatabase, store ObjectStore, options MapGenerateOptions) (ImportResult, error) {
	if store == nil {
		return ImportResult{}, unavailable()
	}
	if options.Size == 0 {
		options.Size = 1024
	}
	if options.Size < 16 || options.Size > 4096 {
		return ImportResult{}, fmt.Errorf("map image size must be between 16 and 4096")
	}
	seenSizes := make(map[int]struct{}, len(options.Sizes))
	for _, size := range options.Sizes {
		if size <= 0 || size >= options.Size {
			return ImportResult{}, fmt.Errorf("derived map image size %d must be positive and smaller than the base size", size)
		}
		if _, exists := seenSizes[size]; exists {
			return ImportResult{}, fmt.Errorf("derived map image size %d is duplicated", size)
		}
		seenSizes[size] = struct{}{}
	}
	if options.Concurrency <= 0 {
		options.Concurrency = 1
	}
	if options.Limit < 0 {
		return ImportResult{}, fmt.Errorf("map image limit must not be negative")
	}
	kinds := []MapKind{options.Kind}
	if options.Kind == "" {
		kinds = []MapKind{MapSystem, MapConstellation, MapRegion}
	}
	for _, kind := range kinds {
		if kind != MapSystem && kind != MapConstellation && kind != MapRegion {
			return ImportResult{}, fmt.Errorf("invalid map kind %q", kind)
		}
	}

	var result ImportResult
	for _, kind := range kinds {
		ids, err := mapIDs(ctx, db, kind, options.ID)
		if err != nil {
			return result, err
		}
		ids = limitMapIDs(ids, options.Limit)
		var done atomic.Int64
		var started atomic.Int64
		group, groupCtx := errgroup.WithContext(ctx)
		input := make(chan int64, options.Concurrency)
		var mu sync.Mutex
		for range options.Concurrency {
			group.Go(func() error {
				for id := range input {
					startIndex := started.Add(1)
					if options.Started != nil {
						options.Started(kind, id, startIndex, int64(len(ids)))
					}
					baseImage, err := renderMap(groupCtx, db, kind, id, options.Size)
					if err != nil {
						return fmt.Errorf("render %s %d: %w", kind, id, err)
					}
					base, err := encodeMap(baseImage)
					if err != nil {
						return fmt.Errorf("encode %s %d: %w", kind, id, err)
					}
					objects := []importObject{{Key: mapObjectKey(kind, id, 0), Body: base, ContentType: "image/png"}}
					for _, size := range options.Sizes {
						body, err := encodeMap(resizeMap(baseImage, size))
						if err != nil {
							return fmt.Errorf("encode %s %d at %dpx: %w", kind, id, size, err)
						}
						objects = append(objects, importObject{Key: mapObjectKey(kind, id, size), Body: body, ContentType: "image/png"})
					}
					for _, object := range objects {
						changed, err := putIfChanged(groupCtx, store, object)
						if err != nil {
							return err
						}
						mu.Lock()
						result.Scanned++
						if changed {
							result.Uploaded++
							result.Bytes += int64(len(object.Body))
						} else {
							result.Skipped++
						}
						mu.Unlock()
					}
					current := done.Add(1)
					if options.Progress != nil {
						options.Progress(current, int64(len(ids)))
					}
				}
				return nil
			})
		}
		group.Go(func() error {
			defer close(input)
			for _, id := range ids {
				select {
				case input <- id:
				case <-groupCtx.Done():
					return groupCtx.Err()
				}
			}
			return nil
		})
		if err := group.Wait(); err != nil {
			return result, err
		}
	}
	return result, nil
}

func limitMapIDs(ids []int64, limit int) []int64 {
	if limit <= 0 || limit >= len(ids) {
		return ids
	}
	return ids[:limit]
}

func mapIDs(ctx context.Context, db MapDatabase, kind MapKind, only int64) ([]int64, error) {
	if only > 0 {
		return []int64{only}, nil
	}
	table, column := "solar_systems", "solar_system_id"
	if kind == MapConstellation {
		table, column = "constellations", "constellation_id"
	}
	if kind == MapRegion {
		table, column = "regions", "region_id"
	}
	rows, err := db.Query(ctx, "SELECT "+column+" FROM "+table+" ORDER BY "+column)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var ids []int64
	for rows.Next() {
		var id int64
		if err := rows.Scan(&id); err != nil {
			return nil, err
		}
		ids = append(ids, id)
	}
	return ids, rows.Err()
}

func renderMap(ctx context.Context, db MapDatabase, kind MapKind, id int64, size int) (image.Image, error) {
	if kind == MapSystem {
		points, err := loadSystemBodies(ctx, db, id)
		if err != nil {
			return nil, err
		}
		return renderSystem(points, size), nil
	}
	points, jumps, err := loadNetwork(ctx, db, kind, id)
	if err != nil {
		return nil, err
	}
	if len(points) == 0 {
		return nil, fmt.Errorf("no systems found")
	}
	return renderNetwork(kind, points, jumps, size), nil
}

func loadSystemBodies(ctx context.Context, db MapDatabase, id int64) ([]mapPoint, error) {
	rows, err := db.Query(ctx, `SELECT item_id, COALESCE(type_id,0), COALESCE(group_id,0), x, z FROM celestials WHERE solar_system_id=$1 AND group_id IN (6,7)`, id)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []mapPoint
	for rows.Next() {
		var p mapPoint
		var x, z *float64
		if err := rows.Scan(&p.id, &p.typeID, &p.group, &x, &z); err != nil {
			return nil, err
		}
		if x != nil && z != nil {
			p.x, p.z, p.hasPosition = *x, *z, true
			out = append(out, p)
		}
	}
	return out, rows.Err()
}

func loadNetwork(ctx context.Context, db MapDatabase, kind MapKind, id int64) ([]mapPoint, []mapJump, error) {
	column := "constellation_id"
	if kind == MapRegion {
		column = "region_id"
	}
	rows, err := db.Query(ctx, `SELECT system.solar_system_id,system.x,system.z,COALESCE(system.security,0),COALESCE(system.constellation_id,0),COALESCE(star.type_id,0)
		FROM solar_systems system
		LEFT JOIN celestials star ON star.solar_system_id=system.solar_system_id AND star.group_id=6
		WHERE system.`+column+`=$1`, id)
	if err != nil {
		return nil, nil, err
	}
	defer rows.Close()
	byID := map[int64]mapPoint{}
	var ids []int64
	for rows.Next() {
		var p mapPoint
		var x, z *float64
		if err := rows.Scan(&p.id, &x, &z, &p.security, &p.constellation, &p.typeID); err != nil {
			return nil, nil, err
		}
		if x != nil && z != nil {
			p.x, p.z, p.hasPosition = *x, *z, true
		}
		p.scoped = true
		byID[p.id] = p
		ids = append(ids, p.id)
	}
	if err := rows.Err(); err != nil {
		return nil, nil, err
	}
	if len(ids) == 0 {
		return nil, nil, nil
	}
	queryIDs := make([]int32, len(ids))
	for index, value := range ids {
		queryIDs[index] = int32(value)
	}
	jumpRows, err := db.Query(ctx, `SELECT from_solar_system_id,to_solar_system_id FROM solar_system_jumps WHERE from_solar_system_id=ANY($1::int[]) OR to_solar_system_id=ANY($1::int[])`, queryIDs)
	if err != nil {
		return nil, nil, err
	}
	defer jumpRows.Close()
	var jumps []mapJump
	external := map[int64]struct{}{}
	for jumpRows.Next() {
		var j mapJump
		if err := jumpRows.Scan(&j.from, &j.to); err != nil {
			return nil, nil, err
		}
		jumps = append(jumps, j)
		if _, ok := byID[j.from]; !ok {
			external[j.from] = struct{}{}
		}
		if _, ok := byID[j.to]; !ok {
			external[j.to] = struct{}{}
		}
	}
	if len(external) > 0 {
		ext := make([]int64, 0, len(external))
		for n := range external {
			ext = append(ext, n)
		}
		extIDs := make([]int32, len(ext))
		for index, value := range ext {
			extIDs[index] = int32(value)
		}
		erows, err := db.Query(ctx, `SELECT system.solar_system_id,system.x,system.z,COALESCE(system.security,0),COALESCE(system.constellation_id,0),COALESCE(star.type_id,0)
			FROM solar_systems system
			LEFT JOIN celestials star ON star.solar_system_id=system.solar_system_id AND star.group_id=6
			WHERE system.solar_system_id=ANY($1::int[])`, extIDs)
		if err != nil {
			return nil, nil, err
		}
		defer erows.Close()
		for erows.Next() {
			var p mapPoint
			var x, z *float64
			if err := erows.Scan(&p.id, &x, &z, &p.security, &p.constellation, &p.typeID); err != nil {
				return nil, nil, err
			}
			if x != nil && z != nil {
				p.x, p.z, p.hasPosition = *x, *z, true
			}
			byID[p.id] = p
		}
	}
	points := make([]mapPoint, 0, len(byID))
	for _, p := range byID {
		points = append(points, p)
	}
	sort.Slice(points, func(i, j int) bool { return points[i].id < points[j].id })
	return points, jumps, nil
}

func mapObjectKey(kind MapKind, id int64, small int) string {
	folder := string(kind) + "s"
	name := strconv.FormatInt(id, 10)
	if small > 0 {
		name += "_" + strconv.Itoa(small)
	}
	return "static/" + folder + "/" + name + ".png"
}

func renderSystem(points []mapPoint, size int) image.Image {
	c := newCanvas(size)
	cx := float64(size) / 2
	unit := float64(size) / 128
	lineWidth := mapLineWidthAt128 * unit
	dotRadius := mapDotRadiusAt128 * unit
	pad := float64(size) * mapPadRatio
	outer := float64(size)/2 - pad
	if len(points) == 0 {
		c.disk(cx, cx, 5.5*unit, rgba("9ca3af", 255))
		return c.image()
	}
	var sun mapPoint
	var planets []mapPoint
	for _, p := range points {
		switch p.group {
		case 6:
			sun = p
		case 7:
			planets = append(planets, p)
		}
	}
	maxOrbit := 1.0
	for _, p := range planets {
		maxOrbit = math.Max(maxOrbit, math.Hypot(p.x-sun.x, p.z-sun.z))
	}
	scale := outer / (maxOrbit * 1.05)
	for _, p := range planets {
		orbit := math.Hypot(p.x-sun.x, p.z-sun.z) * scale
		if orbit >= 1 {
			c.circle(cx, cx, orbit, rgba("6b7280", 105), lineWidth)
		}
	}
	starHalo, starCore := starTypeColors(sun.typeID)
	c.disk(cx, cx, 7*unit, starHalo)
	c.disk(cx, cx, 5.5*unit, starCore)
	c.disk(cx-1.6*unit, cx-1.6*unit, 1.8*unit, rgba("f3f4f6", 210))
	for _, p := range planets {
		c.disk(cx+(p.x-sun.x)*scale, cx-(p.z-sun.z)*scale, dotRadius, planetTypeColor(p.typeID))
	}
	return c.image()
}

func starTypeColors(typeID int64) (halo, core color.RGBA) {
	switch typeID {
	case 3796, 56084: // O1 bright blue and Divine Immanence
		return rgba("075985", 190), rgba("38bdf8", 255)
	case 9, 3801, 34331, 45034, 45046, 56083, 56085, 56097, 56098, 73909, 78350: // A0/B0 blue families
		return rgba("1e3a8a", 190), rgba("60a5fa", 255)
	case 3803, 45042: // B5 white dwarf
		return rgba("64748b", 190), rgba("dbeafe", 255)
	case 10, 45035: // F0 white
		return rgba("9ca3af", 190), rgba("f8fafc", 255)
	case 3799, 45038: // G3 pink small
		return rgba("9d174d", 190), rgba("f9a8d4", 255)
	case 3797, 45036: // G5 pink
		return rgba("9f1239", 190), rgba("fb7185", 255)
	case 56082, 56086: // G5 gold immanence
		return rgba("92400e", 190), rgba("fbbf24", 255)
	case 6, 3802, 45030, 45041, 45047: // G5/K3 yellow
		return rgba("a16207", 190), rgba("fde047", 255)
	case 7, 3798, 45031, 45032, 45037: // K5/K7 orange
		return rgba("9a3412", 190), rgba("fb923c", 255)
	case 8, 45033: // K5 red giant
		return rgba("991b1b", 190), rgba("f87171", 255)
	case 3800, 45039, 45040: // M0 orange radiant
		return rgba("7c2d12", 190), rgba("f97316", 255)
	default:
		return rgba("111827", 180), rgba("9ca3af", 255)
	}
}

func planetTypeColor(typeID int64) color.RGBA {
	switch typeID {
	case 11: // Temperate
		return rgba("22c55e", 255)
	case 12: // Ice
		return rgba("bae6fd", 255)
	case 13: // Gas
		return rgba("c084fc", 255)
	case 2014: // Oceanic
		return rgba("3b82f6", 255)
	case 2015: // Lava
		return rgba("ef4444", 255)
	case 2016: // Barren
		return rgba("a88b6a", 255)
	case 2017: // Storm
		return rgba("64748b", 255)
	case 2063: // Plasma
		return rgba("00d9ff", 255)
	case 30889: // Shattered
		return rgba("e5e7eb", 255)
	case 73911: // Scorched Barren
		return rgba("f97316", 255)
	default:
		return rgba("9ca3af", 255)
	}
}

func renderNetwork(kind MapKind, points []mapPoint, jumps []mapJump, size int) image.Image {
	c := newCanvas(size)
	unit := float64(size) / 128
	lineWidth := mapLineWidthAt128 * unit
	dotRadius := mapDotRadiusAt128 * unit
	byID := map[int64]mapPoint{}
	inside := map[int64]bool{}
	var scoped []mapPoint
	for _, p := range points {
		byID[p.id] = p
		if p.scoped {
			inside[p.id] = true
			if p.hasPosition {
				scoped = append(scoped, p)
			}
		}
	}
	minX, maxX, minY, maxY := scoped[0].x, scoped[0].x, -scoped[0].z, -scoped[0].z
	for _, p := range scoped {
		minX = math.Min(minX, p.x)
		maxX = math.Max(maxX, p.x)
		minY = math.Min(minY, -p.z)
		maxY = math.Max(maxY, -p.z)
	}
	scale, ox, oy := fitMap(minX, maxX, minY, maxY, size)
	project := func(p mapPoint) (float64, float64) { return p.x*scale + ox, -p.z*scale + oy }
	for _, j := range jumps {
		a, aok := byID[j.from]
		b, bok := byID[j.to]
		if !aok || !bok || !a.hasPosition || !b.hasPosition {
			continue
		}
		x1, y1 := project(a)
		x2, y2 := project(b)
		stroke := secRGBA(math.Min(a.security, b.security))
		if kind == MapConstellation && inside[a.id] && inside[b.id] {
			stroke = rgba("ef4444", 255)
		} else if kind == MapRegion {
			if inside[a.id] && inside[b.id] {
				stroke = rgba("2563eb", 255)
			} else {
				stroke = rgba("22c55e", 255)
			}
		}
		c.line(x1, y1, x2, y2, lineWidth, stroke)
	}
	for _, p := range scoped {
		x, y := project(p)
		_, nodeColor := starTypeColors(p.typeID)
		nodeRadius := dotRadius
		outlineWidth := .6 * unit
		if kind == MapRegion {
			nodeRadius *= .5
			outlineWidth *= .5
		}
		c.disk(x, y, nodeRadius, nodeColor)
		c.circle(x, y, nodeRadius, rgba("111827", 255), outlineWidth)
	}
	return c.image()
}

func fitMap(minX, maxX, minY, maxY float64, size int) (float64, float64, float64) {
	pad := float64(size) * mapPadRatio
	inner := float64(size) - 2*pad
	dx := math.Max(maxX-minX, 1)
	dy := math.Max(maxY-minY, 1)
	scale := math.Min(inner/dx, inner/dy)
	return scale, pad + (inner-dx*scale)/2 - minX*scale, pad + (inner-dy*scale)/2 - minY*scale
}
func secRGBA(sec float64) color.RGBA {
	if sec >= .5 {
		return rgba("3b82f6", 255)
	}
	if sec >= .1 {
		return rgba("ef4444", 255)
	}
	return rgba("10b981", 255)
}
func pastelRGBA(seed int64) color.RGBA {
	h := math.Mod(float64(seed%1000)*137.508, 360)
	return hslRGBA(h, .62, .72)
}
func hslRGBA(h, s, l float64) color.RGBA {
	k := func(n float64) float64 { return math.Mod(n+h/30, 12) }
	a := s * math.Min(l, 1-l)
	f := func(n float64) uint8 {
		return uint8(math.Round(255 * (l - a*math.Max(-1, math.Min(math.Min(k(n)-3, 9-k(n)), 1)))))
	}
	return color.RGBA{f(0), f(8), f(4), 255}
}
func rgba(hex string, a uint8) color.RGBA {
	v, _ := strconv.ParseUint(hex, 16, 32)
	return color.RGBA{uint8(v >> 16), uint8(v >> 8), uint8(v), a}
}

type mapCanvas struct {
	size, scale int
	img         *image.RGBA
}

func newCanvas(size int) *mapCanvas {
	scale := 2
	return &mapCanvas{size: size, scale: scale, img: image.NewRGBA(image.Rect(0, 0, size*scale, size*scale))}
}
func (c *mapCanvas) disk(x, y, r float64, fill color.RGBA) { c.shapeCircle(x, y, r, fill, true, 0) }
func (c *mapCanvas) circle(x, y, r float64, stroke color.RGBA, width float64) {
	c.shapeCircle(x, y, r, stroke, false, width)
}
func (c *mapCanvas) shapeCircle(x, y, r float64, col color.RGBA, fill bool, width float64) {
	s := float64(c.scale)
	ras := vector.NewRasterizer(c.size*c.scale, c.size*c.scale)
	k := .5522847498
	x *= s
	y *= s
	drawClockwiseCircle := func(radius float64) {
		ras.MoveTo(float32(x+radius), float32(y))
		ras.CubeTo(float32(x+radius), float32(y+k*radius), float32(x+k*radius), float32(y+radius), float32(x), float32(y+radius))
		ras.CubeTo(float32(x-k*radius), float32(y+radius), float32(x-radius), float32(y+k*radius), float32(x-radius), float32(y))
		ras.CubeTo(float32(x-radius), float32(y-k*radius), float32(x-k*radius), float32(y-radius), float32(x), float32(y-radius))
		ras.CubeTo(float32(x+k*radius), float32(y-radius), float32(x+radius), float32(y-k*radius), float32(x+radius), float32(y))
		ras.ClosePath()
	}
	drawCounterClockwiseCircle := func(radius float64) {
		ras.MoveTo(float32(x+radius), float32(y))
		ras.CubeTo(float32(x+radius), float32(y-k*radius), float32(x+k*radius), float32(y-radius), float32(x), float32(y-radius))
		ras.CubeTo(float32(x-k*radius), float32(y-radius), float32(x-radius), float32(y-k*radius), float32(x-radius), float32(y))
		ras.CubeTo(float32(x-radius), float32(y+k*radius), float32(x-k*radius), float32(y+radius), float32(x), float32(y+radius))
		ras.CubeTo(float32(x+k*radius), float32(y+radius), float32(x+radius), float32(y+k*radius), float32(x+radius), float32(y))
		ras.ClosePath()
	}
	if fill {
		drawClockwiseCircle(r * s)
	} else {
		halfWidth := width * s / 2
		drawClockwiseCircle(r*s + halfWidth)
		drawCounterClockwiseCircle(math.Max(0, r*s-halfWidth))
	}
	ras.Draw(c.img, c.img.Bounds(), image.NewUniform(col), image.Point{})
}
func (c *mapCanvas) line(x1, y1, x2, y2, width float64, col color.RGBA) {
	s := float64(c.scale)
	x1 *= s
	y1 *= s
	x2 *= s
	y2 *= s
	half := width * s / 2
	dx, dy := x2-x1, y2-y1
	l := math.Hypot(dx, dy)
	if l == 0 {
		return
	}
	nx, ny := -dy/l*half, dx/l*half
	ras := vector.NewRasterizer(c.size*c.scale, c.size*c.scale)
	ras.MoveTo(float32(x1+nx), float32(y1+ny))
	ras.LineTo(float32(x2+nx), float32(y2+ny))
	ras.LineTo(float32(x2-nx), float32(y2-ny))
	ras.LineTo(float32(x1-nx), float32(y1-ny))
	ras.ClosePath()
	ras.Draw(c.img, c.img.Bounds(), image.NewUniform(col), image.Point{})
}
func (c *mapCanvas) image() image.Image {
	dst := image.NewRGBA(image.Rect(0, 0, c.size, c.size))
	xdraw.CatmullRom.Scale(dst, dst.Bounds(), c.img, c.img.Bounds(), xdraw.Over, nil)
	return dst
}
func encodeMap(img image.Image) ([]byte, error) {
	var out bytes.Buffer
	err := png.Encode(&out, img)
	return out.Bytes(), err
}
func resizeMap(src image.Image, size int) image.Image {
	dst := image.NewRGBA(image.Rect(0, 0, size, size))
	xdraw.CatmullRom.Scale(dst, dst.Bounds(), src, src.Bounds(), xdraw.Over, nil)
	return dst
}
