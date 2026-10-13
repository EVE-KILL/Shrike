package frontend

import (
	"net/url"
	"time"

	"github.com/eve-kill/shrike/internal/api"
)

// Reuse the public post contract rather than maintaining another post model.
type adminBlogPost struct {
	api.BlogPost
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

type adminBlogData struct {
	Posts  []adminBlogPost `json:"posts"`
	Status string          `json:"-"`
	Failed bool            `json:"-"`
}

func adminBlogStatus(query url.Values) string {
	switch status := query.Get("status"); status {
	case "draft", "published", "archived":
		return status
	default:
		return "all"
	}
}

func adminBlogStatusLabel(status int16) (string, string) {
	switch status {
	case 0:
		return "Draft", "bg-gray-500/20 text-gray-400 border-gray-500/30"
	case 1:
		return "Published", "bg-green-500/20 text-green-300 border-green-500/30"
	case 2:
		return "Archived", "bg-yellow-500/20 text-yellow-400 border-yellow-500/30"
	default:
		return "Unknown", ""
	}
}

func adminBlogDate(t time.Time) string {
	if t.IsZero() {
		return "—"
	}
	return t.UTC().Format("2006-01-02 15:04")
}
