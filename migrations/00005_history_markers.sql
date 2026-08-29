-- Reconcile legacy history rows with their authoritative completion markers.

-- +goose Up
UPDATE characters character
SET corporation_history_fetched_at = now(),
    corporation_history_queued_at = NULL
WHERE character.corporation_history_fetched_at IS NULL
  AND EXISTS (
      SELECT 1
      FROM character_corporation_history history
      WHERE history.character_id = character.character_id
  );

UPDATE corporations corporation
SET alliance_history_fetched_at = now(),
    alliance_history_queued_at = NULL
WHERE corporation.alliance_history_fetched_at IS NULL
  AND EXISTS (
      SELECT 1
      FROM corporation_alliance_history history
      WHERE history.corporation_id = corporation.corporation_id
  );

-- +goose Down
-- Completion markers describe facts already present in the history tables and
-- must not be erased during a rollback.
SELECT 1;
