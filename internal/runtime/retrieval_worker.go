package runtime

import (
	"context"
	"crypto/sha256"
	"encoding/binary"
	"encoding/hex"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type ProjectionWorker struct {
	store    *Store
	embedder Embedder
	options  ProjectionWorkerOptions
}

func NewProjectionWorker(store *Store, embedder Embedder, options ProjectionWorkerOptions) (*ProjectionWorker, error) {
	if store == nil {
		return nil, fmt.Errorf("projection worker store is required")
	}
	if embedder == nil {
		return nil, fmt.Errorf("projection worker embedder is required")
	}
	if err := options.normalize(); err != nil {
		return nil, err
	}
	return &ProjectionWorker{store: store, embedder: embedder, options: options}, nil
}

func (w *ProjectionWorker) RunOnce(ctx context.Context) (ProjectionRunResult, error) {
	tenantCtx, err := withTenantContext(ctx, w.options.TenantID)
	if err != nil {
		return ProjectionRunResult{}, err
	}
	connection, err := w.store.pool.Acquire(tenantCtx)
	if err != nil {
		return ProjectionRunResult{}, fmt.Errorf("acquire projection worker connection: %w", err)
	}
	defer connection.Release()

	lockKey1, lockKey2 := projectionAdvisoryLockKeys(w.options.Profile.ID, w.options.TenantID)
	var locked bool
	if err := connection.QueryRow(tenantCtx, `SELECT pg_try_advisory_lock($1, $2)`, lockKey1, lockKey2).Scan(&locked); err != nil {
		return ProjectionRunResult{}, fmt.Errorf("acquire projection worker lock: %w", err)
	}
	if !locked {
		status, statusErr := retrievalProjectionStatus(tenantCtx, connection, w.options.TenantID, w.options.Profile.ID)
		if statusErr != nil {
			return ProjectionRunResult{}, statusErr
		}
		return projectionResult(status, 0, "already_running", true), nil
	}
	defer func() {
		_, _ = connection.Exec(context.Background(), `SELECT pg_advisory_unlock($1, $2)`, lockKey1, lockKey2)
	}()

	if err := ensureProjectionCursor(tenantCtx, connection, w.options.TenantID, w.options.Profile.ID); err != nil {
		return ProjectionRunResult{}, err
	}
	processed := 0
	for processed < w.options.BatchSize {
		event, found, err := nextProjectionEvent(tenantCtx, connection, w.options.TenantID, w.options.Profile.ID)
		if err != nil {
			return w.fail(ctx, connection, processed, "projection_read_error")
		}
		if !found {
			break
		}
		if err := w.processEvent(tenantCtx, connection, event); err != nil {
			var coded projectionRunError
			if errors.As(err, &coded) {
				return w.fail(ctx, connection, processed, coded.code)
			}
			return w.fail(ctx, connection, processed, "projection_write_error")
		}
		processed++
	}
	if _, err := connection.Exec(tenantCtx, `
UPDATE memory_projection_cursors
SET status = 'idle', last_error_code = '', updated_at = now()
WHERE tenant_id = $1 AND profile_id = $2`, w.options.TenantID, w.options.Profile.ID); err != nil {
		return ProjectionRunResult{}, fmt.Errorf("finish projection worker cursor: %w", err)
	}
	status, err := retrievalProjectionStatus(tenantCtx, connection, w.options.TenantID, w.options.Profile.ID)
	if err != nil {
		return ProjectionRunResult{}, err
	}
	return projectionResult(status, processed, "", false), nil
}

func projectionAdvisoryLockKeys(profileID, tenantID string) (int32, int32) {
	digest := sha256.Sum256([]byte(profileID + "\x00" + tenantID))
	return int32(binary.BigEndian.Uint32(digest[:4])), int32(binary.BigEndian.Uint32(digest[4:8]))
}

func (w *ProjectionWorker) Run(ctx context.Context) error {
	for {
		if _, err := w.RunOnce(ctx); err != nil {
			return err
		}
		timer := time.NewTimer(w.options.PollInterval)
		select {
		case <-ctx.Done():
			timer.Stop()
			return ctx.Err()
		case <-timer.C:
		}
	}
}

func (w *ProjectionWorker) processEvent(ctx context.Context, connection *pgxpool.Conn, event ProjectionEvent) error {
	before, err := loadProjectionMemory(ctx, connection, event.TenantID, event.MemoryID)
	if err != nil {
		return projectionRunError{code: "projection_read_error"}
	}
	shouldProject := before.Exists && before.Kind == "fact" && before.Status == "active" && before.Content != "[redacted]"
	var vector []float32
	if shouldProject {
		vector, err = w.embedder.Embed(ctx, before.Content)
		if err != nil {
			return projectionRunError{code: "embedding_unavailable"}
		}
		if len(vector) != w.options.Profile.Dimensions {
			return projectionRunError{code: "embedding_dimension_mismatch"}
		}
	}

	tx, err := connection.Begin(ctx)
	if err != nil {
		return projectionRunError{code: "projection_write_error"}
	}
	defer tx.Rollback(ctx)
	after, err := loadProjectionMemoryTx(ctx, tx, event.TenantID, event.MemoryID)
	if err != nil {
		return projectionRunError{code: "projection_read_error"}
	}
	if before.Exists != after.Exists || before.ContinuityID != after.ContinuityID || before.Kind != after.Kind || before.Status != after.Status || before.Content != after.Content || !before.UpdatedAt.Equal(after.UpdatedAt) {
		return projectionRunError{code: "authority_changed"}
	}
	if !shouldProject {
		if _, err := tx.Exec(ctx, `
DELETE FROM memory_vector_documents
WHERE profile_id = $1 AND tenant_id = $2 AND memory_id = $3::uuid`,
			w.options.Profile.ID, event.TenantID, event.MemoryID); err != nil {
			return projectionRunError{code: "projection_write_error"}
		}
	} else {
		hash := sha256.Sum256([]byte(after.Content))
		if _, err := tx.Exec(ctx, `
INSERT INTO memory_vector_documents (
  profile_id, tenant_id, continuity_id, memory_id, content_sha256, embedding, updated_at
) VALUES ($1, $2, $3::uuid, $4::uuid, $5, $6::vector, now())
ON CONFLICT (profile_id, tenant_id, memory_id) DO UPDATE SET
  continuity_id = EXCLUDED.continuity_id,
  content_sha256 = EXCLUDED.content_sha256,
  embedding = EXCLUDED.embedding,
  updated_at = now()`,
			w.options.Profile.ID,
			event.TenantID,
			after.ContinuityID,
			event.MemoryID,
			hex.EncodeToString(hash[:]),
			retrievalVectorLiteral(vector),
		); err != nil {
			return projectionRunError{code: "projection_write_error"}
		}
	}
	if _, err := tx.Exec(ctx, `
UPDATE memory_projection_cursors
SET last_event_id = $3,
    status = 'running',
    attempt_count = attempt_count + 1,
    last_error_code = '',
    last_attempt_at = now(),
    updated_at = now()
WHERE tenant_id = $1 AND profile_id = $2`, event.TenantID, w.options.Profile.ID, event.EventID); err != nil {
		return projectionRunError{code: "projection_write_error"}
	}
	if err := tx.Commit(ctx); err != nil {
		return projectionRunError{code: "projection_write_error"}
	}
	return nil
}

func (w *ProjectionWorker) fail(ctx context.Context, connection *pgxpool.Conn, processed int, code string) (ProjectionRunResult, error) {
	tenantCtx, tenantErr := withTenantContext(ctx, w.options.TenantID)
	if tenantErr == nil {
		_, _ = connection.Exec(tenantCtx, `
UPDATE memory_projection_cursors
SET status = 'failed',
    attempt_count = attempt_count + 1,
    last_error_code = $3,
    last_attempt_at = now(),
    updated_at = now()
WHERE tenant_id = $1 AND profile_id = $2`, w.options.TenantID, w.options.Profile.ID, code)
	}
	status, err := retrievalProjectionStatus(tenantCtx, connection, w.options.TenantID, w.options.Profile.ID)
	if err != nil {
		return ProjectionRunResult{}, err
	}
	return projectionResult(status, processed, code, false), projectionRunError{code: code}
}

func ensureProjectionCursor(ctx context.Context, connection *pgxpool.Conn, tenantID, profileID string) error {
	_, err := connection.Exec(ctx, `
INSERT INTO memory_projection_cursors (tenant_id, profile_id, status, last_attempt_at)
VALUES ($1, $2, 'running', now())
ON CONFLICT (tenant_id, profile_id) DO UPDATE SET
  status = 'running',
  last_attempt_at = now(),
  updated_at = now()`, tenantID, profileID)
	if err != nil {
		return fmt.Errorf("initialize projection cursor: %w", err)
	}
	return nil
}

func nextProjectionEvent(ctx context.Context, connection *pgxpool.Conn, tenantID, profileID string) (ProjectionEvent, bool, error) {
	var event ProjectionEvent
	err := connection.QueryRow(ctx, `
SELECT event.event_id, event.tenant_id, event.continuity_id::text,
       event.memory_id::text, event.desired_state, event.authority_version
FROM memory_projection_events event
JOIN memory_projection_cursors cursor
  ON cursor.tenant_id = event.tenant_id AND cursor.profile_id = $2
WHERE event.tenant_id = $1 AND event.event_id > cursor.last_event_id
ORDER BY event.event_id
LIMIT 1`, tenantID, profileID).Scan(
		&event.EventID,
		&event.TenantID,
		&event.ContinuityID,
		&event.MemoryID,
		&event.DesiredState,
		&event.AuthorityVersion,
	)
	if err == pgx.ErrNoRows {
		return ProjectionEvent{}, false, nil
	}
	if err != nil {
		return ProjectionEvent{}, false, err
	}
	return event, true, nil
}

type projectionMemory struct {
	Exists       bool
	ContinuityID string
	Kind         string
	Status       string
	Content      string
	UpdatedAt    time.Time
}

func loadProjectionMemory(ctx context.Context, connection *pgxpool.Conn, tenantID, memoryID string) (projectionMemory, error) {
	var memory projectionMemory
	err := connection.QueryRow(ctx, `
SELECT continuity_id::text, memory_kind, lifecycle_status, content, updated_at
FROM governed_memories
WHERE tenant_id = $1 AND id = $2::uuid`, tenantID, memoryID).Scan(
		&memory.ContinuityID,
		&memory.Kind,
		&memory.Status,
		&memory.Content,
		&memory.UpdatedAt,
	)
	if err == pgx.ErrNoRows {
		return memory, nil
	}
	if err != nil {
		return projectionMemory{}, err
	}
	memory.Exists = true
	return memory, nil
}

func loadProjectionMemoryTx(ctx context.Context, tx pgx.Tx, tenantID, memoryID string) (projectionMemory, error) {
	var memory projectionMemory
	err := tx.QueryRow(ctx, `
SELECT continuity_id::text, memory_kind, lifecycle_status, content, updated_at
FROM governed_memories
WHERE tenant_id = $1 AND id = $2::uuid
FOR SHARE`, tenantID, memoryID).Scan(
		&memory.ContinuityID,
		&memory.Kind,
		&memory.Status,
		&memory.Content,
		&memory.UpdatedAt,
	)
	if err == pgx.ErrNoRows {
		return memory, nil
	}
	if err != nil {
		return projectionMemory{}, err
	}
	memory.Exists = true
	return memory, nil
}

func projectionResult(status ProjectionStatus, processed int, failureCode string, alreadyRunning bool) ProjectionRunResult {
	return ProjectionRunResult{
		Processed:      processed,
		LastEventID:    status.LastEventID,
		LatestEventID:  status.LatestEventID,
		Lag:            status.Lag,
		Status:         status.Status,
		FailureCode:    failureCode,
		AlreadyRunning: alreadyRunning,
	}
}

func retrievalVectorLiteral(vector []float32) string {
	parts := make([]string, len(vector))
	for index, value := range vector {
		parts[index] = strconv.FormatFloat(float64(value), 'g', -1, 32)
	}
	return "[" + strings.Join(parts, ",") + "]"
}
