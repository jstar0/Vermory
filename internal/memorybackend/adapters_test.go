package memorybackend

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestMem0BackendWritesAndSearchesWithinExplicitScope(t *testing.T) {
	t.Helper()
	var putBody map[string]any
	var searchBody map[string]any
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch {
		case r.Method == http.MethodPost && r.URL.Path == "/memories":
			decodeTestJSON(t, r, &putBody)
			_, _ = w.Write([]byte(`{"results":[{"id":"provider-1","memory":"PostgreSQL is current","event":"ADD"}]}`))
		case r.Method == http.MethodPost && r.URL.Path == "/search":
			decodeTestJSON(t, r, &searchBody)
			_, _ = w.Write([]byte(`{"results":[{"id":"provider-1","memory":"PostgreSQL is current","score":0.91,"metadata":{"record_id":"record-1","source_id":"decision:database","source_version":"2","lifecycle_status":"active","continuity_line":"workspace"},"user_id":"tenant-a","agent_id":"workspace-a"}]}`))
		default:
			http.Error(w, "unexpected request", http.StatusNotFound)
		}
	}))
	defer server.Close()

	backend, err := NewMem0Backend(Mem0Config{BaseURL: server.URL, Client: server.Client()})
	if err != nil {
		t.Fatal(err)
	}
	record := Record{
		ID:       "record-1",
		Scope:    Scope{TenantID: "tenant-a", ContinuityID: "workspace-a", ContinuityLine: "workspace"},
		SourceID: "decision:database", SourceVersion: 2, Status: "active", Content: "PostgreSQL is current",
	}
	if err := backend.Put(context.Background(), record); err != nil {
		t.Fatal(err)
	}
	results, err := backend.Search(context.Background(), Query{Scope: record.Scope, Text: "current database", Limit: 5})
	if err != nil {
		t.Fatal(err)
	}
	if len(results) != 1 || results[0].Record.ID != record.ID || results[0].Score != 0.91 {
		t.Fatalf("unexpected results: %#v", results)
	}
	if putBody["user_id"] != "tenant-a" || putBody["agent_id"] != "workspace-a" || putBody["infer"] != false {
		t.Fatalf("put did not preserve scope or exact-storage mode: %#v", putBody)
	}
	filters, _ := searchBody["filters"].(map[string]any)
	if filters["user_id"] != "tenant-a" || filters["agent_id"] != "workspace-a" {
		t.Fatalf("search did not enforce both scope dimensions: %#v", searchBody)
	}
}

func TestMem0ResetScopeUsesServerSideScopedDelete(t *testing.T) {
	var userID string
	var agentID string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodDelete || r.URL.Path != "/memories" {
			http.Error(w, "unexpected request", http.StatusNotFound)
			return
		}
		userID = r.URL.Query().Get("user_id")
		agentID = r.URL.Query().Get("agent_id")
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"message":"All relevant memories deleted"}`))
	}))
	defer server.Close()

	backend, err := NewMem0Backend(Mem0Config{BaseURL: server.URL, Client: server.Client()})
	if err != nil {
		t.Fatal(err)
	}
	if err := backend.ResetScope(context.Background(), Scope{TenantID: "tenant-a", ContinuityID: "workspace-a"}); err != nil {
		t.Fatal(err)
	}
	if userID != "tenant-a" || agentID != "workspace-a" {
		t.Fatalf("scope delete was not constrained: user=%q agent=%q", userID, agentID)
	}
}

func TestSupermemoryBackendUsesOpaqueContainerAndPreservesContextMeshID(t *testing.T) {
	var putBody map[string]any
	var searchBody map[string]any
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch {
		case r.Method == http.MethodPost && r.URL.Path == "/v4/memories":
			decodeTestJSON(t, r, &putBody)
			_, _ = w.Write([]byte(`{"documentId":"doc-1","memories":[{"id":"provider-1","memory":"PostgreSQL is current","isStatic":false,"createdAt":"2026-01-01T00:00:00Z","forgetAfter":null,"forgetReason":null,"metadata":{"record_id":"record-1"}}]}`))
		case r.Method == http.MethodPost && r.URL.Path == "/v4/search":
			decodeTestJSON(t, r, &searchBody)
			_, _ = w.Write([]byte(`{"results":[{"id":"provider-1","memory":"PostgreSQL is current","similarity":0.92,"metadata":{"record_id":"record-1","source_id":"decision:database","source_version":"2","lifecycle_status":"active","continuity_line":"workspace"}}],"timing":10,"total":1}`))
		default:
			http.Error(w, "unexpected request", http.StatusNotFound)
		}
	}))
	defer server.Close()

	backend, err := NewSupermemoryBackend(SupermemoryConfig{BaseURL: server.URL, Client: server.Client()})
	if err != nil {
		t.Fatal(err)
	}
	record := Record{
		ID:       "record-1",
		Scope:    Scope{TenantID: "tenant-a", ContinuityID: "workspace-a", ContinuityLine: "workspace"},
		SourceID: "decision:database", SourceVersion: 2, Status: "active", Content: "PostgreSQL is current",
	}
	if err := backend.Put(context.Background(), record); err != nil {
		t.Fatal(err)
	}
	results, err := backend.Search(context.Background(), Query{Scope: record.Scope, Text: "current database", Limit: 5})
	if err != nil {
		t.Fatal(err)
	}
	if len(results) != 1 || results[0].Record.ID != record.ID || results[0].Score != 0.92 {
		t.Fatalf("unexpected results: %#v", results)
	}
	putTag, _ := putBody["containerTag"].(string)
	searchTag, _ := searchBody["containerTag"].(string)
	if putTag == "" || putTag != searchTag || putTag == record.Scope.ContinuityID {
		t.Fatalf("expected stable opaque combined scope tag, put=%q search=%q", putTag, searchTag)
	}
}

func TestMemOSBackendUsesCubeIsolationAndPreservesContextMeshID(t *testing.T) {
	var putBody map[string]any
	var searchBody map[string]any
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch {
		case r.Method == http.MethodPost && r.URL.Path == "/product/add":
			decodeTestJSON(t, r, &putBody)
			_, _ = w.Write([]byte(`{"code":200,"message":"Memory added successfully","data":[{"memory":"PostgreSQL is current","memory_id":"provider-1","memory_type":"UserMemory","cube_id":"cube-1"}]}`))
		case r.Method == http.MethodPost && r.URL.Path == "/product/search":
			decodeTestJSON(t, r, &searchBody)
			_, _ = w.Write([]byte(`{"code":200,"message":"Search completed successfully","data":{"text_mem":[{"cube_id":"cube-1","memories":[{"id":"provider-1","memory":"PostgreSQL is current","metadata":{"cm_record_id":"record-1","cm_source_id":"decision:database","cm_source_version":2,"cm_lifecycle_status":"active","cm_continuity_line":"workspace","relativity":0.93}}]}]}}`))
		default:
			http.Error(w, "unexpected request", http.StatusNotFound)
		}
	}))
	defer server.Close()

	backend, err := NewMemOSBackend(MemOSConfig{BaseURL: server.URL, Client: server.Client()})
	if err != nil {
		t.Fatal(err)
	}
	record := Record{
		ID:       "record-1",
		Scope:    Scope{TenantID: "tenant-a", ContinuityID: "workspace-a", ContinuityLine: "workspace"},
		SourceID: "decision:database", SourceVersion: 2, Status: "active", Content: "PostgreSQL is current",
	}
	if err := backend.Put(context.Background(), record); err != nil {
		t.Fatal(err)
	}
	results, err := backend.Search(context.Background(), Query{Scope: record.Scope, Text: "current database", Limit: 5})
	if err != nil {
		t.Fatal(err)
	}
	if len(results) != 1 || results[0].Record.ID != record.ID || results[0].Score != 0.93 {
		t.Fatalf("unexpected results: %#v", results)
	}
	writeCubes, _ := putBody["writable_cube_ids"].([]any)
	readCubes, _ := searchBody["readable_cube_ids"].([]any)
	if len(writeCubes) != 1 || len(readCubes) != 1 || writeCubes[0] != readCubes[0] || writeCubes[0] == record.Scope.ContinuityID {
		t.Fatalf("expected stable opaque cube scope, put=%#v search=%#v", putBody, searchBody)
	}
	if putBody["async_mode"] != "sync" || putBody["mode"] != "fast" {
		t.Fatalf("expected deterministic synchronous fast add: %#v", putBody)
	}
	filter, _ := searchBody["filter"].(map[string]any)
	if filter["cm_lifecycle_status"] != "active" || searchBody["dedup"] != "no" || searchBody["rerank"] != false {
		t.Fatalf("expected current-state filtering before MemOS dedup/rerank: %#v", searchBody)
	}
}

func decodeTestJSON(t *testing.T, r *http.Request, target any) {
	t.Helper()
	defer r.Body.Close()
	if err := json.NewDecoder(r.Body).Decode(target); err != nil {
		t.Fatal(err)
	}
}
