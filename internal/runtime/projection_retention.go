package runtime

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
)

type ProjectionRetention struct {
	TenantID             string     `json:"tenant_id"`
	PrunedThroughEventID int64      `json:"pruned_through_event_id"`
	LastPrunedAt         *time.Time `json:"last_pruned_at,omitempty"`
	UpdatedAt            *time.Time `json:"updated_at,omitempty"`
}

type projectionRetentionQuerier interface {
	QueryRow(context.Context, string, ...any) pgx.Row
}

func (s *Store) ProjectionRetention(ctx context.Context, tenantID string) (ProjectionRetention, error) {
	tenantID = strings.TrimSpace(tenantID)
	ctx, err := withTenantContext(ctx, tenantID)
	if err != nil {
		return ProjectionRetention{}, err
	}
	return projectionRetention(ctx, s.pool, tenantID)
}

func projectionRetention(ctx context.Context, querier projectionRetentionQuerier, tenantID string) (ProjectionRetention, error) {
	retention := ProjectionRetention{TenantID: tenantID}
	err := querier.QueryRow(ctx, `
SELECT pruned_through_event_id, last_pruned_at, updated_at
FROM memory_projection_retention
WHERE tenant_id = $1`, tenantID).Scan(
		&retention.PrunedThroughEventID,
		&retention.LastPrunedAt,
		&retention.UpdatedAt,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return retention, nil
	}
	if err != nil {
		return ProjectionRetention{}, fmt.Errorf("read projection retention: %w", err)
	}
	return retention, nil
}
