package runtime

import (
	"context"
	"errors"
	"strings"
	"testing"

	"vermory/internal/provider"
)

type recordingProvider struct {
	calls  []provider.GenerateRequest
	output string
	err    error
}

func (p *recordingProvider) Generate(_ context.Context, request provider.GenerateRequest) (provider.GenerateResponse, error) {
	p.calls = append(p.calls, request)
	if p.err != nil {
		return provider.GenerateResponse{}, p.err
	}
	return provider.GenerateResponse{Output: p.output, Model: "test-model"}, nil
}

func TestConversationServiceIncludesPriorTurnsOnTheSameThread(t *testing.T) {
	store := openTestStore(t)
	llm := &recordingProvider{output: "assistant answer"}
	service := NewConversationService(store, "local", llm, "test-model", ConversationServiceConfig{})
	ctx := context.Background()
	anchor := ConversationAnchor{Channel: "web_chat", ThreadID: "matter-1"}

	first, err := service.Chat(ctx, ChatTurnRequest{
		OperationID: "turn-1",
		Anchor:      anchor,
		Message:     "first user message",
	})
	requireNoError(t, err)
	if first.Status != ChatTurnCompleted {
		t.Fatalf("first turn did not complete: %#v", first)
	}
	_, err = service.Chat(ctx, ChatTurnRequest{
		OperationID: "turn-2",
		Anchor:      anchor,
		Message:     "second user message",
	})
	requireNoError(t, err)

	if len(llm.calls) != 2 {
		t.Fatalf("expected two provider calls, got %d", len(llm.calls))
	}
	packet := llm.calls[1].ContextPacket
	for _, expected := range []string{"Recent conversation:", "User: first user message", "Assistant: assistant answer"} {
		if !strings.Contains(packet, expected) {
			t.Fatalf("second turn packet is missing %q: %s", expected, packet)
		}
	}
	if strings.Contains(packet, "second user message") {
		t.Fatalf("current user message was duplicated into history: %s", packet)
	}
}

func TestConversationServiceDoesNotCrossThreadBoundary(t *testing.T) {
	store := openTestStore(t)
	llm := &recordingProvider{output: "answer"}
	service := NewConversationService(store, "local", llm, "test-model", ConversationServiceConfig{})
	ctx := context.Background()

	_, err := service.Chat(ctx, ChatTurnRequest{
		OperationID: "isolated-1",
		Anchor:      ConversationAnchor{Channel: "web_chat", ThreadID: "matter-a"},
		Message:     "private fact alpha",
	})
	requireNoError(t, err)
	_, err = service.Chat(ctx, ChatTurnRequest{
		OperationID: "isolated-2",
		Anchor:      ConversationAnchor{Channel: "web_chat", ThreadID: "matter-b"},
		Message:     "unrelated question",
	})
	requireNoError(t, err)

	if strings.Contains(llm.calls[1].ContextPacket, "private fact alpha") {
		t.Fatalf("conversation context leaked across threads: %s", llm.calls[1].ContextPacket)
	}
}

func TestConversationServiceReplaysCompletedTurnWithoutCallingProviderAgain(t *testing.T) {
	store := openTestStore(t)
	llm := &recordingProvider{output: "stable answer"}
	service := NewConversationService(store, "local", llm, "test-model", ConversationServiceConfig{})
	request := ChatTurnRequest{
		OperationID: "idempotent-turn",
		Anchor:      ConversationAnchor{Channel: "web_chat", ThreadID: "matter-replay"},
		Message:     "one request",
	}

	first, err := service.Chat(context.Background(), request)
	requireNoError(t, err)
	second, err := service.Chat(context.Background(), request)
	requireNoError(t, err)

	if len(llm.calls) != 1 {
		t.Fatalf("replayed turn called provider %d times", len(llm.calls))
	}
	if first.ID != second.ID || second.Answer != "stable answer" || !second.Replayed {
		t.Fatalf("unexpected replay receipts: first=%#v second=%#v", first, second)
	}
}

func TestConversationServicePersistsFailedTurnWithoutAssistantObservation(t *testing.T) {
	store := openTestStore(t)
	llm := &recordingProvider{err: errors.New("provider unavailable")}
	service := NewConversationService(store, "local", llm, "test-model", ConversationServiceConfig{})
	request := ChatTurnRequest{
		OperationID: "failed-turn",
		Anchor:      ConversationAnchor{Channel: "web_chat", ThreadID: "matter-failure"},
		Message:     "retain this user input",
	}

	first, err := service.Chat(context.Background(), request)
	requireNoError(t, err)
	second, err := service.Chat(context.Background(), request)
	requireNoError(t, err)
	if first.Status != ChatTurnFailed || second.Status != ChatTurnFailed || !second.Replayed {
		t.Fatalf("failed turn was not persisted: first=%#v second=%#v", first, second)
	}
	if len(llm.calls) != 1 {
		t.Fatalf("failed replay called provider %d times", len(llm.calls))
	}

	resolution, err := store.ResolveOrCreateConversation(context.Background(), "local", request.Anchor)
	requireNoError(t, err)
	recent, err := store.ListRecentConversationObservations(context.Background(), "local", resolution.ContinuityID, "", 12)
	requireNoError(t, err)
	if len(recent) != 1 || recent[0].Kind != ObservationKindUserMessage || recent[0].Content != request.Message {
		t.Fatalf("provider failure wrote an assistant observation: %#v", recent)
	}
}
