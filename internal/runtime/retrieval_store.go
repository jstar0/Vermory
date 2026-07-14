package runtime

import (
	"context"
	"errors"
	"fmt"

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
