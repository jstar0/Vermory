package runtime

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/jackc/pgx/v5"
)

func createBridgeOperationTx(ctx context.Context, tx pgx.Tx, input bridgeLedgerInput) (BridgeReceipt, bool, error) {
	if err := input.normalize(); err != nil {
		return BridgeReceipt{}, false, err
	}
	var existing BridgeReceipt
	var existingFingerprint string
	err := tx.QueryRow(ctx, `
SELECT id::text, operation_id, action, status, request_fingerprint
FROM bridge_operations
WHERE tenant_id = $1 AND operation_id = $2`, input.TenantID, input.OperationID).Scan(
		&existing.ID,
		&existing.OperationID,
		&existing.Action,
		&existing.Status,
		&existingFingerprint,
	)
	if err == nil {
		if existing.Action != input.Action || existingFingerprint != input.RequestFingerprint {
			return BridgeReceipt{}, false, fmt.Errorf("operation_id is already bound to another bridge request")
		}
		existing.Replayed = true
		return existing, true, nil
	}
	if !errors.Is(err, pgx.ErrNoRows) {
		return BridgeReceipt{}, false, fmt.Errorf("lookup bridge operation: %w", err)
	}

	var receipt BridgeReceipt
	err = tx.QueryRow(ctx, `
INSERT INTO bridge_operations (
  tenant_id, operation_id, action, status,
  source_continuity_id, target_continuity_id,
  source_anchor, target_anchor, target_profile, title, export_body, request_fingerprint
)
VALUES (
  $1, $2, $3, 'active',
  NULLIF($4, '')::uuid, NULLIF($5, '')::uuid,
  $6, $7, $8, $9, $10, $11
)
RETURNING id::text, operation_id, action, status`,
		input.TenantID,
		input.OperationID,
		input.Action,
		input.SourceContinuityID,
		input.TargetContinuityID,
		input.SourceAnchor,
		input.TargetAnchor,
		input.TargetProfile,
		input.Title,
		input.ExportBody,
		input.RequestFingerprint,
	).Scan(&receipt.ID, &receipt.OperationID, &receipt.Action, &receipt.Status)
	if err != nil {
		return BridgeReceipt{}, false, fmt.Errorf("create bridge operation: %w", err)
	}
	if _, err := tx.Exec(ctx, `
INSERT INTO bridge_events (tenant_id, bridge_id, event_type, operation_id)
VALUES ($1, $2::uuid, 'created', $3)`, input.TenantID, receipt.ID, input.OperationID); err != nil {
		return BridgeReceipt{}, false, fmt.Errorf("record bridge creation event: %w", err)
	}
	return receipt, false, nil
}

func markBridgeOperationReversedTx(ctx context.Context, tx pgx.Tx, tenantID, bridgeID, operationID string, targetStatus BridgeStatus) (BridgeReceipt, error) {
	tenantID = strings.TrimSpace(tenantID)
	bridgeID = strings.TrimSpace(bridgeID)
	operationID = strings.TrimSpace(operationID)
	if tenantID == "" {
		return BridgeReceipt{}, fmt.Errorf("tenant_id is required")
	}
	if bridgeID == "" {
		return BridgeReceipt{}, fmt.Errorf("bridge_id is required")
	}
	if operationID == "" {
		return BridgeReceipt{}, fmt.Errorf("operation_id is required")
	}
	if targetStatus != BridgeStatusReversed && targetStatus != BridgeStatusRevoked {
		return BridgeReceipt{}, fmt.Errorf("bridge reversal status %q is unsupported", targetStatus)
	}

	var receipt BridgeReceipt
	err := tx.QueryRow(ctx, `
SELECT id::text, operation_id, action, status, reverse_operation_id
FROM bridge_operations
WHERE id = $1::uuid AND tenant_id = $2
FOR UPDATE`, bridgeID, tenantID).Scan(
		&receipt.ID,
		&receipt.OperationID,
		&receipt.Action,
		&receipt.Status,
		&receipt.ReverseOperationID,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return BridgeReceipt{}, fmt.Errorf("bridge does not exist")
	}
	if err != nil {
		return BridgeReceipt{}, fmt.Errorf("lock bridge operation: %w", err)
	}
	if receipt.Status != BridgeStatusActive {
		if receipt.Status == targetStatus && receipt.ReverseOperationID == operationID {
			receipt.Replayed = true
			return receipt, nil
		}
		return BridgeReceipt{}, fmt.Errorf("bridge is already %s", receipt.Status)
	}
	var operationUsed bool
	if err := tx.QueryRow(ctx, `
SELECT EXISTS (
  SELECT 1 FROM bridge_operations
  WHERE tenant_id = $1 AND (operation_id = $2 OR reverse_operation_id = $2)
)`, tenantID, operationID).Scan(&operationUsed); err != nil {
		return BridgeReceipt{}, fmt.Errorf("check bridge reversal operation: %w", err)
	}
	if operationUsed {
		return BridgeReceipt{}, fmt.Errorf("operation_id is already bound to another bridge request")
	}
	eventType := BridgeEventReversed
	if targetStatus == BridgeStatusRevoked {
		eventType = BridgeEventRevoked
	}
	if _, err := tx.Exec(ctx, `
UPDATE bridge_operations
SET status = $1, reverse_operation_id = $2, reversed_at = now()
WHERE id = $3::uuid`, targetStatus, operationID, bridgeID); err != nil {
		return BridgeReceipt{}, fmt.Errorf("reverse bridge operation: %w", err)
	}
	if _, err := tx.Exec(ctx, `
INSERT INTO bridge_events (tenant_id, bridge_id, event_type, operation_id)
VALUES ($1, $2::uuid, $3, $4)`, tenantID, bridgeID, eventType, operationID); err != nil {
		return BridgeReceipt{}, fmt.Errorf("record bridge reversal event: %w", err)
	}
	receipt.Status = targetStatus
	receipt.ReverseOperationID = operationID
	return receipt, nil
}

func (s *Store) InspectBridge(ctx context.Context, tenantID, bridgeID string) (BridgeReceipt, error) {
	tenantID = strings.TrimSpace(tenantID)
	bridgeID = strings.TrimSpace(bridgeID)
	var receipt BridgeReceipt
	err := s.pool.QueryRow(ctx, `
SELECT id::text, operation_id, action, status,
       COALESCE(source_continuity_id::text, ''), COALESCE(target_continuity_id::text, ''),
       source_anchor, target_anchor, target_profile, title, export_body, reverse_operation_id
FROM bridge_operations
WHERE id = $1::uuid AND tenant_id = $2`, bridgeID, tenantID).Scan(
		&receipt.ID,
		&receipt.OperationID,
		&receipt.Action,
		&receipt.Status,
		&receipt.SourceContinuityID,
		&receipt.TargetContinuityID,
		&receipt.SourceAnchor,
		&receipt.TargetAnchor,
		&receipt.TargetProfile,
		&receipt.Title,
		&receipt.ExportBody,
		&receipt.ReverseOperationID,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return BridgeReceipt{}, fmt.Errorf("bridge does not exist")
	}
	if err != nil {
		return BridgeReceipt{}, fmt.Errorf("inspect bridge operation: %w", err)
	}
	events, err := s.listBridgeEvents(ctx, tenantID, bridgeID)
	if err != nil {
		return BridgeReceipt{}, err
	}
	effects, err := s.listBridgeMemoryEffects(ctx, tenantID, bridgeID)
	if err != nil {
		return BridgeReceipt{}, err
	}
	receipt.Events = events
	receipt.MemoryEffects = effects
	return receipt, nil
}

func (s *Store) listBridgeEvents(ctx context.Context, tenantID, bridgeID string) ([]BridgeEvent, error) {
	rows, err := s.pool.Query(ctx, `
SELECT id::text, event_type, operation_id, created_at::text
FROM bridge_events
WHERE tenant_id = $1 AND bridge_id = $2::uuid
ORDER BY created_at ASC, id ASC`, tenantID, bridgeID)
	if err != nil {
		return nil, fmt.Errorf("list bridge events: %w", err)
	}
	defer rows.Close()
	events := make([]BridgeEvent, 0)
	for rows.Next() {
		var event BridgeEvent
		if err := rows.Scan(&event.ID, &event.EventType, &event.OperationID, &event.CreatedAt); err != nil {
			return nil, fmt.Errorf("scan bridge event: %w", err)
		}
		events = append(events, event)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate bridge events: %w", err)
	}
	return events, nil
}

func (s *Store) listBridgeMemoryEffects(ctx context.Context, tenantID, bridgeID string) ([]BridgeMemoryEffect, error) {
	rows, err := s.pool.Query(ctx, `
SELECT effect_kind, source_memory_id::text, COALESCE(target_memory_id::text, ''), order_index
FROM bridge_memory_effects
WHERE tenant_id = $1 AND bridge_id = $2::uuid
ORDER BY order_index ASC, source_memory_id ASC`, tenantID, bridgeID)
	if err != nil {
		return nil, fmt.Errorf("list bridge memory effects: %w", err)
	}
	defer rows.Close()
	effects := make([]BridgeMemoryEffect, 0)
	for rows.Next() {
		var effect BridgeMemoryEffect
		if err := rows.Scan(&effect.EffectKind, &effect.SourceMemoryID, &effect.TargetMemoryID, &effect.OrderIndex); err != nil {
			return nil, fmt.Errorf("scan bridge memory effect: %w", err)
		}
		effects = append(effects, effect)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate bridge memory effects: %w", err)
	}
	return effects, nil
}
