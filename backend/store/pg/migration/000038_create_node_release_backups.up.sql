CREATE TABLE IF NOT EXISTS node_release_backup (
    id           text        NOT NULL,
    kb_id        text        NOT NULL,
    node_id      text        NOT NULL,
    doc_id       text        NOT NULL,
    type         int2,
    name         text,
    meta         jsonb,
    content      text,
    parent_id    text,
    position     float8,
    created_at   timestamptz,
    updated_at   timestamptz,
    publisher_id text,
    editor_id    text,
    deleted_at   timestamptz NOT NULL DEFAULT now(),
    CONSTRAINT node_release_backup_pkey PRIMARY KEY (id)
);

CREATE INDEX IF NOT EXISTS node_release_backup_deleted_at_idx ON node_release_backup (deleted_at);