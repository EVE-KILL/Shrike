-- Accelerate container-child detection on killmail detail responses.

-- +goose NO TRANSACTION

-- +goose Up
CREATE INDEX CONCURRENTLY IF NOT EXISTS killmail_items_parent_idx
    ON killmail_items (killmail_id, parent_index)
    WHERE parent_index IS NOT NULL
      AND parent_index <> 0;

-- +goose Down
DROP INDEX CONCURRENTLY IF EXISTS killmail_items_parent_idx;
