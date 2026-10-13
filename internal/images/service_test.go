package images

import (
	"bytes"
	"context"
	"image"
	"image/color"
	"net/http"
	"net/http/httptest"
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
	_, err := service.Type(context.Background(), 2_116_802_917, "icon", 64, "webp")
	status, _ := asStatus(err)
	if status != http.StatusServiceUnavailable {
		t.Fatalf("Type error = %v (status %d), want 503", err, status)
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

func TestHumaImageNamespaceServesConditionalWebP(t *testing.T) {
	source := solidPNG(t, 32, 32, colorValue(90, 80, 70))
	store := newMemoryStore()
	store.objects["types/42_64.png"] = &objectstore.Object{
		Body: source, Key: "types/42_64.png", ContentType: "image/png",
	}
	service := New(Options{Store: store, CacheBytes: 64 << 20})
	mux := chi.NewRouter()
	api := humachi.New(mux, huma.DefaultConfig("images", "test"))
	Register(api, service)

	path := "/images/types/42/icon?size=32"
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
	if api.OpenAPI().Paths["/images/types/{id}/{variant}"] == nil {
		t.Fatal("type image operation is missing from OpenAPI")
	}
	for _, category := range []string{"characters", "corporations", "alliances"} {
		if api.OpenAPI().Paths["/images/"+category+"/{id}/{variant}"] != nil {
			t.Fatalf("%s image route remains in OpenAPI", category)
		}
		request := httptest.NewRequest(http.MethodGet, "/images/"+category+"/42/logo", nil)
		recorder := httptest.NewRecorder()
		mux.ServeHTTP(recorder, request)
		if recorder.Code != http.StatusNotFound {
			t.Errorf("%s image route returned %d, want 404", category, recorder.Code)
		}
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
			"/images/types/7/icon",
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
