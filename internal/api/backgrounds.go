package api

import (
	"context"
	"html"
	"io"
	"net/http"
	"regexp"
	"time"

	"github.com/danielgtaylor/huma/v2"
)

const (
	redditBackgroundFeedURL = "https://old.reddit.com/r/eveporn/hot/.rss?limit=100"
	redditBackgroundMaxBody = 10 << 20
)

var (
	redditEntryPattern   = regexp.MustCompile(`(?s)<entry>(.*?)</entry>`)
	redditTitlePattern   = regexp.MustCompile(`(?s)<title>(.*?)</title>`)
	redditLinkPattern    = regexp.MustCompile(`<link href="([^"]+)"`)
	redditContentPattern = regexp.MustCompile(`(?s)<content[^>]*>(.*?)</content>`)
	redditImagePattern   = regexp.MustCompile(
		`(?i)href="(https://i\.redd\.it/[^"]+?\.(?:jpg|jpeg|png|webp)|https://i\.imgur\.com/[^"]+?\.(?:jpg|jpeg|png|webp))"`,
	)
)

type redditBackground struct {
	URL       string `json:"url"`
	Title     string `json:"title"`
	Source    string `json:"source"`
	Subreddit string `json:"subreddit"`
}

func registerBackgroundRoutes(a huma.API, opts Options) {
	client := &http.Client{Timeout: 15 * time.Second}
	registerLegacy(a, huma.Operation{
		OperationID: "backgrounds-reddit",
		Method:      http.MethodGet,
		Path:        "/backgrounds/reddit",
		Summary:     "EVE community backgrounds",
		Description: "Returns direct images from the cached r/eveporn Atom feed.",
		Tags:        []string{"backgrounds"},
	}, routeJSONCache(
		opts,
		30*time.Minute,
		"public, max-age=300, s-maxage=1800, stale-while-revalidate=1800",
		redditBackgroundHandler(client),
	))
}

func redditBackgroundHandler(client *http.Client) legacyHandler {
	return func(ctx context.Context, _ *legacyRequest) (legacyPayload, error) {
		request, err := http.NewRequestWithContext(
			ctx, http.MethodGet, redditBackgroundFeedURL, nil,
		)
		if err != nil {
			return redditBackgroundPayload(nil), nil
		}
		request.Header.Set(
			"User-Agent",
			"eve-kill.com background rotator (contact: admin@eve-kill.com)",
		)
		response, err := client.Do(request)
		if err != nil {
			return redditBackgroundPayload(nil), nil
		}
		defer response.Body.Close()
		if response.StatusCode < 200 || response.StatusCode >= 300 {
			return redditBackgroundPayload(nil), nil
		}
		body, err := io.ReadAll(io.LimitReader(
			response.Body, redditBackgroundMaxBody+1,
		))
		if err != nil || len(body) > redditBackgroundMaxBody {
			return redditBackgroundPayload(nil), nil
		}
		return redditBackgroundPayload(parseRedditBackgrounds(string(body))), nil
	}
}

func redditBackgroundPayload(images []redditBackground) legacyPayload {
	if images == nil {
		images = []redditBackground{}
	}
	return jsonPayload(map[string]any{"images": images})
}

func parseRedditBackgrounds(feed string) []redditBackground {
	entries := redditEntryPattern.FindAllStringSubmatch(feed, -1)
	images := make([]redditBackground, 0, len(entries))
	for _, match := range entries {
		entry := match[1]
		title := html.UnescapeString(firstRegexGroup(redditTitlePattern, entry))
		source := firstRegexGroup(redditLinkPattern, entry)
		content := html.UnescapeString(firstRegexGroup(redditContentPattern, entry))
		imageURL := firstRegexGroup(redditImagePattern, content)
		if imageURL == "" {
			continue
		}
		images = append(images, redditBackground{
			URL: imageURL, Title: title, Source: source, Subreddit: "eveporn",
		})
	}
	return images
}

func firstRegexGroup(pattern *regexp.Regexp, value string) string {
	match := pattern.FindStringSubmatch(value)
	if len(match) < 2 {
		return ""
	}
	return match[1]
}
