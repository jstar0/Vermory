package runtime

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
)

func (s *Store) RetrievalProjectionStatus(ctx context.Context, tenantID, profileID string) (ProjectionStatus, error) {
	ctx, err := withTenantContext(ctx, tenantID)
	if err != nil {
		return ProjectionStatus{}, err
	}
	return retrievalProjectionStatus(ctx, s.pool, tenantID, profileID)
}

type retrievalStatusQuerier interface {
	QueryRow(context.Context, string, ...any) pgx.Row
}

func retrievalProjectionStatus(ctx context.Context, querier retrievalStatusQuerier, tenantID, profileID string) (ProjectionStatus, error) {
	if profileID != ProductionRetrievalProfileID {
		return ProjectionStatus{}, fmt.Errorf("unsupported retrieval profile")
	}
	status := ProjectionStatus{
		TenantID:  tenantID,
		ProfileID: profileID,
		Status:    "idle",
	}
	err := querier.QueryRow(ctx, `
SELECT last_event_id, status, attempt_count, last_error_code, last_attempt_at
FROM memory_projection_cursors
WHERE tenant_id = $1 AND profile_id = $2`, tenantID, profileID).Scan(
		&status.LastEventID,
		&status.Status,
		&status.AttemptCount,
		&status.LastErrorCode,
		&status.LastAttemptAt,
	)
	if err != nil && !errors.Is(err, pgx.ErrNoRows) {
		return ProjectionStatus{}, fmt.Errorf("read retrieval projection cursor: %w", err)
	}
	if err := querier.QueryRow(ctx, `
SELECT COALESCE(max(event_id), 0)
FROM memory_projection_events
WHERE tenant_id = $1`, tenantID).Scan(&status.LatestEventID); err != nil {
		return ProjectionStatus{}, fmt.Errorf("read latest retrieval projection event: %w", err)
	}
	if err := querier.QueryRow(ctx, `
SELECT count(*)
FROM memory_vector_documents
WHERE tenant_id = $1 AND profile_id = $2`, tenantID, profileID).Scan(&status.VectorCount); err != nil {
		return ProjectionStatus{}, fmt.Errorf("count retrieval vector documents: %w", err)
	}
	status.Lag = status.LatestEventID - status.LastEventID
	if status.Lag < 0 {
		status.Lag = 0
	}
	return status, nil
}

func (s *Store) ResetVectorProjection(ctx context.Context, tenantID, profileID string) error {
	ctx, err := withTenantContext(ctx, tenantID)
	if err != nil {
		return err
	}
	if profileID != ProductionRetrievalProfileID {
		return fmt.Errorf("unsupported retrieval profile")
	}
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("begin vector projection reset: %w", err)
	}
	defer tx.Rollback(ctx)
	if _, err := tx.Exec(ctx, `
DELETE FROM memory_vector_documents
WHERE tenant_id = $1 AND profile_id = $2`, tenantID, profileID); err != nil {
		return fmt.Errorf("clear vector projection: %w", err)
	}
	if _, err := tx.Exec(ctx, `
INSERT INTO memory_projection_cursors (
  tenant_id, profile_id, last_event_id, status, attempt_count,
  last_error_code, last_attempt_at, updated_at
) VALUES ($1, $2, 0, 'idle', 0, '', NULL, now())
ON CONFLICT (tenant_id, profile_id) DO UPDATE SET
  last_event_id = 0,
  status = 'idle',
  attempt_count = 0,
  last_error_code = '',
  last_attempt_at = NULL,
  updated_at = now()`, tenantID, profileID); err != nil {
		return fmt.Errorf("reset vector projection cursor: %w", err)
	}
	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("commit vector projection reset: %w", err)
	}
	return nil
}

func (s *Store) searchActiveVectorMemory(ctx context.Context, tenantID string, continuityIDs []string, queryVector []float32, limit int, profileID string) ([]Memory, error) {
	ctx, err := withTenantContext(ctx, tenantID)
	if err != nil {
		return nil, err
	}
	candidateLimit := limit * 4
	if candidateLimit < 20 {
		candidateLimit = 20
	}
	if candidateLimit > 100 {
		candidateLimit = 100
	}
	rows, err := s.pool.Query(ctx, `
WITH candidates AS (
  SELECT document.memory_id, document.content_sha256,
         document.embedding <=> $4::vector AS distance
  FROM memory_vector_documents document
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
ORDER BY candidate.distance, memory.id
LIMIT $6`, profileID, tenantID, continuityIDs, retrievalVectorLiteral(queryVector), candidateLimit, limit)
	if err != nil {
		return nil, fmt.Errorf("search active vector memory: %w", err)
	}
	defer rows.Close()
	memories := make([]Memory, 0)
	for rows.Next() {
		var memory Memory
		if err := rows.Scan(&memory.ID, &memory.Content); err != nil {
			return nil, fmt.Errorf("scan active vector memory: %w", err)
		}
		memories = append(memories, memory)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate active vector memory: %w", err)
	}
	return memories, nil
}

func (s *Store) checkRetrievalAuditReplay(ctx context.Context, tenantID, operationID, fingerprint string) (string, error) {
	ctx, err := withTenantContext(ctx, tenantID)
	if err != nil {
		return "", err
	}
	var auditID, existingFingerprint string
	err = s.pool.QueryRow(ctx, `
SELECT id::text, request_fingerprint
FROM memory_retrieval_runs
WHERE tenant_id = $1 AND operation_id = $2`, tenantID, operationID).Scan(&auditID, &existingFingerprint)
	if errors.Is(err, pgx.ErrNoRows) {
		return "", nil
	}
	if err != nil {
		return "", fmt.Errorf("lookup retrieval audit replay: %w", err)
	}
	if existingFingerprint != fingerprint {
		return "", fmt.Errorf("retrieval audit operation conflict")
	}
	return auditID, nil
}

func (s *Store) recordRetrievalAudit(ctx context.Context, input retrievalAuditInput) (string, error) {
	ctx, err := withTenantContext(ctx, input.TenantID)
	if err != nil {
		return "", err
	}
	var auditID string
	err = s.pool.QueryRow(ctx, `
INSERT INTO memory_retrieval_runs (
  tenant_id, primary_continuity_id, continuity_ids, operation_id,
  request_fingerprint, requested_mode, effective_mode, profile_id,
  query_sha256, lexical_memory_ids, vector_memory_ids, delivered_memory_ids,
  projection_current, degraded, failure_code, lexical_latency_ms, vector_latency_ms
) VALUES (
  $1, $2::uuid, $3::uuid[], $4,
  $5, $6, $7, $8,
  $9, $10::uuid[], $11::uuid[], $12::uuid[],
  $13, $14, $15, $16, $17
)
ON CONFLICT (tenant_id, operation_id) DO UPDATE SET
  operation_id = memory_retrieval_runs.operation_id
WHERE memory_retrieval_runs.request_fingerprint = EXCLUDED.request_fingerprint
RETURNING id::text`,
		input.TenantID,
		input.PrimaryContinuityID,
		input.ContinuityIDs,
		input.OperationID,
		input.RequestFingerprint,
		input.RequestedMode,
		input.EffectiveMode,
		input.ProfileID,
		input.QuerySHA256,
		input.LexicalMemoryIDs,
		input.VectorMemoryIDs,
		input.DeliveredMemoryIDs,
		input.ProjectionCurrent,
		input.Degraded,
		input.FailureCode,
		durationMilliseconds(input.LexicalLatency),
		durationMilliseconds(input.VectorLatency),
	).Scan(&auditID)
	if errors.Is(err, pgx.ErrNoRows) {
		return "", fmt.Errorf("retrieval audit operation conflict")
	}
	if err != nil {
		return "", fmt.Errorf("record retrieval audit: %w", err)
	}
	return auditID, nil
}

func durationMilliseconds(duration time.Duration) int {
	milliseconds := duration.Milliseconds()
	if milliseconds < 0 {
		return 0
	}
	if milliseconds > int64(^uint(0)>>1) {
		return int(^uint(0) >> 1)
	}
	return int(milliseconds)
}
