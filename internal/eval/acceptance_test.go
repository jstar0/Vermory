package eval

import "testing"

func TestWorkspaceAcceptancePassesThreshold(t *testing.T) {
	score := AcceptanceScore{
		Continuation:  0.90,
		Isolation:     0.97,
		Groundedness:  0.88,
		Governance:    0.90,
		TargetFitness: 0.87,
		CostFriction:  0.70,
	}

	if !WorkspaceInternalReady(score).Pass {
		t.Fatalf("expected workspace acceptance to pass")
	}
}

func TestWorkspaceAcceptanceReportsFailedGates(t *testing.T) {
	score := AcceptanceScore{
		Continuation: 0.79,
		Isolation:    0.89,
		Groundedness: 0.78,
	}

	result := WorkspaceInternalReady(score)
	if result.Pass {
		t.Fatalf("expected workspace acceptance to fail")
	}
	if len(result.Failed) != 3 {
		t.Fatalf("expected three failed gates, got %d", len(result.Failed))
	}
}

func TestWorkspaceAcceptanceIgnoresNonGatingDimensionsInMinimalSlice(t *testing.T) {
	score := AcceptanceScore{
		Continuation:  0.90,
		Isolation:     0.95,
		Groundedness:  0.85,
		Governance:    0.00,
		TargetFitness: 0.00,
		CostFriction:  0.00,
	}

	result := WorkspaceInternalReady(score)
	if !result.Pass {
		t.Fatalf("expected workspace acceptance to ignore non-gating dimensions in this slice")
	}
	if len(result.Failed) != 0 {
		t.Fatalf("expected no failed gates, got %d", len(result.Failed))
	}
}

func TestConversationAcceptancePassesThreshold(t *testing.T) {
	score := AcceptanceScore{
		Continuation:  0.86,
		Isolation:     0.95,
		Groundedness:  0.84,
		Governance:    0.00,
		TargetFitness: 0.00,
		CostFriction:  0.00,
	}

	result := ConversationInternalReady(score)
	if !result.Pass {
		t.Fatalf("expected conversation acceptance to pass, failed gates: %v", result.Failed)
	}
}

func TestConversationAcceptanceReportsFailedGates(t *testing.T) {
	score := AcceptanceScore{
		Continuation: 0.79,
		Isolation:    0.89,
		Groundedness: 0.79,
	}

	result := ConversationInternalReady(score)
	if result.Pass {
		t.Fatalf("expected conversation acceptance to fail")
	}
	if len(result.Failed) != 3 {
		t.Fatalf("expected three failed gates, got %d: %v", len(result.Failed), result.Failed)
	}
}

func TestBridgeAcceptancePassesThreshold(t *testing.T) {
	score := AcceptanceScore{
		Continuation:  0.88,
		Isolation:     0.00,
		Groundedness:  0.84,
		Governance:    0.00,
		TargetFitness: 0.83,
		CostFriction:  0.00,
	}

	result := BridgeInternalReady(score)
	if !result.Pass {
		t.Fatalf("expected bridge acceptance to pass, failed gates: %v", result.Failed)
	}
}

func TestBridgeAcceptanceReportsFailedGates(t *testing.T) {
	score := AcceptanceScore{
		Continuation:  0.84,
		Groundedness:  0.79,
		TargetFitness: 0.79,
	}

	result := BridgeInternalReady(score)
	if result.Pass {
		t.Fatalf("expected bridge acceptance to fail")
	}
	if len(result.Failed) != 3 {
		t.Fatalf("expected three failed gates, got %d: %v", len(result.Failed), result.Failed)
	}
}
