package eval

import (
	"fmt"
	"strings"
)

func MarkdownReport(task Task, score Score) string {
	var b strings.Builder
	b.WriteString("# WCEF Report\n\n")
	b.WriteString(fmt.Sprintf("Task: `%s`\n\n", task.ID))
	b.WriteString(fmt.Sprintf("- Must include hit rate: %.2f\n", score.MustIncludeHitRate))
	b.WriteString(fmt.Sprintf("- Missing includes: %s\n", strings.Join(score.MissingIncludes, ", ")))
	b.WriteString(fmt.Sprintf("- Forbidden violations: %s\n", strings.Join(score.MustNotIncludeViolations, ", ")))
	return b.String()
}
