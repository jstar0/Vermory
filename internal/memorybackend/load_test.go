package memorybackend

import (
	"context"
	"testing"
)

func TestRunLoadSuiteMeasuresIdentifierRecall(t *testing.T) {
	report := RunLoadSuite(context.Background(), newTestBackend(false), LoadOptions{
		Records: 5, Queries: 5, Concurrency: 4, ScopeSuffix: "test",
	})
	if !report.Pass || report.ResetDuration <= 0 || report.WritesSucceeded != 5 || report.QueryHits != 5 || report.Recall != 1 {
		t.Fatalf("unexpected load report: %#v", report)
	}
}
