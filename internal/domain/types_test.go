package domain

import "testing"

func TestClaimStatusAllowsGovernanceLifecycle(t *testing.T) {
	statuses := []ClaimStatus{
		ClaimStatusDraft,
		ClaimStatusConfirmed,
		ClaimStatusActive,
		ClaimStatusStaleCandidate,
		ClaimStatusConflict,
		ClaimStatusArchived,
		ClaimStatusDeleted,
	}
	if len(statuses) != 7 {
		t.Fatalf("expected 7 statuses, got %d", len(statuses))
	}
}

func TestContinuityLineValues(t *testing.T) {
	got := []ContinuityLine{
		ContinuityLineWorkspace,
		ContinuityLineConversation,
		ContinuityLineGlobalDefaults,
	}
	want := []ContinuityLine{"workspace", "conversation", "global_defaults"}

	if len(got) != len(want) {
		t.Fatalf("expected %d continuity lines, got %d", len(want), len(got))
	}

	seen := make(map[ContinuityLine]struct{}, len(got))
	for i, line := range got {
		if line != want[i] {
			t.Fatalf("continuity line %d: expected %q, got %q", i, want[i], line)
		}
		if _, exists := seen[line]; exists {
			t.Fatalf("continuity line %q is duplicated", line)
		}
		seen[line] = struct{}{}
	}
}

func TestBridgeActionValues(t *testing.T) {
	got := []BridgeAction{
		BridgeActionPromote,
		BridgeActionLink,
		BridgeActionExport,
		BridgeActionAdopt,
		BridgeActionRebind,
	}
	want := []BridgeAction{"promote", "link", "export", "adopt", "rebind"}

	if len(got) != len(want) {
		t.Fatalf("expected %d bridge actions, got %d", len(want), len(got))
	}

	seen := make(map[BridgeAction]struct{}, len(got))
	for i, action := range got {
		if action != want[i] {
			t.Fatalf("bridge action %d: expected %q, got %q", i, want[i], action)
		}
		if _, exists := seen[action]; exists {
			t.Fatalf("bridge action %q is duplicated", action)
		}
		seen[action] = struct{}{}
	}
}

func TestAnchorStrengthValues(t *testing.T) {
	got := []AnchorStrength{
		AnchorStrengthStrong,
		AnchorStrengthWeak,
	}
	want := []AnchorStrength{"strong", "weak"}

	if len(got) != len(want) {
		t.Fatalf("expected %d anchor strengths, got %d", len(want), len(got))
	}

	seen := make(map[AnchorStrength]struct{}, len(got))
	for i, strength := range got {
		if strength != want[i] {
			t.Fatalf("anchor strength %d: expected %q, got %q", i, want[i], strength)
		}
		if _, exists := seen[strength]; exists {
			t.Fatalf("anchor strength %q is duplicated", strength)
		}
		seen[strength] = struct{}{}
	}
}

func TestDomainBoundaryTypesUsage(t *testing.T) {
	benchmark := BenchmarkName("public-benchmark")
	if benchmark != "public-benchmark" {
		t.Fatalf("expected benchmark name to be preserved, got %q", benchmark)
	}
	capability := BenchmarkCapability("continuity")
	if capability != "continuity" {
		t.Fatalf("expected benchmark capability to be preserved, got %q", capability)
	}

	space := ContinuitySpace{
		ID:             ID("space-1"),
		Name:           "workspace-alpha",
		Line:           ContinuityLineWorkspace,
		Anchor:         "repo:/workspace-alpha",
		AnchorStrength: AnchorStrengthStrong,
	}
	if space.AnchorStrength != AnchorStrengthStrong {
		t.Fatalf("expected anchor strength to be strong, got %q", space.AnchorStrength)
	}

	packet := Packet{
		ID:              ID("packet-1"),
		ProjectID:       ID("project-1"),
		CapsuleID:       ID("capsule-1"),
		TargetProfileID: TargetProfileID("conversation_export"),
	}
	if packet.TargetProfileID != "conversation_export" {
		t.Fatalf("expected target profile id to be preserved, got %q", packet.TargetProfileID)
	}

	log := AuditLog{
		ID:        ID("audit-1"),
		ProjectID: ID("project-1"),
		Action:    AuditAction("bridge.evaluate"),
		TargetID:  ID("target-1"),
	}
	if log.Action != "bridge.evaluate" {
		t.Fatalf("expected audit action to be preserved, got %q", log.Action)
	}
}
