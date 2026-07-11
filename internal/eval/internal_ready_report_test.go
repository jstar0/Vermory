package eval

import (
	"strings"
	"testing"
)

func TestInternalReadyReportSummarizesThreeLines(t *testing.T) {
	report := BuildInternalReadyReport([]AcceptanceResult{
		{Pass: true},
		{Pass: true},
		{Pass: false, Failed: []string{"conversation contamination"}},
	})
	if report == "" {
		t.Fatalf("expected markdown report")
	}
	if !strings.Contains(report, "# Internal Ready Report") {
		t.Fatalf("expected report header, got %q", report)
	}
	if !strings.Contains(report, "result_3") {
		t.Fatalf("expected result_3 summary, got %q", report)
	}
	if !strings.Contains(report, "conversation contamination") {
		t.Fatalf("expected failed gate detail, got %q", report)
	}
}
