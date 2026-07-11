package bridge

import (
	"strings"
	"testing"

	"vermory/internal/domain"
)

func TestPromoteConversationToWorkspaceKeepsStableFactsAndAudit(t *testing.T) {
	result := Promote(PromoteRequest{
		ConversationID: "thread:openclaw:vendor-scorecard",
		WorkspaceID:    "workspace:vendor-scorecard",
		Claims: []domain.Claim{
			{ID: "stable", Type: domain.ClaimTypeDecision, Content: "Launch with CSV import first.", Status: domain.ClaimStatusConfirmed, VerifiedByUser: true},
			{ID: "noise", Type: domain.ClaimTypeRejectedOption, Content: "Mascot joke should stay chat noise.", Status: domain.ClaimStatusDraft, VerifiedByUser: false},
		},
	})

	if result.Action != domain.BridgeActionPromote {
		t.Fatalf("expected promote action, got %q", result.Action)
	}
	if len(result.Claims) != 1 {
		t.Fatalf("expected one promoted claim, got %d", len(result.Claims))
	}
	if strings.Contains(result.Summary, "Mascot") {
		t.Fatalf("expected noise to be excluded, got %q", result.Summary)
	}
	if len(result.Audit) != 1 || result.Audit[0].Action != "bridge_promote" {
		t.Fatalf("expected promote audit record, got %#v", result.Audit)
	}
}

func TestExportWorkspaceUsesTargetProfileAndAudit(t *testing.T) {
	result := Export(ExportRequest{
		WorkspaceID:      "workspace:contextmesh",
		TargetProfileID:  "team_handoff",
		ExportTitle:      "ContextMesh handoff",
		ContinuityClaims: []domain.Claim{{Content: "Use workspace resolver before memory search.", Status: domain.ClaimStatusConfirmed, VerifiedByUser: true}},
	})

	if result.Action != domain.BridgeActionExport {
		t.Fatalf("expected export action, got %q", result.Action)
	}
	if result.TargetProfileID != "team_handoff" {
		t.Fatalf("expected target profile, got %q", result.TargetProfileID)
	}
	if !strings.Contains(result.Summary, "Use workspace resolver") {
		t.Fatalf("expected exported summary to include claim, got %q", result.Summary)
	}
	if len(result.Audit) != 1 || result.Audit[0].Action != "bridge_export" {
		t.Fatalf("expected export audit record, got %#v", result.Audit)
	}
}

func TestLinkConversationsKeepsBoundaryAndAudit(t *testing.T) {
	result := Link(LinkRequest{
		PrimaryConversationID: "conversation:openclaw:housing-search",
		LinkedConversationIDs: []string{
			"conversation:phone:housing-search",
			"conversation:email-forward:housing-search",
		},
		Claims: []domain.Claim{
			{ID: "stable", Type: domain.ClaimTypeDecision, Content: "Seattle housing search keeps commute under 35 minutes.", Status: domain.ClaimStatusConfirmed, VerifiedByUser: true},
			{ID: "noise", Type: domain.ClaimTypeRejectedOption, Content: "Random travel tangent stays out.", Status: domain.ClaimStatusDraft, VerifiedByUser: false},
		},
	})

	if result.Action != domain.BridgeActionLink {
		t.Fatalf("expected link action, got %q", result.Action)
	}
	if result.ContinuityID != "conversation:openclaw:housing-search" {
		t.Fatalf("expected primary continuity id, got %q", result.ContinuityID)
	}
	if !strings.Contains(result.TargetID, "conversation:phone:housing-search") {
		t.Fatalf("expected linked targets, got %q", result.TargetID)
	}
	if strings.Contains(result.Summary, "Random travel") {
		t.Fatalf("expected noise to be excluded, got %q", result.Summary)
	}
	if len(result.Audit) != 2 || result.Audit[0].Action != "bridge_link" {
		t.Fatalf("expected link audit records, got %#v", result.Audit)
	}
}

func TestAdoptRecordsConfirmedAnchor(t *testing.T) {
	result := Adopt(AdoptRequest{
		ContinuityID:    "workspace:contextmesh",
		CandidateAnchor: "/tmp/ContextMesh-copy",
		ConfirmedAnchor: "/workspaces/vermory",
	})

	if result.Action != domain.BridgeActionAdopt {
		t.Fatalf("expected adopt action, got %q", result.Action)
	}
	if result.ContinuityID != "workspace:contextmesh" {
		t.Fatalf("expected continuity id to be preserved, got %q", result.ContinuityID)
	}
	if result.NewAnchor != "/workspaces/vermory" {
		t.Fatalf("expected confirmed anchor, got %q", result.NewAnchor)
	}
	if len(result.Audit) != 1 || result.Audit[0].Action != "bridge_adopt" {
		t.Fatalf("expected adopt audit record, got %#v", result.Audit)
	}
}

func TestRebindPreservesContinuityIDWithNewAnchor(t *testing.T) {
	result := Rebind(RebindRequest{
		ContinuityID: "workspace:contextmesh",
		OldAnchor:    "/old/path/ContextMesh",
		NewAnchor:    "/new/path/ContextMesh",
	})

	if result.Action != domain.BridgeActionRebind {
		t.Fatalf("expected rebind action, got %q", result.Action)
	}
	if result.ContinuityID != "workspace:contextmesh" {
		t.Fatalf("expected continuity id to be preserved, got %q", result.ContinuityID)
	}
	if result.NewAnchor != "/new/path/ContextMesh" {
		t.Fatalf("expected new anchor, got %q", result.NewAnchor)
	}
}
