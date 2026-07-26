// Package objectstore provides the small S3-compatible object-store surface
// used by Shrike. It intentionally owns HTTP and signing rather than exposing
// an SDK client to API packages.
package objectstore

import (
	"bytes"
	"container/list"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/aws/signer/v4"
)

const (
	BackblazeRegion            = "eu-central-003"
	defaultCacheTTL            = time.Hour
	defaultCacheMaximumBytes   = 64 << 20
	defaultCacheMaximumEntries = 1024
	defaultMaximumBytes        = 8 << 20
)

type S3Options struct {
	Endpoint          string
	Bucket            string
	Region            string
	AccessKeyID       string
	SecretAccessKey   string
	HTTPClient        *http.Client
	CacheTTL          time.Duration
	CacheMaximumBytes int64
	MaximumBytes      int64

	now func() time.Time
}

type cachedObject struct {
	body      []byte
	expiresAt time.Time
	element   *list.Element
}

// S3Store is a path-style, Signature V4 S3 client with a small in-process
// read cache. Backblaze B2 implements this protocol and uses path-style bucket
// URLs at the endpoint already present in the TypeScript deployment.
type S3Store struct {
	endpoint     *url.URL
	bucket       string
	region       string
	client       *http.Client
	signer       *v4.Signer
	credential   aws.Credentials
	cacheTTL     time.Duration
	cacheMaximum int64
	maximum      int64
	now          func() time.Time

	cacheMu    sync.Mutex
	cacheBytes int64
	cache      map[string]*cachedObject
	cacheOrder *list.List
}

func NewS3Store(options S3Options) (*S3Store, error) {
	endpoint, err := url.Parse(strings.TrimSpace(options.Endpoint))
	if err != nil || endpoint.Host == "" ||
		(endpoint.Scheme != "https" && endpoint.Scheme != "http") ||
		endpoint.User != nil || endpoint.RawQuery != "" || endpoint.Fragment != "" {
		return nil, errors.New("object storage endpoint must be an HTTP(S) origin")
	}
	bucket := strings.TrimSpace(options.Bucket)
	if bucket == "" || strings.ContainsAny(bucket, `/\`) {
		return nil, errors.New("object storage bucket is invalid")
	}
	if options.AccessKeyID == "" || options.SecretAccessKey == "" {
		return nil, errors.New("object storage credentials are incomplete")
	}
	region := strings.TrimSpace(options.Region)
	if region == "" {
		return nil, errors.New("object storage region is required")
	}
	client := options.HTTPClient
	if client == nil {
		client = &http.Client{Timeout: 20 * time.Second}
	}
	cacheTTL := options.CacheTTL
	if cacheTTL == 0 {
		cacheTTL = defaultCacheTTL
	}
	if cacheTTL < 0 {
		return nil, errors.New("object storage cache TTL cannot be negative")
	}
	cacheMaximum := options.CacheMaximumBytes
	if cacheMaximum == 0 {
		cacheMaximum = defaultCacheMaximumBytes
	}
	if cacheMaximum < 1 {
		return nil, errors.New(
			"object storage cache maximum size must be positive",
		)
	}
	maximum := options.MaximumBytes
	if maximum == 0 {
		maximum = defaultMaximumBytes
	}
	if maximum < 1 {
		return nil, errors.New("object storage maximum size must be positive")
	}
	now := options.now
	if now == nil {
		now = time.Now
	}
	endpoint.Path = strings.TrimRight(endpoint.Path, "/")
	endpoint.RawPath = ""
	return &S3Store{
		endpoint: endpoint,
		bucket:   bucket,
		region:   region,
		client:   client,
		signer:   v4.NewSigner(),
		credential: aws.Credentials{
			AccessKeyID: options.AccessKeyID, SecretAccessKey: options.SecretAccessKey,
			Source: "shrike-static-object-storage",
		},
		cacheTTL:     cacheTTL,
		cacheMaximum: cacheMaximum,
		maximum:      maximum,
		now:          now,
		cache:        make(map[string]*cachedObject),
		cacheOrder:   list.New(),
	}, nil
}

func (s *S3Store) Put(
	ctx context.Context,
	key string,
	body []byte,
	contentType string,
) error {
	if int64(len(body)) > s.maximum {
		return fmt.Errorf("object %q exceeds the configured size limit", key)
	}
	response, err := s.request(ctx, http.MethodPut, key, body, contentType)
	if err != nil {
		return err
	}
	defer response.Body.Close()
	if response.StatusCode < 200 || response.StatusCode >= 300 {
		return objectStoreStatusError(response)
	}
	s.cachePut(key, body)
	return nil
}

func (s *S3Store) Get(ctx context.Context, key string) ([]byte, error) {
	if body, ok := s.cacheGet(key); ok {
		return body, nil
	}
	response, err := s.request(ctx, http.MethodGet, key, nil, "")
	if err != nil {
		return nil, err
	}
	defer response.Body.Close()
	if response.StatusCode == http.StatusNotFound {
		return nil, nil
	}
	if response.StatusCode < 200 || response.StatusCode >= 300 {
		return nil, objectStoreStatusError(response)
	}
	body, err := io.ReadAll(io.LimitReader(response.Body, s.maximum+1))
	if err != nil {
		return nil, fmt.Errorf("read object storage response: %w", err)
	}
	if int64(len(body)) > s.maximum {
		return nil, fmt.Errorf("object %q exceeds the configured size limit", key)
	}
	s.cachePut(key, body)
	return append([]byte(nil), body...), nil
}

func (s *S3Store) Delete(ctx context.Context, key string) error {
	response, err := s.request(ctx, http.MethodDelete, key, nil, "")
	if err != nil {
		return err
	}
	defer response.Body.Close()
	if response.StatusCode != http.StatusNotFound &&
		(response.StatusCode < 200 || response.StatusCode >= 300) {
		return objectStoreStatusError(response)
	}
	s.cacheDelete(key)
	return nil
}

func (s *S3Store) request(
	ctx context.Context,
	method, key string,
	body []byte,
	contentType string,
) (*http.Response, error) {
	objectURL, err := s.objectURL(key)
	if err != nil {
		return nil, err
	}
	var reader io.Reader
	if body != nil {
		reader = bytes.NewReader(body)
	}
	request, err := http.NewRequestWithContext(ctx, method, objectURL, reader)
	if err != nil {
		return nil, fmt.Errorf("build object storage request: %w", err)
	}
	if contentType != "" {
		request.Header.Set("Content-Type", contentType)
	}
	digest := sha256.Sum256(body)
	payloadHash := hex.EncodeToString(digest[:])
	request.Header.Set("X-Amz-Content-Sha256", payloadHash)
	if err := s.signer.SignHTTP(
		ctx,
		s.credential,
		request,
		payloadHash,
		"s3",
		s.region,
		s.now().UTC(),
	); err != nil {
		return nil, fmt.Errorf("sign object storage request: %w", err)
	}
	response, err := s.client.Do(request)
	if err != nil {
		return nil, fmt.Errorf("object storage request: %w", err)
	}
	return response, nil
}

func (s *S3Store) objectURL(key string) (string, error) {
	if key == "" || strings.HasPrefix(key, "/") ||
		strings.ContainsRune(key, '\\') {
		return "", errors.New("object storage key is invalid")
	}
	parts := strings.Split(key, "/")
	for _, part := range parts {
		if part == "" || part == "." || part == ".." {
			return "", errors.New("object storage key is invalid")
		}
	}
	value := *s.endpoint
	value.Path = s.endpoint.Path + "/" + s.bucket + "/" + strings.Join(parts, "/")
	value.RawPath = ""
	return value.String(), nil
}

func (s *S3Store) cacheGet(key string) ([]byte, bool) {
	if s.cacheTTL == 0 {
		return nil, false
	}
	now := s.now()
	s.cacheMu.Lock()
	defer s.cacheMu.Unlock()
	entry, ok := s.cache[key]
	if !ok {
		return nil, false
	}
	if !entry.expiresAt.After(now) {
		s.cacheRemoveLocked(key, entry)
		return nil, false
	}
	s.cacheOrder.MoveToFront(entry.element)
	return append([]byte(nil), entry.body...), true
}

func (s *S3Store) cachePut(key string, body []byte) {
	if s.cacheTTL == 0 {
		return
	}
	size := int64(len(body))
	var copied []byte
	if size <= s.cacheMaximum {
		copied = append([]byte(nil), body...)
	}
	now := s.now()

	s.cacheMu.Lock()
	defer s.cacheMu.Unlock()

	if current, ok := s.cache[key]; ok {
		s.cacheRemoveLocked(key, current)
	}
	if size > s.cacheMaximum {
		return
	}
	s.cacheRemoveExpiredLocked(now)

	element := s.cacheOrder.PushFront(key)
	s.cache[key] = &cachedObject{
		body:      copied,
		expiresAt: now.Add(s.cacheTTL),
		element:   element,
	}
	s.cacheBytes += size
	for s.cacheBytes > s.cacheMaximum ||
		len(s.cache) > defaultCacheMaximumEntries {
		oldest := s.cacheOrder.Back()
		if oldest == nil {
			break
		}
		oldestKey, _ := oldest.Value.(string)
		entry, ok := s.cache[oldestKey]
		if !ok {
			s.cacheOrder.Remove(oldest)
			continue
		}
		s.cacheRemoveLocked(oldestKey, entry)
	}
}

func (s *S3Store) cacheDelete(key string) {
	if s.cacheTTL == 0 {
		return
	}
	s.cacheMu.Lock()
	defer s.cacheMu.Unlock()
	if entry, ok := s.cache[key]; ok {
		s.cacheRemoveLocked(key, entry)
	}
}

func (s *S3Store) cacheRemoveExpiredLocked(now time.Time) {
	for key, entry := range s.cache {
		if !entry.expiresAt.After(now) {
			s.cacheRemoveLocked(key, entry)
		}
	}
}

func (s *S3Store) cacheRemoveLocked(
	key string,
	entry *cachedObject,
) {
	delete(s.cache, key)
	s.cacheOrder.Remove(entry.element)
	s.cacheBytes -= int64(len(entry.body))
	if s.cacheBytes < 0 {
		s.cacheBytes = 0
	}
}

func objectStoreStatusError(response *http.Response) error {
	body, _ := io.ReadAll(io.LimitReader(response.Body, 4<<10))
	message := strings.TrimSpace(string(body))
	if message == "" {
		message = http.StatusText(response.StatusCode)
	}
	return fmt.Errorf(
		"object storage returned %d: %s",
		response.StatusCode,
		message,
	)
}
