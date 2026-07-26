package api

import (
	"errors"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/danielgtaylor/huma/v2"
	"github.com/danielgtaylor/huma/v2/adapters/humago"
)

func TestContentRoutesRegisterCanonicalAndLiveFrontendContracts(
	t *testing.T,
) {
	mux := http.NewServeMux()
	a := humago.New(mux, huma.DefaultConfig("content-test", "test"))
	registerBlogRoutes(a, Options{})
	registerAnnouncementAdminRoutes(a, Options{})
	registerCommentRoutes(a, Options{})
	registerModerationRoutes(a, Options{})

	for _, route := range []struct {
		method string
		path   string
	}{
		{http.MethodGet, "/blog"},
		{http.MethodGet, "/blog/{slug}"},
		{http.MethodGet, "/admin/blog"},
		{http.MethodPost, "/admin/blog"},
		{http.MethodGet, "/admin/blog/{id}"},
		{http.MethodPatch, "/admin/blog/{id}"},
		{http.MethodDelete, "/admin/blog/{id}"},
		{http.MethodGet, "/admin/blog/preview/{slug}"},
		{http.MethodGet, "/admin/announcements"},
		{http.MethodPost, "/admin/announcements"},
		{http.MethodGet, "/admin/announcements/{id}"},
		{http.MethodPatch, "/admin/announcements/{id}"},
		{http.MethodDelete, "/admin/announcements/{id}"},
		{http.MethodPost, "/admin/announcements/{id}/archive"},
		{http.MethodGet, "/comments"},
		{http.MethodPost, "/comments"},
		{http.MethodGet, "/comments/thread"},
		{http.MethodPost, "/comments/preview"},
		{http.MethodGet, "/comments/klipy/search"},
		{http.MethodGet, "/comments/klipy/trending"},
		{http.MethodGet, "/comments/{id}"},
		{http.MethodPatch, "/comments/{id}"},
		{http.MethodDelete, "/comments/{id}"},
		{http.MethodPost, "/comments/{id}/report"},
		{http.MethodGet, "/me/comments"},
		{http.MethodDelete, "/me/comments/{id}"},
		{http.MethodGet, "/user/comments"},
		{http.MethodDelete, "/user/comments/{id}"},
		{http.MethodGet, "/admin/comments"},
		{http.MethodPatch, "/admin/comments/{id}"},
		{http.MethodGet, "/admin/comments/queue"},
		{http.MethodPost, "/admin/comments/{id}/hide"},
		{http.MethodPost, "/admin/comments/{id}/restore"},
		{http.MethodPatch, "/admin/comment-reports/{id}"},
		{
			http.MethodPost,
			"/admin/comments/reports/{id}/resolve",
		},
		{http.MethodGet, "/admin/moderation"},
		{http.MethodPatch, "/admin/moderation/{id}"},
		{http.MethodGet, "/admin/moderation/queue"},
		{http.MethodPost, "/admin/moderation/{id}/approve"},
		{http.MethodPost, "/admin/moderation/{id}/reject"},
	} {
		operation := contentTestOperation(
			a.OpenAPI(),
			route.method,
			route.path,
		)
		if operation == nil {
			t.Errorf("%s %s is not registered", route.method, route.path)
		}
	}
}

func TestContentMutationsRequireSessionInOpenAPI(t *testing.T) {
	mux := http.NewServeMux()
	a := humago.New(mux, huma.DefaultConfig("content-test", "test"))
	registerCommentRoutes(a, Options{})
	registerModerationRoutes(a, Options{})

	for _, route := range []struct {
		method string
		path   string
	}{
		{http.MethodPost, "/comments"},
		{http.MethodPatch, "/comments/{id}"},
		{http.MethodDelete, "/comments/{id}"},
		{http.MethodPost, "/comments/{id}/report"},
		{http.MethodGet, "/me/comments"},
		{http.MethodGet, "/admin/moderation"},
		{http.MethodPatch, "/admin/moderation/{id}"},
	} {
		operation := contentTestOperation(
			a.OpenAPI(),
			route.method,
			route.path,
		)
		if operation == nil {
			t.Fatalf("%s %s is not registered", route.method, route.path)
		}
		if len(operation.Security) != 1 {
			t.Errorf(
				"%s %s security = %#v",
				route.method,
				route.path,
				operation.Security,
			)
			continue
		}
		if _, ok := operation.Security[0]["eveSession"]; !ok {
			t.Errorf(
				"%s %s does not require eveSession",
				route.method,
				route.path,
			)
		}
	}
}

func TestCommentValidationUsesJavaScriptUTF16Length(t *testing.T) {
	if value, err := validateCommentBody(
		strings.Repeat("😀", 1000),
	); err != nil || utf16Length(value) != commentMaximumLength {
		t.Fatalf("valid astral body = %d units, %v", utf16Length(value), err)
	}
	_, err := validateCommentBody(strings.Repeat("😀", 1001))
	var apiErr *legacyAPIError
	if !errors.As(err, &apiErr) ||
		apiErr.Status != http.StatusBadRequest ||
		!strings.Contains(apiErr.Message, "exceeds 2000") {
		t.Fatalf("oversized astral body error = %#v", err)
	}
	_, err = validateCommentBodyValue(nil)
	if !errors.As(err, &apiErr) ||
		apiErr.Message != "Comment is too short" {
		t.Fatalf("missing body error = %#v", err)
	}
}

func TestCommentTargetValidationMatchesSlugTargets(t *testing.T) {
	for _, target := range []int64{
		commentTargetPage,
		commentTargetFit,
		commentTargetCampaign,
	} {
		if err := validateCommentTarget(target, 0, nil); err == nil {
			t.Errorf("target %d accepted without slug", target)
		}
		slug := "target-abc"
		if err := validateCommentTarget(target, 0, &slug); err != nil {
			t.Errorf("target %d with slug rejected: %v", target, err)
		}
	}
	if err := validateCommentTarget(
		commentTargetKillmail,
		-1,
		nil,
	); err == nil {
		t.Fatal("negative numeric target was accepted")
	}
}

func TestCommentRendererSanitizesAndHardensOutput(t *testing.T) {
	renderer := newCommentRenderer(nil, nil)
	rendered := renderer.Render(
		t.Context(),
		`[safe](https://example.com) `+
			`[bad](javascript:alert(1)) `+
			`<script>alert("x")</script> :ship:`,
	)
	for _, forbidden := range []string{
		"javascript:", "<script", `onerror=`,
	} {
		if strings.Contains(strings.ToLower(rendered), forbidden) {
			t.Fatalf("%q survived sanitization: %s", forbidden, rendered)
		}
	}
	for _, expected := range []string{
		`target="_blank"`,
		`rel="noopener noreferrer nofollow"`,
		`class="emoji"`,
		`data-emoji="ship"`,
	} {
		if !strings.Contains(rendered, expected) {
			t.Errorf("rendered output lacks %q: %s", expected, rendered)
		}
	}
}

func TestCommentRendererAllowsOnlyGeneratedVideoFrames(t *testing.T) {
	renderer := newCommentRenderer(nil, nil)
	rendered := renderer.Render(
		t.Context(),
		"https://youtu.be/dQw4w9WgXcQ",
	)
	if !strings.Contains(
		rendered,
		`src="https://www.youtube.com/embed/dQw4w9WgXcQ"`,
	) {
		t.Fatalf("YouTube URL was not embedded: %s", rendered)
	}
	if bad := stripBadCommentIframes(
		`<iframe src="https://evil.example/embed/x"></iframe>`,
	); strings.Contains(bad, "iframe") {
		t.Fatalf("untrusted iframe survived: %s", bad)
	}
}

func TestCommentModerationUsesEVETunedThresholds(t *testing.T) {
	pass := decideCommentModeration(map[string]float64{
		"harassment": 0.97,
		"violence":   0.97,
	})
	if pass.Action != commentActionPass {
		t.Fatalf("EVE-banter verdict = %#v", pass)
	}
	flag := decideCommentModeration(map[string]float64{
		"hate": 0.80,
	})
	if flag.Action != commentActionFlag {
		t.Fatalf("hate verdict = %#v", flag)
	}
	remove := decideCommentModeration(map[string]float64{
		"hate":          0.99,
		"sexual/minors": 0.40,
	})
	if remove.Action != commentActionDelete {
		t.Fatalf("zero-tolerance verdict = %#v", remove)
	}
}

func TestCommentDomainMembershipUsesOwnerAndEntitySnapshots(
	t *testing.T,
) {
	corporationID := int32(98000001)
	allianceID := int32(99000001)
	domain := &commentDomain{
		OwnerID: 90000001,
		Entities: []commentDomainEntity{
			{Type: "corporation", ID: int64(corporationID)},
			{Type: "alliance", ID: int64(allianceID)},
		},
	}
	if !commentDomainMember(domain, &Principal{
		CharacterID: 90000001,
	}) {
		t.Fatal("domain owner was not a member")
	}
	if !commentDomainMember(domain, &Principal{
		CharacterID:   90000002,
		CorporationID: &corporationID,
	}) {
		t.Fatal("tracked corporation member was not a member")
	}
	if !commentDomainMember(domain, &Principal{
		CharacterID: 90000003,
		AllianceID:  &allianceID,
	}) {
		t.Fatal("tracked alliance member was not a member")
	}
	if commentDomainMember(domain, &Principal{
		CharacterID: 90000004,
	}) {
		t.Fatal("untracked character was treated as a member")
	}
}

func TestCommentRelayPayloadKeepsEstablishedWireShape(t *testing.T) {
	now := time.Date(2026, 7, 26, 12, 34, 56, 0, time.UTC)
	row := map[string]any{
		"id": 44, "target_type": 1, "target_id": 123,
		"target_slug": nil, "domain_id": nil, "parent_id": 43,
		"root_id": 40, "depth": 2, "body_md": "server only",
		"body_html": "<p>hello</p>", "character_id": 9001,
		"character_name": "Pilot", "corporation_id": 9801,
		"corporation_name": "Corp", "alliance_id": nil,
		"alliance_name": nil, "created_at": now, "edited_at": nil,
		"reports_count": 7,
	}
	ancestors := []commentAncestor{
		{ID: 40, CharacterID: 9002, CharacterName: "Root"},
		{ID: 43, CharacterID: 9003, CharacterName: "Parent"},
	}
	payload := buildCommentEventPayload("new", row, ancestors)
	if payload["parent_character_id"] != int64(9003) ||
		payload["parent_character_name"] != "Parent" {
		t.Fatalf("parent convenience fields = %#v", payload)
	}
	comment, ok := payload["comment"].(map[string]any)
	if !ok {
		t.Fatalf("comment payload = %#v", payload["comment"])
	}
	if _, exists := comment["body_md"]; exists {
		t.Fatalf("server-only body_md leaked: %#v", comment)
	}
	if _, exists := comment["reports_count"]; exists {
		t.Fatalf("server-only reports_count leaked: %#v", comment)
	}
	deleted := buildCommentEventPayload("deleted", row, ancestors)
	if _, exists := deleted["comment"]; exists {
		t.Fatalf("deleted event contains comment: %#v", deleted)
	}
	if got := deleted["ancestors"]; len(got.([]commentAncestor)) != 0 {
		t.Fatalf("deleted event ancestry = %#v", got)
	}
}

func TestBlogAndAnnouncementMarkdownDropsActiveContent(t *testing.T) {
	source := `[safe](https://example.com) ` +
		`[bad](javascript:alert(1)) <script>alert(2)</script>`
	for name, rendered := range map[string]string{
		"blog":         renderBlogMarkdown(source),
		"announcement": renderAnnouncementMarkdown(source),
	} {
		if strings.Contains(strings.ToLower(rendered), "javascript:") ||
			strings.Contains(strings.ToLower(rendered), "<script") {
			t.Errorf("%s active content survived: %s", name, rendered)
		}
	}
}

func TestBlogTagNormalizationPreservesTypeScriptHyphenSemantics(
	t *testing.T,
) {
	if got := normalizeBlogTag("  Café  !  Fleet--Ops  "); got != "cafe--fleet--ops" {
		t.Fatalf("normalized tag = %q", got)
	}
	tags := normalizeBlogTags([]any{
		" Fleet Ops ", "fleet ops", "Café", 42,
	})
	if len(tags) != 2 || tags[0] != "fleet-ops" ||
		tags[1] != "cafe" {
		t.Fatalf("normalized tags = %#v", tags)
	}
}

func contentTestOperation(
	document *huma.OpenAPI,
	method string,
	path string,
) *huma.Operation {
	item := document.Paths[path]
	if item == nil {
		return nil
	}
	switch method {
	case http.MethodGet:
		return item.Get
	case http.MethodPost:
		return item.Post
	case http.MethodPatch:
		return item.Patch
	case http.MethodPut:
		return item.Put
	case http.MethodDelete:
		return item.Delete
	default:
		return nil
	}
}
