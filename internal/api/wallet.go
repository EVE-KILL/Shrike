package api

import (
	"context"
	"fmt"
	"math"
	"math/big"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/danielgtaylor/huma/v2"
	"github.com/eve-kill/shrike/internal/campaign"
	"github.com/eve-kill/shrike/internal/queue"
	"github.com/jackc/pgx/v5/pgxpool"
)

const (
	walletPageSize            = 50
	walletMaximumPage         = 1000
	walletReferencePageSize   = 100
	walletSettlementPageSize  = 250
	walletDepositsEnabledKey  = "ek-wallet:deposits-enabled-at"
	walletCorporationName     = "EVE-KILL.com"
	walletCorporationTicker   = "EVEKI"
	walletRequiredScope       = "esi-wallet.read_corporation_wallets.v1"
	walletCharacterScope      = "esi-wallet.read_character_wallet.v1"
	walletPublicCacheDuration = time.Minute
)

var walletAuthorizationScopes = []string{
	"publicData",
	"esi-killmails.read_killmails.v1",
	"esi-killmails.read_corporation_killmails.v1",
	walletCharacterScope,
	walletRequiredScope,
}

type walletService struct {
	auth     *authService
	db       Database
	queue    *queue.Client
	queueErr error
}

func newWalletService(opts Options) *walletService {
	service := &walletService{
		auth: newAuthService(opts),
		db:   opts.DB,
	}
	if pool, ok := opts.DB.(*pgxpool.Pool); ok && pool != nil {
		service.queue, service.queueErr = queue.New(queue.Options{Pool: pool})
	} else {
		service.queueErr = fmt.Errorf("wallet queue is not configured")
	}
	return service
}

// registerWalletRoutes installs public wallet transparency, signed-in EK
// Wallet views, and corporation-wallet administration into the shared API.
func registerWalletRoutes(a huma.API, opts Options) {
	service := newWalletService(opts)
	requiredSession := []map[string][]string{{"eveSession": {}}}

	registerLegacy(a, huma.Operation{
		OperationID: "wallet-public",
		Method:      http.MethodGet,
		Path:        "/wallet",
		Summary:     "Public EVE-KILL corporation wallet",
		Tags:        []string{"wallet"},
	}, routeJSONCache(
		opts,
		walletPublicCacheDuration,
		"public, max-age=30, s-maxage=60, stale-while-revalidate=60",
		service.publicWalletHandler(),
	))

	accountWallet := service.accountWalletHandler()
	for _, route := range []struct {
		id, path string
	}{
		{"wallet-account", "/me/wallet"},
		{"wallet-account-legacy", "/user/wallet"},
	} {
		registerLegacy(a, huma.Operation{
			OperationID: route.id,
			Method:      http.MethodGet,
			Path:        route.path,
			Summary:     "Current user's EK Wallet",
			Tags:        []string{"account", "wallet"},
			Security:    requiredSession,
		}, accountWallet)
	}

	accountBalance := service.accountBalanceHandler()
	for _, route := range []struct {
		id, path string
	}{
		{"wallet-account-balance", "/me/wallet/balance"},
		{"wallet-account-balance-legacy", "/user/wallet/balance"},
	} {
		registerLegacy(a, huma.Operation{
			OperationID: route.id,
			Method:      http.MethodGet,
			Path:        route.path,
			Summary:     "Current user's EK Wallet balance",
			Tags:        []string{"account", "wallet"},
			Security:    requiredSession,
		}, accountBalance)
	}

	registerLegacy(a, huma.Operation{
		OperationID: "wallet-admin",
		Method:      http.MethodGet,
		Path:        "/admin/wallet",
		Summary:     "Corporation wallet administration",
		Tags:        []string{"admin", "wallet"},
		Security:    requiredSession,
	}, service.adminWalletHandler())
	registerLegacy(a, huma.Operation{
		OperationID: "wallet-admin-sync",
		Method:      http.MethodPost,
		Path:        "/admin/wallet/sync",
		Summary:     "Queue a corporation wallet sync",
		Tags:        []string{"admin", "wallet"},
		Security:    requiredSession,
	}, service.adminWalletSyncHandler())
	registerLegacy(a, huma.Operation{
		OperationID: "wallet-admin-authorize",
		Method:      http.MethodGet,
		Path:        "/admin/wallet/authorize",
		Summary:     "Create a corporation-wallet authorization URL",
		Tags:        []string{"admin", "wallet", "auth"},
		Security:    requiredSession,
	}, service.adminWalletAuthorizeHandler())
}

func (s *walletService) publicWalletHandler() legacyHandler {
	return func(ctx context.Context, req *legacyRequest) (legacyPayload, error) {
		page := boundedWalletPage(req.Query.Get("page"))
		division := walletDivision(req.Query.Get("division"))
		args := []any{campaign.EveKillCorporationID, walletPageSize + 1,
			(page - 1) * walletPageSize}
		divisionSQL := ""
		if division != nil {
			args = append(args, *division)
			divisionSQL = fmt.Sprintf(" AND journal.division = $%d", len(args))
		}

		results, err := queryMapsConcurrent(ctx, s.db,
			databaseQuery{
				SQL: `
					SELECT corporation_id, name, ticker
					FROM corporations WHERE corporation_id = $1 LIMIT 1`,
				Args: []any{campaign.EveKillCorporationID},
			},
			databaseQuery{
				SQL: `
					SELECT last_journal_sync
					FROM corporation_wallet_tokens
					WHERE corporation_id = $1 LIMIT 1`,
				Args: []any{campaign.EveKillCorporationID},
			},
			databaseQuery{
				SQL: `
					SELECT COALESCE(SUM(balance), 0)::text AS total_balance
					FROM corporation_wallet_balances WHERE corporation_id = $1`,
				Args: []any{campaign.EveKillCorporationID},
			},
			databaseQuery{
				SQL: fmt.Sprintf(`
					SELECT journal.division, journal.journal_id, journal.date,
					       journal.ref_type, journal.description,
					       journal.amount::text AS amount,
					       journal.balance::text AS balance,
					       journal.first_party_id, journal.second_party_id,
					       journal.context_id, journal.context_id_type,
					       journal.reason, journal.tax::text AS tax,
					       journal.tax_receiver_id
					FROM corporation_wallet_journal journal
					WHERE journal.corporation_id = $1%s
					ORDER BY journal.date DESC, journal.journal_id DESC
					LIMIT $2 OFFSET $3`, divisionSQL),
				Args: args,
			},
		)
		if err != nil {
			return legacyPayload{}, err
		}

		journal := results[3]
		hasMore := len(journal) > walletPageSize
		if hasMore {
			journal = journal[:walletPageSize]
		}
		return jsonPayload(map[string]any{
			"corporation": walletCorporation(results[0]),
			"totalBalance": firstString(
				results[2], "total_balance", "0.00",
			),
			"lastSynced": firstValue(results[1], "last_journal_sync"),
			"journal":    journal,
			"page":       page,
			"division":   pointerValue(division),
			"hasMore":    hasMore,
			"pageSize":   walletPageSize,
		}), nil
	}
}

func (s *walletService) accountBalanceHandler() legacyHandler {
	return func(ctx context.Context, req *legacyRequest) (legacyPayload, error) {
		setAccountNoStore(req.Huma)
		principal, err := s.auth.requirePrincipal(ctx, req)
		if err != nil {
			return legacyPayload{}, err
		}
		row, err := queryMap(ctx, s.db, `
			SELECT balance::text AS total_balance,
			       reserved_balance::text AS reserved_balance,
			       (balance - reserved_balance)::text AS available_balance
			FROM ek_wallet_accounts
			WHERE character_id = $1
			LIMIT 1`, principal.CharacterID)
		if err != nil {
			return legacyPayload{}, err
		}
		return accountNoStorePayload(walletBalance(row)), nil
	}
}

func (s *walletService) accountWalletHandler() legacyHandler {
	return func(ctx context.Context, req *legacyRequest) (legacyPayload, error) {
		setAccountNoStore(req.Huma)
		principal, err := s.auth.requirePrincipal(ctx, req)
		if err != nil {
			return legacyPayload{}, err
		}
		page := boundedWalletPage(req.Query.Get("page"))
		results, err := queryMapsConcurrent(ctx, s.db,
			databaseQuery{
				SQL: `
					SELECT balance::text AS total_balance,
					       reserved_balance::text AS reserved_balance,
					       (balance - reserved_balance)::text AS available_balance
					FROM ek_wallet_accounts
					WHERE character_id = $1 LIMIT 1`,
				Args: []any{principal.CharacterID},
			},
			databaseQuery{
				SQL: `
					SELECT id, type, amount::text AS amount,
					       balance_after::text AS balance_after,
					       description, external_reference,
					       related_transaction_id, campaign_id, metadata, created_at
					FROM ek_wallet_transactions
					WHERE character_id = $1
					ORDER BY created_at DESC, id DESC
					LIMIT $2 OFFSET $3`,
				Args: []any{
					principal.CharacterID,
					walletPageSize + 1,
					(page - 1) * walletPageSize,
				},
			},
			databaseQuery{
				SQL: `
					SELECT id, external_reference, transaction_type,
					       amount::text AS amount, description, expires_at, created_at
					FROM ek_wallet_reservations
					WHERE character_id = $1 AND status = 0
					ORDER BY created_at DESC, id DESC
					LIMIT 50`,
				Args: []any{principal.CharacterID},
			},
			databaseQuery{
				SQL: `
					SELECT corporation_id, name, ticker
					FROM corporations WHERE corporation_id = $1 LIMIT 1`,
				Args: []any{campaign.EveKillCorporationID},
			},
			databaseQuery{
				SQL: `
					SELECT last_journal_sync
					FROM corporation_wallet_tokens
					WHERE corporation_id = $1 LIMIT 1`,
				Args: []any{campaign.EveKillCorporationID},
			},
			databaseQuery{
				SQL:  `SELECT value FROM config WHERE key = $1 LIMIT 1`,
				Args: []any{walletDepositsEnabledKey},
			},
		)
		if err != nil {
			return legacyPayload{}, err
		}

		transactions := results[1]
		hasMore := len(transactions) > walletPageSize
		if hasMore {
			transactions = transactions[:walletPageSize]
		}
		balance := walletBalance(firstRow(results[0]))
		return accountNoStorePayload(map[string]any{
			"character": map[string]any{
				"character_id":   principal.CharacterID,
				"character_name": principal.CharacterName,
			},
			"corporation":       walletCorporation(results[3]),
			"balance":           balance["balance"],
			"totalBalance":      balance["totalBalance"],
			"reservedBalance":   balance["reservedBalance"],
			"availableBalance":  balance["availableBalance"],
			"lastSynced":        firstValue(results[4], "last_journal_sync"),
			"depositsEnabledAt": firstValue(results[5], "value"),
			"transactions":      transactions,
			"reservations":      results[2],
			"page":              page,
			"hasMore":           hasMore,
			"pageSize":          walletPageSize,
		}), nil
	}
}

func (s *walletService) adminWalletHandler() legacyHandler {
	return func(ctx context.Context, req *legacyRequest) (legacyPayload, error) {
		setAccountNoStore(req.Huma)
		if _, err := s.requireAdmin(ctx, req); err != nil {
			return legacyPayload{}, err
		}
		page := boundedWalletPage(req.Query.Get("page"))
		division := walletDivision(req.Query.Get("division"))
		journalArgs := []any{
			campaign.EveKillCorporationID,
			walletPageSize + 1,
			(page - 1) * walletPageSize,
		}
		divisionSQL := ""
		if division != nil {
			journalArgs = append(journalArgs, *division)
			divisionSQL = fmt.Sprintf(
				" AND journal.division = $%d", len(journalArgs),
			)
		}

		results, err := queryMapsConcurrent(ctx, s.db,
			databaseQuery{
				SQL: `
					SELECT corporation_id, name, ticker
					FROM corporations WHERE corporation_id = $1 LIMIT 1`,
				Args: []any{campaign.EveKillCorporationID},
			},
			databaseQuery{
				SQL: `
					SELECT authorized_character_id, authorized_character_name,
					       authorized_by_admin_character_id, token_expiry, scopes,
					       disabled, last_balance_sync, last_journal_sync,
					       last_error, created_at, updated_at
					FROM corporation_wallet_tokens
					WHERE corporation_id = $1 LIMIT 1`,
				Args: []any{campaign.EveKillCorporationID},
			},
			databaseQuery{
				SQL: `
					SELECT division, balance::text AS balance, updated_at
					FROM corporation_wallet_balances
					WHERE corporation_id = $1
					ORDER BY division`,
				Args: []any{campaign.EveKillCorporationID},
			},
			databaseQuery{
				SQL: fmt.Sprintf(`
					SELECT journal.division, journal.journal_id, journal.date,
					       journal.ref_type, journal.description,
					       journal.amount::text AS amount,
					       journal.balance::text AS balance,
					       journal.first_party_id, journal.second_party_id,
					       journal.context_id, journal.context_id_type,
					       journal.reason, journal.tax::text AS tax,
					       journal.tax_receiver_id
					FROM corporation_wallet_journal journal
					WHERE journal.corporation_id = $1%s
					ORDER BY journal.date DESC, journal.journal_id DESC
					LIMIT $2 OFFSET $3`, divisionSQL),
				Args: journalArgs,
			},
			databaseQuery{
				SQL: `
					SELECT pool.campaign_id, campaign.name AS campaign_name,
					       pool.status AS pool_status,
					       pool.funded_total::text AS funded_total,
					       pool.finalized_at, result.rank, result.character_id,
					       result.character_name,
					       result.metric_value::text AS metric_value,
					       result.payout_percentage,
					       result.payout_amount::text AS payout_amount,
					       result.claimed_at, result.paid_at, result.payment_note
					FROM campaign_prize_pools pool
					JOIN campaigns campaign
					  ON campaign.campaign_id = pool.campaign_id
					LEFT JOIN campaign_prize_results result
					  ON result.campaign_id = pool.campaign_id
					WHERE pool.status IN (1, 2)
					ORDER BY pool.finalized_at DESC NULLS LAST, result.rank
					LIMIT $1`,
				Args: []any{walletSettlementPageSize},
			},
			databaseQuery{
				SQL: `
					SELECT reference.corporation_id, reference.division,
					       reference.journal_id, reference.reference_type,
					       reference.reference_id, reference.status,
					       reference.amount::text AS amount, reference.note,
					       reference.created_at, journal.date,
					       journal.first_party_id, journal.reason
					FROM wallet_journal_references reference
					JOIN corporation_wallet_journal journal
					  ON journal.corporation_id = reference.corporation_id
					 AND journal.division = reference.division
					 AND journal.journal_id = reference.journal_id
					WHERE reference.corporation_id = $1
					ORDER BY journal.date DESC, journal.journal_id DESC
					LIMIT $2`,
				Args: []any{
					campaign.EveKillCorporationID,
					walletReferencePageSize,
				},
			},
		)
		if err != nil {
			return legacyPayload{}, err
		}

		journal := results[3]
		hasMore := len(journal) > walletPageSize
		if hasMore {
			journal = journal[:walletPageSize]
		}
		return accountNoStorePayload(map[string]any{
			"corporation":      walletCorporation(results[0]),
			"authorization":    firstRow(results[1]),
			"requiredScopes":   append([]string(nil), walletAuthorizationScopes...),
			"balances":         results[2],
			"totalBalance":     sumWalletBalances(results[2]),
			"journal":          journal,
			"page":             page,
			"division":         pointerValue(division),
			"hasMore":          hasMore,
			"pageSize":         walletPageSize,
			"prizeSettlements": results[4],
			"walletReferences": results[5],
		}), nil
	}
}

func (s *walletService) adminWalletSyncHandler() legacyHandler {
	return func(ctx context.Context, req *legacyRequest) (legacyPayload, error) {
		setAccountNoStore(req.Huma)
		if err := requireSameOriginMutation(req.Huma); err != nil {
			return legacyPayload{}, err
		}
		if _, err := s.requireAdmin(ctx, req); err != nil {
			return legacyPayload{}, err
		}
		row, err := queryMap(ctx, s.db, `
			SELECT corporation_id
			FROM corporation_wallet_tokens
			WHERE corporation_id = $1 AND disabled = false
			LIMIT 1`, campaign.EveKillCorporationID)
		if err != nil {
			return legacyPayload{}, err
		}
		if row == nil {
			return legacyPayload{}, apiError(
				http.StatusConflict,
				"Authorize the corporation wallet before syncing",
			)
		}
		if s.queueErr != nil || s.queue == nil {
			return legacyPayload{}, apiError(
				http.StatusServiceUnavailable,
				"The wallet sync worker is unavailable",
			)
		}
		if _, err := queue.Dispatch(
			ctx,
			s.queue,
			queue.CorporationWalletArgs{
				CorporationID: campaign.EveKillCorporationID,
				Force:         true,
			},
			queue.Immediate,
		); err != nil {
			return legacyPayload{}, apiError(
				http.StatusServiceUnavailable,
				"The wallet sync worker is unavailable",
			)
		}
		return accountNoStorePayload(map[string]any{"queued": true}), nil
	}
}

func (s *walletService) adminWalletAuthorizeHandler() legacyHandler {
	return func(ctx context.Context, req *legacyRequest) (legacyPayload, error) {
		setAccountNoStore(req.Huma)
		admin, err := s.requireAdmin(ctx, req)
		if err != nil {
			return legacyPayload{}, err
		}
		authorizationURL, cookie, err := s.auth.beginWalletAuthorization(
			ctx,
			admin.CharacterID,
			campaign.EveKillCorporationID,
		)
		if err != nil {
			return legacyPayload{}, err
		}
		headers := make(http.Header)
		headers.Set("Cache-Control", "private, no-store")
		headers.Set("Pragma", "no-cache")
		headers.Add("Set-Cookie", cookie.String())
		return legacyPayload{
			Headers: headers,
			Body:    map[string]any{"url": authorizationURL},
		}, nil
	}
}

func (s *walletService) requireAdmin(
	ctx context.Context,
	req *legacyRequest,
) (*Principal, error) {
	principal, err := s.auth.requirePrincipal(ctx, req)
	if err != nil {
		return nil, err
	}
	if !principal.IsAdmin {
		return nil, apiError(http.StatusForbidden, "Administrator access required")
	}
	return principal, nil
}

func boundedWalletPage(raw string) int {
	value, err := strconv.ParseFloat(strings.TrimSpace(raw), 64)
	if err != nil || math.IsNaN(value) || math.IsInf(value, 0) || value < 1 {
		return 1
	}
	page := int(math.Floor(value))
	if page > walletMaximumPage {
		return walletMaximumPage
	}
	return page
}

func walletDivision(raw string) *int16 {
	value, err := strconv.ParseFloat(strings.TrimSpace(raw), 64)
	if err != nil || math.Trunc(value) != value || value < 1 || value > 7 {
		return nil
	}
	division := int16(value)
	return &division
}

func walletCorporation(rows []map[string]any) map[string]any {
	if len(rows) > 0 {
		return rows[0]
	}
	return map[string]any{
		"corporation_id": campaign.EveKillCorporationID,
		"name":           walletCorporationName,
		"ticker":         walletCorporationTicker,
	}
}

func walletBalance(row map[string]any) map[string]any {
	total := mapString(row, "total_balance", "0.00")
	reserved := mapString(row, "reserved_balance", "0.00")
	available := mapString(row, "available_balance", "0.00")
	return map[string]any{
		// Compatibility: the historical `balance` field means spendable funds.
		"balance":          available,
		"totalBalance":     total,
		"reservedBalance":  reserved,
		"availableBalance": available,
	}
}

func sumWalletBalances(rows []map[string]any) string {
	total := new(big.Int)
	for _, row := range rows {
		value := mapString(row, "balance", "0")
		parsed, err := parseWalletCentsBig(value)
		if err == nil {
			total.Add(total, parsed)
		}
	}
	return formatWalletCentsBig(total)
}

func firstRow(rows []map[string]any) map[string]any {
	if len(rows) == 0 {
		return nil
	}
	return rows[0]
}

func firstValue(rows []map[string]any, key string) any {
	if len(rows) == 0 {
		return nil
	}
	return rows[0][key]
}

func firstString(rows []map[string]any, key, fallback string) string {
	return mapString(firstRow(rows), key, fallback)
}

func mapString(row map[string]any, key, fallback string) string {
	if row == nil || row[key] == nil {
		return fallback
	}
	if value, ok := row[key].(string); ok {
		return value
	}
	return fmt.Sprint(row[key])
}
