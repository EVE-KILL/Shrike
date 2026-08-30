-- Extensions present in production (verified 2026-07-25: pg_trgm 1.6, plpgsql).
--
-- pg_trgm is not optional: entity search is a per-type UNION using similarity()
-- and the % operator (api/src/routes/search.ts). Without it, every search query
-- fails rather than merely running slowly.
CREATE EXTENSION IF NOT EXISTS pg_trgm;
CREATE EXTENSION IF NOT EXISTS pg_stat_statements;
