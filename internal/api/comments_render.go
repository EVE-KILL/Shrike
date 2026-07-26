package api

import (
	"bytes"
	"context"
	"crypto/sha1"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	stdhtml "html"
	"io"
	"net/http"
	"net/url"
	"os"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/redis/go-redis/v9"
	"github.com/russross/blackfriday/v2"
	xhtml "golang.org/x/net/html"
	"golang.org/x/net/html/atom"
)

var commentMarkdownProfile = contentMarkdownProfile{
	allowedTags: valuesSet(
		"a", "b", "i", "em", "strong", "u", "s", "del",
		"code", "pre", "br", "hr", "p", "blockquote", "ul",
		"ol", "li", "h1", "h2", "h3", "h4", "h5", "h6",
		"table", "thead", "tbody", "tr", "th", "td",
		"img", "video", "source", "iframe", "div", "span",
	),
	allowedAttrs: valuesSet(
		"href", "title", "alt", "src", "class", "lang",
		"data-emoji", "data-id", "controls", "preload", "loading",
		"allowfullscreen", "frameborder", "allow", "autoplay", "loop",
		"muted", "playsinline", "poster", "target", "rel", "colspan",
		"rowspan",
	),
}

var (
	commentAnchorPattern = regexp.MustCompile(
		`<a href="([^"]+)">[^<]*</a>`,
	)
	commentYouTubePattern = regexp.MustCompile(
		`(?i)^(?:https?://)?(?:www\.)?(?:youtube\.com/(?:watch\?v=|embed/|v/|shorts/)|youtu\.be/)([a-zA-Z0-9_-]{11})`,
	)
	commentImagePattern = regexp.MustCompile(
		`(?i)\.(?:jpg|jpeg|png|gif|webp|svg)(?:\?.*)?$`,
	)
	commentVideoPattern = regexp.MustCompile(
		`(?i)\.(?:mp4|webm|ogg|mov)(?:\?.*)?$`,
	)
	commentResolvedVideoPattern = regexp.MustCompile(
		`(?i)\.(?:mp4|webm)(?:\?.*)?$`,
	)
	commentResolvedImagePattern = regexp.MustCompile(
		`(?i)\.(?:jpg|jpeg|png|gif|webp)(?:\?.*)?$`,
	)
	commentEmojiPattern = regexp.MustCompile(
		`:([a-zA-Z0-9_+\-]{1,32}):`,
	)
	commentRichHostPattern = regexp.MustCompile(
		`(?i)^(?:www\.|i\.|m\.|media[0-9]?\.)?(?:imgur\.com|tenor\.com|giphy\.com|gph\.is)$`,
	)
)

type resolvedCommentMedia struct {
	Type   string `json:"type"`
	Source string `json:"src"`
	Poster string `json:"poster,omitempty"`
	Width  int    `json:"width,omitempty"`
	Height int    `json:"height,omitempty"`
}

type commentRenderer struct {
	cache  *redis.Client
	client *http.Client
}

func newCommentRenderer(
	cache *redis.Client,
	client *http.Client,
) *commentRenderer {
	if client == nil {
		client = &http.Client{Timeout: 5 * time.Second}
	}
	return &commentRenderer{cache: cache, client: client}
}

func (r *commentRenderer) Render(
	ctx context.Context,
	source string,
) string {
	if strings.TrimSpace(source) == "" {
		return ""
	}
	renderer := blackfriday.NewHTMLRenderer(
		blackfriday.HTMLRendererParameters{
			Flags: blackfriday.SkipHTML,
		},
	)
	raw := string(blackfriday.Run(
		[]byte(source),
		blackfriday.WithRenderer(renderer),
		blackfriday.WithExtensions(blackfriday.CommonExtensions),
	))
	resolved := map[string]*resolvedCommentMedia{}
	for _, href := range extractCommentRichMediaURLs(raw) {
		resolved[href] = r.resolveMedia(ctx, href)
	}
	embedded := commentAnchorPattern.ReplaceAllStringFunc(
		raw,
		func(match string) string {
			parts := commentAnchorPattern.FindStringSubmatch(match)
			if len(parts) != 2 {
				return match
			}
			escapedHref := parts[1]
			href := stdhtml.UnescapeString(escapedHref)
			if found := commentYouTubePattern.FindStringSubmatch(href); len(found) == 2 {
				id := stdhtml.EscapeString(found[1])
				return `<div class="embed embed-youtube"><iframe src="https://www.youtube.com/embed/` +
					id +
					`" frameborder="0" allow="accelerometer; clipboard-write; encrypted-media; gyroscope; picture-in-picture" allowfullscreen loading="lazy"></iframe></div>`
			}
			if isCommentRichMediaURL(href) {
				if media := resolved[href]; media != nil {
					return renderResolvedCommentMedia(media)
				}
			}
			safeHref := stdhtml.EscapeString(href)
			if commentImagePattern.MatchString(href) {
				return `<div class="embed embed-media"><img src="` +
					safeHref +
					`" alt="image" loading="lazy"></div>`
			}
			if commentVideoPattern.MatchString(href) {
				return `<div class="embed embed-media"><video controls loop muted autoplay playsinline preload="metadata" src="` +
					safeHref + `"></video></div>`
			}
			return match
		},
	)
	withEmoji := commentEmojiPattern.ReplaceAllString(
		embedded,
		`<span class="emoji" data-emoji="$1">:$1:</span>`,
	)
	safe := sanitizeCommentHTML(withEmoji)
	return hardenCommentAnchors(stripBadCommentIframes(safe))
}

func extractCommentRichMediaURLs(raw string) []string {
	seen := map[string]bool{}
	result := []string{}
	for _, match := range commentAnchorPattern.FindAllStringSubmatch(raw, -1) {
		if len(match) != 2 {
			continue
		}
		href := stdhtml.UnescapeString(match[1])
		if isCommentRichMediaURL(href) && !seen[href] {
			seen[href] = true
			result = append(result, href)
		}
	}
	return result
}

func isCommentRichMediaURL(raw string) bool {
	parsed, err := url.Parse(raw)
	if err != nil ||
		(parsed.Scheme != "http" && parsed.Scheme != "https") {
		return false
	}
	return commentRichHostPattern.MatchString(
		strings.ToLower(parsed.Hostname()),
	)
}

func renderResolvedCommentMedia(media *resolvedCommentMedia) string {
	if media == nil {
		return ""
	}
	source := stdhtml.EscapeString(media.Source)
	if media.Type == "video" {
		poster := ""
		if media.Poster != "" {
			poster = ` poster="` +
				stdhtml.EscapeString(media.Poster) + `"`
		}
		return `<div class="embed embed-media"><video controls loop muted autoplay playsinline preload="metadata"` +
			poster + ` src="` + source + `"></video></div>`
	}
	return `<div class="embed embed-media"><img src="` +
		source + `" alt="" loading="lazy"></div>`
}

func sanitizeCommentHTML(raw string) string {
	contextNode := &xhtml.Node{
		Type: xhtml.ElementNode, Data: "div", DataAtom: atom.Div,
	}
	nodes, err := xhtml.ParseFragment(
		strings.NewReader(raw),
		contextNode,
	)
	if err != nil {
		return ""
	}
	var output bytes.Buffer
	for _, node := range nodes {
		appendContentNode(&output, node, commentMarkdownProfile)
	}
	return output.String()
}

func stripBadCommentIframes(raw string) string {
	contextNode := &xhtml.Node{
		Type: xhtml.ElementNode, Data: "div", DataAtom: atom.Div,
	}
	nodes, err := xhtml.ParseFragment(
		strings.NewReader(raw),
		contextNode,
	)
	if err != nil {
		return ""
	}
	var prune func(*xhtml.Node)
	prune = func(parent *xhtml.Node) {
		for node := parent.FirstChild; node != nil; {
			next := node.NextSibling
			if node.Type == xhtml.ElementNode &&
				strings.EqualFold(node.Data, "iframe") {
				source := htmlAttribute(node, "src")
				parsed, parseErr := url.Parse(source)
				host := strings.ToLower(parsed.Hostname())
				if parseErr != nil ||
					parsed.Scheme != "https" ||
					(host != "youtube.com" &&
						host != "www.youtube.com" &&
						host != "player.vimeo.com") {
					parent.RemoveChild(node)
					node = next
					continue
				}
			}
			prune(node)
			node = next
		}
	}
	root := &xhtml.Node{
		Type: xhtml.ElementNode, Data: "div", DataAtom: atom.Div,
	}
	for _, node := range nodes {
		root.AppendChild(node)
	}
	prune(root)
	var output bytes.Buffer
	for node := root.FirstChild; node != nil; node = node.NextSibling {
		_ = xhtml.Render(&output, node)
	}
	return output.String()
}

func hardenCommentAnchors(raw string) string {
	contextNode := &xhtml.Node{
		Type: xhtml.ElementNode, Data: "div", DataAtom: atom.Div,
	}
	nodes, err := xhtml.ParseFragment(
		strings.NewReader(raw),
		contextNode,
	)
	if err != nil {
		return ""
	}
	var walk func(*xhtml.Node)
	walk = func(node *xhtml.Node) {
		if node.Type == xhtml.ElementNode &&
			strings.EqualFold(node.Data, "a") &&
			htmlAttribute(node, "href") != "" {
			setHTMLAttribute(node, "target", "_blank")
			setHTMLAttribute(
				node,
				"rel",
				"noopener noreferrer nofollow",
			)
		}
		for child := node.FirstChild; child != nil; child = child.NextSibling {
			walk(child)
		}
	}
	var output bytes.Buffer
	for _, node := range nodes {
		walk(node)
		_ = xhtml.Render(&output, node)
	}
	return output.String()
}

func htmlAttribute(node *xhtml.Node, name string) string {
	for _, attr := range node.Attr {
		if strings.EqualFold(attr.Key, name) {
			return attr.Val
		}
	}
	return ""
}

func setHTMLAttribute(node *xhtml.Node, name, value string) {
	for index := range node.Attr {
		if strings.EqualFold(node.Attr[index].Key, name) {
			node.Attr[index].Val = value
			return
		}
	}
	node.Attr = append(node.Attr, xhtml.Attribute{
		Key: name, Val: value,
	})
}

func commentImageURLs(raw string) []string {
	contextNode := &xhtml.Node{
		Type: xhtml.ElementNode, Data: "div", DataAtom: atom.Div,
	}
	nodes, err := xhtml.ParseFragment(
		strings.NewReader(raw),
		contextNode,
	)
	if err != nil {
		return nil
	}
	seen := map[string]bool{}
	result := []string{}
	var walk func(*xhtml.Node)
	walk = func(node *xhtml.Node) {
		if node.Type == xhtml.ElementNode &&
			strings.EqualFold(node.Data, "img") {
			source := htmlAttribute(node, "src")
			if (strings.HasPrefix(source, "http://") ||
				strings.HasPrefix(source, "https://")) &&
				!seen[source] {
				seen[source] = true
				result = append(result, source)
			}
		}
		for child := node.FirstChild; child != nil; child = child.NextSibling {
			walk(child)
		}
	}
	for _, node := range nodes {
		walk(node)
	}
	return result
}

func (r *commentRenderer) resolveMedia(
	ctx context.Context,
	rawURL string,
) *resolvedCommentMedia {
	sum := sha1.Sum([]byte(rawURL))
	key := "media:resolved:" + hex.EncodeToString(sum[:])[:24]
	if r.cache != nil {
		value, err := r.cache.Get(ctx, key).Result()
		if err == nil {
			if value == "null" {
				return nil
			}
			var media resolvedCommentMedia
			if json.Unmarshal([]byte(value), &media) == nil {
				return &media
			}
		}
	}
	media := r.fetchCommentMedia(ctx, rawURL)
	if r.cache != nil {
		value := "null"
		ttl := time.Hour
		if media != nil {
			if raw, err := json.Marshal(media); err == nil {
				value = string(raw)
				ttl = 30 * 24 * time.Hour
			}
		}
		_ = r.cache.Set(
			context.WithoutCancel(ctx),
			key,
			value,
			ttl,
		).Err()
	}
	return media
}

func (r *commentRenderer) fetchCommentMedia(
	ctx context.Context,
	rawURL string,
) *resolvedCommentMedia {
	requestCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	current := rawURL
	var document []byte
	for hop := 0; hop <= 3; hop++ {
		if !isCommentRichMediaURL(current) {
			return nil
		}
		request, err := http.NewRequestWithContext(
			requestCtx,
			http.MethodGet,
			current,
			nil,
		)
		if err != nil {
			return nil
		}
		request.Header.Set(
			"User-Agent",
			"Mozilla/5.0 (Macintosh; Intel Mac OS X 10.15) Firefox/123.0",
		)
		request.Header.Set(
			"Accept",
			"text/html,application/xhtml+xml",
		)
		client := *r.client
		client.CheckRedirect = func(
			_ *http.Request,
			_ []*http.Request,
		) error {
			return http.ErrUseLastResponse
		}
		response, err := client.Do(request)
		if err != nil {
			return nil
		}
		if response.StatusCode >= 300 &&
			response.StatusCode < 400 {
			_ = response.Body.Close()
			location := response.Header.Get("Location")
			base, baseErr := url.Parse(current)
			next, nextErr := url.Parse(location)
			if baseErr != nil || nextErr != nil || location == "" {
				return nil
			}
			current = base.ResolveReference(next).String()
			continue
		}
		if response.StatusCode < 200 || response.StatusCode >= 300 {
			_ = response.Body.Close()
			return nil
		}
		document, err = io.ReadAll(
			io.LimitReader(response.Body, 2<<20),
		)
		_ = response.Body.Close()
		if err != nil {
			return nil
		}
		break
	}
	if document == nil {
		return nil
	}
	metadata := parseCommentOpenGraph(document)
	video := firstNonEmpty(
		metadata["og:video:secure_url"],
		metadata["og:video"],
	)
	image := firstNonEmpty(
		metadata["og:image:secure_url"],
		metadata["og:image"],
	)
	width := firstPositiveInteger(
		metadata["og:video:width"],
		metadata["og:image:width"],
	)
	height := firstPositiveInteger(
		metadata["og:video:height"],
		metadata["og:image:height"],
	)
	if video != "" && commentResolvedVideoPattern.MatchString(video) {
		return &resolvedCommentMedia{
			Type: "video", Source: cleanCommentMediaURL(video),
			Poster: cleanCommentMediaURL(image),
			Width:  width, Height: height,
		}
	}
	if image != "" && commentResolvedImagePattern.MatchString(image) {
		return &resolvedCommentMedia{
			Type: "image", Source: cleanCommentMediaURL(image),
			Width: width, Height: height,
		}
	}
	return nil
}

func parseCommentOpenGraph(document []byte) map[string]string {
	result := map[string]string{}
	root, err := xhtml.Parse(bytes.NewReader(document))
	if err != nil {
		return result
	}
	var walk func(*xhtml.Node)
	walk = func(node *xhtml.Node) {
		if node.Type == xhtml.ElementNode &&
			strings.EqualFold(node.Data, "meta") {
			property := htmlAttribute(node, "property")
			if property == "" {
				property = htmlAttribute(node, "name")
			}
			content := htmlAttribute(node, "content")
			if property != "" && content != "" {
				result[strings.ToLower(property)] =
					stdhtml.UnescapeString(content)
			}
		}
		for child := node.FirstChild; child != nil; child = child.NextSibling {
			walk(child)
		}
	}
	walk(root)
	return result
}

func cleanCommentMediaURL(raw string) string {
	if raw == "" {
		return ""
	}
	parsed, err := url.Parse(raw)
	if err != nil {
		return raw
	}
	parsed.RawQuery = ""
	parsed.Fragment = ""
	return parsed.String()
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if value != "" {
			return value
		}
	}
	return ""
}

func firstPositiveInteger(values ...string) int {
	for _, value := range values {
		parsed, err := strconv.Atoi(value)
		if err == nil && parsed > 0 {
			return parsed
		}
	}
	return 0
}

type openAICommentModerator struct {
	cache  *redis.Client
	client *http.Client
	apiKey string
}

func newOpenAICommentModerator(
	cache *redis.Client,
	client *http.Client,
) commentModerator {
	if client == nil {
		client = &http.Client{Timeout: 8 * time.Second}
	}
	return &openAICommentModerator{
		cache: cache, client: client,
		apiKey: strings.TrimSpace(firstNonEmpty(
			os.Getenv("OPENAI_API_KEY"),
			os.Getenv("NUXT_OPENAI_API_KEY"),
		)),
	}
}

type commentModerationRule struct {
	Threshold float64
	Action    commentModerationAction
	Label     string
}

var commentModerationRules = map[string]commentModerationRule{
	"sexual/minors": {
		0.40, commentActionDelete,
		"sexual content involving minors",
	},
	"hate/threatening": {
		0.65, commentActionDelete, "identity-based threats",
	},
	"self-harm/instructions": {
		0.55, commentActionDelete, "self-harm instructions",
	},
	"illicit/violent": {
		0.65, commentActionDelete,
		"instructions for real-world violence",
	},
	"hate": {
		0.75, commentActionFlag, "hate speech",
	},
	"self-harm/intent": {
		0.70, commentActionFlag, "self-harm content",
	},
	"sexual": {
		0.85, commentActionFlag, "sexual content",
	},
	"self-harm": {
		0.92, commentActionFlag, "self-harm content",
	},
	"harassment/threatening": {
		0.95, commentActionFlag, "real-world threats",
	},
	"harassment": {
		0.98, commentActionFlag, "severe harassment",
	},
	"violence/graphic": {
		0.95, commentActionFlag, "graphic real-world violence",
	},
	"violence": {
		0.98, commentActionFlag, "graphic real-world violence",
	},
	"illicit": {
		0.92, commentActionFlag, "illicit activity",
	},
}

// OpenAI returns category_scores in this stable schema order. Keeping the
// same order matters when two categories of the same severity cross their
// threshold: the TypeScript Object.entries loop retains the first one.
var commentModerationCategoryOrder = []string{
	"harassment",
	"harassment/threatening",
	"hate",
	"hate/threatening",
	"illicit",
	"illicit/violent",
	"self-harm",
	"self-harm/instructions",
	"self-harm/intent",
	"sexual",
	"sexual/minors",
	"violence",
	"violence/graphic",
}

func (m *openAICommentModerator) Moderate(
	ctx context.Context,
	text string,
	imageURLs []string,
) commentModerationResult {
	pass := commentModerationResult{
		Action: commentActionPass, Scores: map[string]float64{},
		Source: "stub",
	}
	if m == nil || m.apiKey == "" {
		return pass
	}
	sortedImages := append([]string(nil), imageURLs...)
	sort.Strings(sortedImages)
	canonical := text + "\x00" +
		strings.Join(sortedImages, "\x01")
	sum := sha256.Sum256([]byte(canonical))
	key := "cmt:mod:" + hex.EncodeToString(sum[:])[:24]
	if m.cache != nil {
		raw, err := m.cache.Get(ctx, key).Bytes()
		if err == nil {
			var cached commentModerationResult
			if json.Unmarshal(raw, &cached) == nil {
				cached.Source = "cached"
				return cached
			}
		}
	}
	input := []map[string]any{
		{"type": "text", "text": text},
	}
	for _, imageURL := range imageURLs {
		input = append(input, map[string]any{
			"type": "image_url",
			"image_url": map[string]string{
				"url": imageURL,
			},
		})
	}
	requestBody, err := json.Marshal(map[string]any{
		"model": "omni-moderation-latest",
		"input": input,
	})
	if err != nil {
		return pass
	}
	requestCtx, cancel := context.WithTimeout(ctx, 8*time.Second)
	defer cancel()
	request, err := http.NewRequestWithContext(
		requestCtx,
		http.MethodPost,
		"https://api.openai.com/v1/moderations",
		bytes.NewReader(requestBody),
	)
	if err != nil {
		return pass
	}
	request.Header.Set("Authorization", "Bearer "+m.apiKey)
	request.Header.Set("Content-Type", "application/json")
	response, err := m.client.Do(request)
	if err != nil {
		return pass
	}
	defer response.Body.Close()
	if response.StatusCode < 200 || response.StatusCode >= 300 {
		_, _ = io.Copy(
			io.Discard,
			io.LimitReader(response.Body, 4096),
		)
		return pass
	}
	var upstream struct {
		Results []struct {
			Scores map[string]float64 `json:"category_scores"`
		} `json:"results"`
	}
	if json.NewDecoder(
		io.LimitReader(response.Body, 2<<20),
	).Decode(&upstream) != nil || len(upstream.Results) == 0 {
		return pass
	}
	result := decideCommentModeration(upstream.Results[0].Scores)
	if m.cache != nil {
		if raw, marshalErr := json.Marshal(result); marshalErr == nil {
			_ = m.cache.Set(
				context.WithoutCancel(ctx),
				key,
				raw,
				time.Hour,
			).Err()
		}
	}
	return result
}

func decideCommentModeration(
	scores map[string]float64,
) commentModerationResult {
	result := commentModerationResult{
		Action: commentActionPass, Scores: scores, Source: "openai",
	}
	rank := map[commentModerationAction]int{
		commentActionPass: 0, commentActionFlag: 1,
		commentActionDelete: 2,
	}
	worstLabel := ""
	for _, score := range scores {
		if score > result.MaxScore {
			result.MaxScore = score
		}
	}
	for _, category := range commentModerationCategoryOrder {
		score := scores[category]
		rule, exists := commentModerationRules[category]
		if !exists || score < rule.Threshold ||
			rank[rule.Action] <= rank[result.Action] {
			continue
		}
		result.Action = rule.Action
		value := category
		result.Category = &value
		worstLabel = rule.Label
	}
	if worstLabel != "" {
		message := "Your comment was rejected: " +
			worstLabel + " is not allowed."
		result.UserMessage = &message
	}
	return result
}

type commentRateLimitResult struct {
	OK         bool
	RetryAfter int
	Reason     string
}

type commentRateWindow struct {
	Key string
	TTL time.Duration
	Max int64
}

func checkCommentRateLimit(
	ctx context.Context,
	cache *redis.Client,
	characterID int32,
	body string,
) commentRateLimitResult {
	if cache == nil {
		return commentRateLimitResult{OK: true}
	}
	sum := sha1.Sum([]byte(body))
	duplicateKey := fmt.Sprintf(
		"cmt:rl:dup:%d:%s",
		characterID,
		hex.EncodeToString(sum[:])[:16],
	)
	ok, err := cache.SetNX(
		ctx,
		duplicateKey,
		"1",
		time.Minute,
	).Result()
	if err == nil && !ok {
		return commentRateLimitResult{
			RetryAfter: 60, Reason: "duplicate",
		}
	}
	return checkCommentRateWindows(
		ctx,
		cache,
		characterID,
		[]commentRateWindow{
			{"min", time.Minute, 5},
			{"hr", time.Hour, 30},
			{"day", 24 * time.Hour, 200},
		},
	)
}

func checkCommentPreviewRateLimit(
	ctx context.Context,
	cache *redis.Client,
	characterID int32,
) commentRateLimitResult {
	return checkCommentRateWindows(
		ctx,
		cache,
		characterID,
		[]commentRateWindow{
			{"pv:min", time.Minute, 30},
			{"pv:hr", time.Hour, 300},
		},
	)
}

func checkCommentKlipyRateLimit(
	ctx context.Context,
	cache *redis.Client,
	characterID int32,
) commentRateLimitResult {
	return checkCommentRateWindows(
		ctx,
		cache,
		characterID,
		[]commentRateWindow{
			{"kl:min", time.Minute, 60},
			{"kl:hr", time.Hour, 600},
		},
	)
}

func checkCommentRateWindows(
	ctx context.Context,
	cache *redis.Client,
	characterID int32,
	windows []commentRateWindow,
) commentRateLimitResult {
	if cache == nil {
		return commentRateLimitResult{OK: true}
	}
	for _, window := range windows {
		key := fmt.Sprintf(
			"cmt:rl:%s:%d",
			window.Key,
			characterID,
		)
		count, err := cache.Incr(ctx, key).Result()
		if err != nil {
			continue
		}
		if count == 1 {
			_ = cache.Expire(ctx, key, window.TTL).Err()
		}
		if count > window.Max {
			ttl, ttlErr := cache.TTL(ctx, key).Result()
			if ttlErr != nil || ttl <= 0 {
				ttl = window.TTL
			}
			return commentRateLimitResult{
				RetryAfter: int(ttl.Seconds()),
				Reason:     "rate_limited",
			}
		}
	}
	return commentRateLimitResult{OK: true}
}

func checkCommentModerationCooldown(
	ctx context.Context,
	cache *redis.Client,
	characterID int32,
) commentRateLimitResult {
	if cache == nil {
		return commentRateLimitResult{OK: true}
	}
	key := fmt.Sprintf("cmt:rl:mod:%d", characterID)
	count, err := cache.Get(ctx, key).Int64()
	if err != nil && !errors.Is(err, redis.Nil) {
		return commentRateLimitResult{OK: true}
	}
	if count < 3 {
		return commentRateLimitResult{OK: true}
	}
	ttl, err := cache.TTL(ctx, key).Result()
	if err != nil || ttl <= 0 {
		ttl = 5 * time.Minute
	}
	return commentRateLimitResult{
		RetryAfter: int(ttl.Seconds()),
		Reason:     "rate_limited",
	}
}

func recordCommentModerationRejection(
	ctx context.Context,
	cache *redis.Client,
	characterID int32,
) {
	if cache == nil {
		return
	}
	key := fmt.Sprintf("cmt:rl:mod:%d", characterID)
	count, err := cache.Incr(ctx, key).Result()
	if err == nil && count == 1 {
		_ = cache.Expire(ctx, key, 5*time.Minute).Err()
	}
}

type klipyClient struct {
	cache  *redis.Client
	client *http.Client
	apiKey string
}

func newKlipyClient(
	cache *redis.Client,
	client *http.Client,
) *klipyClient {
	if client == nil {
		client = &http.Client{Timeout: 5 * time.Second}
	}
	return &klipyClient{
		cache: cache, client: client,
		apiKey: strings.TrimSpace(firstNonEmpty(
			os.Getenv("KLIPY_API_KEY"),
			os.Getenv("NUXT_KLIPY_API_KEY"),
		)),
	}
}

func (k *klipyClient) Fetch(
	ctx context.Context,
	kind string,
	search string,
	page int,
	perPage int,
) (map[string]any, error) {
	if k == nil || k.apiKey == "" {
		return nil, apiError(
			http.StatusInternalServerError,
			"Klipy API key not configured",
		)
	}
	signature := fmt.Sprintf(
		"q=%s&page=%d&per_page=%d",
		search,
		page,
		perPage,
	)
	sum := sha1.Sum([]byte(signature))
	key := "klipy:" + kind + ":" +
		hex.EncodeToString(sum[:])[:16]
	if k.cache != nil {
		raw, err := k.cache.Get(ctx, key).Bytes()
		if err == nil {
			var cached map[string]any
			if json.Unmarshal(raw, &cached) == nil {
				return cached, nil
			}
		}
	}
	upstream, err := url.Parse(
		"https://api.klipy.com/api/v1/" +
			url.PathEscape(k.apiKey) + "/gifs/" + kind,
	)
	if err != nil {
		return nil, err
	}
	query := upstream.Query()
	if kind == "search" {
		query.Set("q", search)
	}
	query.Set("page", strconv.Itoa(page))
	query.Set("per_page", strconv.Itoa(perPage))
	upstream.RawQuery = query.Encode()
	requestCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	request, err := http.NewRequestWithContext(
		requestCtx,
		http.MethodGet,
		upstream.String(),
		nil,
	)
	if err != nil {
		return nil, err
	}
	response, err := k.client.Do(request)
	if err != nil {
		return nil, apiError(
			http.StatusBadGateway,
			"Klipy upstream unavailable",
		)
	}
	defer response.Body.Close()
	if response.StatusCode < 200 || response.StatusCode >= 300 {
		return nil, apiError(
			http.StatusBadGateway,
			fmt.Sprintf("Klipy upstream %d", response.StatusCode),
		)
	}
	var raw struct {
		Result bool `json:"result"`
		Data   *struct {
			Items   []any `json:"data"`
			Page    int   `json:"current_page"`
			PerPage int   `json:"per_page"`
			HasNext bool  `json:"has_next"`
		} `json:"data"`
	}
	if err := json.NewDecoder(
		io.LimitReader(response.Body, 10<<20),
	).Decode(&raw); err != nil {
		return nil, apiError(
			http.StatusBadGateway,
			"Klipy returned invalid data",
		)
	}
	if !raw.Result || raw.Data == nil {
		return nil, apiError(
			http.StatusBadGateway,
			"Klipy returned no data",
		)
	}
	result := map[string]any{
		"items": raw.Data.Items, "page": raw.Data.Page,
		"per_page": raw.Data.PerPage,
		"has_next": raw.Data.HasNext,
	}
	if k.cache != nil {
		ttl := time.Hour
		if kind == "trending" {
			ttl = 5 * time.Minute
		}
		if encoded, marshalErr := json.Marshal(result); marshalErr == nil {
			_ = k.cache.Set(
				context.WithoutCancel(ctx),
				key,
				encoded,
				ttl,
			).Err()
		}
	}
	return result, nil
}
