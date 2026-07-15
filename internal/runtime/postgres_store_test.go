package runtime

import (
	"context"
	"fmt"
	"os"
	"testing"
)

func TestStoreMigrateAcceptsPoolConfiguration(t *testing.T) {
	cluster := startDisposablePostgres18(t)
	defer cluster.stop(t, "fast")

	store, err := OpenStore(context.Background(), fmt.Sprintf("%s&pool_max_conns=2", cluster.databaseURL))
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()

	if err := store.Migrate(context.Background()); err != nil {
		t.Fatal(err)
	}
}

func TestStoreResolveWorkspaceRequiresConfirmationForUnknownAnchor(t *testing.T) {
	store := openTestStore(t)
	result, err := store.ResolveWorkspace(context.Background(), "local", WorkspaceAnchor{RepoRoot: "/repo/new-workspace"})
	if err != nil {
		t.Fatal(err)
	}
	if result.Status != ResolutionNeedsConfirmation {
		t.Fatalf("expected needs confirmation, got %#v", result)
	}
	if result.ContinuityID != "" {
		t.Fatalf("unknown workspace must not create a continuity, got %#v", result)
	}
}

func TestStoreCommitObservationIsIdempotent(t *testing.T) {
	store := openTestStore(t)
	ctx := context.Background()
	continuityID, err := store.ConfirmWorkspaceBinding(ctx, "local", "/repo/web-checkout")
	if err != nil {
		t.Fatal(err)
	}
	req := CommitObservationRequest{
		OperationID: "writeback-1",
		Kind:        ObservationKindUserCorrection,
		Content:     "Use checkout_eta_v2.",
	}
	first, err := store.CommitObservation(ctx, "local", continuityID, req)
	if err != nil {
		t.Fatal(err)
	}
	second, err := store.CommitObservation(ctx, "local", continuityID, req)
	if err != nil {
		t.Fatal(err)
	}
	if first.ObservationID != second.ObservationID || !second.Replayed {
		t.Fatalf("expected idempotent receipt, first=%#v second=%#v", first, second)
	}
}

func openTestStore(t *testing.T) *Store {
	t.Helper()
	databaseURL := os.Getenv("VERMORY_TEST_DATABASE_URL")
	if databaseURL == "" {
		t.Skip("VERMORY_TEST_DATABASE_URL is not set")
	}
	store, err := OpenStore(context.Background(), databaseURL)
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
	return store
}
