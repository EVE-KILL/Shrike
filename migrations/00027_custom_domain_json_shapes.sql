-- +goose Up
-- Early custom-domain writes stored JSON documents as JSON strings. Decode
-- exactly once; invalid documents fail the transaction instead of losing data.
UPDATE custom_domains SET entities = (entities #>> '{}')::jsonb
WHERE jsonb_typeof(entities) = 'string';
UPDATE custom_domains SET theme = (theme #>> '{}')::jsonb
WHERE jsonb_typeof(theme) = 'string';
UPDATE custom_domains SET navbar_links = (navbar_links #>> '{}')::jsonb
WHERE jsonb_typeof(navbar_links) = 'string';
UPDATE custom_domains SET widgets = (widgets #>> '{}')::jsonb
WHERE jsonb_typeof(widgets) = 'string';

-- Allow adoption after the same repair was applied during incident recovery.
-- +goose StatementBegin
DO $$
BEGIN
    IF NOT EXISTS (
        SELECT 1 FROM pg_constraint
        WHERE conrelid = 'custom_domains'::regclass
          AND conname = 'custom_domains_json_shapes_check'
    ) THEN
        ALTER TABLE custom_domains ADD CONSTRAINT custom_domains_json_shapes_check
        CHECK (
            jsonb_typeof(entities) = 'array'
            AND jsonb_typeof(theme) = 'object'
            AND jsonb_typeof(navbar_links) = 'array'
            AND jsonb_typeof(widgets) = 'object'
        );
    END IF;
END
$$;
-- +goose StatementEnd

-- +goose Down
ALTER TABLE custom_domains DROP CONSTRAINT IF EXISTS custom_domains_json_shapes_check;
-- Keep normalized documents: re-encoding them would break custom-board reads.
