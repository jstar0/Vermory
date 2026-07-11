package app

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

func TestInternalReadyRunsExecutableAcceptanceAndWritesArtifacts(t *testing.T) {
	artifactRoot := t.TempDir()

	report, err := InternalReady(context.Background(), InternalReadyOptions{
		CaseRoot:     "../../casebook/cases",
		BenchmarkMap: "../../casebook/benchmarks/public-benchmark-map.json",
		ArtifactRoot: artifactRoot,
		Provider:     "mock",
		Model:        "mock-model",
		RunID:        "internal-ready-run",
	})
	if err != nil {
		t.Fatalf("InternalReady returned error: %v", err)
	}

	if !report.Pass {
		t.Fatalf("expected internal ready to pass, gates: %#v", report.Gates)
	}
	if !report.Gates.CasesDefined.Pass || !report.Gates.ExecutableCases.Pass || !report.Gates.BenchmarkExecutability.Pass {
		t.Fatalf("expected core gates to pass, got %#v", report.Gates)
	}
	if !report.Gates.ContinuityLineCoverage.Pass || !report.Gates.ArtifactPipeline.Pass {
		t.Fatalf("expected coverage and artifact gates to pass, got %#v", report.Gates)
	}
	if report.Casebook.Total < 15 || report.Casebook.Executed < 10 {
		t.Fatalf("expected casebook suite to meet internal ready minimums, got %#v", report.Casebook)
	}
	if report.Benchmark.ExecutableCount < 4 {
		t.Fatalf("expected benchmark executable count >= 4, got %#v", report.Benchmark)
	}
	if report.Artifacts.JSONURI == "" || report.Artifacts.MDURI == "" {
		t.Fatalf("expected internal ready artifact URIs, got %#v", report.Artifacts)
	}

	jsonPath := filepath.Join(artifactRoot, "internal-ready", "internal-ready-run", "report.json")
	mdPath := filepath.Join(artifactRoot, "internal-ready", "internal-ready-run", "report.md")
	for _, path := range []string{jsonPath, mdPath} {
		if _, err := os.Stat(path); err != nil {
			t.Fatalf("expected artifact %s: %v", path, err)
		}
	}

	var persisted InternalReadyArtifact
	data, err := os.ReadFile(jsonPath)
	if err != nil {
		t.Fatalf("read internal ready json: %v", err)
	}
	if err := json.Unmarshal(data, &persisted); err != nil {
		t.Fatalf("unmarshal internal ready json: %v", err)
	}
	if !persisted.Pass {
		t.Fatalf("expected persisted report to pass, got %#v", persisted)
	}
}
