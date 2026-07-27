package images

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net/http"
	"path"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/eve-kill/shrike/internal/objectstore"
)

type typeMetadata map[string]map[string]string

type typeDataCache struct {
	mu             sync.RWMutex
	dustLoaded     bool
	dust           typeMetadata
	overlaysLoaded bool
	overlays       map[int64]string
}

func (s *Service) Type(
	ctx context.Context,
	id int64,
	variant string,
	size int,
	format string,
) (Result, error) {
	if !s.Available() {
		return Result{}, unavailable()
	}
	if err := validateID(id); err != nil {
		return Result{}, err
	}
	switch variant {
	case "icon", "render", "overlayrender", "bp", "bpc":
	default:
		return Result{}, statusError(
			http.StatusBadRequest,
			"Type image must be icon, render, overlayrender, bp, or bpc",
			nil,
		)
	}
	if err := validateSize(size, false); err != nil {
		return Result{}, err
	}
	if format != "" && format != "webp" {
		return Result{}, statusError(
			http.StatusBadRequest,
			"Type images only support source format or WebP",
			nil,
		)
	}
	cacheKey := fmt.Sprintf("type/%d/%s/%d/%s", id, variant, size, format)
	return s.cacheResult(cacheKey, func() (Result, error) {
		return s.loadType(ctx, id, variant, size, format)
	})
}

func (s *Service) loadType(
	ctx context.Context,
	id int64,
	variant string,
	size int,
	format string,
) (Result, error) {
	sourceVariant := variant
	if sourceVariant == "overlayrender" {
		sourceVariant = "render"
	}

	var (
		source      *objectstore.Object
		sourceKey   string
		sourceStamp time.Time
	)
	sourceKey = typeExportSourceKey(id, sourceVariant)
	source, err := s.store.GetObject(ctx, sourceKey)
	if err != nil {
		return Result{}, fmt.Errorf("read type image: %w", err)
	}
	if source == nil {
		dust, dustErr := s.dustTypeMetadata(ctx)
		if dustErr != nil {
			return Result{}, dustErr
		}
		entry := dust[strconv.FormatInt(id, 10)]
		if filename := entry[sourceVariant]; filename != "" {
			if !safeAssetName(filename) {
				return Result{}, fmt.Errorf(
					"Dust 514 type metadata contains unsafe filename %q",
					filename,
				)
			}
			sourceKey = "types/dust514/" + filename
			source, err = s.store.GetObject(ctx, sourceKey)
			if err != nil {
				return Result{}, fmt.Errorf("read Dust 514 type image: %w", err)
			}
		}
	}
	if source == nil {
		source, err = s.loadUpstreamType(ctx, id, sourceVariant)
		if err != nil {
			return Result{}, err
		}
		sourceKey = source.Key
	}
	sourceStamp = source.LastModified
	if sourceStamp.IsZero() {
		sourceStamp = s.now()
	}
	sourceFormat := formatForContentType(source.ContentType)
	if sourceFormat == "" {
		sourceFormat, err = detectImageFormat(source.Body)
		if err != nil {
			return Result{}, statusError(
				http.StatusBadGateway,
				"Stored type image is invalid",
				err,
			)
		}
	}
	targetFormat := format
	if targetFormat == "" {
		targetFormat = sourceFormat
	}

	var overlay []byte
	var overlayType string
	if variant == "overlayrender" {
		overlays, overlayErr := s.typeOverlays(ctx)
		if overlayErr != nil {
			return Result{}, overlayErr
		}
		overlayType = overlays[id]
	}
	if variant == "overlayrender" && overlayType != "" {
		object, getErr := s.store.GetObject(
			ctx,
			"types/overlays/"+overlayType+".png",
		)
		if getErr != nil {
			return Result{}, fmt.Errorf("read type overlay: %w", getErr)
		}
		if object != nil {
			overlay = object.Body
		}
	}

	sum := sha256.Sum256(source.Body)
	sourceDigest := hex.EncodeToString(sum[:])
	derivedKey := fmt.Sprintf(
		"types/derived/%s/%d/%s/%d.%s",
		sourceDigest,
		id,
		variant,
		size,
		extensionForFormat(targetFormat),
	)
	if size != 0 || targetFormat != sourceFormat || overlay != nil {
		derived, getErr := s.store.GetObject(ctx, derivedKey)
		if getErr != nil {
			return Result{}, fmt.Errorf("read type variant: %w", getErr)
		}
		if derived != nil {
			contentType := derived.ContentType
			if contentType == "" {
				contentType = contentTypeForFormat(targetFormat)
			}
			return newResult(derived.Body, contentType, sourceStamp), nil
		}
	}

	body, contentType, err := transformImage(
		source.Body,
		sourceFormat,
		transformSpec{Size: size, Format: targetFormat, Overlay: overlay},
	)
	if err != nil {
		return Result{}, statusError(
			http.StatusBadGateway,
			"Stored type image is invalid",
			err,
		)
	}
	if size != 0 || targetFormat != sourceFormat || overlay != nil {
		if err := s.store.PutWithOptions(
			context.WithoutCancel(ctx),
			derivedKey,
			body,
			objectstore.PutOptions{
				ContentType: contentType, CacheControl: immutableCacheControl,
				Metadata: map[string]string{
					"source-key": sourceKey,
				},
			},
		); err != nil {
			return Result{}, fmt.Errorf("store type variant: %w", err)
		}
	}
	return newResult(body, contentType, sourceStamp), nil
}

func typeExportSourceKey(id int64, variant string) string {
	switch variant {
	case "render":
		return fmt.Sprintf("types/%d_512.jpg", id)
	case "bpc":
		return fmt.Sprintf("types/%d_bpc_64.png", id)
	default:
		// Image Export Collection uses the ordinary 64px icon for both the
		// "icon" and blueprint-original ("bp") variants.
		return fmt.Sprintf("types/%d_64.png", id)
	}
}

func (s *Service) loadUpstreamType(
	ctx context.Context,
	id int64,
	variant string,
) (*objectstore.Object, error) {
	key := fmt.Sprintf("types/upstream/%d/%s/original", id, variant)
	if object, err := s.store.GetObject(ctx, key); err != nil || object != nil {
		return object, err
	}
	url := fmt.Sprintf("%s/types/%d/%s", s.upstream, id, variant)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}
	if s.userAgent != "" {
		req.Header.Set("User-Agent", s.userAgent)
	}
	response, err := s.client.Do(req)
	if err != nil {
		return nil, statusError(
			http.StatusBadGateway,
			"Type image origin is unavailable",
			err,
		)
	}
	defer response.Body.Close()
	if response.StatusCode == http.StatusNotFound {
		return nil, statusError(http.StatusNotFound, "Type image not found", nil)
	}
	if response.StatusCode < 200 || response.StatusCode >= 300 {
		return nil, statusError(
			http.StatusBadGateway,
			fmt.Sprintf("Type image origin returned %d", response.StatusCode),
			nil,
		)
	}
	body, err := readBounded(response.Body, s.maximum)
	if err != nil {
		return nil, statusError(
			http.StatusBadGateway,
			"Type image origin response is invalid",
			err,
		)
	}
	format, err := detectImageFormat(body)
	if err != nil {
		return nil, statusError(
			http.StatusBadGateway,
			"Type image origin returned invalid data",
			err,
		)
	}
	contentType := contentTypeForFormat(format)
	if err := s.store.PutWithOptions(
		context.WithoutCancel(ctx),
		key,
		body,
		objectstore.PutOptions{
			ContentType: contentType, CacheControl: immutableCacheControl,
		},
	); err != nil {
		return nil, fmt.Errorf("store upstream type image: %w", err)
	}
	return &objectstore.Object{
		Body: body,
		ObjectInfo: objectstore.ObjectInfo{
			Key: key, Size: int64(len(body)), ContentType: contentType,
			LastModified: s.now(),
		},
	}, nil
}

func (s *Service) dustTypeMetadata(ctx context.Context) (typeMetadata, error) {
	s.typeData.mu.RLock()
	if s.typeData.dustLoaded {
		dust := s.typeData.dust
		s.typeData.mu.RUnlock()
		return dust, nil
	}
	s.typeData.mu.RUnlock()

	s.typeData.mu.Lock()
	defer s.typeData.mu.Unlock()
	if !s.typeData.dustLoaded {
		s.typeData.dust = typeMetadata{}
		if object, getErr := s.store.GetObject(
			ctx,
			"types/dust514/service_metadata.json",
		); getErr != nil {
			return nil, getErr
		} else if object != nil {
			if err := json.Unmarshal(object.Body, &s.typeData.dust); err != nil {
				return nil, fmt.Errorf("decode Dust 514 metadata: %w", err)
			}
		}
		s.typeData.dustLoaded = true
	}
	return s.typeData.dust, nil
}

func (s *Service) typeOverlays(ctx context.Context) (map[int64]string, error) {
	s.typeData.mu.RLock()
	if s.typeData.overlaysLoaded {
		overlays := s.typeData.overlays
		s.typeData.mu.RUnlock()
		return overlays, nil
	}
	s.typeData.mu.RUnlock()

	s.typeData.mu.Lock()
	defer s.typeData.mu.Unlock()
	if !s.typeData.overlaysLoaded {
		s.typeData.overlays = make(map[int64]string)
		if object, getErr := s.store.GetObject(
			ctx,
			"types/overlays/ids.json",
		); getErr != nil {
			return nil, getErr
		} else if object != nil {
			var groups map[string][]int64
			if err := json.Unmarshal(object.Body, &groups); err != nil {
				return nil, fmt.Errorf("decode type overlays: %w", err)
			}
			for overlay, ids := range groups {
				for _, id := range ids {
					s.typeData.overlays[id] = overlay
				}
			}
		}
		s.typeData.overlaysLoaded = true
	}
	return s.typeData.overlays, nil
}

func (s *Service) Static(
	ctx context.Context,
	category string,
	name string,
	size int,
	format string,
) (Result, error) {
	if !s.Available() {
		return Result{}, unavailable()
	}
	switch category {
	case "regions", "systems", "constellations":
		if _, err := strconv.ParseInt(name, 10, 64); err != nil {
			return Result{}, statusError(http.StatusBadRequest, "Invalid image ID", nil)
		}
	case "ui":
		if !safeAssetName(name) {
			return Result{}, statusError(http.StatusBadRequest, "Invalid UI image name", nil)
		}
	default:
		return Result{}, statusError(http.StatusNotFound, "Unknown image category", nil)
	}
	if err := validateSize(size, true); err != nil {
		return Result{}, err
	}
	if format == "" {
		format = "png"
	}
	if format != "png" && format != "webp" {
		return Result{}, statusError(
			http.StatusBadRequest,
			"Static images only support PNG or WebP",
			nil,
		)
	}
	cacheKey := fmt.Sprintf("static/%s/%s/%d/%s", category, name, size, format)
	return s.cacheResult(cacheKey, func() (Result, error) {
		key := fmt.Sprintf("static/%s/%s.png", category, name)
		if size == 32 {
			small := fmt.Sprintf("static/%s/%s_32.png", category, name)
			if object, err := s.store.GetObject(ctx, small); err != nil {
				return Result{}, err
			} else if object != nil {
				key = small
			}
		}
		source, err := s.store.GetObject(ctx, key)
		if err != nil {
			return Result{}, err
		}
		if source == nil {
			return Result{}, statusError(http.StatusNotFound, "Image not found", nil)
		}
		resize := size
		if strings.HasSuffix(key, "_32.png") {
			resize = 0
		}
		body, contentType, err := transformImage(
			source.Body,
			"png",
			transformSpec{Size: resize, Format: format},
		)
		if err != nil {
			return Result{}, statusError(
				http.StatusBadGateway,
				"Stored static image is invalid",
				err,
			)
		}
		modified := source.LastModified
		if modified.IsZero() {
			modified = s.now()
		}
		return newResult(body, contentType, modified), nil
	})
}

func (s *Service) OldCharacter(
	ctx context.Context,
	id int64,
	format string,
) (Result, error) {
	if !s.Available() {
		return Result{}, unavailable()
	}
	if err := validateID(id); err != nil {
		return Result{}, err
	}
	if format == "" {
		format = "jpeg"
	}
	if format != "jpeg" && format != "webp" {
		return Result{}, statusError(
			http.StatusBadRequest,
			"Old character portraits only support JPEG or WebP",
			nil,
		)
	}
	cacheKey := fmt.Sprintf("oldcharacter/%d/%s", id, format)
	return s.cacheResult(cacheKey, func() (Result, error) {
		source, err := s.loadOldCharacterSource(ctx, id, true)
		if err != nil {
			return Result{}, err
		}
		if source == nil {
			return Result{}, statusError(
				http.StatusNotFound,
				"Old character image not found",
				nil,
			)
		}
		body, contentType, err := transformImage(
			source.Body,
			"jpeg",
			transformSpec{Format: format},
		)
		if err != nil {
			return Result{}, statusError(
				http.StatusBadGateway,
				"Stored old character image is invalid",
				err,
			)
		}
		modified := source.LastModified
		if modified.IsZero() {
			modified = s.now()
		}
		return newResult(body, contentType, modified), nil
	})
}

func (s *Service) loadOldCharacterSource(
	ctx context.Context,
	id int64,
	fallback bool,
) (*objectstore.Object, error) {
	raw := strconv.FormatInt(id, 10)
	padded := strings.Repeat("0", max(0, 2-len(raw))) + raw
	key := fmt.Sprintf(
		"oldcharacters/%s/%s/%s_256.jpg",
		padded[len(padded)-2:len(padded)-1],
		padded[len(padded)-1:],
		raw,
	)
	source, err := s.store.GetObject(ctx, key)
	if err != nil || source != nil || !fallback {
		return source, err
	}
	return s.store.GetObject(ctx, "oldcharacters/missing_256.jpg")
}

func safeAssetName(name string) bool {
	return name != "" &&
		name == path.Base(name) &&
		name != "." &&
		name != ".." &&
		!strings.ContainsAny(name, `/\`) &&
		!strings.ContainsAny(name, "\r\n\x00")
}
