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
