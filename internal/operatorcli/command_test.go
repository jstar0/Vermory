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

func newTestRoot() *cobra.Command {
	root := &cobra.Command{Use: "vermory", SilenceErrors: true, SilenceUsage: true}
	root.AddCommand(NewWorkspaceCommand(), NewMemoryCommand())
	return root
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
