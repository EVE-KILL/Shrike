-- +goose Up
-- The official Imperium roster on the Goon Wiki was updated on 2026-09-02
-- and contains ten member alliances. Eve Monthly also included four auxiliary
-- alliances; remove those from the community directory and retain the source
-- correction in the same public audit trail used by authenticated edits.
INSERT INTO coalition_edits (
    coalition_id, editor_character_name, action, summary, changes
)
SELECT
    c.coalition_id,
    'EVE-KILL',
    'update',
    'Reconciled membership with the official Imperium wiki; removed 4 alliances',
    jsonb_build_object(
        'official_source', 'https://wiki.goonswarm.org/w/The_Imperium',
        'before', jsonb_build_object(
            'name', c.name,
            'description', c.description,
            'source_url', c.source_url,
            'alliance_ids', COALESCE((
                SELECT jsonb_agg(cm.alliance_id ORDER BY cm.alliance_id)
                FROM coalition_memberships cm
                WHERE cm.coalition_id = c.coalition_id
            ), '[]'::jsonb)
        ),
        'after', jsonb_build_object(
            'name', c.name,
            'description', c.description,
            'source_url', 'https://wiki.goonswarm.org/w/The_Imperium',
            'alliance_ids', COALESCE((
                SELECT jsonb_agg(cm.alliance_id ORDER BY cm.alliance_id)
                FROM coalition_memberships cm
                WHERE cm.coalition_id = c.coalition_id
                  AND cm.alliance_id <> ALL (ARRAY[
                      99010079,
                      99011239,
                      99008165,
                      99012849
                  ]::integer[])
            ), '[]'::jsonb)
        )
    )
FROM coalitions c
WHERE c.slug = 'the-imperium'
  AND NOT EXISTS (
      SELECT 1
      FROM coalition_edits e
      WHERE e.coalition_id = c.coalition_id
        AND e.changes->>'official_source' = 'https://wiki.goonswarm.org/w/The_Imperium'
  );

DELETE FROM coalition_memberships
WHERE coalition_id = (
    SELECT coalition_id FROM coalitions WHERE slug = 'the-imperium'
)
  AND alliance_id = ANY (ARRAY[
      99010079,
      99011239,
      99008165,
      99012849
  ]::integer[]);

UPDATE coalitions
SET source_url = 'https://wiki.goonswarm.org/w/The_Imperium',
    revision = revision + 1,
    updated_by_character_id = NULL,
    updated_at = now()
WHERE slug = 'the-imperium'
  AND source_url IS DISTINCT FROM 'https://wiki.goonswarm.org/w/The_Imperium';

-- +goose Down
INSERT INTO coalition_memberships (coalition_id, alliance_id)
SELECT c.coalition_id, a.alliance_id
FROM coalitions c
JOIN alliances a ON a.alliance_id = ANY (ARRAY[
    99010079,
    99011239,
    99008165,
    99012849
]::integer[])
WHERE c.slug = 'the-imperium'
ON CONFLICT DO NOTHING;

DELETE FROM coalition_edits
WHERE coalition_id = (
    SELECT coalition_id FROM coalitions WHERE slug = 'the-imperium'
)
  AND changes->>'official_source' = 'https://wiki.goonswarm.org/w/The_Imperium';

UPDATE coalitions
SET source_url = 'https://www.evemonthly.com/coalition/the-imperium',
    revision = GREATEST(1, revision - 1),
    updated_at = now()
WHERE slug = 'the-imperium';
