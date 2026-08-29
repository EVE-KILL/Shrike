-- Keep dead-row cleanup proportional on large, frequently updated tables.

-- +goose Up
ALTER TABLE stats_breakdowns SET (
    autovacuum_vacuum_scale_factor = 0.01,
    autovacuum_analyze_scale_factor = 0.01
);
ALTER TABLE entity_snapshots SET (
    autovacuum_vacuum_scale_factor = 0.02,
    autovacuum_analyze_scale_factor = 0.02
);
ALTER TABLE fitting_items SET (
    autovacuum_vacuum_scale_factor = 0.02,
    autovacuum_analyze_scale_factor = 0.02
);
ALTER TABLE killmail_processing SET (
    autovacuum_vacuum_scale_factor = 0.02,
    autovacuum_analyze_scale_factor = 0.02
);
ALTER TABLE killmail_fittings SET (
    autovacuum_vacuum_scale_factor = 0.02,
    autovacuum_analyze_scale_factor = 0.02
);

-- +goose Down
ALTER TABLE killmail_fittings RESET (
    autovacuum_vacuum_scale_factor,
    autovacuum_analyze_scale_factor
);
ALTER TABLE killmail_processing RESET (
    autovacuum_vacuum_scale_factor,
    autovacuum_analyze_scale_factor
);
ALTER TABLE fitting_items RESET (
    autovacuum_vacuum_scale_factor,
    autovacuum_analyze_scale_factor
);
ALTER TABLE entity_snapshots RESET (
    autovacuum_vacuum_scale_factor,
    autovacuum_analyze_scale_factor
);
ALTER TABLE stats_breakdowns RESET (
    autovacuum_vacuum_scale_factor,
    autovacuum_analyze_scale_factor
);
