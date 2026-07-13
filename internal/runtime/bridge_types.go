package runtime

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"strings"
)

type BridgeAction string

const (
	BridgeActionPromote BridgeAction = "promote"
	BridgeActionLink    BridgeAction = "link"
	BridgeActionExport  BridgeAction = "export"
	BridgeActionAdopt   BridgeAction = "adopt"
	BridgeActionRebind  BridgeAction = "rebind"
)

func (a BridgeAction) Valid() bool {
	switch a {
	case BridgeActionPromote, BridgeActionLink, BridgeActionExport, BridgeActionAdopt, BridgeActionRebind:
		return true
	default:
		return false
	}
}

type BridgeStatus string

const (
	BridgeStatusActive   BridgeStatus = "active"
	BridgeStatusReversed BridgeStatus = "reversed"
	BridgeStatusRevoked  BridgeStatus = "revoked"
)

type BridgeEventType string

const (
	BridgeEventCreated  BridgeEventType = "created"
	BridgeEventReversed BridgeEventType = "reversed"
	BridgeEventRevoked  BridgeEventType = "revoked"
)

type BridgeEvent struct {
	ID          string          `json:"id"`
	EventType   BridgeEventType `json:"event_type"`
	OperationID string          `json:"operation_id"`
	CreatedAt   string          `json:"created_at"`
}

type BridgeMemoryEffect struct {
	EffectKind     string `json:"effect_kind"`
	SourceMemoryID string `json:"source_memory_id"`
	TargetMemoryID string `json:"target_memory_id,omitempty"`
	OrderIndex     int    `json:"order_index"`
}

type BridgeReceipt struct {
	ID                 string               `json:"bridge_id"`
	OperationID        string               `json:"operation_id"`
	Action             BridgeAction         `json:"action"`
	Status             BridgeStatus         `json:"status"`
	SourceContinuityID string               `json:"source_continuity_id,omitempty"`
	TargetContinuityID string               `json:"target_continuity_id,omitempty"`
	SourceAnchor       string               `json:"source_anchor,omitempty"`
	TargetAnchor       string               `json:"target_anchor,omitempty"`
	TargetProfile      string               `json:"target_profile,omitempty"`
	Title              string               `json:"title,omitempty"`
	ExportBody         string               `json:"export_body,omitempty"`
	ReverseOperationID string               `json:"reverse_operation_id,omitempty"`
	Replayed           bool                 `json:"replayed"`
	Events             []BridgeEvent        `json:"events"`
	MemoryEffects      []BridgeMemoryEffect `json:"memory_effects"`
}

type bridgeLedgerInput struct {
	TenantID           string
	OperationID        string
	Action             BridgeAction
	RequestFingerprint string
	SourceContinuityID string
	TargetContinuityID string
	SourceAnchor       string
	TargetAnchor       string
	TargetProfile      string
	Title              string
	ExportBody         string
}

func (i *bridgeLedgerInput) normalize() error {
	i.TenantID = strings.TrimSpace(i.TenantID)
	i.OperationID = strings.TrimSpace(i.OperationID)
	i.RequestFingerprint = strings.TrimSpace(i.RequestFingerprint)
	i.SourceContinuityID = strings.TrimSpace(i.SourceContinuityID)
	i.TargetContinuityID = strings.TrimSpace(i.TargetContinuityID)
	i.SourceAnchor = strings.TrimSpace(i.SourceAnchor)
	i.TargetAnchor = strings.TrimSpace(i.TargetAnchor)
	i.TargetProfile = strings.TrimSpace(i.TargetProfile)
	i.Title = strings.TrimSpace(i.Title)
	i.ExportBody = strings.TrimSpace(i.ExportBody)
	if i.TenantID == "" {
		return fmt.Errorf("tenant_id is required")
	}
	if i.OperationID == "" {
		return fmt.Errorf("operation_id is required")
	}
	if len(i.OperationID) > 512 {
		return fmt.Errorf("operation_id is too long")
	}
	if !i.Action.Valid() {
		return fmt.Errorf("bridge action %q is unsupported", i.Action)
	}
	if i.RequestFingerprint == "" {
		return fmt.Errorf("request fingerprint is required")
	}
	return nil
}

func bridgeRequestFingerprint(parts ...string) string {
	hash := sha256.New()
	for _, part := range parts {
		value := strings.TrimSpace(part)
		_, _ = fmt.Fprintf(hash, "%d:%s|", len(value), value)
	}
	return hex.EncodeToString(hash.Sum(nil))
}
