package eval

import "strings"

type ContinuitySignalInput struct {
	ExpectedSignals  []string
	ForbiddenNoise   []string
	SourceEvidence   []string
	Output           string
	BaselineScore    float64
	ContextMeshScore float64
}

type ContinuitySignalScore struct {
	Continuation        float64 `json:"continuation"`
	Noise               float64 `json:"noise"`
	Groundedness        float64 `json:"groundedness"`
	RelativeImprovement float64 `json:"relative_improvement"`
}

func ScoreContinuitySignals(input ContinuitySignalInput) ContinuitySignalScore {
	output := strings.ToLower(input.Output)
	continuation := hitRate(output, input.ExpectedSignals)
	noise := hitRate(output, input.ForbiddenNoise)
	groundedness := groundednessRate(output, input.ExpectedSignals, input.ForbiddenNoise, input.SourceEvidence)

	return ContinuitySignalScore{
		Continuation:        continuation,
		Noise:               noise,
		Groundedness:        groundedness,
		RelativeImprovement: input.ContextMeshScore - input.BaselineScore,
	}
}

func hitRate(output string, phrases []string) float64 {
	if len(phrases) == 0 {
		return 0
	}
	hits := 0
	for _, phrase := range phrases {
		if strings.Contains(output, strings.ToLower(phrase)) {
			hits++
		}
	}
	return float64(hits) / float64(len(phrases))
}

func groundednessRate(output string, expected []string, forbidden []string, evidence []string) float64 {
	claims := append([]string{}, expected...)
	claims = append(claims, forbidden...)
	if len(claims) == 0 {
		return 1
	}

	supported := 0
	for _, claim := range claims {
		claim = strings.ToLower(claim)
		if !strings.Contains(output, claim) {
			continue
		}
		if containsPhrase(evidence, claim) {
			supported++
		}
	}
	mentioned := 0
	for _, claim := range claims {
		if strings.Contains(output, strings.ToLower(claim)) {
			mentioned++
		}
	}
	if mentioned == 0 {
		return 1
	}
	return float64(supported) / float64(mentioned)
}

func containsPhrase(items []string, phrase string) bool {
	for _, item := range items {
		if strings.Contains(strings.ToLower(item), phrase) {
			return true
		}
	}
	return false
}
