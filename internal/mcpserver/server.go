package mcpserver

import (
	"context"
	"fmt"
	"strings"

	"vermory/internal/brand"
	"vermory/internal/runtime"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

type Config struct {
	TenantID string
}

type Handler struct {
	service  *runtime.Service
	tenantID string
}

type PrepareContextInput struct {
	OperationID string `json:"operation_id" jsonschema:"stable id for this context preparation operation"`
	RepoRoot    string `json:"repo_root" jsonschema:"absolute repository root for the current workspace"`
	CWD         string `json:"cwd,omitempty" jsonschema:"optional absolute current working directory"`
	Task        string `json:"task" jsonschema:"current task that needs governed context"`
	MaxItems    int    `json:"max_items,omitempty" jsonschema:"maximum number of governed facts to return"`
}

type PrepareContextOutput struct {
	Status     string `json:"status" jsonschema:"workspace resolution status"`
	DeliveryID string `json:"delivery_id,omitempty" jsonschema:"opaque receipt for a later observation writeback"`
	Context    string `json:"context" jsonschema:"governed semantic context for the current task"`
}

type CommitObservationInput struct {
	OperationID string `json:"operation_id" jsonschema:"stable id for this post-task observation"`
	DeliveryID  string `json:"delivery_id" jsonschema:"opaque receipt returned by prepare_context"`
	Content     string `json:"content" jsonschema:"post-task result or observation proposed by the coding agent"`
	SourceRef   string `json:"source_ref,omitempty" jsonschema:"optional non-sensitive source reference for audit"`
}

type CommitObservationOutput struct {
	ObservationID string `json:"observation_id" jsonschema:"opaque observation receipt"`
	MemoryID      string `json:"memory_id,omitempty" jsonschema:"opaque proposed memory receipt"`
	MemoryStatus  string `json:"memory_status" jsonschema:"governance state assigned to this writeback"`
	Replayed      bool   `json:"replayed" jsonschema:"whether this operation id replayed an existing result"`
}

func New(service *runtime.Service, config Config) *Handler {
	return &Handler{service: service, tenantID: strings.TrimSpace(config.TenantID)}
}

func NewServer(handler *Handler) *mcp.Server {
	server := mcp.NewServer(&mcp.Implementation{Name: brand.Slug, Version: "0.1.0"}, nil)
	mcp.AddTool(server, &mcp.Tool{
		Name:        "prepare_context",
		Description: "Resolve a workspace and return governed context for the current task.",
	}, handler.PrepareContext)
	mcp.AddTool(server, &mcp.Tool{
		Name:        "commit_observation",
		Description: "Record a coding-agent result as a proposed observation for the prepared workspace.",
	}, handler.CommitObservation)
	return server
}

func (h *Handler) PrepareContext(ctx context.Context, _ *mcp.CallToolRequest, input PrepareContextInput) (*mcp.CallToolResult, PrepareContextOutput, error) {
	if err := h.validate(); err != nil {
		return nil, PrepareContextOutput{}, err
	}
	result, err := h.service.PrepareContext(ctx, runtime.PrepareContextRequest{
		OperationID: input.OperationID,
		Workspace: runtime.WorkspaceAnchor{
			RepoRoot: input.RepoRoot,
			CWD:      input.CWD,
		},
		Task:     input.Task,
		MaxItems: input.MaxItems,
	})
	if err != nil {
		return nil, PrepareContextOutput{}, err
	}
	return nil, PrepareContextOutput{
		Status:     string(result.Status),
		DeliveryID: result.DeliveryID,
		Context:    result.Context,
	}, nil
}

func (h *Handler) CommitObservation(ctx context.Context, _ *mcp.CallToolRequest, input CommitObservationInput) (*mcp.CallToolResult, CommitObservationOutput, error) {
	if err := h.validate(); err != nil {
		return nil, CommitObservationOutput{}, err
	}
	result, err := h.service.CommitObservation(ctx, runtime.CommitObservationRequest{
		OperationID: input.OperationID,
		DeliveryID:  input.DeliveryID,
		Kind:        runtime.ObservationKindAgentResult,
		Content:     input.Content,
		SourceRef:   input.SourceRef,
	})
	if err != nil {
		return nil, CommitObservationOutput{}, err
	}
	return nil, CommitObservationOutput{
		ObservationID: result.ObservationID,
		MemoryID:      result.MemoryID,
		MemoryStatus:  result.MemoryStatus,
		Replayed:      result.Replayed,
	}, nil
}

func (h *Handler) validate() error {
	if h == nil || h.service == nil || h.tenantID == "" {
		return fmt.Errorf("MCP handler is not configured")
	}
	return nil
}
