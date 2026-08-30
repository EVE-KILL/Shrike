-- pg_stat_statements is preloaded in production and CNPG. The extension object
-- exposes aggregate planning and execution counters without requiring query
-- text to leave PostgreSQL.

-- +goose Up
CREATE EXTENSION IF NOT EXISTS pg_stat_statements;

-- +goose Down
DROP EXTENSION IF EXISTS pg_stat_statements;
