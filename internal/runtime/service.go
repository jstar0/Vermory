package runtime

import (
	"context"
	"fmt"
	"strings"
)

type PrepareContextResponse struct {
	Status     ResolutionStatus `json:"status"`
	DeliveryID string           `json:"delivery_id,omitempty"`
	Context    string           `json:"context"`
}

type CommitObservationResponse struct {
	ObservationID string `json:"observation_id"`
	MemoryID      string `json:"memory_id,omitempty"`
	MemoryStatus  string `json:"memory_status,omitempty"`
	Replayed      bool   `json:"replayed"`
}

type Service struct {
	store    *Store
	tenantID string
}

func NewService(store *Store, tenantID string) *Service {
	return &Service{store: store, tenantID: strings.TrimSpace(tenantID)}
}

func (s *Service) PrepareContext(ctx context.Context, request PrepareContextRequest) (PrepareContextResponse, error) {
	if s.store == nil || s.tenantID == "" {
		return PrepareContextResponse{}, fmt.Errorf("runtime service is not configured")
	}
	if err := request.Validate(); err != nil {
		return PrepareContextResponse{}, err
	}
	resolution, err := s.store.ResolveWorkspace(ctx, s.tenantID, request.Workspace)
	if err != nil {
		return PrepareContextResponse{}, err
	}
	if resolution.Status == ResolutionNeedsConfirmation {
		return PrepareContextResponse{Status: resolution.Status}, nil
	}
	memories, err := s.store.SearchActiveMemory(ctx, s.tenantID, resolution.ContinuityID, request.Task, request.MaxItems)
	if err != nil {
		return PrepareContextResponse{}, err
	}
	content := make([]string, 0, len(memories))
	for _, memory := range memories {
		content = append(content, memory.Content)
	}
	delivery, err := s.store.RecordDelivery(ctx, s.tenantID, resolution.ContinuityID, request.OperationID, request.Task, strings.Join(content, "\n"))
	if err != nil {
		return PrepareContextResponse{}, err
	}
	return PrepareContextResponse{
		Status:     ResolutionResolved,
		DeliveryID: delivery.DeliveryID,
		Context:    delivery.Context,
	}, nil
}

func (s *Service) CommitObservation(ctx context.Context, request CommitObservationRequest) (CommitObservationResponse, error) {
	if s.store == nil || s.tenantID == "" {
		return CommitObservationResponse{}, fmt.Errorf("runtime service is not configured")
	}
	if err := request.Validate(); err != nil {
		return CommitObservationResponse{}, err
	}
	if request.DeliveryID == "" {
		return CommitObservationResponse{}, fmt.Errorf("delivery_id is required")
	}
	continuityID, err := s.store.DeliveryContinuity(ctx, s.tenantID, request.DeliveryID)
	if err != nil {
		return CommitObservationResponse{}, err
	}
	result, err := s.store.CommitGovernedObservation(ctx, s.tenantID, continuityID, request)
	if err != nil {
		return CommitObservationResponse{}, err
	}
	return CommitObservationResponse{
		ObservationID: result.Observation.ObservationID,
		MemoryID:      result.Memory.MemoryID,
		MemoryStatus:  result.Memory.Status,
		Replayed:      result.Observation.Replayed || result.Memory.Replayed,
	}, nil
}
