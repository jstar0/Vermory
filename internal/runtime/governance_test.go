package runtime

import (
	"context"
	"strings"
	"testing"
)

func TestGovernanceInspectDoesNotCreateUnknownWorkspace(t *testing.T) {
	service := NewGovernanceService(openTestStore(t), "local")

	got, err := service.InspectWorkspace(context.Background(), "/repo/unknown")
	requireNoError(t, err)
	if got.Status != ResolutionNeedsConfirmation || got.ContinuityID != "" {
		t.Fatalf("unknown workspace was attached: %#v", got)
	}
}

func TestGovernanceSourceCorrectionAndForgetStayScoped(t *testing.T) {
	ctx := context.Background()
	store := openTestStore(t)
	service := NewGovernanceService(store, "local")
	_, err := service.ConfirmWorkspace(ctx, "/repo/web-checkout")
	requireNoError(t, err)
	_, err = service.ConfirmWorkspace(ctx, "/repo/ops-console")
	requireNoError(t, err)

	source, err := service.AddSource(ctx, "/repo/web-checkout", GovernanceWriteRequest{
		OperationID: "operator-source-v1",
		Content:     "Use checkout_eta_v1 for the staged checkout release.",
		SourceRef:   "fixture:operator:v1",
	})
	requireNoError(t, err)
	if source.Memory.Status != "active" {
		t.Fatalf("source fact is not active: %#v", source)
	}

	corrected, err := service.Correct(ctx, "/repo/web-checkout", source.Memory.MemoryID, GovernanceWriteRequest{
		OperationID: "operator-correct-v2",
		Content:     "Use checkout_eta_v2 for the staged checkout release.",
	})
	requireNoError(t, err)
	if corrected.Memory.Status != "active" {
		t.Fatalf("correction is not active: %#v", corrected)
	}

	resolution, err := service.InspectWorkspace(ctx, "/repo/web-checkout")
	requireNoError(t, err)
	requireNoError(t, store.RebuildProjection(ctx, "local", resolution.ContinuityID))
	matches := mustSearch(t, store, resolution.ContinuityID, "checkout_eta_v1")
	if len(matches) != 1 || !strings.Contains(matches[0].Content, "checkout_eta_v2") {
		t.Fatalf("stale source was not replaced: %#v", matches)
	}

	forgotten, err := service.Forget(ctx, "/repo/web-checkout", corrected.Memory.MemoryID, "operator-forget-v2")
	requireNoError(t, err)
	if forgotten.Memory.Status != "deleted" {
		t.Fatalf("forget did not delete the named fact: %#v", forgotten)
	}
	var forgetContent string
	err = store.pool.QueryRow(ctx, `
SELECT content FROM observations
WHERE tenant_id = 'local' AND operation_id = 'operator-forget-v2'`).Scan(&forgetContent)
	requireNoError(t, err)
	if forgetContent != "Operator requested deletion." {
		t.Fatalf("forget observation retained free-text content: %q", forgetContent)
	}
	requireNoError(t, store.RebuildProjection(ctx, "local", resolution.ContinuityID))
	for _, query := range []string{"checkout_eta_v2", "Which checkout flag should the staged release use?"} {
		if got := mustSearch(t, store, resolution.ContinuityID, query); len(got) != 0 {
			t.Fatalf("deleted fact returned for %q: %#v", query, got)
		}
	}
}

func TestGovernanceRejectsCrossWorkspaceCorrectionAndReplaysSource(t *testing.T) {
	ctx := context.Background()
	store := openTestStore(t)
	service := NewGovernanceService(store, "local")
	_, err := service.ConfirmWorkspace(ctx, "/repo/web-checkout")
	requireNoError(t, err)
	_, err = service.ConfirmWorkspace(ctx, "/repo/ops-console")
	requireNoError(t, err)
	source, err := service.AddSource(ctx, "/repo/ops-console", GovernanceWriteRequest{
		OperationID: "operator-ops-source",
		Content:     "Run ops_exception_queue_refresh before handling incidents.",
		SourceRef:   "fixture:operator:ops",
	})
	requireNoError(t, err)

	_, err = service.Correct(ctx, "/repo/web-checkout", source.Memory.MemoryID, GovernanceWriteRequest{
		OperationID: "operator-cross-scope-correction",
		Content:     "Do not cross scope.",
	})
	if err == nil {
		t.Fatal("expected cross-workspace correction to fail")
	}

	replay, err := service.AddSource(ctx, "/repo/ops-console", GovernanceWriteRequest{
		OperationID: "operator-ops-source",
		Content:     "Run ops_exception_queue_refresh before handling incidents.",
		SourceRef:   "fixture:operator:ops",
	})
	requireNoError(t, err)
	if !replay.Observation.Replayed || !replay.Memory.Replayed || replay.Memory.MemoryID != source.Memory.MemoryID {
		t.Fatalf("source replay was not idempotent: first=%#v replay=%#v", source, replay)
	}
}
