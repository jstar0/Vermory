package app

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	"vermory/internal/artifact"
	"vermory/internal/casebook"
)

type BenchmarkCoverageOptions struct {
	MapPath      string
	ArtifactRoot string
	RunID        string
}

type BenchmarkCoverageArtifact struct {
	RunID                  string                       `json:"run_id"`
	MapPath                string                       `json:"map_path"`
	Total                  int                          `json:"total"`
	TranslatedOrBetter     int                          `json:"translated_or_better"`
	ExecutableCount        int                          `json:"executable_count"`
	TranslatedProxyCount   int                          `json:"translated_proxy_count"`
	DesignMappingCount     int                          `json:"design_mapping_count"`
	OriginalExecutionCount int                          `json:"original_execution_count"`
	MissingTranslatedTask  []string                     `json:"missing_translated_task,omitempty"`
	ExecutableWithoutCases []string                     `json:"executable_without_cases,omitempty"`
	ByLine                 map[string]int               `json:"by_line"`
	Entries                []casebook.BenchmarkMapEntry `json:"entries"`
	Artifacts              EvalCasebookArtifactReport   `json:"artifacts"`
}

func BenchmarkCoverage(ctx context.Context, opts BenchmarkCoverageOptions) (BenchmarkCoverageArtifact, error) {
	if strings.TrimSpace(opts.MapPath) == "" {
		return BenchmarkCoverageArtifact{}, errors.New("benchmark-coverage requires map-path")
	}
	if strings.TrimSpace(opts.ArtifactRoot) == "" {
		opts.ArtifactRoot = "./artifacts"
	}

	entries, err := casebook.LoadBenchmarkMap(opts.MapPath)
	if err != nil {
		return BenchmarkCoverageArtifact{}, err
	}
	coverage := casebook.ValidateBenchmarkCoverage(entries)
	translatedProxyCount, designMappingCount := benchmarkEvidenceCounts(entries)
	runID := chooseRunID(opts.RunID, "benchmark-coverage")
	store := artifact.NewLocalStore(opts.ArtifactRoot)

	report := BenchmarkCoverageArtifact{
		RunID:                  runID,
		MapPath:                opts.MapPath,
		Total:                  coverage.Total,
		TranslatedOrBetter:     coverage.TranslatedOrBetter,
		ExecutableCount:        coverage.ExecutableCount,
		TranslatedProxyCount:   translatedProxyCount,
		DesignMappingCount:     designMappingCount,
		OriginalExecutionCount: 0,
		MissingTranslatedTask:  coverage.MissingTranslatedTask,
		ExecutableWithoutCases: coverage.ExecutableWithoutCases,
		ByLine:                 coverage.ByLine,
		Entries:                entries,
	}

	jsonKey := strings.Join([]string{"benchmark-coverage", runID, "report.json"}, "/")
	mdKey := strings.Join([]string{"benchmark-coverage", runID, "report.md"}, "/")
	jsonURI, err := localArtifactURI(opts.ArtifactRoot, jsonKey)
	if err != nil {
		return BenchmarkCoverageArtifact{}, err
	}
	mdURI, err := localArtifactURI(opts.ArtifactRoot, mdKey)
	if err != nil {
		return BenchmarkCoverageArtifact{}, err
	}
	report.Artifacts = EvalCasebookArtifactReport{
		JSONURI: jsonURI,
		MDURI:   mdURI,
	}

	jsonBytes, err := json.MarshalIndent(report, "", "  ")
	if err != nil {
		return BenchmarkCoverageArtifact{}, err
	}
	if _, err := store.Put(ctx, jsonKey, jsonBytes); err != nil {
		return BenchmarkCoverageArtifact{}, err
	}
	if _, err := store.Put(ctx, mdKey, []byte(markdownBenchmarkCoverage(report))); err != nil {
		return BenchmarkCoverageArtifact{}, err
	}

	return report, nil
}

func markdownBenchmarkCoverage(report BenchmarkCoverageArtifact) string {
	var b strings.Builder
	b.WriteString("# ContextMesh Benchmark Coverage Report\n\n")
	b.WriteString(fmt.Sprintf("- Run ID: `%s`\n", report.RunID))
	b.WriteString(fmt.Sprintf("- Map: `%s`\n", report.MapPath))
	b.WriteString(fmt.Sprintf("- Total benchmarks: `%d`\n", report.Total))
	b.WriteString(fmt.Sprintf("- Translated task or better: `%d`\n", report.TranslatedOrBetter))
	b.WriteString(fmt.Sprintf("- Executable evaluation: `%d`\n\n", report.ExecutableCount))
	b.WriteString(fmt.Sprintf("- Translated proxy: `%d`\n", report.TranslatedProxyCount))
	b.WriteString(fmt.Sprintf("- Design mapping: `%d`\n", report.DesignMappingCount))
	b.WriteString(fmt.Sprintf("- Original execution: `%d`\n\n", report.OriginalExecutionCount))

	if len(report.MissingTranslatedTask) > 0 {
		b.WriteString(fmt.Sprintf("- Missing translated task: `%s`\n", strings.Join(report.MissingTranslatedTask, ", ")))
	}
	if len(report.ExecutableWithoutCases) > 0 {
		b.WriteString(fmt.Sprintf("- Executable without cases: `%s`\n", strings.Join(report.ExecutableWithoutCases, ", ")))
	}

	b.WriteString("\n## Entries\n\n")
	for _, entry := range report.Entries {
		b.WriteString(fmt.Sprintf("- `%s`: line=`%s`, capability=`%s`, level=`%s`, cases=`%s`\n",
			entry.Benchmark,
			entry.Line,
			entry.Capability,
			entry.EvaluationLevel,
			strings.Join(entry.CaseIDs, ", "),
		))
	}
	return b.String()
}

func benchmarkEvidenceCounts(entries []casebook.BenchmarkMapEntry) (translatedProxy, designMapping int) {
	for _, entry := range entries {
		switch entry.ExecutionMode {
		case "casebook_translated_proxy":
			translatedProxy++
		case "design_mapping":
			designMapping++
		}
	}
	return translatedProxy, designMapping
}
