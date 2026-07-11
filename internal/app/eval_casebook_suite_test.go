package app

import (
	"context"
	"os"
	"path/filepath"
	"testing"
)

func TestEvalCasebookSuiteRunsWorkspaceConversationAndBridgeCases(t *testing.T) {
	caseRoot := t.TempDir()
	for _, spec := range []struct {
		id      string
		taskID  string
		prompt  string
		include []string
	}{
		{id: "101-workspace-case", taskID: "workspace-task", prompt: "Summarize ContextMesh workspace continuity.", include: []string{"ContextMesh", "workspace"}},
		{id: "201-conversation-case", taskID: "conversation-task", prompt: "Continue the ContextMesh conversation.", include: []string{"ContextMesh", "workspace"}},
		{id: "301-bridge-case", taskID: "bridge-task", prompt: "Promote the ContextMesh chat into a workspace.", include: []string{"ContextMesh", "workspace"}},
	} {
		source := writeCasebookFixture(t, spec.id, []fixtureSpecTask{{
			ID:             spec.taskID,
			Prompt:         spec.prompt,
			MustInclude:    spec.include,
			MustNotInclude: []string{"jstarctl"},
		}})
		target := filepath.Join(caseRoot, spec.id)
		if err := os.Rename(source, target); err != nil {
			t.Fatalf("move fixture %s: %v", spec.id, err)
		}
	}

	artifactRoot := t.TempDir()
	report, err := EvalCasebookSuite(context.Background(), EvalCasebookSuiteOptions{
		CaseRoot:     caseRoot,
		ArtifactRoot: artifactRoot,
		Provider:     "mock",
		Model:        "mock-model",
		RunID:        "suite-run",
	})
	if err != nil {
		t.Fatalf("EvalCasebookSuite returned error: %v", err)
	}

	if report.Total != 3 || report.Executed != 3 || report.Failed != 0 {
		t.Fatalf("expected 3 successful runs, got %#v", report)
	}
	if report.ByLine["workspace"] != 1 || report.ByLine["conversation"] != 1 || report.ByLine["bridge"] != 1 {
		t.Fatalf("expected line distribution across all three modes, got %#v", report.ByLine)
	}
	if len(report.Runs) != 3 {
		t.Fatalf("expected 3 run entries, got %d", len(report.Runs))
	}
	for _, run := range report.Runs {
		if run.Status != "ok" {
			t.Fatalf("expected run %s to pass, got %#v", run.CaseID, run)
		}
		if run.ReportURI == "" {
			t.Fatalf("expected run %s to include report URI, got %#v", run.CaseID, run)
		}
	}
	if report.Artifacts.JSONURI == "" || report.Artifacts.MDURI == "" {
		t.Fatalf("expected suite artifact URIs, got %#v", report.Artifacts)
	}
	if _, err := os.Stat(filepath.Join(artifactRoot, "casebook-suite", "suite-run", "report.json")); err != nil {
		t.Fatalf("expected suite json artifact: %v", err)
	}
}
