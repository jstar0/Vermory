package runtime

import (
	"strings"
	"testing"
)

func TestWorkspaceAnchorNormalizesRepoRoot(t *testing.T) {
	anchor, err := (WorkspaceAnchor{RepoRoot: "/work/acorn/../acorn/"}).Normalized()
	if err != nil {
		t.Fatal(err)
	}
	if anchor.RepoRoot != "/work/acorn" {
		t.Fatalf("expected normalized root, got %q", anchor.RepoRoot)
	}
}

func TestPrepareContextRequestRejectsMissingOperationID(t *testing.T) {
	req := PrepareContextRequest{
		Workspace: WorkspaceAnchor{RepoRoot: "/work/acorn"},
		Task:      "Continue checkout work.",
	}
	err := req.Validate()
	if err == nil || !strings.Contains(err.Error(), "operation_id") {
		t.Fatalf("expected operation_id validation error, got %v", err)
	}
}

func TestCommitObservationRequestRejectsUnsupportedKind(t *testing.T) {
	req := CommitObservationRequest{
		OperationID: "writeback-1",
		Kind:        "unknown",
		Content:     "Completed the task.",
	}
	err := req.Validate()
	if err == nil || !strings.Contains(err.Error(), "kind") {
		t.Fatalf("expected kind validation error, got %v", err)
	}
}

func TestPrepareContextRequestClampsMaxItems(t *testing.T) {
	req := PrepareContextRequest{
		OperationID: "prepare-1",
		Workspace:   WorkspaceAnchor{RepoRoot: "/work/acorn"},
		Task:        "Continue checkout work.",
		MaxItems:    100,
	}
	if err := req.Validate(); err != nil {
		t.Fatal(err)
	}
	if req.MaxItems != maxContextItems {
		t.Fatalf("expected max items %d, got %d", maxContextItems, req.MaxItems)
	}
}
