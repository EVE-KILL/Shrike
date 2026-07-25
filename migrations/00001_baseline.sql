-- Baseline of the production schema, captured 2026-07-25
-- from pg_dump --schema-only --schema=public (server 18.3).
--
-- READ THIS BEFORE RUNNING GOOSE AGAINST PRODUCTION.
--
-- Production already has all 102 of these tables; they were created by drizzle
-- and are tracked in drizzle.__drizzle_migrations plus
-- public.__evekill_schema_migrations. This file must therefore NEVER be
-- executed there. Instead run:
--
--     shrike db:baseline --apply
--
-- which stamps the goose ledger as though version 1 had already run, without
-- touching a single table. `db:migrate` independently refuses to execute
-- anything when it finds an existing schema with no goose ledger, so the
-- accident is blocked from both directions.
--
-- On a fresh database this file runs normally and creates everything.
--
-- Excluded on purpose: the drizzle schema (drizzle's own bookkeeping) and
-- migration_archive_20260721 (the five pre-2026-07-21 stats tables, kept for
-- reference only). Neither is needed to run the application.

-- +goose Up

-- pg_dump omits extensions when scoped to a schema, but entity search is a
-- similarity()/% query and hard-fails without this. Declared here so the
-- migration is self-sufficient on any fresh database, not just one created by
-- the docker-compose init script. IF NOT EXISTS keeps it a no-op on production.
CREATE EXTENSION IF NOT EXISTS pg_trgm;
--
-- PostgreSQL database dump
--


-- Dumped from database version 18.3 (Debian 18.3-1.pgdg13+1)
-- Dumped by pg_dump version 18.4

SET statement_timeout = 0;
SET lock_timeout = 0;
SET idle_in_transaction_session_timeout = 0;
SET transaction_timeout = 0;
SET client_encoding = 'UTF8';
SET standard_conforming_strings = on;
SELECT pg_catalog.set_config('search_path', 'public', false);
SET check_function_bodies = false;
SET xmloption = content;
SET client_min_messages = warning;
SET row_security = off;

--
-- Name: public; Type: SCHEMA; Schema: -; Owner: -
--



--
-- Name: enforce_ek_wallet_reservation_lifecycle(); Type: FUNCTION; Schema: public; Owner: -
--

-- +goose StatementBegin
CREATE FUNCTION public.enforce_ek_wallet_reservation_lifecycle() RETURNS trigger
    LANGUAGE plpgsql
    AS $$
BEGIN
    IF TG_OP = 'DELETE' THEN
        RAISE EXCEPTION 'EK Wallet reservations are auditable and cannot be deleted';
    END IF;

    IF OLD."status" <> 0 THEN
        RAISE EXCEPTION 'Closed EK Wallet reservations are immutable';
    END IF;

    IF ROW(
        NEW."character_id",
        NEW."external_reference",
        NEW."transaction_type",
        NEW."campaign_id",
        NEW."amount",
        NEW."description",
        NEW."metadata",
        NEW."expires_at",
        NEW."created_at"
    ) IS DISTINCT FROM ROW(
        OLD."character_id",
        OLD."external_reference",
        OLD."transaction_type",
        OLD."campaign_id",
        OLD."amount",
        OLD."description",
        OLD."metadata",
        OLD."expires_at",
        OLD."created_at"
    ) THEN
        RAISE EXCEPTION 'EK Wallet reservation identity is immutable';
    END IF;

    IF NEW."status" NOT IN (1, 2, 3) THEN
        RAISE EXCEPTION 'EK Wallet reservations may only transition once to a terminal state';
    END IF;

    RETURN NEW;
END;
$$;
-- +goose StatementEnd



--
-- Name: prevent_ek_wallet_transaction_mutation(); Type: FUNCTION; Schema: public; Owner: -
--

-- +goose StatementBegin
CREATE FUNCTION public.prevent_ek_wallet_transaction_mutation() RETURNS trigger
    LANGUAGE plpgsql
    AS $$
BEGIN
    RAISE EXCEPTION 'EK Wallet transactions are append-only; create a compensating entry instead';
END;
$$;
-- +goose StatementEnd



SET default_table_access_method = heap;

--
-- Name: __drizzle_migrations; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.__drizzle_migrations (
    id integer NOT NULL,
    hash text NOT NULL,
    created_at timestamp with time zone DEFAULT now()
);


--
-- Name: __drizzle_migrations_id_seq; Type: SEQUENCE; Schema: public; Owner: -
--

CREATE SEQUENCE public.__drizzle_migrations_id_seq
    AS integer
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;


--
-- Name: __drizzle_migrations_id_seq; Type: SEQUENCE OWNED BY; Schema: public; Owner: -
--

ALTER SEQUENCE public.__drizzle_migrations_id_seq OWNED BY public.__drizzle_migrations.id;


--
-- Name: __evekill_schema_migrations; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.__evekill_schema_migrations (
    id bigint NOT NULL,
    name text NOT NULL,
    checksum text NOT NULL,
    kind text NOT NULL,
    applied_at timestamp with time zone DEFAULT now() NOT NULL,
    execution_ms integer,
    metadata jsonb DEFAULT '{}'::jsonb NOT NULL,
    CONSTRAINT __evekill_schema_migrations_checksum_check CHECK ((checksum ~ '^[0-9a-f]{64}$'::text)),
    CONSTRAINT __evekill_schema_migrations_kind_check CHECK ((kind = ANY (ARRAY['baseline'::text, 'migration'::text])))
);


--
-- Name: __evekill_schema_migrations_id_seq; Type: SEQUENCE; Schema: public; Owner: -
--

ALTER TABLE public.__evekill_schema_migrations ALTER COLUMN id ADD GENERATED ALWAYS AS IDENTITY (
    SEQUENCE NAME public.__evekill_schema_migrations_id_seq
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1
);


--
-- Name: alliances; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.alliances (
    alliance_id integer NOT NULL,
    name text NOT NULL,
    ticker text NOT NULL,
    creator_id integer,
    creator_corporation_id integer,
    executor_corporation_id integer,
    date_founded timestamp with time zone,
    faction_id integer,
    corporation_count integer,
    member_count integer,
    deleted boolean DEFAULT false,
    created_at timestamp with time zone DEFAULT now(),
    updated_at timestamp with time zone DEFAULT now(),
    custom_description text,
    custom_description_format text DEFAULT 'markdown'::text
);


--
-- Name: announcement_dismissals; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.announcement_dismissals (
    character_id bigint NOT NULL,
    announcement_id bigint NOT NULL,
    dismissed_at timestamp with time zone DEFAULT now() NOT NULL
);


--
-- Name: announcements; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.announcements (
    id integer NOT NULL,
    tier smallint NOT NULL,
    title text NOT NULL,
    body_md text DEFAULT ''::text NOT NULL,
    body_html text DEFAULT ''::text NOT NULL,
    color text DEFAULT 'info'::text NOT NULL,
    icon text,
    link_url text,
    link_label text,
    starts_at timestamp with time zone DEFAULT now() NOT NULL,
    expires_at timestamp with time zone NOT NULL,
    created_by bigint NOT NULL,
    created_at timestamp with time zone DEFAULT now() NOT NULL,
    updated_at timestamp with time zone DEFAULT now() NOT NULL,
    archived_at timestamp with time zone
);


--
-- Name: announcements_id_seq; Type: SEQUENCE; Schema: public; Owner: -
--

CREATE SEQUENCE public.announcements_id_seq
    AS integer
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;


--
-- Name: announcements_id_seq; Type: SEQUENCE OWNED BY; Schema: public; Owner: -
--

ALTER SEQUENCE public.announcements_id_seq OWNED BY public.announcements.id;


--
-- Name: battle_team_members; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.battle_team_members (
    id integer NOT NULL,
    battle_team_id integer NOT NULL,
    corporation_id integer NOT NULL,
    alliance_id integer,
    kills integer,
    losses integer,
    isk_destroyed double precision,
    isk_lost double precision
);


--
-- Name: battle_team_members_id_seq; Type: SEQUENCE; Schema: public; Owner: -
--

CREATE SEQUENCE public.battle_team_members_id_seq
    AS integer
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;


--
-- Name: battle_team_members_id_seq; Type: SEQUENCE OWNED BY; Schema: public; Owner: -
--

ALTER SEQUENCE public.battle_team_members_id_seq OWNED BY public.battle_team_members.id;


--
-- Name: battle_teams; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.battle_teams (
    id integer NOT NULL,
    battle_id integer NOT NULL,
    team_index smallint NOT NULL,
    total_kills integer,
    total_losses integer,
    total_isk_destroyed double precision,
    total_isk_lost double precision
);


--
-- Name: battle_teams_id_seq; Type: SEQUENCE; Schema: public; Owner: -
--

CREATE SEQUENCE public.battle_teams_id_seq
    AS integer
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;


--
-- Name: battle_teams_id_seq; Type: SEQUENCE OWNED BY; Schema: public; Owner: -
--

ALTER SEQUENCE public.battle_teams_id_seq OWNED BY public.battle_teams.id;


--
-- Name: battles; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.battles (
    battle_id integer NOT NULL,
    solar_system_id integer NOT NULL,
    region_id integer,
    start_time timestamp with time zone NOT NULL,
    end_time timestamp with time zone NOT NULL,
    duration_minutes integer,
    kill_count integer,
    total_isk_destroyed double precision,
    is_multi_party boolean DEFAULT false,
    is_custom boolean DEFAULT false,
    created_at timestamp with time zone DEFAULT now(),
    updated_at timestamp with time zone DEFAULT now(),
    created_by_character_id integer
);


--
-- Name: battles_battle_id_seq; Type: SEQUENCE; Schema: public; Owner: -
--

CREATE SEQUENCE public.battles_battle_id_seq
    AS integer
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;


--
-- Name: battles_battle_id_seq; Type: SEQUENCE OWNED BY; Schema: public; Owner: -
--

ALTER SEQUENCE public.battles_battle_id_seq OWNED BY public.battles.battle_id;


--
-- Name: blog_posts; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.blog_posts (
    id integer NOT NULL,
    slug text NOT NULL,
    title text NOT NULL,
    excerpt text,
    body_md text DEFAULT ''::text NOT NULL,
    body_html text DEFAULT ''::text NOT NULL,
    cover_image_url text,
    status smallint DEFAULT 0 NOT NULL,
    author_id bigint NOT NULL,
    author_name text NOT NULL,
    published_at timestamp with time zone,
    created_at timestamp with time zone DEFAULT now() NOT NULL,
    updated_at timestamp with time zone DEFAULT now() NOT NULL,
    tags text[] DEFAULT '{}'::text[] NOT NULL,
    author_corporation_id integer,
    author_corporation_name text,
    author_alliance_id integer,
    author_alliance_name text
);


--
-- Name: blog_posts_id_seq; Type: SEQUENCE; Schema: public; Owner: -
--

CREATE SEQUENCE public.blog_posts_id_seq
    AS integer
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;


--
-- Name: blog_posts_id_seq; Type: SEQUENCE OWNED BY; Schema: public; Owner: -
--

ALTER SEQUENCE public.blog_posts_id_seq OWNED BY public.blog_posts.id;


--
-- Name: bloodlines; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.bloodlines (
    bloodline_id integer NOT NULL,
    bloodline_name text,
    race_id integer,
    description text,
    male_description text,
    female_description text,
    short_description text,
    short_male_description text,
    short_female_description text,
    ship_type_id integer,
    corporation_id integer,
    perception integer,
    willpower integer,
    charisma integer,
    memory integer,
    intelligence integer,
    icon_id integer
);


--
-- Name: blueprint_activities; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.blueprint_activities (
    blueprint_type_id integer NOT NULL,
    activity text NOT NULL,
    "time" integer NOT NULL
);


--
-- Name: blueprint_activity_materials; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.blueprint_activity_materials (
    blueprint_type_id integer NOT NULL,
    activity text NOT NULL,
    material_type_id integer NOT NULL,
    quantity integer NOT NULL
);


--
-- Name: blueprint_activity_products; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.blueprint_activity_products (
    blueprint_type_id integer NOT NULL,
    activity text NOT NULL,
    product_type_id integer NOT NULL,
    quantity integer NOT NULL,
    probability real
);


--
-- Name: blueprint_activity_skills; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.blueprint_activity_skills (
    blueprint_type_id integer NOT NULL,
    activity text NOT NULL,
    skill_type_id integer NOT NULL,
    level integer NOT NULL
);


--
-- Name: blueprints; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.blueprints (
    blueprint_type_id integer NOT NULL,
    max_production_limit integer
);


--
-- Name: campaign_prize_pools; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.campaign_prize_pools (
    campaign_id text NOT NULL,
    metric smallint NOT NULL,
    winner_count smallint NOT NULL,
    payout_percentages smallint[] NOT NULL,
    status smallint DEFAULT 0 NOT NULL,
    funded_total numeric(30,2) DEFAULT 0 NOT NULL,
    rules_locked_at timestamp with time zone,
    finalized_at timestamp with time zone,
    created_at timestamp with time zone DEFAULT now() NOT NULL,
    updated_at timestamp with time zone DEFAULT now() NOT NULL,
    CONSTRAINT campaign_prize_pools_funded_total_check CHECK ((funded_total >= (0)::numeric)),
    CONSTRAINT campaign_prize_pools_metric_check CHECK (((metric >= 0) AND (metric <= 3))),
    CONSTRAINT campaign_prize_pools_percentages_check CHECK (((cardinality(payout_percentages) = winner_count) AND (0 < ALL (payout_percentages)))),
    CONSTRAINT campaign_prize_pools_status_check CHECK (((status >= 0) AND (status <= 3))),
    CONSTRAINT campaign_prize_pools_winner_count_check CHECK (((winner_count >= 3) AND (winner_count <= 10)))
);


--
-- Name: campaign_prize_results; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.campaign_prize_results (
    campaign_id text NOT NULL,
    rank smallint NOT NULL,
    character_id integer NOT NULL,
    character_name text,
    metric_value numeric(30,2) NOT NULL,
    secondary_value numeric(30,2) NOT NULL,
    payout_percentage smallint NOT NULL,
    payout_amount numeric(30,2),
    claimed_at timestamp with time zone,
    claimed_by_character_id integer,
    paid_at timestamp with time zone,
    paid_by_admin_character_id integer,
    payment_note text,
    created_at timestamp with time zone DEFAULT now() NOT NULL,
    updated_at timestamp with time zone DEFAULT now() NOT NULL,
    CONSTRAINT campaign_prize_results_payout_percentage_check CHECK (((payout_percentage >= 1) AND (payout_percentage <= 100))),
    CONSTRAINT campaign_prize_results_rank_check CHECK (((rank >= 1) AND (rank <= 10)))
);


--
-- Name: campaign_scratch_candidates; Type: TABLE; Schema: public; Owner: -
--

CREATE UNLOGGED TABLE public.campaign_scratch_candidates (
    campaign_id text NOT NULL,
    killmail_id integer NOT NULL
);


--
-- Name: campaign_scratch_killmails; Type: TABLE; Schema: public; Owner: -
--

CREATE UNLOGGED TABLE public.campaign_scratch_killmails (
    campaign_id text NOT NULL,
    killmail_id integer NOT NULL,
    killmail_time timestamp with time zone NOT NULL,
    day date NOT NULL,
    adj_value double precision NOT NULL,
    victim_ship_type_id integer,
    victim_ship_group_id integer,
    victim_character_id integer,
    victim_corporation_id integer,
    victim_alliance_id integer,
    solar_system_id integer,
    region_id integer,
    victim_side smallint,
    attacker_mask integer DEFAULT 0 NOT NULL
);


--
-- Name: campaign_side_entities; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.campaign_side_entities (
    id integer NOT NULL,
    campaign_id text NOT NULL,
    side_index smallint NOT NULL,
    entity_type smallint NOT NULL,
    entity_id integer NOT NULL,
    kills integer DEFAULT 0 NOT NULL,
    losses integer DEFAULT 0 NOT NULL,
    isk_destroyed double precision DEFAULT 0 NOT NULL,
    isk_lost double precision DEFAULT 0 NOT NULL
);


--
-- Name: campaign_side_entities_id_seq; Type: SEQUENCE; Schema: public; Owner: -
--

ALTER TABLE public.campaign_side_entities ALTER COLUMN id ADD GENERATED ALWAYS AS IDENTITY (
    SEQUENCE NAME public.campaign_side_entities_id_seq
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1
);


--
-- Name: campaign_sides; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.campaign_sides (
    campaign_id text NOT NULL,
    side_index smallint NOT NULL,
    name text NOT NULL,
    kills integer DEFAULT 0 NOT NULL,
    losses integer DEFAULT 0 NOT NULL,
    isk_destroyed double precision DEFAULT 0 NOT NULL,
    isk_lost double precision DEFAULT 0 NOT NULL
);


--
-- Name: campaign_stats_daily; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.campaign_stats_daily (
    campaign_id text NOT NULL,
    side_index smallint NOT NULL,
    day date NOT NULL,
    kills integer DEFAULT 0 NOT NULL,
    losses integer DEFAULT 0 NOT NULL,
    isk_destroyed double precision DEFAULT 0 NOT NULL,
    isk_lost double precision DEFAULT 0 NOT NULL
);


--
-- Name: campaigns; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.campaigns (
    campaign_id text NOT NULL,
    name text NOT NULL,
    description text,
    created_by_character_id integer NOT NULL,
    status smallint DEFAULT 0 NOT NULL,
    start_time timestamp with time zone NOT NULL,
    end_time timestamp with time zone,
    location jsonb,
    stats jsonb,
    processed_through timestamp with time zone,
    last_activity_at timestamp with time zone,
    stats_updated_at timestamp with time zone,
    created_at timestamp with time zone DEFAULT now() NOT NULL,
    updated_at timestamp with time zone DEFAULT now() NOT NULL,
    visibility smallint DEFAULT 0 NOT NULL,
    allowed_entities jsonb DEFAULT '[]'::jsonb NOT NULL,
    processing_paused boolean DEFAULT false NOT NULL,
    processing_note text,
    estimated_killmails integer,
    last_processing_started_at timestamp with time zone,
    last_processing_duration_ms integer,
    last_processing_killmails integer,
    last_processing_error text
);


--
-- Name: celestials; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.celestials (
    item_id integer NOT NULL,
    item_name text,
    type_id integer,
    group_id integer,
    solar_system_id integer,
    constellation_id integer,
    region_id integer,
    orbit_id integer,
    x double precision,
    y double precision,
    z double precision,
    radius double precision,
    security double precision,
    celestial_index integer,
    orbit_index integer
);


--
-- Name: character_corporation_history; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.character_corporation_history (
    character_id integer NOT NULL,
    record_id integer NOT NULL,
    corporation_id integer NOT NULL,
    start_date timestamp with time zone NOT NULL
);


--
-- Name: characters; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.characters (
    character_id integer NOT NULL,
    name text NOT NULL,
    description text,
    birthday timestamp with time zone,
    gender text,
    race_id integer,
    bloodline_id integer,
    security_status real DEFAULT 0,
    title text,
    corporation_id integer,
    alliance_id integer,
    faction_id integer,
    deleted boolean DEFAULT false,
    last_active timestamp with time zone,
    created_at timestamp with time zone DEFAULT now(),
    updated_at timestamp with time zone DEFAULT now(),
    custom_description text,
    custom_description_format text DEFAULT 'markdown'::text,
    achievement_points integer DEFAULT 0 NOT NULL,
    corporation_history_queued_at timestamp with time zone,
    corporation_history_fetched_at timestamp with time zone,
    corporation_history_synced_corporation_id integer
)
WITH (autovacuum_vacuum_scale_factor='0.02', autovacuum_analyze_scale_factor='0.02');


--
-- Name: comment_edits; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.comment_edits (
    id bigint NOT NULL,
    comment_id bigint NOT NULL,
    body_md text NOT NULL,
    edited_at timestamp with time zone DEFAULT now() NOT NULL,
    edited_by bigint NOT NULL
);


--
-- Name: comment_edits_id_seq; Type: SEQUENCE; Schema: public; Owner: -
--

CREATE SEQUENCE public.comment_edits_id_seq
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;


--
-- Name: comment_edits_id_seq; Type: SEQUENCE OWNED BY; Schema: public; Owner: -
--

ALTER SEQUENCE public.comment_edits_id_seq OWNED BY public.comment_edits.id;


--
-- Name: comment_reports; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.comment_reports (
    id bigint NOT NULL,
    comment_id bigint NOT NULL,
    reporter_id bigint NOT NULL,
    reporter_name text NOT NULL,
    reason text NOT NULL,
    message text,
    created_at timestamp with time zone DEFAULT now() NOT NULL,
    resolved_at timestamp with time zone,
    resolved_by bigint,
    resolution text
);


--
-- Name: comment_reports_id_seq; Type: SEQUENCE; Schema: public; Owner: -
--

CREATE SEQUENCE public.comment_reports_id_seq
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;


--
-- Name: comment_reports_id_seq; Type: SEQUENCE OWNED BY; Schema: public; Owner: -
--

ALTER SEQUENCE public.comment_reports_id_seq OWNED BY public.comment_reports.id;


--
-- Name: comments; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.comments (
    id bigint NOT NULL,
    target_type smallint NOT NULL,
    target_id bigint NOT NULL,
    target_slug text,
    domain_id integer,
    parent_id bigint,
    root_id bigint,
    depth smallint DEFAULT 0 NOT NULL,
    body_md text NOT NULL,
    body_html text NOT NULL,
    character_id bigint NOT NULL,
    character_name text NOT NULL,
    corporation_id bigint NOT NULL,
    corporation_name text NOT NULL,
    alliance_id bigint,
    alliance_name text,
    created_at timestamp with time zone DEFAULT now() NOT NULL,
    updated_at timestamp with time zone DEFAULT now() NOT NULL,
    edited_at timestamp with time zone,
    deleted_at timestamp with time zone,
    deleted_by bigint,
    reports_count integer DEFAULT 0 NOT NULL,
    flagged boolean DEFAULT false NOT NULL,
    moderation_status smallint DEFAULT 0 NOT NULL,
    visibility smallint DEFAULT 0 NOT NULL
);


--
-- Name: comments_id_seq; Type: SEQUENCE; Schema: public; Owner: -
--

CREATE SEQUENCE public.comments_id_seq
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;


--
-- Name: comments_id_seq; Type: SEQUENCE OWNED BY; Schema: public; Owner: -
--

ALTER SEQUENCE public.comments_id_seq OWNED BY public.comments.id;


--
-- Name: config; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.config (
    key text NOT NULL,
    value text NOT NULL
);


--
-- Name: constellations; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.constellations (
    constellation_id integer NOT NULL,
    constellation_name text,
    region_id integer,
    x double precision,
    y double precision,
    z double precision,
    faction_id integer,
    radius double precision
);


--
-- Name: corporation_alliance_history; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.corporation_alliance_history (
    corporation_id integer NOT NULL,
    record_id integer NOT NULL,
    alliance_id integer,
    start_date timestamp with time zone NOT NULL
);


--
-- Name: corporation_wallet_balances; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.corporation_wallet_balances (
    corporation_id integer NOT NULL,
    division integer NOT NULL,
    balance numeric(30,2) NOT NULL,
    updated_at timestamp with time zone DEFAULT now() NOT NULL,
    CONSTRAINT corporation_wallet_balances_division_check CHECK (((division >= 1) AND (division <= 7)))
);


--
-- Name: corporation_wallet_journal; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.corporation_wallet_journal (
    corporation_id integer NOT NULL,
    division integer NOT NULL,
    journal_id bigint NOT NULL,
    date timestamp with time zone NOT NULL,
    ref_type text NOT NULL,
    description text NOT NULL,
    amount numeric(30,2),
    balance numeric(30,2),
    first_party_id bigint,
    second_party_id bigint,
    context_id bigint,
    context_id_type text,
    reason text,
    tax numeric(30,2),
    tax_receiver_id bigint,
    created_at timestamp with time zone DEFAULT now() NOT NULL,
    CONSTRAINT corporation_wallet_journal_division_check CHECK (((division >= 1) AND (division <= 7)))
);


--
-- Name: corporation_wallet_tokens; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.corporation_wallet_tokens (
    corporation_id integer NOT NULL,
    authorized_character_id integer NOT NULL,
    authorized_character_name text NOT NULL,
    authorized_character_owner_hash text CONSTRAINT corporation_wallet_tokens_authorized_character_owner_h_not_null NOT NULL,
    authorized_by_admin_character_id integer CONSTRAINT corporation_wallet_tokens_authorized_by_admin_characte_not_null NOT NULL,
    access_token text NOT NULL,
    refresh_token text NOT NULL,
    token_expiry timestamp with time zone NOT NULL,
    token_type text DEFAULT 'Bearer'::text NOT NULL,
    scopes text[] DEFAULT '{}'::text[] NOT NULL,
    disabled boolean DEFAULT false NOT NULL,
    last_balance_sync timestamp with time zone,
    last_journal_sync timestamp with time zone,
    last_error text,
    created_at timestamp with time zone DEFAULT now() NOT NULL,
    updated_at timestamp with time zone DEFAULT now() NOT NULL
);


--
-- Name: corporations; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.corporations (
    corporation_id integer NOT NULL,
    name text NOT NULL,
    ticker text NOT NULL,
    description text,
    date_founded timestamp with time zone,
    url text,
    ceo_id integer,
    creator_id integer,
    home_station_id integer,
    alliance_id integer,
    faction_id integer,
    member_count integer DEFAULT 0,
    shares bigint,
    tax_rate real DEFAULT 0,
    war_eligible boolean,
    deleted boolean DEFAULT false,
    created_at timestamp with time zone DEFAULT now(),
    updated_at timestamp with time zone DEFAULT now(),
    custom_description text,
    custom_description_format text DEFAULT 'markdown'::text,
    lp_tax_rate real,
    friendly_fire text,
    state text,
    type text,
    palette jsonb,
    alliance_history_queued_at timestamp with time zone,
    alliance_history_fetched_at timestamp with time zone,
    alliance_history_synced_alliance_id integer
);


--
-- Name: custom_domain_campaigns; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.custom_domain_campaigns (
    domain_id integer NOT NULL,
    campaign_id text NOT NULL,
    added_by_character_id integer NOT NULL,
    created_at timestamp with time zone DEFAULT now() NOT NULL
);


--
-- Name: custom_domains; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.custom_domains (
    id integer NOT NULL,
    subdomain text NOT NULL,
    custom_hostname text,
    user_id integer NOT NULL,
    entities jsonb DEFAULT '[]'::jsonb NOT NULL,
    theme jsonb DEFAULT '{}'::jsonb NOT NULL,
    navbar_links jsonb DEFAULT '[]'::jsonb NOT NULL,
    site_name text,
    site_description text,
    active boolean DEFAULT true NOT NULL,
    created_at timestamp with time zone DEFAULT now() NOT NULL,
    updated_at timestamp with time zone DEFAULT now() NOT NULL,
    widgets jsonb DEFAULT '{"top": [{"type": "mostValuable", "enabled": true}], "left": [{"type": "topCharacters", "enabled": true}, {"type": "topCorporations", "enabled": true}, {"type": "topAlliances", "enabled": true}, {"type": "topShips", "enabled": true}, {"type": "topSystems", "enabled": true}, {"type": "topRegions", "enabled": true}], "right": [{"type": "killList", "enabled": true, "killlistType": "latest"}], "columnRatio": "250px_1fr"}'::jsonb NOT NULL,
    campaign_policy integer DEFAULT 0 NOT NULL
);


--
-- Name: custom_domains_id_seq; Type: SEQUENCE; Schema: public; Owner: -
--

CREATE SEQUENCE public.custom_domains_id_seq
    AS integer
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;


--
-- Name: custom_domains_id_seq; Type: SEQUENCE OWNED BY; Schema: public; Owner: -
--

ALTER SEQUENCE public.custom_domains_id_seq OWNED BY public.custom_domains.id;


--
-- Name: custom_prices; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.custom_prices (
    type_id integer NOT NULL,
    date date NOT NULL,
    price double precision NOT NULL
);


--
-- Name: domain_assets; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.domain_assets (
    id integer NOT NULL,
    domain_id integer NOT NULL,
    type text NOT NULL,
    status text DEFAULT 'pending'::text NOT NULL,
    storage_key text NOT NULL,
    content_type text NOT NULL,
    file_size integer NOT NULL,
    uploaded_by integer NOT NULL,
    reviewed_by integer,
    reviewed_at timestamp with time zone,
    reject_reason text,
    created_at timestamp with time zone DEFAULT now() NOT NULL,
    file_hash text DEFAULT ''::text NOT NULL
);


--
-- Name: domain_assets_id_seq; Type: SEQUENCE; Schema: public; Owner: -
--

CREATE SEQUENCE public.domain_assets_id_seq
    AS integer
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;


--
-- Name: domain_assets_id_seq; Type: SEQUENCE OWNED BY; Schema: public; Owner: -
--

ALTER SEQUENCE public.domain_assets_id_seq OWNED BY public.domain_assets.id;


--
-- Name: ek_wallet_accounts; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.ek_wallet_accounts (
    character_id integer NOT NULL,
    balance numeric(30,2) DEFAULT 0 NOT NULL,
    created_at timestamp with time zone DEFAULT now() NOT NULL,
    updated_at timestamp with time zone DEFAULT now() NOT NULL,
    reserved_balance numeric(30,2) DEFAULT 0 NOT NULL,
    CONSTRAINT ek_wallet_accounts_available_balance_check CHECK ((reserved_balance <= balance)),
    CONSTRAINT ek_wallet_accounts_balance_check CHECK ((balance >= (0)::numeric)),
    CONSTRAINT ek_wallet_accounts_reserved_balance_check CHECK ((reserved_balance >= (0)::numeric))
);


--
-- Name: ek_wallet_reservations; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.ek_wallet_reservations (
    id bigint NOT NULL,
    character_id integer NOT NULL,
    external_reference text NOT NULL,
    transaction_type smallint NOT NULL,
    amount numeric(30,2) NOT NULL,
    status smallint DEFAULT 0 NOT NULL,
    description text NOT NULL,
    metadata jsonb DEFAULT '{}'::jsonb NOT NULL,
    expires_at timestamp with time zone,
    captured_amount numeric(30,2),
    captured_transaction_id bigint,
    close_reason text,
    closed_at timestamp with time zone,
    created_at timestamp with time zone DEFAULT now() NOT NULL,
    updated_at timestamp with time zone DEFAULT now() NOT NULL,
    campaign_id text,
    CONSTRAINT ek_wallet_reservations_amount_check CHECK ((amount > (0)::numeric)),
    CONSTRAINT ek_wallet_reservations_campaign_source_check CHECK (((transaction_type <> 3) OR (campaign_id IS NOT NULL))),
    CONSTRAINT ek_wallet_reservations_expiry_check CHECK (((expires_at IS NULL) OR (expires_at > created_at))),
    CONSTRAINT ek_wallet_reservations_lifecycle_check CHECK ((((status = 0) AND (captured_amount IS NULL) AND (captured_transaction_id IS NULL) AND (closed_at IS NULL)) OR ((status = 1) AND (captured_amount > (0)::numeric) AND (captured_amount <= amount) AND (captured_transaction_id IS NOT NULL) AND (closed_at IS NOT NULL)) OR ((status = ANY (ARRAY[2, 3])) AND (captured_amount IS NULL) AND (captured_transaction_id IS NULL) AND (closed_at IS NOT NULL)))),
    CONSTRAINT ek_wallet_reservations_status_check CHECK (((status >= 0) AND (status <= 3))),
    CONSTRAINT ek_wallet_reservations_transaction_type_check CHECK ((transaction_type = ANY (ARRAY[1, 3])))
);


--
-- Name: ek_wallet_reservations_id_seq; Type: SEQUENCE; Schema: public; Owner: -
--

CREATE SEQUENCE public.ek_wallet_reservations_id_seq
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;


--
-- Name: ek_wallet_reservations_id_seq; Type: SEQUENCE OWNED BY; Schema: public; Owner: -
--

ALTER SEQUENCE public.ek_wallet_reservations_id_seq OWNED BY public.ek_wallet_reservations.id;


--
-- Name: ek_wallet_transactions; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.ek_wallet_transactions (
    id bigint NOT NULL,
    character_id integer NOT NULL,
    type smallint NOT NULL,
    amount numeric(30,2) NOT NULL,
    balance_after numeric(30,2) NOT NULL,
    description text NOT NULL,
    external_reference text NOT NULL,
    corporation_id integer,
    division integer,
    journal_id bigint,
    metadata jsonb DEFAULT '{}'::jsonb NOT NULL,
    created_at timestamp with time zone DEFAULT now() NOT NULL,
    related_transaction_id bigint,
    campaign_id text,
    CONSTRAINT ek_wallet_transactions_amount_check CHECK ((amount <> (0)::numeric)),
    CONSTRAINT ek_wallet_transactions_balance_after_check CHECK ((balance_after >= (0)::numeric)),
    CONSTRAINT ek_wallet_transactions_campaign_source_check CHECK (((type <> 3) OR (campaign_id IS NOT NULL))),
    CONSTRAINT ek_wallet_transactions_direction_check CHECK ((((type = ANY (ARRAY[0, 2, 4])) AND (amount > (0)::numeric)) OR ((type = ANY (ARRAY[1, 3])) AND (amount < (0)::numeric)) OR (type = 5))),
    CONSTRAINT ek_wallet_transactions_journal_source_check CHECK ((((corporation_id IS NULL) AND (division IS NULL) AND (journal_id IS NULL)) OR ((corporation_id IS NOT NULL) AND (division IS NOT NULL) AND (journal_id IS NOT NULL)))),
    CONSTRAINT ek_wallet_transactions_refund_source_check CHECK (((type <> 2) OR (related_transaction_id IS NOT NULL))),
    CONSTRAINT ek_wallet_transactions_type_check CHECK (((type >= 0) AND (type <= 5)))
);


--
-- Name: ek_wallet_transactions_id_seq; Type: SEQUENCE; Schema: public; Owner: -
--

CREATE SEQUENCE public.ek_wallet_transactions_id_seq
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;


--
-- Name: ek_wallet_transactions_id_seq; Type: SEQUENCE OWNED BY; Schema: public; Owner: -
--

ALTER SEQUENCE public.ek_wallet_transactions_id_seq OWNED BY public.ek_wallet_transactions.id;


--
-- Name: entity_achievements; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.entity_achievements (
    entity_id integer NOT NULL,
    achievement_id text NOT NULL,
    current_count integer DEFAULT 0,
    threshold integer NOT NULL,
    completion_tiers integer DEFAULT 0,
    is_completed boolean DEFAULT false,
    points integer DEFAULT 0,
    completed_at timestamp with time zone,
    last_updated timestamp with time zone DEFAULT now()
)
WITH (autovacuum_vacuum_scale_factor='0.02', autovacuum_analyze_scale_factor='0.02');


--
-- Name: entity_snapshots; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.entity_snapshots (
    entity_type text NOT NULL,
    entity_id integer NOT NULL,
    date date NOT NULL,
    member_count integer,
    avg_sec_status real,
    pirate_members integer,
    carebear_members integer,
    neutral_members integer,
    total_achievement_points integer,
    avg_achievement_points real,
    top_achiever_id integer,
    top_achiever_points integer
);


--
-- Name: esi_killmail_delayed; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.esi_killmail_delayed (
    killmail_id integer NOT NULL,
    killmail_hash text NOT NULL,
    delayed_until timestamp with time zone NOT NULL,
    created_at timestamp with time zone DEFAULT now() NOT NULL
);


--
-- Name: esi_request_logs; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.esi_request_logs (
    id bigint NOT NULL,
    character_id integer NOT NULL,
    endpoint text NOT NULL,
    method text DEFAULT 'GET'::text NOT NULL,
    status_code smallint,
    success boolean DEFAULT true NOT NULL,
    error_message text,
    items_returned integer DEFAULT 0,
    new_items integer DEFAULT 0,
    source text NOT NULL,
    request_duration_ms integer,
    esi_error_limit_remain smallint,
    created_at timestamp with time zone DEFAULT now() NOT NULL,
    new_item_ids jsonb
);


--
-- Name: esi_request_logs_id_seq; Type: SEQUENCE; Schema: public; Owner: -
--

CREATE SEQUENCE public.esi_request_logs_id_seq
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;


--
-- Name: esi_request_logs_id_seq; Type: SEQUENCE OWNED BY; Schema: public; Owner: -
--

ALTER SEQUENCE public.esi_request_logs_id_seq OWNED BY public.esi_request_logs.id;


--
-- Name: factions; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.factions (
    faction_id integer NOT NULL,
    name text,
    description text,
    corporation_id integer,
    militia_corporation_id integer,
    solar_system_id integer,
    icon_id integer,
    size_factor double precision,
    station_count integer,
    station_system_count integer,
    race_ids integer[]
);


--
-- Name: feed_queue; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.feed_queue (
    seq bigint NOT NULL,
    killmail_id integer NOT NULL,
    inserted_at timestamp with time zone DEFAULT now() NOT NULL
);


--
-- Name: feed_queue_seq_seq; Type: SEQUENCE; Schema: public; Owner: -
--

CREATE SEQUENCE public.feed_queue_seq_seq
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;


--
-- Name: feed_queue_seq_seq; Type: SEQUENCE OWNED BY; Schema: public; Owner: -
--

ALTER SEQUENCE public.feed_queue_seq_seq OWNED BY public.feed_queue.seq;


--
-- Name: fitting_items; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.fitting_items (
    fit_hash text NOT NULL,
    slot_group smallint NOT NULL,
    ordinal smallint NOT NULL,
    type_id integer NOT NULL,
    charge_type_id integer,
    quantity smallint DEFAULT 1 NOT NULL,
    CONSTRAINT fitting_items_quantity_positive CHECK ((quantity >= 1))
);


--
-- Name: fittings; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.fittings (
    fit_hash text NOT NULL,
    ship_type_id integer NOT NULL,
    family_hash text NOT NULL,
    first_seen_at timestamp with time zone DEFAULT now() NOT NULL
);


--
-- Name: fw_faction_stats; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.fw_faction_stats (
    faction_id integer NOT NULL,
    pilots integer DEFAULT 0 NOT NULL,
    systems_controlled integer DEFAULT 0 NOT NULL,
    kills_yesterday integer DEFAULT 0 NOT NULL,
    kills_last_week integer DEFAULT 0 NOT NULL,
    kills_total integer DEFAULT 0 NOT NULL,
    vp_yesterday integer DEFAULT 0 NOT NULL,
    vp_last_week integer DEFAULT 0 NOT NULL,
    vp_total bigint DEFAULT 0 NOT NULL,
    updated_at timestamp with time zone DEFAULT now() NOT NULL
);


--
-- Name: fw_leaderboards; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.fw_leaderboards (
    entity_type text NOT NULL,
    entity_id integer NOT NULL,
    metric text NOT NULL,
    period text NOT NULL,
    amount bigint DEFAULT 0 NOT NULL,
    updated_at timestamp with time zone DEFAULT now() NOT NULL
);


--
-- Name: fw_system_history; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.fw_system_history (
    id integer NOT NULL,
    solar_system_id integer NOT NULL,
    old_occupier_faction_id integer NOT NULL,
    new_occupier_faction_id integer NOT NULL,
    flipped_at timestamp with time zone DEFAULT now() NOT NULL
);


--
-- Name: fw_system_history_id_seq; Type: SEQUENCE; Schema: public; Owner: -
--

ALTER TABLE public.fw_system_history ALTER COLUMN id ADD GENERATED ALWAYS AS IDENTITY (
    SEQUENCE NAME public.fw_system_history_id_seq
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1
);


--
-- Name: fw_systems; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.fw_systems (
    solar_system_id integer NOT NULL,
    owner_faction_id integer NOT NULL,
    occupier_faction_id integer NOT NULL,
    contested text NOT NULL,
    victory_points integer DEFAULT 0 NOT NULL,
    victory_points_threshold integer DEFAULT 0 NOT NULL,
    updated_at timestamp with time zone DEFAULT now() NOT NULL
);


--
-- Name: insurance_prices; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.insurance_prices (
    type_id integer NOT NULL,
    level_name text NOT NULL,
    cost double precision NOT NULL,
    payout double precision NOT NULL
);


--
-- Name: inv_categories; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.inv_categories (
    category_id integer NOT NULL,
    name text,
    published boolean,
    icon_id integer
);


--
-- Name: inv_flags; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.inv_flags (
    flag_id integer NOT NULL,
    flag_name text,
    flag_text text,
    order_id integer
);


--
-- Name: inv_groups; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.inv_groups (
    group_id integer NOT NULL,
    category_id integer,
    name text,
    published boolean,
    icon_id integer
);


--
-- Name: inv_market_groups; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.inv_market_groups (
    market_group_id integer NOT NULL,
    parent_group_id integer,
    name text,
    description text,
    icon_id integer,
    has_types boolean
);


--
-- Name: inv_meta_groups; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.inv_meta_groups (
    meta_group_id integer NOT NULL,
    name text,
    icon_id integer,
    icon_suffix text
);


--
-- Name: inv_types; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.inv_types (
    type_id integer NOT NULL,
    group_id integer,
    category_id integer,
    name text,
    description text,
    mass double precision,
    volume double precision,
    capacity double precision,
    portion_size integer,
    packaged_volume double precision,
    radius double precision,
    published boolean,
    market_group_id integer,
    icon_id integer,
    sound_id integer,
    graphic_id integer,
    meta_group_id integer,
    race_id integer,
    faction_id integer,
    base_price double precision,
    variation_parent_type_id integer
);


--
-- Name: killmail_attackers; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.killmail_attackers (
    killmail_id integer CONSTRAINT killmail_attackers_flat_killmail_id_not_null NOT NULL,
    attacker_index integer CONSTRAINT killmail_attackers_flat_attacker_index_not_null NOT NULL,
    character_id integer,
    corporation_id integer,
    alliance_id integer,
    faction_id integer,
    ship_type_id integer,
    ship_group_id integer,
    weapon_type_id integer,
    damage_done integer DEFAULT 0,
    final_blow boolean DEFAULT false,
    security_status real,
    killmail_time timestamp with time zone NOT NULL
)
WITH (autovacuum_vacuum_scale_factor='0.02', autovacuum_analyze_scale_factor='0.02');


--
-- Name: killmail_fittings; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.killmail_fittings (
    killmail_id integer NOT NULL,
    fit_hash text NOT NULL,
    ship_type_id integer NOT NULL,
    kill_time timestamp with time zone NOT NULL,
    victim_alliance_id integer,
    victim_corporation_id integer
);


--
-- Name: killmail_items; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.killmail_items (
    killmail_id integer CONSTRAINT killmail_items_flat_killmail_id_not_null NOT NULL,
    item_index integer CONSTRAINT killmail_items_flat_item_index_not_null NOT NULL,
    type_id integer CONSTRAINT killmail_items_flat_type_id_not_null NOT NULL,
    flag_id integer CONSTRAINT killmail_items_flat_flag_id_not_null NOT NULL,
    quantity_dropped bigint DEFAULT 0,
    quantity_destroyed bigint DEFAULT 0,
    singleton smallint DEFAULT 0,
    parent_index integer
)
WITH (autovacuum_vacuum_scale_factor='0.02', autovacuum_analyze_scale_factor='0.02');


--
-- Name: killmail_processing; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.killmail_processing (
    killmail_id integer NOT NULL,
    effects_completed integer DEFAULT 0 NOT NULL,
    created_at timestamp with time zone DEFAULT now() NOT NULL,
    updated_at timestamp with time zone DEFAULT now() NOT NULL,
    CONSTRAINT killmail_processing_effects_completed_check CHECK ((effects_completed >= 0))
);


--
-- Name: killmails; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.killmails (
    killmail_id integer CONSTRAINT killmails_flat_killmail_id_not_null NOT NULL,
    killmail_time timestamp with time zone CONSTRAINT killmails_flat_killmail_time_not_null NOT NULL,
    killmail_hash text CONSTRAINT killmails_flat_killmail_hash_not_null NOT NULL,
    solar_system_id integer CONSTRAINT killmails_flat_solar_system_id_not_null NOT NULL,
    constellation_id integer,
    region_id integer,
    position_x double precision,
    position_y double precision,
    position_z double precision,
    victim_character_id integer,
    victim_corporation_id integer,
    victim_alliance_id integer,
    victim_faction_id integer,
    victim_ship_type_id integer,
    victim_ship_group_id integer,
    victim_damage_taken integer,
    total_value double precision DEFAULT 0,
    fitted_value double precision DEFAULT 0,
    dropped_value double precision DEFAULT 0,
    destroyed_value double precision DEFAULT 0,
    points integer DEFAULT 0,
    attacker_count integer DEFAULT 0,
    is_npc boolean DEFAULT false,
    is_solo boolean DEFAULT false,
    war_id integer,
    blob boolean DEFAULT false
)
WITH (autovacuum_vacuum_scale_factor='0.02', autovacuum_analyze_scale_factor='0.02');


--
-- Name: kills_daily_count; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.kills_daily_count (
    date date NOT NULL,
    type text NOT NULL,
    count bigint DEFAULT 0 NOT NULL
);


--
-- Name: moderation_queue; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.moderation_queue (
    id bigint NOT NULL,
    target_kind smallint NOT NULL,
    target_id bigint NOT NULL,
    body text NOT NULL,
    body_format text,
    rendered_html text,
    character_id bigint NOT NULL,
    character_name text NOT NULL,
    corporation_id bigint,
    corporation_name text,
    alliance_id bigint,
    alliance_name text,
    ai_action text,
    ai_category text,
    ai_max_score real,
    ai_scores jsonb,
    ai_source text,
    status smallint DEFAULT 0 NOT NULL,
    submitted_at timestamp with time zone DEFAULT now() NOT NULL,
    reviewed_at timestamp with time zone,
    reviewed_by bigint,
    review_notes text
);


--
-- Name: moderation_queue_id_seq; Type: SEQUENCE; Schema: public; Owner: -
--

CREATE SEQUENCE public.moderation_queue_id_seq
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;


--
-- Name: moderation_queue_id_seq; Type: SEQUENCE OWNED BY; Schema: public; Owner: -
--

ALTER SEQUENCE public.moderation_queue_id_seq OWNED BY public.moderation_queue.id;


--
-- Name: npc_corporations; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.npc_corporations (
    corporation_id integer NOT NULL,
    name text,
    description text,
    ticker_name text,
    ceo_id integer,
    station_id integer,
    size text,
    extent text,
    tax_rate double precision,
    deleted boolean
);


--
-- Name: old_killmail_attackers; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.old_killmail_attackers (
    id integer NOT NULL,
    killmail_id integer NOT NULL,
    name text,
    corp text,
    alliance text,
    ship text,
    weapon text,
    security_status real,
    final_blow boolean DEFAULT false,
    character_id integer,
    corporation_id integer,
    alliance_id integer,
    ship_type_id integer,
    weapon_type_id integer
);


--
-- Name: old_killmail_attackers_id_seq; Type: SEQUENCE; Schema: public; Owner: -
--

CREATE SEQUENCE public.old_killmail_attackers_id_seq
    AS integer
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;


--
-- Name: old_killmail_attackers_id_seq; Type: SEQUENCE OWNED BY; Schema: public; Owner: -
--

ALTER SEQUENCE public.old_killmail_attackers_id_seq OWNED BY public.old_killmail_attackers.id;


--
-- Name: old_killmail_items; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.old_killmail_items (
    id integer NOT NULL,
    killmail_id integer NOT NULL,
    name text NOT NULL,
    quantity integer DEFAULT 1,
    location text,
    type_id integer
);


--
-- Name: old_killmail_items_id_seq; Type: SEQUENCE; Schema: public; Owner: -
--

CREATE SEQUENCE public.old_killmail_items_id_seq
    AS integer
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;


--
-- Name: old_killmail_items_id_seq; Type: SEQUENCE OWNED BY; Schema: public; Owner: -
--

ALTER SEQUENCE public.old_killmail_items_id_seq OWNED BY public.old_killmail_items.id;


--
-- Name: old_killmails; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.old_killmails (
    killmail_id integer NOT NULL,
    killmail_time timestamp without time zone,
    raw_text text NOT NULL,
    victim_name text,
    victim_corp text,
    victim_alliance text,
    victim_ship text,
    system_name text,
    security real,
    total_value real,
    final_blow_name text,
    victim_character_id integer,
    victim_corporation_id integer,
    victim_alliance_id integer,
    victim_ship_type_id integer,
    solar_system_id integer
);


--
-- Name: prices; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.prices (
    type_id integer NOT NULL,
    region_id integer NOT NULL,
    date date NOT NULL,
    average double precision,
    highest double precision,
    lowest double precision,
    order_count integer,
    volume bigint
)
WITH (autovacuum_vacuum_scale_factor='0.02', autovacuum_analyze_scale_factor='0.02');


--
-- Name: races; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.races (
    race_id integer NOT NULL,
    race_name text,
    description text,
    short_description text,
    icon_id integer
);


--
-- Name: regions; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.regions (
    region_id integer NOT NULL,
    name text,
    description text,
    center_x double precision,
    center_y double precision,
    center_z double precision,
    faction_id integer,
    wormhole_class_id integer,
    nebula_id integer
);


--
-- Name: scans; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.scans (
    hash text NOT NULL,
    scan_type text NOT NULL,
    input text NOT NULL,
    result jsonb NOT NULL,
    character_id integer NOT NULL,
    created_at timestamp with time zone DEFAULT now() NOT NULL
);


--
-- Name: solar_system_jumps; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.solar_system_jumps (
    from_solar_system_id integer NOT NULL,
    to_solar_system_id integer NOT NULL,
    from_constellation_id integer,
    to_constellation_id integer,
    from_region_id integer,
    to_region_id integer
);


--
-- Name: solar_systems; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.solar_systems (
    solar_system_id integer NOT NULL,
    system_name text,
    constellation_id integer,
    region_id integer,
    x double precision,
    y double precision,
    z double precision,
    security double precision,
    security_class text,
    faction_id integer,
    border boolean,
    fringe boolean,
    corridor boolean,
    hub boolean,
    international boolean,
    regional boolean,
    luminosity double precision,
    radius double precision,
    sun_type_id integer,
    x2d double precision,
    z2d double precision
);


--
-- Name: sovereignty; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.sovereignty (
    system_id integer NOT NULL,
    alliance_id integer,
    corporation_id integer,
    faction_id integer,
    date_added timestamp with time zone NOT NULL,
    updated_at timestamp with time zone DEFAULT now()
);


--
-- Name: sovereignty_history; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.sovereignty_history (
    id integer NOT NULL,
    system_id integer NOT NULL,
    alliance_id integer,
    corporation_id integer,
    faction_id integer,
    date_added timestamp with time zone NOT NULL
);


--
-- Name: sovereignty_history_id_seq; Type: SEQUENCE; Schema: public; Owner: -
--

CREATE SEQUENCE public.sovereignty_history_id_seq
    AS integer
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;


--
-- Name: sovereignty_history_id_seq; Type: SEQUENCE OWNED BY; Schema: public; Owner: -
--

ALTER SEQUENCE public.sovereignty_history_id_seq OWNED BY public.sovereignty_history.id;


--
-- Name: station_operations; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.station_operations (
    operation_id integer NOT NULL,
    operation_name text,
    description text,
    activity_id integer,
    services integer[]
);


--
-- Name: stations; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.stations (
    station_id integer NOT NULL,
    station_name text,
    type_id integer,
    corporation_id integer,
    solar_system_id integer,
    constellation_id integer,
    region_id integer,
    operation_id integer,
    orbit_id integer,
    orbit_index integer,
    celestial_index integer,
    x double precision,
    y double precision,
    z double precision,
    security double precision,
    reprocessing_efficiency double precision,
    reprocessing_stations_take double precision
);


--
-- Name: stats; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.stats (
    entity_type smallint NOT NULL,
    entity_id integer NOT NULL,
    period_type smallint NOT NULL,
    period_start date NOT NULL,
    kills integer DEFAULT 0,
    losses integer DEFAULT 0,
    solo_kills integer DEFAULT 0,
    solo_losses integer DEFAULT 0,
    npc_losses integer DEFAULT 0,
    final_blows integer DEFAULT 0,
    points integer DEFAULT 0,
    isk_destroyed double precision DEFAULT 0,
    isk_lost double precision DEFAULT 0,
    damage_dealt bigint DEFAULT 0,
    damage_taken bigint DEFAULT 0,
    sum_attacker_count bigint DEFAULT 0
);


--
-- Name: stats_breakdowns; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.stats_breakdowns (
    entity_type smallint NOT NULL,
    entity_id integer NOT NULL,
    period_type smallint NOT NULL,
    period_start date NOT NULL,
    dim_category smallint NOT NULL,
    dim_id integer NOT NULL,
    kills integer DEFAULT 0,
    losses integer DEFAULT 0,
    isk_destroyed double precision DEFAULT 0,
    isk_lost double precision DEFAULT 0,
    last_killmail_id integer,
    last_killmail_time timestamp with time zone
);


--
-- Name: stats_leaderboards; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.stats_leaderboards (
    entity_type smallint NOT NULL,
    period_type smallint NOT NULL,
    period_start date NOT NULL,
    metric smallint NOT NULL,
    rank smallint NOT NULL,
    entity_id integer NOT NULL,
    value double precision DEFAULT 0
);


--
-- Name: structures; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.structures (
    structure_id bigint NOT NULL,
    name text,
    owner_id integer,
    solar_system_id integer,
    constellation_id integer,
    region_id integer,
    type_id integer,
    x double precision,
    y double precision,
    z double precision,
    is_public boolean,
    is_market boolean,
    first_seen timestamp with time zone,
    last_seen timestamp with time zone
);


--
-- Name: system_activity; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.system_activity (
    system_id integer NOT NULL,
    "timestamp" timestamp with time zone NOT NULL,
    ship_kills integer DEFAULT 0,
    npc_kills integer DEFAULT 0,
    pod_kills integer DEFAULT 0,
    ship_jumps integer DEFAULT 0
);


--
-- Name: type_dogma_attributes; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.type_dogma_attributes (
    type_id integer NOT NULL,
    attribute_id integer NOT NULL,
    value double precision NOT NULL
);


--
-- Name: type_dogma_effects; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.type_dogma_effects (
    type_id integer NOT NULL,
    effect_id integer NOT NULL,
    is_default boolean DEFAULT false
);


--
-- Name: type_materials; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.type_materials (
    type_id integer NOT NULL,
    material_type_id integer NOT NULL,
    quantity integer NOT NULL
);


--
-- Name: user_config; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.user_config (
    character_id integer NOT NULL,
    key text NOT NULL,
    value jsonb NOT NULL,
    updated_at timestamp with time zone DEFAULT now() NOT NULL
);


--
-- Name: user_esi_tokens; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.user_esi_tokens (
    id integer NOT NULL,
    character_id integer NOT NULL,
    access_token text NOT NULL,
    refresh_token text NOT NULL,
    token_expiry timestamp with time zone NOT NULL,
    token_type text DEFAULT 'Bearer'::text NOT NULL,
    scopes text[] DEFAULT '{}'::text[] NOT NULL,
    created_at timestamp with time zone DEFAULT now() NOT NULL,
    updated_at timestamp with time zone DEFAULT now() NOT NULL,
    last_fetched timestamp with time zone,
    disabled boolean DEFAULT false NOT NULL,
    delay integer DEFAULT 0 NOT NULL,
    revoked_scopes text[] DEFAULT '{}'::text[] NOT NULL,
    character_failure_count integer DEFAULT 0 NOT NULL,
    scopes_revoked_at timestamp with time zone
);


--
-- Name: user_esi_tokens_id_seq; Type: SEQUENCE; Schema: public; Owner: -
--

CREATE SEQUENCE public.user_esi_tokens_id_seq
    AS integer
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;


--
-- Name: user_esi_tokens_id_seq; Type: SEQUENCE OWNED BY; Schema: public; Owner: -
--

ALTER SEQUENCE public.user_esi_tokens_id_seq OWNED BY public.user_esi_tokens.id;


--
-- Name: user_fitting_items; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.user_fitting_items (
    fit_id text NOT NULL,
    slot_group smallint NOT NULL,
    ordinal smallint NOT NULL,
    type_id integer NOT NULL,
    state smallint NOT NULL,
    charge_type_id integer,
    quantity smallint DEFAULT 1 NOT NULL,
    CONSTRAINT user_fitting_items_quantity_positive CHECK ((quantity >= 1))
);


--
-- Name: user_fitting_ratings; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.user_fitting_ratings (
    fit_id text NOT NULL,
    rater_character_id integer NOT NULL,
    rating smallint NOT NULL,
    created_at timestamp with time zone DEFAULT now() NOT NULL,
    updated_at timestamp with time zone DEFAULT now() NOT NULL,
    CONSTRAINT user_fitting_ratings_rating_check CHECK (((rating >= 1) AND (rating <= 5)))
);


--
-- Name: user_fittings; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.user_fittings (
    fit_id text NOT NULL,
    owner_character_id integer NOT NULL,
    ship_type_id integer NOT NULL,
    name text NOT NULL,
    description text,
    visibility smallint NOT NULL,
    created_at timestamp with time zone DEFAULT now() NOT NULL,
    updated_at timestamp with time zone DEFAULT now() NOT NULL,
    rating_avg real,
    rating_count integer DEFAULT 0 NOT NULL
);


--
-- Name: user_sessions; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.user_sessions (
    id integer NOT NULL,
    session_id uuid NOT NULL,
    character_id integer NOT NULL,
    ip_address text,
    country_code text,
    user_agent text,
    client_hint text,
    created_at timestamp with time zone DEFAULT now() NOT NULL,
    last_seen_at timestamp with time zone DEFAULT now() NOT NULL,
    expires_at timestamp with time zone NOT NULL
);


--
-- Name: user_sessions_id_seq; Type: SEQUENCE; Schema: public; Owner: -
--

CREATE SEQUENCE public.user_sessions_id_seq
    AS integer
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;


--
-- Name: user_sessions_id_seq; Type: SEQUENCE OWNED BY; Schema: public; Owner: -
--

ALTER SEQUENCE public.user_sessions_id_seq OWNED BY public.user_sessions.id;


--
-- Name: users; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.users (
    character_id integer NOT NULL,
    character_name text NOT NULL,
    character_owner_hash text NOT NULL,
    session_id uuid NOT NULL,
    is_admin boolean DEFAULT false NOT NULL,
    last_login timestamp with time zone DEFAULT now(),
    created_at timestamp with time zone DEFAULT now() NOT NULL,
    updated_at timestamp with time zone DEFAULT now() NOT NULL,
    last_seen_notification_id bigint DEFAULT 0 NOT NULL,
    discord_user_id text
);


--
-- Name: wallet_journal_references; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.wallet_journal_references (
    corporation_id integer NOT NULL,
    division integer NOT NULL,
    journal_id bigint NOT NULL,
    reference_type text NOT NULL,
    reference_id text NOT NULL,
    status smallint NOT NULL,
    amount numeric(30,2),
    note text,
    created_at timestamp with time zone DEFAULT now() NOT NULL,
    updated_at timestamp with time zone DEFAULT now() NOT NULL,
    CONSTRAINT wallet_journal_references_status_check CHECK (((status >= 0) AND (status <= 3)))
);


--
-- Name: war_allies; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.war_allies (
    id integer NOT NULL,
    war_id integer NOT NULL,
    alliance_id integer,
    corporation_id integer
);


--
-- Name: war_allies_id_seq; Type: SEQUENCE; Schema: public; Owner: -
--

CREATE SEQUENCE public.war_allies_id_seq
    AS integer
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;


--
-- Name: war_allies_id_seq; Type: SEQUENCE OWNED BY; Schema: public; Owner: -
--

ALTER SEQUENCE public.war_allies_id_seq OWNED BY public.war_allies.id;


--
-- Name: war_interactions; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.war_interactions (
    war_id integer NOT NULL,
    side smallint NOT NULL,
    category smallint NOT NULL,
    target_type smallint NOT NULL,
    target_id integer NOT NULL,
    count integer DEFAULT 0,
    isk_value double precision DEFAULT 0,
    last_killmail_id integer,
    last_killmail_time timestamp with time zone
);


--
-- Name: wars; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.wars (
    war_id integer NOT NULL,
    declared timestamp with time zone,
    started timestamp with time zone,
    finished timestamp with time zone,
    retracted timestamp with time zone,
    mutual boolean DEFAULT false,
    open_for_allies boolean DEFAULT false,
    aggressor_alliance_id integer,
    aggressor_corporation_id integer,
    aggressor_isk_destroyed double precision,
    aggressor_ships_killed integer,
    defender_alliance_id integer,
    defender_corporation_id integer,
    defender_isk_destroyed double precision,
    defender_ships_killed integer,
    created_at timestamp with time zone DEFAULT now(),
    updated_at timestamp with time zone DEFAULT now()
);


--
-- Name: __drizzle_migrations id; Type: DEFAULT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.__drizzle_migrations ALTER COLUMN id SET DEFAULT nextval('__drizzle_migrations_id_seq'::regclass);


--
-- Name: announcements id; Type: DEFAULT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.announcements ALTER COLUMN id SET DEFAULT nextval('announcements_id_seq'::regclass);


--
-- Name: battle_team_members id; Type: DEFAULT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.battle_team_members ALTER COLUMN id SET DEFAULT nextval('battle_team_members_id_seq'::regclass);


--
-- Name: battle_teams id; Type: DEFAULT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.battle_teams ALTER COLUMN id SET DEFAULT nextval('battle_teams_id_seq'::regclass);


--
-- Name: battles battle_id; Type: DEFAULT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.battles ALTER COLUMN battle_id SET DEFAULT nextval('battles_battle_id_seq'::regclass);


--
-- Name: blog_posts id; Type: DEFAULT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.blog_posts ALTER COLUMN id SET DEFAULT nextval('blog_posts_id_seq'::regclass);


--
-- Name: comment_edits id; Type: DEFAULT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.comment_edits ALTER COLUMN id SET DEFAULT nextval('comment_edits_id_seq'::regclass);


--
-- Name: comment_reports id; Type: DEFAULT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.comment_reports ALTER COLUMN id SET DEFAULT nextval('comment_reports_id_seq'::regclass);


--
-- Name: comments id; Type: DEFAULT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.comments ALTER COLUMN id SET DEFAULT nextval('comments_id_seq'::regclass);


--
-- Name: custom_domains id; Type: DEFAULT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.custom_domains ALTER COLUMN id SET DEFAULT nextval('custom_domains_id_seq'::regclass);


--
-- Name: domain_assets id; Type: DEFAULT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.domain_assets ALTER COLUMN id SET DEFAULT nextval('domain_assets_id_seq'::regclass);


--
-- Name: ek_wallet_reservations id; Type: DEFAULT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.ek_wallet_reservations ALTER COLUMN id SET DEFAULT nextval('ek_wallet_reservations_id_seq'::regclass);


--
-- Name: ek_wallet_transactions id; Type: DEFAULT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.ek_wallet_transactions ALTER COLUMN id SET DEFAULT nextval('ek_wallet_transactions_id_seq'::regclass);


--
-- Name: esi_request_logs id; Type: DEFAULT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.esi_request_logs ALTER COLUMN id SET DEFAULT nextval('esi_request_logs_id_seq'::regclass);


--
-- Name: feed_queue seq; Type: DEFAULT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.feed_queue ALTER COLUMN seq SET DEFAULT nextval('feed_queue_seq_seq'::regclass);


--
-- Name: moderation_queue id; Type: DEFAULT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.moderation_queue ALTER COLUMN id SET DEFAULT nextval('moderation_queue_id_seq'::regclass);


--
-- Name: old_killmail_attackers id; Type: DEFAULT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.old_killmail_attackers ALTER COLUMN id SET DEFAULT nextval('old_killmail_attackers_id_seq'::regclass);


--
-- Name: old_killmail_items id; Type: DEFAULT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.old_killmail_items ALTER COLUMN id SET DEFAULT nextval('old_killmail_items_id_seq'::regclass);


--
-- Name: sovereignty_history id; Type: DEFAULT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.sovereignty_history ALTER COLUMN id SET DEFAULT nextval('sovereignty_history_id_seq'::regclass);


--
-- Name: user_esi_tokens id; Type: DEFAULT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.user_esi_tokens ALTER COLUMN id SET DEFAULT nextval('user_esi_tokens_id_seq'::regclass);


--
-- Name: user_sessions id; Type: DEFAULT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.user_sessions ALTER COLUMN id SET DEFAULT nextval('user_sessions_id_seq'::regclass);


--
-- Name: war_allies id; Type: DEFAULT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.war_allies ALTER COLUMN id SET DEFAULT nextval('war_allies_id_seq'::regclass);


--
-- Name: __drizzle_migrations __drizzle_migrations_hash_key; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.__drizzle_migrations
    ADD CONSTRAINT __drizzle_migrations_hash_key UNIQUE (hash);


--
-- Name: __drizzle_migrations __drizzle_migrations_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.__drizzle_migrations
    ADD CONSTRAINT __drizzle_migrations_pkey PRIMARY KEY (id);


--
-- Name: __evekill_schema_migrations __evekill_schema_migrations_name_key; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.__evekill_schema_migrations
    ADD CONSTRAINT __evekill_schema_migrations_name_key UNIQUE (name);


--
-- Name: __evekill_schema_migrations __evekill_schema_migrations_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.__evekill_schema_migrations
    ADD CONSTRAINT __evekill_schema_migrations_pkey PRIMARY KEY (id);


--
-- Name: alliances alliances_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.alliances
    ADD CONSTRAINT alliances_pkey PRIMARY KEY (alliance_id);


--
-- Name: announcements announcements_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.announcements
    ADD CONSTRAINT announcements_pkey PRIMARY KEY (id);


--
-- Name: battle_team_members battle_team_members_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.battle_team_members
    ADD CONSTRAINT battle_team_members_pkey PRIMARY KEY (id);


--
-- Name: battle_teams battle_teams_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.battle_teams
    ADD CONSTRAINT battle_teams_pkey PRIMARY KEY (id);


--
-- Name: battles battles_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.battles
    ADD CONSTRAINT battles_pkey PRIMARY KEY (battle_id);


--
-- Name: blog_posts blog_posts_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.blog_posts
    ADD CONSTRAINT blog_posts_pkey PRIMARY KEY (id);


--
-- Name: bloodlines bloodlines_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.bloodlines
    ADD CONSTRAINT bloodlines_pkey PRIMARY KEY (bloodline_id);


--
-- Name: blueprint_activities blueprint_activities_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.blueprint_activities
    ADD CONSTRAINT blueprint_activities_pkey PRIMARY KEY (blueprint_type_id, activity);


--
-- Name: blueprint_activity_materials blueprint_activity_materials_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.blueprint_activity_materials
    ADD CONSTRAINT blueprint_activity_materials_pkey PRIMARY KEY (blueprint_type_id, activity, material_type_id);


--
-- Name: blueprint_activity_products blueprint_activity_products_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.blueprint_activity_products
    ADD CONSTRAINT blueprint_activity_products_pkey PRIMARY KEY (blueprint_type_id, activity, product_type_id);


--
-- Name: blueprint_activity_skills blueprint_activity_skills_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.blueprint_activity_skills
    ADD CONSTRAINT blueprint_activity_skills_pkey PRIMARY KEY (blueprint_type_id, activity, skill_type_id);


--
-- Name: blueprints blueprints_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.blueprints
    ADD CONSTRAINT blueprints_pkey PRIMARY KEY (blueprint_type_id);


--
-- Name: campaign_prize_pools campaign_prize_pools_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.campaign_prize_pools
    ADD CONSTRAINT campaign_prize_pools_pkey PRIMARY KEY (campaign_id);


--
-- Name: campaign_prize_results campaign_prize_results_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.campaign_prize_results
    ADD CONSTRAINT campaign_prize_results_pkey PRIMARY KEY (campaign_id, rank);


--
-- Name: campaign_scratch_candidates campaign_scratch_candidates_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.campaign_scratch_candidates
    ADD CONSTRAINT campaign_scratch_candidates_pkey PRIMARY KEY (campaign_id, killmail_id);


--
-- Name: campaign_scratch_killmails campaign_scratch_killmails_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.campaign_scratch_killmails
    ADD CONSTRAINT campaign_scratch_killmails_pkey PRIMARY KEY (campaign_id, killmail_id);


--
-- Name: campaign_side_entities campaign_side_entities_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.campaign_side_entities
    ADD CONSTRAINT campaign_side_entities_pkey PRIMARY KEY (id);


--
-- Name: campaign_sides campaign_sides_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.campaign_sides
    ADD CONSTRAINT campaign_sides_pkey PRIMARY KEY (campaign_id, side_index);


--
-- Name: campaign_stats_daily campaign_stats_daily_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.campaign_stats_daily
    ADD CONSTRAINT campaign_stats_daily_pkey PRIMARY KEY (campaign_id, side_index, day);


--
-- Name: campaigns campaigns_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.campaigns
    ADD CONSTRAINT campaigns_pkey PRIMARY KEY (campaign_id);


--
-- Name: celestials celestials_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.celestials
    ADD CONSTRAINT celestials_pkey PRIMARY KEY (item_id);


--
-- Name: character_corporation_history character_corporation_history_character_id_record_id_pk; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.character_corporation_history
    ADD CONSTRAINT character_corporation_history_character_id_record_id_pk PRIMARY KEY (character_id, record_id);


--
-- Name: characters characters_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.characters
    ADD CONSTRAINT characters_pkey PRIMARY KEY (character_id);


--
-- Name: comment_edits comment_edits_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.comment_edits
    ADD CONSTRAINT comment_edits_pkey PRIMARY KEY (id);


--
-- Name: comment_reports comment_reports_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.comment_reports
    ADD CONSTRAINT comment_reports_pkey PRIMARY KEY (id);


--
-- Name: comments comments_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.comments
    ADD CONSTRAINT comments_pkey PRIMARY KEY (id);


--
-- Name: config config_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.config
    ADD CONSTRAINT config_pkey PRIMARY KEY (key);


--
-- Name: constellations constellations_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.constellations
    ADD CONSTRAINT constellations_pkey PRIMARY KEY (constellation_id);


--
-- Name: corporation_alliance_history corporation_alliance_history_corporation_id_record_id_pk; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.corporation_alliance_history
    ADD CONSTRAINT corporation_alliance_history_corporation_id_record_id_pk PRIMARY KEY (corporation_id, record_id);


--
-- Name: corporation_wallet_balances corporation_wallet_balances_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.corporation_wallet_balances
    ADD CONSTRAINT corporation_wallet_balances_pkey PRIMARY KEY (corporation_id, division);


--
-- Name: corporation_wallet_journal corporation_wallet_journal_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.corporation_wallet_journal
    ADD CONSTRAINT corporation_wallet_journal_pkey PRIMARY KEY (corporation_id, division, journal_id);


--
-- Name: corporation_wallet_tokens corporation_wallet_tokens_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.corporation_wallet_tokens
    ADD CONSTRAINT corporation_wallet_tokens_pkey PRIMARY KEY (corporation_id);


--
-- Name: corporations corporations_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.corporations
    ADD CONSTRAINT corporations_pkey PRIMARY KEY (corporation_id);


--
-- Name: custom_domain_campaigns custom_domain_campaigns_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.custom_domain_campaigns
    ADD CONSTRAINT custom_domain_campaigns_pkey PRIMARY KEY (domain_id, campaign_id);


--
-- Name: custom_domains custom_domains_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.custom_domains
    ADD CONSTRAINT custom_domains_pkey PRIMARY KEY (id);


--
-- Name: custom_prices custom_prices_type_id_date_pk; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.custom_prices
    ADD CONSTRAINT custom_prices_type_id_date_pk PRIMARY KEY (type_id, date);


--
-- Name: domain_assets domain_assets_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.domain_assets
    ADD CONSTRAINT domain_assets_pkey PRIMARY KEY (id);


--
-- Name: ek_wallet_accounts ek_wallet_accounts_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.ek_wallet_accounts
    ADD CONSTRAINT ek_wallet_accounts_pkey PRIMARY KEY (character_id);


--
-- Name: ek_wallet_reservations ek_wallet_reservations_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.ek_wallet_reservations
    ADD CONSTRAINT ek_wallet_reservations_pkey PRIMARY KEY (id);


--
-- Name: ek_wallet_transactions ek_wallet_transactions_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.ek_wallet_transactions
    ADD CONSTRAINT ek_wallet_transactions_pkey PRIMARY KEY (id);


--
-- Name: entity_achievements entity_achievements_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.entity_achievements
    ADD CONSTRAINT entity_achievements_pkey PRIMARY KEY (entity_id, achievement_id);


--
-- Name: entity_snapshots entity_snapshots_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.entity_snapshots
    ADD CONSTRAINT entity_snapshots_pkey PRIMARY KEY (entity_type, entity_id, date);


--
-- Name: esi_killmail_delayed esi_killmail_delayed_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.esi_killmail_delayed
    ADD CONSTRAINT esi_killmail_delayed_pkey PRIMARY KEY (killmail_id);


--
-- Name: esi_request_logs esi_request_logs_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.esi_request_logs
    ADD CONSTRAINT esi_request_logs_pkey PRIMARY KEY (id);


--
-- Name: factions factions_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.factions
    ADD CONSTRAINT factions_pkey PRIMARY KEY (faction_id);


--
-- Name: feed_queue feed_queue_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.feed_queue
    ADD CONSTRAINT feed_queue_pkey PRIMARY KEY (seq);


--
-- Name: fitting_items fitting_items_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.fitting_items
    ADD CONSTRAINT fitting_items_pkey PRIMARY KEY (fit_hash, slot_group, ordinal);


--
-- Name: fittings fittings_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.fittings
    ADD CONSTRAINT fittings_pkey PRIMARY KEY (fit_hash);


--
-- Name: fw_faction_stats fw_faction_stats_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.fw_faction_stats
    ADD CONSTRAINT fw_faction_stats_pkey PRIMARY KEY (faction_id);


--
-- Name: fw_leaderboards fw_leaderboards_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.fw_leaderboards
    ADD CONSTRAINT fw_leaderboards_pkey PRIMARY KEY (entity_type, entity_id, metric, period);


--
-- Name: fw_system_history fw_system_history_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.fw_system_history
    ADD CONSTRAINT fw_system_history_pkey PRIMARY KEY (id);


--
-- Name: fw_systems fw_systems_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.fw_systems
    ADD CONSTRAINT fw_systems_pkey PRIMARY KEY (solar_system_id);


--
-- Name: insurance_prices insurance_prices_type_id_level_name_pk; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.insurance_prices
    ADD CONSTRAINT insurance_prices_type_id_level_name_pk PRIMARY KEY (type_id, level_name);


--
-- Name: inv_categories inv_categories_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.inv_categories
    ADD CONSTRAINT inv_categories_pkey PRIMARY KEY (category_id);


--
-- Name: inv_flags inv_flags_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.inv_flags
    ADD CONSTRAINT inv_flags_pkey PRIMARY KEY (flag_id);


--
-- Name: inv_groups inv_groups_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.inv_groups
    ADD CONSTRAINT inv_groups_pkey PRIMARY KEY (group_id);


--
-- Name: inv_market_groups inv_market_groups_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.inv_market_groups
    ADD CONSTRAINT inv_market_groups_pkey PRIMARY KEY (market_group_id);


--
-- Name: inv_meta_groups inv_meta_groups_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.inv_meta_groups
    ADD CONSTRAINT inv_meta_groups_pkey PRIMARY KEY (meta_group_id);


--
-- Name: inv_types inv_types_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.inv_types
    ADD CONSTRAINT inv_types_pkey PRIMARY KEY (type_id);


--
-- Name: killmail_attackers killmail_attackers_killmail_id_attacker_index_pk; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.killmail_attackers
    ADD CONSTRAINT killmail_attackers_killmail_id_attacker_index_pk PRIMARY KEY (killmail_id, attacker_index);


--
-- Name: killmail_fittings killmail_fittings_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.killmail_fittings
    ADD CONSTRAINT killmail_fittings_pkey PRIMARY KEY (killmail_id);


--
-- Name: killmail_items killmail_items_killmail_id_item_index_pk; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.killmail_items
    ADD CONSTRAINT killmail_items_killmail_id_item_index_pk PRIMARY KEY (killmail_id, item_index);


--
-- Name: killmail_processing killmail_processing_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.killmail_processing
    ADD CONSTRAINT killmail_processing_pkey PRIMARY KEY (killmail_id);


--
-- Name: killmails killmails_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.killmails
    ADD CONSTRAINT killmails_pkey PRIMARY KEY (killmail_id);


--
-- Name: kills_daily_count kills_daily_count_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.kills_daily_count
    ADD CONSTRAINT kills_daily_count_pkey PRIMARY KEY (date, type);


--
-- Name: moderation_queue moderation_queue_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.moderation_queue
    ADD CONSTRAINT moderation_queue_pkey PRIMARY KEY (id);


--
-- Name: npc_corporations npc_corporations_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.npc_corporations
    ADD CONSTRAINT npc_corporations_pkey PRIMARY KEY (corporation_id);


--
-- Name: old_killmail_attackers old_killmail_attackers_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.old_killmail_attackers
    ADD CONSTRAINT old_killmail_attackers_pkey PRIMARY KEY (id);


--
-- Name: old_killmail_items old_killmail_items_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.old_killmail_items
    ADD CONSTRAINT old_killmail_items_pkey PRIMARY KEY (id);


--
-- Name: old_killmails old_killmails_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.old_killmails
    ADD CONSTRAINT old_killmails_pkey PRIMARY KEY (killmail_id);


--
-- Name: prices prices_type_id_region_id_date_pk; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.prices
    ADD CONSTRAINT prices_type_id_region_id_date_pk PRIMARY KEY (type_id, region_id, date);


--
-- Name: races races_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.races
    ADD CONSTRAINT races_pkey PRIMARY KEY (race_id);


--
-- Name: regions regions_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.regions
    ADD CONSTRAINT regions_pkey PRIMARY KEY (region_id);


--
-- Name: scans scans_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.scans
    ADD CONSTRAINT scans_pkey PRIMARY KEY (hash);


--
-- Name: solar_system_jumps solar_system_jumps_from_solar_system_id_to_solar_system_id_pk; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.solar_system_jumps
    ADD CONSTRAINT solar_system_jumps_from_solar_system_id_to_solar_system_id_pk PRIMARY KEY (from_solar_system_id, to_solar_system_id);


--
-- Name: solar_systems solar_systems_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.solar_systems
    ADD CONSTRAINT solar_systems_pkey PRIMARY KEY (solar_system_id);


--
-- Name: sovereignty_history sovereignty_history_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.sovereignty_history
    ADD CONSTRAINT sovereignty_history_pkey PRIMARY KEY (id);


--
-- Name: sovereignty sovereignty_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.sovereignty
    ADD CONSTRAINT sovereignty_pkey PRIMARY KEY (system_id);


--
-- Name: station_operations station_operations_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.station_operations
    ADD CONSTRAINT station_operations_pkey PRIMARY KEY (operation_id);


--
-- Name: stations stations_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.stations
    ADD CONSTRAINT stations_pkey PRIMARY KEY (station_id);


--
-- Name: stats_breakdowns stats_breakdowns_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.stats_breakdowns
    ADD CONSTRAINT stats_breakdowns_pkey PRIMARY KEY (entity_type, entity_id, period_type, period_start, dim_category, dim_id);


--
-- Name: stats_leaderboards stats_leaderboards_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.stats_leaderboards
    ADD CONSTRAINT stats_leaderboards_pkey PRIMARY KEY (entity_type, period_type, period_start, metric, rank);


--
-- Name: stats stats_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.stats
    ADD CONSTRAINT stats_pkey PRIMARY KEY (entity_type, entity_id, period_type, period_start);


--
-- Name: structures structures_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.structures
    ADD CONSTRAINT structures_pkey PRIMARY KEY (structure_id);


--
-- Name: system_activity system_activity_system_id_timestamp_pk; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.system_activity
    ADD CONSTRAINT system_activity_system_id_timestamp_pk PRIMARY KEY (system_id, "timestamp");


--
-- Name: type_dogma_attributes type_dogma_attributes_type_id_attribute_id_pk; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.type_dogma_attributes
    ADD CONSTRAINT type_dogma_attributes_type_id_attribute_id_pk PRIMARY KEY (type_id, attribute_id);


--
-- Name: type_dogma_effects type_dogma_effects_type_id_effect_id_pk; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.type_dogma_effects
    ADD CONSTRAINT type_dogma_effects_type_id_effect_id_pk PRIMARY KEY (type_id, effect_id);


--
-- Name: type_materials type_materials_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.type_materials
    ADD CONSTRAINT type_materials_pkey PRIMARY KEY (type_id, material_type_id);


--
-- Name: user_config user_config_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.user_config
    ADD CONSTRAINT user_config_pkey PRIMARY KEY (character_id, key);


--
-- Name: user_esi_tokens user_esi_tokens_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.user_esi_tokens
    ADD CONSTRAINT user_esi_tokens_pkey PRIMARY KEY (id);


--
-- Name: user_fitting_items user_fitting_items_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.user_fitting_items
    ADD CONSTRAINT user_fitting_items_pkey PRIMARY KEY (fit_id, slot_group, ordinal);


--
-- Name: user_fitting_ratings user_fitting_ratings_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.user_fitting_ratings
    ADD CONSTRAINT user_fitting_ratings_pkey PRIMARY KEY (fit_id, rater_character_id);


--
-- Name: user_fittings user_fittings_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.user_fittings
    ADD CONSTRAINT user_fittings_pkey PRIMARY KEY (fit_id);


--
-- Name: user_sessions user_sessions_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.user_sessions
    ADD CONSTRAINT user_sessions_pkey PRIMARY KEY (id);


--
-- Name: users users_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.users
    ADD CONSTRAINT users_pkey PRIMARY KEY (character_id);


--
-- Name: wallet_journal_references wallet_journal_references_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.wallet_journal_references
    ADD CONSTRAINT wallet_journal_references_pkey PRIMARY KEY (corporation_id, division, journal_id);


--
-- Name: war_allies war_allies_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.war_allies
    ADD CONSTRAINT war_allies_pkey PRIMARY KEY (id);


--
-- Name: war_interactions war_interactions_war_id_side_category_target_type_target_id_pk; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.war_interactions
    ADD CONSTRAINT war_interactions_war_id_side_category_target_type_target_id_pk PRIMARY KEY (war_id, side, category, target_type, target_id);


--
-- Name: wars wars_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.wars
    ADD CONSTRAINT wars_pkey PRIMARY KEY (war_id);


--
-- Name: alliances_executor_corp_idx; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX alliances_executor_corp_idx ON public.alliances USING btree (executor_corporation_id);


--
-- Name: alliances_faction_idx; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX alliances_faction_idx ON public.alliances USING btree (faction_id);


--
-- Name: alliances_name_hash; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX alliances_name_hash ON public.alliances USING hash (name);


--
-- Name: alliances_name_trgm; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX alliances_name_trgm ON public.alliances USING gin (name gin_trgm_ops);


--
-- Name: alliances_ticker_trgm; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX alliances_ticker_trgm ON public.alliances USING gin (ticker gin_trgm_ops);


--
-- Name: alliances_updated_at_idx; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX alliances_updated_at_idx ON public.alliances USING btree (updated_at);


--
-- Name: announcement_dismissals_pk; Type: INDEX; Schema: public; Owner: -
--

CREATE UNIQUE INDEX announcement_dismissals_pk ON public.announcement_dismissals USING btree (character_id, announcement_id);


--
-- Name: announcements_active_idx; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX announcements_active_idx ON public.announcements USING btree (starts_at, expires_at) WHERE (archived_at IS NULL);


--
-- Name: announcements_tier_idx; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX announcements_tier_idx ON public.announcements USING btree (tier, created_at DESC);


--
-- Name: battle_team_members_corp_idx; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX battle_team_members_corp_idx ON public.battle_team_members USING btree (corporation_id);


--
-- Name: battle_team_members_team_idx; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX battle_team_members_team_idx ON public.battle_team_members USING btree (battle_team_id);


--
-- Name: battle_teams_battle_idx; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX battle_teams_battle_idx ON public.battle_teams USING btree (battle_id);


--
-- Name: battles_region_idx; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX battles_region_idx ON public.battles USING btree (region_id);


--
-- Name: battles_solar_system_idx; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX battles_solar_system_idx ON public.battles USING btree (solar_system_id);


--
-- Name: battles_start_time_brin; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX battles_start_time_brin ON public.battles USING brin (start_time);


--
-- Name: battles_system_time_idx; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX battles_system_time_idx ON public.battles USING btree (solar_system_id, start_time);


--
-- Name: blog_posts_admin_idx; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX blog_posts_admin_idx ON public.blog_posts USING btree (status, created_at DESC);


--
-- Name: blog_posts_public_feed_idx; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX blog_posts_public_feed_idx ON public.blog_posts USING btree (published_at DESC) WHERE (status = 1);


--
-- Name: blog_posts_slug_uidx; Type: INDEX; Schema: public; Owner: -
--

CREATE UNIQUE INDEX blog_posts_slug_uidx ON public.blog_posts USING btree (slug);


--
-- Name: blog_posts_tags_gin; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX blog_posts_tags_gin ON public.blog_posts USING gin (tags);


--
-- Name: bloodlines_race_id_idx; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX bloodlines_race_id_idx ON public.bloodlines USING btree (race_id);


--
-- Name: blueprint_activity_products_product_idx; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX blueprint_activity_products_product_idx ON public.blueprint_activity_products USING btree (product_type_id, activity);


--
-- Name: campaign_prize_pools_status_idx; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX campaign_prize_pools_status_idx ON public.campaign_prize_pools USING btree (status, finalized_at);


--
-- Name: campaign_prize_results_character_unique; Type: INDEX; Schema: public; Owner: -
--

CREATE UNIQUE INDEX campaign_prize_results_character_unique ON public.campaign_prize_results USING btree (campaign_id, character_id);


--
-- Name: campaign_prize_results_claim_idx; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX campaign_prize_results_claim_idx ON public.campaign_prize_results USING btree (character_id, claimed_at);


--
-- Name: campaign_side_entities_entity_idx; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX campaign_side_entities_entity_idx ON public.campaign_side_entities USING btree (entity_type, entity_id);


--
-- Name: campaign_side_entities_unique; Type: INDEX; Schema: public; Owner: -
--

CREATE UNIQUE INDEX campaign_side_entities_unique ON public.campaign_side_entities USING btree (campaign_id, entity_type, entity_id);


--
-- Name: campaigns_activity_idx; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX campaigns_activity_idx ON public.campaigns USING btree (last_activity_at DESC NULLS LAST);


--
-- Name: campaigns_creator_idx; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX campaigns_creator_idx ON public.campaigns USING btree (created_by_character_id);


--
-- Name: campaigns_processing_idx; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX campaigns_processing_idx ON public.campaigns USING btree (status, processing_paused);


--
-- Name: campaigns_status_idx; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX campaigns_status_idx ON public.campaigns USING btree (status);


--
-- Name: celestials_constellation_id_idx; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX celestials_constellation_id_idx ON public.celestials USING btree (constellation_id);


--
-- Name: celestials_group_constellation_idx; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX celestials_group_constellation_idx ON public.celestials USING btree (group_id, constellation_id);


--
-- Name: celestials_group_id_idx; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX celestials_group_id_idx ON public.celestials USING btree (group_id);


--
-- Name: celestials_group_region_idx; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX celestials_group_region_idx ON public.celestials USING btree (group_id, region_id);


--
-- Name: celestials_group_system_idx; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX celestials_group_system_idx ON public.celestials USING btree (group_id, solar_system_id);


--
-- Name: celestials_orbit_id_idx; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX celestials_orbit_id_idx ON public.celestials USING btree (orbit_id);


--
-- Name: celestials_region_id_idx; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX celestials_region_id_idx ON public.celestials USING btree (region_id);


--
-- Name: celestials_solar_system_id_idx; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX celestials_solar_system_id_idx ON public.celestials USING btree (solar_system_id);


--
-- Name: celestials_type_id_idx; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX celestials_type_id_idx ON public.celestials USING btree (type_id);


--
-- Name: char_corp_history_char_idx; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX char_corp_history_char_idx ON public.character_corporation_history USING btree (character_id);


--
-- Name: char_corp_history_corp_recent_idx; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX char_corp_history_corp_recent_idx ON public.character_corporation_history USING btree (corporation_id, start_date DESC);


--
-- Name: characters_alliance_idx; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX characters_alliance_idx ON public.characters USING btree (alliance_id);


--
-- Name: characters_corporation_idx; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX characters_corporation_idx ON public.characters USING btree (corporation_id);


--
-- Name: characters_last_active_idx; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX characters_last_active_idx ON public.characters USING btree (last_active);


--
-- Name: characters_name_hash; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX characters_name_hash ON public.characters USING hash (name);


--
-- Name: characters_name_trgm; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX characters_name_trgm ON public.characters USING gin (name gin_trgm_ops);


--
-- Name: characters_updated_at_idx; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX characters_updated_at_idx ON public.characters USING btree (updated_at);


--
-- Name: comment_edits_comment_idx; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX comment_edits_comment_idx ON public.comment_edits USING btree (comment_id, edited_at DESC);


--
-- Name: comment_reports_queue_idx; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX comment_reports_queue_idx ON public.comment_reports USING btree (resolved_at, created_at DESC);


--
-- Name: comment_reports_unique; Type: INDEX; Schema: public; Owner: -
--

CREATE UNIQUE INDEX comment_reports_unique ON public.comment_reports USING btree (comment_id, reporter_id);


--
-- Name: comments_body_trgm_idx; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX comments_body_trgm_idx ON public.comments USING gin (body_md gin_trgm_ops) WHERE (deleted_at IS NULL);


--
-- Name: comments_by_character_idx; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX comments_by_character_idx ON public.comments USING btree (character_id, created_at DESC) WHERE (deleted_at IS NULL);


--
-- Name: comments_domain_target_idx; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX comments_domain_target_idx ON public.comments USING btree (domain_id, target_type, target_id, created_at DESC) WHERE ((deleted_at IS NULL) AND (domain_id IS NOT NULL));


--
-- Name: comments_flagged_idx; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX comments_flagged_idx ON public.comments USING btree (flagged, created_at DESC) WHERE ((deleted_at IS NULL) AND (flagged = true));


--
-- Name: comments_global_feed_idx; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX comments_global_feed_idx ON public.comments USING btree (created_at DESC) WHERE ((deleted_at IS NULL) AND (domain_id IS NULL) AND (parent_id IS NULL));


--
-- Name: comments_root_idx; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX comments_root_idx ON public.comments USING btree (root_id, created_at) WHERE (deleted_at IS NULL);


--
-- Name: comments_target_active_idx; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX comments_target_active_idx ON public.comments USING btree (target_type, target_id, created_at DESC) WHERE (deleted_at IS NULL);


--
-- Name: comments_visible_target_idx; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX comments_visible_target_idx ON public.comments USING btree (target_type, target_id, created_at DESC) WHERE ((deleted_at IS NULL) AND (visibility = 0));


--
-- Name: constellations_region_id_idx; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX constellations_region_id_idx ON public.constellations USING btree (region_id);


--
-- Name: corp_alliance_history_corp_idx; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX corp_alliance_history_corp_idx ON public.corporation_alliance_history USING btree (corporation_id);


--
-- Name: corporation_wallet_balances_corporation_idx; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX corporation_wallet_balances_corporation_idx ON public.corporation_wallet_balances USING btree (corporation_id);


--
-- Name: corporation_wallet_journal_corporation_date_idx; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX corporation_wallet_journal_corporation_date_idx ON public.corporation_wallet_journal USING btree (corporation_id, date, journal_id);


--
-- Name: corporation_wallet_journal_first_party_idx; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX corporation_wallet_journal_first_party_idx ON public.corporation_wallet_journal USING btree (first_party_id);


--
-- Name: corporation_wallet_journal_second_party_idx; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX corporation_wallet_journal_second_party_idx ON public.corporation_wallet_journal USING btree (second_party_id);


--
-- Name: corporation_wallet_tokens_character_idx; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX corporation_wallet_tokens_character_idx ON public.corporation_wallet_tokens USING btree (authorized_character_id);


--
-- Name: corporations_alliance_idx; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX corporations_alliance_idx ON public.corporations USING btree (alliance_id);


--
-- Name: corporations_ceo_idx; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX corporations_ceo_idx ON public.corporations USING btree (ceo_id);


--
-- Name: corporations_faction_idx; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX corporations_faction_idx ON public.corporations USING btree (faction_id);


--
-- Name: corporations_name_hash; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX corporations_name_hash ON public.corporations USING hash (name);


--
-- Name: corporations_name_trgm; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX corporations_name_trgm ON public.corporations USING gin (name gin_trgm_ops);


--
-- Name: corporations_ticker_trgm; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX corporations_ticker_trgm ON public.corporations USING gin (ticker gin_trgm_ops);


--
-- Name: corporations_updated_at_idx; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX corporations_updated_at_idx ON public.corporations USING btree (updated_at);


--
-- Name: corporations_war_eligible_members_idx; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX corporations_war_eligible_members_idx ON public.corporations USING btree (member_count DESC NULLS LAST) WHERE (war_eligible AND (corporation_id > 1999999) AND (deleted IS NOT TRUE));


--
-- Name: custom_domain_campaigns_campaign_idx; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX custom_domain_campaigns_campaign_idx ON public.custom_domain_campaigns USING btree (campaign_id);


--
-- Name: ek_wallet_reservations_active_expiry_idx; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX ek_wallet_reservations_active_expiry_idx ON public.ek_wallet_reservations USING btree (expires_at, id) WHERE (status = 0);


--
-- Name: ek_wallet_reservations_campaign_idx; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX ek_wallet_reservations_campaign_idx ON public.ek_wallet_reservations USING btree (campaign_id);


--
-- Name: ek_wallet_reservations_captured_transaction_unique; Type: INDEX; Schema: public; Owner: -
--

CREATE UNIQUE INDEX ek_wallet_reservations_captured_transaction_unique ON public.ek_wallet_reservations USING btree (captured_transaction_id) WHERE (captured_transaction_id IS NOT NULL);


--
-- Name: ek_wallet_reservations_character_created_idx; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX ek_wallet_reservations_character_created_idx ON public.ek_wallet_reservations USING btree (character_id, created_at DESC, id DESC);


--
-- Name: ek_wallet_reservations_external_reference_unique; Type: INDEX; Schema: public; Owner: -
--

CREATE UNIQUE INDEX ek_wallet_reservations_external_reference_unique ON public.ek_wallet_reservations USING btree (external_reference);


--
-- Name: ek_wallet_transactions_campaign_idx; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX ek_wallet_transactions_campaign_idx ON public.ek_wallet_transactions USING btree (campaign_id);


--
-- Name: ek_wallet_transactions_character_created_idx; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX ek_wallet_transactions_character_created_idx ON public.ek_wallet_transactions USING btree (character_id, created_at DESC, id DESC);


--
-- Name: ek_wallet_transactions_external_reference_unique; Type: INDEX; Schema: public; Owner: -
--

CREATE UNIQUE INDEX ek_wallet_transactions_external_reference_unique ON public.ek_wallet_transactions USING btree (external_reference);


--
-- Name: ek_wallet_transactions_journal_unique; Type: INDEX; Schema: public; Owner: -
--

CREATE UNIQUE INDEX ek_wallet_transactions_journal_unique ON public.ek_wallet_transactions USING btree (corporation_id, division, journal_id) WHERE (journal_id IS NOT NULL);


--
-- Name: ek_wallet_transactions_related_idx; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX ek_wallet_transactions_related_idx ON public.ek_wallet_transactions USING btree (related_transaction_id);


--
-- Name: entity_snapshots_entity_idx; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX entity_snapshots_entity_idx ON public.entity_snapshots USING btree (entity_id, entity_type);


--
-- Name: entity_snapshots_type_date_idx; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX entity_snapshots_type_date_idx ON public.entity_snapshots USING btree (entity_type, date);


--
-- Name: esi_request_logs_character_created_idx; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX esi_request_logs_character_created_idx ON public.esi_request_logs USING btree (character_id, created_at);


--
-- Name: esi_request_logs_source_created_idx; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX esi_request_logs_source_created_idx ON public.esi_request_logs USING btree (source, created_at);


--
-- Name: esi_request_logs_success_created_idx; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX esi_request_logs_success_created_idx ON public.esi_request_logs USING btree (success, created_at);


--
-- Name: factions_name_trgm; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX factions_name_trgm ON public.factions USING gin (name gin_trgm_ops);


--
-- Name: fittings_family_idx; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX fittings_family_idx ON public.fittings USING btree (family_hash);


--
-- Name: fittings_ship_family_idx; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX fittings_ship_family_idx ON public.fittings USING btree (ship_type_id, family_hash);


--
-- Name: fw_leaderboards_type_metric_period_idx; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX fw_leaderboards_type_metric_period_idx ON public.fw_leaderboards USING btree (entity_type, metric, period);


--
-- Name: fw_system_history_flipped_at_idx; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX fw_system_history_flipped_at_idx ON public.fw_system_history USING btree (flipped_at);


--
-- Name: fw_system_history_system_idx; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX fw_system_history_system_idx ON public.fw_system_history USING btree (solar_system_id);


--
-- Name: fw_systems_contested_idx; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX fw_systems_contested_idx ON public.fw_systems USING btree (contested);


--
-- Name: fw_systems_occupier_faction_id_idx; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX fw_systems_occupier_faction_id_idx ON public.fw_systems USING btree (occupier_faction_id);


--
-- Name: idx_att_alliance_time; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_att_alliance_time ON public.killmail_attackers USING btree (alliance_id, killmail_time DESC) WHERE (alliance_id IS NOT NULL);


--
-- Name: idx_att_character_time; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_att_character_time ON public.killmail_attackers USING btree (character_id, killmail_time DESC) WHERE (character_id IS NOT NULL);


--
-- Name: idx_att_corporation_time; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_att_corporation_time ON public.killmail_attackers USING btree (corporation_id, killmail_time DESC) WHERE (corporation_id IS NOT NULL);


--
-- Name: idx_att_final_blow; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_att_final_blow ON public.killmail_attackers USING btree (killmail_id) WHERE (final_blow = true);


--
-- Name: idx_att_killmail_time; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_att_killmail_time ON public.killmail_attackers USING btree (killmail_time);


--
-- Name: idx_battle_team_members_alliance; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_battle_team_members_alliance ON public.battle_team_members USING btree (alliance_id) WHERE (alliance_id IS NOT NULL);


--
-- Name: idx_battles_kill_count; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_battles_kill_count ON public.battles USING btree (kill_count DESC NULLS LAST);


--
-- Name: idx_battles_start_time_btree; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_battles_start_time_btree ON public.battles USING btree (start_time DESC);


--
-- Name: idx_battles_total_isk_destroyed; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_battles_total_isk_destroyed ON public.battles USING btree (total_isk_destroyed DESC NULLS LAST);


--
-- Name: idx_cd_custom_hostname_active; Type: INDEX; Schema: public; Owner: -
--

CREATE UNIQUE INDEX idx_cd_custom_hostname_active ON public.custom_domains USING btree (custom_hostname) WHERE ((custom_hostname IS NOT NULL) AND (active = true));


--
-- Name: idx_cd_subdomain_active; Type: INDEX; Schema: public; Owner: -
--

CREATE UNIQUE INDEX idx_cd_subdomain_active ON public.custom_domains USING btree (subdomain) WHERE (active = true);


--
-- Name: idx_cd_user; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_cd_user ON public.custom_domains USING btree (user_id);


--
-- Name: idx_char_ach_pts; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_char_ach_pts ON public.characters USING btree (achievement_points DESC) WHERE (achievement_points > 0);


--
-- Name: idx_char_alliance_ach_pts; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_char_alliance_ach_pts ON public.characters USING btree (alliance_id, achievement_points DESC) WHERE ((alliance_id IS NOT NULL) AND (achievement_points > 0));


--
-- Name: idx_char_corp_ach_pts; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_char_corp_ach_pts ON public.characters USING btree (corporation_id, achievement_points DESC) WHERE ((corporation_id IS NOT NULL) AND (achievement_points > 0));


--
-- Name: idx_da_domain_type; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_da_domain_type ON public.domain_assets USING btree (domain_id, type);


--
-- Name: idx_da_status; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_da_status ON public.domain_assets USING btree (status);


--
-- Name: idx_esi_killmail_delayed_until; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_esi_killmail_delayed_until ON public.esi_killmail_delayed USING btree (delayed_until);


--
-- Name: idx_feed_queue_inserted_at; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_feed_queue_inserted_at ON public.feed_queue USING btree (inserted_at);


--
-- Name: idx_item_type; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_item_type ON public.killmail_items USING btree (type_id, killmail_id DESC);


--
-- Name: idx_item_type_fitted; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_item_type_fitted ON public.killmail_items USING btree (type_id, killmail_id DESC) WHERE ((parent_index IS NULL) AND (((flag_id >= 11) AND (flag_id <= 34)) OR ((flag_id >= 92) AND (flag_id <= 99)) OR ((flag_id >= 125) AND (flag_id <= 132)) OR (flag_id = 87)));


--
-- Name: idx_kills_daily_count_type_date; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_kills_daily_count_type_date ON public.kills_daily_count USING btree (type, date DESC);


--
-- Name: idx_km_blob; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_km_blob ON public.killmails USING btree (blob, killmail_id DESC) WHERE (blob = true);


--
-- Name: idx_km_constellation; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_km_constellation ON public.killmails USING btree (constellation_id, killmail_id) WHERE (constellation_id IS NOT NULL);


--
-- Name: idx_km_killmail_time; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_km_killmail_time ON public.killmails USING btree (killmail_time DESC);


--
-- Name: idx_km_npc_id; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_km_npc_id ON public.killmails USING btree (killmail_id DESC) WHERE (is_npc = true);


--
-- Name: idx_km_npc_value; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_km_npc_value ON public.killmails USING btree (total_value DESC, killmail_id DESC) WHERE (is_npc = true);


--
-- Name: idx_km_region; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_km_region ON public.killmails USING btree (region_id, killmail_id DESC) WHERE (region_id IS NOT NULL);


--
-- Name: idx_km_solar_system; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_km_solar_system ON public.killmails USING btree (solar_system_id, killmail_id DESC);


--
-- Name: idx_km_system_time; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_km_system_time ON public.killmails USING btree (solar_system_id, killmail_time);


--
-- Name: idx_km_total_value; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_km_total_value ON public.killmails USING btree (total_value DESC, killmail_id DESC);


--
-- Name: idx_km_victim_alliance; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_km_victim_alliance ON public.killmails USING btree (victim_alliance_id, killmail_id DESC) WHERE (victim_alliance_id IS NOT NULL);


--
-- Name: idx_km_victim_char_system_time; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_km_victim_char_system_time ON public.killmails USING btree (victim_character_id, solar_system_id, killmail_time) WHERE (victim_character_id IS NOT NULL);


--
-- Name: idx_km_victim_character; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_km_victim_character ON public.killmails USING btree (victim_character_id, killmail_id DESC) WHERE (victim_character_id IS NOT NULL);


--
-- Name: idx_km_victim_corporation; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_km_victim_corporation ON public.killmails USING btree (victim_corporation_id, killmail_id DESC) WHERE (victim_corporation_id IS NOT NULL);


--
-- Name: idx_km_victim_faction; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_km_victim_faction ON public.killmails USING btree (victim_faction_id, killmail_id DESC) WHERE (victim_faction_id IS NOT NULL);


--
-- Name: idx_km_victim_ship_group; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_km_victim_ship_group ON public.killmails USING btree (victim_ship_group_id, killmail_id DESC) WHERE (victim_ship_group_id IS NOT NULL);


--
-- Name: idx_km_victim_ship_type; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_km_victim_ship_type ON public.killmails USING btree (victim_ship_type_id, killmail_id) WHERE (victim_ship_type_id IS NOT NULL);


--
-- Name: idx_km_war; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_km_war ON public.killmails USING btree (war_id, killmail_id DESC) WHERE (war_id IS NOT NULL);


--
-- Name: idx_system_activity_system; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_system_activity_system ON public.system_activity USING btree (system_id, "timestamp");


--
-- Name: idx_system_activity_timestamp; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_system_activity_timestamp ON public.system_activity USING btree ("timestamp");


--
-- Name: inv_groups_category_id_idx; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX inv_groups_category_id_idx ON public.inv_groups USING btree (category_id);


--
-- Name: inv_market_groups_parent_group_id_idx; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX inv_market_groups_parent_group_id_idx ON public.inv_market_groups USING btree (parent_group_id);


--
-- Name: inv_types_category_id_idx; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX inv_types_category_id_idx ON public.inv_types USING btree (category_id);


--
-- Name: inv_types_group_id_idx; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX inv_types_group_id_idx ON public.inv_types USING btree (group_id);


--
-- Name: inv_types_market_group_id_idx; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX inv_types_market_group_id_idx ON public.inv_types USING btree (market_group_id);


--
-- Name: inv_types_name_trgm; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX inv_types_name_trgm ON public.inv_types USING gin (name gin_trgm_ops);


--
-- Name: inv_types_published_idx; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX inv_types_published_idx ON public.inv_types USING btree (published);


--
-- Name: inv_types_variation_parent_idx; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX inv_types_variation_parent_idx ON public.inv_types USING btree (variation_parent_type_id) WHERE (variation_parent_type_id IS NOT NULL);


--
-- Name: killmail_fittings_fit_idx; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX killmail_fittings_fit_idx ON public.killmail_fittings USING btree (fit_hash);


--
-- Name: killmail_fittings_ship_alliance_time_idx; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX killmail_fittings_ship_alliance_time_idx ON public.killmail_fittings USING btree (ship_type_id, victim_alliance_id, kill_time DESC) WHERE (victim_alliance_id IS NOT NULL);


--
-- Name: killmail_fittings_ship_time_idx; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX killmail_fittings_ship_time_idx ON public.killmail_fittings USING btree (ship_type_id, kill_time DESC);


--
-- Name: moderation_queue_pending_idx; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX moderation_queue_pending_idx ON public.moderation_queue USING btree (submitted_at DESC) WHERE (status = 0);


--
-- Name: moderation_queue_pending_unique; Type: INDEX; Schema: public; Owner: -
--

CREATE UNIQUE INDEX moderation_queue_pending_unique ON public.moderation_queue USING btree (target_kind, target_id) WHERE (status = 0);


--
-- Name: moderation_queue_submitter_idx; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX moderation_queue_submitter_idx ON public.moderation_queue USING btree (character_id, submitted_at DESC);


--
-- Name: moderation_queue_target_idx; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX moderation_queue_target_idx ON public.moderation_queue USING btree (target_kind, target_id);


--
-- Name: old_killmail_attackers_killmail_id_idx; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX old_killmail_attackers_killmail_id_idx ON public.old_killmail_attackers USING btree (killmail_id);


--
-- Name: old_killmail_attackers_name_idx; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX old_killmail_attackers_name_idx ON public.old_killmail_attackers USING btree (name);


--
-- Name: old_killmail_attackers_name_trgm; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX old_killmail_attackers_name_trgm ON public.old_killmail_attackers USING gin (name gin_trgm_ops);


--
-- Name: old_killmail_items_killmail_id_idx; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX old_killmail_items_killmail_id_idx ON public.old_killmail_items USING btree (killmail_id);


--
-- Name: old_killmails_killmail_time_idx; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX old_killmails_killmail_time_idx ON public.old_killmails USING btree (killmail_time);


--
-- Name: old_killmails_system_name_idx; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX old_killmails_system_name_idx ON public.old_killmails USING btree (system_name);


--
-- Name: old_killmails_system_name_trgm; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX old_killmails_system_name_trgm ON public.old_killmails USING gin (system_name gin_trgm_ops);


--
-- Name: old_killmails_victim_alliance_trgm; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX old_killmails_victim_alliance_trgm ON public.old_killmails USING gin (victim_alliance gin_trgm_ops);


--
-- Name: old_killmails_victim_corp_trgm; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX old_killmails_victim_corp_trgm ON public.old_killmails USING gin (victim_corp gin_trgm_ops);


--
-- Name: old_killmails_victim_name_idx; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX old_killmails_victim_name_idx ON public.old_killmails USING btree (victim_name);


--
-- Name: old_killmails_victim_name_trgm; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX old_killmails_victim_name_trgm ON public.old_killmails USING gin (victim_name gin_trgm_ops);


--
-- Name: old_killmails_victim_ship_idx; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX old_killmails_victim_ship_idx ON public.old_killmails USING btree (victim_ship);


--
-- Name: prices_date_idx; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX prices_date_idx ON public.prices USING btree (date);


--
-- Name: prices_type_date_idx; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX prices_type_date_idx ON public.prices USING btree (type_id, date);


--
-- Name: regions_faction_id_idx; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX regions_faction_id_idx ON public.regions USING btree (faction_id);


--
-- Name: regions_name_trgm; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX regions_name_trgm ON public.regions USING gin (name gin_trgm_ops);


--
-- Name: scans_character_idx; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX scans_character_idx ON public.scans USING btree (character_id);


--
-- Name: scans_type_created_idx; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX scans_type_created_idx ON public.scans USING btree (scan_type, created_at);


--
-- Name: solar_system_jumps_from_idx; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX solar_system_jumps_from_idx ON public.solar_system_jumps USING btree (from_solar_system_id);


--
-- Name: solar_system_jumps_to_idx; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX solar_system_jumps_to_idx ON public.solar_system_jumps USING btree (to_solar_system_id);


--
-- Name: solar_systems_constellation_id_idx; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX solar_systems_constellation_id_idx ON public.solar_systems USING btree (constellation_id);


--
-- Name: solar_systems_faction_id_idx; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX solar_systems_faction_id_idx ON public.solar_systems USING btree (faction_id);


--
-- Name: solar_systems_name_trgm; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX solar_systems_name_trgm ON public.solar_systems USING gin (system_name gin_trgm_ops);


--
-- Name: solar_systems_region_id_idx; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX solar_systems_region_id_idx ON public.solar_systems USING btree (region_id);


--
-- Name: solar_systems_security_idx; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX solar_systems_security_idx ON public.solar_systems USING btree (security);


--
-- Name: sov_history_alliance_idx; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX sov_history_alliance_idx ON public.sovereignty_history USING btree (alliance_id);


--
-- Name: sov_history_system_idx; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX sov_history_system_idx ON public.sovereignty_history USING btree (system_id);


--
-- Name: sovereignty_alliance_idx; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX sovereignty_alliance_idx ON public.sovereignty USING btree (alliance_id);


--
-- Name: sovereignty_corporation_idx; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX sovereignty_corporation_idx ON public.sovereignty USING btree (corporation_id);


--
-- Name: sovereignty_faction_idx; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX sovereignty_faction_idx ON public.sovereignty USING btree (faction_id);


--
-- Name: stations_corporation_id_idx; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX stations_corporation_id_idx ON public.stations USING btree (corporation_id);


--
-- Name: stations_region_id_idx; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX stations_region_id_idx ON public.stations USING btree (region_id);


--
-- Name: stations_solar_system_id_idx; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX stations_solar_system_id_idx ON public.stations USING btree (solar_system_id);


--
-- Name: stats_breakdowns_entity_category_idx; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX stats_breakdowns_entity_category_idx ON public.stats_breakdowns USING btree (entity_type, entity_id, dim_category, period_type, period_start);


--
-- Name: stats_breakdowns_period_brin; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX stats_breakdowns_period_brin ON public.stats_breakdowns USING brin (period_start, period_type);


--
-- Name: stats_leaderboard_kills_idx; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX stats_leaderboard_kills_idx ON public.stats USING btree (entity_type, period_type, period_start, kills);


--
-- Name: stats_period_start_brin; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX stats_period_start_brin ON public.stats USING brin (period_start, period_type);


--
-- Name: structures_owner_id_idx; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX structures_owner_id_idx ON public.structures USING btree (owner_id);


--
-- Name: structures_region_id_idx; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX structures_region_id_idx ON public.structures USING btree (region_id);


--
-- Name: structures_solar_system_id_idx; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX structures_solar_system_id_idx ON public.structures USING btree (solar_system_id);


--
-- Name: structures_type_id_idx; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX structures_type_id_idx ON public.structures USING btree (type_id);


--
-- Name: type_dogma_attrs_attribute_idx; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX type_dogma_attrs_attribute_idx ON public.type_dogma_attributes USING btree (attribute_id);


--
-- Name: type_dogma_effects_effect_idx; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX type_dogma_effects_effect_idx ON public.type_dogma_effects USING btree (effect_id);


--
-- Name: user_esi_tokens_character_id_idx; Type: INDEX; Schema: public; Owner: -
--

CREATE UNIQUE INDEX user_esi_tokens_character_id_idx ON public.user_esi_tokens USING btree (character_id);


--
-- Name: user_fitting_ratings_rater_idx; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX user_fitting_ratings_rater_idx ON public.user_fitting_ratings USING btree (rater_character_id, updated_at DESC);


--
-- Name: user_fittings_owner_idx; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX user_fittings_owner_idx ON public.user_fittings USING btree (owner_character_id);


--
-- Name: user_fittings_owner_ship_idx; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX user_fittings_owner_ship_idx ON public.user_fittings USING btree (owner_character_id, ship_type_id, created_at DESC);


--
-- Name: user_fittings_public_ship_idx; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX user_fittings_public_ship_idx ON public.user_fittings USING btree (ship_type_id, created_at DESC) WHERE (visibility = 3);


--
-- Name: user_fittings_top_rated_idx; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX user_fittings_top_rated_idx ON public.user_fittings USING btree (rating_avg DESC, rating_count DESC, created_at DESC) WHERE ((visibility = 3) AND (rating_count > 0));


--
-- Name: user_sessions_character_id_idx; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX user_sessions_character_id_idx ON public.user_sessions USING btree (character_id);


--
-- Name: user_sessions_expires_at_idx; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX user_sessions_expires_at_idx ON public.user_sessions USING btree (expires_at);


--
-- Name: user_sessions_session_id_idx; Type: INDEX; Schema: public; Owner: -
--

CREATE UNIQUE INDEX user_sessions_session_id_idx ON public.user_sessions USING btree (session_id);


--
-- Name: users_discord_user_id_idx; Type: INDEX; Schema: public; Owner: -
--

CREATE UNIQUE INDEX users_discord_user_id_idx ON public.users USING btree (discord_user_id) WHERE (discord_user_id IS NOT NULL);


--
-- Name: users_session_id_idx; Type: INDEX; Schema: public; Owner: -
--

CREATE UNIQUE INDEX users_session_id_idx ON public.users USING btree (session_id);


--
-- Name: wallet_journal_references_lookup_idx; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX wallet_journal_references_lookup_idx ON public.wallet_journal_references USING btree (reference_type, reference_id, status);


--
-- Name: war_allies_war_idx; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX war_allies_war_idx ON public.war_allies USING btree (war_id);


--
-- Name: war_interact_lookup_idx; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX war_interact_lookup_idx ON public.war_interactions USING btree (war_id, side, category, target_type);


--
-- Name: wars_aggressor_alliance_idx; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX wars_aggressor_alliance_idx ON public.wars USING btree (aggressor_alliance_id);


--
-- Name: wars_aggressor_corp_idx; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX wars_aggressor_corp_idx ON public.wars USING btree (aggressor_corporation_id);


--
-- Name: wars_defender_alliance_idx; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX wars_defender_alliance_idx ON public.wars USING btree (defender_alliance_id);


--
-- Name: wars_defender_corp_idx; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX wars_defender_corp_idx ON public.wars USING btree (defender_corporation_id);


--
-- Name: wars_finished_idx; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX wars_finished_idx ON public.wars USING btree (finished);


--
-- Name: wars_started_idx; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX wars_started_idx ON public.wars USING btree (started);


--
-- Name: ek_wallet_reservations ek_wallet_reservations_lifecycle; Type: TRIGGER; Schema: public; Owner: -
--

CREATE TRIGGER ek_wallet_reservations_lifecycle BEFORE DELETE OR UPDATE ON public.ek_wallet_reservations FOR EACH ROW EXECUTE FUNCTION enforce_ek_wallet_reservation_lifecycle();


--
-- Name: ek_wallet_transactions ek_wallet_transactions_immutable; Type: TRIGGER; Schema: public; Owner: -
--

CREATE TRIGGER ek_wallet_transactions_immutable BEFORE DELETE OR UPDATE ON public.ek_wallet_transactions FOR EACH ROW EXECUTE FUNCTION prevent_ek_wallet_transaction_mutation();


--
-- Name: campaign_prize_pools campaign_prize_pools_campaign_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.campaign_prize_pools
    ADD CONSTRAINT campaign_prize_pools_campaign_id_fkey FOREIGN KEY (campaign_id) REFERENCES campaigns(campaign_id) ON DELETE CASCADE;


--
-- Name: campaign_prize_results campaign_prize_results_campaign_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.campaign_prize_results
    ADD CONSTRAINT campaign_prize_results_campaign_id_fkey FOREIGN KEY (campaign_id) REFERENCES campaigns(campaign_id) ON DELETE CASCADE;


--
-- Name: campaign_side_entities campaign_side_entities_campaign_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.campaign_side_entities
    ADD CONSTRAINT campaign_side_entities_campaign_id_fkey FOREIGN KEY (campaign_id) REFERENCES campaigns(campaign_id) ON DELETE CASCADE;


--
-- Name: campaign_sides campaign_sides_campaign_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.campaign_sides
    ADD CONSTRAINT campaign_sides_campaign_id_fkey FOREIGN KEY (campaign_id) REFERENCES campaigns(campaign_id) ON DELETE CASCADE;


--
-- Name: campaign_stats_daily campaign_stats_daily_campaign_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.campaign_stats_daily
    ADD CONSTRAINT campaign_stats_daily_campaign_id_fkey FOREIGN KEY (campaign_id) REFERENCES campaigns(campaign_id) ON DELETE CASCADE;


--
-- Name: comment_edits comment_edits_comment_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.comment_edits
    ADD CONSTRAINT comment_edits_comment_id_fkey FOREIGN KEY (comment_id) REFERENCES comments(id) ON DELETE CASCADE;


--
-- Name: comment_reports comment_reports_comment_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.comment_reports
    ADD CONSTRAINT comment_reports_comment_id_fkey FOREIGN KEY (comment_id) REFERENCES comments(id) ON DELETE CASCADE;


--
-- Name: comments comments_parent_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.comments
    ADD CONSTRAINT comments_parent_id_fkey FOREIGN KEY (parent_id) REFERENCES comments(id) ON DELETE CASCADE;


--
-- Name: corporation_wallet_balances corporation_wallet_balances_corporation_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.corporation_wallet_balances
    ADD CONSTRAINT corporation_wallet_balances_corporation_id_fkey FOREIGN KEY (corporation_id) REFERENCES corporations(corporation_id) ON DELETE CASCADE;


--
-- Name: corporation_wallet_journal corporation_wallet_journal_corporation_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.corporation_wallet_journal
    ADD CONSTRAINT corporation_wallet_journal_corporation_id_fkey FOREIGN KEY (corporation_id) REFERENCES corporations(corporation_id) ON DELETE CASCADE;


--
-- Name: corporation_wallet_tokens corporation_wallet_tokens_corporation_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.corporation_wallet_tokens
    ADD CONSTRAINT corporation_wallet_tokens_corporation_id_fkey FOREIGN KEY (corporation_id) REFERENCES corporations(corporation_id) ON DELETE CASCADE;


--
-- Name: custom_domain_campaigns custom_domain_campaigns_campaign_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.custom_domain_campaigns
    ADD CONSTRAINT custom_domain_campaigns_campaign_id_fkey FOREIGN KEY (campaign_id) REFERENCES campaigns(campaign_id) ON DELETE CASCADE;


--
-- Name: custom_domain_campaigns custom_domain_campaigns_domain_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.custom_domain_campaigns
    ADD CONSTRAINT custom_domain_campaigns_domain_id_fkey FOREIGN KEY (domain_id) REFERENCES custom_domains(id) ON DELETE CASCADE;


--
-- Name: ek_wallet_accounts ek_wallet_accounts_character_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.ek_wallet_accounts
    ADD CONSTRAINT ek_wallet_accounts_character_id_fkey FOREIGN KEY (character_id) REFERENCES users(character_id);


--
-- Name: ek_wallet_reservations ek_wallet_reservations_campaign_fk; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.ek_wallet_reservations
    ADD CONSTRAINT ek_wallet_reservations_campaign_fk FOREIGN KEY (campaign_id) REFERENCES campaign_prize_pools(campaign_id);


--
-- Name: ek_wallet_reservations ek_wallet_reservations_captured_transaction_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.ek_wallet_reservations
    ADD CONSTRAINT ek_wallet_reservations_captured_transaction_id_fkey FOREIGN KEY (captured_transaction_id) REFERENCES ek_wallet_transactions(id);


--
-- Name: ek_wallet_reservations ek_wallet_reservations_character_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.ek_wallet_reservations
    ADD CONSTRAINT ek_wallet_reservations_character_id_fkey FOREIGN KEY (character_id) REFERENCES ek_wallet_accounts(character_id);


--
-- Name: ek_wallet_transactions ek_wallet_transactions_campaign_fk; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.ek_wallet_transactions
    ADD CONSTRAINT ek_wallet_transactions_campaign_fk FOREIGN KEY (campaign_id) REFERENCES campaign_prize_pools(campaign_id);


--
-- Name: ek_wallet_transactions ek_wallet_transactions_character_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.ek_wallet_transactions
    ADD CONSTRAINT ek_wallet_transactions_character_id_fkey FOREIGN KEY (character_id) REFERENCES ek_wallet_accounts(character_id);


--
-- Name: ek_wallet_transactions ek_wallet_transactions_journal_fk; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.ek_wallet_transactions
    ADD CONSTRAINT ek_wallet_transactions_journal_fk FOREIGN KEY (corporation_id, division, journal_id) REFERENCES corporation_wallet_journal(corporation_id, division, journal_id);


--
-- Name: ek_wallet_transactions ek_wallet_transactions_related_fk; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.ek_wallet_transactions
    ADD CONSTRAINT ek_wallet_transactions_related_fk FOREIGN KEY (related_transaction_id) REFERENCES ek_wallet_transactions(id);


--
-- Name: esi_request_logs esi_request_logs_character_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.esi_request_logs
    ADD CONSTRAINT esi_request_logs_character_id_fkey FOREIGN KEY (character_id) REFERENCES users(character_id) ON DELETE CASCADE;


--
-- Name: killmail_processing killmail_processing_killmail_id_killmails_killmail_id_fk; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.killmail_processing
    ADD CONSTRAINT killmail_processing_killmail_id_killmails_killmail_id_fk FOREIGN KEY (killmail_id) REFERENCES killmails(killmail_id) ON DELETE CASCADE;


--
-- Name: scans scans_character_id_users_character_id_fk; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.scans
    ADD CONSTRAINT scans_character_id_users_character_id_fk FOREIGN KEY (character_id) REFERENCES users(character_id) ON DELETE CASCADE;


--
-- Name: user_config user_config_character_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.user_config
    ADD CONSTRAINT user_config_character_id_fkey FOREIGN KEY (character_id) REFERENCES users(character_id) ON DELETE CASCADE;


--
-- Name: user_esi_tokens user_esi_tokens_character_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.user_esi_tokens
    ADD CONSTRAINT user_esi_tokens_character_id_fkey FOREIGN KEY (character_id) REFERENCES users(character_id) ON DELETE CASCADE;


--
-- Name: user_fitting_items user_fitting_items_fit_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.user_fitting_items
    ADD CONSTRAINT user_fitting_items_fit_id_fkey FOREIGN KEY (fit_id) REFERENCES user_fittings(fit_id) ON DELETE CASCADE;


--
-- Name: user_fitting_ratings user_fitting_ratings_fit_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.user_fitting_ratings
    ADD CONSTRAINT user_fitting_ratings_fit_id_fkey FOREIGN KEY (fit_id) REFERENCES user_fittings(fit_id) ON DELETE CASCADE;


--
-- Name: user_fitting_ratings user_fitting_ratings_rater_character_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.user_fitting_ratings
    ADD CONSTRAINT user_fitting_ratings_rater_character_id_fkey FOREIGN KEY (rater_character_id) REFERENCES characters(character_id) ON DELETE CASCADE;


--
-- Name: user_fittings user_fittings_owner_character_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.user_fittings
    ADD CONSTRAINT user_fittings_owner_character_id_fkey FOREIGN KEY (owner_character_id) REFERENCES characters(character_id) ON DELETE CASCADE;


--
-- Name: user_sessions user_sessions_character_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.user_sessions
    ADD CONSTRAINT user_sessions_character_id_fkey FOREIGN KEY (character_id) REFERENCES users(character_id) ON DELETE CASCADE;


--
-- Name: wallet_journal_references wallet_journal_references_journal_fk; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.wallet_journal_references
    ADD CONSTRAINT wallet_journal_references_journal_fk FOREIGN KEY (corporation_id, division, journal_id) REFERENCES corporation_wallet_journal(corporation_id, division, journal_id) ON DELETE CASCADE;


--
-- PostgreSQL database dump complete
--



-- +goose Down
-- Intentionally empty. The "down" of a baseline is dropping the entire schema,
-- which against production means destroying ~285 GB across 102 tables. If you
-- genuinely want a clean local database, delete the volume instead:
--
--     docker compose down && rm -rf .data/postgres && docker compose up -d
SELECT 1;
