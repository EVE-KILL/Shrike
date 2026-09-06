-- +goose Up
-- Match the kill's distinct targets against the partial tracker index rather
-- than running correlated attacker searches for every enabled tracker.
-- +goose StatementBegin
CREATE OR REPLACE FUNCTION record_entity_tracker_events()
RETURNS trigger LANGUAGE plpgsql AS $$
BEGIN
    WITH targets AS MATERIALIZED (
        SELECT kind, id, bool_or(is_victim) AS victim, bool_or(is_attacker) AS attacker
        FROM (
            SELECT kind, id, true AS is_victim, false AS is_attacker
            FROM (VALUES
                ('character', NEW.victim_character_id),
                ('corporation', NEW.victim_corporation_id),
                ('alliance', NEW.victim_alliance_id),
                ('system', NEW.solar_system_id),
                ('constellation', NEW.constellation_id),
                ('region', NEW.region_id)
            ) AS victim_targets(kind, id)
            UNION ALL
            SELECT target.kind, target.id, false, true
            FROM killmail_attackers a
            CROSS JOIN LATERAL (VALUES
                ('character', a.character_id),
                ('corporation', a.corporation_id),
                ('alliance', a.alliance_id)
            ) AS target(kind, id)
            WHERE a.killmail_id = NEW.killmail_id
        ) all_targets
        WHERE id > 0
        GROUP BY kind, id
    ), matching AS MATERIALIZED (
        SELECT t.tracker_id, t.character_id, t.notifications_enabled,
            CASE
                WHEN targets.kind IN ('system', 'constellation', 'region') THEN 'location'
                WHEN targets.victim AND targets.attacker THEN 'both'
                WHEN targets.victim THEN 'victim'
                ELSE 'attacker'
            END AS match_role
        FROM targets
        JOIN entity_trackers t ON t.target_type = targets.kind AND t.target_id = targets.id
        WHERE t.enabled
    ), inserted AS (
        INSERT INTO entity_tracker_events (tracker_id, killmail_id, match_role, occurred_at)
        SELECT tracker_id, NEW.killmail_id, match_role, NEW.killmail_time FROM matching
        ON CONFLICT (tracker_id, killmail_id) DO NOTHING
        RETURNING event_id, tracker_id
    )
    INSERT INTO entity_tracker_notifications (event_id, character_id)
    SELECT inserted.event_id, matching.character_id
    FROM inserted JOIN matching USING (tracker_id)
    WHERE matching.notifications_enabled
    ON CONFLICT (event_id) DO NOTHING;
    RETURN NEW;
END;
$$;
-- +goose StatementEnd

-- +goose Down
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

