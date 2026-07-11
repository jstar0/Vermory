package app

import (
	"strings"
	"testing"

	"vermory/internal/domain"
)

func TestPreparePacketClaimsRemovesLegacySystemNamesFromExportText(t *testing.T) {
	claims := []domain.Claim{
		{
			Type:    domain.ClaimTypeConstraint,
			Content: "The product core is a self-designed Context Governance Engine, not MemOS, jstarctl, or a wrapper around a third-party memory framework.",
			Status:  domain.ClaimStatusConfirmed,
		},
	}

	prepared := preparePacketClaims(claims)

	if strings.Contains(prepared[0].Content, "MemOS") || strings.Contains(prepared[0].Content, "jstarctl") {
		t.Fatalf("expected exported claim text to remove legacy system names, got %q", prepared[0].Content)
	}
	if !strings.Contains(prepared[0].Content, "Context Governance Engine") {
		t.Fatalf("expected exported claim text to retain current product core, got %q", prepared[0].Content)
	}
}
