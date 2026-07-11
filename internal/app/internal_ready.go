package app

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	"vermory/internal/artifact"
)

type InternalReadyOptions struct {
	CaseRoot     string
	BenchmarkMap string
	ArtifactRoot string
	Provider     string
	BaseURL      string
	APIKeyEnv    string
	Model        string
	RunID        string
	MaxTokens    int
}

type InternalReadyArtifact struct {
	RunID     string                     `json:"run_id"`
	Pass      bool                       `json:"pass"`
	Gates     InternalReadyGates         `json:"gates"`
	Casebook  EvalCasebookSuiteArtifact  `json:"casebook"`
	Benchmark BenchmarkCoverageArtifact  `json:"benchmark"`
	Artifacts EvalCasebookArtifactReport `json:"artifacts"`
}

type InternalReadyGates struct {
	CasesDefined           InternalReadyGate `json:"cases_defined"`
	ExecutableCases        InternalReadyGate `json:"executable_cases"`
	BenchmarkExecutability InternalReadyGate `json:"benchmark_executability"`
	ContinuityLineCoverage InternalReadyGate `json:"continuity_line_coverage"`
	ArtifactPipeline       InternalReadyGate `json:"artifact_pipeline"`
}

type InternalReadyGate struct {
	Pass    bool   `json:"pass"`
	Actual  int    `json:"actual,omitempty"`
	Minimum int    `json:"minimum,omitempty"`
	Message string `json:"message,omitempty"`
}

func InternalReady(ctx context.Context, opts InternalReadyOptions) (InternalReadyArtifact, error) {
	if strings.TrimSpace(opts.CaseRoot) == "" {
		return InternalReadyArtifact{}, errors.New("internal-ready requires case-root")
	}
	if strings.TrimSpace(opts.BenchmarkMap) == "" {
		return InternalReadyArtifact{}, errors.New("internal-ready requires benchmark-map")
	}
	if strings.TrimSpace(opts.ArtifactRoot) == "" {
		opts.ArtifactRoot = "./artifacts"
	}

	runID := chooseRunID(opts.RunID, "internal-ready")
	casebookReport, err := EvalCasebookSuite(ctx, EvalCasebookSuiteOptions{
		CaseRoot:     opts.CaseRoot,
		ArtifactRoot: opts.ArtifactRoot,
		Provider:     opts.Provider,
		BaseURL:      opts.BaseURL,
		APIKeyEnv:    opts.APIKeyEnv,
		Model:        opts.Model,
		RunID:        runID + "-casebook",
		MaxTokens:    opts.MaxTokens,
	})
	if err != nil {
		return InternalReadyArtifact{}, err
	}

	benchmarkReport, err := BenchmarkCoverage(ctx, BenchmarkCoverageOptions{
		MapPath:      opts.BenchmarkMap,
		ArtifactRoot: opts.ArtifactRoot,
		RunID:        runID + "-benchmarks",
	})
	if err != nil {
		return InternalReadyArtifact{}, err
	}

	report := InternalReadyArtifact{
		RunID:     runID,
		Casebook:  casebookReport,
		Benchmark: benchmarkReport,
	}
	report.Gates = buildInternalReadyGates(casebookReport, benchmarkReport)
	report.Pass = allInternalReadyGatesPass(report.Gates)

	if err := writeInternalReadyArtifacts(ctx, opts.ArtifactRoot, &report); err != nil {
		return InternalReadyArtifact{}, err
	}

	return report, nil
}

func buildInternalReadyGates(casebook EvalCasebookSuiteArtifact, benchmark BenchmarkCoverageArtifact) InternalReadyGates {
	return InternalReadyGates{
		CasesDefined: InternalReadyGate{
			Pass:    casebook.Total >= 15,
			Actual:  casebook.Total,
			Minimum: 15,
			Message: "at least 15 main cases defined",
		},
		ExecutableCases: InternalReadyGate{
			Pass:    casebook.Executed >= 10,
			Actual:  casebook.Executed,
			Minimum: 10,
			Message: "at least 10 executable casebook cases",
		},
		BenchmarkExecutability: InternalReadyGate{
			Pass:    benchmark.ExecutableCount >= 4 && len(benchmark.MissingTranslatedTask) == 0 && len(benchmark.ExecutableWithoutCases) == 0,
			Actual:  benchmark.ExecutableCount,
			Minimum: 4,
			Message: "all named benchmarks translated and at least 4 executable evaluations",
		},
		ContinuityLineCoverage: InternalReadyGate{
			Pass:    casebook.ByLine[casebookLineWorkspace] > 0 && casebook.ByLine[casebookLineConversation] > 0 && casebook.ByLine[casebookLineBridge] > 0,
			Actual:  nonEmptyLineCount(casebook.ByLine),
			Minimum: 3,
			Message: "workspace, conversation, and bridge lines must all execute",
		},
		ArtifactPipeline: InternalReadyGate{
			Pass:    casebook.Artifacts.JSONURI != "" && casebook.Artifacts.MDURI != "" && benchmark.Artifacts.JSONURI != "" && benchmark.Artifacts.MDURI != "",
			Actual:  artifactPipelineCount(casebook, benchmark),
			Minimum: 4,
			Message: "suite and benchmark JSON/Markdown artifacts must be present",
		},
	}
}

func allInternalReadyGatesPass(gates InternalReadyGates) bool {
	return gates.CasesDefined.Pass &&
		gates.ExecutableCases.Pass &&
		gates.BenchmarkExecutability.Pass &&
		gates.ContinuityLineCoverage.Pass &&
		gates.ArtifactPipeline.Pass
}

func nonEmptyLineCount(byLine map[string]int) int {
	count := 0
	for _, line := range []string{casebookLineWorkspace, casebookLineConversation, casebookLineBridge} {
		if byLine[line] > 0 {
			count++
		}
	}
	return count
}

func artifactPipelineCount(casebook EvalCasebookSuiteArtifact, benchmark BenchmarkCoverageArtifact) int {
	count := 0
	for _, uri := range []string{
		casebook.Artifacts.JSONURI,
		casebook.Artifacts.MDURI,
		benchmark.Artifacts.JSONURI,
		benchmark.Artifacts.MDURI,
	} {
		if uri != "" {
			count++
		}
	}
	return count
}

func writeInternalReadyArtifacts(ctx context.Context, artifactRoot string, report *InternalReadyArtifact) error {
	store := artifact.NewLocalStore(artifactRoot)
	jsonKey := strings.Join([]string{"internal-ready", report.RunID, "report.json"}, "/")
	mdKey := strings.Join([]string{"internal-ready", report.RunID, "report.md"}, "/")

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
	if _, err := store.Put(ctx, mdKey, []byte(markdownInternalReadyReport(*report))); err != nil {
		return err
	}
	return nil
}

func markdownInternalReadyReport(report InternalReadyArtifact) string {
	var b strings.Builder
	b.WriteString("# ContextMesh Internal Ready Report\n\n")
	b.WriteString(fmt.Sprintf("- Run ID: `%s`\n", report.RunID))
	b.WriteString(fmt.Sprintf("- Pass: `%t`\n", report.Pass))
	b.WriteString(fmt.Sprintf("- Casebook: total=`%d`, executed=`%d`, failed=`%d`\n", report.Casebook.Total, report.Casebook.Executed, report.Casebook.Failed))
	b.WriteString(fmt.Sprintf("- Benchmarks: total=`%d`, executable=`%d`\n\n", report.Benchmark.Total, report.Benchmark.ExecutableCount))

	b.WriteString("## Gates\n\n")
	writeInternalReadyGateMarkdown(&b, "cases_defined", report.Gates.CasesDefined)
	writeInternalReadyGateMarkdown(&b, "executable_cases", report.Gates.ExecutableCases)
	writeInternalReadyGateMarkdown(&b, "benchmark_executability", report.Gates.BenchmarkExecutability)
	writeInternalReadyGateMarkdown(&b, "continuity_line_coverage", report.Gates.ContinuityLineCoverage)
	writeInternalReadyGateMarkdown(&b, "artifact_pipeline", report.Gates.ArtifactPipeline)
	return b.String()
}

func writeInternalReadyGateMarkdown(b *strings.Builder, name string, gate InternalReadyGate) {
	b.WriteString(fmt.Sprintf("- `%s`: pass=`%t`, actual=`%d`, minimum=`%d`, message=%s\n", name, gate.Pass, gate.Actual, gate.Minimum, gate.Message))
}
