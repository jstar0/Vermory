package runner

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"vermory/internal/artifact"
	"vermory/internal/eval"
	"vermory/internal/provider"
)

type recordingProvider struct {
	calls []provider.GenerateRequest
}

func (p *recordingProvider) Generate(_ context.Context, req provider.GenerateRequest) (provider.GenerateResponse, error) {
	p.calls = append(p.calls, req)
	output := req.Prompt
	if req.ContextPacket != "" {
		output += "\n" + req.ContextPacket
	}
	return provider.GenerateResponse{
		Output:      output,
		Model:       req.Model,
		RawArtifact: []byte(`{"ok":true}`),
	}, nil
}

func TestRunEvaluationRunsAllBaselinesAndStoresArtifacts(t *testing.T) {
	root := t.TempDir()
	recorder := &recordingProvider{}

	report, err := RunEvaluation(context.Background(), recorder, EvaluationOptions{
		RunID:         "run-test",
		ProviderMode:  "real",
		ProviderName:  "test-provider",
		Model:         "test-model",
		ArtifactStore: artifact.NewLocalStore(root),
		Task: eval.Task{
			ID:             "task-1",
			Prompt:         "Explain the current project direction.",
			MustInclude:    []string{"ContextMesh"},
			MustNotInclude: []string{"jstarctl"},
		},
		StaleContext:   "The project is jstarctl.",
		PlainSummary:   "The project is ContextMesh.",
		ContextPacket:  "The confirmed platform name is ContextMesh.",
		MaxTokens:      128,
		SystemPrompt:   "Answer as a platform evaluator.",
		ArtifactPrefix: "platform-runs",
	})
	if err != nil {
		t.Fatalf("RunEvaluation returned error: %v", err)
	}

	if len(report.Results) != 4 {
		t.Fatalf("expected 4 baseline results, got %d", len(report.Results))
	}
	if len(recorder.calls) != 4 {
		t.Fatalf("expected 4 provider calls, got %d", len(recorder.calls))
	}

	if recorder.calls[0].ContextPacket != "" {
		t.Fatalf("no_context baseline should not pass context, got %q", recorder.calls[0].ContextPacket)
	}
	if recorder.calls[1].ContextPacket != "The project is jstarctl." {
		t.Fatalf("stale_context baseline used wrong context: %q", recorder.calls[1].ContextPacket)
	}
	if recorder.calls[3].ContextPacket != "The confirmed platform name is ContextMesh." {
		t.Fatalf("contextmesh_packet baseline used wrong context: %q", recorder.calls[3].ContextPacket)
	}

	contextmesh := report.Results[3]
	if contextmesh.Baseline != BaselineContextMeshPacket {
		t.Fatalf("expected contextmesh result last, got %s", contextmesh.Baseline)
	}
	if contextmesh.Score.MustIncludeHitRate != 1 {
		t.Fatalf("expected contextmesh baseline to hit required context, got %#v", contextmesh.Score)
	}
	if len(report.Results[1].Score.MustNotIncludeViolations) == 0 {
		t.Fatalf("expected stale_context to violate stale forbidden term")
	}

	expectedFiles := []string{
		"platform-runs/run-test/no_context/input.md",
		"platform-runs/run-test/no_context/output.md",
		"platform-runs/run-test/no_context/raw.json",
		"platform-runs/run-test/no_context/score.json",
		"platform-runs/run-test/contextmesh_packet/packet.md",
		"platform-runs/run-test/report.md",
	}
	for _, file := range expectedFiles {
		if _, err := os.Stat(filepath.Join(root, file)); err != nil {
			t.Fatalf("expected artifact %s: %v", file, err)
		}
	}
}

func TestRunConversationEvaluationStoresArtifactsAndScoresOutput(t *testing.T) {
	root := t.TempDir()
	recorder := &recordingProvider{}

	report, err := RunConversationEvaluation(context.Background(), recorder, ConversationEvaluationOptions{
		RunID:          "chat-run",
		ProviderMode:   "real",
		ProviderName:   "test-provider",
		Model:          "test-model",
		SystemFrame:    "You are a chat assistant.",
		ThreadID:       "thread-1",
		ThreadHistory:  []string{"User: hello", "Assistant: hi"},
		ContinuityView: "Current matter: housing search in Hangzhou",
		CurrentTurn:    "What about budget limits for ContextMesh?",
		ArtifactStore:  artifact.NewLocalStore(root),
		Task: eval.Task{
			ID:             "chat-task-1",
			Prompt:         "Answer the user's current turn about ContextMesh budget limits.",
			MustInclude:    []string{"ContextMesh"},
			MustNotInclude: []string{"jstarctl"},
		},
		MaxTokens: 128,
	})
	if err != nil {
		t.Fatalf("RunConversationEvaluation returned error: %v", err)
	}

	if len(recorder.calls) != 1 {
		t.Fatalf("expected 1 provider call, got %d", len(recorder.calls))
	}
	if recorder.calls[0].System != "You are a chat assistant." {
		t.Fatalf("expected provider system frame, got %q", recorder.calls[0].System)
	}
	if !strings.Contains(recorder.calls[0].Prompt, "History:\nUser: hello\nAssistant: hi") {
		t.Fatalf("expected history in provider prompt, got %q", recorder.calls[0].Prompt)
	}
	if !strings.Contains(recorder.calls[0].Prompt, "Current turn:\nWhat about budget limits for ContextMesh?") {
		t.Fatalf("expected current turn in provider prompt, got %q", recorder.calls[0].Prompt)
	}
	if strings.Contains(recorder.calls[0].Prompt, "System:\n") || strings.Contains(recorder.calls[0].Prompt, "You are a chat assistant.") {
		t.Fatalf("prompt should not duplicate system frame, got %q", recorder.calls[0].Prompt)
	}
	if report.Score.MustIncludeHitRate != 1 {
		t.Fatalf("expected full include hit rate, got %#v", report.Score)
	}
	if report.InputURI == "" || report.OutputURI == "" || report.ScoreURI == "" {
		t.Fatalf("expected stored artifact URIs, got %#v", report)
	}

	expectedFiles := []string{
		"platform-runs/chat-run/conversation/thread-1/input.md",
		"platform-runs/chat-run/conversation/thread-1/output.md",
		"platform-runs/chat-run/conversation/thread-1/raw.json",
		"platform-runs/chat-run/conversation/thread-1/score.json",
	}
	for _, file := range expectedFiles {
		if _, err := os.Stat(filepath.Join(root, file)); err != nil {
			t.Fatalf("expected artifact %s: %v", file, err)
		}
	}
}

func TestRunConversationEvaluationIsolatesArtifactsByThreadID(t *testing.T) {
	root := t.TempDir()
	recorder := &recordingProvider{}
	store := artifact.NewLocalStore(root)
	baseTask := eval.Task{
		ID:             "chat-task-2",
		Prompt:         "Answer the user's current turn about ContextMesh budget limits.",
		MustInclude:    []string{"ContextMesh"},
		MustNotInclude: []string{"jstarctl"},
	}

	first, err := RunConversationEvaluation(context.Background(), recorder, ConversationEvaluationOptions{
		RunID:         "chat-run",
		ProviderMode:  "real",
		ProviderName:  "test-provider",
		Model:         "test-model",
		ThreadID:      "thread-a",
		CurrentTurn:   "Tell me about ContextMesh budget limits.",
		ArtifactStore: store,
		Task:          baseTask,
	})
	if err != nil {
		t.Fatalf("first RunConversationEvaluation returned error: %v", err)
	}

	second, err := RunConversationEvaluation(context.Background(), recorder, ConversationEvaluationOptions{
		RunID:         "chat-run",
		ProviderMode:  "real",
		ProviderName:  "test-provider",
		Model:         "test-model",
		ThreadID:      "thread-b",
		CurrentTurn:   "Tell me about ContextMesh budget limits.",
		ArtifactStore: store,
		Task:          baseTask,
	})
	if err != nil {
		t.Fatalf("second RunConversationEvaluation returned error: %v", err)
	}

	if first.InputURI == second.InputURI {
		t.Fatalf("expected different input URIs for different threads, got %q", first.InputURI)
	}
	if first.OutputURI == second.OutputURI {
		t.Fatalf("expected different output URIs for different threads, got %q", first.OutputURI)
	}

	expectedFiles := []string{
		"platform-runs/chat-run/conversation/thread-a/input.md",
		"platform-runs/chat-run/conversation/thread-b/input.md",
	}
	for _, file := range expectedFiles {
		if _, err := os.Stat(filepath.Join(root, file)); err != nil {
			t.Fatalf("expected artifact %s: %v", file, err)
		}
	}
}
