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

type typeBundlePointer struct {
	Version     int       `json:"version"`
	Release     string    `json:"release"`
	Digest      string    `json:"digest"`
	MetadataKey string    `json:"metadata_key"`
	UpdatedAt   time.Time `json:"updated_at"`
}

type typeMetadata map[string]map[string]string

type typeDataCache struct {
	mu             sync.RWMutex
	version        string
	metadata       typeMetadata
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
	metadata, dust, overlays, version, err := s.typeMetadata(ctx)
	if err != nil {
		return Result{}, err
	}
	idKey := strconv.FormatInt(id, 10)
	entry := metadata[idKey]
	prefix := "types/blobs/"
	if entry == nil {
		entry = dust[idKey]
		prefix = "types/dust514/"
	}
	sourceVariant := variant
	if sourceVariant == "overlayrender" {
		sourceVariant = "render"
	}

	var (
		source      *objectstore.Object
		sourceKey   string
		sourceStamp time.Time
	)
	if filename := entry[sourceVariant]; filename != "" {
		if !safeAssetName(filename) {
			return Result{}, fmt.Errorf(
				"type metadata contains unsafe filename %q",
				filename,
			)
		}
		sourceKey = prefix + filename
		source, err = s.store.GetObject(ctx, sourceKey)
		if err != nil {
			return Result{}, fmt.Errorf("read type image: %w", err)
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
	overlayType := overlays[id]
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
					"bundle":     version,
				},
			},
		); err != nil {
			return Result{}, fmt.Errorf("store type variant: %w", err)
		}
	}
	return newResult(body, contentType, sourceStamp), nil
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

func (s *Service) typeMetadata(
	ctx context.Context,
) (typeMetadata, typeMetadata, map[int64]string, string, error) {
	pointerObject, err := s.store.GetObject(ctx, "types/current.json")
	if err != nil {
		return nil, nil, nil, "", err
	}
	var pointer typeBundlePointer
	if pointerObject != nil {
		if err := json.Unmarshal(pointerObject.Body, &pointer); err != nil {
			return nil, nil, nil, "", fmt.Errorf("decode type bundle pointer: %w", err)
		}
	}
	version := pointer.Digest
	if version == "" {
		version = "upstream-only"
	}

	s.typeData.mu.RLock()
	ready := s.typeData.version == version &&
		s.typeData.dustLoaded &&
		s.typeData.overlaysLoaded
	if ready {
		metadata := s.typeData.metadata
		dust := s.typeData.dust
		overlays := s.typeData.overlays
		s.typeData.mu.RUnlock()
		return metadata, dust, overlays, version, nil
	}
	s.typeData.mu.RUnlock()

	s.typeData.mu.Lock()
	defer s.typeData.mu.Unlock()
	if s.typeData.version != version {
		s.typeData.metadata = typeMetadata{}
		if pointer.MetadataKey != "" {
			object, getErr := s.store.GetObject(ctx, pointer.MetadataKey)
			if getErr != nil {
				return nil, nil, nil, "", getErr
			}
			if object == nil {
				return nil, nil, nil, "", fmt.Errorf(
					"type metadata %q is missing",
					pointer.MetadataKey,
				)
			}
			if err := json.Unmarshal(object.Body, &s.typeData.metadata); err != nil {
				return nil, nil, nil, "", fmt.Errorf("decode type metadata: %w", err)
			}
		}
		s.typeData.version = version
	}
	if !s.typeData.dustLoaded {
		s.typeData.dust = typeMetadata{}
		if object, getErr := s.store.GetObject(
			ctx,
			"types/dust514/service_metadata.json",
		); getErr != nil {
			return nil, nil, nil, "", getErr
		} else if object != nil {
			if err := json.Unmarshal(object.Body, &s.typeData.dust); err != nil {
				return nil, nil, nil, "", fmt.Errorf("decode Dust 514 metadata: %w", err)
			}
		}
		s.typeData.dustLoaded = true
	}
	if !s.typeData.overlaysLoaded {
		s.typeData.overlays = make(map[int64]string)
		if object, getErr := s.store.GetObject(
			ctx,
			"types/overlays/ids.json",
		); getErr != nil {
			return nil, nil, nil, "", getErr
		} else if object != nil {
			var groups map[string][]int64
			if err := json.Unmarshal(object.Body, &groups); err != nil {
				return nil, nil, nil, "", fmt.Errorf("decode type overlays: %w", err)
			}
			for overlay, ids := range groups {
				for _, id := range ids {
					s.typeData.overlays[id] = overlay
				}
			}
		}
		s.typeData.overlaysLoaded = true
	}
	return s.typeData.metadata, s.typeData.dust,
		s.typeData.overlays, version, nil
}

func (s *Service) ServiceMetadata(ctx context.Context) ([]byte, error) {
	if !s.Available() {
		return nil, unavailable()
	}
	pointer, err := s.store.GetObject(ctx, "types/current.json")
	if err != nil {
		return nil, err
	}
	if pointer == nil {
		return nil, statusError(http.StatusNotFound, "Type metadata not installed", nil)
	}
	var current typeBundlePointer
	if err := json.Unmarshal(pointer.Body, &current); err != nil {
		return nil, err
	}
	object, err := s.store.GetObject(ctx, current.MetadataKey)
	if err != nil {
		return nil, err
	}
	if object == nil {
		return nil, statusError(http.StatusNotFound, "Type metadata not installed", nil)
	}
	return object.Body, nil
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
