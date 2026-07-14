package mcpserver

import (
	"context"
	"encoding/json"
	"os"
	"reflect"
	"testing"

	"vermory/internal/brand"
	"vermory/internal/runtime"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

func TestServerVersionUsesBrandVersion(t *testing.T) {
	if got := serverImplementation().Version; got != brand.Version {
		t.Fatalf("MCP version %q does not match brand version %q", got, brand.Version)
	}
}

func TestPrepareContextToolReturnsNeedsConfirmationWithoutContext(t *testing.T) {
	handler, _ := testHandler(t)
	_, out, err := handler.PrepareContext(context.Background(), nil, PrepareContextInput{
		OperationID: "prepare-unknown-workspace",
		RepoRoot:    "/ambiguous/repo",
		Task:        "Continue work.",
	})
	if err != nil {
		t.Fatal(err)
	}
	if out.Status != "needs_confirmation" || out.Context != "" || out.DeliveryID != "" {
		t.Fatalf("unexpected ambiguous workspace result: %#v", out)
	}
}

func TestCommitObservationToolCreatesOnlyProposedMemory(t *testing.T) {
	handler, store := testHandler(t)
	ctx := context.Background()
	if _, err := store.ConfirmWorkspaceBinding(ctx, "local", "/repo/web-checkout"); err != nil {
		t.Fatal(err)
	}
	_, prepared, err := handler.PrepareContext(ctx, nil, PrepareContextInput{
		OperationID: "prepare-agent-writeback",
		RepoRoot:    "/repo/web-checkout",
		Task:        "Continue checkout work.",
	})
	if err != nil {
		t.Fatal(err)
	}
	_, committed, err := handler.CommitObservation(ctx, nil, CommitObservationInput{
		OperationID: "commit-agent-writeback",
		DeliveryID:  prepared.DeliveryID,
		Content:     "Use checkout_eta_unreviewed.",
	})
	if err != nil {
		t.Fatal(err)
	}
	if committed.MemoryStatus != "proposed" {
		t.Fatalf("MCP agent writeback must remain proposed: %#v", committed)
	}
	resolution, err := store.ResolveWorkspace(ctx, "local", runtime.WorkspaceAnchor{RepoRoot: "/repo/web-checkout"})
	if err != nil {
		t.Fatal(err)
	}
	matches, err := store.SearchActiveMemory(ctx, "local", resolution.ContinuityID, "checkout_eta_unreviewed", 5)
	if err != nil {
		t.Fatal(err)
	}
	if len(matches) != 0 {
		t.Fatalf("MCP agent writeback became active memory: %#v", matches)
	}
}

func TestCommitObservationToolHasNoTenantOrAuthorityInput(t *testing.T) {
	inputType := reflect.TypeFor[CommitObservationInput]()
	for _, forbidden := range []string{"TenantID", "Kind", "SupersedesMemoryID", "TargetMemoryID"} {
		if _, ok := inputType.FieldByName(forbidden); ok {
			t.Fatalf("MCP input must not accept %s", forbidden)
		}
	}
}

func TestPrepareContextToolHasNoBindingOverrideInput(t *testing.T) {
	inputType := reflect.TypeFor[PrepareContextInput]()
	if _, ok := inputType.FieldByName("ExplicitBindingID"); ok {
		t.Fatal("MCP input must not let an agent override workspace binding")
	}
}

func TestServerAdvertisesOnlyNormalFlowTools(t *testing.T) {
	handler, _ := testHandler(t)
	ctx := context.Background()
	serverTransport, clientTransport := mcp.NewInMemoryTransports()
	serverSession, err := NewServer(handler).Connect(ctx, serverTransport, nil)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = serverSession.Close() })
	client := mcp.NewClient(&mcp.Implementation{Name: "test-client", Version: "0.1.0"}, nil)
	clientSession, err := client.Connect(ctx, clientTransport, nil)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = clientSession.Close() })

	tools, err := clientSession.ListTools(ctx, nil)
	if err != nil {
		t.Fatal(err)
	}
	if len(tools.Tools) != 2 {
		t.Fatalf("expected two normal-flow tools, got %#v", tools.Tools)
	}
	names := map[string]bool{}
	for _, tool := range tools.Tools {
		names[tool.Name] = true
	}
	if !names["prepare_context"] || !names["commit_observation"] {
		t.Fatalf("unexpected MCP tools: %#v", tools.Tools)
	}

	result, err := clientSession.CallTool(ctx, &mcp.CallToolParams{
		Name: "prepare_context",
		Arguments: map[string]any{
			"operation_id": "prepare-mcp-protocol",
			"repo_root":    "/ambiguous/repo",
			"task":         "Continue work.",
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	if result.IsError {
		t.Fatalf("expected structured needs_confirmation response, got %#v", result)
	}
	encoded, err := json.Marshal(result.StructuredContent)
	if err != nil {
		t.Fatal(err)
	}
	var output PrepareContextOutput
	if err := json.Unmarshal(encoded, &output); err != nil {
		t.Fatal(err)
	}
	if output.Status != "needs_confirmation" || output.Context != "" || output.DeliveryID != "" {
		t.Fatalf("unexpected MCP structured output: %#v", output)
	}
}

func testHandler(t *testing.T) (*Handler, *runtime.Store) {
	t.Helper()
	databaseURL := os.Getenv("VERMORY_TEST_DATABASE_URL")
	if databaseURL == "" {
		t.Skip("VERMORY_TEST_DATABASE_URL is not set")
	}
	store, err := runtime.OpenStore(context.Background(), databaseURL)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(store.Close)
	if err := store.Migrate(context.Background()); err != nil {
		t.Fatal(err)
	}
	if err := store.ResetForTest(context.Background()); err != nil {
		t.Fatal(err)
	}
	return New(runtime.NewService(store, "local"), Config{TenantID: "local"}), store
}
