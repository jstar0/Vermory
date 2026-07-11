package runner

import "strings"

func BuildChatContractPrompt(req ChatContractRequest) string {
	var parts []string
	if len(req.History) > 0 {
		parts = append(parts, "History:\n"+strings.Join(req.History, "\n"))
	}
	if strings.TrimSpace(req.ContinuityView) != "" {
		parts = append(parts, "Continuity:\n"+req.ContinuityView)
	}
	parts = append(parts, "Current turn:\n"+req.CurrentTurn)
	return strings.Join(parts, "\n\n")
}
