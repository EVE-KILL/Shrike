-- +goose Up
-- Curated coalition membership is deliberately separate from the inferred
-- coalition graph. EVE has no first-party coalition entity, so this table is
-- the community-maintained source of truth exposed by the public API.
CREATE TABLE IF NOT EXISTS coalitions (
    coalition_id bigint GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    slug text NOT NULL UNIQUE,
    name text NOT NULL,
    description text NOT NULL DEFAULT '',
    source_url text,
    revision integer NOT NULL DEFAULT 1,
    created_by_character_id integer REFERENCES users(character_id) ON DELETE SET NULL,
    updated_by_character_id integer REFERENCES users(character_id) ON DELETE SET NULL,
    created_at timestamptz NOT NULL DEFAULT now(),
    updated_at timestamptz NOT NULL DEFAULT now(),
    CONSTRAINT coalitions_slug_check CHECK (slug ~ '^[a-z0-9]+(?:-[a-z0-9]+)*$'),
    CONSTRAINT coalitions_name_check CHECK (length(name) BETWEEN 2 AND 100),
    CONSTRAINT coalitions_description_check CHECK (length(description) <= 2000),
    CONSTRAINT coalitions_source_url_check CHECK (source_url IS NULL OR length(source_url) <= 2048),
    CONSTRAINT coalitions_revision_check CHECK (revision > 0)
);

CREATE UNIQUE INDEX IF NOT EXISTS coalitions_name_unique_idx
    ON coalitions (lower(name));

CREATE TABLE IF NOT EXISTS coalition_memberships (
    coalition_id bigint NOT NULL REFERENCES coalitions(coalition_id) ON DELETE CASCADE,
    alliance_id integer NOT NULL REFERENCES alliances(alliance_id),
    added_by_character_id integer REFERENCES users(character_id) ON DELETE SET NULL,
    added_at timestamptz NOT NULL DEFAULT now(),
    PRIMARY KEY (coalition_id, alliance_id),
    CONSTRAINT coalition_memberships_one_coalition_per_alliance UNIQUE (alliance_id)
);

CREATE INDEX IF NOT EXISTS coalition_memberships_coalition_idx
    ON coalition_memberships (coalition_id, alliance_id);

CREATE TABLE IF NOT EXISTS coalition_edits (
    edit_id bigint GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    coalition_id bigint NOT NULL REFERENCES coalitions(coalition_id) ON DELETE CASCADE,
    editor_character_id integer REFERENCES users(character_id) ON DELETE SET NULL,
    editor_character_name text NOT NULL,
    action text NOT NULL,
    summary text NOT NULL,
    changes jsonb NOT NULL DEFAULT '{}'::jsonb,
    created_at timestamptz NOT NULL DEFAULT now(),
    CONSTRAINT coalition_edits_action_check CHECK (action IN ('seed', 'create', 'update')),
    CONSTRAINT coalition_edits_summary_check CHECK (length(summary) BETWEEN 1 AND 500)
);

CREATE INDEX IF NOT EXISTS coalition_edits_coalition_created_idx
    ON coalition_edits (coalition_id, created_at DESC, edit_id DESC);

-- Seed the two coalitions verified against Eve Monthly on 2026-09-03. The
-- alliance join keeps fresh/throwaway databases migratable when their entity
-- corpus is intentionally incomplete; production already carries these IDs.
INSERT INTO coalitions (slug, name, description, source_url)
VALUES
    (
        'the-imperium',
        'The Imperium',
        'Community-maintained alliance membership for The Imperium.',
        'https://www.evemonthly.com/coalition/the-imperium'
    ),
    (
        'winter-coalition',
        'Winter Coalition',
        'Community-maintained alliance membership for Winter Coalition.',
        'https://www.evemonthly.com/coalition/winter-coalition'
    )
ON CONFLICT (slug) DO NOTHING;

WITH membership(slug, alliance_id) AS (
    VALUES
        ('winter-coalition', 99003581),
        ('winter-coalition', 99013541),
        ('winter-coalition', 498125261),
        ('winter-coalition', 1042504553),
        ('winter-coalition', 1727758877),
        ('winter-coalition', 386292982),
        ('winter-coalition', 99005830),
        ('winter-coalition', 99008783),
        ('winter-coalition', 99002685),
        ('winter-coalition', 99014081),
        ('winter-coalition', 99005393),
        ('winter-coalition', 99006125),
        ('winter-coalition', 99009129),
        ('winter-coalition', 99001317),
        ('winter-coalition', 99005100),
        ('winter-coalition', 99007681),
        ('winter-coalition', 99000163),
        ('winter-coalition', 99006494),
        ('winter-coalition', 99011521),
        ('winter-coalition', 99002542),
        ('the-imperium', 1354830081),
        ('the-imperium', 99009163),
        ('the-imperium', 99003214),
        ('the-imperium', 99003995),
        ('the-imperium', 99011223),
        ('the-imperium', 99010079),
        ('the-imperium', 99010877),
        ('the-imperium', 131511956),
        ('the-imperium', 99012042),
        ('the-imperium', 99011239),
        ('the-imperium', 99001969),
        ('the-imperium', 99008165),
        ('the-imperium', 99009331),
        ('the-imperium', 99012849)
)
INSERT INTO coalition_memberships (coalition_id, alliance_id)
SELECT c.coalition_id, membership.alliance_id
FROM membership
JOIN coalitions c ON c.slug = membership.slug
JOIN alliances a ON a.alliance_id = membership.alliance_id
ON CONFLICT DO NOTHING;

INSERT INTO coalition_edits (
    coalition_id, editor_character_name, action, summary, changes
)
SELECT
    c.coalition_id,
    'EVE-KILL',
    'seed',
    'Seeded the initial community-maintained coalition record',
    jsonb_build_object(
        'after', jsonb_build_object(
            'name', c.name,
            'description', c.description,
            'source_url', c.source_url,
            'alliance_ids', COALESCE((
                SELECT jsonb_agg(cm.alliance_id ORDER BY cm.alliance_id)
                FROM coalition_memberships cm
                WHERE cm.coalition_id = c.coalition_id
            ), '[]'::jsonb)
        )
    )
FROM coalitions c
WHERE c.slug IN ('the-imperium', 'winter-coalition')
  AND NOT EXISTS (
      SELECT 1 FROM coalition_edits e
      WHERE e.coalition_id = c.coalition_id AND e.action = 'seed'
  );

-- +goose Down
DROP TABLE IF EXISTS coalition_edits;
DROP TABLE IF EXISTS coalition_memberships;
DROP TABLE IF EXISTS coalitions;
