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
  observations, continuity_bindings, continuity_spaces CASCADE`)
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

	var existingID, existingContinuityID string
	err = tx.QueryRow(ctx, `
SELECT id::text, continuity_id::text FROM observations
WHERE tenant_id = $1 AND operation_id = $2`, tenantID, request.OperationID).Scan(&existingID, &existingContinuityID)
	if err == nil {
		if existingContinuityID != continuityID {
			return ObservationReceipt{}, fmt.Errorf("operation_id is already bound to another continuity")
		}
		if err := tx.Commit(ctx); err != nil {
			return ObservationReceipt{}, fmt.Errorf("commit replayed observation: %w", err)
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
	if err := tx.Commit(ctx); err != nil {
		return ObservationReceipt{}, fmt.Errorf("commit observation: %w", err)
	}
	return ObservationReceipt{ObservationID: observationID}, nil
}

func migrationDir() string {
	_, file, _, ok := runtime.Caller(0)
	if !ok {
		return filepath.Join("internal", "store", "postgres", "migrations")
	}
	return filepath.Clean(filepath.Join(filepath.Dir(file), "..", "store", "postgres", "migrations"))
}
