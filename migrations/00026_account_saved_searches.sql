-- +goose Up
CREATE TABLE IF NOT EXISTS account_saved_searches (
    search_id bigint GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    character_id integer NOT NULL REFERENCES users(character_id) ON DELETE CASCADE,
    name text NOT NULL CHECK (char_length(name) BETWEEN 1 AND 100),
    document jsonb NOT NULL CHECK (jsonb_typeof(document) = 'object' AND octet_length(document::text) <= 16384),
    revision integer NOT NULL DEFAULT 1 CHECK (revision > 0),
    created_at timestamptz NOT NULL DEFAULT now(),
    updated_at timestamptz NOT NULL DEFAULT now()
);
CREATE UNIQUE INDEX IF NOT EXISTS account_saved_searches_name_idx ON account_saved_searches (character_id, lower(name));

-- +goose Down
DROP TABLE IF EXISTS account_saved_searches;
