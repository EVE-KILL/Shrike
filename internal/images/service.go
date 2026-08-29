// Package images implements the durable EVE-KILL image surface.
package images

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"reflect"
	"sync"
	"time"

	"github.com/eve-kill/shrike/internal/objectstore"
	"golang.org/x/sync/singleflight"
)

const (
	entityFreshness       = 30 * 24 * time.Hour
	defaultMaximumObject  = 32 << 20
	responseCacheControl  = "public, max-age=86400, s-maxage=2592000, stale-while-revalidate=86400"
	immutableCacheControl = "public, max-age=31536000, immutable"
)

type ObjectStore interface {
	GetObject(context.Context, string) (*objectstore.Object, error)
	Stat(context.Context, string) (*objectstore.ObjectInfo, error)
	PutWithOptions(context.Context, string, []byte, objectstore.PutOptions) error
	Delete(context.Context, string) error
}

type EntityKind string

const (
	Character   EntityKind = "characters"
	Corporation EntityKind = "corporations"
	Alliance    EntityKind = "alliances"
)

type RefreshEnqueuer interface {
	EnqueueImageRefresh(context.Context, EntityKind, int64) error
}

type Options struct {
	Store       ObjectStore
	Social      SocialLoader
	HTTPClient  *http.Client
	UserAgent   string
	CacheBytes  int64
	CacheTTL    time.Duration
	Refresh     RefreshEnqueuer
	Now         func() time.Time
	UpstreamURL string
}

type Service struct {
	store     ObjectStore
	client    *http.Client
	userAgent string
	cache     *Cache
	refresh   RefreshEnqueuer
	now       func() time.Time
	upstream  string
	single    singleflight.Group
	maximum   int64
	typeData  typeDataCache

	defaultCharacterMu   sync.RWMutex
	defaultCharacterETag string

	social SocialLoader
}

type Result struct {
	Body         []byte
	ContentType  string
	ETag         string
	LastModified time.Time
	CacheControl string
}

type Error struct {
	Status  int
	Message string
	Err     error
}

func (e *Error) Error() string {
	if e.Err == nil {
		return e.Message
	}
	return e.Message + ": " + e.Err.Error()
}

func (e *Error) Unwrap() error { return e.Err }

func New(options Options) *Service {
	store := normalizeObjectStore(options.Store)
	client := options.HTTPClient
	if client == nil {
		client = &http.Client{Timeout: 20 * time.Second}
	}
	now := options.Now
	if now == nil {
		now = time.Now
	}
	upstream := options.UpstreamURL
	if upstream == "" {
		upstream = "https://images.evetech.net"
	}
	return &Service{
		store: store, client: client, userAgent: options.UserAgent,
		cache:   NewCache(options.CacheBytes, options.CacheTTL),
		refresh: options.Refresh, now: now, upstream: upstream,
		maximum: defaultMaximumObject, social: options.Social,
	}
}

// ObjectStore is an interface, so assigning a typed nil pointer to it produces
// a non-nil interface. Normalize that state at the service boundary: disabled
// image storage must return 503 rather than reaching a method with a nil
// receiver and panicking.
func normalizeObjectStore(store ObjectStore) ObjectStore {
	if store == nil {
		return nil
	}
	value := reflect.ValueOf(store)
	switch value.Kind() {
	case reflect.Chan, reflect.Func, reflect.Interface, reflect.Map,
		reflect.Pointer, reflect.Slice:
		if value.IsNil() {
			return nil
		}
	}
	return store
}

func (s *Service) Available() bool { return s != nil && s.store != nil }

func (s *Service) CacheStats() CacheStats {
	if s == nil {
		return CacheStats{}
	}
	return s.cache.Stats()
}

func unavailable() error {
	return &Error{
		Status:  http.StatusServiceUnavailable,
		Message: "Image storage is not configured",
	}
}

func statusError(status int, message string, err error) error {
	return &Error{Status: status, Message: message, Err: err}
}

func asStatus(err error) (int, string) {
	var imageErr *Error
	if errors.As(err, &imageErr) {
		return imageErr.Status, imageErr.Message
	}
	return http.StatusInternalServerError, "Internal server error"
}

func validateID(id int64) error {
	if id <= 0 {
		return statusError(http.StatusBadRequest, "Invalid image ID", nil)
	}
	return nil
}

func (s *Service) cacheResult(key string, load func() (Result, error)) (Result, error) {
	if result, ok := s.cache.Get(key); ok {
		return result, nil
	}
	value, err, _ := s.single.Do(key, func() (any, error) {
		if result, ok := s.cache.Get(key); ok {
			return result, nil
		}
		result, loadErr := load()
		if loadErr != nil {
			return Result{}, loadErr
		}
		s.cache.Put(key, result)
		return result, nil
	})
	if err != nil {
		return Result{}, err
	}
	result, ok := value.(Result)
	if !ok {
		return Result{}, fmt.Errorf("image singleflight returned %T", value)
	}
	return result, nil
}
