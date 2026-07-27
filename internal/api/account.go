package api

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"time"

	"github.com/eve-kill/shrike/internal/queue"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/redis/go-redis/v9"
)

const (
	accountBodyLimit           = 64 << 10
	maxBoardEntries            = 12
	maxBoardKeyLength          = 100
	maxDescriptionLength       = 4000
	bioKindCharacter     int16 = 1
	bioKindCorporation         = 2
	bioKindAlliance            = 3
	bioStatusPending     int16 = 0
)

type accountBoardState struct {
	Pinned    []string `json:"pinned"`
	Dismissed []string `json:"dismissed"`
}

type accountBoardDomain struct {
	ID             int32
	Subdomain      string
	CustomHostname *string
	SiteName       *string
	Entities       []accountBoardEntity
}

type accountBoardEntity struct {
	Type string `json:"type"`
	ID   int64  `json:"id"`
}

type accountBoardData struct {
	Account accountBoardState
	Domains []accountBoardDomain
}

type accountOverview struct {
	LastLogin       *time.Time
	CreatedAt       *time.Time
	TotalRequests   int64
	TotalErrors     int64
	TotalNewItems   int64
	LastRequest     *time.Time
	Requests24Hours int64
	Errors24Hours   int64
	NewItems24Hours int64
	TokenFound      bool
	TokenScopeCount int
	TokenExpiry     *time.Time
	TokenLastFetch  *time.Time
}

type accountPendingBio struct {
	Body       string    `json:"body"`
	BodyFormat string    `json:"body_format"`
	Submitted  time.Time `json:"submitted_at"`
}

type accountCharacterBio struct {
	ID                int32
	Name              string
	ESIDescription    *string
	CustomDescription *string
	CustomFormat      *string
	Pending           *accountPendingBio
}

type accountCorporationBio struct {
	ID                int32
	Name              string
	Ticker            string
	CEOID             *int32
	CEOName           *string
	ESIDescription    *string
	CustomDescription *string
	CustomFormat      *string
	Pending           *accountPendingBio
}

type accountAllianceBio struct {
	ID                    int32
	Name                  string
	Ticker                string
	ExecutorCorporationID *int32
	ExecutorCEOID         *int32
	ExecutorCEOName       *string
	CustomDescription     *string
	CustomFormat          *string
	Pending               *accountPendingBio
}

type accountManageableEntities struct {
	Character   accountCharacterBio
	Corporation *accountCorporationBio
	Alliance    *accountAllianceBio
}

type accountBioTarget struct {
	Entity string
	Kind   int16
	ID     int64
}

type accountBioSubmission struct {
	Target          accountBioTarget
	Body            string
	BodyFormat      string
	RenderedHTML    *string
	CharacterID     int32
	CharacterName   string
	CorporationID   *int32
	CorporationName *string
	AllianceID      *int32
	AllianceName    *string
	SubmittedAt     time.Time
}

type accountESIVolume struct {
	Hour     string `json:"hour"`
	Total    int64  `json:"total"`
	Errors   int64  `json:"errors"`
	NewItems int64  `json:"new_items"`
}

type accountESIMetrics struct {
	Volume       []accountESIVolume
	RequestCount int64
	AverageMS    *int32
	P95MS        *int32
}

type accountESILogQuery struct {
	CharacterID int32
	Limit       int
	Page        int
	Source      string
	Status      string
	Endpoint    string
	AfterID     *int64
}

type accountESILogRow struct {
	ID                int64     `json:"id"`
	Endpoint          string    `json:"endpoint"`
	Method            string    `json:"method"`
	StatusCode        *int16    `json:"status_code"`
	Success           bool      `json:"success"`
	ErrorMessage      *string   `json:"error_message"`
	ItemsReturned     *int32    `json:"items_returned"`
	NewItems          *int32    `json:"new_items"`
	Source            string    `json:"source"`
	RequestDurationMS *int32    `json:"request_duration_ms"`
	CreatedAt         time.Time `json:"created_at"`
	EndpointType      string    `json:"endpoint_type"`
	EndpointAction    string    `json:"endpoint_action"`
}

type accountESILogResult struct {
	Rows    []accountESILogRow
	Total   int64
	Sources []string
}

type accountAnnouncement struct {
	ID         int32      `json:"id"`
	Tier       int16      `json:"tier"`
	Title      string     `json:"title"`
	BodyMD     string     `json:"body_md"`
	BodyHTML   string     `json:"body_html"`
	Color      string     `json:"color"`
	Icon       *string    `json:"icon"`
	LinkURL    *string    `json:"link_url"`
	LinkLabel  *string    `json:"link_label"`
	StartsAt   time.Time  `json:"starts_at"`
	ExpiresAt  time.Time  `json:"expires_at"`
	CreatedBy  int64      `json:"created_by"`
	CreatedAt  time.Time  `json:"created_at"`
	UpdatedAt  time.Time  `json:"updated_at"`
	ArchivedAt *time.Time `json:"archived_at"`
}

// Announcement is the stable public shape shared by editorial announcements
// stored in Postgres and ephemeral ticker announcements stored in Valkey.
//
// Database-only bookkeeping such as created_by, updated_at, and archived_at is
// intentionally not part of the public contract. Ephemeral announcements never
// had those fields, and the frontend only consumes the presentation fields.
type Announcement struct {
	ID        int64     `json:"id"`
	Tier      int16     `json:"tier" enum:"1,2,3"`
	Title     string    `json:"title"`
	BodyMD    string    `json:"body_md"`
	BodyHTML  string    `json:"body_html"`
	Color     string    `json:"color" enum:"info,warning,danger,success"`
	Icon      *string   `json:"icon"`
	LinkURL   *string   `json:"link_url"`
	LinkLabel *string   `json:"link_label"`
	StartsAt  time.Time `json:"starts_at"`
	ExpiresAt time.Time `json:"expires_at"`
	CreatedAt time.Time `json:"created_at"`
}

type accountNotificationQuery struct {
	CharacterID int32
	DomainID    *int32
	Since       int64
	Limit       int
}

type accountNotificationReply struct {
	ID              int64   `json:"id"`
	TargetType      int16   `json:"target_type"`
	TargetID        int64   `json:"target_id"`
	ParentID        *int64  `json:"parent_id"`
	RootID          *int64  `json:"root_id"`
	BodyHTML        string  `json:"body_html"`
	CreatedAt       string  `json:"created_at"`
	CharacterID     int64   `json:"character_id"`
	CharacterName   string  `json:"character_name"`
	CorporationID   int64   `json:"corporation_id"`
	CorporationName string  `json:"corporation_name"`
	AllianceID      *int64  `json:"alliance_id"`
	AllianceName    *string `json:"alliance_name"`
	ParentCommentID int64   `json:"parent_comment_id"`
	ParentSnippet   string  `json:"parent_snippet"`
}

type accountStore interface {
	LoadPreferences(context.Context, int32) (map[string]any, error)
	SavePreferences(context.Context, int32, map[string]any, time.Time) error
	LoadBoardData(context.Context, *Principal) (accountBoardData, error)
	LoadOverview(context.Context, int32, time.Time) (accountOverview, error)
	LoadManageableEntities(
		context.Context,
		int32,
	) (accountManageableEntities, error)
	ResolveBioTarget(
		context.Context,
		int32,
		string,
	) (accountBioTarget, error)
	ClearBio(context.Context, accountBioTarget) error
	EnqueueBio(context.Context, accountBioSubmission) (int64, error)
	LoadESIMetrics(context.Context, int32, time.Time) (accountESIMetrics, error)
	LoadESILogs(context.Context, accountESILogQuery) (accountESILogResult, error)
	LoadActiveAnnouncements(context.Context, time.Time) ([]accountAnnouncement, error)
	LoadDismissedAnnouncementIDs(context.Context, int32) ([]int64, error)
	DismissAnnouncement(context.Context, int32, int64) error
	ResolveCommentDomainID(context.Context, string) (*int32, error)
	LoadNotificationReplies(
		context.Context,
		accountNotificationQuery,
	) ([]accountNotificationReply, error)
	MarkNotificationsRead(context.Context, int32, int64) (int64, error)
}

type bioEventDispatcher interface {
	BioPending(context.Context, accountBioSubmission, int64)
}

type accountService struct {
	auth          *authService
	store         accountStore
	storeErr      error
	cache         *redis.Client
	now           func() time.Time
	dispatch      bioEventDispatcher
	loadEphemeral func(context.Context, time.Time) []map[string]any
}

func newAccountService(opts Options) *accountService {
	service := &accountService{
		auth:  newAuthService(opts),
		cache: opts.Cache,
		now:   time.Now,
	}
	db, err := mutationDatabase(opts)
	if err != nil {
		service.storeErr = err
	} else {
		service.store = &postgresAccountStore{db: db}
	}
	if pool, ok := opts.DB.(*pgxpool.Pool); ok && pool != nil {
		if client, err := queue.New(queue.Options{Pool: pool}); err == nil {
			service.dispatch = &riverBioEventDispatcher{client: client}
		}
	}
	service.loadEphemeral = service.ephemeralAnnouncements
	return service
}

func (s *accountService) requireStore() error {
	if s.storeErr != nil {
		return s.storeErr
	}
	if s.store == nil {
		return apiError(http.StatusServiceUnavailable, "Account storage is not configured")
	}
	return nil
}

func (s *accountService) principal(
	ctx context.Context,
	req *legacyRequest,
) (*Principal, error) {
	if err := s.requireStore(); err != nil {
		return nil, err
	}
	return s.auth.requirePrincipal(ctx, req)
}

func decodeAccountBody(req *legacyRequest) (map[string]any, error) {
	limited := io.LimitReader(req.Body, accountBodyLimit+1)
	decoder := json.NewDecoder(limited)
	decoder.UseNumber()
	var body map[string]any
	if err := decoder.Decode(&body); err != nil {
		return nil, apiError(http.StatusBadRequest, "Body must be a JSON object")
	}
	if body == nil {
		return nil, apiError(http.StatusBadRequest, "Body must be a JSON object")
	}
	var extra any
	err := decoder.Decode(&extra)
	switch {
	case errors.Is(err, io.EOF):
		return body, nil
	case err == nil:
		return nil, apiError(http.StatusBadRequest, "Body must contain one JSON value")
	default:
		return nil, apiError(http.StatusBadRequest, "Invalid JSON body")
	}
}

type discordBioEventArgs struct {
	Type        string `json:"type"`
	QueueItemID int64  `json:"queueItemId"`
	EntityKind  string `json:"entityKind"`
	EntityID    int64  `json:"entityId"`
	EntityName  string `json:"entityName"`
	Submitter   struct {
		ID              int32   `json:"id"`
		Name            string  `json:"name"`
		CorporationID   *int32  `json:"corporation_id"`
		CorporationName *string `json:"corporation_name"`
		AllianceID      *int32  `json:"alliance_id"`
		AllianceName    *string `json:"alliance_name"`
	} `json:"submitter"`
	BodySnippet string `json:"bodySnippet"`
	BodyFormat  string `json:"bodyFormat"`
}

func (discordBioEventArgs) Kind() string { return "discord_events" }

type riverBioEventDispatcher struct {
	client *queue.Client
}

func (d *riverBioEventDispatcher) BioPending(
	ctx context.Context,
	submission accountBioSubmission,
	queueItemID int64,
) {
	if d == nil || d.client == nil {
		return
	}
	args := discordBioEventArgs{
		Type: "moderation.bio", QueueItemID: queueItemID,
		EntityKind:  submission.Target.Entity,
		EntityID:    submission.Target.ID,
		EntityName:  submission.CharacterName,
		BodySnippet: truncateRunes(submission.Body, 1800),
		BodyFormat:  submission.BodyFormat,
	}
	args.Submitter.ID = submission.CharacterID
	args.Submitter.Name = submission.CharacterName
	args.Submitter.CorporationID = submission.CorporationID
	args.Submitter.CorporationName = submission.CorporationName
	args.Submitter.AllianceID = submission.AllianceID
	args.Submitter.AllianceName = submission.AllianceName
	_, _ = queue.Dispatch(
		context.WithoutCancel(ctx),
		d.client,
		args,
		queue.Live,
	)
}
