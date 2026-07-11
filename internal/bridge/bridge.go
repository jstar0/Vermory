package bridge

import (
	"strings"

	"vermory/internal/domain"
)

type AuditRecord struct {
	Action   domain.AuditAction
	SourceID string
	TargetID string
}

type PromoteRequest struct {
	ConversationID string
	WorkspaceID    string
	Claims         []domain.Claim
}

type ExportRequest struct {
	WorkspaceID      string
	TargetProfileID  domain.TargetProfileID
	ExportTitle      string
	ContinuityClaims []domain.Claim
}

type LinkRequest struct {
	PrimaryConversationID string
	LinkedConversationIDs []string
	Claims                []domain.Claim
}

type AdoptRequest struct {
	ContinuityID    string
	CandidateAnchor string
	ConfirmedAnchor string
}

type RebindRequest struct {
	ContinuityID string
	OldAnchor    string
	NewAnchor    string
}

type BridgeResult struct {
	Action          domain.BridgeAction
	ContinuityID    string
	SourceID        string
	TargetID        string
	TargetProfileID domain.TargetProfileID
	Title           string
	OldAnchor       string
	NewAnchor       string
	Claims          []domain.Claim
	Summary         string
	Audit           []AuditRecord
}

func Promote(req PromoteRequest) BridgeResult {
	claims := stableClaims(req.Claims)
	return BridgeResult{
		Action:       domain.BridgeActionPromote,
		ContinuityID: req.WorkspaceID,
		SourceID:     req.ConversationID,
		TargetID:     req.WorkspaceID,
		Claims:       claims,
		Summary:      summarizeClaims(claims),
		Audit: []AuditRecord{{
			Action:   "bridge_promote",
			SourceID: req.ConversationID,
			TargetID: req.WorkspaceID,
		}},
	}
}

func Export(req ExportRequest) BridgeResult {
	claims := stableClaims(req.ContinuityClaims)
	return BridgeResult{
		Action:          domain.BridgeActionExport,
		ContinuityID:    req.WorkspaceID,
		SourceID:        req.WorkspaceID,
		TargetID:        string(req.TargetProfileID),
		TargetProfileID: req.TargetProfileID,
		Title:           req.ExportTitle,
		Claims:          claims,
		Summary:         summarizeClaims(claims),
		Audit: []AuditRecord{{
			Action:   "bridge_export",
			SourceID: req.WorkspaceID,
			TargetID: string(req.TargetProfileID),
		}},
	}
}

func Link(req LinkRequest) BridgeResult {
	claims := stableClaims(req.Claims)
	audit := make([]AuditRecord, 0, len(req.LinkedConversationIDs))
	for _, linkedID := range req.LinkedConversationIDs {
		audit = append(audit, AuditRecord{
			Action:   "bridge_link",
			SourceID: linkedID,
			TargetID: req.PrimaryConversationID,
		})
	}
	return BridgeResult{
		Action:       domain.BridgeActionLink,
		ContinuityID: req.PrimaryConversationID,
		SourceID:     req.PrimaryConversationID,
		TargetID:     strings.Join(req.LinkedConversationIDs, ","),
		Claims:       claims,
		Summary:      summarizeClaims(claims),
		Audit:        audit,
	}
}

func Adopt(req AdoptRequest) BridgeResult {
	return BridgeResult{
		Action:       domain.BridgeActionAdopt,
		ContinuityID: req.ContinuityID,
		SourceID:     req.CandidateAnchor,
		TargetID:     req.ConfirmedAnchor,
		OldAnchor:    req.CandidateAnchor,
		NewAnchor:    req.ConfirmedAnchor,
		Audit: []AuditRecord{{
			Action:   "bridge_adopt",
			SourceID: req.CandidateAnchor,
			TargetID: req.ConfirmedAnchor,
		}},
	}
}

func Rebind(req RebindRequest) BridgeResult {
	return BridgeResult{
		Action:       domain.BridgeActionRebind,
		ContinuityID: req.ContinuityID,
		SourceID:     req.OldAnchor,
		TargetID:     req.NewAnchor,
		OldAnchor:    req.OldAnchor,
		NewAnchor:    req.NewAnchor,
		Audit: []AuditRecord{{
			Action:   "bridge_rebind",
			SourceID: req.OldAnchor,
			TargetID: req.NewAnchor,
		}},
	}
}

func stableClaims(claims []domain.Claim) []domain.Claim {
	out := make([]domain.Claim, 0, len(claims))
	for _, claim := range claims {
		if !claim.VerifiedByUser {
			continue
		}
		if claim.Status != domain.ClaimStatusConfirmed && claim.Status != domain.ClaimStatusActive {
			continue
		}
		out = append(out, claim)
	}
	return out
}

func summarizeClaims(claims []domain.Claim) string {
	var b strings.Builder
	for _, claim := range claims {
		b.WriteString("- ")
		b.WriteString(claim.Content)
		b.WriteString("\n")
	}
	return b.String()
}
