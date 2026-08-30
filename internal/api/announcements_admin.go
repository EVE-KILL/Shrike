package api

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/danielgtaylor/huma/v2"
	"github.com/eve-kill/shrike/internal/queue"
	"github.com/jackc/pgx/v5"
)

const announcementColumns = `
	id,
	tier,
	title,
	body_md,
	body_html,
	color,
	icon,
	link_url,
	link_label,
	starts_at,
	expires_at,
	created_by,
	created_at,
	updated_at,
	archived_at`

type announcementAdminCreate struct {
	Tier      int16
	Title     string
	BodyMD    string
	BodyHTML  string
	Color     string
	Icon      *string
	LinkURL   *string
	LinkLabel *string
	StartsAt  time.Time
	ExpiresAt time.Time
	CreatedBy int32
}

type announcementAdminStore interface {
	List(
		context.Context,
		string,
		*int16,
		int,
		time.Time,
	) ([]map[string]any, error)
	Get(context.Context, int64) (map[string]any, error)
	Create(context.Context, announcementAdminCreate) (map[string]any, error)
	Update(
		context.Context,
		int64,
		map[string]any,
		time.Time,
	) (map[string]any, error)
	Archive(context.Context, int64, time.Time) (map[string]any, error)
}

type postgresAnnouncementAdminStore struct {
	db MutationDatabase
}

type announcementEventDispatcher interface {
	DispatchAnnouncement(
		context.Context,
		string,
		map[string]any,
	)
}

type riverAnnouncementEventDispatcher struct {
	client *queue.Client
}

type announcementsAdminService struct {
	auth     *authService
	store    announcementAdminStore
	storeErr error
	now      func() time.Time
	dispatch announcementEventDispatcher
}

func newAnnouncementsAdminService(opts Options) *announcementsAdminService {
	service := &announcementsAdminService{
		auth: newAuthService(opts),
		now:  time.Now,
	}
	db, err := mutationDatabase(opts)
	if err != nil {
		service.storeErr = err
	} else {
		service.store = &postgresAnnouncementAdminStore{db: db}
	}
	if pool := primaryPool(opts); pool != nil {
		if client, err := queue.New(queue.Options{Pool: pool}); err == nil {
			service.dispatch = &riverAnnouncementEventDispatcher{
				client: client,
			}
		}
	}
	return service
}

func (s *announcementsAdminService) requireStore() error {
	if s.storeErr != nil {
		return s.storeErr
	}
	if s.store == nil {
		return apiError(
			http.StatusServiceUnavailable,
			"Announcement storage is not configured",
		)
	}
	return nil
}

func registerAnnouncementAdminRoutes(a huma.API, opts Options) {
	registerAnnouncementAdminServiceRoutes(
		a,
		newAnnouncementsAdminService(opts),
	)
}

func registerAnnouncementAdminServiceRoutes(
	a huma.API,
	service *announcementsAdminService,
) {
	requiredSession := []map[string][]string{{"eveSession": {}}}
	registerLegacy(a, huma.Operation{
		OperationID: "announcement-admin-list",
		Method:      http.MethodGet,
		Path:        "/admin/announcements",
		Summary:     "Announcement administration",
		Tags:        []string{"admin", "announcements"},
		Security:    requiredSession,
	}, service.listHandler())
	registerLegacy(a, documentJSONBody[announcementCreateDocument](a, huma.Operation{
		OperationID: "announcement-admin-create",
		Method:      http.MethodPost,
		Path:        "/admin/announcements",
		Summary:     "Create an announcement",
		Tags:        []string{"admin", "announcements"},
		Security:    requiredSession,
	}), service.createHandler())
	registerLegacy(a, huma.Operation{
		OperationID: "announcement-admin-detail",
		Method:      http.MethodGet,
		Path:        "/admin/announcements/{id}",
		Summary:     "Announcement administration detail",
		Tags:        []string{"admin", "announcements"},
		Security:    requiredSession,
	}, service.detailHandler())
	registerLegacy(a, documentJSONBody[announcementUpdateDocument](a, huma.Operation{
		OperationID: "announcement-admin-update",
		Method:      http.MethodPatch,
		Path:        "/admin/announcements/{id}",
		Summary:     "Update an announcement",
		Tags:        []string{"admin", "announcements"},
		Security:    requiredSession,
	}), service.updateHandler())
	archive := service.archiveHandler()
	registerLegacy(a, huma.Operation{
		OperationID: "announcement-admin-archive",
		Method:      http.MethodDelete,
		Path:        "/admin/announcements/{id}",
		Summary:     "Archive an announcement",
		Tags:        []string{"admin", "announcements"},
		Security:    requiredSession,
	}, archive)
	registerLegacy(a, huma.Operation{
		OperationID: "announcement-admin-archive-compat",
		Method:      http.MethodPost,
		Path:        "/admin/announcements/{id}/archive",
		Summary:     "Archive an announcement",
		Tags:        []string{"admin", "announcements"},
		Security:    requiredSession,
	}, archive)
}

func (s *announcementsAdminService) authorize(
	ctx context.Context,
	req *legacyRequest,
) (*Principal, error) {
	setAccountNoStore(req.Huma)
	if err := s.requireStore(); err != nil {
		return nil, err
	}
	return requireContentAdmin(ctx, req, s.auth)
}

func (s *announcementsAdminService) listHandler() legacyHandler {
	return func(ctx context.Context, req *legacyRequest) (legacyPayload, error) {
		if _, err := s.authorize(ctx, req); err != nil {
			return legacyPayload{}, err
		}
		var tier *int16
		if value, ok := contentInteger(req.Query.Get("tier")); ok &&
			value > 0 {
			converted := int16(value)
			tier = &converted
		}
		rows, err := s.store.List(
			ctx,
			req.Query.Get("status"),
			tier,
			boundedContentInt(req.Query.Get("limit"), 50, 1, 200),
			s.now().UTC(),
		)
		if err != nil {
			return legacyPayload{}, err
		}
		return accountNoStorePayload(map[string]any{
			"announcements": rows,
		}), nil
	}
}

func (s *announcementsAdminService) detailHandler() legacyHandler {
	return func(ctx context.Context, req *legacyRequest) (legacyPayload, error) {
		if _, err := s.authorize(ctx, req); err != nil {
			return legacyPayload{}, err
		}
		id, err := contentID(
			req.Param("id"),
			"Invalid announcement id",
		)
		if err != nil {
			return legacyPayload{}, err
		}
		row, err := s.store.Get(ctx, id)
		if err != nil {
			return legacyPayload{}, err
		}
		if row == nil {
			return legacyPayload{}, apiError(
				http.StatusNotFound,
				"Announcement not found",
			)
		}
		return accountNoStorePayload(map[string]any{
			"announcement": row,
		}), nil
	}
}

type announcementCreateBody struct {
	Title     json.RawMessage `json:"title,omitempty" doc:"Announcement headline."`
	BodyMD    json.RawMessage `json:"body_md,omitempty" doc:"Announcement text, in Markdown."`
	Tier      json.RawMessage `json:"tier,omitempty" doc:"Severity tier."`
	Color     json.RawMessage `json:"color,omitempty" doc:"Accent color."`
	Icon      json.RawMessage `json:"icon,omitempty" doc:"Icon name."`
	LinkURL   json.RawMessage `json:"link_url,omitempty" doc:"Optional call-to-action URL."`
	LinkLabel json.RawMessage `json:"link_label,omitempty" doc:"Label for the call-to-action."`
	StartsAt  json.RawMessage `json:"starts_at,omitempty" doc:"When the announcement becomes visible."`
	ExpiresAt json.RawMessage `json:"expires_at,omitempty" doc:"When it stops being shown."`
}

// field resolves a wire key to its raw value, so the column loops below can
// stay table-driven instead of unrolling into one branch per field.
func (b *announcementUpdateBody) field(name string) json.RawMessage {
	switch name {
	case "title":
		return b.Title
	case "body_md":
		return b.BodyMD
	case "tier":
		return b.Tier
	case "color":
		return b.Color
	case "icon":
		return b.Icon
	case "link_url":
		return b.LinkURL
	case "link_label":
		return b.LinkLabel
	case "starts_at":
		return b.StartsAt
	case "expires_at":
		return b.ExpiresAt
	}
	return nil
}

// announcementUpdateBody carries the same fields as create. Every one is
// optional, and an absent field leaves the stored value alone.
type announcementUpdateBody struct {
	Title     json.RawMessage `json:"title,omitempty" doc:"New headline."`
	BodyMD    json.RawMessage `json:"body_md,omitempty" doc:"New text, in Markdown."`
	Tier      json.RawMessage `json:"tier,omitempty" doc:"New severity tier."`
	Color     json.RawMessage `json:"color,omitempty" doc:"New accent color."`
	Icon      json.RawMessage `json:"icon,omitempty" doc:"New icon name."`
	LinkURL   json.RawMessage `json:"link_url,omitempty" doc:"New call-to-action URL."`
	LinkLabel json.RawMessage `json:"link_label,omitempty" doc:"New call-to-action label."`
	StartsAt  json.RawMessage `json:"starts_at,omitempty" doc:"New visibility start."`
	ExpiresAt json.RawMessage `json:"expires_at,omitempty" doc:"New expiry."`
}

func (s *announcementsAdminService) createHandler() legacyHandler {
	return func(ctx context.Context, req *legacyRequest) (legacyPayload, error) {
		if err := requireSameOriginMutation(req.Huma); err != nil {
			return legacyPayload{}, err
		}
		principal, err := s.authorize(ctx, req)
		if err != nil {
			return legacyPayload{}, err
		}
		body, err := decodeJSONBody[announcementCreateBody](req, contentBodyLimit)
		if err != nil {
			return legacyPayload{}, err
		}
		tier, ok := contentInteger(rawJSONValue(body.Tier))
		if !ok || (tier != 1 && tier != 2 && tier != 3) {
			return legacyPayload{}, apiError(
				http.StatusBadRequest,
				"tier must be 1, 2, or 3",
			)
		}
		title := strings.TrimSpace(stringField(rawJSONValue(body.Title)))
		if title == "" {
			return legacyPayload{}, apiError(
				http.StatusBadRequest,
				"title is required",
			)
		}
		expiresAt, ok := parseContentTime(rawJSONValue(body.ExpiresAt))
		if !ok {
			return legacyPayload{}, apiError(
				http.StatusBadRequest,
				"expires_at is required and must be a valid date",
			)
		}
		startsAt := s.now().UTC()
		if raw, found := rawJSONField(body.StartsAt); found && raw != nil &&
			stringField(raw) != "" {
			startsAt, ok = parseContentTime(raw)
			if !ok {
				return legacyPayload{}, apiError(
					http.StatusBadRequest,
					"starts_at must be a valid date",
				)
			}
		}
		if !expiresAt.After(startsAt) {
			return legacyPayload{}, apiError(
				http.StatusBadRequest,
				"expires_at must be after starts_at",
			)
		}
		bodyMD := strings.TrimSpace(stringField(rawJSONValue(body.BodyMD)))
		input := announcementAdminCreate{
			Tier: int16(tier), Title: title,
			BodyMD:   bodyMD,
			BodyHTML: renderAnnouncementMarkdown(bodyMD),
			Color:    announcementColor(rawJSONValue(body.Color)),
			Icon:     optionalTrimmedString(rawJSONValue(body.Icon), 512),
			LinkURL: optionalTrimmedString(
				rawJSONValue(body.LinkURL), 4096,
			),
			LinkLabel: optionalTrimmedString(
				rawJSONValue(body.LinkLabel), 512,
			),
			StartsAt: startsAt, ExpiresAt: expiresAt,
			CreatedBy: principal.CharacterID,
		}
		row, err := s.store.Create(ctx, input)
		if err != nil {
			return legacyPayload{}, err
		}
		now := s.now().UTC()
		if !startsAt.After(now) && expiresAt.After(now) {
			s.dispatchEvent(ctx, "new", row)
		}
		return accountNoStorePayload(map[string]any{
			"announcement": row,
		}), nil
	}
}

func (s *announcementsAdminService) updateHandler() legacyHandler {
	return func(ctx context.Context, req *legacyRequest) (legacyPayload, error) {
		if err := requireSameOriginMutation(req.Huma); err != nil {
			return legacyPayload{}, err
		}
		if _, err := s.authorize(ctx, req); err != nil {
			return legacyPayload{}, err
		}
		id, err := contentID(
			req.Param("id"),
			"Invalid announcement id",
		)
		if err != nil {
			return legacyPayload{}, err
		}
		body, err := decodeJSONBody[announcementUpdateBody](req, contentBodyLimit)
		if err != nil {
			return legacyPayload{}, err
		}
		update := map[string]any{}
		if raw, found := rawJSONField(body.Tier); found {
			value, ok := contentInteger(raw)
			if !ok || (value != 1 && value != 2 && value != 3) {
				return legacyPayload{}, apiError(
					http.StatusBadRequest,
					"tier must be 1, 2, or 3",
				)
			}
			update["tier"] = int16(value)
		}
		if raw, found := rawJSONField(body.Title); found {
			title := strings.TrimSpace(stringField(raw))
			if title == "" {
				return legacyPayload{}, apiError(
					http.StatusBadRequest,
					"title cannot be empty",
				)
			}
			update["title"] = title
		}
		if raw, found := rawJSONField(body.BodyMD); found {
			bodyMD := strings.TrimSpace(stringField(raw))
			update["body_md"] = bodyMD
			update["body_html"] = renderAnnouncementMarkdown(bodyMD)
		}
		if raw, found := rawJSONField(body.Color); found {
			update["color"] = announcementColor(raw)
		}
		for bodyKey, column := range map[string]string{
			"icon": "icon", "link_url": "link_url",
			"link_label": "link_label",
		} {
			if raw, found := rawJSONField(body.field(bodyKey)); found {
				update[column] = optionalTrimmedString(raw, 4096)
			}
		}
		for bodyKey, column := range map[string]string{
			"starts_at": "starts_at", "expires_at": "expires_at",
		} {
			if raw, found := rawJSONField(body.field(bodyKey)); found {
				value, ok := parseContentTime(raw)
				if !ok {
					return legacyPayload{}, apiError(
						http.StatusBadRequest,
						"Invalid "+bodyKey,
					)
				}
				update[column] = value
			}
		}
		row, err := s.store.Update(ctx, id, update, s.now().UTC())
		if err != nil {
			return legacyPayload{}, err
		}
		if row == nil {
			return legacyPayload{}, apiError(
				http.StatusNotFound,
				"Announcement not found",
			)
		}
		now := s.now().UTC()
		starts, startsOK := timeFrom(row["starts_at"])
		expires, expiresOK := timeFrom(row["expires_at"])
		if row["archived_at"] == nil && startsOK && expiresOK &&
			!starts.After(now) && expires.After(now) {
			s.dispatchEvent(ctx, "updated", row)
		}
		return accountNoStorePayload(map[string]any{
			"announcement": row,
		}), nil
	}
}

func (s *announcementsAdminService) archiveHandler() legacyHandler {
	return func(ctx context.Context, req *legacyRequest) (legacyPayload, error) {
		if err := requireSameOriginMutation(req.Huma); err != nil {
			return legacyPayload{}, err
		}
		if _, err := s.authorize(ctx, req); err != nil {
			return legacyPayload{}, err
		}
		id, err := contentID(
			req.Param("id"),
			"Invalid announcement id",
		)
		if err != nil {
			return legacyPayload{}, err
		}
		row, err := s.store.Archive(ctx, id, s.now().UTC())
		if err != nil {
			return legacyPayload{}, err
		}
		if row == nil {
			return legacyPayload{}, apiError(
				http.StatusNotFound,
				"Announcement not found",
			)
		}
		s.dispatchEvent(ctx, "archived", row)
		return accountNoStorePayload(map[string]any{
			"announcement": row,
		}), nil
	}
}

func announcementColor(value any) string {
	color := stringField(value)
	switch color {
	case "info", "warning", "danger", "success":
		return color
	default:
		return "info"
	}
}

func (s *announcementsAdminService) dispatchEvent(
	ctx context.Context,
	event string,
	row map[string]any,
) {
	if s.dispatch != nil {
		s.dispatch.DispatchAnnouncement(
			context.WithoutCancel(ctx),
			event,
			row,
		)
	}
}

func (d *riverAnnouncementEventDispatcher) DispatchAnnouncement(
	ctx context.Context,
	event string,
	row map[string]any,
) {
	if d == nil || d.client == nil {
		return
	}
	announcement := make(map[string]any)
	for _, key := range []string{
		"id", "tier", "title", "body_md", "body_html", "color",
		"icon", "link_url", "link_label", "starts_at", "expires_at",
	} {
		announcement[key] = row[key]
	}
	raw, err := json.Marshal(normalizeJSON(map[string]any{
		"event_type":   event,
		"announcement": announcement,
	}))
	if err != nil {
		return
	}
	_, _ = queue.Dispatch(
		ctx,
		d.client,
		queue.AnnouncementEventArgs{Payload: raw},
		queue.Live,
	)
}

func (s *postgresAnnouncementAdminStore) List(
	ctx context.Context,
	status string,
	tier *int16,
	limit int,
	now time.Time,
) ([]map[string]any, error) {
	filters := []string{}
	args := []any{}
	add := func(value any) string {
		args = append(args, value)
		return fmt.Sprintf("$%d", len(args))
	}
	if tier != nil {
		filters = append(filters, "tier = "+add(*tier))
	}
	switch status {
	case "active":
		filters = append(
			filters,
			"archived_at IS NULL",
			"starts_at <= "+add(now),
			"expires_at > "+add(now),
		)
	case "scheduled":
		filters = append(
			filters,
			"archived_at IS NULL",
			"starts_at > "+add(now),
		)
	case "expired":
		filters = append(
			filters,
			"archived_at IS NULL",
			"expires_at <= "+add(now),
		)
	case "archived":
		filters = append(filters, "archived_at IS NOT NULL")
	}
	query := `SELECT ` + announcementColumns + ` FROM announcements`
	if len(filters) > 0 {
		query += ` WHERE ` + strings.Join(filters, " AND ")
	}
	query += ` ORDER BY created_at DESC LIMIT ` + add(limit)
	return queryMaps(ctx, s.db, query, args...)
}

func (s *postgresAnnouncementAdminStore) Get(
	ctx context.Context,
	id int64,
) (map[string]any, error) {
	return queryMap(ctx, s.db, `
		SELECT `+announcementColumns+`
		FROM announcements
		WHERE id = $1
		LIMIT 1`,
		id,
	)
}

func (s *postgresAnnouncementAdminStore) Create(
	ctx context.Context,
	input announcementAdminCreate,
) (result map[string]any, err error) {
	tx, err := s.db.Begin(ctx)
	if err != nil {
		return nil, err
	}
	defer func() { _ = tx.Rollback(ctx) }()
	if err := lockAnnouncementTiers(ctx, tx); err != nil {
		return nil, err
	}
	if err := enforceAnnouncementLimit(
		ctx,
		tx,
		input.Tier,
		input.StartsAt,
		input.ExpiresAt,
		0,
	); err != nil {
		return nil, err
	}
	result, err = queryMap(ctx, txDatabase{tx}, `
		INSERT INTO announcements (
			tier,
			title,
			body_md,
			body_html,
			color,
			icon,
			link_url,
			link_label,
			starts_at,
			expires_at,
			created_by
		)
		VALUES (
			$1, $2, $3, $4, $5, $6,
			$7, $8, $9, $10, $11
		)
		RETURNING `+announcementColumns,
		input.Tier,
		input.Title,
		input.BodyMD,
		input.BodyHTML,
		input.Color,
		input.Icon,
		input.LinkURL,
		input.LinkLabel,
		input.StartsAt,
		input.ExpiresAt,
		input.CreatedBy,
	)
	if err != nil {
		return nil, err
	}
	if err := tx.Commit(ctx); err != nil {
		return nil, err
	}
	return result, nil
}

func (s *postgresAnnouncementAdminStore) Update(
	ctx context.Context,
	id int64,
	update map[string]any,
	now time.Time,
) (result map[string]any, err error) {
	tx, err := s.db.Begin(ctx)
	if err != nil {
		return nil, err
	}
	defer func() { _ = tx.Rollback(ctx) }()
	if err := lockAnnouncementTiers(ctx, tx); err != nil {
		return nil, err
	}
	current, err := queryMap(ctx, txDatabase{tx}, `
		SELECT `+announcementColumns+`
		FROM announcements
		WHERE id = $1
		FOR UPDATE`,
		id,
	)
	if err != nil || current == nil {
		return current, err
	}
	tier := int16From(current["tier"])
	if value, found := update["tier"]; found {
		tier = int16From(value)
	}
	startsAt, _ := timeFrom(current["starts_at"])
	if value, found := update["starts_at"]; found {
		startsAt, _ = timeFrom(value)
	}
	expiresAt, _ := timeFrom(current["expires_at"])
	if value, found := update["expires_at"]; found {
		expiresAt, _ = timeFrom(value)
	}
	if !expiresAt.After(startsAt) {
		return nil, apiError(
			http.StatusBadRequest,
			"expires_at must be after starts_at",
		)
	}
	if err := enforceAnnouncementLimit(
		ctx,
		tx,
		tier,
		startsAt,
		expiresAt,
		id,
	); err != nil {
		return nil, err
	}
	update["updated_at"] = now
	keys := make([]string, 0, len(update))
	for key := range update {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	args := make([]any, 0, len(keys)+1)
	assignments := make([]string, 0, len(keys))
	for _, key := range keys {
		args = append(args, update[key])
		assignments = append(
			assignments,
			fmt.Sprintf("%s = $%d", key, len(args)),
		)
	}
	args = append(args, id)
	result, err = queryMap(ctx, txDatabase{tx}, `
		UPDATE announcements
		SET `+strings.Join(assignments, ", ")+`
		WHERE id = $`+strconv.Itoa(len(args))+`
		RETURNING `+announcementColumns,
		args...,
	)
	if err != nil {
		return nil, err
	}
	if err := tx.Commit(ctx); err != nil {
		return nil, err
	}
	return result, nil
}

func (s *postgresAnnouncementAdminStore) Archive(
	ctx context.Context,
	id int64,
	now time.Time,
) (map[string]any, error) {
	return queryMap(ctx, s.db, `
		UPDATE announcements
		SET archived_at = $2,
		    updated_at = $2
		WHERE id = $1
		RETURNING `+announcementColumns,
		id,
		now,
	)
}

type txDatabase struct {
	pgx.Tx
}

func (d txDatabase) Ping(context.Context) error { return nil }

func lockAnnouncementTiers(ctx context.Context, tx pgx.Tx) error {
	// Both locks are always taken in the same order. This closes the
	// count-then-insert race in the old implementation without changing the
	// announcements schema.
	_, err := tx.Exec(ctx, `
		SELECT pg_advisory_xact_lock(41002),
		       pg_advisory_xact_lock(41003)`)
	return err
}

func enforceAnnouncementLimit(
	ctx context.Context,
	tx pgx.Tx,
	tier int16,
	startsAt time.Time,
	expiresAt time.Time,
	excludeID int64,
) error {
	if tier != 2 && tier != 3 {
		return nil
	}
	var count int
	if err := tx.QueryRow(ctx, `
		SELECT count(*)::int
		FROM announcements
		WHERE tier = $1
		  AND id <> $4
		  AND archived_at IS NULL
		  AND starts_at <= $3
		  AND expires_at > $2`,
		tier,
		startsAt,
		expiresAt,
		excludeID,
	).Scan(&count); err != nil {
		return err
	}
	maximum := 2
	label := "banner"
	if tier == 3 {
		maximum = 1
		label = "modal"
	}
	if count < maximum {
		return nil
	}
	plural := ""
	if maximum > 1 {
		plural = "s"
	}
	return apiError(
		http.StatusConflict,
		fmt.Sprintf(
			"Maximum %d active %s%s allowed. Archive or let existing ones expire first.",
			maximum,
			label,
			plural,
		),
	)
}
