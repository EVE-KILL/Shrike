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
