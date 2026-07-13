package runtime

import (
	"fmt"
	"path/filepath"
	"strings"
)

const (
	defaultContextItems = 6
	maxContextItems     = 12
)

type ObservationKind string

const (
	ObservationKindAgentResult      ObservationKind = "agent_result"
	ObservationKindUserCorrection   ObservationKind = "user_correction"
	ObservationKindSourceUpdate     ObservationKind = "source_update"
	ObservationKindForgetRequest    ObservationKind = "forget_request"
	ObservationKindUserMessage      ObservationKind = "user_message"
	ObservationKindAssistantMessage ObservationKind = "assistant_message"
	ObservationKindUserConfirmation ObservationKind = "user_confirmation"
	ObservationKindGlobalDefaultSet ObservationKind = "global_default_set"
	ObservationKindBridgePromote    ObservationKind = "bridge_promote"
)

type WorkspaceAnchor struct {
	RepoRoot          string `json:"repo_root"`
	CWD               string `json:"cwd,omitempty"`
	ExplicitBindingID string `json:"explicit_binding_id,omitempty"`
}

func (a WorkspaceAnchor) Normalized() (WorkspaceAnchor, error) {
	repoRoot, err := normalizeAbsolutePath(a.RepoRoot)
	if err != nil {
		return WorkspaceAnchor{}, fmt.Errorf("workspace repo_root: %w", err)
	}
	a.RepoRoot = repoRoot
	a.ExplicitBindingID = strings.TrimSpace(a.ExplicitBindingID)
	if strings.TrimSpace(a.CWD) == "" {
		return a, nil
	}
	cwd, err := normalizeAbsolutePath(a.CWD)
	if err != nil {
		return WorkspaceAnchor{}, fmt.Errorf("workspace cwd: %w", err)
	}
	a.CWD = cwd
	return a, nil
}

type PrepareContextRequest struct {
	OperationID string          `json:"operation_id"`
	Workspace   WorkspaceAnchor `json:"workspace"`
	Task        string          `json:"task"`
	MaxItems    int             `json:"max_items,omitempty"`
}

func (r *PrepareContextRequest) Validate() error {
	if strings.TrimSpace(r.OperationID) == "" {
		return fmt.Errorf("operation_id is required")
	}
	if strings.TrimSpace(r.Task) == "" {
		return fmt.Errorf("task is required")
	}
	anchor, err := r.Workspace.Normalized()
	if err != nil {
		return err
	}
	r.Workspace = anchor
	r.OperationID = strings.TrimSpace(r.OperationID)
	r.Task = strings.TrimSpace(r.Task)
	if r.MaxItems < 0 {
		return fmt.Errorf("max_items cannot be negative")
	}
	if r.MaxItems == 0 {
		r.MaxItems = defaultContextItems
	}
	if r.MaxItems > maxContextItems {
		r.MaxItems = maxContextItems
	}
	return nil
}

type CommitObservationRequest struct {
	OperationID        string          `json:"operation_id"`
	DeliveryID         string          `json:"delivery_id,omitempty"`
	Kind               ObservationKind `json:"kind"`
	Content            string          `json:"content"`
	SourceRef          string          `json:"source_ref,omitempty"`
	SupersedesMemoryID string          `json:"supersedes_memory_id,omitempty"`
	TargetMemoryID     string          `json:"target_memory_id,omitempty"`
}

func (r *CommitObservationRequest) Validate() error {
	if strings.TrimSpace(r.OperationID) == "" {
		return fmt.Errorf("operation_id is required")
	}
	if !r.Kind.Valid() {
		return fmt.Errorf("kind %q is unsupported", r.Kind)
	}
	if strings.TrimSpace(r.Content) == "" {
		return fmt.Errorf("content is required")
	}
	r.OperationID = strings.TrimSpace(r.OperationID)
	r.DeliveryID = strings.TrimSpace(r.DeliveryID)
	r.Content = strings.TrimSpace(r.Content)
	r.SourceRef = strings.TrimSpace(r.SourceRef)
	r.SupersedesMemoryID = strings.TrimSpace(r.SupersedesMemoryID)
	r.TargetMemoryID = strings.TrimSpace(r.TargetMemoryID)
	if r.SupersedesMemoryID != "" && r.Kind != ObservationKindUserCorrection && r.Kind != ObservationKindSourceUpdate {
		return fmt.Errorf("supersedes_memory_id is only allowed for user_correction or source_update")
	}
	if r.Kind == ObservationKindForgetRequest && r.TargetMemoryID == "" {
		return fmt.Errorf("target_memory_id is required for forget_request")
	}
	if r.TargetMemoryID != "" && r.Kind != ObservationKindForgetRequest {
		return fmt.Errorf("target_memory_id is only allowed for forget_request")
	}
	return nil
}

func (k ObservationKind) Valid() bool {
	switch k {
	case ObservationKindAgentResult,
		ObservationKindUserCorrection,
		ObservationKindSourceUpdate,
		ObservationKindForgetRequest,
		ObservationKindUserMessage,
		ObservationKindAssistantMessage,
		ObservationKindUserConfirmation,
		ObservationKindGlobalDefaultSet,
		ObservationKindBridgePromote:
		return true
	default:
		return false
	}
}

func normalizeAbsolutePath(value string) (string, error) {
	value = strings.TrimSpace(value)
	if value == "" {
		return "", fmt.Errorf("path is required")
	}
	if !filepath.IsAbs(value) {
		return "", fmt.Errorf("path must be absolute")
	}
	return filepath.Clean(value), nil
}
