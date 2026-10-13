package api

import (
	"encoding/base64"
	"strings"
	"testing"
	"time"
)

func TestBlogCursorPreservesPublicationPrecisionAndID(t *testing.T) {
	want := BlogCursor{PublishedAt: time.Date(2026, 10, 4, 9, 30, 0, 123456000, time.FixedZone("offset", 7200)), ID: 91234}
	got, err := ParseBlogCursor(want.String())
	if err != nil || !got.PublishedAt.Equal(want.PublishedAt) || got.ID != want.ID {
		t.Fatalf("cursor round trip: %v, %v", got, err)
	}
	legacy, err := ParseBlogCursor("2026-10-04T07:30:00.123Z")
	if err != nil || legacy.ID != 0 || !legacy.PublishedAt.Equal(want.PublishedAt.Truncate(time.Millisecond)) {
		t.Fatalf("legacy cursor: %v, %v", legacy, err)
	}
	for _, raw := range []string{"", "not-a-date", "v2:abc", "v1:@@@", strings.Repeat("x", 257),
		"v1:" + base64.RawURLEncoding.EncodeToString([]byte("2026-10-04T07:30:00Z/0")),
		"v1:" + base64.RawURLEncoding.EncodeToString([]byte("2026-10-04T07:30:00Z/-1")),
		"v1:" + base64.RawURLEncoding.EncodeToString([]byte("bad/1")),
	} {
		if _, err := ParseBlogCursor(raw); err == nil {
			t.Errorf("accepted invalid cursor %q", raw)
		}
	}
}
