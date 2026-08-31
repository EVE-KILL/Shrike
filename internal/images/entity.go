package images

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"image"
	"io"
	"net/http"
	"slices"
	"strconv"
	"time"

	"github.com/eve-kill/shrike/internal/objectstore"
)

type sourcePointer struct {
	Version     int       `json:"version"`
	Digest      string    `json:"digest"`
	ObjectKey   string    `json:"object_key"`
	ContentType string    `json:"content_type"`
	OriginETag  string    `json:"origin_etag,omitempty"`
	CheckedAt   time.Time `json:"checked_at"`
}

type entityRequest struct {
	Kind   EntityKind
	ID     int64
	Size   int
	Format string
}

func (s *Service) Entity(
	ctx context.Context,
	kind EntityKind,
	id int64,
	size int,
	format string,
) (Result, error) {
	if !s.Available() {
		return Result{}, unavailable()
	}
	if err := validateID(id); err != nil {
		return Result{}, err
	}
	spec, ok := entitySpecs[kind]
	if !ok {
		return Result{}, statusError(http.StatusNotFound, "Unknown image kind", nil)
	}
	if err := validateSize(size, false); err != nil {
		return Result{}, err
	}
	if format == "" {
		format = spec.format
	}
	if format != spec.format && format != "webp" {
		return Result{}, statusError(
			http.StatusBadRequest,
			"Unsupported image format",
			nil,
		)
	}
	request := entityRequest{Kind: kind, ID: id, Size: size, Format: format}
	cacheKey := fmt.Sprintf(
		"entity/%s/%d/%d/%s",
		kind, id, size, format,
	)
	return s.cacheResult(cacheKey, func() (Result, error) {
		return s.loadEntity(ctx, request)
	})
}

var entitySpecs = map[EntityKind]struct {
	path   string
	format string
}{
	Character:   {path: "portrait", format: "jpeg"},
	Corporation: {path: "logo", format: "png"},
	Alliance:    {path: "logo", format: "png"},
}

func (s *Service) loadEntity(
	ctx context.Context,
	request entityRequest,
) (Result, error) {
	pointer, err := s.loadPointer(ctx, entityPointerKey(request.Kind, request.ID))
	if err != nil {
		return Result{}, err
	}
	if pointer == nil {
		pointer, err = s.fetchEntity(ctx, request.Kind, request.ID, nil)
		if err != nil {
			return Result{}, err
		}
	} else if s.now().Sub(pointer.CheckedAt) >= entityFreshness &&
		s.refresh != nil {
		// River uniqueness collapses concurrent/still-cached requests. Enqueue
		// inline so request cancellation cannot leave an untracked goroutine.
		_ = s.refresh.EnqueueImageRefresh(ctx, request.Kind, request.ID)
	}

	source, err := s.store.GetObject(ctx, pointer.ObjectKey)
	if err != nil {
		return Result{}, fmt.Errorf("read entity source: %w", err)
	}
	if source == nil {
		// A pointer without its content is an interrupted import. Repair it
		// from the origin instead of making the missing object permanent.
		pointer, err = s.fetchEntity(ctx, request.Kind, request.ID, nil)
		if err != nil {
			return Result{}, err
		}
		source, err = s.store.GetObject(ctx, pointer.ObjectKey)
		if err != nil || source == nil {
			return Result{}, fmt.Errorf("read repaired entity source: %w", err)
		}
	}

	sourceFormat := formatForContentType(pointer.ContentType)
	variantKey := entityVariantKey(pointer, request)
	if request.Size != 0 || request.Format != sourceFormat {
		variant, getErr := s.store.GetObject(ctx, variantKey)
		if getErr != nil {
			return Result{}, fmt.Errorf("read entity variant: %w", getErr)
		}
		if variant != nil {
			contentType := variant.ContentType
			if contentType == "" {
				contentType = contentTypeForFormat(request.Format)
			}
			modified := variant.LastModified
			if modified.IsZero() {
				modified = pointer.CheckedAt
			}
			return newResult(variant.Body, contentType, modified), nil
		}
	}
	body, contentType, err := transformImage(source.Body, sourceFormat, transformSpec{
		Size: request.Size, Format: request.Format,
	})
	if err != nil {
		return Result{}, statusError(
			http.StatusBadGateway,
			"Stored image is invalid",
			err,
		)
	}
	if request.Size != 0 || request.Format != sourceFormat {
		if putErr := s.store.PutWithOptions(
			context.WithoutCancel(ctx),
			variantKey,
			body,
			objectstore.PutOptions{
				ContentType: contentType, CacheControl: immutableCacheControl,
			},
		); putErr != nil {
			return Result{}, fmt.Errorf("store entity variant: %w", putErr)
		}
	}
	return newResult(body, contentType, pointer.CheckedAt), nil
}

func (s *Service) RefreshEntity(
	ctx context.Context,
	kind EntityKind,
	id int64,
) error {
	if !s.Available() {
		return unavailable()
	}
	if err := validateID(id); err != nil {
		return err
	}
	pointer, err := s.loadPointer(ctx, entityPointerKey(kind, id))
	if err != nil {
		return err
	}
	updated, err := s.fetchEntity(ctx, kind, id, pointer)
	if err != nil {
		return err
	}
	if pointer == nil || updated.Digest != pointer.Digest {
		s.cache.RemovePrefix(fmt.Sprintf("entity/%s/%d/", kind, id))
	}
	return nil
}

func (s *Service) fetchEntity(
	ctx context.Context,
	kind EntityKind,
	id int64,
	current *sourcePointer,
) (*sourcePointer, error) {
	spec, ok := entitySpecs[kind]
	if !ok {
		return nil, statusError(http.StatusNotFound, "Unknown image kind", nil)
	}
	url := fmt.Sprintf("%s/%s/%d/%s", s.upstream, kind, id, spec.path)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}
	if s.userAgent != "" {
		req.Header.Set("User-Agent", s.userAgent)
	}
	if current != nil && current.OriginETag != "" {
		req.Header.Set("If-None-Match", current.OriginETag)
	}
	response, err := s.client.Do(req)
	if err != nil {
		return nil, statusError(
			http.StatusBadGateway,
			"Image origin is unavailable",
			err,
		)
	}
	defer response.Body.Close()

	if response.StatusCode == http.StatusNotModified && current != nil {
		current.CheckedAt = s.now().UTC()
		if err := s.savePointer(
			context.WithoutCancel(ctx),
			entityPointerKey(kind, id),
			current,
		); err != nil {
			return nil, err
		}
		return current, nil
	}
	if response.StatusCode == http.StatusNotFound {
		return nil, statusError(http.StatusNotFound, "Image not found", nil)
	}
	if response.StatusCode < 200 || response.StatusCode >= 300 {
		return nil, statusError(
			http.StatusBadGateway,
			"Image origin returned "+strconv.Itoa(response.StatusCode),
			nil,
		)
	}
	body, err := io.ReadAll(io.LimitReader(response.Body, s.maximum+1))
	if err != nil {
		return nil, fmt.Errorf("read image origin: %w", err)
	}
	if int64(len(body)) > s.maximum {
		return nil, statusError(
			http.StatusBadGateway,
			"Image origin response is too large",
			nil,
		)
	}
	if kind == Character {
		defaultETag, defaultErr := s.loadDefaultCharacterETag(ctx)
		if defaultErr == nil && defaultETag != "" &&
			response.Header.Get("ETag") == defaultETag {
			old, oldErr := s.loadOldCharacterSource(ctx, id, true)
			if oldErr != nil {
				return nil, fmt.Errorf("read legacy character image: %w", oldErr)
			}
			if old != nil {
				body = old.Body
			}
		}
	}
	format, decodeErr := detectImageFormat(body)
	if decodeErr != nil {
		return nil, statusError(
			http.StatusBadGateway,
			"Image origin returned invalid data",
			decodeErr,
		)
	}
	contentType := contentTypeForFormat(format)
	digestBytes := sha256.Sum256(body)
	digest := hex.EncodeToString(digestBytes[:])
	objectKey := fmt.Sprintf(
		"entities/%s/%d/%s/original.%s",
		kind,
		id,
		digest,
		extensionForFormat(format),
	)
	if err := s.store.PutWithOptions(
		context.WithoutCancel(ctx),
		objectKey,
		body,
		objectstore.PutOptions{
			ContentType: contentType, CacheControl: immutableCacheControl,
		},
	); err != nil {
		return nil, fmt.Errorf("store entity source: %w", err)
	}
	pointer := &sourcePointer{
		Version: 1, Digest: digest, ObjectKey: objectKey,
		ContentType: contentType,
		OriginETag:  response.Header.Get("ETag"),
		CheckedAt:   s.now().UTC(),
	}
	if err := s.savePointer(
		context.WithoutCancel(ctx),
		entityPointerKey(kind, id),
		pointer,
	); err != nil {
		return nil, err
	}
	return pointer, nil
}

func (s *Service) loadDefaultCharacterETag(
	ctx context.Context,
) (string, error) {
	s.defaultCharacterMu.RLock()
	cached := s.defaultCharacterETag
	s.defaultCharacterMu.RUnlock()
	if cached != "" {
		return cached, nil
	}
	value, err, _ := s.single.Do("origin/default-character-etag", func() (any, error) {
		s.defaultCharacterMu.RLock()
		cached := s.defaultCharacterETag
		s.defaultCharacterMu.RUnlock()
		if cached != "" {
			return cached, nil
		}
		req, requestErr := http.NewRequestWithContext(
			ctx,
			http.MethodHead,
			s.upstream+"/characters/1/portrait",
			nil,
		)
		if requestErr != nil {
			return "", requestErr
		}
		if s.userAgent != "" {
			req.Header.Set("User-Agent", s.userAgent)
		}
		response, requestErr := s.client.Do(req)
		if requestErr != nil {
			return "", requestErr
		}
		defer response.Body.Close()
		if response.StatusCode < 200 || response.StatusCode >= 300 {
			return "", fmt.Errorf(
				"default character origin returned %d",
				response.StatusCode,
			)
		}
		etag := response.Header.Get("ETag")
		if etag != "" {
			s.defaultCharacterMu.Lock()
			s.defaultCharacterETag = etag
			s.defaultCharacterMu.Unlock()
		}
		return etag, nil
	})
	if err != nil {
		return "", err
	}
	etag, _ := value.(string)
	return etag, nil
}

func (s *Service) loadPointer(
	ctx context.Context,
	key string,
) (*sourcePointer, error) {
	object, err := s.store.GetObject(ctx, key)
	if err != nil || object == nil {
		return nil, err
	}
	var pointer sourcePointer
	if err := json.Unmarshal(object.Body, &pointer); err != nil {
		return nil, fmt.Errorf("decode image pointer %q: %w", key, err)
	}
	if pointer.Version != 1 || pointer.Digest == "" ||
		pointer.ObjectKey == "" || pointer.ContentType == "" {
		return nil, fmt.Errorf("image pointer %q is incomplete", key)
	}
	return &pointer, nil
}

func (s *Service) savePointer(
	ctx context.Context,
	key string,
	pointer *sourcePointer,
) error {
	body, err := json.Marshal(pointer)
	if err != nil {
		return err
	}
	if err := s.store.PutWithOptions(ctx, key, body, objectstore.PutOptions{
		ContentType:  "application/json",
		CacheControl: "no-cache",
	}); err != nil {
		return fmt.Errorf("store image pointer: %w", err)
	}
	return nil
}

func entityPointerKey(kind EntityKind, id int64) string {
	return fmt.Sprintf("entities/%s/%d/current.json", kind, id)
}

func entityVariantKey(pointer *sourcePointer, request entityRequest) string {
	size := "original"
	if request.Size > 0 {
		size = strconv.Itoa(request.Size)
	}
	return fmt.Sprintf(
		"entities/%s/%d/%s/%s.%s",
		request.Kind,
		request.ID,
		pointer.Digest,
		size,
		extensionForFormat(request.Format),
	)
}

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
		if size == 32 || size == 64 || size == 128 || size == 512 || size == 1024 {
			return nil
		}
		return statusError(
			http.StatusBadRequest,
			"Image size must be 32, 64, 128, 512, or 1024",
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
