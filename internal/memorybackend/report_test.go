package memorybackend

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestWriteLifecycleArtifacts(t *testing.T) {
	report := LifecycleReport{
		Backend: "native", Pass: true, StartedAt: time.Date(2026, 7, 11, 12, 0, 0, 0, time.UTC),
		Duration: 1500 * time.Millisecond,
		Gates:    []GateResult{{Name: "health", Pass: true}, {Name: "delete_residue", Pass: true}},
		Stats:    Stats{RecordCount: 3, DiskBytes: 4096},
	}
	artifacts, err := WriteLifecycleArtifacts(t.TempDir(), "run-1", report)
	if err != nil {
		t.Fatal(err)
	}
	if filepath.Base(artifacts.JSONPath) != "report.json" || filepath.Base(artifacts.MarkdownPath) != "report.md" {
		t.Fatalf("unexpected artifact paths: %#v", artifacts)
	}
	markdown, err := os.ReadFile(artifacts.MarkdownPath)
	if err != nil {
		t.Fatal(err)
	}
	text := string(markdown)
	for _, expected := range []string{"# Memory Backend Lifecycle: native", "Overall: PASS", "delete_residue", "4096"} {
		if !strings.Contains(text, expected) {
			t.Fatalf("markdown missing %q:\n%s", expected, text)
		}
	}
}

func TestWriteQualityArtifacts(t *testing.T) {
	report := QualityReport{
		Backend: "mem0", Pass: true, StartedAt: time.Date(2026, 7, 11, 12, 0, 0, 0, time.UTC),
		Metrics:   QualityMetrics{Assertions: 20, PassedAssertions: 20, Recall: 1, SearchP95: 120 * time.Millisecond},
		Scenarios: []QualityScenarioResult{{ID: "B01", Description: "workspace isolation", Pass: true}},
	}
	artifacts, err := WriteQualityArtifacts(t.TempDir(), "run-1", report)
	if err != nil {
		t.Fatal(err)
	}
	markdown, err := os.ReadFile(artifacts.MarkdownPath)
	if err != nil {
		t.Fatal(err)
	}
	text := string(markdown)
	for _, expected := range []string{"# Memory Backend Quality: mem0", "Overall: PASS", "B01", "120ms"} {
		if !strings.Contains(text, expected) {
			t.Fatalf("markdown missing %q:\n%s", expected, text)
		}
	}
}

func TestWriteLoadArtifacts(t *testing.T) {
	report := LoadReport{
		Backend: "native", Pass: true, RecordsRequested: 200, WritesSucceeded: 200,
		Queries: 20, QueryHits: 19, Recall: 0.95, IngestDuration: 3 * time.Second,
		SearchP95: 90 * time.Millisecond,
	}
	artifacts, err := WriteLoadArtifacts(t.TempDir(), "run-1", report)
	if err != nil {
		t.Fatal(err)
	}
	markdown, err := os.ReadFile(artifacts.MarkdownPath)
	if err != nil {
		t.Fatal(err)
	}
	text := string(markdown)
	for _, expected := range []string{"# Memory Backend Load: native", "200/200", "0.9500", "90ms"} {
		if !strings.Contains(text, expected) {
			t.Fatalf("markdown missing %q:\n%s", expected, text)
		}
	}
}
