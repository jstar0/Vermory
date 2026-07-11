package app

import (
	"context"
	"os"
	"path/filepath"
	"testing"
)

func TestEvalMatrixMockRunsMultipleTasksAndModels(t *testing.T) {
	root := t.TempDir()

	report, err := EvalMatrix(context.Background(), EvalMatrixOptions{
		ArtifactRoot: root,
		Provider:     "mock",
		RunID:        "matrix-mock",
		Models:       []string{"mock-a", "mock-b"},
	})
	if err != nil {
		t.Fatalf("EvalMatrix returned error: %v", err)
	}

	if report.ProviderMode != "mock" {
		t.Fatalf("expected mock mode, got %q", report.ProviderMode)
	}
	if len(report.Models) != 2 {
		t.Fatalf("expected 2 model reports, got %d", len(report.Models))
	}
	for _, modelReport := range report.Models {
		if len(modelReport.Tasks) != 5 {
			t.Fatalf("expected 5 task reports per model, got %d", len(modelReport.Tasks))
		}
		for _, task := range modelReport.Tasks {
			if task.Status != "ok" {
				t.Fatalf("expected ok task status, got %#v", task)
			}
		}
	}
	if _, err := os.Stat(filepath.Join(root, "matrix-runs", "matrix-mock", "report.md")); err != nil {
		t.Fatalf("expected matrix report artifact: %v", err)
	}
}
