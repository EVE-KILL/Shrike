-- +goose Up
-- Trackers and notifications are deliberately separate. Every matching
-- killmail creates a tracker event; only trackers with notifications enabled
-- also create a notification row at match time.
CREATE TABLE IF NOT EXISTS entity_trackers (
    tracker_id bigint GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    character_id integer NOT NULL REFERENCES users(character_id) ON DELETE CASCADE,
    target_type text NOT NULL,
    target_id integer NOT NULL,
    target_name text NOT NULL,
    target_ticker text,
    enabled boolean NOT NULL DEFAULT true,
    notifications_enabled boolean NOT NULL DEFAULT false,
    created_at timestamptz NOT NULL DEFAULT now(),
    updated_at timestamptz NOT NULL DEFAULT now(),
    CONSTRAINT entity_trackers_target_type_check CHECK (
        target_type IN ('character', 'corporation', 'alliance', 'system', 'constellation', 'region')
    ),
    CONSTRAINT entity_trackers_target_id_check CHECK (target_id > 0),
    CONSTRAINT entity_trackers_target_name_check CHECK (length(target_name) BETWEEN 1 AND 200),
    CONSTRAINT entity_trackers_account_target_unique UNIQUE (character_id, target_type, target_id)
);

CREATE INDEX IF NOT EXISTS entity_trackers_match_idx
    ON entity_trackers (target_type, target_id)
    WHERE enabled;

CREATE INDEX IF NOT EXISTS entity_trackers_account_idx
    ON entity_trackers (character_id, created_at DESC);

CREATE TABLE IF NOT EXISTS entity_tracker_events (
    event_id bigint GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    tracker_id bigint NOT NULL REFERENCES entity_trackers(tracker_id) ON DELETE CASCADE,
    killmail_id integer NOT NULL REFERENCES killmails(killmail_id) ON DELETE CASCADE,
    match_role text NOT NULL,
    occurred_at timestamptz NOT NULL,
    created_at timestamptz NOT NULL DEFAULT now(),
    CONSTRAINT entity_tracker_events_role_check CHECK (
        match_role IN ('victim', 'attacker', 'both', 'location')
    ),
    CONSTRAINT entity_tracker_events_once UNIQUE (tracker_id, killmail_id)
);

CREATE INDEX IF NOT EXISTS entity_tracker_events_tracker_time_idx
    ON entity_tracker_events (tracker_id, occurred_at DESC, event_id DESC);

CREATE INDEX IF NOT EXISTS entity_tracker_events_killmail_idx
    ON entity_tracker_events (killmail_id);

CREATE TABLE IF NOT EXISTS entity_tracker_notifications (
    notification_id bigint GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    event_id bigint NOT NULL UNIQUE REFERENCES entity_tracker_events(event_id) ON DELETE CASCADE,
    character_id integer NOT NULL REFERENCES users(character_id) ON DELETE CASCADE,
    created_at timestamptz NOT NULL DEFAULT now(),
    read_at timestamptz
);

CREATE INDEX IF NOT EXISTS entity_tracker_notifications_account_idx
    ON entity_tracker_notifications (character_id, notification_id DESC);

CREATE INDEX IF NOT EXISTS entity_tracker_notifications_unread_idx
    ON entity_tracker_notifications (character_id, notification_id DESC)
    WHERE read_at IS NULL;

-- This is deferred until transaction commit because killmail_attackers are
-- written after the parent killmail row. At that point the function sees the
-- complete mail and records each tracker at most once.
-- +goose StatementBegin
CREATE OR REPLACE FUNCTION record_entity_tracker_events()
RETURNS trigger
LANGUAGE plpgsql
AS $$
BEGIN
    WITH matching AS (
        SELECT
            tracker.tracker_id,
            tracker.character_id,
            tracker.notifications_enabled,
            CASE
                WHEN tracker.target_type IN ('system', 'constellation', 'region')
                    THEN 'location'
                WHEN (
                    (tracker.target_type = 'character' AND NEW.victim_character_id = tracker.target_id) OR
                    (tracker.target_type = 'corporation' AND NEW.victim_corporation_id = tracker.target_id) OR
                    (tracker.target_type = 'alliance' AND NEW.victim_alliance_id = tracker.target_id)
                ) AND EXISTS (
                    SELECT 1
                    FROM killmail_attackers attacker
                    WHERE attacker.killmail_id = NEW.killmail_id
                      AND (
                        (tracker.target_type = 'character' AND attacker.character_id = tracker.target_id) OR
                        (tracker.target_type = 'corporation' AND attacker.corporation_id = tracker.target_id) OR
                        (tracker.target_type = 'alliance' AND attacker.alliance_id = tracker.target_id)
                      )
                ) THEN 'both'
                WHEN (
                    (tracker.target_type = 'character' AND NEW.victim_character_id = tracker.target_id) OR
                    (tracker.target_type = 'corporation' AND NEW.victim_corporation_id = tracker.target_id) OR
                    (tracker.target_type = 'alliance' AND NEW.victim_alliance_id = tracker.target_id)
                ) THEN 'victim'
                ELSE 'attacker'
            END AS match_role
        FROM entity_trackers tracker
        WHERE tracker.enabled
          AND (
            (tracker.target_type = 'character' AND (
                NEW.victim_character_id = tracker.target_id OR EXISTS (
                    SELECT 1 FROM killmail_attackers attacker
                    WHERE attacker.killmail_id = NEW.killmail_id
                      AND attacker.character_id = tracker.target_id
                )
            )) OR
            (tracker.target_type = 'corporation' AND (
                NEW.victim_corporation_id = tracker.target_id OR EXISTS (
                    SELECT 1 FROM killmail_attackers attacker
                    WHERE attacker.killmail_id = NEW.killmail_id
                      AND attacker.corporation_id = tracker.target_id
                )
            )) OR
            (tracker.target_type = 'alliance' AND (
                NEW.victim_alliance_id = tracker.target_id OR EXISTS (
                    SELECT 1 FROM killmail_attackers attacker
                    WHERE attacker.killmail_id = NEW.killmail_id
                      AND attacker.alliance_id = tracker.target_id
                )
            )) OR
            (tracker.target_type = 'system' AND NEW.solar_system_id = tracker.target_id) OR
            (tracker.target_type = 'constellation' AND NEW.constellation_id = tracker.target_id) OR
            (tracker.target_type = 'region' AND NEW.region_id = tracker.target_id)
          )
    ), inserted AS (
        INSERT INTO entity_tracker_events (
            tracker_id, killmail_id, match_role, occurred_at
        )
        SELECT tracker_id, NEW.killmail_id, match_role, NEW.killmail_time
        FROM matching
        ON CONFLICT (tracker_id, killmail_id) DO NOTHING
        RETURNING event_id, tracker_id
    )
    INSERT INTO entity_tracker_notifications (event_id, character_id)
    SELECT inserted.event_id, matching.character_id
    FROM inserted
    JOIN matching USING (tracker_id)
    WHERE matching.notifications_enabled
    ON CONFLICT (event_id) DO NOTHING;

    RETURN NEW;
END;
$$;
-- +goose StatementEnd

DROP TRIGGER IF EXISTS entity_tracker_match_new_killmail ON killmails;
CREATE CONSTRAINT TRIGGER entity_tracker_match_new_killmail
AFTER INSERT ON killmails
DEFERRABLE INITIALLY DEFERRED
FOR EACH ROW
EXECUTE FUNCTION record_entity_tracker_events();

-- +goose Down
DROP TRIGGER IF EXISTS entity_tracker_match_new_killmail ON killmails;
DROP FUNCTION IF EXISTS record_entity_tracker_events();
DROP TABLE IF EXISTS entity_tracker_notifications;
DROP TABLE IF EXISTS entity_tracker_events;
DROP TABLE IF EXISTS entity_trackers;
