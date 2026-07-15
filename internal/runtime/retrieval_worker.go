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

func (w *ProjectionWorker) RebuildCurrent(ctx context.Context) (ProjectionRebuildResult, error) {
	tenantCtx, err := withTenantContext(ctx, w.options.TenantID)
	if err != nil {
		return ProjectionRebuildResult{}, err
	}
	projectionSQL, err := retrievalProjectionSQLForClass(w.options.Profile.ProjectionClass)
	if err != nil {
		return ProjectionRebuildResult{}, err
	}
	connection, err := w.store.pool.Acquire(tenantCtx)
	if err != nil {
		return ProjectionRebuildResult{}, fmt.Errorf("acquire projection rebuild connection: %w", err)
	}
	defer connection.Release()

	lockKey1, lockKey2 := projectionAdvisoryLockKeys(w.options.Profile.ID, w.options.TenantID)
	var locked bool
	if err := connection.QueryRow(tenantCtx, `SELECT pg_try_advisory_lock($1, $2)`, lockKey1, lockKey2).Scan(&locked); err != nil {
		return ProjectionRebuildResult{}, fmt.Errorf("acquire projection rebuild lock: %w", err)
	}
	if !locked {
		status, statusErr := retrievalProjectionStatus(tenantCtx, connection, w.options.TenantID, w.options.Profile.ID)
		if statusErr != nil {
			return ProjectionRebuildResult{}, statusErr
		}
		return projectionRebuildResult(status, 0, 0, 0, status.LastEventID, "already_running", true), nil
	}
	defer func() {
		_, _ = connection.Exec(context.Background(), `SELECT pg_advisory_unlock($1, $2)`, lockKey1, lockKey2)
	}()

	var watermark int64
	if err := connection.QueryRow(tenantCtx, `
SELECT COALESCE(max(event_id), 0)
FROM memory_projection_events
WHERE tenant_id = $1`, w.options.TenantID).Scan(&watermark); err != nil {
		return ProjectionRebuildResult{}, fmt.Errorf("read projection rebuild watermark: %w", err)
	}
	tx, err := connection.Begin(tenantCtx)
	if err != nil {
		return ProjectionRebuildResult{}, fmt.Errorf("begin projection rebuild reset: %w", err)
	}
	defer tx.Rollback(tenantCtx)
	if _, err := tx.Exec(tenantCtx, projectionSQL.clearTenant, w.options.TenantID, w.options.Profile.ID); err != nil {
		return ProjectionRebuildResult{}, fmt.Errorf("clear projection rebuild vectors: %w", err)
	}
	if _, err := tx.Exec(tenantCtx, `
INSERT INTO memory_projection_cursors (
  tenant_id, profile_id, status, attempt_count, last_error_code, last_attempt_at, updated_at
) VALUES ($1, $2, 'running', 0, '', now(), now())
ON CONFLICT (tenant_id, profile_id) DO UPDATE SET
  status = 'running',
  last_error_code = '',
  last_attempt_at = now(),
  updated_at = now()`, w.options.TenantID, w.options.Profile.ID); err != nil {
		return ProjectionRebuildResult{}, fmt.Errorf("initialize projection rebuild cursor: %w", err)
	}
	if err := tx.Commit(tenantCtx); err != nil {
		return ProjectionRebuildResult{}, fmt.Errorf("commit projection rebuild reset: %w", err)
	}

	result := ProjectionRebuildResult{Watermark: watermark}
	lastMemoryID := ""
	for {
		page, err := loadProjectionSnapshotPage(
			tenantCtx, connection, w.options.TenantID, lastMemoryID, w.options.SnapshotPageSize,
		)
		if err != nil {
			return w.failRebuild(ctx, connection, result, "projection_read_error")
		}
		if len(page) == 0 {
			break
		}
		for _, memory := range page {
			result.Scanned++
			vector, err := w.embedder.Embed(tenantCtx, memory.Content)
			if err != nil {
				return w.failRebuild(ctx, connection, result, "embedding_unavailable")
			}
			if len(vector) != w.options.Profile.Dimensions {
				return w.failRebuild(ctx, connection, result, "embedding_dimension_mismatch")
			}
			hash := sha256.Sum256([]byte(memory.Content))
			mutation, err := connection.Exec(tenantCtx, projectionSQL.upsertSnapshot,
				w.options.Profile.ID,
				w.options.TenantID,
				memory.MemoryID,
				memory.ContinuityID,
				hex.EncodeToString(hash[:]),
				retrievalVectorLiteral(vector),
				memory.Content,
				memory.UpdatedAt,
			)
			if err != nil {
				return w.failRebuild(ctx, connection, result, "projection_write_error")
			}
			if mutation.RowsAffected() == 0 {
				result.SkippedChanged++
			} else {
				result.Projected++
			}
			lastMemoryID = memory.MemoryID
		}
	}
	if _, err := connection.Exec(tenantCtx, `
UPDATE memory_projection_cursors
SET last_event_id = GREATEST(last_event_id, $3),
    status = 'idle',
    attempt_count = attempt_count + 1,
    last_error_code = '',
    last_attempt_at = now(),
    updated_at = now()
WHERE tenant_id = $1 AND profile_id = $2`, w.options.TenantID, w.options.Profile.ID, watermark); err != nil {
		return w.failRebuild(ctx, connection, result, "projection_write_error")
	}
	status, err := retrievalProjectionStatus(tenantCtx, connection, w.options.TenantID, w.options.Profile.ID)
	if err != nil {
		return ProjectionRebuildResult{}, err
	}
	return projectionRebuildResult(status, result.Scanned, result.Projected, result.SkippedChanged, watermark, "", false), nil
}

type projectionSnapshotMemory struct {
	MemoryID     string
	ContinuityID string
	Content      string
	UpdatedAt    time.Time
}

func loadProjectionSnapshotPage(
	ctx context.Context,
	connection *pgxpool.Conn,
	tenantID string,
	lastMemoryID string,
	limit int,
) ([]projectionSnapshotMemory, error) {
	rows, err := connection.Query(ctx, `
SELECT id::text, continuity_id::text, content, updated_at
FROM governed_memories
WHERE tenant_id = $1
  AND memory_kind = 'fact'
  AND lifecycle_status = 'active'
  AND content <> '[redacted]'
  AND id > COALESCE(NULLIF($2, '')::uuid, '00000000-0000-0000-0000-000000000000'::uuid)
ORDER BY id
LIMIT $3`, tenantID, lastMemoryID, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	page := make([]projectionSnapshotMemory, 0, limit)
	for rows.Next() {
		var memory projectionSnapshotMemory
		if err := rows.Scan(&memory.MemoryID, &memory.ContinuityID, &memory.Content, &memory.UpdatedAt); err != nil {
			return nil, err
		}
		page = append(page, memory)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return page, nil
}

func (w *ProjectionWorker) failRebuild(
	ctx context.Context,
	connection *pgxpool.Conn,
	result ProjectionRebuildResult,
	code string,
) (ProjectionRebuildResult, error) {
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
		if ctxErr := ctx.Err(); ctxErr != nil {
			return ProjectionRebuildResult{}, ctxErr
		}
		return ProjectionRebuildResult{}, err
	}
	return projectionRebuildResult(
		status, result.Scanned, result.Projected, result.SkippedChanged, result.Watermark, code, false,
	), projectionRunError{code: code}
}

func projectionRebuildResult(
	status ProjectionStatus,
	scanned int,
	projected int,
	skippedChanged int,
	watermark int64,
	failureCode string,
	alreadyRunning bool,
) ProjectionRebuildResult {
	return ProjectionRebuildResult{
		Scanned:        scanned,
		Projected:      projected,
		SkippedChanged: skippedChanged,
		Watermark:      watermark,
		LastEventID:    status.LastEventID,
		LatestEventID:  status.LatestEventID,
		Lag:            status.Lag,
		Status:         status.Status,
		FailureCode:    failureCode,
		AlreadyRunning: alreadyRunning,
	}
}

func projectionAdvisoryLockKeys(profileID, tenantID string) (int32, int32) {
	digest := sha256.Sum256([]byte(profileID + "\x00" + tenantID))
	return int32(binary.BigEndian.Uint32(digest[:4])), int32(binary.BigEndian.Uint32(digest[4:8]))
}

func (w *ProjectionWorker) Run(ctx context.Context) error {
	for {
		if _, err := w.RunOnce(ctx); err != nil {
			if ctxErr := ctx.Err(); ctxErr != nil {
				return ctxErr
			}
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
	projectionSQL, err := retrievalProjectionSQLForClass(w.options.Profile.ProjectionClass)
	if err != nil {
		return projectionRunError{code: "projection_write_error"}
	}
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
		if _, err := tx.Exec(ctx, projectionSQL.deleteMemory,
			w.options.Profile.ID, event.TenantID, event.MemoryID); err != nil {
			return projectionRunError{code: "projection_write_error"}
		}
	} else {
		hash := sha256.Sum256([]byte(after.Content))
		if _, err := tx.Exec(ctx, projectionSQL.upsertMemory,
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
