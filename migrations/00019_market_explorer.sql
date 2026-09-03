-- +goose Up
-- The explorer deliberately keeps its mutable market data separate from
-- `prices`. That table is the immutable Jita valuation history used by stored
-- killmails; correcting an EVE Ref archive here must never revalue old kills.
CREATE TABLE IF NOT EXISTS market_source_files (
    source_path text PRIMARY KEY,
    dataset text NOT NULL,
    etag text,
    size_bytes bigint NOT NULL,
    source_last_modified timestamptz,
    file_time timestamptz,
    rows_imported bigint NOT NULL,
    imported_at timestamptz NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS market_source_files_dataset_time_idx
    ON market_source_files (dataset, file_time DESC);

-- One complete current snapshot. A new EVE Ref snapshot replaces this table
-- atomically, so readers see either the old complete book or the new one.
CREATE TABLE IF NOT EXISTS market_orders (
    order_id bigint PRIMARY KEY,
    duration smallint NOT NULL,
    is_buy_order boolean NOT NULL,
    issued timestamptz NOT NULL,
    location_id bigint NOT NULL,
    min_volume bigint NOT NULL,
    price double precision NOT NULL,
    order_range text NOT NULL,
    system_id integer NOT NULL,
    type_id integer NOT NULL,
    volume_remain bigint NOT NULL,
    volume_total bigint NOT NULL,
    http_last_modified timestamptz,
    station_id bigint,
    region_id integer NOT NULL,
    constellation_id integer,
    snapshot_at timestamptz NOT NULL
);

CREATE INDEX IF NOT EXISTS market_orders_type_sell_idx
    ON market_orders (type_id, region_id, price, order_id)
    WHERE is_buy_order IS FALSE;
CREATE INDEX IF NOT EXISTS market_orders_type_buy_idx
    ON market_orders (type_id, region_id, price DESC, order_id)
    WHERE is_buy_order IS TRUE;
CREATE INDEX IF NOT EXISTS market_orders_system_type_idx
    ON market_orders (system_id, type_id);

-- Correctable, all-region daily history for charts and market summaries.
-- Each changed EVE Ref day is deleted and reinserted in one transaction.
CREATE TABLE IF NOT EXISTS market_region_history (
    type_id integer NOT NULL,
    region_id integer NOT NULL,
    date date NOT NULL,
    average double precision,
    highest double precision,
    lowest double precision,
    order_count integer,
    volume bigint,
    http_last_modified timestamptz,
    PRIMARY KEY (type_id, region_id, date)
);

CREATE INDEX IF NOT EXISTS market_region_history_region_date_idx
    ON market_region_history (region_id, date DESC);

-- +goose Down
DROP TABLE IF EXISTS market_region_history;
DROP TABLE IF EXISTS market_orders;
DROP TABLE IF EXISTS market_source_files;
