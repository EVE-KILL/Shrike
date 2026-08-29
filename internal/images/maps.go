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

const mapPadRatio = 0.08

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
	Size        int
	SmallSize   int
	Concurrency int
	Progress    func(done, total int64)
}

type mapPoint struct {
	id, group, constellation int64
	x, z, security           float64
	hasPosition              bool
	scoped                   bool
}

type mapJump struct{ from, to int64 }

func GenerateMapImages(ctx context.Context, db MapDatabase, store ObjectStore, options MapGenerateOptions) (ImportResult, error) {
	if store == nil {
		return ImportResult{}, unavailable()
	}
	if options.Size == 0 {
		options.Size = 128
	}
	if options.Size < 16 || options.Size > 4096 {
		return ImportResult{}, fmt.Errorf("map image size must be between 16 and 4096")
	}
	if options.SmallSize < 0 || options.SmallSize >= options.Size {
		return ImportResult{}, fmt.Errorf("small map image size must be positive and smaller than the base size")
	}
	if options.Concurrency <= 0 {
		options.Concurrency = 16
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
		var done atomic.Int64
		group, groupCtx := errgroup.WithContext(ctx)
		input := make(chan int64, options.Concurrency)
		var mu sync.Mutex
		for range options.Concurrency {
			group.Go(func() error {
				for id := range input {
					base, err := renderMapPNG(groupCtx, db, kind, id, options.Size)
					if err != nil {
						return fmt.Errorf("render %s %d: %w", kind, id, err)
					}
					objects := []importObject{{Key: mapObjectKey(kind, id, 0), Body: base, ContentType: "image/png"}}
					if options.SmallSize > 0 {
						objects = append(objects, importObject{Key: mapObjectKey(kind, id, options.SmallSize), Body: resizePNG(base, options.SmallSize), ContentType: "image/png"})
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

func renderMapPNG(ctx context.Context, db MapDatabase, kind MapKind, id int64, size int) ([]byte, error) {
	if kind == MapSystem {
		points, err := loadSystemBodies(ctx, db, id)
		if err != nil {
			return nil, err
		}
		return encodeMap(renderSystem(points, size))
	}
	points, jumps, err := loadNetwork(ctx, db, kind, id)
	if err != nil {
		return nil, err
	}
	if len(points) == 0 {
		return nil, fmt.Errorf("no systems found")
	}
	return encodeMap(renderNetwork(kind, points, jumps, size))
}

func loadSystemBodies(ctx context.Context, db MapDatabase, id int64) ([]mapPoint, error) {
	rows, err := db.Query(ctx, `SELECT COALESCE(group_id,0), x, z FROM celestials WHERE solar_system_id=$1 AND group_id IN (6,7)`, id)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []mapPoint
	for rows.Next() {
		var p mapPoint
		var x, z *float64
		if err := rows.Scan(&p.group, &x, &z); err != nil {
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
	rows, err := db.Query(ctx, `SELECT solar_system_id,x,z,COALESCE(security,0),COALESCE(constellation_id,0) FROM solar_systems WHERE `+column+`=$1`, id)
	if err != nil {
		return nil, nil, err
	}
	defer rows.Close()
	byID := map[int64]mapPoint{}
	var ids []int64
	for rows.Next() {
		var p mapPoint
		var x, z *float64
		if err := rows.Scan(&p.id, &x, &z, &p.security, &p.constellation); err != nil {
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
		erows, err := db.Query(ctx, `SELECT solar_system_id,x,z,COALESCE(security,0),COALESCE(constellation_id,0) FROM solar_systems WHERE solar_system_id=ANY($1::int[])`, extIDs)
		if err != nil {
			return nil, nil, err
		}
		defer erows.Close()
		for erows.Next() {
			var p mapPoint
			var x, z *float64
			if err := erows.Scan(&p.id, &x, &z, &p.security, &p.constellation); err != nil {
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
	pad := float64(size) * mapPadRatio
	outer := float64(size)/2 - pad
	if len(points) == 0 {
		c.circle(cx, cx, outer, rgba("6b7280", 178), .6)
		c.disk(cx, cx, 2, rgba("6b7280", 128))
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
			c.circle(cx, cx, orbit, rgba("6b7280", 115), 1.2)
		}
	}
	c.circle(cx, cx, outer, rgba("6b7280", 178), .6)
	c.disk(cx, cx, 3, rgba("f59e0b", 255))
	for _, p := range planets {
		c.disk(cx+(p.x-sun.x)*scale, cx-(p.z-sun.z)*scale, 2.4, rgba("16a34a", 255))
	}
	return c.image()
}

func renderNetwork(kind MapKind, points []mapPoint, jumps []mapJump, size int) image.Image {
	c := newCanvas(size)
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
			stroke = pastelRGBA(a.id)
		}
		c.line(x1, y1, x2, y2, 1.2, stroke)
	}
	for _, p := range scoped {
		x, y := project(p)
		seed := p.id
		if kind == MapRegion {
			seed = p.constellation
		}
		c.disk(x, y, 2.4, pastelRGBA(seed))
		c.circle(x, y, 2.4, rgba("111827", 255), .6)
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
	scale := 4
	return &mapCanvas{size: size, scale: scale, img: image.NewRGBA(image.Rect(0, 0, size*scale, size*scale))}
}
func (c *mapCanvas) disk(x, y, r float64, fill color.RGBA) { c.shapeCircle(x, y, r, fill, true, 0) }
func (c *mapCanvas) circle(x, y, r float64, stroke color.RGBA, width float64) {
	segments := int(math.Max(32, r*3))
	lastX, lastY := x+r, y
	for i := 1; i <= segments; i++ {
		angle := 2 * math.Pi * float64(i) / float64(segments)
		nextX, nextY := x+r*math.Cos(angle), y+r*math.Sin(angle)
		c.line(lastX, lastY, nextX, nextY, width, stroke)
		lastX, lastY = nextX, nextY
	}
}
func (c *mapCanvas) shapeCircle(x, y, r float64, col color.RGBA, fill bool, width float64) {
	s := float64(c.scale)
	ras := vector.NewRasterizer(c.size*c.scale, c.size*c.scale)
	k := .5522847498
	r *= s
	x *= s
	y *= s
	ras.MoveTo(float32(x+r), float32(y))
	ras.CubeTo(float32(x+r), float32(y+k*r), float32(x+k*r), float32(y+r), float32(x), float32(y+r))
	ras.CubeTo(float32(x-k*r), float32(y+r), float32(x-r), float32(y+k*r), float32(x-r), float32(y))
	ras.CubeTo(float32(x-r), float32(y-k*r), float32(x-k*r), float32(y-r), float32(x), float32(y-r))
	ras.CubeTo(float32(x+k*r), float32(y-r), float32(x+r), float32(y-k*r), float32(x+r), float32(y))
	ras.ClosePath()
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
func resizePNG(body []byte, size int) []byte {
	src, err := png.Decode(bytes.NewReader(body))
	if err != nil {
		return nil
	}
	dst := image.NewRGBA(image.Rect(0, 0, size, size))
	xdraw.CatmullRom.Scale(dst, dst.Bounds(), src, src.Bounds(), xdraw.Over, nil)
	var out bytes.Buffer
	_ = png.Encode(&out, dst)
	return out.Bytes()
}
