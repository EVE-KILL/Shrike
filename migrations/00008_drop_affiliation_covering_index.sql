-- The updated-at order makes this covering index scan most characters before
-- it finds dormant candidates. The existing bitmap indexes are substantially
-- faster for that branch.

-- +goose NO TRANSACTION

-- +goose Up
DROP INDEX CONCURRENTLY IF EXISTS characters_affiliation_due_idx;

-- +goose Down
SELECT 1;
