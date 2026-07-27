package api

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"time"

	"github.com/danielgtaylor/huma/v2"
)

const (
	announcementsCacheTTL     = 30 * time.Second
	announcementsCacheControl = "public, max-age=30, s-maxage=30"
)

// AnnouncementsResponse is the canonical frontend and public API payload.
// Defining it here, rather than importing a database row in Nuxt, makes the
// generated Huma contract authoritative for both persisted and live tickers.
type AnnouncementsResponse struct {
	Announcements []Announcement `json:"announcements" nullable:"false"`
}

type DismissedAnnouncementIDsResponse struct {
	DismissedIDs []int64 `json:"dismissedIds" nullable:"false"`
}

type AnnouncementDismissalResponse struct {
	OK bool `json:"ok"`
}

type announcementsOutput struct {
	CacheControl string `header:"Cache-Control"`
	CacheStatus  string `header:"X-Cache"`
	Body         AnnouncementsResponse
}

type dismissedAnnouncementsOutput struct {
	CacheControl string `header:"Cache-Control"`
	Body         DismissedAnnouncementIDsResponse
}

type announcementDismissalInput struct {
	ID int64 `path:"id" minimum:"1"`
}

type announcementDismissalOutput struct {
	CacheControl string `header:"Cache-Control"`
	Body         AnnouncementDismissalResponse
}

type accountHumaContextKey struct{}

func captureAccountHumaContext(
	ctx huma.Context,
	next func(huma.Context),
) {
	next(huma.WithValue(ctx, accountHumaContextKey{}, ctx))
}

func accountHumaContext(ctx context.Context) huma.Context {
	humaCtx, _ := ctx.Value(accountHumaContextKey{}).(huma.Context)
	return humaCtx
}

func registerCanonicalAnnouncementsRoute(
	a huma.API,
	opts Options,
	service *accountService,
) {
	huma.Register(a, huma.Operation{
		OperationID: "announcements",
		Method:      http.MethodGet,
		Path:        "/announcements",
		Summary:     "Active site announcements",
		Description: "Returns active editorial announcements and live ticker events using one stable presentation contract.",
		Tags:        []string{"announcements"},
		Extensions:  map[string]any{"x-audience": "public"},
	}, func(
		ctx context.Context,
		_ *struct{},
	) (*announcementsOutput, error) {
		body, cacheStatus, err := service.loadCanonicalAnnouncements(ctx, opts)
		if err != nil {
			var apiErr *legacyAPIError
			if errors.As(err, &apiErr) {
				return nil, huma.NewError(apiErr.Status, apiErr.Message)
			}
			return nil, err
		}
		return &announcementsOutput{
			CacheControl: announcementsCacheControl,
			CacheStatus:  cacheStatus,
			Body:         body,
		}, nil
	})
}

func registerCanonicalAnnouncementAccountRoutes(
	a huma.API,
	service *accountService,
	requiredSession []map[string][]string,
) {
	accountMiddleware := huma.Middlewares{captureAccountHumaContext}
	huma.Register(a, huma.Operation{
		OperationID: "account-dismissed-announcements",
		Method:      http.MethodGet,
		Path:        "/me/announcements/dismissed",
		Summary:     "Announcements dismissed by this account",
		Tags:        []string{"account", "announcements"},
		Security:    requiredSession,
		Middlewares: accountMiddleware,
	}, func(
		ctx context.Context,
		_ *struct{},
	) (*dismissedAnnouncementsOutput, error) {
		humaCtx := accountHumaContext(ctx)
		if humaCtx == nil {
			return nil, huma.Error500InternalServerError(
				"Request context is unavailable",
			)
		}
		setAccountNoStore(humaCtx)
		principal, err := service.principal(
			ctx,
			&legacyRequest{Huma: humaCtx},
		)
		if err != nil {
			return nil, humaAccountError(err)
		}
		ids, err := service.store.LoadDismissedAnnouncementIDs(
			ctx,
			principal.CharacterID,
		)
		if err != nil {
			return nil, err
		}
		if ids == nil {
			ids = []int64{}
		}
		return &dismissedAnnouncementsOutput{
			CacheControl: "private, no-store",
			Body: DismissedAnnouncementIDsResponse{
				DismissedIDs: ids,
			},
		}, nil
	})

	huma.Register(a, huma.Operation{
		OperationID: "account-announcement-dismissal",
		Method:      http.MethodPut,
		Path:        "/me/announcements/{id}/dismissal",
		Summary:     "Dismiss an announcement",
		Tags:        []string{"account", "announcements"},
		Security:    requiredSession,
		Middlewares: accountMiddleware,
	}, func(
		ctx context.Context,
		input *announcementDismissalInput,
	) (*announcementDismissalOutput, error) {
		humaCtx := accountHumaContext(ctx)
		if humaCtx == nil {
			return nil, huma.Error500InternalServerError(
				"Request context is unavailable",
			)
		}
		setAccountNoStore(humaCtx)
		if err := requireSameOriginMutation(humaCtx); err != nil {
			return nil, humaAccountError(err)
		}
		principal, err := service.principal(
			ctx,
			&legacyRequest{Huma: humaCtx},
		)
		if err != nil {
			return nil, humaAccountError(err)
		}
		if err := service.store.DismissAnnouncement(
			ctx,
			principal.CharacterID,
			input.ID,
		); err != nil {
			return nil, err
		}
		return &announcementDismissalOutput{
			CacheControl: "private, no-store",
			Body:         AnnouncementDismissalResponse{OK: true},
		}, nil
	})
}

func humaAccountError(err error) error {
	var apiErr *legacyAPIError
	if errors.As(err, &apiErr) {
		return huma.NewError(apiErr.Status, apiErr.Message)
	}
	return err
}

func (s *accountService) loadCanonicalAnnouncements(
	ctx context.Context,
	opts Options,
) (AnnouncementsResponse, string, error) {
	key := canonicalAnnouncementsCacheKey(opts.Commit)
	if entry, ok := cacheLoad(ctx, opts.Cache, key); ok {
		var cached AnnouncementsResponse
		if json.Unmarshal(entry.Body, &cached) == nil {
			if cached.Announcements == nil {
				cached.Announcements = []Announcement{}
			}
			return cached, "HIT", nil
		}
	}

	if err := s.requireStore(); err != nil {
		return AnnouncementsResponse{}, "", err
	}
	now := s.now().UTC()
	stored, err := s.store.LoadActiveAnnouncements(ctx, now)
	if err != nil {
		return AnnouncementsResponse{}, "", err
	}

	announcements := make(
		[]Announcement,
		0,
		len(stored)+4,
	)
	for _, item := range stored {
		announcements = append(announcements, publicAnnouncement(item))
	}
	if s.loadEphemeral != nil {
		for _, item := range s.loadEphemeral(ctx, now) {
			announcement, ok := publicEphemeralAnnouncement(item)
			if ok {
				announcements = append(announcements, announcement)
			}
		}
	}

	body := AnnouncementsResponse{Announcements: announcements}
	if encoded, marshalErr := json.Marshal(body); marshalErr == nil {
		cacheStore(context.WithoutCancel(ctx), opts.Cache, key, cachedResponse{
			ContentType: "application/json",
			Body:        encoded,
		}, announcementsCacheTTL)
	}
	return body, "MISS", nil
}

func canonicalAnnouncementsCacheKey(commit string) string {
	if commit == "" {
		commit = "dev"
	}
	return "shrike:web-api:" + commit + ":announcements:canonical"
}

func publicAnnouncement(item accountAnnouncement) Announcement {
	return Announcement{
		ID:        int64(item.ID),
		Tier:      item.Tier,
		Title:     item.Title,
		BodyMD:    item.BodyMD,
		BodyHTML:  item.BodyHTML,
		Color:     item.Color,
		Icon:      item.Icon,
		LinkURL:   item.LinkURL,
		LinkLabel: item.LinkLabel,
		StartsAt:  item.StartsAt,
		ExpiresAt: item.ExpiresAt,
		CreatedAt: item.CreatedAt,
	}
}

func publicEphemeralAnnouncement(item map[string]any) (Announcement, bool) {
	raw, err := json.Marshal(item)
	if err != nil {
		return Announcement{}, false
	}
	var announcement Announcement
	if json.Unmarshal(raw, &announcement) != nil ||
		announcement.ID == 0 ||
		announcement.Title == "" ||
		announcement.StartsAt.IsZero() ||
		announcement.ExpiresAt.IsZero() ||
		announcement.CreatedAt.IsZero() {
		return Announcement{}, false
	}
	return announcement, true
}
