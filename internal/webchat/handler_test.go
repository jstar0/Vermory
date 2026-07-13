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

func TestGlobalDefaultsEndpointsUseServerOwnedTenantAndDurableState(t *testing.T) {
	handler, store := testHandler(t, provider.Mock{Output: "unused"})
	setResponse := performJSON(t, handler, http.MethodPost, "/v1/defaults/set", `{
  "operation_id":"http-default-set",
  "key":"reply_language",
  "content":"Default user-facing replies to Chinese unless the active task explicitly requests another language."
}`)
	if setResponse.Code != http.StatusOK {
		t.Fatalf("set returned %d: %s", setResponse.Code, setResponse.Body.String())
	}
	var created runtime.GlobalDefaultMutationReceipt
	decodeResponse(t, setResponse, &created)
	if created.ContinuityID == "" || created.MemoryID == "" || created.MemoryStatus != "active" {
		t.Fatalf("unexpected set receipt: %#v", created)
	}

	forbidden := performJSON(t, handler, http.MethodPost, "/v1/defaults/set", `{
  "operation_id":"http-default-forbidden-tenant",
  "key":"output_format",
  "content":"Use Markdown.",
  "tenant_id":"attacker"
}`)
	if forbidden.Code != http.StatusBadRequest {
		t.Fatalf("request-owned tenant was accepted: %d %s", forbidden.Code, forbidden.Body.String())
	}

	restarted := NewHandler(
		runtime.NewConversationService(store, "local", provider.Mock{Output: "unused"}, "test-model", runtime.ConversationServiceConfig{}),
		runtime.NewGlobalDefaultsService(store, "local"),
	)
	inspectRequest := httptest.NewRequest(http.MethodGet, "/v1/defaults", nil)
	inspectResponse := httptest.NewRecorder()
	restarted.ServeHTTP(inspectResponse, inspectRequest)
	if inspectResponse.Code != http.StatusOK {
		t.Fatalf("inspect returned %d: %s", inspectResponse.Code, inspectResponse.Body.String())
	}
	var inspection runtime.GlobalDefaultsInspection
	decodeResponse(t, inspectResponse, &inspection)
	if inspection.ContinuityID != created.ContinuityID || len(inspection.Defaults) != 1 || inspection.Defaults[0].ID != created.MemoryID {
		t.Fatalf("restarted handler lost durable default: %#v", inspection)
	}

	correctResponse := performJSON(t, restarted, http.MethodPost, "/v1/defaults/correct", `{
  "operation_id":"http-default-correct",
  "memory_id":"`+created.MemoryID+`",
  "content":"Default user-facing replies to Chinese."
}`)
	if correctResponse.Code != http.StatusOK {
		t.Fatalf("correct returned %d: %s", correctResponse.Code, correctResponse.Body.String())
	}
	var corrected runtime.GlobalDefaultMutationReceipt
	decodeResponse(t, correctResponse, &corrected)
	if corrected.MemoryID == created.MemoryID || corrected.MemoryStatus != "active" {
		t.Fatalf("unexpected correction receipt: %#v", corrected)
	}

	forgetResponse := performJSON(t, restarted, http.MethodPost, "/v1/defaults/forget", `{
  "operation_id":"http-default-forget",
  "memory_id":"`+corrected.MemoryID+`"
}`)
	if forgetResponse.Code != http.StatusOK {
		t.Fatalf("forget returned %d: %s", forgetResponse.Code, forgetResponse.Body.String())
	}
	var forgotten runtime.GlobalDefaultMutationReceipt
	decodeResponse(t, forgetResponse, &forgotten)
	if forgotten.MemoryStatus != "deleted" || forgotten.MemoryID != corrected.MemoryID {
		t.Fatalf("unexpected forget receipt: %#v", forgotten)
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
	var confirmationJSON map[string]any
	decodeResponse(t, confirmResponse, &confirmationJSON)
	if _, ok := confirmationJSON["memory_id"]; !ok {
		t.Fatalf("confirmation receipt does not use stable JSON fields: %s", confirmResponse.Body.String())
	}
	if _, ok := confirmationJSON["MemoryID"]; ok {
		t.Fatalf("confirmation receipt exposed Go field names: %s", confirmResponse.Body.String())
	}

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
	defaults := runtime.NewGlobalDefaultsService(store, "local")
	return NewHandler(service, defaults), store
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
