-- +goose Up
-- Region summaries read both sides of a type's order book at once, so neither
-- partial side index can serve them. Keep the grouping columns first and carry
-- the two aggregate inputs in the index payload.
CREATE INDEX IF NOT EXISTS market_orders_type_region_idx
    ON market_orders (type_id, region_id)
    INCLUDE (is_buy_order, price);

-- +goose Down
DROP INDEX IF EXISTS market_orders_type_region_idx;
