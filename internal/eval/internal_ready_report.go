package eval

import (
	"fmt"
	"strings"
)

func BuildInternalReadyReport(results []AcceptanceResult) string {
	var b strings.Builder
	b.WriteString("# Internal Ready Report\n\n")
	if len(results) == 0 {
		b.WriteString("No acceptance results were provided.\n")
		return b.String()
	}

	passCount := 0
	for i, result := range results {
		if result.Pass {
			passCount++
		}
		fmt.Fprintf(&b, "- result_%d: pass=%t failed=%v\n", i+1, result.Pass, result.Failed)
	}
	fmt.Fprintf(&b, "\nSummary: %d/%d results passed.\n", passCount, len(results))
	return b.String()
}
