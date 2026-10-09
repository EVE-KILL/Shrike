package api

import (
	"encoding/json"
	"strings"
	"testing"
	"time"
)

func TestAnnouncementEventRetainsPublicOrderingTimestampWithoutAdminFields(t *testing.T) {
	now := time.Now().UTC().Truncate(time.Millisecond)
	raw, err := announcementEventPayload("new", map[string]any{"id": int64(12), "tier": int16(1), "title": "Ticker", "starts_at": now, "expires_at": now.Add(time.Hour), "created_at": now, "created_by": 42, "private_token": "secret"})
	if err != nil {
		t.Fatal(err)
	}
	var event struct {
		Type         string `json:"event_type"`
		Announcement Announcement
	}
	if json.Unmarshal(raw, &event) != nil || event.Type != "new" || event.Announcement.CreatedAt.IsZero() || !event.Announcement.CreatedAt.Equal(now) {
		t.Fatal("lifecycle event lost canonical ordering metadata")
	}
	if strings.Contains(string(raw), "created_by") || strings.Contains(string(raw), "private_token") {
		t.Fatal("event exposed admin metadata")
	}
}
