package app

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

func TestBenchmarkCoverageWritesInternalReadyArtifacts(t *testing.T) {
	artifactRoot := t.TempDir()

	report, err := BenchmarkCoverage(context.Background(), BenchmarkCoverageOptions{
		MapPath:      "../../casebook/benchmarks/public-benchmark-map.json",
		ArtifactRoot: artifactRoot,
		RunID:        "benchmark-coverage-run",
	})
	if err != nil {
		t.Fatalf("BenchmarkCoverage returned error: %v", err)
	}

	if report.Total != 11 {
		t.Fatalf("expected 11 benchmark entries, got %d", report.Total)
	}
	if report.ExecutableCount < 4 {
		t.Fatalf("expected at least 4 executable benchmark mappings, got %d", report.ExecutableCount)
	}
	if len(report.MissingTranslatedTask) != 0 {
		t.Fatalf("expected no missing translated benchmark mappings, got %v", report.MissingTranslatedTask)
	}
	if report.Artifacts.JSONURI == "" || report.Artifacts.MDURI == "" {
		t.Fatalf("expected benchmark coverage artifact URIs, got %#v", report.Artifacts)
	}

	jsonPath := filepath.Join(artifactRoot, "benchmark-coverage", "benchmark-coverage-run", "report.json")
	mdPath := filepath.Join(artifactRoot, "benchmark-coverage", "benchmark-coverage-run", "report.md")
	for _, path := range []string{jsonPath, mdPath} {
		if _, err := os.Stat(path); err != nil {
			t.Fatalf("expected artifact %s: %v", path, err)
		}
	}

	var persisted BenchmarkCoverageArtifact
	data, err := os.ReadFile(jsonPath)
	if err != nil {
		t.Fatalf("read benchmark coverage json: %v", err)
	}
	if err := json.Unmarshal(data, &persisted); err != nil {
		t.Fatalf("unmarshal benchmark coverage json: %v", err)
	}
	if persisted.ExecutableCount != report.ExecutableCount {
		t.Fatalf("expected persisted executable count %d, got %d", report.ExecutableCount, persisted.ExecutableCount)
	}
}
