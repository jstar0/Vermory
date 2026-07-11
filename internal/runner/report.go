package runner

import (
	"fmt"
	"strings"
)

func MarkdownReport(report EvaluationReport) string {
	var b strings.Builder
	b.WriteString("# ContextMesh Platform Run Report\n\n")
	b.WriteString(fmt.Sprintf("- Run ID: `%s`\n", report.RunID))
	b.WriteString(fmt.Sprintf("- Task ID: `%s`\n", report.TaskID))
	b.WriteString(fmt.Sprintf("- Provider mode: `%s`\n", report.ProviderMode))
	b.WriteString(fmt.Sprintf("- Provider: `%s`\n", report.ProviderName))
	b.WriteString(fmt.Sprintf("- Model: `%s`\n\n", report.Model))
	b.WriteString("This report records direct model/tool consumption evidence. Deterministic string scoring is not a substitute for human evaluation or full AI coding tool evaluation.\n\n")
	b.WriteString("| Baseline | Include Hit Rate | Missing Includes | Forbidden Violations |\n")
	b.WriteString("| --- | ---: | --- | --- |\n")
	for _, result := range report.Results {
		b.WriteString(fmt.Sprintf(
			"| `%s` | %.2f | %s | %s |\n",
			result.Baseline,
			result.Score.MustIncludeHitRate,
			joinOrNone(result.Score.MissingIncludes),
			joinOrNone(result.Score.MustNotIncludeViolations),
		))
	}
	return b.String()
}

func joinOrNone(items []string) string {
	if len(items) == 0 {
		return "-"
	}
	return strings.Join(items, ", ")
}
