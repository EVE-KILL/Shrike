package images

import (
	"bytes"
	"fmt"
	"image"
	"image/draw"
	"image/jpeg"
	"image/png"

	"github.com/deepteams/webp"
	xdraw "golang.org/x/image/draw"
)

const maximumDecodedPixels = 40_000_000

type transformSpec struct {
	Size    int
	Format  string
	Overlay []byte
}

func transformImage(
	body []byte,
	sourceFormat string,
	spec transformSpec,
) ([]byte, string, error) {
	if spec.Size == 0 && spec.Overlay == nil &&
		(spec.Format == "" || spec.Format == sourceFormat) {
		return body, contentTypeForFormat(sourceFormat), nil
	}

	config, _, err := image.DecodeConfig(bytes.NewReader(body))
	if err != nil {
		return nil, "", fmt.Errorf("decode image configuration: %w", err)
	}
	if config.Width <= 0 || config.Height <= 0 ||
		config.Width > maximumDecodedPixels/config.Height {
		return nil, "", fmt.Errorf(
			"image dimensions %dx%d exceed the safety limit",
			config.Width,
			config.Height,
		)
	}
	decoded, _, err := image.Decode(bytes.NewReader(body))
	if err != nil {
		return nil, "", fmt.Errorf("decode image: %w", err)
	}

	output := decoded
	if spec.Size > 0 &&
		(decoded.Bounds().Dx() > spec.Size || decoded.Bounds().Dy() > spec.Size) {
		width, height := fitInside(
			decoded.Bounds().Dx(),
			decoded.Bounds().Dy(),
			spec.Size,
		)
		resized := image.NewNRGBA(image.Rect(0, 0, width, height))
		xdraw.CatmullRom.Scale(
			resized,
			resized.Bounds(),
			decoded,
			decoded.Bounds(),
			draw.Over,
			nil,
		)
		output = resized
	}

	if spec.Overlay != nil {
		overlay, _, decodeErr := image.Decode(bytes.NewReader(spec.Overlay))
		if decodeErr != nil {
			return nil, "", fmt.Errorf("decode image overlay: %w", decodeErr)
		}
		bounds := output.Bounds()
		canvas := image.NewNRGBA(image.Rect(0, 0, bounds.Dx(), bounds.Dy()))
		draw.Draw(canvas, canvas.Bounds(), output, bounds.Min, draw.Src)
		overlaySize := max(4, bounds.Dx()/5)
		resizedOverlay := image.NewNRGBA(image.Rect(0, 0, overlaySize, overlaySize))
		xdraw.CatmullRom.Scale(
			resizedOverlay,
			resizedOverlay.Bounds(),
			overlay,
			overlay.Bounds(),
			draw.Over,
			nil,
		)
		draw.Draw(
			canvas,
			image.Rect(0, 0, overlaySize, overlaySize),
			resizedOverlay,
			image.Point{},
			draw.Over,
		)
		output = canvas
	}

	format := spec.Format
	if format == "" {
		format = sourceFormat
	}
	var encoded bytes.Buffer
	switch format {
	case "jpeg":
		err = jpeg.Encode(&encoded, output, &jpeg.Options{Quality: 90})
	case "png":
		encoder := png.Encoder{CompressionLevel: png.DefaultCompression}
		err = encoder.Encode(&encoded, output)
	case "webp":
		err = webp.Encode(
			&encoded,
			output,
			webp.OptionsForPreset(webp.PresetPicture, 82),
		)
	default:
		return nil, "", fmt.Errorf("unsupported output image format %q", format)
	}
	if err != nil {
		return nil, "", fmt.Errorf("encode %s image: %w", format, err)
	}
	return encoded.Bytes(), contentTypeForFormat(format), nil
}

func fitInside(width, height, maximum int) (int, int) {
	if width <= maximum && height <= maximum {
		return width, height
	}
	if width >= height {
		return maximum, max(1, height*maximum/width)
	}
	return max(1, width*maximum/height), maximum
}

func contentTypeForFormat(format string) string {
	switch format {
	case "jpeg", "jpg":
		return "image/jpeg"
	case "png":
		return "image/png"
	case "webp":
		return "image/webp"
	default:
		return "application/octet-stream"
	}
}

func formatForContentType(contentType string) string {
	switch contentType {
	case "image/jpeg", "image/jpg":
		return "jpeg"
	case "image/png":
		return "png"
	case "image/webp":
		return "webp"
	default:
		return ""
	}
}
