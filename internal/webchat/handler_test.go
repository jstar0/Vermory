package webchat

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"

	"vermory/internal/provider"
	"vermory/internal/runtime"
)

func TestChatTurnReturnsPersistentReceipt(t *testing.T) {
	handler, _ := testHandler(t, provider.Mock{Output: "assistant response"})
	response := performJSON(t, handler, http.MethodPost, "/v1/chat/turn", `{
  "operation_id":"http-turn-1",
  "channel":"web_chat",
  "thread_id":"matter-http",
  "message":"hello"
}`)
	if response.Code != http.StatusOK {
		t.Fatalf("unexpected status %d: %s", response.Code, response.Body.String())
	}
	var receipt runtime.ChatTurnReceipt
	decodeResponse(t, response, &receipt)
	if receipt.Status != runtime.ChatTurnCompleted || receipt.Answer != "assistant response" || receipt.ContinuityID == "" || receipt.DeliveryID == "" {
		t.Fatalf("unexpected chat receipt: %#v", receipt)
	}
}

func TestChatTurnRejectsMissingAnchorAndAuthorityFields(t *testing.T) {
	handler, _ := testHandler(t, provider.Mock{Output: "unused"})
	missingThread := performJSON(t, handler, http.MethodPost, "/v1/chat/turn", `{
  "operation_id":"missing-thread",
  "channel":"web_chat",
  "message":"hello"
}`)
	if missingThread.Code != http.StatusBadRequest {
		t.Fatalf("missing thread returned %d: %s", missingThread.Code, missingThread.Body.String())
	}

	forbidden := performJSON(t, handler, http.MethodPost, "/v1/chat/turn", `{
  "operation_id":"forbidden-field",
  "channel":"web_chat",
  "thread_id":"matter-http",
  "message":"hello",
  "tenant_id":"attacker",
  "continuity_id":"attacker",
  "provider":"attacker"
}`)
	if forbidden.Code != http.StatusBadRequest {
		t.Fatalf("authority fields were accepted: %d %s", forbidden.Code, forbidden.Body.String())
	}
}

func TestChatTurnRejectsTrailingJSONAndOversizedBody(t *testing.T) {
	handler, _ := testHandler(t, provider.Mock{Output: "unused"})
	trailing := performJSON(t, handler, http.MethodPost, "/v1/chat/turn", `{"operation_id":"one","channel":"web_chat","thread_id":"matter","message":"hello"} {}`)
	if trailing.Code != http.StatusBadRequest {
		t.Fatalf("trailing JSON returned %d: %s", trailing.Code, trailing.Body.String())
	}

	large := `{"operation_id":"large","channel":"web_chat","thread_id":"matter","message":"` + strings.Repeat("x", int(maxRequestBodyBytes)) + `"}`
	oversized := performJSON(t, handler, http.MethodPost, "/v1/chat/turn", large)
	if oversized.Code != http.StatusRequestEntityTooLarge {
		t.Fatalf("oversized body returned %d: %s", oversized.Code, oversized.Body.String())
	}
}

func TestMemoryGovernanceAndInspectionUseExactConversation(t *testing.T) {
	handler, _ := testHandler(t, provider.Mock{Output: "fact for matter A"})
	turnResponse := performJSON(t, handler, http.MethodPost, "/v1/chat/turn", `{
  "operation_id":"govern-turn",
  "channel":"web_chat",
  "thread_id":"matter-a",
  "message":"produce a fact"
}`)
	var turn runtime.ChatTurnReceipt
	decodeResponse(t, turnResponse, &turn)

	confirmResponse := performJSON(t, handler, http.MethodPost, "/v1/memories/confirm", `{
  "operation_id":"govern-confirm",
  "channel":"web_chat",
  "thread_id":"matter-a",
  "observation_id":"`+turn.AssistantObservationID+`"
}`)
	if confirmResponse.Code != http.StatusOK {
		t.Fatalf("confirm failed: %d %s", confirmResponse.Code, confirmResponse.Body.String())
	}
	var confirmed runtime.MemoryReceipt
	decodeResponse(t, confirmResponse, &confirmed)

	inspectA := httptest.NewRecorder()
	handler.ServeHTTP(inspectA, httptest.NewRequest(http.MethodGet, "/v1/conversations/inspect?channel=web_chat&thread_id=matter-a", nil))
	if inspectA.Code != http.StatusOK || !strings.Contains(inspectA.Body.String(), "fact for matter A") {
		t.Fatalf("inspect A failed: %d %s", inspectA.Code, inspectA.Body.String())
	}
	inspectB := httptest.NewRecorder()
	handler.ServeHTTP(inspectB, httptest.NewRequest(http.MethodGet, "/v1/conversations/inspect?channel=web_chat&thread_id=matter-b", nil))
	if inspectB.Code != http.StatusNotFound || strings.Contains(inspectB.Body.String(), "fact for matter A") {
		t.Fatalf("inspect crossed continuity: %d %s", inspectB.Code, inspectB.Body.String())
	}

	forgetResponse := performJSON(t, handler, http.MethodPost, "/v1/memories/forget", `{
  "operation_id":"govern-forget",
  "channel":"web_chat",
  "thread_id":"matter-a",
  "memory_id":"`+confirmed.MemoryID+`"
}`)
	if forgetResponse.Code != http.StatusOK || strings.Contains(forgetResponse.Body.String(), "fact for matter A") {
		t.Fatalf("forget failed or disclosed content: %d %s", forgetResponse.Code, forgetResponse.Body.String())
	}
}

func TestProviderFailureReturnsPersistedBadGatewayReceipt(t *testing.T) {
	handler, _ := testHandler(t, failingProvider{})
	body := `{"operation_id":"provider-failure","channel":"web_chat","thread_id":"matter","message":"hello"}`
	first := performJSON(t, handler, http.MethodPost, "/v1/chat/turn", body)
	second := performJSON(t, handler, http.MethodPost, "/v1/chat/turn", body)
	if first.Code != http.StatusBadGateway || second.Code != http.StatusBadGateway {
		t.Fatalf("provider failure statuses: first=%d second=%d", first.Code, second.Code)
	}
	if strings.Contains(first.Body.String(), "database") || strings.Contains(first.Body.String(), "provider unavailable internal detail") {
		t.Fatalf("failure response exposed internal detail: %s", first.Body.String())
	}
}

type failingProvider struct{}

func (failingProvider) Generate(context.Context, provider.GenerateRequest) (provider.GenerateResponse, error) {
	return provider.GenerateResponse{}, errProviderUnavailable
}

var errProviderUnavailable = &providerError{"provider unavailable internal detail"}

type providerError struct{ message string }

func (e *providerError) Error() string { return e.message }

func testHandler(t *testing.T, llm provider.Provider) (http.Handler, *runtime.Store) {
	t.Helper()
	databaseURL := os.Getenv("VERMORY_TEST_DATABASE_URL")
	if databaseURL == "" {
		t.Skip("VERMORY_TEST_DATABASE_URL is not set")
	}
	store, err := runtime.OpenStore(context.Background(), databaseURL)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(store.Close)
	if err := store.Migrate(context.Background()); err != nil {
		t.Fatal(err)
	}
	if err := store.ResetForTest(context.Background()); err != nil {
		t.Fatal(err)
	}
	service := runtime.NewConversationService(store, "local", llm, "test-model", runtime.ConversationServiceConfig{})
	return NewHandler(service), store
}

func performJSON(t *testing.T, handler http.Handler, method, path, body string) *httptest.ResponseRecorder {
	t.Helper()
	request := httptest.NewRequest(method, path, bytes.NewBufferString(body))
	request.Header.Set("Content-Type", "application/json")
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, request)
	return response
}

func decodeResponse(t *testing.T, response *httptest.ResponseRecorder, target any) {
	t.Helper()
	if err := json.Unmarshal(response.Body.Bytes(), target); err != nil {
		t.Fatalf("decode response %q: %v", response.Body.String(), err)
	}
}
