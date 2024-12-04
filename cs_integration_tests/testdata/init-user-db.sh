#!/bin/bash
set -e

psql -v ON_ERROR_STOP=1 --username "$POSTGRES_USER" --dbname "$POSTGRES_DB" <<-EOSQL
    CREATE TABLE "content_items" (
        "id" UUID DEFAULT gen_random_uuid() PRIMARY KEY,
        "base_path" VARCHAR UNIQUE,
        "content_id" VARCHAR,
        "title" VARCHAR,
        "description" JSONB DEFAULT '{"value": null}'::jsonb,
        "document_type" VARCHAR,
        "content_purpose_document_supertype" VARCHAR DEFAULT '',
        "content_purpose_subgroup" VARCHAR DEFAULT '',
        "content_purpose_supergroup" VARCHAR DEFAULT '',
        "email_document_supertype" VARCHAR DEFAULT '',
        "government_document_supertype" VARCHAR DEFAULT '',
        "navigation_document_supertype" VARCHAR DEFAULT '',
        "search_user_need_document_supertype" VARCHAR DEFAULT '',
        "user_journey_document_supertype" VARCHAR DEFAULT '',
        "schema_name" VARCHAR,
        "locale" VARCHAR DEFAULT 'en',
        "first_published_at" TIMESTAMP,
        "public_updated_at" TIMESTAMP,
        "publishing_scheduled_at" TIMESTAMP,
        "details" JSONB DEFAULT '{}'::jsonb,
        "publishing_app" VARCHAR,
        "rendering_app" VARCHAR,
        "routes" JSONB DEFAULT '[]'::jsonb,
        "redirects" JSONB DEFAULT '[]'::jsonb,
        "expanded_links" JSONB DEFAULT '{}'::jsonb,
        "access_limited" JSONB DEFAULT '{}'::jsonb,
        "auth_bypass_ids" VARCHAR[] DEFAULT '{}',
        "phase" VARCHAR DEFAULT 'live',
        "analytics_identifier" VARCHAR,
        "payload_version" INTEGER,
        "withdrawn_notice" JSONB DEFAULT '{}'::jsonb,
        "publishing_request_id" VARCHAR,
        "created_at" TIMESTAMP,
        "updated_at" TIMESTAMP,
        "_id" VARCHAR,
        "scheduled_publishing_delay_seconds" BIGINT
    );

    CREATE TABLE "publish_intents" (
        "id" UUID DEFAULT gen_random_uuid() PRIMARY KEY,
        "base_path" VARCHAR UNIQUE,
        "publish_time" TIMESTAMP,
        "publishing_app" VARCHAR,
        "rendering_app" VARCHAR,
        "routes" JSONB DEFAULT '[]'::jsonb,
        "created_at" TIMESTAMP,
        "updated_at" TIMESTAMP
    );

    CREATE INDEX "index_content_items_on_content_id" ON "content_items" ("content_id");
    CREATE INDEX "index_content_items_on_created_at" ON "content_items" ("created_at");
    CREATE INDEX "index_content_items_on_redirects" ON "content_items" USING gin("redirects");
    CREATE INDEX "ix_ci_redirects_jsonb_path_ops" ON "content_items" USING gin("redirects" jsonb_path_ops);
    CREATE INDEX "index_content_items_on_routes" ON "content_items" USING gin("routes");
    CREATE INDEX "ix_ci_routes_jsonb_path_ops" ON "content_items" USING gin("routes" jsonb_path_ops);
    CREATE INDEX "index_content_items_on_updated_at" ON "content_items" ("updated_at");

    CREATE INDEX "index_publish_intents_on_created_at" ON "publish_intents" ("created_at");
    CREATE INDEX "index_publish_intents_on_publish_time" ON "publish_intents" ("publish_time");
    CREATE INDEX "index_publish_intents_on_routes" ON "publish_intents" USING gin("routes");
    CREATE INDEX "ix_pi_routes_jsonb_path_ops" ON "publish_intents" USING gin("routes" jsonb_path_ops);
    CREATE INDEX "index_publish_intents_on_updated_at" ON "publish_intents" ("updated_at");

    CREATE OR REPLACE FUNCTION notify_route_change() RETURNS trigger AS \$$
    BEGIN
        PERFORM pg_notify('route_changes', '');
        RETURN OLD;
    END;
    \$$ LANGUAGE plpgsql;

    CREATE TRIGGER content_item_change_trigger
    AFTER INSERT OR UPDATE OR DELETE ON content_items
    FOR EACH ROW EXECUTE PROCEDURE notify_route_change();

    CREATE TRIGGER publish_intent_change_trigger
    AFTER INSERT OR UPDATE OR DELETE ON publish_intents
    FOR EACH ROW EXECUTE PROCEDURE notify_route_change();
EOSQL