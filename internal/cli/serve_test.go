package cli

import (
	"strings"
	"testing"

	"github.com/eve-kill/shrike/internal/config"
)

func TestNewDomainAssetStorageAllowsDisabledLocalConfiguration(t *testing.T) {
	store, err := newDomainAssetStorage(&config.Config{})
	if err != nil {
		t.Fatalf("newDomainAssetStorage: %v", err)
	}
	if store != nil {
		t.Fatalf("store = %T, want nil when B2 is disabled", store)
	}
}

func TestNewDomainAssetStorageRejectsPartialConfiguration(t *testing.T) {
	_, err := newDomainAssetStorage(&config.Config{
		B2Endpoint: "https://s3.example.test",
	})
	if err == nil {
		t.Fatal("newDomainAssetStorage should reject partial B2 configuration")
	}
	for _, name := range []string{
		"B2_ENDPOINT", "B2_MEDIA_BUCKET", "B2_KEY_ID", "B2_APP_KEY",
	} {
		if !strings.Contains(err.Error(), name) {
			t.Errorf("error %q does not name %s", err, name)
		}
	}
}

func TestNewDomainAssetStorageBuildsCompleteConfiguration(t *testing.T) {
	store, err := newDomainAssetStorage(&config.Config{
		B2Endpoint:    "https://s3.example.test",
		B2MediaBucket: "evekill-media",
		B2KeyID:       "key-id",
		B2AppKey:      "application-key",
	})
	if err != nil {
		t.Fatalf("newDomainAssetStorage: %v", err)
	}
	if store == nil {
		t.Fatal("store is nil for complete B2 configuration")
	}
}

func TestNewImageStorageAllowsDisabledLocalConfiguration(t *testing.T) {
	store, err := newImageStorage(&config.Config{})
	if err != nil {
		t.Fatalf("newImageStorage: %v", err)
	}
	if store != nil {
		t.Fatalf("store = %T, want nil when B2 images are disabled", store)
	}
}

func TestNewImageStorageRejectsPartialConfiguration(t *testing.T) {
	_, err := newImageStorage(&config.Config{
		B2ImagesBucket: "evekill-images",
	})
	if err == nil {
		t.Fatal("newImageStorage should reject partial B2 configuration")
	}
	for _, name := range []string{
		"B2_ENDPOINT", "B2_IMAGES_BUCKET", "B2_KEY_ID", "B2_APP_KEY",
	} {
		if !strings.Contains(err.Error(), name) {
			t.Errorf("error %q does not name %s", err, name)
		}
	}
}

func TestNewImageStorageBuildsUncachedStore(t *testing.T) {
	store, err := newImageStorage(&config.Config{
		B2Endpoint:     "https://s3.example.test",
		B2ImagesBucket: "evekill-images",
		B2KeyID:        "key-id",
		B2AppKey:       "application-key",
	})
	if err != nil {
		t.Fatalf("newImageStorage: %v", err)
	}
	if store == nil {
		t.Fatal("store is nil for complete image B2 configuration")
	}
}
