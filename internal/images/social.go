package images

import (
	"bytes"
	"context"
	"fmt"
	"image"
	"image/color"
	"image/draw"
	"image/png"
	"math"
	"net/http"
	"strings"
	"sync"
	"unicode/utf8"

	"github.com/eve-kill/shrike/internal/objectstore"
	xdraw "golang.org/x/image/draw"
	"golang.org/x/image/font"
	"golang.org/x/image/font/gofont/gobold"
	"golang.org/x/image/font/gofont/goitalic"
	"golang.org/x/image/font/gofont/goregular"
	"golang.org/x/image/font/opentype"
	"golang.org/x/image/math/fixed"
)

const (
	socialWidth  = 550
	socialHeight = 200
	socialIcon   = 64
)

// SocialLoader owns the database lookup so rendering stays testable without a
// live Postgres instance.
type SocialLoader interface {
	LoadKillmailSocial(context.Context, int64) (SocialKillmail, bool, error)
}

type SocialParty struct {
	CharacterID     int64
	CharacterName   string
	CorporationID   int64
	CorporationName string
	AllianceID      int64
	AllianceName    string
	ShipTypeID      int64
	ShipName        string
}

type SocialKillmail struct {
	TotalValue      float64
	SolarSystemName string
	RegionName      string
	Victim          SocialParty
	FinalBlow       *SocialParty
}

func (s *Service) KillmailSocial(
	ctx context.Context,
	id int64,
) (Result, error) {
	if !s.Available() {
		return Result{}, unavailable()
	}
	if err := validateID(id); err != nil {
		return Result{}, err
	}
	if s.social == nil {
		return Result{}, statusError(
			http.StatusServiceUnavailable,
			"Killmail image data is not configured",
			nil,
		)
	}
	cacheKey := fmt.Sprintf("social/killmail/%d", id)
	return s.cacheResult(cacheKey, func() (Result, error) {
		objectKey := fmt.Sprintf("social/killmails/v1/%d.png", id)
		if object, err := s.store.GetObject(ctx, objectKey); err != nil {
			return Result{}, fmt.Errorf("read killmail social image: %w", err)
		} else if object != nil {
			modified := object.LastModified
			if modified.IsZero() {
				modified = s.now()
			}
			result := newResult(object.Body, "image/png", modified)
			result.CacheControl = immutableCacheControl
			return result, nil
		}

		killmail, found, err := s.social.LoadKillmailSocial(ctx, id)
		if err != nil {
			return Result{}, fmt.Errorf("load killmail social data: %w", err)
		}
		if !found {
			return Result{}, statusError(http.StatusNotFound, "Killmail not found", nil)
		}
		body, err := s.renderKillmailSocial(ctx, killmail)
		if err != nil {
			return Result{}, err
		}
		if err := s.store.PutWithOptions(
			context.WithoutCancel(ctx),
			objectKey,
			body,
			objectstore.PutOptions{
				ContentType: "image/png", CacheControl: immutableCacheControl,
			},
		); err != nil {
			return Result{}, fmt.Errorf("store killmail social image: %w", err)
		}
		result := newResult(body, "image/png", s.now())
		result.CacheControl = immutableCacheControl
		return result, nil
	})
}

func (s *Service) renderKillmailSocial(
	ctx context.Context,
	killmail SocialKillmail,
) ([]byte, error) {
	canvas := image.NewNRGBA(image.Rect(0, 0, socialWidth, socialHeight))
	drawSocialBackground(canvas)

	const (
		groupWidth = 4 * socialIcon
		separator  = 30
		iconsY     = 6
	)
	totalWidth := 2*groupWidth + separator
	leftStart := (socialWidth - totalWidth) / 2
	rightStart := leftStart + groupWidth + separator
	leftCenter := leftStart + groupWidth/2
	rightCenter := rightStart + groupWidth/2
	separatorX := leftStart + groupWidth + separator/2

	for y := iconsY + 6; y <= iconsY+socialIcon-6; y++ {
		canvas.SetNRGBA(separatorX, y, color.NRGBA{R: 255, G: 255, B: 255, A: 64})
	}

	parties := []SocialParty{killmail.Victim, {}}
	if killmail.FinalBlow != nil {
		parties[1] = *killmail.FinalBlow
	}
	icons := [2][4]image.Image{}
	var wait sync.WaitGroup
	for partyIndex := range parties {
		party := parties[partyIndex]
		loaders := []func() (Result, error){
			func() (Result, error) {
				return s.Entity(ctx, Character, party.CharacterID, socialIcon, "jpeg")
			},
			func() (Result, error) {
				return s.Entity(ctx, Corporation, party.CorporationID, socialIcon, "png")
			},
			func() (Result, error) {
				return s.Entity(ctx, Alliance, party.AllianceID, socialIcon, "png")
			},
			func() (Result, error) {
				return s.Type(ctx, party.ShipTypeID, "icon", socialIcon, "")
			},
		}
		ids := []int64{
			party.CharacterID,
			party.CorporationID,
			party.AllianceID,
			party.ShipTypeID,
		}
		for iconIndex := range loaders {
			if ids[iconIndex] <= 0 {
				continue
			}
			wait.Add(1)
			go func(partyIndex, iconIndex int, load func() (Result, error)) {
				defer wait.Done()
				result, err := load()
				if err != nil {
					return
				}
				decoded, _, err := image.Decode(bytes.NewReader(result.Body))
				if err == nil {
					icons[partyIndex][iconIndex] = squareImage(decoded, socialIcon)
				}
			}(partyIndex, iconIndex, loaders[iconIndex])
		}
	}
	wait.Wait()
	drawPackedIcons(canvas, leftStart, groupWidth, iconsY, icons[0][:])
	drawPackedIcons(canvas, rightStart, groupWidth, iconsY, icons[1][:])

	faces, err := newSocialFaces()
	if err != nil {
		return nil, fmt.Errorf("load social image fonts: %w", err)
	}
	defer faces.close()
	drawPartyText(canvas, leftCenter, killmail.Victim, "VICTIM", faces)
	if killmail.FinalBlow != nil {
		drawPartyText(canvas, rightCenter, *killmail.FinalBlow, "FINAL BLOW", faces)
	}
	drawText(
		canvas,
		8,
		socialHeight-8,
		formatSocialISK(killmail.TotalValue)+" ISK",
		color.NRGBA{R: 251, G: 191, B: 36, A: 255},
		faces.bold14,
		false,
	)
	location := killmail.SolarSystemName
	if location == "" {
		location = "Unknown"
	}
	if killmail.RegionName != "" {
		location += " (" + killmail.RegionName + ")"
	}
	drawText(
		canvas,
		socialWidth-8,
		socialHeight-8,
		location,
		color.NRGBA{R: 107, G: 114, B: 128, A: 255},
		faces.regular12,
		true,
	)

	var output bytes.Buffer
	encoder := png.Encoder{CompressionLevel: png.BestCompression}
	if err := encoder.Encode(&output, canvas); err != nil {
		return nil, fmt.Errorf("encode killmail social image: %w", err)
	}
	return output.Bytes(), nil
}

func drawSocialBackground(canvas *image.NRGBA) {
	from := color.NRGBA{R: 11, G: 15, B: 26, A: 255}
	to := color.NRGBA{R: 26, G: 31, B: 46, A: 255}
	for y := range socialHeight {
		ratio := float64(y) / float64(socialHeight-1)
		line := color.NRGBA{
			R: uint8(math.Round(float64(from.R)*(1-ratio) + float64(to.R)*ratio)),
			G: uint8(math.Round(float64(from.G)*(1-ratio) + float64(to.G)*ratio)),
			B: uint8(math.Round(float64(from.B)*(1-ratio) + float64(to.B)*ratio)),
			A: 255,
		}
		draw.Draw(canvas, image.Rect(0, y, socialWidth, y+1), &image.Uniform{C: line}, image.Point{}, draw.Src)
	}
}

func squareImage(source image.Image, size int) image.Image {
	width := source.Bounds().Dx()
	height := source.Bounds().Dy()
	if width <= 0 || height <= 0 {
		return nil
	}
	scale := math.Max(float64(size)/float64(width), float64(size)/float64(height))
	scaledWidth := max(size, int(math.Ceil(float64(width)*scale)))
	scaledHeight := max(size, int(math.Ceil(float64(height)*scale)))
	resized := image.NewNRGBA(image.Rect(0, 0, scaledWidth, scaledHeight))
	xdraw.CatmullRom.Scale(
		resized,
		resized.Bounds(),
		source,
		source.Bounds(),
		draw.Over,
		nil,
	)
	x := (scaledWidth - size) / 2
	y := (scaledHeight - size) / 2
	square := image.NewNRGBA(image.Rect(0, 0, size, size))
	draw.Draw(square, square.Bounds(), resized, image.Pt(x, y), draw.Src)
	return square
}

func drawPackedIcons(
	canvas *image.NRGBA,
	startX int,
	groupWidth int,
	y int,
	icons []image.Image,
) {
	present := 0
	for _, icon := range icons {
		if icon != nil {
			present++
		}
	}
	x := startX + (groupWidth-present*socialIcon)/2
	for _, icon := range icons {
		if icon == nil {
			continue
		}
		draw.Draw(
			canvas,
			image.Rect(x, y, x+socialIcon, y+socialIcon),
			icon,
			icon.Bounds().Min,
			draw.Over,
		)
		x += socialIcon
	}
}

type socialFaces struct {
	regular10 font.Face
	regular11 font.Face
	regular12 font.Face
	regular14 font.Face
	bold9     font.Face
	bold14    font.Face
	italic12  font.Face
}

var (
	socialFontOnce sync.Once
	socialRegular  *opentype.Font
	socialBold     *opentype.Font
	socialItalic   *opentype.Font
	socialFontErr  error
)

func newSocialFaces() (socialFaces, error) {
	socialFontOnce.Do(func() {
		socialRegular, socialFontErr = opentype.Parse(goregular.TTF)
		if socialFontErr != nil {
			return
		}
		socialBold, socialFontErr = opentype.Parse(gobold.TTF)
		if socialFontErr != nil {
			return
		}
		socialItalic, socialFontErr = opentype.Parse(goitalic.TTF)
	})
	if socialFontErr != nil {
		return socialFaces{}, socialFontErr
	}
	face := func(parsed *opentype.Font, size float64) (font.Face, error) {
		return opentype.NewFace(parsed, &opentype.FaceOptions{
			Size: size, DPI: 72, Hinting: font.HintingFull,
		})
	}
	var out socialFaces
	var err error
	for _, item := range []struct {
		target *font.Face
		parsed *opentype.Font
		size   float64
	}{
		{&out.regular10, socialRegular, 10},
		{&out.regular11, socialRegular, 11},
		{&out.regular12, socialRegular, 12},
		{&out.regular14, socialRegular, 14},
		{&out.bold9, socialBold, 9},
		{&out.bold14, socialBold, 14},
		{&out.italic12, socialItalic, 12},
	} {
		*item.target, err = face(item.parsed, item.size)
		if err != nil {
			out.close()
			return socialFaces{}, err
		}
	}
	return out, nil
}

func (f socialFaces) close() {
	for _, face := range []font.Face{
		f.regular10, f.regular11, f.regular12, f.regular14,
		f.bold9, f.bold14, f.italic12,
	} {
		if closer, ok := face.(interface{ Close() error }); ok {
			_ = closer.Close()
		}
	}
}

func drawPartyText(
	canvas draw.Image,
	center int,
	party SocialParty,
	header string,
	faces socialFaces,
) {
	y := 84
	drawCenteredText(canvas, center, y, header, color.NRGBA{R: 156, G: 163, B: 175, A: 255}, faces.bold9)
	y += 14
	character := party.CharacterName
	if character == "" {
		character = "Unknown"
	}
	drawCenteredText(canvas, center, y, truncateRunes(character, 28), color.NRGBA{R: 229, G: 231, B: 235, A: 255}, faces.bold14)
	y += 15
	if party.CorporationName != "" {
		drawCenteredText(canvas, center, y, truncateRunes(party.CorporationName, 34), color.NRGBA{R: 156, G: 163, B: 175, A: 255}, faces.regular11)
		y += 14
	}
	if party.AllianceName != "" {
		drawCenteredText(canvas, center, y, truncateRunes(party.AllianceName, 34), color.NRGBA{R: 107, G: 114, B: 128, A: 255}, faces.regular10)
		y += 13
	}
	if party.ShipName != "" {
		drawCenteredText(canvas, center, y+1, truncateRunes(party.ShipName, 30), color.NRGBA{R: 147, G: 197, B: 253, A: 255}, faces.italic12)
	}
}

func drawCenteredText(
	canvas draw.Image,
	x int,
	y int,
	value string,
	colour color.Color,
	face font.Face,
) {
	drawer := &font.Drawer{
		Dst: canvas, Src: image.NewUniform(colour), Face: face,
		Dot: fixed.P(x, y),
	}
	drawer.Dot.X -= drawer.MeasureString(value) / 2
	drawer.DrawString(value)
}

func drawText(
	canvas draw.Image,
	x int,
	y int,
	value string,
	colour color.Color,
	face font.Face,
	right bool,
) {
	drawer := &font.Drawer{
		Dst: canvas, Src: image.NewUniform(colour), Face: face,
		Dot: fixed.P(x, y),
	}
	if right {
		drawer.Dot.X -= drawer.MeasureString(value)
	}
	drawer.DrawString(value)
}

func truncateRunes(value string, maximum int) string {
	if utf8.RuneCountInString(value) <= maximum {
		return value
	}
	runes := []rune(value)
	return string(runes[:maximum-1]) + "…"
}

func formatSocialISK(value float64) string {
	switch {
	case value >= 1_000_000_000_000:
		return fmt.Sprintf("%.2fT", value/1_000_000_000_000)
	case value >= 1_000_000_000:
		return fmt.Sprintf("%.2fB", value/1_000_000_000)
	case value >= 1_000_000:
		return fmt.Sprintf("%.1fM", value/1_000_000)
	case value >= 1_000:
		return fmt.Sprintf("%.0fK", value/1_000)
	default:
		return strings.TrimSuffix(strings.TrimSuffix(fmt.Sprintf("%.2f", value), "0"), ".0")
	}
}
