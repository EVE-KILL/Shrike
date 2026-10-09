package api

import (
	"encoding/base64"
	"fmt"
	"strconv"
	"strings"
	"time"
)

// BlogCursor follows the listing's publication-time and ID ordering. ID zero
// represents a legacy timestamp-only cursor, which remains accepted.
type BlogCursor struct {
	PublishedAt time.Time
	ID          int64
}

func ParseBlogCursor(raw string) (BlogCursor, error) {
	if len(raw) <= 256 && strings.HasPrefix(raw, "v1:") {
		decoded, err := base64.RawURLEncoding.DecodeString(strings.TrimPrefix(raw, "v1:"))
		if err == nil {
			stamp, id, found := strings.Cut(string(decoded), "/")
			published, timeErr := time.Parse(time.RFC3339Nano, stamp)
			value, idErr := strconv.ParseInt(id, 10, 64)
			if found && timeErr == nil && idErr == nil && value > 0 {
				return BlogCursor{PublishedAt: published, ID: value}, nil
			}
		}
	} else if len(raw) <= 256 {
		if published, ok := parseContentTime(raw); ok {
			return BlogCursor{PublishedAt: published}, nil
		}
	}
	return BlogCursor{}, fmt.Errorf("invalid blog cursor")
}

func (cursor BlogCursor) String() string {
	stamp := cursor.PublishedAt.UTC().Format(time.RFC3339Nano)
	if cursor.ID == 0 {
		return stamp
	}
	return "v1:" + base64.RawURLEncoding.EncodeToString([]byte(stamp+"/"+strconv.FormatInt(cursor.ID, 10)))
}

func blogRowCursor(row map[string]any) string {
	published, hasTime := timeFrom(row["published_at"])
	id, hasID := int64Value(row["id"])
	if !hasTime || !hasID || id <= 0 {
		return ""
	}
	return (BlogCursor{PublishedAt: published, ID: id}).String()
}
