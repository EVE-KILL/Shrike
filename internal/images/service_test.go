package images

import (
	"bytes"
	"context"
	"encoding/json"
	"image"
	"image/color"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/danielgtaylor/huma/v2"
	"github.com/danielgtaylor/huma/v2/adapters/humachi"
	"github.com/eve-kill/shrike/internal/objectstore"
	"github.com/go-chi/chi/v5"
)

type memoryStore struct {
	mu      sync.Mutex
	objects map[string]*objectstore.Object
	gets    []string
	puts    []string
}

func newMemoryStore() *memoryStore {
	return &memoryStore{objects: make(map[string]*objectstore.Object)}
}

func TestNewTreatsTypedNilObjectStoreAsUnavailable(t *testing.T) {
	var store *objectstore.S3Store
	service := New(Options{Store: store})

	if service.Available() {
		t.Fatal("service is available with a typed nil object store")
	}
	_, err := service.Entity(
		context.Background(),
		Character,
		2_116_802_917,
		64,
		"webp",
	)
	status, _ := asStatus(err)
	if status != http.StatusServiceUnavailable {
		t.Fatalf("Entity error = %v (status %d), want 503", err, status)
	}
}

func (s *memoryStore) GetObject(
	_ context.Context,
	key string,
) (*objectstore.Object, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.gets = append(s.gets, key)
	object := s.objects[key]
	if object == nil {
		return nil, nil
	}
	copy := *object
	copy.Body = append([]byte(nil), object.Body...)
	return &copy, nil
}

func TestTypeUsesDirectImageExportCollectionKey(t *testing.T) {
	icon := solidPNG(t, 64, 64, colorValue(30, 100, 180))
	store := newMemoryStore()
	store.objects["types/648_64.png"] = &objectstore.Object{
		Body: icon,
		Key:  "types/648_64.png", ContentType: "image/png",
	}
	service := New(Options{Store: store})
	result, err := service.Type(
		context.Background(),
		648,
		"icon",
		0,
		"",
	)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(result.Body, icon) || result.ContentType != "image/png" {
		t.Fatalf("result = type %q, %d bytes", result.ContentType, len(result.Body))
	}
	if len(store.gets) != 1 || store.gets[0] != "types/648_64.png" {
		t.Fatalf("object reads = %v", store.gets)
	}
}

func TestTypeExportSourceKeysMatchCollectionNames(t *testing.T) {
	for _, test := range []struct {
		variant string
		want    string
	}{
		{variant: "icon", want: "types/983_64.png"},
		{variant: "bp", want: "types/983_64.png"},
		{variant: "bpc", want: "types/983_bpc_64.png"},
		{variant: "render", want: "types/983_512.jpg"},
	} {
		if got := typeExportSourceKey(983, test.variant); got != test.want {
			t.Errorf("%s key = %q, want %q", test.variant, got, test.want)
		}
	}
}

func TestOldCharacterStoresSizedWebPVariant(t *testing.T) {
	source, _, err := transformImage(
		solidPNG(t, 256, 256, colorValue(20, 120, 210)),
		"png",
		transformSpec{Format: "jpeg"},
	)
	if err != nil {
		t.Fatal(err)
	}
	store := newMemoryStore()
	store.objects["oldcharacters/0/7/7_256.jpg"] = &objectstore.Object{
		Body:        source,
		Key:         "oldcharacters/0/7/7_256.jpg",
		ContentType: "image/jpeg",
	}
	service := New(Options{Store: store})
	result, err := service.OldCharacter(
		context.Background(),
		7,
		64,
		"webp",
	)
	if err != nil {
		t.Fatal(err)
	}
	config, format, err := image.DecodeConfig(bytes.NewReader(result.Body))
	if err != nil {
		t.Fatal(err)
	}
	if format != "webp" || config.Width != 64 || config.Height != 64 {
		t.Fatalf(
			"variant = %s %dx%d",
			format,
			config.Width,
			config.Height,
		)
	}
	if len(store.puts) != 1 ||
		!strings.HasPrefix(store.puts[0], "oldcharacters/derived/") {
		t.Fatalf("stored variants = %v", store.puts)
	}

	// A new service instance proves the derived object is durable rather than
	// only present in the in-process LRU.
	service = New(Options{Store: store})
	if _, err := service.OldCharacter(
		context.Background(),
		7,
		64,
		"webp",
	); err != nil {
		t.Fatal(err)
	}
	if len(store.puts) != 1 {
		t.Fatalf("variant was encoded again: %v", store.puts)
	}
}

func (s *memoryStore) Stat(
	ctx context.Context,
	key string,
) (*objectstore.ObjectInfo, error) {
	object, err := s.GetObject(ctx, key)
	if err != nil || object == nil {
		return nil, err
	}
	info := object.ObjectInfo
	return &info, nil
}

func (s *memoryStore) PutWithOptions(
	_ context.Context,
	key string,
	body []byte,
	options objectstore.PutOptions,
) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.puts = append(s.puts, key)
	s.objects[key] = &objectstore.Object{
		Body: append([]byte(nil), body...),
		Key:  key, Size: int64(len(body)), ContentType: options.ContentType,
		CacheControl: options.CacheControl,
		LastModified: time.Date(2026, 7, 27, 12, 0, 0, 0, time.UTC),
		Metadata:     options.Metadata,
	}
	return nil
}

func (s *memoryStore) Delete(_ context.Context, key string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.objects, key)
	return nil
}

type refreshRecorder struct {
	mu    sync.Mutex
	calls []string
}

func (r *refreshRecorder) EnqueueImageRefresh(
	_ context.Context,
	kind EntityKind,
	id int64,
) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.calls = append(r.calls, string(kind)+"/"+strconv.FormatInt(id, 10))
	return nil
}

func TestEntityColdMissPersistsSourceVariantAndUsesFinalCache(t *testing.T) {
	source := solidPNG(t, 64, 32, colorValue(10, 80, 190))
	var upstreamCalls int
	upstream := httptest.NewServer(http.HandlerFunc(func(
		w http.ResponseWriter,
		r *http.Request,
	) {
		upstreamCalls++
		w.Header().Set("Content-Type", "image/png")
		w.Header().Set("ETag", `"ccp-1"`)
		_, _ = w.Write(source)
	}))
	defer upstream.Close()

	store := newMemoryStore()
	now := time.Date(2026, 7, 27, 12, 0, 0, 0, time.UTC)
	service := New(Options{
		Store: store, UpstreamURL: upstream.URL, CacheBytes: 64 << 20,
		Now: func() time.Time { return now },
	})
	first, err := service.Entity(
		context.Background(),
		Corporation,
		42,
		32,
		"webp",
	)
	if err != nil {
		t.Fatal(err)
	}
	if first.ContentType != "image/webp" || first.ETag == "" {
		t.Fatalf("first result = %+v", first)
	}
	second, err := service.Entity(
		context.Background(),
		Corporation,
		42,
		32,
		"webp",
	)
	if err != nil {
		t.Fatal(err)
	}
	if first.ETag != second.ETag || upstreamCalls != 1 {
		t.Fatalf("cache result ETags %q/%q, upstream calls %d",
			first.ETag, second.ETag, upstreamCalls)
	}
	pointerObject := store.objects["entities/corporations/42/current.json"]
	if pointerObject == nil {
		t.Fatal("entity pointer was not stored")
	}
	var pointer sourcePointer
	if err := json.Unmarshal(pointerObject.Body, &pointer); err != nil {
		t.Fatal(err)
	}
	if store.objects[pointer.ObjectKey] == nil {
		t.Fatalf("source object %q was not stored", pointer.ObjectKey)
	}
	variant := entityVariantKey(&pointer, entityRequest{
		Kind: Corporation, ID: 42, Size: 32, Format: "webp",
	})
	if store.objects[variant] == nil {
		t.Fatalf("variant %q was not stored", variant)
	}
}

func TestCharacterDefaultPortraitUsesLegacyArchiveSource(t *testing.T) {
	defaultPortrait := solidPNG(t, 64, 64, colorValue(80, 80, 80))
	legacyPortrait, _, err := transformImage(
		solidPNG(t, 64, 64, colorValue(20, 120, 210)),
		"png",
		transformSpec{Format: "jpeg"},
	)
	if err != nil {
		t.Fatal(err)
	}
	var headCalls int
	upstream := httptest.NewServer(http.HandlerFunc(func(
		w http.ResponseWriter,
		r *http.Request,
	) {
		w.Header().Set("ETag", `"default-character"`)
		if r.Method == http.MethodHead {
			headCalls++
			return
		}
		_, _ = w.Write(defaultPortrait)
	}))
	defer upstream.Close()

	store := newMemoryStore()
	store.objects["oldcharacters/missing_256.jpg"] = &objectstore.Object{
		Body: legacyPortrait,
		Key:  "oldcharacters/missing_256.jpg", ContentType: "image/jpeg",
	}
	service := New(Options{Store: store, UpstreamURL: upstream.URL})
	result, err := service.Entity(
		context.Background(),
		Character,
		90000001,
		0,
		"jpeg",
	)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(result.Body, legacyPortrait) {
		t.Fatal("default CCP portrait was served instead of the legacy fallback")
	}
	if headCalls != 1 {
		t.Fatalf("default portrait HEAD calls = %d, want 1", headCalls)
	}
}

func TestStaleEntityServesAndEnqueuesRefresh(t *testing.T) {
	store := newMemoryStore()
	now := time.Date(2026, 7, 27, 12, 0, 0, 0, time.UTC)
	body := solidPNG(t, 16, 16, colorValue(40, 50, 60))
	pointer := sourcePointer{
		Version: 1, Digest: "digest",
		ObjectKey:   "entities/alliances/7/digest/original.png",
		ContentType: "image/png",
		CheckedAt:   now.Add(-31 * 24 * time.Hour),
	}
	pointerBody, _ := json.Marshal(pointer)
	store.objects[entityPointerKey(Alliance, 7)] = &objectstore.Object{Body: pointerBody}
	store.objects[pointer.ObjectKey] = &objectstore.Object{
		Body: body,
		Key:  pointer.ObjectKey, ContentType: "image/png",
	}
	refresh := &refreshRecorder{}
	service := New(Options{
		Store: store, CacheBytes: 64 << 20, Refresh: refresh,
		Now: func() time.Time { return now },
	})
	if _, err := service.Entity(
		context.Background(), Alliance, 7, 0, "",
	); err != nil {
		t.Fatal(err)
	}
	if len(refresh.calls) != 1 || refresh.calls[0] != "alliances/7" {
		t.Fatalf("refresh calls = %v", refresh.calls)
	}
}

func TestHumaImageNamespaceServesConditionalWebP(t *testing.T) {
	source := solidPNG(t, 32, 32, colorValue(90, 80, 70))
	upstream := httptest.NewServer(http.HandlerFunc(func(
		w http.ResponseWriter,
		_ *http.Request,
	) {
		w.Header().Set("Content-Type", "image/png")
		_, _ = w.Write(source)
	}))
	defer upstream.Close()
	service := New(Options{
		Store: newMemoryStore(), UpstreamURL: upstream.URL,
		CacheBytes: 64 << 20,
	})
	mux := chi.NewRouter()
	api := humachi.New(mux, huma.DefaultConfig("images", "test"))
	Register(api, service)

	path := "/images/corporations/42/logo?size=16"
	request := httptest.NewRequest(http.MethodGet, path, nil)
	request.Header.Set("Accept", "image/webp")
	recorder := httptest.NewRecorder()
	mux.ServeHTTP(recorder, request)
	if recorder.Code != http.StatusOK ||
		recorder.Header().Get("Content-Type") != "image/webp" {
		t.Fatalf("response = %d %s: %s",
			recorder.Code, recorder.Header().Get("Content-Type"),
			recorder.Body.String())
	}
	etag := recorder.Header().Get("ETag")
	request = httptest.NewRequest(http.MethodGet, path, nil)
	request.Header.Set("Accept", "image/webp")
	request.Header.Set("If-None-Match", etag)
	recorder = httptest.NewRecorder()
	mux.ServeHTTP(recorder, request)
	if recorder.Code != http.StatusNotModified || recorder.Body.Len() != 0 {
		t.Fatalf("conditional response = %d, body %d",
			recorder.Code, recorder.Body.Len())
	}

	request = httptest.NewRequest(
		http.MethodGet,
		path+"&format=source",
		nil,
	)
	request.Header.Set("Accept", "image/webp")
	recorder = httptest.NewRecorder()
	mux.ServeHTTP(recorder, request)
	if recorder.Code != http.StatusOK ||
		recorder.Header().Get("Content-Type") != "image/png" {
		t.Fatalf(
			"source override response = %d %s: %s",
			recorder.Code,
			recorder.Header().Get("Content-Type"),
			recorder.Body.String(),
		)
	}

	request = httptest.NewRequest(
		http.MethodGet,
		path+"&format=gif",
		nil,
	)
	recorder = httptest.NewRecorder()
	mux.ServeHTTP(recorder, request)
	if recorder.Code != http.StatusBadRequest ||
		!strings.Contains(recorder.Body.String(), "auto, source, or webp") {
		t.Fatalf(
			"invalid format response = %d: %s",
			recorder.Code,
			recorder.Body.String(),
		)
	}
	if api.OpenAPI().Paths["/images/corporations/{id}/{variant}"] == nil {
		t.Fatal("image operation is missing from OpenAPI")
	}
}

func TestUnconfiguredImageServiceReturns503(t *testing.T) {
	mux := chi.NewRouter()
	api := humachi.New(mux, huma.DefaultConfig("images", "test"))
	Register(api, New(Options{}))
	recorder := httptest.NewRecorder()
	mux.ServeHTTP(
		recorder,
		httptest.NewRequest(
			http.MethodGet,
			"/images/characters/7/portrait",
			nil,
		),
	)
	if recorder.Code != http.StatusServiceUnavailable ||
		!strings.Contains(recorder.Body.String(), "not configured") {
		t.Fatalf("response = %d %s", recorder.Code, recorder.Body.String())
	}
}

func colorValue(r, g, b byte) color.NRGBA {
	return color.NRGBA{R: r, G: g, B: b, A: 255}
}
