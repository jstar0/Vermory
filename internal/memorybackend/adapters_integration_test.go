package memorybackend

import (
	"context"
	"os"
	"testing"
	"time"
)

func TestMem0LifecycleIntegration(t *testing.T) {
	baseURL := os.Getenv("CONTEXTMESH_TEST_MEM0_URL")
	if baseURL == "" {
		t.Skip("CONTEXTMESH_TEST_MEM0_URL is not set")
	}
	backend, err := NewMem0Backend(Mem0Config{BaseURL: baseURL, APIKey: os.Getenv("CONTEXTMESH_TEST_MEM0_API_KEY")})
	if err != nil {
		t.Fatal(err)
	}
	runLifecycleIntegration(t, backend)
}

func TestSupermemoryLifecycleIntegration(t *testing.T) {
	baseURL := os.Getenv("CONTEXTMESH_TEST_SUPERMEMORY_URL")
	if baseURL == "" {
		t.Skip("CONTEXTMESH_TEST_SUPERMEMORY_URL is not set")
	}
	backend, err := NewSupermemoryBackend(SupermemoryConfig{BaseURL: baseURL, APIKey: os.Getenv("CONTEXTMESH_TEST_SUPERMEMORY_API_KEY")})
	if err != nil {
		t.Fatal(err)
	}
	runLifecycleIntegration(t, backend)
}

func TestNativeLifecycleIntegration(t *testing.T) {
	databaseURL := os.Getenv("CONTEXTMESH_TEST_NATIVE_DATABASE_URL")
	if databaseURL == "" {
		t.Skip("CONTEXTMESH_TEST_NATIVE_DATABASE_URL is not set")
	}
	backend, err := NewNativeBackend(context.Background(), NativeConfig{
		DatabaseURL:      databaseURL,
		EmbeddingBaseURL: os.Getenv("CONTEXTMESH_TEST_EMBEDDING_BASE_URL"),
		EmbeddingAPIKey:  os.Getenv("CONTEXTMESH_TEST_EMBEDDING_API_KEY"),
		EmbeddingModel:   "bge-m3",
		Dimensions:       1024,
	})
	if err != nil {
		t.Fatal(err)
	}
	defer backend.Close()
	var hasVectorIndex bool
	if err := backend.pool.QueryRow(context.Background(), `
		SELECT EXISTS (
			SELECT 1 FROM pg_indexes
			WHERE tablename = 'contextmesh_memory_index'
			  AND indexdef ILIKE '%USING hnsw%'
		)`).Scan(&hasVectorIndex); err != nil {
		t.Fatal(err)
	}
	if !hasVectorIndex {
		t.Fatal("native backend requires an HNSW vector index")
	}
	runLifecycleIntegration(t, backend)
}

func TestMemOSLifecycleIntegration(t *testing.T) {
	baseURL := os.Getenv("CONTEXTMESH_TEST_MEMOS_URL")
	if baseURL == "" {
		t.Skip("CONTEXTMESH_TEST_MEMOS_URL is not set")
	}
	backend, err := NewMemOSBackend(MemOSConfig{BaseURL: baseURL, APIKey: os.Getenv("CONTEXTMESH_TEST_MEMOS_API_KEY")})
	if err != nil {
		t.Fatal(err)
	}
	runLifecycleIntegration(t, backend)
}

func runLifecycleIntegration(t *testing.T, backend Backend) {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()
	report := RunLifecycleSuite(ctx, backend)
	for _, gate := range report.Gates {
		t.Logf("gate=%s pass=%t details=%s", gate.Name, gate.Pass, gate.Details)
	}
	if !report.Pass {
		t.Fatalf("%s lifecycle suite failed", backend.Name())
	}
}
