CREATE EXTENSION IF NOT EXISTS vector;

CREATE TABLE IF NOT EXISTS rag_chunks (
    id          TEXT PRIMARY KEY,
    dataset_id  TEXT NOT NULL,
    doc_id      TEXT NOT NULL,
    content     TEXT NOT NULL,
    embedding   vector(1024) NOT NULL,
    group_ids   INT[] NOT NULL DEFAULT '{}',
    tags        TEXT[] NOT NULL DEFAULT '{}',
    seq         INT NOT NULL DEFAULT 0,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_rag_chunks_dataset_id ON rag_chunks (dataset_id);
CREATE INDEX IF NOT EXISTS idx_rag_chunks_doc_id ON rag_chunks (dataset_id, doc_id);
CREATE INDEX IF NOT EXISTS idx_rag_chunks_group_ids ON rag_chunks USING gin (group_ids);
CREATE INDEX IF NOT EXISTS idx_rag_chunks_embedding ON rag_chunks USING hnsw (embedding vector_cosine_ops);
