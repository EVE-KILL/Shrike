package api

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
)

type postgresAuthStore struct {
	db MutationDatabase
}

func (s *postgresAuthStore) ResolveSession(
	ctx context.Context,
	sessionID string,
	now time.Time,
) (Principal, authSessionActivity, error) {
	var principal Principal
	var activity authSessionActivity
	err := s.db.QueryRow(ctx, `
		SELECT
			us.id,
			us.ip_address,
			us.country_code,
			us.user_agent,
			us.client_hint,
			us.last_seen_at,
			u.character_id,
			u.character_name,
			u.is_admin,
			coalesce(u.last_seen_notification_id, 0),
			c.corporation_id,
			corp.name,
			c.alliance_id,
			ally.name
		FROM user_sessions us
		INNER JOIN users u
			ON u.character_id = us.character_id
		LEFT JOIN characters c
			ON c.character_id = u.character_id
		LEFT JOIN corporations corp
			ON corp.corporation_id = c.corporation_id
		LEFT JOIN alliances ally
			ON ally.alliance_id = c.alliance_id
		WHERE us.session_id = $1
		  AND us.expires_at > $2
		LIMIT 1`,
		sessionID,
		now,
	).Scan(
		&activity.RowID,
		&activity.IPAddress,
		&activity.CountryCode,
		&activity.UserAgent,
		&activity.ClientHint,
		&activity.LastSeenAt,
		&principal.CharacterID,
		&principal.CharacterName,
		&principal.IsAdmin,
		&principal.LastSeenNotificationID,
		&principal.CorporationID,
		&principal.CorporationName,
		&principal.AllianceID,
		&principal.AllianceName,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return Principal{}, authSessionActivity{}, errNoAuthSession
	}
	if err != nil {
		return Principal{}, authSessionActivity{}, err
	}
	return principal, activity, nil
}

func (s *postgresAuthStore) TouchSession(
	ctx context.Context,
	rowID int64,
	metadata authSessionMetadata,
	now time.Time,
) error {
	tag, err := s.db.Exec(ctx, `
		UPDATE user_sessions
		SET last_seen_at = $2,
		    ip_address = coalesce($3, ip_address),
		    country_code = coalesce($4, country_code),
		    user_agent = coalesce($5, user_agent),
		    client_hint = coalesce($6, client_hint)
		WHERE id = $1`,
		rowID,
		now,
		nullableString(metadata.IPAddress),
		nullableString(metadata.CountryCode),
		nullableString(metadata.UserAgent),
		nullableString(metadata.ClientHint),
	)
	if err != nil {
		return err
	}
	if tag.RowsAffected() != 1 {
		return errNoAuthSession
	}
	return nil
}

func (s *postgresAuthStore) LoadMeDetails(
	ctx context.Context,
	characterID int32,
) (authMeDetails, error) {
	var details authMeDetails
	var settingsJSON []byte
	err := s.db.QueryRow(ctx, `
		SELECT
			u.character_owner_hash,
			u.last_login,
			u.created_at,
			coalesce(
				(
					SELECT jsonb_object_agg(uc.key, uc.value)
					FROM user_config uc
					WHERE uc.character_id = u.character_id
				),
				'{}'::jsonb
			)
		FROM users u
		WHERE u.character_id = $1`,
		characterID,
	).Scan(
		&details.CharacterOwnerHash,
		&details.LastLogin,
		&details.CreatedAt,
		&settingsJSON,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return authMeDetails{}, errNoAuthSession
	}
	if err != nil {
		return authMeDetails{}, err
	}
	if err := json.Unmarshal(settingsJSON, &details.Settings); err != nil {
		return authMeDetails{}, fmt.Errorf("decode user settings: %w", err)
	}
	if details.Settings == nil {
		details.Settings = map[string]any{}
	}
	return details, nil
}

func (s *postgresAuthStore) LoadTokenInfo(
	ctx context.Context,
	characterID int32,
) (authTokenInfo, bool, error) {
	var token authTokenInfo
	err := s.db.QueryRow(ctx, `
		SELECT
			coalesce(scopes, '{}'),
			coalesce(revoked_scopes, '{}'),
			token_expiry,
			last_fetched,
			coalesce(disabled, false)
		FROM user_esi_tokens
		WHERE character_id = $1
		LIMIT 1`,
		characterID,
	).Scan(
		&token.Scopes,
		&token.RevokedScopes,
		&token.TokenExpiry,
		&token.LastFetched,
		&token.Disabled,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return authTokenInfo{}, false, nil
	}
	if err != nil {
		return authTokenInfo{}, false, err
	}
	return token, true, nil
}

func (s *postgresAuthStore) CompleteLogin(
	ctx context.Context,
	login authLoginCommit,
) (err error) {
	tx, err := s.db.Begin(ctx)
	if err != nil {
		return err
	}
	defer func() {
		_ = tx.Rollback(ctx)
	}()

	_, err = tx.Exec(ctx, `
		INSERT INTO users (
			character_id,
			character_name,
			character_owner_hash,
			session_id,
			is_admin,
			last_login,
			created_at,
			updated_at
		) VALUES ($1, $2, $3, $4, false, $5, $5, $5)
		ON CONFLICT (character_id) DO UPDATE SET
			character_name = excluded.character_name,
			character_owner_hash = excluded.character_owner_hash,
			session_id = excluded.session_id,
			last_login = excluded.last_login,
			updated_at = excluded.updated_at`,
		login.Claims.CharacterID,
		login.Claims.CharacterName,
		login.Claims.CharacterOwnerHash,
		login.SessionID,
		login.Now,
	)
	if err != nil {
		return err
	}

	_, err = tx.Exec(ctx, `
		INSERT INTO user_sessions (
			session_id,
			character_id,
			ip_address,
			country_code,
			user_agent,
			client_hint,
			created_at,
			last_seen_at,
			expires_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $7, $8)`,
		login.SessionID,
		login.Claims.CharacterID,
		nullableString(login.Metadata.IPAddress),
		nullableString(login.Metadata.CountryCode),
		nullableString(login.Metadata.UserAgent),
		nullableString(login.Metadata.ClientHint),
		login.Now,
		login.ExpiresAt,
	)
	if err != nil {
		return err
	}

	tokenExpiry := login.Now.Add(
		time.Duration(login.Tokens.ExpiresIn) * time.Second,
	)
	scopes := login.Claims.Scopes
	if scopes == nil {
		// pgx encodes a nil slice as SQL NULL, but the inherited schema keeps
		// scopes NOT NULL. An authorization carrying no ESI scopes is the
		// empty PostgreSQL array, matching Drizzle's previous write.
		scopes = []string{}
	}
	_, err = tx.Exec(ctx, `
		INSERT INTO user_esi_tokens (
			character_id,
			access_token,
			refresh_token,
			token_expiry,
			token_type,
			scopes,
			delay,
			created_at,
			updated_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $8)
		ON CONFLICT (character_id) DO UPDATE SET
			access_token = excluded.access_token,
			refresh_token = excluded.refresh_token,
			token_expiry = excluded.token_expiry,
			token_type = excluded.token_type,
			scopes = excluded.scopes,
			delay = excluded.delay,
			disabled = false,
			revoked_scopes = '{}',
			scopes_revoked_at = NULL,
			character_failure_count = 0,
			updated_at = excluded.updated_at`,
		login.Claims.CharacterID,
		login.Tokens.AccessToken,
		login.Tokens.RefreshToken,
		tokenExpiry,
		login.Tokens.TokenType,
		scopes,
		login.Delay,
		login.Now,
	)
	if err != nil {
		return err
	}
	return tx.Commit(ctx)
}

func (s *postgresAuthStore) DeleteSession(
	ctx context.Context,
	sessionID string,
) error {
	_, err := s.db.Exec(ctx,
		`DELETE FROM user_sessions WHERE session_id = $1`,
		sessionID,
	)
	return err
}

func (s *postgresAuthStore) ListSessions(
	ctx context.Context,
	characterID int32,
	currentSessionID string,
	now time.Time,
) ([]authSessionSummary, error) {
	rows, err := s.db.Query(ctx, `
		SELECT
			id,
			session_id = $2::uuid AS current,
			ip_address,
			country_code,
			user_agent,
			client_hint,
			created_at,
			last_seen_at,
			expires_at
		FROM user_sessions
		WHERE character_id = $1
		  AND expires_at > $3
		ORDER BY current DESC, last_seen_at DESC`,
		characterID,
		currentSessionID,
		now,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	sessions := make([]authSessionSummary, 0)
	for rows.Next() {
		var session authSessionSummary
		if err := rows.Scan(
			&session.ID,
			&session.Current,
			&session.IPAddress,
			&session.CountryCode,
			&session.UserAgent,
			&session.ClientHint,
			&session.CreatedAt,
			&session.LastSeenAt,
			&session.ExpiresAt,
		); err != nil {
			return nil, err
		}
		sessions = append(sessions, session)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return sessions, nil
}

func (s *postgresAuthStore) RevokeSession(
	ctx context.Context,
	characterID int32,
	id int64,
	currentSessionID string,
) (bool, bool, error) {
	var current bool
	err := s.db.QueryRow(ctx, `
		DELETE FROM user_sessions
		WHERE id = $1
		  AND character_id = $2
		RETURNING session_id = $3::uuid`,
		id,
		characterID,
		currentSessionID,
	).Scan(&current)
	if errors.Is(err, pgx.ErrNoRows) {
		return false, false, nil
	}
	if err != nil {
		return false, false, err
	}
	return true, current, nil
}

func (s *postgresAuthStore) RevokeOtherSessions(
	ctx context.Context,
	characterID int32,
	currentSessionID string,
) (int64, error) {
	tag, err := s.db.Exec(ctx, `
		DELETE FROM user_sessions
		WHERE character_id = $1
		  AND session_id <> $2::uuid`,
		characterID,
		currentSessionID,
	)
	if err != nil {
		return 0, err
	}
	return tag.RowsAffected(), nil
}

func nullableString(value *string) any {
	if value == nil {
		return nil
	}
	return *value
}
