package runtime

import (
	"context"
	"fmt"
	"strings"
)

type BridgeService struct {
	store    *Store
	tenantID string
}

func NewBridgeService(store *Store, tenantID string) *BridgeService {
	return &BridgeService{store: store, tenantID: strings.TrimSpace(tenantID)}
}

func (s *BridgeService) PromoteConversationToWorkspace(ctx context.Context, request PromoteConversationToWorkspaceRequest) (BridgeReceipt, error) {
	if err := s.configured(); err != nil {
		return BridgeReceipt{}, err
	}
	if err := request.Validate(); err != nil {
		return BridgeReceipt{}, err
	}
	source, err := s.store.ResolveConversation(ctx, s.tenantID, request.Source)
	if err != nil {
		return BridgeReceipt{}, err
	}
	if source.Status != ResolutionResolved {
		return BridgeReceipt{}, fmt.Errorf("source conversation does not exist")
	}
	target, err := s.store.ResolveWorkspace(ctx, s.tenantID, WorkspaceAnchor{RepoRoot: request.TargetRepoRoot})
	if err != nil {
		return BridgeReceipt{}, err
	}
	if target.Status != ResolutionResolved {
		return BridgeReceipt{}, fmt.Errorf("target workspace requires confirmation")
	}
	return s.store.PromoteConversationMemory(
		ctx,
		s.tenantID,
		request.OperationID,
		source.ContinuityID,
		target.ContinuityID,
		request.Source.Channel+"/"+request.Source.ThreadID,
		request.TargetRepoRoot,
		request.MemoryIDs,
	)
}

func (s *BridgeService) ExportWorkspace(ctx context.Context, request ExportWorkspaceRequest) (BridgeReceipt, error) {
	if err := s.configured(); err != nil {
		return BridgeReceipt{}, err
	}
	if err := request.Validate(); err != nil {
		return BridgeReceipt{}, err
	}
	resolution, err := s.store.ResolveWorkspace(ctx, s.tenantID, WorkspaceAnchor{RepoRoot: request.RepoRoot})
	if err != nil {
		return BridgeReceipt{}, err
	}
	if resolution.Status != ResolutionResolved {
		return BridgeReceipt{}, fmt.Errorf("workspace requires confirmation")
	}
	return s.store.ExportWorkspaceMemory(
		ctx,
		s.tenantID,
		request.OperationID,
		resolution.ContinuityID,
		request.RepoRoot,
		request.MemoryIDs,
		request.Title,
		request.TargetProfile,
	)
}

func (s *BridgeService) Reverse(ctx context.Context, request ReverseBridgeRequest) (BridgeReceipt, error) {
	if err := s.configured(); err != nil {
		return BridgeReceipt{}, err
	}
	if err := request.Validate(); err != nil {
		return BridgeReceipt{}, err
	}
	return s.store.ReverseBridge(ctx, s.tenantID, request.OperationID, request.BridgeID)
}

func (s *BridgeService) Inspect(ctx context.Context, bridgeID string) (BridgeReceipt, error) {
	if err := s.configured(); err != nil {
		return BridgeReceipt{}, err
	}
	bridgeID = strings.TrimSpace(bridgeID)
	if bridgeID == "" {
		return BridgeReceipt{}, fmt.Errorf("bridge_id is required")
	}
	return s.store.InspectBridge(ctx, s.tenantID, bridgeID)
}

func (s *BridgeService) configured() error {
	if s.store == nil || s.tenantID == "" {
		return fmt.Errorf("bridge service is not configured")
	}
	return nil
}
