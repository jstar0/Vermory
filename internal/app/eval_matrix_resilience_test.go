package app

import (
	"context"
	"errors"
	"strings"
	"testing"

	"vermory/internal/provider"
)

type failingProviderFactory func(model string) (provider.Provider, string, string, string, error)

func TestEvalMatrixContinuesWhenOneTaskRunFails(t *testing.T) {
	root := t.TempDir()

	report, err := evalMatrixWithFactory(context.Background(), EvalMatrixOptions{
		ArtifactRoot: root,
		Provider:     "mock",
		RunID:        "matrix-resilient",
		Models:       []string{"mock-a"},
		MaxTokens:    64,
	}, func(model string) (provider.Provider, string, string, string, error) {
		return scriptedProvider{
			failOnPromptSubstring: "真实案例",
		}, "mock", "mock", model, nil
	})
	if err != nil {
		t.Fatalf("expected matrix to continue and report failures, got error: %v", err)
	}

	if len(report.Models) != 1 {
		t.Fatalf("expected single model report, got %d", len(report.Models))
	}
	var failed int
	for _, task := range report.Models[0].Tasks {
		if task.Status == "error" {
			failed++
		}
	}
	if failed == 0 {
		t.Fatal("expected at least one task failure recorded")
	}
}

type scriptedProvider struct {
	failOnPromptSubstring string
}

func (p scriptedProvider) Generate(_ context.Context, req provider.GenerateRequest) (provider.GenerateResponse, error) {
	if p.failOnPromptSubstring != "" && strings.Contains(req.Prompt, p.failOnPromptSubstring) {
		return provider.GenerateResponse{}, errors.New("scripted provider failure")
	}
	return provider.GenerateResponse{
		Output:      req.Prompt + "\n" + req.ContextPacket,
		Model:       req.Model,
		RawArtifact: []byte(`{"ok":true}`),
	}, nil
}
