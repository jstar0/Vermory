package runtime

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	_ "github.com/jackc/pgx/v5/stdlib"
	"github.com/pressly/goose/v3"
)

type ResolutionStatus string

const (
	ResolutionResolved          ResolutionStatus = "resolved"
	ResolutionNeedsConfirmation ResolutionStatus = "needs_confirmation"
)

type WorkspaceResolution struct {
	Status       ResolutionStatus
	ContinuityID string
	RepoRoot     string
}

type ObservationReceipt struct {
	ObservationID string
	Replayed      bool
}

type Memory struct {
	ID      string
	Content string
}

type GovernedMemory struct {
	ID                 string `json:"id"`
	LifecycleStatus    string `json:"lifecycle_status"`
	Content            string `json:"content"`
	SupersedesMemoryID string `json:"supersedes_memory_id,omitempty"`
}

type DeliveryReceipt struct {
	DeliveryID string
	Context    string
	Replayed   bool
}

type MemoryReceipt struct {
	MemoryID string
	Status   string
	Replayed bool
}

type GovernedObservationReceipt struct {
	Observation ObservationReceipt
	Memory      MemoryReceipt
}

type Store struct {
	pool        *pgxpool.Pool
	databaseURL string
}

func OpenStore(ctx context.Context, databaseURL string) (*Store, error) {
	if strings.TrimSpace(databaseURL) == "" {
		return nil, errors.New("runtime store: database URL is required")
	}
	pool, err := pgxpool.New(ctx, databaseURL)
	if err != nil {
		return nil, fmt.Errorf("open runtime store: %w", err)
	}
	return &Store{pool: pool, databaseURL: databaseURL}, nil
}

func (s *Store) Close() {
	s.pool.Close()
}

func (s *Store) Migrate(ctx context.Context) error {
	db, err := sql.Open("pgx", s.databaseURL)
	if err != nil {
		return fmt.Errorf("open migration database: %w", err)
	}
	defer db.Close()
	if err := goose.SetDialect("postgres"); err != nil {
		return fmt.Errorf("set migration dialect: %w", err)
	}
	return goose.UpContext(ctx, db, migrationDir())
}

func (s *Store) ResetForTest(ctx context.Context) error {
	_, err := s.pool.Exec(ctx, `
TRUNCATE memory_search_documents, memory_deliveries, governed_memories,
  conversation_turns, observations, conversation_bindings, continuity_bindings, continuity_spaces CASCADE`)
	if err != nil {
		return fmt.Errorf("reset runtime store: %w", err)
	}
	return nil
}

func (s *Store) ResolveWorkspace(ctx context.Context, tenantID string, anchor WorkspaceAnchor) (WorkspaceResolution, error) {
	anchor, err := anchor.Normalized()
	if err != nil {
		return WorkspaceResolution{}, err
	}
	if anchor.ExplicitBindingID != "" {
		var continuityID string
		err := s.pool.QueryRow(ctx, `
SELECT id::text FROM continuity_spaces
WHERE id::text = $1 AND tenant_id = $2 AND continuity_line = 'workspace' AND state = 'active'`,
			anchor.ExplicitBindingID, tenantID).Scan(&continuityID)
		if err == nil {
			return WorkspaceResolution{Status: ResolutionResolved, ContinuityID: continuityID, RepoRoot: anchor.RepoRoot}, nil
		}
		if !errors.Is(err, pgx.ErrNoRows) {
			return WorkspaceResolution{}, fmt.Errorf("resolve explicit workspace binding: %w", err)
		}
	}

	var continuityID string
	err = s.pool.QueryRow(ctx, `
SELECT b.continuity_id::text
FROM continuity_bindings b
JOIN continuity_spaces c ON c.id = b.continuity_id
WHERE b.tenant_id = $1 AND b.repo_root = $2 AND b.binding_state = 'confirmed'
  AND c.continuity_line = 'workspace' AND c.state = 'active'`, tenantID, anchor.RepoRoot).Scan(&continuityID)
	if err == nil {
		return WorkspaceResolution{Status: ResolutionResolved, ContinuityID: continuityID, RepoRoot: anchor.RepoRoot}, nil
	}
	if errors.Is(err, pgx.ErrNoRows) {
		return WorkspaceResolution{Status: ResolutionNeedsConfirmation, RepoRoot: anchor.RepoRoot}, nil
	}
	return WorkspaceResolution{}, fmt.Errorf("resolve workspace binding: %w", err)
}

func (s *Store) ConfirmWorkspaceBinding(ctx context.Context, tenantID, repoRoot string) (string, error) {
	anchor, err := (WorkspaceAnchor{RepoRoot: repoRoot}).Normalized()
	if err != nil {
		return "", err
	}
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return "", fmt.Errorf("begin workspace binding: %w", err)
	}
	defer tx.Rollback(ctx)

	var existingID string
	err = tx.QueryRow(ctx, `
SELECT continuity_id::text FROM continuity_bindings
WHERE tenant_id = $1 AND repo_root = $2 AND binding_state = 'confirmed'`, tenantID, anchor.RepoRoot).Scan(&existingID)
	if err == nil {
		if err := tx.Commit(ctx); err != nil {
			return "", fmt.Errorf("commit existing workspace binding: %w", err)
		}
		return existingID, nil
	}
	if !errors.Is(err, pgx.ErrNoRows) {
		return "", fmt.Errorf("lookup workspace binding: %w", err)
	}

	var continuityID string
	if err := tx.QueryRow(ctx, `
INSERT INTO continuity_spaces (tenant_id, continuity_line, state)
VALUES ($1, 'workspace', 'active')
RETURNING id::text`, tenantID).Scan(&continuityID); err != nil {
		return "", fmt.Errorf("create workspace continuity: %w", err)
	}
	if _, err := tx.Exec(ctx, `
INSERT INTO continuity_bindings (continuity_id, tenant_id, repo_root, binding_state)
VALUES ($1::uuid, $2, $3, 'confirmed')`, continuityID, tenantID, anchor.RepoRoot); err != nil {
		return "", fmt.Errorf("create workspace binding: %w", err)
	}
	if err := tx.Commit(ctx); err != nil {
		return "", fmt.Errorf("commit workspace binding: %w", err)
	}
	return continuityID, nil
}

func (s *Store) CommitObservation(ctx context.Context, tenantID, continuityID string, request CommitObservationRequest) (ObservationReceipt, error) {
	if err := request.Validate(); err != nil {
		return ObservationReceipt{}, err
	}
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return ObservationReceipt{}, fmt.Errorf("begin observation: %w", err)
	}
	defer tx.Rollback(ctx)

	receipt, err := commitObservationTx(ctx, tx, tenantID, continuityID, request)
	if err != nil {
		return ObservationReceipt{}, err
	}
	if err := tx.Commit(ctx); err != nil {
		return ObservationReceipt{}, fmt.Errorf("commit observation: %w", err)
	}
	return receipt, nil
}

func (s *Store) CommitGovernedObservation(ctx context.Context, tenantID, continuityID string, request CommitObservationRequest) (GovernedObservationReceipt, error) {
	if err := request.Validate(); err != nil {
		return GovernedObservationReceipt{}, err
	}
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return GovernedObservationReceipt{}, fmt.Errorf("begin governed observation: %w", err)
	}
	defer tx.Rollback(ctx)

	observation, err := commitObservationTx(ctx, tx, tenantID, continuityID, request)
	if err != nil {
		return GovernedObservationReceipt{}, err
	}
	var memory MemoryReceipt
	if request.Kind == ObservationKindForgetRequest {
		if err := deleteMemoryTx(ctx, tx, tenantID, continuityID, request.TargetMemoryID); err != nil {
			return GovernedObservationReceipt{}, err
		}
		memory = MemoryReceipt{MemoryID: request.TargetMemoryID, Status: "deleted", Replayed: observation.Replayed}
	} else {
		memory, err = governObservationTx(ctx, tx, tenantID, continuityID, observation.ObservationID, request)
		if err != nil {
			return GovernedObservationReceipt{}, err
		}
	}
	if err := tx.Commit(ctx); err != nil {
		return GovernedObservationReceipt{}, fmt.Errorf("commit governed observation: %w", err)
	}
	return GovernedObservationReceipt{Observation: observation, Memory: memory}, nil
}

func commitObservationTx(ctx context.Context, tx pgx.Tx, tenantID, continuityID string, request CommitObservationRequest) (ObservationReceipt, error) {
	var validContinuity bool
	if err := tx.QueryRow(ctx, `
SELECT EXISTS (
  SELECT 1 FROM continuity_spaces
  WHERE id = $1::uuid AND tenant_id = $2
    AND continuity_line IN ('workspace', 'conversation') AND state = 'active'
)`, continuityID, tenantID).Scan(&validContinuity); err != nil {
		return ObservationReceipt{}, fmt.Errorf("check observation continuity: %w", err)
	}
	if !validContinuity {
		return ObservationReceipt{}, fmt.Errorf("continuity is not active for this tenant")
	}

	var existingID, existingContinuityID, existingKind, existingContent, existingSourceRef string
	err := tx.QueryRow(ctx, `
SELECT id::text, continuity_id::text, observation_kind, content, source_ref
FROM observations
WHERE tenant_id = $1 AND operation_id = $2`, tenantID, request.OperationID).Scan(
		&existingID,
		&existingContinuityID,
		&existingKind,
		&existingContent,
		&existingSourceRef,
	)
	if err == nil {
		if existingContinuityID != continuityID ||
			existingKind != string(request.Kind) ||
			existingContent != request.Content ||
			existingSourceRef != request.SourceRef {
			return ObservationReceipt{}, fmt.Errorf("operation_id is already bound to another logical observation")
		}
		return ObservationReceipt{ObservationID: existingID, Replayed: true}, nil
	}
	if !errors.Is(err, pgx.ErrNoRows) {
		return ObservationReceipt{}, fmt.Errorf("lookup observation receipt: %w", err)
	}

	var observationID string
	if err := tx.QueryRow(ctx, `
INSERT INTO observations (tenant_id, continuity_id, operation_id, observation_kind, content, source_ref)
VALUES ($1, $2::uuid, $3, $4, $5, $6)
RETURNING id::text`, tenantID, continuityID, request.OperationID, request.Kind, request.Content, request.SourceRef).Scan(&observationID); err != nil {
		return ObservationReceipt{}, fmt.Errorf("insert observation: %w", err)
	}
	return ObservationReceipt{ObservationID: observationID}, nil
}

func (s *Store) RecordDelivery(ctx context.Context, tenantID, continuityID, operationID, task, contextBody string) (DeliveryReceipt, error) {
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return DeliveryReceipt{}, fmt.Errorf("begin context delivery: %w", err)
	}
	defer tx.Rollback(ctx)

	var deliveryID, existingContinuityID, existingContext string
	err = tx.QueryRow(ctx, `
SELECT id::text, continuity_id::text, context_body
FROM memory_deliveries
WHERE tenant_id = $1 AND operation_id = $2`, tenantID, operationID).Scan(&deliveryID, &existingContinuityID, &existingContext)
	if err == nil {
		if existingContinuityID != continuityID {
			return DeliveryReceipt{}, fmt.Errorf("operation_id is already bound to another continuity")
		}
		if err := tx.Commit(ctx); err != nil {
			return DeliveryReceipt{}, fmt.Errorf("commit replayed context delivery: %w", err)
		}
		return DeliveryReceipt{DeliveryID: deliveryID, Context: existingContext, Replayed: true}, nil
	}
	if !errors.Is(err, pgx.ErrNoRows) {
		return DeliveryReceipt{}, fmt.Errorf("lookup context delivery: %w", err)
	}
	if err := tx.QueryRow(ctx, `
INSERT INTO memory_deliveries (tenant_id, continuity_id, operation_id, task, context_body)
VALUES ($1, $2::uuid, $3, $4, $5)
RETURNING id::text`, tenantID, continuityID, operationID, task, contextBody).Scan(&deliveryID); err != nil {
		return DeliveryReceipt{}, fmt.Errorf("record context delivery: %w", err)
	}
	if err := tx.Commit(ctx); err != nil {
		return DeliveryReceipt{}, fmt.Errorf("commit context delivery: %w", err)
	}
	return DeliveryReceipt{DeliveryID: deliveryID, Context: contextBody}, nil
}

func (s *Store) DeliveryContinuity(ctx context.Context, tenantID, deliveryID string) (string, error) {
	var continuityID string
	err := s.pool.QueryRow(ctx, `
SELECT continuity_id::text
FROM memory_deliveries
WHERE id = $1::uuid AND tenant_id = $2`, deliveryID, tenantID).Scan(&continuityID)
	if errors.Is(err, pgx.ErrNoRows) {
		return "", fmt.Errorf("delivery does not belong to this tenant")
	}
	if err != nil {
		return "", fmt.Errorf("resolve delivery continuity: %w", err)
	}
	return continuityID, nil
}

func (s *Store) GovernObservation(ctx context.Context, tenantID, continuityID, observationID string, request CommitObservationRequest) (MemoryReceipt, error) {
	if err := request.Validate(); err != nil {
		return MemoryReceipt{}, err
	}
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return MemoryReceipt{}, fmt.Errorf("begin governed memory: %w", err)
	}
	defer tx.Rollback(ctx)
	memory, err := governObservationTx(ctx, tx, tenantID, continuityID, observationID, request)
	if err != nil {
		return MemoryReceipt{}, err
	}
	if err := tx.Commit(ctx); err != nil {
		return MemoryReceipt{}, fmt.Errorf("commit governed memory: %w", err)
	}
	return memory, nil
}

func governObservationTx(ctx context.Context, tx pgx.Tx, tenantID, continuityID, observationID string, request CommitObservationRequest) (MemoryReceipt, error) {
	var observationBelongsToContinuity bool
	if err := tx.QueryRow(ctx, `
SELECT EXISTS (
  SELECT 1 FROM observations
  WHERE id = $1::uuid AND tenant_id = $2 AND continuity_id = $3::uuid
)`, observationID, tenantID, continuityID).Scan(&observationBelongsToContinuity); err != nil {
		return MemoryReceipt{}, fmt.Errorf("check governed observation scope: %w", err)
	}
	if !observationBelongsToContinuity {
		return MemoryReceipt{}, fmt.Errorf("observation does not belong to this continuity")
	}

	var existingID, existingStatus string
	err := tx.QueryRow(ctx, `
SELECT id::text, lifecycle_status
FROM governed_memories
WHERE origin_observation_id = $1::uuid`, observationID).Scan(&existingID, &existingStatus)
	if err == nil {
		return MemoryReceipt{MemoryID: existingID, Status: existingStatus, Replayed: true}, nil
	}
	if !errors.Is(err, pgx.ErrNoRows) {
		return MemoryReceipt{}, fmt.Errorf("lookup governed memory: %w", err)
	}

	status := "proposed"
	if request.Kind == ObservationKindUserCorrection || request.Kind == ObservationKindSourceUpdate {
		status = "active"
	}
	if request.SupersedesMemoryID != "" {
		command, err := tx.Exec(ctx, `
UPDATE governed_memories
SET lifecycle_status = 'superseded', updated_at = now()
WHERE id = $1::uuid AND tenant_id = $2 AND continuity_id = $3::uuid AND lifecycle_status = 'active'`, request.SupersedesMemoryID, tenantID, continuityID)
		if err != nil {
			return MemoryReceipt{}, fmt.Errorf("supersede governed memory: %w", err)
		}
		if command.RowsAffected() != 1 {
			return MemoryReceipt{}, fmt.Errorf("superseded memory must be an active fact in the delivery continuity")
		}
		if _, err := tx.Exec(ctx, `DELETE FROM memory_search_documents WHERE memory_id = $1::uuid`, request.SupersedesMemoryID); err != nil {
			return MemoryReceipt{}, fmt.Errorf("remove superseded search document: %w", err)
		}
	}

	var memoryID string
	if err := tx.QueryRow(ctx, `
INSERT INTO governed_memories (
  tenant_id, continuity_id, origin_observation_id, memory_kind, lifecycle_status, content, supersedes_memory_id
)
VALUES ($1, $2::uuid, $3::uuid, 'fact', $4, $5, NULLIF($6, '')::uuid)
RETURNING id::text`, tenantID, continuityID, observationID, status, request.Content, request.SupersedesMemoryID).Scan(&memoryID); err != nil {
		return MemoryReceipt{}, fmt.Errorf("create governed memory: %w", err)
	}
	if status == "active" {
		if _, err := tx.Exec(ctx, `
INSERT INTO memory_search_documents (memory_id, tenant_id, continuity_id, content, search_document)
VALUES ($1::uuid, $2, $3::uuid, $4, to_tsvector('simple', $4))`, memoryID, tenantID, continuityID, request.Content); err != nil {
			return MemoryReceipt{}, fmt.Errorf("project governed memory: %w", err)
		}
	}
	return MemoryReceipt{MemoryID: memoryID, Status: status}, nil
}

func (s *Store) DeleteMemory(ctx context.Context, tenantID, continuityID, memoryID string) error {
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("begin delete governed memory: %w", err)
	}
	defer tx.Rollback(ctx)
	if err := deleteMemoryTx(ctx, tx, tenantID, continuityID, memoryID); err != nil {
		return err
	}
	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("commit delete governed memory: %w", err)
	}
	return nil
}

func deleteMemoryTx(ctx context.Context, tx pgx.Tx, tenantID, continuityID, memoryID string) error {
	var lifecycleStatus string
	var originObservationID *string
	err := tx.QueryRow(ctx, `
SELECT lifecycle_status, origin_observation_id::text
FROM governed_memories
WHERE id = $1::uuid AND tenant_id = $2 AND continuity_id = $3::uuid`, memoryID, tenantID, continuityID).Scan(&lifecycleStatus, &originObservationID)
	if errors.Is(err, pgx.ErrNoRows) {
		return fmt.Errorf("memory does not belong to this continuity")
	}
	if err != nil {
		return fmt.Errorf("lookup governed memory for deletion: %w", err)
	}
	if lifecycleStatus == "deleted" {
		return nil
	}
	if _, err := tx.Exec(ctx, `
UPDATE governed_memories
SET lifecycle_status = 'deleted', content = '[redacted]', updated_at = now()
WHERE id = $1::uuid`, memoryID); err != nil {
		return fmt.Errorf("redact governed memory: %w", err)
	}
	if _, err := tx.Exec(ctx, `DELETE FROM memory_search_documents WHERE memory_id = $1::uuid`, memoryID); err != nil {
		return fmt.Errorf("remove deleted search document: %w", err)
	}
	if originObservationID != nil {
		if _, err := tx.Exec(ctx, `UPDATE observations SET content = '[redacted]' WHERE id = $1::uuid`, *originObservationID); err != nil {
			return fmt.Errorf("redact origin observation: %w", err)
		}
	}
	return nil
}

func (s *Store) RebuildProjection(ctx context.Context, tenantID, continuityID string) error {
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("begin projection rebuild: %w", err)
	}
	defer tx.Rollback(ctx)
	if _, err := tx.Exec(ctx, `
DELETE FROM memory_search_documents
WHERE tenant_id = $1 AND continuity_id = $2::uuid`, tenantID, continuityID); err != nil {
		return fmt.Errorf("clear search projection: %w", err)
	}
	if _, err := tx.Exec(ctx, `
INSERT INTO memory_search_documents (memory_id, tenant_id, continuity_id, content, search_document)
SELECT id, tenant_id, continuity_id, content, to_tsvector('simple', content)
FROM governed_memories
WHERE tenant_id = $1 AND continuity_id = $2::uuid AND lifecycle_status = 'active'`, tenantID, continuityID); err != nil {
		return fmt.Errorf("rebuild search projection: %w", err)
	}
	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("commit projection rebuild: %w", err)
	}
	return nil
}

func (s *Store) ListGovernedMemories(ctx context.Context, tenantID, continuityID string) ([]GovernedMemory, error) {
	rows, err := s.pool.Query(ctx, `
SELECT id::text, lifecycle_status, content, COALESCE(supersedes_memory_id::text, '')
FROM governed_memories
WHERE tenant_id = $1 AND continuity_id = $2::uuid
ORDER BY created_at ASC, id ASC`, tenantID, continuityID)
	if err != nil {
		return nil, fmt.Errorf("list governed memories: %w", err)
	}
	defer rows.Close()

	memories := make([]GovernedMemory, 0)
	for rows.Next() {
		var memory GovernedMemory
		if err := rows.Scan(&memory.ID, &memory.LifecycleStatus, &memory.Content, &memory.SupersedesMemoryID); err != nil {
			return nil, fmt.Errorf("scan governed memory: %w", err)
		}
		memories = append(memories, memory)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate governed memories: %w", err)
	}
	return memories, nil
}

func (s *Store) SearchActiveMemory(ctx context.Context, tenantID, continuityID, query string, limit int) ([]Memory, error) {
	query = strings.TrimSpace(query)
	if query == "" {
		return nil, fmt.Errorf("search query is required")
	}
	if limit <= 0 {
		limit = defaultContextItems
	}
	if limit > maxContextItems {
		limit = maxContextItems
	}
	rows, err := s.pool.Query(ctx, `
WITH query_terms AS (
  SELECT
    lower($3)::text AS exact_query,
    plainto_tsquery('simple', $3) AS all_terms,
    to_tsquery('simple', array_to_string(tsvector_to_array(to_tsvector('simple', $3)), ' | ')) AS any_terms
), exact_matches AS (
  SELECT 1
  FROM memory_search_documents document
  JOIN governed_memories memory ON memory.id = document.memory_id
  CROSS JOIN query_terms
  WHERE document.tenant_id = $1
    AND document.continuity_id = $2::uuid
    AND memory.tenant_id = $1
    AND memory.continuity_id = $2::uuid
    AND memory.lifecycle_status = 'active'
    AND position(query_terms.exact_query IN lower(document.content)) > 0
  LIMIT 1
)
SELECT memory.id::text, memory.content
FROM memory_search_documents document
JOIN governed_memories memory ON memory.id = document.memory_id
CROSS JOIN query_terms
WHERE document.tenant_id = $1
  AND document.continuity_id = $2::uuid
  AND memory.tenant_id = $1
  AND memory.continuity_id = $2::uuid
  AND memory.lifecycle_status = 'active'
  AND (
    position(query_terms.exact_query IN lower(document.content)) > 0
    OR (
      NOT EXISTS (SELECT 1 FROM exact_matches)
      AND (
        document.search_document @@ query_terms.any_terms
        OR similarity(lower(document.content), query_terms.exact_query) >= 0.2
      )
    )
  )
ORDER BY
  (position(query_terms.exact_query IN lower(document.content)) > 0) DESC,
  ts_rank(document.search_document, query_terms.all_terms) DESC,
  ts_rank(document.search_document, query_terms.any_terms) DESC,
  similarity(lower(document.content), query_terms.exact_query) DESC,
  memory.updated_at DESC
LIMIT $4`, tenantID, continuityID, query, limit)
	if err != nil {
		return nil, fmt.Errorf("search active memory: %w", err)
	}
	defer rows.Close()
	memories := make([]Memory, 0)
	for rows.Next() {
		var memory Memory
		if err := rows.Scan(&memory.ID, &memory.Content); err != nil {
			return nil, fmt.Errorf("scan active memory: %w", err)
		}
		memories = append(memories, memory)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate active memory: %w", err)
	}
	return memories, nil
}

func migrationDir() string {
	_, file, _, ok := runtime.Caller(0)
	if !ok {
		return filepath.Join("internal", "store", "postgres", "migrations")
	}
	return filepath.Clean(filepath.Join(filepath.Dir(file), "..", "store", "postgres", "migrations"))
}
