package memorybackend

import (
	"context"
	"testing"
)

func TestRunQualitySuiteChecksRequiredAndForbiddenContent(t *testing.T) {
	scope := Scope{TenantID: "quality", ContinuityID: "workspace-a", ContinuityLine: "workspace"}
	suite := QualitySuite{Version: "1.0", Scenarios: []QualityScenario{{
		ID: "B01", Description: "workspace isolation",
		Steps: []QualityStep{
			{Op: "reset", Scope: &scope},
			{Op: "put", Record: &Record{ID: "a", Scope: scope, Status: "active", Content: "flag checkout_eta_v2 owner #checkout-ops"}},
			{Op: "search", Query: &QualityQuery{Scope: scope, Text: "flag owner", Limit: 10, Required: []string{"checkout_eta_v2"}, Forbidden: []string{"ops_exception_queue_refresh"}}},
		},
	}}}

	report := RunQualitySuite(context.Background(), newTestBackend(false), suite)
	if !report.Pass || report.Metrics.Assertions != 2 || report.Metrics.PassedAssertions != 2 {
		t.Fatalf("unexpected report: %#v", report)
	}
}

func TestRunQualitySuiteReportsForbiddenLeakage(t *testing.T) {
	scope := Scope{TenantID: "quality", ContinuityID: "workspace-a", ContinuityLine: "workspace"}
	suite := QualitySuite{Version: "1.0", Scenarios: []QualityScenario{{
		ID: "B01",
		Steps: []QualityStep{
			{Op: "put", Record: &Record{ID: "a", Scope: scope, Status: "active", Content: "wrong ops_exception_queue_refresh"}},
			{Op: "search", Query: &QualityQuery{Scope: scope, Text: "wrong", Forbidden: []string{"ops_exception_queue_refresh"}}},
		},
	}}}

	report := RunQualitySuite(context.Background(), newTestBackend(false), suite)
	if report.Pass || report.Metrics.ForbiddenLeakage != 1 || report.Metrics.Recall != 1 {
		t.Fatalf("expected forbidden leakage, got %#v", report)
	}
}

func TestLoadBackendQualityCasebook(t *testing.T) {
	suite, err := LoadQualitySuite("../../backend-casebook/scenarios.json")
	if err != nil {
		t.Fatal(err)
	}
	if len(suite.Scenarios) != 10 {
		t.Fatalf("expected 10 scenarios, got %d", len(suite.Scenarios))
	}
}
