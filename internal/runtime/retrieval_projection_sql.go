package runtime

import "fmt"

type retrievalProjectionSQL struct {
	count          string
	clearTenant    string
	search         string
	deleteMemory   string
	upsertMemory   string
	upsertSnapshot string
}

func retrievalProjectionSQLForProfile(profileID string) (retrievalProjectionSQL, error) {
	spec, ok := SupportedRetrievalProfile(profileID)
	if !ok {
		return retrievalProjectionSQL{}, fmt.Errorf("unsupported retrieval profile")
	}
	return retrievalProjectionSQLForClass(spec.ProjectionClass)
}

func retrievalProjectionSQLForClass(class ProjectionClass) (retrievalProjectionSQL, error) {
	switch class {
	case ProjectionClass1024:
		return retrievalProjectionSQL{
			count:          projectionCount1024SQL,
			clearTenant:    projectionClearTenant1024SQL,
			search:         projectionSearch1024SQL,
			deleteMemory:   projectionDeleteMemory1024SQL,
			upsertMemory:   projectionUpsertMemory1024SQL,
			upsertSnapshot: projectionUpsertSnapshot1024SQL,
		}, nil
	case ProjectionClass512:
		return retrievalProjectionSQL{
			count:          projectionCount512SQL,
			clearTenant:    projectionClearTenant512SQL,
			search:         projectionSearch512SQL,
			deleteMemory:   projectionDeleteMemory512SQL,
			upsertMemory:   projectionUpsertMemory512SQL,
			upsertSnapshot: projectionUpsertSnapshot512SQL,
		}, nil
	default:
		return retrievalProjectionSQL{}, fmt.Errorf("unsupported retrieval projection class %q", class)
	}
}

const projectionCount1024SQL = `
SELECT count(*)
FROM memory_vector_documents
WHERE tenant_id = $1 AND profile_id = $2`

const projectionCount512SQL = `
SELECT count(*)
FROM memory_vector_documents_512
WHERE tenant_id = $1 AND profile_id = $2`

const projectionClearTenant1024SQL = `
DELETE FROM memory_vector_documents
WHERE tenant_id = $1 AND profile_id = $2`

const projectionClearTenant512SQL = `
DELETE FROM memory_vector_documents_512
WHERE tenant_id = $1 AND profile_id = $2`

const projectionSearch1024SQL = `
WITH candidates AS (
  SELECT document.memory_id, document.content_sha256,
         document.embedding <=> $4::vector AS distance,
         CASE origin.observation_kind
           WHEN 'user_correction' THEN 4
           WHEN 'user_confirmation' THEN 4
           WHEN 'source_update' THEN 3
           WHEN 'bridge_promote' THEN 2
           ELSE 1
         END AS authority_rank
  FROM memory_vector_documents document
  JOIN governed_memories memory
    ON memory.tenant_id = $2 AND memory.id = document.memory_id
  JOIN observations origin
    ON origin.tenant_id = $2 AND origin.id = memory.origin_observation_id
  WHERE document.profile_id = $1
    AND document.tenant_id = $2
    AND document.continuity_id = ANY($3::uuid[])
  ORDER BY document.embedding <=> $4::vector, document.memory_id
  LIMIT $5
)
SELECT memory.id::text, memory.content
FROM candidates candidate
JOIN governed_memories memory
  ON memory.tenant_id = $2 AND memory.id = candidate.memory_id
WHERE memory.continuity_id = ANY($3::uuid[])
  AND memory.memory_kind = 'fact'
  AND memory.lifecycle_status = 'active'
  AND memory.content <> '[redacted]'
  AND encode(digest(convert_to(memory.content, 'UTF8'), 'sha256'), 'hex') = candidate.content_sha256
ORDER BY candidate.distance, candidate.authority_rank DESC, memory.id
LIMIT $6`

const projectionSearch512SQL = `
WITH candidates AS (
  SELECT document.memory_id, document.content_sha256,
         document.embedding <=> $4::vector AS distance,
         CASE origin.observation_kind
           WHEN 'user_correction' THEN 4
           WHEN 'user_confirmation' THEN 4
           WHEN 'source_update' THEN 3
           WHEN 'bridge_promote' THEN 2
           ELSE 1
         END AS authority_rank
  FROM memory_vector_documents_512 document
  JOIN governed_memories memory
    ON memory.tenant_id = $2 AND memory.id = document.memory_id
  JOIN observations origin
    ON origin.tenant_id = $2 AND origin.id = memory.origin_observation_id
  WHERE document.profile_id = $1
    AND document.tenant_id = $2
    AND document.continuity_id = ANY($3::uuid[])
  ORDER BY document.embedding <=> $4::vector, document.memory_id
  LIMIT $5
)
SELECT memory.id::text, memory.content
FROM candidates candidate
JOIN governed_memories memory
  ON memory.tenant_id = $2 AND memory.id = candidate.memory_id
WHERE memory.continuity_id = ANY($3::uuid[])
  AND memory.memory_kind = 'fact'
  AND memory.lifecycle_status = 'active'
  AND memory.content <> '[redacted]'
  AND encode(digest(convert_to(memory.content, 'UTF8'), 'sha256'), 'hex') = candidate.content_sha256
ORDER BY candidate.distance, candidate.authority_rank DESC, memory.id
LIMIT $6`

const projectionDeleteMemory1024SQL = `
DELETE FROM memory_vector_documents
WHERE profile_id = $1 AND tenant_id = $2 AND memory_id = $3::uuid`

const projectionDeleteMemory512SQL = `
DELETE FROM memory_vector_documents_512
WHERE profile_id = $1 AND tenant_id = $2 AND memory_id = $3::uuid`

const projectionUpsertMemory1024SQL = `
INSERT INTO memory_vector_documents (
  profile_id, tenant_id, continuity_id, memory_id, content_sha256, embedding, updated_at
) VALUES ($1, $2, $3::uuid, $4::uuid, $5, $6::vector, now())
ON CONFLICT (profile_id, tenant_id, memory_id) DO UPDATE SET
  continuity_id = EXCLUDED.continuity_id,
  content_sha256 = EXCLUDED.content_sha256,
  embedding = EXCLUDED.embedding,
  updated_at = now()`

const projectionUpsertMemory512SQL = `
INSERT INTO memory_vector_documents_512 (
  profile_id, tenant_id, continuity_id, memory_id, content_sha256, embedding, updated_at
) VALUES ($1, $2, $3::uuid, $4::uuid, $5, $6::vector, now())
ON CONFLICT (profile_id, tenant_id, memory_id) DO UPDATE SET
  continuity_id = EXCLUDED.continuity_id,
  content_sha256 = EXCLUDED.content_sha256,
  embedding = EXCLUDED.embedding,
  updated_at = now()`

const projectionUpsertSnapshot1024SQL = `
INSERT INTO memory_vector_documents (
  profile_id, tenant_id, continuity_id, memory_id, content_sha256, embedding, updated_at
)
SELECT $1, memory.tenant_id, memory.continuity_id, memory.id, $5, $6::vector, now()
FROM governed_memories memory
WHERE memory.tenant_id = $2
  AND memory.id = $3::uuid
  AND memory.continuity_id = $4::uuid
  AND memory.memory_kind = 'fact'
  AND memory.lifecycle_status = 'active'
  AND memory.content = $7
  AND memory.updated_at = $8
  AND memory.content <> '[redacted]'
ON CONFLICT (profile_id, tenant_id, memory_id) DO UPDATE SET
  continuity_id = EXCLUDED.continuity_id,
  content_sha256 = EXCLUDED.content_sha256,
  embedding = EXCLUDED.embedding,
  updated_at = now()`

const projectionUpsertSnapshot512SQL = `
INSERT INTO memory_vector_documents_512 (
  profile_id, tenant_id, continuity_id, memory_id, content_sha256, embedding, updated_at
)
SELECT $1, memory.tenant_id, memory.continuity_id, memory.id, $5, $6::vector, now()
FROM governed_memories memory
WHERE memory.tenant_id = $2
  AND memory.id = $3::uuid
  AND memory.continuity_id = $4::uuid
  AND memory.memory_kind = 'fact'
  AND memory.lifecycle_status = 'active'
  AND memory.content = $7
  AND memory.updated_at = $8
  AND memory.content <> '[redacted]'
ON CONFLICT (profile_id, tenant_id, memory_id) DO UPDATE SET
  continuity_id = EXCLUDED.continuity_id,
  content_sha256 = EXCLUDED.content_sha256,
  embedding = EXCLUDED.embedding,
  updated_at = now()`
