-- Auto-generated full fresh deploy SQL.
-- Source: backend/store/pg/migration/*.up.sql (sorted by version).
-- Generated at 2026-02-14T13:04:25Z.

-- >>> BEGIN 000001_init.up.sql
-- Create "apps" table
CREATE TABLE
    "public"."apps" (
        "id" text NOT NULL,
        "kb_id" text NULL,
        "name" text NULL,
        "type" smallint NULL,
        "settings" jsonb NULL,
        "created_at" timestamptz NULL,
        "updated_at" timestamptz NULL,
        PRIMARY KEY ("id")
    );

-- Create index "idx_apps_kb_id" to table: "apps"
CREATE INDEX "idx_apps_kb_id" ON "public"."apps" ("kb_id");

-- Create "conversation_messages" table
CREATE TABLE
    "public"."conversation_messages" (
        "id" text NOT NULL,
        "conversation_id" text NULL,
        "app_id" text NULL,
        "role" text NULL,
        "content" text NULL,
        "provider" text NULL,
        "model" text NULL,
        "prompt_tokens" bigint NULL DEFAULT 0,
        "completion_tokens" bigint NULL DEFAULT 0,
        "total_tokens" bigint NULL DEFAULT 0,
        "remote_ip" text NULL,
        "created_at" timestamptz NULL,
        PRIMARY KEY ("id")
    );

-- Create index "idx_conversation_messages_app_id" to table: "conversation_messages"
CREATE INDEX "idx_conversation_messages_app_id" ON "public"."conversation_messages" ("app_id");

-- Create index "idx_conversation_messages_conversation_id" to table: "conversation_messages"
CREATE INDEX "idx_conversation_messages_conversation_id" ON "public"."conversation_messages" ("conversation_id");

-- Create "conversation_references" table
CREATE TABLE
    "public"."conversation_references" (
        "conversation_id" text NULL,
        "app_id" text NULL,
        "node_id" text NULL,
        "name" text NULL,
        "url" text NULL,
        "favicon" text NULL
    );

-- Create index "idx_conversation_references_conversation_id" to table: "conversation_references"
CREATE INDEX "idx_conversation_references_conversation_id" ON "public"."conversation_references" ("conversation_id");

-- Create "conversations" table
CREATE TABLE
    "public"."conversations" (
        "id" text NOT NULL,
        "nonce" text NULL,
        "kb_id" text NULL,
        "app_id" text NULL,
        "subject" text NULL,
        "remote_ip" text NULL,
        "created_at" timestamptz NULL,
        PRIMARY KEY ("id")
    );

-- Create index "idx_conversations_kb_id" to table: "conversations"
CREATE INDEX "idx_conversations_kb_id" ON "public"."conversations" ("kb_id");

-- Create index "idx_conversations_app_id" to table: "conversations"
CREATE INDEX "idx_conversations_app_id" ON "public"."conversations" ("app_id");

-- Create "nodes" table
CREATE TABLE
    "public"."nodes" (
        "id" text NOT NULL,
        "kb_id" text NULL,
        "doc_id" text NULL,
        "type" smallint,
        "name" text NULL,
        "content" text NULL,
        "meta" jsonb NULL,
        "parent_id" text NULL,
        "position" float NULL,
        "created_at" timestamptz NULL,
        "updated_at" timestamptz NULL,
        PRIMARY KEY ("id")
    );

-- Create index "idx_nodes_kb_id" to table: "nodes"
CREATE INDEX "idx_nodes_kb_id" ON "public"."nodes" ("kb_id");

-- Create index "idx_nodes_doc_id" to table: "nodes"
CREATE INDEX "idx_nodes_doc_id" ON "public"."nodes" ("doc_id");

-- Create index "idx_nodes_parent_id" to table: "nodes"
CREATE INDEX "idx_nodes_parent_id" ON "public"."nodes" ("parent_id");

-- Create "knowledge_bases" table
CREATE TABLE
    "public"."knowledge_bases" (
        "id" text NOT NULL,
        "name" text NULL,
        "access_settings" jsonb NULL,
        "created_at" timestamptz NULL,
        "updated_at" timestamptz NULL,
        PRIMARY KEY ("id")
    );

-- Create "models" table
CREATE TABLE
    "public"."models" (
        "id" text NOT NULL,
        "provider" text NULL,
        "model" text NULL,
        "api_key" text NULL,
        "api_header" text NULL,
        "base_url" text NULL,
        "api_version" text NULL,
        "prompt_tokens" bigint NULL DEFAULT 0,
        "completion_tokens" bigint NULL DEFAULT 0,
        "total_tokens" bigint NULL DEFAULT 0,
        "created_at" timestamptz NULL,
        "updated_at" timestamptz NULL,
        "is_active" boolean NULL DEFAULT false,
        PRIMARY KEY ("id")
    );

-- Create "users" table
CREATE TABLE
    "public"."users" (
        "id" text NOT NULL,
        "account" text NULL,
        "password" text NULL,
        "created_at" timestamptz NULL,
        "last_access" timestamptz NULL,
        PRIMARY KEY ("id")
    );

-- Create index "idx_users_account" to table: "users"
CREATE UNIQUE INDEX "idx_users_account" ON "public"."users" ("account");

-- <<< END 000001_init.up.sql

-- >>> BEGIN 000002_add_type_for_model.up.sql
-- add type for model
alter table models add column type varchar(255) not null default 'chat';

-- add unique index for type
create unique index idx_models_type on models (type);

-- <<< END 000002_add_type_for_model.up.sql

-- >>> BEGIN 000003_update_rerank_type.up.sql
-- delete embedding and rerank models
DELETE FROM models WHERE type = 'embedding' OR type = 'rerank';

-- <<< END 000003_update_rerank_type.up.sql

-- >>> BEGIN 000004_kb_dataset_id.up.sql
-- add dataset_id to knowledge_bases table
ALTER TABLE "public"."knowledge_bases" ADD COLUMN "dataset_id" text NULL;

-- <<< END 000004_kb_dataset_id.up.sql

-- >>> BEGIN 000005_app_kb_id_type_uniq.up.sql
-- Create unique index "idx_apps_kb_id_type" to table: "apps"
CREATE UNIQUE INDEX "idx_apps_kb_id_type" ON "public"."apps" ("kb_id", "type");

-- Drop index "idx_apps_kb_id" to table: "apps"
DROP INDEX IF EXISTS "idx_apps_kb_id";

-- <<< END 000005_app_kb_id_type_uniq.up.sql

-- >>> BEGIN 000006_node_version.up.sql
-- create node_releases
CREATE TABLE
    "public"."node_releases" (
    id text NOT NULL,
    kb_id text NOT NULL,
    node_id text NOT NULL,
    doc_id text NOT NULL,
    type smallint NULL,
    visibility smallint NULL,
    name text NULL,
    meta JSONB NULL,
    content text NULL,
    parent_id text null,
    position float null,
    created_at timestamptz NULL,
    PRIMARY KEY (id)
);

-- create index on node_releases table
CREATE INDEX "idx_node_releases_kb_id" ON "public"."node_releases" ("kb_id");
CREATE INDEX "idx_node_releases_node_id" ON "public"."node_releases" ("node_id");
CREATE INDEX "idx_node_releases_doc_id" ON "public"."node_releases" ("doc_id");

-- create kb_release
CREATE TABLE
    "public"."kb_releases" (
    id text NOT NULL,
    kb_id text NOT NULL,
    tag text NULL,
    message text NULL,
    created_at timestamptz NULL,
    PRIMARY KEY (id)
);

-- create index on kb_releases table
CREATE INDEX "idx_kb_releases_kb_id" ON "public"."kb_releases" ("kb_id");

-- create kb_release_node_releases
CREATE TABLE
    "public"."kb_release_node_releases" (
    id text NOT NULL,
    kb_id text NOT NULL,
    release_id text NOT NULL,
    node_id text NOT NULL,
    node_release_id text NOT NULL,
    created_at timestamptz NULL,
    PRIMARY KEY (id)
);

-- create index on kb_release_node_releases table
CREATE INDEX "idx_kb_release_node_releases_kb_id" ON "public"."kb_release_node_releases" ("kb_id");
CREATE INDEX "idx_kb_release_node_releases_release_id_node_release_id" ON "public"."kb_release_node_releases" ("release_id", "node_release_id");
CREATE INDEX "idx_kb_release_node_releases_node_id" ON "public"."kb_release_node_releases" ("node_id");

-- update nodes table
ALTER TABLE "public"."nodes" ADD COLUMN "status" smallint NOT NULL DEFAULT 1;
ALTER TABLE "public"."nodes" ADD COLUMN "visibility" smallint NOT NULL DEFAULT 1;

-- update nodes table
UPDATE "public"."nodes" SET "visibility" = 2;


-- create table migrations
CREATE TABLE "public"."migrations" (
    "id" serial PRIMARY KEY,
    "name" varchar(255) NOT NULL,
    "executed_at" timestamptz NOT NULL DEFAULT CURRENT_TIMESTAMP
);

-- create index on migrations table
CREATE UNIQUE INDEX "idx_migrations_name" ON "public"."migrations" ("name");

-- <<< END 000006_node_version.up.sql

-- >>> BEGIN 000007_node_release_updated_at.up.sql
-- add updated_at to node_releases
ALTER TABLE node_releases ADD COLUMN updated_at timestamptz NULL;

-- update existing node_releases
UPDATE node_releases SET updated_at = created_at;

-- <<< END 000007_node_release_updated_at.up.sql

-- >>> BEGIN 000008_add_conversation_info.up.sql
ALTER TABLE conversations ADD COLUMN info jsonb;
-- <<< END 000008_add_conversation_info.up.sql

-- >>> BEGIN 000009_create_stat_pages.up.sql
-- create table stats_pages for 24-hour retention
CREATE TABLE IF NOT EXISTS stat_pages (
    id BIGSERIAL PRIMARY KEY,
    kb_id TEXT NOT NULL,
    node_id TEXT NOT NULL,
    user_id TEXT,
    session_id TEXT,
    scene INT NOT NULL,
    ip TEXT,
    ua TEXT,
    browser_name TEXT,
    browser_os TEXT,
    referer TEXT,
    referer_host TEXT,
    created_at timestamptz NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_stat_pages_kb_id_node_id ON stat_pages(kb_id, node_id);

-- <<< END 000009_create_stat_pages.up.sql

-- >>> BEGIN 000010_add_conversation_message_feedback.up.sql
ALTER TABLE conversation_messages ADD COLUMN info jsonb default '{}';
-- <<< END 000010_add_conversation_message_feedback.up.sql

-- >>> BEGIN 000011_create_user_comment.up.sql
CREATE TABLE "public"."comments" (
        "id" TEXT NOT NULL,
        "user_id" text NULL,
        "node_id" text NOT NULL ,
        "kb_id" text NOT NULL,
        "info" JSONB NULL,
        "parent_id" text DEFAULT NULL,
        "root_id" text DEFAULT NULL,
        "content" text NOT NULL,
        "created_at" timestamptz NULL,
        PRIMARY KEY ("id")
);

CREATE INDEX "idx_comments_node_id" ON "public"."comments" ("node_id");
CREATE INDEX "idx_comments_kb_id" ON "public"."comments"("kb_id");


-- <<< END 000011_create_user_comment.up.sql

-- >>> BEGIN 000012_add_conversation_message_kb_id_parent_id.up.sql
ALTER TABLE conversation_messages ADD COLUMN kb_id TEXT NOT NULL DEFAULT '';

UPDATE conversation_messages as cm
    SET kb_id = (SELECT kb_id from conversations WHERE cm.conversation_id = conversations.id);

ALTER Table conversation_messages ADD COLUMN parent_id TEXT DEFAULT '';
-- <<< END 000012_add_conversation_message_kb_id_parent_id.up.sql

-- >>> BEGIN 000013_create_license.up.sql
-- create table licenses
CREATE TABLE IF NOT EXISTS licenses (
    id SERIAL PRIMARY KEY,
    "type" text,
    code text,
    data bytea,
    created_at timestamptz NOT NULL DEFAULT NOW()
);

-- <<< END 000013_create_license.up.sql

-- >>> BEGIN 000014_add_user_comment_status.up.sql
ALTER Table comments ADD COLUMN status smallint NOT NULL DEFAULT 0;

UPDATE comments SET status = 1;
-- <<< END 000014_add_user_comment_status.up.sql

-- >>> BEGIN 000015_create_auth.up.sql
-- create table auths
CREATE TABLE IF NOT EXISTS auths (
    id SERIAL PRIMARY KEY,
    user_info JSONB NULL,
    union_id text NOT NULL,
    ip text NOT NULL,
    kb_id text NOT NULL,
    source_type text NOT NULL,
    last_login_time timestamptz NOT NULL,
    created_at timestamptz NOT NULL DEFAULT NOW(),
    updated_at timestamptz NOT NULL DEFAULT NOW()
);
-- create table auth_configs
CREATE TABLE IF NOT EXISTS auth_configs (
    id SERIAL PRIMARY KEY,
    kb_id text NOT NULL,
    auth_setting JSONB NULL,
    source_type text NOT NULL UNIQUE,
    created_at timestamptz NOT NULL DEFAULT NOW(),
    updated_at timestamptz NOT NULL DEFAULT NOW()
);

-- <<< END 000015_create_auth.up.sql

-- >>> BEGIN 000016_create_document_feedback.up.sql
CREATE TABLE IF NOT EXISTS document_feedbacks (
    id BIGSERIAL PRIMARY KEY,
    user_id TEXT NULL,
    kb_id TEXT NOT NULL,
    node_id TEXT NOT NULL DEFAULT '',
    content TEXT NOT NULL DEFAULT '', 
    correction_suggestion TEXT NOT NULL DEFAULT '',
    info JSONB DEFAULT '{}',
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- <<< END 000016_create_document_feedback.up.sql

-- >>> BEGIN 000017_updtate_conversation_message_feedback.up.sql
UPDATE conversation_messages
SET info = jsonb_set(
    info,
    '{feedback_type}',
    CASE (info->>'feedback_type')::int
        WHEN 1 THEN to_jsonb('内容不准确'::text)
        WHEN 2 THEN to_jsonb('没有帮助'::text)
        WHEN 3 THEN to_jsonb('其他'::text)
        ELSE to_jsonb(''::text)
    END
)
WHERE (info->>'feedback_type') IS NOT NULL;


-- <<< END 000017_updtate_conversation_message_feedback.up.sql

-- >>> BEGIN 000018_create_settings.up.sql
-- Create settings table
CREATE TABLE IF NOT EXISTS settings (
    id SERIAL PRIMARY KEY,
    kb_id TEXT NOT NULL,
    key TEXT NOT NULL,
    value JSONB NOT NULL,
    description TEXT,
    created_at timestamptz NOT NULL DEFAULT NOW(),
    updated_at timestamptz NOT NULL DEFAULT NOW()
);

-- Create unique index for kb_id + key combination
CREATE UNIQUE INDEX idx_settings_kb_id_key ON settings (kb_id, key);
-- <<< END 000018_create_settings.up.sql

-- >>> BEGIN 000019_alter_stat_pages_type.up.sql
UPDATE stat_pages SET user_id = NULL WHERE user_id = '';
ALTER TABLE stat_pages
ALTER COLUMN user_id TYPE bigint USING user_id::bigint;
-- <<< END 000019_alter_stat_pages_type.up.sql

-- >>> BEGIN 000020_add_user_role_and_kb_users.up.sql
-- Add role column to users table
ALTER TABLE "public"."users" ADD COLUMN "role" text NOT NULL DEFAULT 'user';

-- Set existing users as admin
UPDATE "public"."users" SET "role" = 'admin';

-- Create kb_users table for user-kb permissions
CREATE TABLE "public"."kb_users" (
    "id" BIGSERIAL NOT NULL,
    "kb_id" text NOT NULL,
    "user_id" text NOT NULL,
    "perm" text NOT NULL DEFAULT 'full_control',
    "created_at" timestamptz NOT NULL DEFAULT CURRENT_TIMESTAMP,
    PRIMARY KEY ("id")
);

-- Add unique constraint for kb_id and user_id
ALTER TABLE "public"."kb_users" ADD CONSTRAINT "uniq_kb_users_kb_id_user_id" UNIQUE ("kb_id", "user_id");

-- Update auth_configs constraints
ALTER TABLE auth_configs DROP CONSTRAINT auth_configs_source_type_key;
ALTER TABLE auth_configs ADD CONSTRAINT uniq_auth_configs_source_type_kb_id UNIQUE (source_type, kb_id);
-- <<< END 000020_add_user_role_and_kb_users.up.sql

-- >>> BEGIN 000021_create_auth_groups.up.sql
-- Create auth_groups table
CREATE TABLE IF NOT EXISTS auth_groups (
    id SERIAL PRIMARY KEY,
    kb_id TEXT NOT NULL,
    name VARCHAR(100) NOT NULL UNIQUE,
    auth_ids INTEGER[] DEFAULT '{}',
    created_at TIMESTAMP NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP NOT NULL DEFAULT NOW()
);

-- Create node_auth_groups table
CREATE TABLE IF NOT EXISTS node_auth_groups (
    id SERIAL PRIMARY KEY,
    node_id TEXT NOT NULL,
    auth_group_id INTEGER NOT NULL,
    perm TEXT NOT NULL,
    created_at TIMESTAMP NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP NOT NULL DEFAULT NOW(),
    UNIQUE(node_id, auth_group_id, perm)
);


ALTER TABLE nodes ADD COLUMN permissions jsonb default '{}';
UPDATE nodes set permissions='{"answerable":"open","visitable":"open","visible":"open"}'::jsonb;


-- update nodes table
ALTER TABLE "public"."nodes" ADD COLUMN "creator_id" TEXT NOT NULL DEFAULT '';
ALTER TABLE "public"."nodes" ADD COLUMN "editor_id" TEXT NOT NULL DEFAULT '';

UPDATE nodes SET creator_id = u.id, editor_id = u.id FROM "users" u WHERE u.account = 'admin';

UPDATE nodes set "permissions"='{"answerable":"closed","visitable":"closed","visible":"closed"}'::jsonb, "status"=1 where "visibility"=1;

ALTER TABLE nodes ADD COLUMN edit_time TIMESTAMP;

UPDATE nodes SET edit_time=updated_at ;

-- <<< END 000021_create_auth_groups.up.sql

-- >>> BEGIN 000022_alter_model.up.sql
-- Add parameters column to models table
ALTER TABLE "public"."models" ADD COLUMN "parameters" JSONB;
-- <<< END 000022_alter_model.up.sql

-- >>> BEGIN 000023_create_stat_page_hours.up.sql
CREATE TABLE IF NOT EXISTS stat_page_hours (
    id BIGSERIAL PRIMARY KEY,
    kb_id TEXT NOT NULL,
    hour timestamptz NOT NULL,
    ip_count BIGINT NOT NULL DEFAULT 0,
    session_count BIGINT NOT NULL DEFAULT 0,
    page_visit_count BIGINT NOT NULL DEFAULT 0,
    conversation_count BIGINT NOT NULL DEFAULT 0,
    geo_count JSONB NULL,
    conversation_distribution JSONB NULL,
    hot_referer_host JSONB NULL,
    hot_page JSONB NULL,
    hot_os JSONB NULL,
    hot_browser JSONB NULL,
    created_at timestamptz NOT NULL DEFAULT NOW(),
    UNIQUE(kb_id, hour)
);

CREATE INDEX IF NOT EXISTS idx_stat_page_hours_hour ON stat_page_hours (hour);

-- <<< END 000023_create_stat_page_hours.up.sql

-- >>> BEGIN 000024_add_parent_id_to_auth_groups.up.sql
ALTER TABLE auth_groups ADD COLUMN IF NOT EXISTS parent_id INTEGER DEFAULT NULL;

ALTER TABLE auth_groups ADD COLUMN IF NOT EXISTS position FLOAT8 DEFAULT 0;

-- Update existing records with default positions (1000, 2000, 3000, etc.)
UPDATE auth_groups SET position = (id * 1000)::FLOAT8;
-- <<< END 000024_add_parent_id_to_auth_groups.up.sql

-- >>> BEGIN 000025_create_api_tokens_table.up.sql
CREATE TABLE IF NOT EXISTS api_tokens (
    id TEXT PRIMARY KEY,
    kb_id TEXT NOT NULL,
    name TEXT NOT NULL,
    user_id TEXT NOT NULL,
    token TEXT NOT NULL,
    permission TEXT NOT NULL,
    created_at TIMESTAMP NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP NOT NULL DEFAULT NOW(),
    UNIQUE(token)
);
-- <<< END 000025_create_api_tokens_table.up.sql

-- >>> BEGIN 000026_add_sync.up.sql
ALTER TABLE auth_groups ADD COLUMN IF NOT EXISTS sync_id text NOT NULL DEFAULT '';
ALTER TABLE auth_groups ADD COLUMN IF NOT EXISTS sync_parent_id text NOT NULL DEFAULT '';
ALTER TABLE auth_groups ADD COLUMN IF NOT EXISTS source_type text NOT NULL DEFAULT '';
ALTER TABLE auth_groups DROP CONSTRAINT IF EXISTS auth_groups_name_key;


-- <<< END 000026_add_sync.up.sql

-- >>> BEGIN 000027_create_contributes_table.up.sql
CREATE TABLE IF NOT EXISTS contributes (
    id TEXT PRIMARY KEY,
    auth_id BIGINT,
    kb_id TEXT NOT NULL,
    status TEXT NOT NULL,
    type TEXT NOT NULL,
    node_id TEXT,
    name TEXT,
    content TEXT NOT NULL,
    reason TEXT NOT NULL,
    audit_user_id TEXT NOT NULL,
    meta JSONB,
    audit_time TIMESTAMP,
    created_at TIMESTAMP NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP NOT NULL DEFAULT NOW()
);

-- <<< END 000027_create_contributes_table.up.sql

-- >>> BEGIN 000028_add_contributes_ip.up.sql
ALTER TABLE contributes ADD COLUMN IF NOT EXISTS remote_ip text not null default '';

-- <<< END 000028_add_contributes_ip.up.sql

-- >>> BEGIN 000029_add_comment_pic_urls.up.sql
ALTER TABLE comments ADD COLUMN IF NOT EXISTS pic_urls text[] not null default ARRAY[]::text[];
-- <<< END 000029_add_comment_pic_urls.up.sql

-- >>> BEGIN 000030_add_node_status_msg.up.sql
ALTER TABLE nodes ADD COLUMN IF NOT EXISTS rag_info jsonb default '{}';

-- <<< END 000030_add_node_status_msg.up.sql

-- >>> BEGIN 000031_add_node_release_user_id.up.sql
ALTER TABLE node_releases ADD COLUMN IF NOT EXISTS publisher_id text default '';
ALTER TABLE node_releases ADD COLUMN IF NOT EXISTS editor_id text default '';
-- <<< END 000031_add_node_release_user_id.up.sql

-- >>> BEGIN 000032_create_system_settings.up.sql
-- Create settings table
CREATE TABLE IF NOT EXISTS system_settings (
    id SERIAL PRIMARY KEY,
    key TEXT NOT NULL,
    value JSONB NOT NULL,
    description TEXT,
    created_at timestamptz NOT NULL DEFAULT NOW(),
    updated_at timestamptz NOT NULL DEFAULT NOW()
);

CREATE UNIQUE INDEX idx_uniq_system_settings_key ON system_settings(key);

-- Insert model_setting_mode setting
-- If there are existing knowledge bases, set mode to 'manual', otherwise set to 'auto'
INSERT INTO system_settings (key, value, description)
SELECT 
    'model_setting_mode',
    jsonb_build_object(
        'mode', CASE 
            WHEN EXISTS (SELECT 1 FROM knowledge_bases LIMIT 1) THEN 'manual'
            ELSE 'auto'
        END,
        'auto_mode_api_key', '',
        'chat_model', '',
        'is_manual_embedding_updated', false
    ),
    'Model setting mode configuration'
WHERE NOT EXISTS (
    SELECT 1 FROM system_settings WHERE key = 'model_setting_mode'
);
-- <<< END 000032_create_system_settings.up.sql

-- >>> BEGIN 000033_create_mcp_calls.up.sql
CREATE TABLE IF NOT EXISTS mcp_calls (
    id SERIAL PRIMARY KEY,
    mcp_session_id TEXT NOT NULL,
    kb_id TEXT NOT NULL,
    remote_ip TEXT,
    initialize_req JSONB,
    initialize_resp JSONB,
    tool_call_req JSONB,
    tool_call_resp TEXT,
    created_at timestamptz NOT NULL DEFAULT NOW()
);

-- <<< END 000033_create_mcp_calls.up.sql

-- >>> BEGIN 000034_create_node_stats.up.sql
CREATE TABLE IF NOT EXISTS node_stats (
    id BIGSERIAL PRIMARY KEY,
    node_id TEXT NOT NULL UNIQUE,
    pv BIGINT NOT NULL DEFAULT 0,
    created_at timestamptz NOT NULL DEFAULT NOW()
);


-- <<< END 000034_create_node_stats.up.sql

-- >>> BEGIN 000035_add_conversation_image_paths.up.sql
ALTER TABLE conversation_messages ADD COLUMN IF NOT EXISTS image_paths text[] NOT NULL DEFAULT '{}';
-- <<< END 000035_add_conversation_image_paths.up.sql

-- >>> BEGIN 000036_add_kb_release_publisher_id.up.sql
ALTER TABLE kb_releases ADD COLUMN IF NOT EXISTS publisher_id text default '';

-- <<< END 000036_add_kb_release_publisher_id.up.sql

-- >>> BEGIN 000037_create_prompt_versions.up.sql
CREATE TABLE IF NOT EXISTS prompt_versions (
    id BIGSERIAL PRIMARY KEY,
    kb_id TEXT NOT NULL,
    version INTEGER NOT NULL CHECK (version > 0),
    content TEXT NOT NULL,
    summary_content TEXT NOT NULL,
    operator_user_id TEXT NOT NULL,
    created_at timestamptz NOT NULL DEFAULT NOW()
);

CREATE UNIQUE INDEX IF NOT EXISTS idx_prompt_versions_kb_id_version
ON prompt_versions(kb_id, version);

CREATE INDEX IF NOT EXISTS idx_prompt_versions_kb_id_created_at
ON prompt_versions(kb_id, created_at DESC);

CREATE INDEX IF NOT EXISTS idx_prompt_versions_operator_user_id_created_at
ON prompt_versions(operator_user_id, created_at DESC);

-- <<< END 000037_create_prompt_versions.up.sql

-- >>> BEGIN 000038_create_api_call_audits.up.sql
CREATE TABLE IF NOT EXISTS api_call_audits (
    id BIGSERIAL PRIMARY KEY,
    kb_id TEXT NOT NULL,
    api_token_id TEXT,
    endpoint TEXT NOT NULL,
    model TEXT NOT NULL DEFAULT '',
    status_code INTEGER NOT NULL,
    error_type TEXT,
    error_message TEXT,
    prompt_tokens INTEGER NOT NULL DEFAULT 0,
    completion_tokens INTEGER NOT NULL DEFAULT 0,
    total_tokens INTEGER NOT NULL DEFAULT 0,
    latency_ms BIGINT NOT NULL DEFAULT 0 CHECK (latency_ms >= 0),
    remote_ip TEXT,
    request_id TEXT,
    created_at timestamptz NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_api_call_audits_kb_id_created_at
ON api_call_audits(kb_id, created_at DESC);

CREATE INDEX IF NOT EXISTS idx_api_call_audits_api_token_id_created_at
ON api_call_audits(api_token_id, created_at DESC);

CREATE INDEX IF NOT EXISTS idx_api_call_audits_endpoint_created_at
ON api_call_audits(endpoint, created_at DESC);

CREATE INDEX IF NOT EXISTS idx_api_call_audits_model_created_at
ON api_call_audits(model, created_at DESC);

-- <<< END 000038_create_api_call_audits.up.sql

-- >>> BEGIN 000039_create_node_release_audits.up.sql
CREATE TABLE IF NOT EXISTS node_release_audits (
    id BIGSERIAL PRIMARY KEY,
    kb_id TEXT NOT NULL,
    node_id TEXT NOT NULL,
    action TEXT NOT NULL,
    operator_user_id TEXT NOT NULL,
    source_version TEXT,
    target_version TEXT,
    detail JSONB NOT NULL DEFAULT '{}'::jsonb,
    created_at timestamptz NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_node_release_audits_kb_id_node_id_created_at
ON node_release_audits(kb_id, node_id, created_at DESC);

CREATE INDEX IF NOT EXISTS idx_node_release_audits_kb_id_created_at
ON node_release_audits(kb_id, created_at DESC);

CREATE INDEX IF NOT EXISTS idx_node_release_audits_operator_user_id_created_at
ON node_release_audits(operator_user_id, created_at DESC);

CREATE INDEX IF NOT EXISTS idx_node_release_audits_action_created_at
ON node_release_audits(action, created_at DESC);

-- <<< END 000039_create_node_release_audits.up.sql

-- >>> BEGIN 000040_add_api_token_governance_fields.up.sql
ALTER TABLE api_tokens
ADD COLUMN IF NOT EXISTS rate_limit_per_minute INTEGER NOT NULL DEFAULT 0;

ALTER TABLE api_tokens
ADD COLUMN IF NOT EXISTS daily_quota INTEGER NOT NULL DEFAULT 0;
-- <<< END 000040_add_api_token_governance_fields.up.sql

-- >>> BEGIN 000041_set_default_stats_pv_enable.up.sql
UPDATE apps
SET settings = jsonb_set(
    COALESCE(settings, '{}'::jsonb),
    '{stats_setting,pv_enable}',
    'true'::jsonb,
    true
)
WHERE (settings->'stats_setting'->'pv_enable') IS NULL;
-- <<< END 000041_set_default_stats_pv_enable.up.sql

-- Ensure migration version is recorded for fresh deployments.
DO $$
BEGIN
    IF to_regclass('public.schema_migrations') IS NULL THEN
        CREATE TABLE public.schema_migrations (
            version BIGINT NOT NULL PRIMARY KEY,
            dirty BOOLEAN NOT NULL
        );
    END IF;

    DELETE FROM public.schema_migrations;
    INSERT INTO public.schema_migrations (version, dirty) VALUES (41, FALSE);
END $$;
