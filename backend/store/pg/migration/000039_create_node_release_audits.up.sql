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
