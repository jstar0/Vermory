package governance

import (
	"strings"

	"vermory/internal/domain"
)

func BuildCapsule(title string, purpose string, claims []domain.Claim) domain.Capsule {
	var b strings.Builder
	for _, claim := range claims {
		if claim.Status != domain.ClaimStatusConfirmed && claim.Status != domain.ClaimStatusActive {
			continue
		}
		if !claim.VerifiedByUser {
			continue
		}
		b.WriteString("- ")
		b.WriteString(claim.Content)
		b.WriteString("\n")
	}
	return domain.Capsule{
		Title:   title,
		Purpose: purpose,
		Summary: b.String(),
	}
}

func DraftClaim(claimType domain.ClaimType, content string) domain.Claim {
	return domain.Claim{
		Type:           claimType,
		Content:        content,
		Status:         domain.ClaimStatusDraft,
		VerifiedByUser: false,
	}
}

func ConfirmClaim(claim domain.Claim) domain.Claim {
	claim.Status = domain.ClaimStatusConfirmed
	claim.VerifiedByUser = true
	return claim
}

func SupersedeClaim(oldClaim domain.Claim, replacement domain.Claim) (domain.Claim, domain.Claim) {
	oldClaim.Status = domain.ClaimStatusArchived
	replacement.Status = domain.ClaimStatusConfirmed
	replacement.VerifiedByUser = true
	return oldClaim, replacement
}

func ArchiveClaim(claim domain.Claim) domain.Claim {
	claim.Status = domain.ClaimStatusArchived
	return claim
}

func DeleteClaim(claim domain.Claim) domain.Claim {
	claim.Status = domain.ClaimStatusDeleted
	claim.VerifiedByUser = false
	return claim
}
