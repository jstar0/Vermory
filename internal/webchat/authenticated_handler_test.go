package webchat

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"vermory/internal/authn"
	"vermory/internal/provider"
	"vermory/internal/runtime"
)

func TestAuthenticatedHandlerRejectsMissingMalformedUnknownExpiredAndRevokedTokens(t *testing.T) {
	_, store := testHandler(t, provider.Mock{Output: "unused"})
	authenticator := staticAuthenticator{
		errors: map[string]error{
			"unknown-token": ErrSyntheticUnknownToken,
			"expired-token": authn.ErrAuthenticationFailed,
			"revoked-token": authn.ErrAuthenticationFailed,
		},
	}
	handler := NewAuthenticatedHandler(store, provider.Mock{Output: "unused"}, "test-model", authenticator)

	tests := []struct {
		name   string
		header []string
	}{
		{name: "missing"},
		{name: "basic", header: []string{"Basic abc"}},
		{name: "empty bearer", header: []string{"Bearer "}},
		{name: "multiple", header: []string{"Bearer unknown-token", "Bearer other-token"}},
		{name: "unknown", header: []string{"Bearer unknown-token"}},
		{name: "expired", header: []string{"Bearer expired-token"}},
		{name: "revoked", header: []string{"Bearer revoked-token"}},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			request := httptest.NewRequest(http.MethodPost, "/v1/chat/turn", strings.NewReader(`{"operation_id":"auth-fail","channel":"web_chat","thread_id":"auth","message":"hello"}`))
			request.Header.Set("Content-Type", "application/json")
			for _, value := range test.header {
				request.Header.Add("Authorization", value)
			}
			response := httptest.NewRecorder()
			handler.ServeHTTP(response, request)
			if response.Code != http.StatusUnauthorized {
				t.Fatalf("unexpected status %d: %s", response.Code, response.Body.String())
			}
			for _, secret := range []string{"unknown-token", "expired-token", "revoked-token", "identity-a"} {
				if strings.Contains(response.Body.String(), secret) {
					t.Fatalf("authentication error leaked %q: %s", secret, response.Body.String())
				}
			}
		})
	}
}

func TestAuthenticatedHandlerUsesPrincipalTenantAndRolePolicy(t *testing.T) {
	_, store := testHandler(t, provider.Mock{Output: "unused"})
	authenticator := staticAuthenticator{principals: map[string]authn.Principal{
		"client-a":   principal("identity-a", authn.RoleClient),
		"operator-a": principal("identity-a", authn.RoleOperator),
	}}
	handler := NewAuthenticatedHandler(store, provider.Mock{Output: "authenticated answer"}, "test-model", authenticator)

	chat := performAuthenticatedJSON(t, handler, "client-a", http.MethodPost, "/v1/chat/turn", `{
  "operation_id":"authenticated-chat-a",
  "channel":"web_chat",
  "thread_id":"shared-anchor",
  "message":"hello"
}`)
	if chat.Code != http.StatusOK {
		t.Fatalf("client chat failed: %d %s", chat.Code, chat.Body.String())
	}
	resolution, err := store.ResolveConversation(context.Background(), "identity-a", runtime.ConversationAnchor{Channel: "web_chat", ThreadID: "shared-anchor"})
	if err != nil || resolution.Status != runtime.ResolutionResolved {
		t.Fatalf("principal tenant was not persisted: resolution=%#v err=%v", resolution, err)
	}

	clientGovernance := performAuthenticatedJSON(t, handler, "client-a", http.MethodPost, "/v1/defaults/set", `{
  "operation_id":"client-default-denied",
  "key":"reply_language",
  "content":"Chinese"
}`)
	if clientGovernance.Code != http.StatusForbidden {
		t.Fatalf("client governance returned %d: %s", clientGovernance.Code, clientGovernance.Body.String())
	}

	operatorGovernance := performAuthenticatedJSON(t, handler, "operator-a", http.MethodPost, "/v1/defaults/set", `{
  "operation_id":"operator-default-a",
  "key":"reply_language",
  "content":"Default replies to Chinese."
}`)
	if operatorGovernance.Code != http.StatusOK {
		t.Fatalf("operator governance failed: %d %s", operatorGovernance.Code, operatorGovernance.Body.String())
	}
	inspection, err := runtime.NewGlobalDefaultsService(store, "identity-a").Inspect(context.Background())
	if err != nil || len(inspection.Defaults) != 1 {
		t.Fatalf("operator mutation was not tenant-scoped: inspection=%#v err=%v", inspection, err)
	}
}

func TestAuthenticatedHandlerHidesCrossTenantResourcesAndRejectsRequestAuthority(t *testing.T) {
	_, store := testHandler(t, provider.Mock{Output: "unused"})
	authenticator := staticAuthenticator{principals: map[string]authn.Principal{
		"operator-a": principal("identity-a", authn.RoleOperator),
		"operator-b": principal("identity-b", authn.RoleOperator),
	}}
	handler := NewAuthenticatedHandler(store, provider.Mock{Output: "unused"}, "test-model", authenticator)

	createdResponse := performAuthenticatedJSON(t, handler, "operator-a", http.MethodPost, "/v1/defaults/set", `{
  "operation_id":"cross-tenant-default-a",
  "key":"private_fact",
  "content":"TENANT-A-ONLY"
}`)
	if createdResponse.Code != http.StatusOK {
		t.Fatalf("seed default failed: %d %s", createdResponse.Code, createdResponse.Body.String())
	}
	var created runtime.GlobalDefaultMutationReceipt
	decodeResponse(t, createdResponse, &created)

	crossTenant := performAuthenticatedJSON(t, handler, "operator-b", http.MethodPost, "/v1/defaults/correct", `{
  "operation_id":"cross-tenant-attack-b",
  "memory_id":"`+created.MemoryID+`",
  "content":"attacker replacement"
}`)
	if crossTenant.Code != http.StatusBadRequest && crossTenant.Code != http.StatusNotFound {
		t.Fatalf("unexpected cross-tenant status %d: %s", crossTenant.Code, crossTenant.Body.String())
	}
	for _, forbidden := range []string{created.MemoryID, "identity-a", "identity-b", "operator-b", "TENANT-A-ONLY"} {
		if strings.Contains(crossTenant.Body.String(), forbidden) {
			t.Fatalf("cross-tenant error leaked %q: %s", forbidden, crossTenant.Body.String())
		}
	}

	authority := performAuthenticatedJSON(t, handler, "operator-b", http.MethodPost, "/v1/chat/turn", `{
  "operation_id":"authority-attack",
  "channel":"web_chat",
  "thread_id":"shared-anchor",
  "message":"hello",
  "tenant_id":"identity-a"
}`)
	if authority.Code != http.StatusBadRequest {
		t.Fatalf("request-owned tenant was accepted: %d %s", authority.Code, authority.Body.String())
	}
	if strings.Contains(authority.Body.String(), "identity-a") {
		t.Fatalf("authority rejection echoed tenant: %s", authority.Body.String())
	}
}

type staticAuthenticator struct {
	principals map[string]authn.Principal
	errors     map[string]error
}

func (authenticator staticAuthenticator) Authenticate(_ context.Context, raw string) (authn.Principal, error) {
	if err := authenticator.errors[raw]; err != nil {
		return authn.Principal{}, err
	}
	if principal, ok := authenticator.principals[raw]; ok {
		return principal, nil
	}
	return authn.Principal{}, authn.ErrAuthenticationFailed
}

func principal(tenantID string, role authn.Role) authn.Principal {
	return authn.Principal{
		TokenID:   "synthetic-token-id",
		TenantID:  tenantID,
		SubjectID: "synthetic-subject",
		Role:      role,
		ExpiresAt: time.Now().UTC().Add(time.Hour),
	}
}

func performAuthenticatedJSON(t *testing.T, handler http.Handler, token, method, path, body string) *httptest.ResponseRecorder {
	t.Helper()
	request := httptest.NewRequest(method, path, strings.NewReader(body))
	request.Header.Set("Content-Type", "application/json")
	request.Header.Set("Authorization", "Bearer "+token)
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, request)
	return response
}

var ErrSyntheticUnknownToken = errors.Join(authn.ErrAuthenticationFailed, errors.New("synthetic lookup detail that must stay private"))
