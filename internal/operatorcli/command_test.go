package operatorcli

import (
	"bytes"
	"context"
	"encoding/json"
	"os"
	"strings"
	"testing"

	"vermory/internal/runtime"

	"github.com/spf13/cobra"
)

type commandReceipt struct {
	Status       string `json:"status"`
	ContinuityID string `json:"continuity_id"`
	MemoryID     string `json:"memory_id"`
	MemoryStatus string `json:"memory_status"`
	Replayed     bool   `json:"replayed"`
}

type commandMemoryList struct {
	ContinuityID string                   `json:"continuity_id"`
	Memories     []runtime.GovernedMemory `json:"memories"`
}

type commandDefaultList struct {
	ContinuityID string                   `json:"continuity_id"`
	Defaults     []runtime.GovernedMemory `json:"defaults"`
}

type commandBridgeReceipt struct {
	BridgeID      string                       `json:"bridge_id"`
	Action        runtime.BridgeAction         `json:"action"`
	Status        runtime.BridgeStatus         `json:"status"`
	ExportBody    string                       `json:"export_body"`
	Replayed      bool                         `json:"replayed"`
	Events        []runtime.BridgeEvent        `json:"events"`
	MemoryEffects []runtime.BridgeMemoryEffect `json:"memory_effects"`
}

func TestWorkspaceAndMemoryCommandsCompleteGovernedFlow(t *testing.T) {
	databaseURL := resetCommandStore(t)

	unknown := runJSONCommand(t, databaseURL, "workspace", "inspect", "--repo-root", "/repo/web-checkout")
	if unknown.Status != string(runtime.ResolutionNeedsConfirmation) || unknown.ContinuityID != "" {
		t.Fatalf("unknown workspace was attached: %#v", unknown)
	}

	confirmed := runJSONCommand(t, databaseURL, "workspace", "confirm", "--repo-root", "/repo/web-checkout")
	if confirmed.Status != string(runtime.ResolutionResolved) || confirmed.ContinuityID == "" {
		t.Fatalf("workspace was not confirmed: %#v", confirmed)
	}

	source := runJSONCommand(t, databaseURL,
		"memory", "add-source",
		"--repo-root", "/repo/web-checkout",
		"--operation-id", "cli-source-v1",
		"--source-ref", "fixture:cli:v1",
		"--content", "Use checkout_eta_v1 for the staged checkout release.")
	if source.MemoryStatus != "active" || source.MemoryID == "" {
		t.Fatalf("source receipt=%#v", source)
	}

	replay := runJSONCommand(t, databaseURL,
		"memory", "add-source",
		"--repo-root", "/repo/web-checkout",
		"--operation-id", "cli-source-v1",
		"--source-ref", "fixture:cli:v1",
		"--content", "Use checkout_eta_v1 for the staged checkout release.")
	if !replay.Replayed || replay.MemoryID != source.MemoryID {
		t.Fatalf("source replay did not return the original receipt: first=%#v replay=%#v", source, replay)
	}

	corrected := runJSONCommand(t, databaseURL,
		"memory", "correct",
		"--repo-root", "/repo/web-checkout",
		"--operation-id", "cli-correct-v2",
		"--memory-id", source.MemoryID,
		"--content", "Use checkout_eta_v2 for the staged checkout release.")
	if corrected.MemoryStatus != "active" || corrected.MemoryID == "" {
		t.Fatalf("correction receipt=%#v", corrected)
	}

	listed := runMemoryListCommand(t, databaseURL, "/repo/web-checkout")
	if !containsMemory(listed.Memories, source.MemoryID, "superseded") || !containsMemory(listed.Memories, corrected.MemoryID, "active") {
		t.Fatalf("unexpected scoped memory list: %#v", listed)
	}

	forgotten := runJSONCommand(t, databaseURL,
		"memory", "forget",
		"--repo-root", "/repo/web-checkout",
		"--operation-id", "cli-forget-v2",
		"--memory-id", corrected.MemoryID)
	if forgotten.MemoryStatus != "deleted" || forgotten.MemoryID != corrected.MemoryID {
		t.Fatalf("forget receipt=%#v", forgotten)
	}

	assertNoActiveCheckoutFact(t, databaseURL, confirmed.ContinuityID)
}

func TestMemoryCommandsRejectUnconfirmedWorkspace(t *testing.T) {
	databaseURL := resetCommandStore(t)
	err := runCommand(t, databaseURL,
		"memory", "add-source",
		"--repo-root", "/repo/unconfirmed",
		"--operation-id", "cli-reject",
		"--source-ref", "fixture:reject",
		"--content", "Must not persist.")
	if err == nil || !strings.Contains(err.Error(), "workspace requires confirmation") {
		t.Fatalf("unexpected mutation error: %v", err)
	}
}

func TestDefaultsCommandsCompleteExplicitLifecycle(t *testing.T) {
	databaseURL := resetCommandStore(t)
	created := runJSONCommand(t, databaseURL,
		"defaults", "set",
		"--operation-id", "cli-default-set",
		"--key", "reply_language",
		"--content", "Default user-facing replies to Chinese unless the active task explicitly requests another language.")
	if created.ContinuityID == "" || created.MemoryID == "" || created.MemoryStatus != "active" {
		t.Fatalf("unexpected default set receipt: %#v", created)
	}

	listed := runDefaultListCommand(t, databaseURL)
	if listed.ContinuityID != created.ContinuityID || len(listed.Defaults) != 1 || listed.Defaults[0].MemoryKey != "reply_language" {
		t.Fatalf("unexpected default list: %#v", listed)
	}

	corrected := runJSONCommand(t, databaseURL,
		"defaults", "correct",
		"--operation-id", "cli-default-correct",
		"--memory-id", created.MemoryID,
		"--content", "Default user-facing replies to Chinese.")
	if corrected.MemoryID == created.MemoryID || corrected.MemoryStatus != "active" {
		t.Fatalf("unexpected default correction receipt: %#v", corrected)
	}

	forgotten := runJSONCommand(t, databaseURL,
		"defaults", "forget",
		"--operation-id", "cli-default-forget",
		"--memory-id", corrected.MemoryID)
	if forgotten.MemoryID != corrected.MemoryID || forgotten.MemoryStatus != "deleted" {
		t.Fatalf("unexpected default forget receipt: %#v", forgotten)
	}
}

func TestBridgeCommandsExposeAllDurableActions(t *testing.T) {
	databaseURL := resetCommandStore(t)
	store, err := runtime.OpenStore(context.Background(), databaseURL)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(store.Close)
	ctx := context.Background()
	conversationA, memoryA := seedCommandConversationMemory(t, store, "cli-bridge-a", runtime.ConversationAnchor{Channel: "web_chat", ThreadID: "bridge-a"}, "Use checkout_eta_v2 for the staged release.")
	conversationB, _ := seedCommandConversationMemory(t, store, "cli-bridge-b", runtime.ConversationAnchor{Channel: "openclaw_dm", ThreadID: "bridge-b"}, "Run the smoke suite before rollout.")
	_ = conversationA
	_ = conversationB
	governance := runtime.NewGovernanceService(store, "local")
	_, err = governance.ConfirmWorkspace(ctx, "/fixtures/cli-bridge-workspace")
	if err != nil {
		t.Fatal(err)
	}
	workspaceMemory, err := governance.AddSource(ctx, "/fixtures/cli-bridge-workspace", runtime.GovernanceWriteRequest{
		OperationID: "cli-bridge-workspace-source",
		Content:     "Run the smoke suite before rollout.",
		SourceRef:   "fixture:cli:bridge",
	})
	if err != nil {
		t.Fatal(err)
	}

	promoted := runBridgeJSONCommand(t, databaseURL,
		"bridge", "promote",
		"--operation-id", "cli-bridge-promote",
		"--source-channel", "web_chat",
		"--source-thread-id", "bridge-a",
		"--target-repo-root", "/fixtures/cli-bridge-workspace",
		"--memory-id", memoryA)
	if promoted.Action != runtime.BridgeActionPromote || promoted.BridgeID == "" {
		t.Fatalf("unexpected promote command receipt: %#v", promoted)
	}

	linked := runBridgeJSONCommand(t, databaseURL,
		"bridge", "link",
		"--operation-id", "cli-bridge-link",
		"--primary-channel", "web_chat",
		"--primary-thread-id", "bridge-a",
		"--linked-channel", "openclaw_dm",
		"--linked-thread-id", "bridge-b")
	if linked.Action != runtime.BridgeActionLink {
		t.Fatalf("unexpected link command receipt: %#v", linked)
	}

	exported := runBridgeJSONCommand(t, databaseURL,
		"bridge", "export",
		"--operation-id", "cli-bridge-export",
		"--repo-root", "/fixtures/cli-bridge-workspace",
		"--memory-id", workspaceMemory.Memory.MemoryID,
		"--title", "CLI handoff",
		"--target-profile", "team_handoff")
	if exported.Action != runtime.BridgeActionExport || !strings.Contains(exported.ExportBody, "smoke suite") {
		t.Fatalf("unexpected export command receipt: %#v", exported)
	}

	_, err = governance.ConfirmWorkspace(ctx, "/fixtures/cli-adopt-original")
	if err != nil {
		t.Fatal(err)
	}
	adopted := runBridgeJSONCommand(t, databaseURL,
		"bridge", "adopt",
		"--operation-id", "cli-bridge-adopt",
		"--existing-repo-root", "/fixtures/cli-adopt-original",
		"--new-repo-root", "/fixtures/cli-adopt-alias")
	if adopted.Action != runtime.BridgeActionAdopt {
		t.Fatalf("unexpected adopt command receipt: %#v", adopted)
	}

	_, err = governance.ConfirmWorkspace(ctx, "/fixtures/cli-rebind-old")
	if err != nil {
		t.Fatal(err)
	}
	rebound := runBridgeJSONCommand(t, databaseURL,
		"bridge", "rebind",
		"--operation-id", "cli-bridge-rebind",
		"--old-repo-root", "/fixtures/cli-rebind-old",
		"--new-repo-root", "/fixtures/cli-rebind-new")
	if rebound.Action != runtime.BridgeActionRebind {
		t.Fatalf("unexpected rebind command receipt: %#v", rebound)
	}

	inspected := runBridgeJSONCommand(t, databaseURL, "bridge", "inspect", "--bridge-id", promoted.BridgeID)
	if inspected.BridgeID != promoted.BridgeID || len(inspected.Events) != 1 {
		t.Fatalf("unexpected bridge inspection: %#v", inspected)
	}
	reversed := runBridgeJSONCommand(t, databaseURL,
		"bridge", "reverse",
		"--operation-id", "cli-bridge-reverse",
		"--bridge-id", promoted.BridgeID)
	if reversed.Status != runtime.BridgeStatusReversed {
		t.Fatalf("unexpected bridge reversal: %#v", reversed)
	}
}

func resetCommandStore(t *testing.T) string {
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
	return databaseURL
}

func runCommand(t *testing.T, databaseURL string, args ...string) error {
	t.Helper()
	root := newTestRoot()
	root.SetOut(&bytes.Buffer{})
	root.SetErr(&bytes.Buffer{})
	root.SetArgs(append(args, "--database-url", databaseURL, "--tenant-id", "local"))
	return root.Execute()
}

func runJSONCommand(t *testing.T, databaseURL string, args ...string) commandReceipt {
	t.Helper()
	var output bytes.Buffer
	root := newTestRoot()
	root.SetOut(&output)
	root.SetErr(&bytes.Buffer{})
	root.SetArgs(append(args, "--database-url", databaseURL, "--tenant-id", "local"))
	if err := root.Execute(); err != nil {
		t.Fatal(err)
	}
	var receipt commandReceipt
	if err := json.Unmarshal(output.Bytes(), &receipt); err != nil {
		t.Fatal(err)
	}
	return receipt
}

func runMemoryListCommand(t *testing.T, databaseURL, repoRoot string) commandMemoryList {
	t.Helper()
	var output bytes.Buffer
	root := newTestRoot()
	root.SetOut(&output)
	root.SetErr(&bytes.Buffer{})
	root.SetArgs([]string{"memory", "inspect", "--repo-root", repoRoot, "--database-url", databaseURL, "--tenant-id", "local"})
	if err := root.Execute(); err != nil {
		t.Fatal(err)
	}
	var listed commandMemoryList
	if err := json.Unmarshal(output.Bytes(), &listed); err != nil {
		t.Fatal(err)
	}
	return listed
}

func runDefaultListCommand(t *testing.T, databaseURL string) commandDefaultList {
	t.Helper()
	var output bytes.Buffer
	root := newTestRoot()
	root.SetOut(&output)
	root.SetErr(&bytes.Buffer{})
	root.SetArgs([]string{"defaults", "inspect", "--database-url", databaseURL, "--tenant-id", "local"})
	if err := root.Execute(); err != nil {
		t.Fatal(err)
	}
	var listed commandDefaultList
	if err := json.Unmarshal(output.Bytes(), &listed); err != nil {
		t.Fatal(err)
	}
	return listed
}

func runBridgeJSONCommand(t *testing.T, databaseURL string, args ...string) commandBridgeReceipt {
	t.Helper()
	var output bytes.Buffer
	root := newTestRoot()
	root.SetOut(&output)
	root.SetErr(&bytes.Buffer{})
	root.SetArgs(append(args, "--database-url", databaseURL, "--tenant-id", "local"))
	if err := root.Execute(); err != nil {
		t.Fatal(err)
	}
	var receipt commandBridgeReceipt
	if err := json.Unmarshal(output.Bytes(), &receipt); err != nil {
		t.Fatal(err)
	}
	return receipt
}

func newTestRoot() *cobra.Command {
	root := &cobra.Command{Use: "vermory", SilenceErrors: true, SilenceUsage: true}
	root.AddCommand(NewWorkspaceCommand(), NewMemoryCommand(), NewDefaultsCommand(), NewBridgeCommand())
	return root
}

func seedCommandConversationMemory(t *testing.T, store *runtime.Store, operationPrefix string, anchor runtime.ConversationAnchor, content string) (string, string) {
	t.Helper()
	ctx := context.Background()
	resolution, err := store.ResolveOrCreateConversation(ctx, "local", anchor)
	if err != nil {
		t.Fatal(err)
	}
	observation, err := store.CommitObservation(ctx, "local", resolution.ContinuityID, runtime.CommitObservationRequest{
		OperationID: operationPrefix + ":message",
		Kind:        runtime.ObservationKindUserMessage,
		Content:     content,
		SourceRef:   "fixture:cli:conversation",
	})
	if err != nil {
		t.Fatal(err)
	}
	memory, err := store.ConfirmConversationObservation(ctx, "local", resolution.ContinuityID, observation.ObservationID, operationPrefix+":confirm")
	if err != nil {
		t.Fatal(err)
	}
	return resolution.ContinuityID, memory.MemoryID
}

func containsMemory(memories []runtime.GovernedMemory, id, status string) bool {
	for _, memory := range memories {
		if memory.ID == id && memory.LifecycleStatus == status {
			return true
		}
	}
	return false
}

func assertNoActiveCheckoutFact(t *testing.T, databaseURL, continuityID string) {
	t.Helper()
	store, err := runtime.OpenStore(context.Background(), databaseURL)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(store.Close)
	if err := store.RebuildProjection(context.Background(), "local", continuityID); err != nil {
		t.Fatal(err)
	}
	for _, query := range []string{"checkout_eta_v2", "Which checkout flag should the staged release use?"} {
		matches, err := store.SearchActiveMemory(context.Background(), "local", continuityID, query, 6)
		if err != nil {
			t.Fatal(err)
		}
		if len(matches) != 0 {
			t.Fatalf("deleted CLI fact returned for %q: %#v", query, matches)
		}
	}
}
