-- Keep the Goose application schema in lockstep with
-- backend/drizzle/0067_domain_campaign_public.sql.

-- +goose Up
ALTER TABLE "custom_domain_campaigns"
    ADD COLUMN IF NOT EXISTS "public_on_domain" boolean DEFAULT false NOT NULL;

-- +goose Down
ALTER TABLE "custom_domain_campaigns"
    DROP COLUMN "public_on_domain";
