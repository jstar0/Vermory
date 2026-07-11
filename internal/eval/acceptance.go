package eval

const (
	workspaceContinuationThreshold = 0.80
	workspaceIsolationThreshold    = 0.90
	workspaceGroundednessThreshold = 0.80

	conversationContinuationThreshold = 0.80
	conversationIsolationThreshold    = 0.90
	conversationGroundednessThreshold = 0.80

	bridgeContinuationThreshold  = 0.85
	bridgeGroundednessThreshold  = 0.80
	bridgeTargetFitnessThreshold = 0.80
)

// WorkspaceInternalReady applies the Task 3 minimal hard gate. It intentionally
// gates only continuation, isolation, and groundedness; the other acceptance
// dimensions remain in the score shape for later slices but are not enforced
// here yet.
func WorkspaceInternalReady(score AcceptanceScore) AcceptanceResult {
	var failed []string

	if score.Continuation < workspaceContinuationThreshold {
		failed = append(failed, "continuation")
	}
	if score.Isolation < workspaceIsolationThreshold {
		failed = append(failed, "isolation")
	}
	if score.Groundedness < workspaceGroundednessThreshold {
		failed = append(failed, "groundedness")
	}

	return AcceptanceResult{
		Pass:   len(failed) == 0,
		Failed: failed,
	}
}

// ConversationInternalReady applies the minimum V1 gate for weak-anchor
// continuity: the answer must continue the current thread, avoid unrelated
// contamination, and stay grounded in the supplied continuity view.
func ConversationInternalReady(score AcceptanceScore) AcceptanceResult {
	var failed []string

	if score.Continuation < conversationContinuationThreshold {
		failed = append(failed, "continuation")
	}
	if score.Isolation < conversationIsolationThreshold {
		failed = append(failed, "isolation")
	}
	if score.Groundedness < conversationGroundednessThreshold {
		failed = append(failed, "groundedness")
	}

	return AcceptanceResult{
		Pass:   len(failed) == 0,
		Failed: failed,
	}
}

// BridgeInternalReady applies the minimum V1 gate for governed bridge actions:
// the exported/promoted view must retain the relevant signal, remain grounded,
// and fit the target consumer profile. Governance audit completeness is modeled
// by the bridge package today and will become a hard score gate in a later slice.
func BridgeInternalReady(score AcceptanceScore) AcceptanceResult {
	var failed []string

	if score.Continuation < bridgeContinuationThreshold {
		failed = append(failed, "continuation")
	}
	if score.Groundedness < bridgeGroundednessThreshold {
		failed = append(failed, "groundedness")
	}
	if score.TargetFitness < bridgeTargetFitnessThreshold {
		failed = append(failed, "target_fitness")
	}

	return AcceptanceResult{
		Pass:   len(failed) == 0,
		Failed: failed,
	}
}
