package runner

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	"vermory/internal/eval"
	"vermory/internal/provider"
)

func RunEvaluation(ctx context.Context, llm provider.Provider, opts EvaluationOptions) (EvaluationReport, error) {
	if llm == nil {
		return EvaluationReport{}, errors.New("runner: provider is required")
	}
	if opts.ArtifactStore == nil {
		return EvaluationReport{}, errors.New("runner: artifact store is required")
	}
	if strings.TrimSpace(opts.RunID) == "" {
		opts.RunID = "run"
	}
	if strings.TrimSpace(opts.ArtifactPrefix) == "" {
		opts.ArtifactPrefix = "platform-runs"
	}
	if strings.TrimSpace(opts.ProviderMode) == "" {
		opts.ProviderMode = "mock"
	}
	if strings.TrimSpace(opts.ProviderName) == "" {
		opts.ProviderName = "unknown"
	}
	if strings.TrimSpace(opts.SystemPrompt) == "" {
		opts.SystemPrompt = "You are evaluating whether the supplied context helps answer the task. Use only the task and supplied context."
	}
	if opts.MaxTokens <= 0 {
		opts.MaxTokens = 1024
	}

	report := EvaluationReport{
		RunID:        opts.RunID,
		TaskID:       opts.Task.ID,
		ProviderMode: opts.ProviderMode,
		ProviderName: opts.ProviderName,
		Model:        opts.Model,
		Results:      make([]BaselineResult, 0, 4),
	}

	for _, baseline := range baselines(opts) {
		result, err := runBaseline(ctx, llm, opts, baseline)
		if err != nil {
			return EvaluationReport{}, err
		}
		report.Results = append(report.Results, result)
	}

	reportArtifact, err := opts.ArtifactStore.Put(ctx, artifactKey(opts, "report.md"), []byte(MarkdownReport(report)))
	if err != nil {
		return EvaluationReport{}, err
	}
	report.ReportURI = reportArtifact.URI

	return report, nil
}

func runBaseline(ctx context.Context, llm provider.Provider, opts EvaluationOptions, baseline baselineSpec) (BaselineResult, error) {
	input := opts.Task.Prompt
	if strings.TrimSpace(baseline.Context) != "" {
		input = "Context:\n" + baseline.Context + "\n\nTask:\n" + opts.Task.Prompt
	}

	inputArtifact, err := opts.ArtifactStore.Put(ctx, artifactKey(opts, string(baseline.ID), "input.md"), []byte(input))
	if err != nil {
		return BaselineResult{}, err
	}
	packetArtifactURI := ""
	if baseline.ID == BaselineContextMeshPacket && strings.TrimSpace(baseline.Context) != "" {
		packetArtifact, err := opts.ArtifactStore.Put(ctx, artifactKey(opts, string(baseline.ID), "packet.md"), []byte(baseline.Context))
		if err != nil {
			return BaselineResult{}, err
		}
		packetArtifactURI = packetArtifact.URI
	}

	resp, err := llm.Generate(ctx, provider.GenerateRequest{
		Model:         opts.Model,
		System:        opts.SystemPrompt,
		Prompt:        opts.Task.Prompt,
		ContextPacket: baseline.Context,
		MaxTokens:     opts.MaxTokens,
	})
	if err != nil {
		return BaselineResult{}, fmt.Errorf("runner: baseline %s failed: %w", baseline.ID, err)
	}

	score := eval.ScoreOutput(opts.Task, resp.Output)
	scoreBytes, err := json.MarshalIndent(score, "", "  ")
	if err != nil {
		return BaselineResult{}, err
	}

	outputArtifact, err := opts.ArtifactStore.Put(ctx, artifactKey(opts, string(baseline.ID), "output.md"), []byte(resp.Output))
	if err != nil {
		return BaselineResult{}, err
	}
	scoreArtifact, err := opts.ArtifactStore.Put(ctx, artifactKey(opts, string(baseline.ID), "score.json"), scoreBytes)
	if err != nil {
		return BaselineResult{}, err
	}
	rawArtifactURI := ""
	if len(resp.RawArtifact) > 0 {
		rawArtifact, err := opts.ArtifactStore.Put(ctx, artifactKey(opts, string(baseline.ID), "raw.json"), resp.RawArtifact)
		if err != nil {
			return BaselineResult{}, err
		}
		rawArtifactURI = rawArtifact.URI
	}

	model := resp.Model
	if model == "" {
		model = opts.Model
	}
	return BaselineResult{
		Baseline:     baseline.ID,
		ProviderMode: opts.ProviderMode,
		ProviderName: opts.ProviderName,
		Model:        model,
		Output:       resp.Output,
		Score:        score,
		InputURI:     inputArtifact.URI,
		PacketURI:    packetArtifactURI,
		OutputURI:    outputArtifact.URI,
		RawURI:       rawArtifactURI,
		ScoreURI:     scoreArtifact.URI,
	}, nil
}

func RunConversationEvaluation(ctx context.Context, llm provider.Provider, opts ConversationEvaluationOptions) (ConversationEvaluationResult, error) {
	if llm == nil {
		return ConversationEvaluationResult{}, errors.New("runner: provider is required")
	}
	if opts.ArtifactStore == nil {
		return ConversationEvaluationResult{}, errors.New("runner: artifact store is required")
	}
	if strings.TrimSpace(opts.RunID) == "" {
		opts.RunID = "run"
	}
	if strings.TrimSpace(opts.ArtifactPrefix) == "" {
		opts.ArtifactPrefix = "platform-runs"
	}
	if strings.TrimSpace(opts.ProviderMode) == "" {
		opts.ProviderMode = "mock"
	}
	if strings.TrimSpace(opts.ProviderName) == "" {
		opts.ProviderName = "unknown"
	}
	if strings.TrimSpace(opts.ThreadID) == "" {
		opts.ThreadID = "thread"
	}
	if opts.MaxTokens <= 0 {
		opts.MaxTokens = 1024
	}

	prompt := BuildChatContractPrompt(ChatContractRequest{
		ThreadID:       opts.ThreadID,
		SystemFrame:    opts.SystemFrame,
		History:        opts.ThreadHistory,
		ContinuityView: opts.ContinuityView,
		CurrentTurn:    opts.CurrentTurn,
	})
	inputArtifact, err := opts.ArtifactStore.Put(ctx, artifactKey(EvaluationOptions{
		RunID:          opts.RunID,
		ArtifactPrefix: opts.ArtifactPrefix,
	}, "conversation", opts.ThreadID, "input.md"), []byte(prompt))
	if err != nil {
		return ConversationEvaluationResult{}, err
	}

	resp, err := llm.Generate(ctx, provider.GenerateRequest{
		Model:     opts.Model,
		System:    opts.SystemFrame,
		Prompt:    prompt,
		MaxTokens: opts.MaxTokens,
	})
	if err != nil {
		return ConversationEvaluationResult{}, fmt.Errorf("runner: conversation failed: %w", err)
	}

	score := eval.ScoreOutput(opts.Task, resp.Output)
	scoreBytes, err := json.MarshalIndent(score, "", "  ")
	if err != nil {
		return ConversationEvaluationResult{}, err
	}

	outputArtifact, err := opts.ArtifactStore.Put(ctx, artifactKey(EvaluationOptions{
		RunID:          opts.RunID,
		ArtifactPrefix: opts.ArtifactPrefix,
	}, "conversation", opts.ThreadID, "output.md"), []byte(resp.Output))
	if err != nil {
		return ConversationEvaluationResult{}, err
	}
	scoreArtifact, err := opts.ArtifactStore.Put(ctx, artifactKey(EvaluationOptions{
		RunID:          opts.RunID,
		ArtifactPrefix: opts.ArtifactPrefix,
	}, "conversation", opts.ThreadID, "score.json"), scoreBytes)
	if err != nil {
		return ConversationEvaluationResult{}, err
	}

	rawArtifactURI := ""
	if len(resp.RawArtifact) > 0 {
		rawArtifact, err := opts.ArtifactStore.Put(ctx, artifactKey(EvaluationOptions{
			RunID:          opts.RunID,
			ArtifactPrefix: opts.ArtifactPrefix,
		}, "conversation", opts.ThreadID, "raw.json"), resp.RawArtifact)
		if err != nil {
			return ConversationEvaluationResult{}, err
		}
		rawArtifactURI = rawArtifact.URI
	}

	model := resp.Model
	if model == "" {
		model = opts.Model
	}

	return ConversationEvaluationResult{
		RunID:        opts.RunID,
		TaskID:       opts.Task.ID,
		ProviderMode: opts.ProviderMode,
		ProviderName: opts.ProviderName,
		Model:        model,
		Output:       resp.Output,
		Score:        score,
		InputURI:     inputArtifact.URI,
		OutputURI:    outputArtifact.URI,
		RawURI:       rawArtifactURI,
		ScoreURI:     scoreArtifact.URI,
	}, nil
}

func artifactKey(opts EvaluationOptions, parts ...string) string {
	all := []string{opts.ArtifactPrefix, opts.RunID}
	all = append(all, parts...)
	return strings.Join(all, "/")
}
