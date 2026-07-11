package governance

import (
	"testing"

	"vermory/internal/domain"
)

func TestBuildCapsuleUsesOnlyConfirmedClaims(t *testing.T) {
	claims := []domain.Claim{
		{ID: "confirmed", Content: "confirmed context", Status: domain.ClaimStatusConfirmed, VerifiedByUser: true},
		{ID: "active", Content: "active context", Status: domain.ClaimStatusActive, VerifiedByUser: true},
		{ID: "draft", Content: "draft context", Status: domain.ClaimStatusDraft, VerifiedByUser: true},
		{ID: "unverified", Content: "unverified context", Status: domain.ClaimStatusConfirmed, VerifiedByUser: false},
		{ID: "archived", Content: "archived context", Status: domain.ClaimStatusArchived, VerifiedByUser: true},
	}
	capsule := BuildCapsule("ContextMesh", "handoff", claims)
	if capsule.Summary != "- confirmed context\n- active context\n" {
		t.Fatalf("unexpected summary: %q", capsule.Summary)
	}
}

func TestClaimLifecycleConfirmSupersedeArchiveDelete(t *testing.T) {
	draft := DraftClaim(domain.ClaimTypeDecision, "Use workspace resolver first.")
	if draft.Status != domain.ClaimStatusDraft || draft.VerifiedByUser {
		t.Fatalf("expected unverified draft, got %#v", draft)
	}

	confirmed := ConfirmClaim(draft)
	if confirmed.Status != domain.ClaimStatusConfirmed || !confirmed.VerifiedByUser {
		t.Fatalf("expected confirmed verified claim, got %#v", confirmed)
	}

	replacement := DraftClaim(domain.ClaimTypeDecision, "Use workspace resolver before memory search.")
	superseded, active := SupersedeClaim(confirmed, replacement)
	if superseded.Status != domain.ClaimStatusArchived {
		t.Fatalf("expected superseded claim to be archived, got %#v", superseded)
	}
	if active.Status != domain.ClaimStatusConfirmed || !active.VerifiedByUser {
		t.Fatalf("expected replacement to be confirmed, got %#v", active)
	}

	archived := ArchiveClaim(active)
	if archived.Status != domain.ClaimStatusArchived {
		t.Fatalf("expected archived claim, got %#v", archived)
	}

	deleted := DeleteClaim(archived)
	if deleted.Status != domain.ClaimStatusDeleted || deleted.VerifiedByUser {
		t.Fatalf("expected deleted claim to be hidden from export, got %#v", deleted)
	}
}
