package app

import (
	"strings"

	"vermory/internal/domain"
)

func toConfirmedDomainClaims(items []fixtureClaim) []domain.Claim {
	claims := make([]domain.Claim, 0, len(items))
	for _, item := range items {
		claims = append(claims, domain.Claim{
			Type:           item.Type,
			Content:        item.Content,
			Status:         domain.ClaimStatusConfirmed,
			VerifiedByUser: true,
		})
	}
	return claims
}

func plainSummary(claims []domain.Claim) string {
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
	return b.String()
}

func defaultStaleContext() string {
	return strings.Join([]string{
		"旧摘要：项目仍围绕个人基础设施控制台展开。",
		"旧摘要：比赛方向还没有从算法赛调整到蓝桥杯。",
		"旧摘要：系统依赖旧的第三方记忆平台包装方案。",
	}, "\n")
}
