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
