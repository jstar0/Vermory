-- +goose Up

ALTER TABLE memory_retrieval_profiles
  ADD COLUMN projection_class TEXT;

UPDATE memory_retrieval_profiles
SET projection_class = 'vector_1024';

ALTER TABLE memory_retrieval_profiles
  ALTER COLUMN projection_class SET NOT NULL,
  DROP CONSTRAINT memory_retrieval_profiles_dimensions_check,
  ADD CONSTRAINT memory_retrieval_profiles_projection_class_check
    CHECK (projection_class IN ('vector_1024', 'vector_512')),
  ADD CONSTRAINT memory_retrieval_profiles_dimensions_class_check
    CHECK (
      (projection_class = 'vector_1024' AND dimensions = 1024)
      OR (projection_class = 'vector_512' AND dimensions = 512)
    ),
  ADD CONSTRAINT memory_retrieval_profiles_profile_class_uq
    UNIQUE (profile_id, projection_class);

INSERT INTO memory_retrieval_profiles (
  profile_id, provider_base_url, model, dimensions, projection_class,
  lifecycle_status, activated_at
) VALUES (
  'siliconflow-bge-small-zh-512-v3',
  'https://api.siliconflow.cn/v1',
  'BAAI/bge-small-zh-v1.5',
  512,
  'vector_512',
  'candidate',
  NULL
);

ALTER TABLE memory_vector_documents
  ADD COLUMN projection_class TEXT NOT NULL DEFAULT 'vector_1024',
  ADD CONSTRAINT memory_vector_documents_projection_class_check
    CHECK (projection_class = 'vector_1024'),
  ADD CONSTRAINT memory_vector_documents_profile_class_fk
    FOREIGN KEY (profile_id, projection_class)
    REFERENCES memory_retrieval_profiles (profile_id, projection_class);

CREATE TABLE memory_vector_documents_512 (
  profile_id TEXT NOT NULL,
  projection_class TEXT NOT NULL DEFAULT 'vector_512'
    CHECK (projection_class = 'vector_512'),
  tenant_id TEXT NOT NULL CHECK (btrim(tenant_id) <> ''),
  continuity_id UUID NOT NULL,
  memory_id UUID NOT NULL,
  content_sha256 TEXT NOT NULL CHECK (length(content_sha256) = 64),
  embedding vector(512) NOT NULL,
  updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  PRIMARY KEY (profile_id, tenant_id, memory_id),
  CONSTRAINT memory_vector_documents_512_profile_class_fk
    FOREIGN KEY (profile_id, projection_class)
    REFERENCES memory_retrieval_profiles (profile_id, projection_class),
  CONSTRAINT memory_vector_documents_512_tenant_continuity_fk
    FOREIGN KEY (tenant_id, continuity_id)
    REFERENCES continuity_spaces (tenant_id, id) ON DELETE CASCADE,
  CONSTRAINT memory_vector_documents_512_tenant_memory_fk
    FOREIGN KEY (tenant_id, continuity_id, memory_id)
    REFERENCES governed_memories (tenant_id, continuity_id, id) ON DELETE CASCADE
);

CREATE INDEX memory_vector_documents_512_scope_idx
  ON memory_vector_documents_512 (profile_id, tenant_id, continuity_id, memory_id);
CREATE INDEX memory_vector_documents_512_embedding_hnsw_idx
  ON memory_vector_documents_512 USING hnsw (embedding vector_cosine_ops);

ALTER TABLE memory_vector_documents_512 ENABLE ROW LEVEL SECURITY;

CREATE POLICY memory_vector_documents_512_tenant_isolation
  ON memory_vector_documents_512
  USING (tenant_id = NULLIF(current_setting('vermory.tenant_id', true), ''))
  WITH CHECK (tenant_id = NULLIF(current_setting('vermory.tenant_id', true), ''));

REVOKE ALL ON TABLE memory_vector_documents_512 FROM PUBLIC;

-- +goose Down

DROP POLICY IF EXISTS memory_vector_documents_512_tenant_isolation
  ON memory_vector_documents_512;
DROP TABLE IF EXISTS memory_vector_documents_512;

DELETE FROM memory_retrieval_runs
WHERE profile_id = 'siliconflow-bge-small-zh-512-v3';
DELETE FROM memory_projection_cursors
WHERE profile_id = 'siliconflow-bge-small-zh-512-v3';
DELETE FROM memory_retrieval_profiles
WHERE profile_id = 'siliconflow-bge-small-zh-512-v3';

ALTER TABLE memory_vector_documents
  DROP CONSTRAINT IF EXISTS memory_vector_documents_profile_class_fk,
  DROP CONSTRAINT IF EXISTS memory_vector_documents_projection_class_check,
  DROP COLUMN IF EXISTS projection_class;

ALTER TABLE memory_retrieval_profiles
  DROP CONSTRAINT IF EXISTS memory_retrieval_profiles_profile_class_uq,
  DROP CONSTRAINT IF EXISTS memory_retrieval_profiles_dimensions_class_check,
  DROP CONSTRAINT IF EXISTS memory_retrieval_profiles_projection_class_check,
  DROP COLUMN IF EXISTS projection_class,
  ADD CONSTRAINT memory_retrieval_profiles_dimensions_check
    CHECK (dimensions = 1024);
