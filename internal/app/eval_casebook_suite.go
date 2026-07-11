package app

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"vermory/internal/artifact"
)

type EvalCasebookSuiteOptions struct {
	CaseRoot     string
	ArtifactRoot string
	Provider     string
	BaseURL      string
	APIKeyEnv    string
	Model        string
	RunID        string
	MaxTokens    int
}

type EvalCasebookSuiteArtifact struct {
	RunID     string                     `json:"run_id"`
	CaseRoot  string                     `json:"case_root"`
	Total     int                        `json:"total"`
	Executed  int                        `json:"executed"`
	Failed    int                        `json:"failed"`
	ByLine    map[string]int             `json:"by_line"`
	Runs      []EvalCasebookSuiteRun     `json:"runs"`
	Artifacts EvalCasebookArtifactReport `json:"artifacts"`
}

type EvalCasebookSuiteRun struct {
	CaseID    string `json:"case_id"`
	Line      string `json:"line"`
	Status    string `json:"status"`
	ReportURI string `json:"report_uri,omitempty"`
	Error     string `json:"error,omitempty"`
}

func EvalCasebookSuite(ctx context.Context, opts EvalCasebookSuiteOptions) (EvalCasebookSuiteArtifact, error) {
	if strings.TrimSpace(opts.CaseRoot) == "" {
		return EvalCasebookSuiteArtifact{}, errors.New("eval-casebook-suite requires case-root")
	}
	if strings.TrimSpace(opts.ArtifactRoot) == "" {
		opts.ArtifactRoot = "./artifacts"
	}

	caseDirs, err := suiteCaseDirectories(opts.CaseRoot)
	if err != nil {
		return EvalCasebookSuiteArtifact{}, err
	}

	runID := chooseRunID(opts.RunID, "casebook-suite")
	report := EvalCasebookSuiteArtifact{
		RunID:    runID,
		CaseRoot: opts.CaseRoot,
		Total:    len(caseDirs),
		ByLine:   map[string]int{},
		Runs:     make([]EvalCasebookSuiteRun, 0, len(caseDirs)),
	}

	for _, caseDir := range caseDirs {
		caseID := filepath.Base(caseDir)
		line := inferCasebookLine(caseID)
		if line == "" {
			report.Failed++
			report.Runs = append(report.Runs, EvalCasebookSuiteRun{
				CaseID: caseID,
				Status: "failed",
				Error:  "cannot infer continuity line from case id",
			})
			continue
		}

		report.ByLine[line]++
		caseRun, err := EvalCasebook(ctx, EvalCasebookOptions{
			CaseDir:      caseDir,
			Line:         line,
			ArtifactRoot: opts.ArtifactRoot,
			Provider:     opts.Provider,
			BaseURL:      opts.BaseURL,
			APIKeyEnv:    opts.APIKeyEnv,
			Model:        opts.Model,
			RunID:        strings.Join([]string{runID, caseID}, "-"),
			MaxTokens:    opts.MaxTokens,
		})
		if err != nil {
			report.Failed++
			report.Runs = append(report.Runs, EvalCasebookSuiteRun{
				CaseID: caseID,
				Line:   line,
				Status: "failed",
				Error:  err.Error(),
			})
			continue
		}

		report.Executed++
		report.Runs = append(report.Runs, EvalCasebookSuiteRun{
			CaseID:    caseID,
			Line:      line,
			Status:    "ok",
			ReportURI: caseRun.Artifacts.MDURI,
		})
	}

	if err := writeCasebookSuiteArtifacts(ctx, opts.ArtifactRoot, &report); err != nil {
		return EvalCasebookSuiteArtifact{}, err
	}

	return report, nil
}

func suiteCaseDirectories(root string) ([]string, error) {
	entries, err := os.ReadDir(root)
	if err != nil {
		return nil, err
	}

	dirs := make([]string, 0, len(entries))
	for _, entry := range entries {
		if entry.IsDir() {
			dirs = append(dirs, filepath.Join(root, entry.Name()))
		}
	}
	sort.Strings(dirs)
	return dirs, nil
}

func inferCasebookLine(caseID string) string {
	switch {
	case strings.HasPrefix(caseID, "0"), strings.HasPrefix(caseID, "1"):
		return casebookLineWorkspace
	case strings.HasPrefix(caseID, "2"):
		return casebookLineConversation
	case strings.HasPrefix(caseID, "3"):
		return casebookLineBridge
	default:
		return ""
	}
}

func writeCasebookSuiteArtifacts(ctx context.Context, artifactRoot string, report *EvalCasebookSuiteArtifact) error {
	store := artifact.NewLocalStore(artifactRoot)
	jsonKey := strings.Join([]string{"casebook-suite", report.RunID, "report.json"}, "/")
	mdKey := strings.Join([]string{"casebook-suite", report.RunID, "report.md"}, "/")

	jsonURI, err := localArtifactURI(artifactRoot, jsonKey)
	if err != nil {
		return err
	}
	mdURI, err := localArtifactURI(artifactRoot, mdKey)
	if err != nil {
		return err
	}
	report.Artifacts = EvalCasebookArtifactReport{
		JSONURI: jsonURI,
		MDURI:   mdURI,
	}

	jsonBytes, err := json.MarshalIndent(report, "", "  ")
	if err != nil {
		return err
	}
	if _, err := store.Put(ctx, jsonKey, jsonBytes); err != nil {
		return err
	}
	if _, err := store.Put(ctx, mdKey, []byte(markdownCasebookSuiteReport(*report))); err != nil {
		return err
	}
	return nil
}

func markdownCasebookSuiteReport(report EvalCasebookSuiteArtifact) string {
	var b strings.Builder
	b.WriteString("# ContextMesh Casebook Suite Report\n\n")
	b.WriteString(fmt.Sprintf("- Run ID: `%s`\n", report.RunID))
	b.WriteString(fmt.Sprintf("- Case root: `%s`\n", report.CaseRoot))
	b.WriteString(fmt.Sprintf("- Total: `%d`\n", report.Total))
	b.WriteString(fmt.Sprintf("- Executed: `%d`\n", report.Executed))
	b.WriteString(fmt.Sprintf("- Failed: `%d`\n\n", report.Failed))

	b.WriteString("## Runs\n\n")
	for _, run := range report.Runs {
		b.WriteString(fmt.Sprintf("- `%s`: line=`%s`, status=`%s`", run.CaseID, run.Line, run.Status))
		if run.ReportURI != "" {
			b.WriteString(fmt.Sprintf(", report=`%s`", run.ReportURI))
		}
		if run.Error != "" {
			b.WriteString(fmt.Sprintf(", error=`%s`", run.Error))
		}
		b.WriteString("\n")
	}
	return b.String()
}
