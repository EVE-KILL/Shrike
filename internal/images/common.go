package images

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"image"
	"net/http"
	"slices"
	"time"
)

func extensionForFormat(format string) string {
	if format == "jpeg" {
		return "jpg"
	}
	return format
}

func newResult(body []byte, contentType string, modified time.Time) Result {
	sum := sha256.Sum256(body)
	return Result{
		Body: body, ContentType: contentType,
		ETag:         `"` + hex.EncodeToString(sum[:]) + `"`,
		LastModified: modified, CacheControl: responseCacheControl,
	}
}

func validateSize(size int, mapAsset bool) error {
	if size == 0 {
		return nil
	}
	if mapAsset {
		if size == 32 || size == 64 || size == 128 || size == 256 || size == 512 || size == 1024 {
			return nil
		}
		return statusError(
			http.StatusBadRequest,
			"Image size must be 32, 64, 128, 256, 512, or 1024",
			nil,
		)
	}
	if slices.Contains([]int{8, 16, 32, 64, 128, 256, 512, 1024}, size) {
		return nil
	}
	return statusError(
		http.StatusBadRequest,
		"Image size must be one of 8, 16, 32, 64, 128, 256, 512, or 1024",
		nil,
	)
}

func detectImageFormat(body []byte) (string, error) {
	_, format, err := image.DecodeConfig(bytes.NewReader(body))
	return normalizeFormat(format), err
}

func normalizeFormat(format string) string {
	if format == "jpg" {
		return "jpeg"
	}
	return format
}
