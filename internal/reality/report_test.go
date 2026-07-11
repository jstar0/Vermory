package reality

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestValidateRootReportsCaseAndCoverage(t *testing.T) {
	root := t.TempDir()
	dir := cloneCaseTree(t, "../../reality/testdata/valid-public", filepath.Join(root, "valid-public"))
	if _, err := FreezeCase(dir); err != nil {
		t.Fatal(err)
	}

	report := ValidateRoot(root)
	if !report.Pass {
		t.Fatalf("expected validation to pass: %#v", report)
	}
	if report.Cases != 1 || report.ByLine[string(LineWorkspace)] != 1 || report.ByEvidence[string(EvidencePublic)] != 1 {
		t.Fatalf("unexpected coverage: %#v", report)
	}
	if len(report.Results) != 1 || report.Results[0].CaseID != "valid-public" || report.Results[0].LockSHA256 == "" {
		t.Fatalf("unexpected case result: %#v", report.Results)
	}
}

func TestValidateRootNamesInvalidLocalSealedCase(t *testing.T) {
	root := t.TempDir()
	cloneCaseTree(t, "../../reality/testdata/invalid-local-sealed", filepath.Join(root, "invalid-local-sealed"))

	report := ValidateRoot(root)
	if report.Pass || len(report.Results) != 1 || report.Results[0].CaseID != "invalid-local-sealed" {
		t.Fatalf("expected named invalid case: %#v", report)
	}
	if !hasViolation(report, "local_sealed_evidence") {
		t.Fatalf("expected local sealed violation: %#v", report)
	}
}

func TestWriteValidationArtifacts(t *testing.T) {
	report := ValidationReport{
		RunID:      "run-1",
		CaseRoot:   "reality/cases",
		Pass:       true,
		Cases:      1,
		ByLine:     map[string]int{"workspace": 1},
		ByEvidence: map[string]int{"public": 1},
		Results:    []CaseResult{{CaseID: "case-1", Pass: true, LockSHA256: strings.Repeat("a", 64)}},
	}

	artifacts, err := WriteValidationArtifacts(t.TempDir(), report)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.HasSuffix(artifacts.JSONPath, filepath.Join("reality-validation", "run-1", "report.json")) {
		t.Fatalf("unexpected JSON path %q", artifacts.JSONPath)
	}
	data, err := os.ReadFile(artifacts.JSONPath)
	if err != nil {
		t.Fatal(err)
	}
	var decoded ValidationReport
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatal(err)
	}
	if decoded.RunID != report.RunID || !decoded.Pass {
		t.Fatalf("unexpected JSON report: %#v", decoded)
	}
	markdown, err := os.ReadFile(artifacts.MarkdownPath)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(markdown), "# Reality Validation: run-1") || !strings.Contains(string(markdown), "case-1") {
		t.Fatalf("unexpected Markdown report:\n%s", markdown)
	}
}
