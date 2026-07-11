package memorybackend

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"
)

type LifecycleArtifacts struct {
	JSONPath     string `json:"json_path"`
	MarkdownPath string `json:"markdown_path"`
}

func WriteLifecycleArtifacts(root, runID string, report LifecycleReport) (LifecycleArtifacts, error) {
	if root == "" {
		root = "artifacts"
	}
	if runID == "" {
		runID = time.Now().UTC().Format("20060102T150405Z")
	}
	directory := filepath.Join(root, "backend-bakeoff", safeArtifactSegment(runID), safeArtifactSegment(report.Backend))
	if err := os.MkdirAll(directory, 0o755); err != nil {
		return LifecycleArtifacts{}, fmt.Errorf("create lifecycle artifact directory: %w", err)
	}
	jsonPath := filepath.Join(directory, "report.json")
	markdownPath := filepath.Join(directory, "report.md")
	payload, err := json.MarshalIndent(report, "", "  ")
	if err != nil {
		return LifecycleArtifacts{}, fmt.Errorf("encode lifecycle report: %w", err)
	}
	payload = append(payload, '\n')
	if err := os.WriteFile(jsonPath, payload, 0o644); err != nil {
		return LifecycleArtifacts{}, fmt.Errorf("write lifecycle JSON: %w", err)
	}
	if err := os.WriteFile(markdownPath, []byte(renderLifecycleMarkdown(report)), 0o644); err != nil {
		return LifecycleArtifacts{}, fmt.Errorf("write lifecycle Markdown: %w", err)
	}
	return LifecycleArtifacts{JSONPath: jsonPath, MarkdownPath: markdownPath}, nil
}

func WriteQualityArtifacts(root, runID string, report QualityReport) (LifecycleArtifacts, error) {
	if root == "" {
		root = "artifacts"
	}
	if runID == "" {
		runID = time.Now().UTC().Format("20060102T150405Z")
	}
	directory := filepath.Join(root, "backend-bakeoff", safeArtifactSegment(runID), safeArtifactSegment(report.Backend), "quality")
	if err := os.MkdirAll(directory, 0o755); err != nil {
		return LifecycleArtifacts{}, fmt.Errorf("create quality artifact directory: %w", err)
	}
	jsonPath := filepath.Join(directory, "report.json")
	markdownPath := filepath.Join(directory, "report.md")
	payload, err := json.MarshalIndent(report, "", "  ")
	if err != nil {
		return LifecycleArtifacts{}, fmt.Errorf("encode quality report: %w", err)
	}
	if err := os.WriteFile(jsonPath, append(payload, '\n'), 0o644); err != nil {
		return LifecycleArtifacts{}, fmt.Errorf("write quality JSON: %w", err)
	}
	if err := os.WriteFile(markdownPath, []byte(renderQualityMarkdown(report)), 0o644); err != nil {
		return LifecycleArtifacts{}, fmt.Errorf("write quality Markdown: %w", err)
	}
	return LifecycleArtifacts{JSONPath: jsonPath, MarkdownPath: markdownPath}, nil
}

func WriteLoadArtifacts(root, runID string, report LoadReport) (LifecycleArtifacts, error) {
	if root == "" {
		root = "artifacts"
	}
	if runID == "" {
		runID = time.Now().UTC().Format("20060102T150405Z")
	}
	directory := filepath.Join(root, "backend-bakeoff", safeArtifactSegment(runID), safeArtifactSegment(report.Backend), "load")
	if err := os.MkdirAll(directory, 0o755); err != nil {
		return LifecycleArtifacts{}, fmt.Errorf("create load artifact directory: %w", err)
	}
	jsonPath := filepath.Join(directory, "report.json")
	markdownPath := filepath.Join(directory, "report.md")
	payload, err := json.MarshalIndent(report, "", "  ")
	if err != nil {
		return LifecycleArtifacts{}, fmt.Errorf("encode load report: %w", err)
	}
	if err := os.WriteFile(jsonPath, append(payload, '\n'), 0o644); err != nil {
		return LifecycleArtifacts{}, fmt.Errorf("write load JSON: %w", err)
	}
	if err := os.WriteFile(markdownPath, []byte(renderLoadMarkdown(report)), 0o644); err != nil {
		return LifecycleArtifacts{}, fmt.Errorf("write load Markdown: %w", err)
	}
	return LifecycleArtifacts{JSONPath: jsonPath, MarkdownPath: markdownPath}, nil
}

func renderLifecycleMarkdown(report LifecycleReport) string {
	status := "FAIL"
	if report.Pass {
		status = "PASS"
	}
	var builder strings.Builder
	fmt.Fprintf(&builder, "# Memory Backend Lifecycle: %s\n\n", report.Backend)
	fmt.Fprintf(&builder, "- Overall: %s\n", status)
	fmt.Fprintf(&builder, "- Started: `%s`\n", report.StartedAt.Format(time.RFC3339))
	fmt.Fprintf(&builder, "- Duration: `%s`\n", report.Duration)
	fmt.Fprintf(&builder, "- Records: `%d`\n", report.Stats.RecordCount)
	fmt.Fprintf(&builder, "- Disk bytes: `%d`\n\n", report.Stats.DiskBytes)
	builder.WriteString("| Gate | Result | Details |\n|---|---|---|\n")
	for _, gate := range report.Gates {
		result := "FAIL"
		if gate.Pass {
			result = "PASS"
		}
		details := strings.ReplaceAll(gate.Details, "|", "\\|")
		details = strings.ReplaceAll(details, "\n", " ")
		fmt.Fprintf(&builder, "| `%s` | %s | %s |\n", gate.Name, result, details)
	}
	return builder.String()
}

func renderQualityMarkdown(report QualityReport) string {
	status := "FAIL"
	if report.Pass {
		status = "PASS"
	}
	var builder strings.Builder
	fmt.Fprintf(&builder, "# Memory Backend Quality: %s\n\n", report.Backend)
	fmt.Fprintf(&builder, "- Overall: %s\n", status)
	fmt.Fprintf(&builder, "- Assertions: `%d/%d`\n", report.Metrics.PassedAssertions, report.Metrics.Assertions)
	fmt.Fprintf(&builder, "- Recall: `%.4f`\n", report.Metrics.Recall)
	fmt.Fprintf(&builder, "- Forbidden leakage: `%d`\n", report.Metrics.ForbiddenLeakage)
	fmt.Fprintf(&builder, "- Search P50/P95: `%s` / `%s`\n\n", report.Metrics.SearchP50, report.Metrics.SearchP95)
	builder.WriteString("| Scenario | Result | Queries | Error |\n|---|---|---:|---|\n")
	for _, scenario := range report.Scenarios {
		result := "FAIL"
		if scenario.Pass {
			result = "PASS"
		}
		errorText := strings.ReplaceAll(strings.ReplaceAll(scenario.Error, "|", "\\|"), "\n", " ")
		fmt.Fprintf(&builder, "| `%s` | %s | %d | %s |\n", scenario.ID, result, len(scenario.Queries), errorText)
		for _, query := range scenario.Queries {
			if query.Pass {
				continue
			}
			fmt.Fprintf(&builder, "\n- `%s` failed: missing=%v forbidden=%v\n", scenario.ID, query.Missing, query.ForbiddenFound)
		}
	}
	return builder.String()
}

func renderLoadMarkdown(report LoadReport) string {
	status := "FAIL"
	if report.Pass {
		status = "PASS"
	}
	var builder strings.Builder
	fmt.Fprintf(&builder, "# Memory Backend Load: %s\n\n", report.Backend)
	fmt.Fprintf(&builder, "- Overall: %s\n", status)
	fmt.Fprintf(&builder, "- Writes: `%d/%d`\n", report.WritesSucceeded, report.RecordsRequested)
	fmt.Fprintf(&builder, "- Scope reset duration: `%s`\n", report.ResetDuration)
	fmt.Fprintf(&builder, "- Ingest duration: `%s`\n", report.IngestDuration)
	fmt.Fprintf(&builder, "- Identifier queries: `%d/%d`\n", report.QueryHits, report.Queries)
	fmt.Fprintf(&builder, "- Recall: `%.4f`\n", report.Recall)
	fmt.Fprintf(&builder, "- Search P50/P95: `%s` / `%s`\n", report.SearchP50, report.SearchP95)
	fmt.Fprintf(&builder, "- Indexed records: `%d`\n", report.Stats.RecordCount)
	fmt.Fprintf(&builder, "- Index disk bytes: `%d`\n", report.Stats.DiskBytes)
	if len(report.WriteErrors) > 0 {
		builder.WriteString("\n## Write Errors\n\n")
		for _, writeError := range report.WriteErrors {
			fmt.Fprintf(&builder, "- %s\n", writeError)
		}
	}
	return builder.String()
}

func safeArtifactSegment(value string) string {
	value = strings.TrimSpace(value)
	if value == "" {
		return "unknown"
	}
	return strings.Map(func(r rune) rune {
		if r >= 'a' && r <= 'z' || r >= 'A' && r <= 'Z' || r >= '0' && r <= '9' || r == '-' || r == '_' || r == '.' {
			return r
		}
		return '_'
	}, value)
}
