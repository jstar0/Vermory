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

func TestConversationConsumerAppliesGlobalDefaultWithoutPersistingLocalOverride(t *testing.T) {
	ctx := context.Background()
	store := openTestStore(t)
	defaults := NewGlobalDefaultsService(store, "local")
	created, err := defaults.Set(ctx, SetGlobalDefaultRequest{
		OperationID: "conversation-global-language-set",
		Key:         "reply_language",
		Content:     "Default user-facing replies to Chinese unless the active task explicitly requests another language.",
	})
	requireNoError(t, err)
	llm := &recordingProvider{output: "English table deliverable"}
	service := NewConversationService(store, "local", llm, "test-model", ConversationServiceConfig{})

	_, err = service.Chat(ctx, ChatTurnRequest{
		OperationID: "conversation-local-english-override",
		Anchor:      ConversationAnchor{Channel: "web_chat", ThreadID: "mcm-table-task"},
		Message:     "For this task only, produce the table-facing deliverable in English.",
	})
	requireNoError(t, err)
	if len(llm.calls) != 1 {
		t.Fatalf("expected one provider call, got %d", len(llm.calls))
	}
	requireContains(t, llm.calls[0].ContextPacket, "Global defaults:\nDefault user-facing replies to Chinese")
	requireContains(t, llm.calls[0].Prompt, "For this task only")

	inspection, err := defaults.Inspect(ctx)
	requireNoError(t, err)
	if len(inspection.Defaults) != 1 || inspection.Defaults[0].ID != created.MemoryID || inspection.Defaults[0].LifecycleStatus != "active" {
		t.Fatalf("local override mutated the global default: %#v", inspection)
	}
	requireContains(t, inspection.Defaults[0].Content, "Chinese")
	requireNotContains(t, inspection.Defaults[0].Content, "English")

	llm.output = "新的中文回答"
	_, err = service.Chat(ctx, ChatTurnRequest{
		OperationID: "conversation-new-chinese-task",
		Anchor:      ConversationAnchor{Channel: "web_chat", ThreadID: "unrelated-chinese-task"},
		Message:     "请解释一个新的无关问题。",
	})
	requireNoError(t, err)
	requireContains(t, llm.calls[1].ContextPacket, "Default user-facing replies to Chinese")
	requireNotContains(t, llm.calls[1].ContextPacket, "table-facing deliverable in English")

	_, err = defaults.Forget(ctx, ForgetGlobalDefaultRequest{
		OperationID: "conversation-global-language-forget",
		MemoryID:    created.MemoryID,
	})
	requireNoError(t, err)
	_, err = service.Chat(ctx, ChatTurnRequest{
		OperationID: "conversation-after-global-forget",
		Anchor:      ConversationAnchor{Channel: "web_chat", ThreadID: "after-default-delete"},
		Message:     "继续一个新的任务。",
	})
	requireNoError(t, err)
	requireNotContains(t, llm.calls[2].ContextPacket, "Default user-facing replies to Chinese")
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

func TestConversationGovernanceRequiresExplicitConfirmationForAssistantOutput(t *testing.T) {
	store := openTestStore(t)
	llm := &recordingProvider{output: "The durable fact is ALPHA-17."}
	service := NewConversationService(store, "local", llm, "test-model", ConversationServiceConfig{})
	ctx := context.Background()
	anchor := ConversationAnchor{Channel: "web_chat", ThreadID: "matter-confirm"}

	turn, err := service.Chat(ctx, ChatTurnRequest{
		OperationID: "confirm-turn",
		Anchor:      anchor,
		Message:     "Report the durable fact.",
	})
	requireNoError(t, err)
	resolution, err := store.ResolveOrCreateConversation(ctx, "local", anchor)
	requireNoError(t, err)
	before, err := store.SearchActiveMemory(ctx, "local", resolution.ContinuityID, "ALPHA-17", 5)
	requireNoError(t, err)
	if len(before) != 0 {
		t.Fatalf("assistant output became active before confirmation: %#v", before)
	}

	confirmed, err := service.Confirm(ctx, ConfirmConversationMemoryRequest{
		OperationID:   "confirm-observation",
		Anchor:        anchor,
		ObservationID: turn.AssistantObservationID,
	})
	requireNoError(t, err)
	if confirmed.Status != "active" {
		t.Fatalf("confirmation did not create active memory: %#v", confirmed)
	}
	after, err := store.SearchActiveMemory(ctx, "local", resolution.ContinuityID, "ALPHA-17", 5)
	requireNoError(t, err)
	if len(after) != 1 || after[0].Content != "The durable fact is ALPHA-17." {
		t.Fatalf("confirmed memory is not retrievable: %#v", after)
	}
}

func TestConversationGovernanceRejectsObservationFromAnotherThread(t *testing.T) {
	store := openTestStore(t)
	llm := &recordingProvider{output: "thread-a fact"}
	service := NewConversationService(store, "local", llm, "test-model", ConversationServiceConfig{})
	ctx := context.Background()

	turn, err := service.Chat(ctx, ChatTurnRequest{
		OperationID: "cross-thread-turn",
		Anchor:      ConversationAnchor{Channel: "web_chat", ThreadID: "thread-a"},
		Message:     "produce a fact",
	})
	requireNoError(t, err)
	_, err = service.Confirm(ctx, ConfirmConversationMemoryRequest{
		OperationID:   "cross-thread-confirm",
		Anchor:        ConversationAnchor{Channel: "web_chat", ThreadID: "thread-b"},
		ObservationID: turn.AssistantObservationID,
	})
	if err == nil {
		t.Fatal("confirmation must reject an observation from another conversation")
	}
}

func TestConversationGovernanceCorrectsAndForgetsTargetedMemory(t *testing.T) {
	store := openTestStore(t)
	llm := &recordingProvider{output: "Use release flag old_mode."}
	service := NewConversationService(store, "local", llm, "test-model", ConversationServiceConfig{})
	ctx := context.Background()
	anchor := ConversationAnchor{Channel: "web_chat", ThreadID: "matter-lifecycle"}

	turn, err := service.Chat(ctx, ChatTurnRequest{
		OperationID: "lifecycle-turn",
		Anchor:      anchor,
		Message:     "Which release flag should be used?",
	})
	requireNoError(t, err)
	confirmed, err := service.Confirm(ctx, ConfirmConversationMemoryRequest{
		OperationID:   "lifecycle-confirm",
		Anchor:        anchor,
		ObservationID: turn.AssistantObservationID,
	})
	requireNoError(t, err)
	corrected, err := service.Correct(ctx, CorrectConversationMemoryRequest{
		OperationID: "lifecycle-correct",
		Anchor:      anchor,
		MemoryID:    confirmed.MemoryID,
		Content:     "Use release flag new_mode.",
	})
	requireNoError(t, err)
	if corrected.Memory.Status != "active" {
		t.Fatalf("correction did not create active memory: %#v", corrected)
	}

	resolution, err := store.ResolveOrCreateConversation(ctx, "local", anchor)
	requireNoError(t, err)
	matches, err := store.SearchActiveMemory(ctx, "local", resolution.ContinuityID, "release flag", 5)
	requireNoError(t, err)
	if len(matches) != 1 || strings.Contains(matches[0].Content, "old_mode") || !strings.Contains(matches[0].Content, "new_mode") {
		t.Fatalf("correction lifecycle is wrong: %#v", matches)
	}

	forgotten, err := service.Forget(ctx, ForgetConversationMemoryRequest{
		OperationID: "lifecycle-forget",
		Anchor:      anchor,
		MemoryID:    corrected.Memory.MemoryID,
	})
	requireNoError(t, err)
	if forgotten.Memory.Status != "deleted" {
		t.Fatalf("forget did not delete targeted memory: %#v", forgotten)
	}
	requireNoError(t, store.RebuildProjection(ctx, "local", resolution.ContinuityID))
	matches, err = store.SearchActiveMemory(ctx, "local", resolution.ContinuityID, "new_mode", 5)
	requireNoError(t, err)
	if len(matches) != 0 {
		t.Fatalf("forgotten memory returned after rebuild: %#v", matches)
	}
}

func TestConversationForgetRedactsOriginHistoryTurnReplayAndDelivery(t *testing.T) {
	store := openTestStore(t)
	const secret = "ORCHID-7419"
	llm := &recordingProvider{output: "The temporary recovery code is " + secret + "."}
	service := NewConversationService(store, "local", llm, "test-model", ConversationServiceConfig{})
	ctx := context.Background()
	anchor := ConversationAnchor{Channel: "web_chat", ThreadID: "matter-delete"}
	request := ChatTurnRequest{OperationID: "delete-turn", Anchor: anchor, Message: "Show the temporary code."}

	turn, err := service.Chat(ctx, request)
	requireNoError(t, err)
	confirmed, err := service.Confirm(ctx, ConfirmConversationMemoryRequest{
		OperationID:   "delete-confirm",
		Anchor:        anchor,
		ObservationID: turn.AssistantObservationID,
	})
	requireNoError(t, err)
	llm.output = "Use the governed recovery guidance."
	followup, err := service.Chat(ctx, ChatTurnRequest{
		OperationID: "delete-followup",
		Anchor:      anchor,
		Message:     "What recovery information is retained?",
	})
	requireNoError(t, err)
	_, err = service.Forget(ctx, ForgetConversationMemoryRequest{
		OperationID: "delete-forget",
		Anchor:      anchor,
		MemoryID:    confirmed.MemoryID,
	})
	requireNoError(t, err)

	replayed, err := service.Chat(ctx, request)
	requireNoError(t, err)
	if strings.Contains(replayed.Answer, secret) {
		t.Fatalf("replayed answer retained deleted content: %#v", replayed)
	}
	inspection, err := service.Inspect(ctx, anchor)
	requireNoError(t, err)
	encoded := inspectionText(inspection)
	if strings.Contains(encoded, secret) {
		t.Fatalf("inspection retained deleted content: %s", encoded)
	}
	if !strings.Contains(encoded, "[redacted]") {
		t.Fatalf("inspection did not retain a redacted marker: %s", encoded)
	}

	var deliveryBody string
	err = store.pool.QueryRow(ctx, `
SELECT context_body FROM memory_deliveries WHERE id = $1::uuid`, followup.DeliveryID).Scan(&deliveryBody)
	requireNoError(t, err)
	if strings.Contains(deliveryBody, secret) {
		t.Fatalf("delivery retained deleted content: %s", deliveryBody)
	}
}

func inspectionText(inspection ConversationInspection) string {
	var parts []string
	for _, observation := range inspection.Observations {
		parts = append(parts, observation.Content)
	}
	for _, memory := range inspection.Memories {
		parts = append(parts, memory.Content)
	}
	return strings.Join(parts, "\n")
}
