package main

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"os"
	"strings"
	"testing"

	"vermory/internal/runtime"
)

func TestRetrievalRuntimeCommandsAndSharedFlagsAreRegistered(t *testing.T) {
	root := newRootCommand()
	commands := map[string]bool{}
	for _, command := range root.Commands() {
		commands[command.Name()] = true
		if command.Name() == "mcp-stdio" || command.Name() == "web-chat" || command.Name() == "serve" {
			for _, flag := range []string{
				"retrieval-mode", "retrieval-profile", "embedding-base-url",
				"embedding-api-key-env", "embedding-model", "embedding-dimensions",
			} {
				if command.Flags().Lookup(flag) == nil {
					t.Fatalf("%s is missing --%s", command.Name(), flag)
				}
			}
		}
	}
	for _, name := range []string{"retrieval-worker", "retrieval-status", "retrieval-rebuild"} {
		if !commands[name] {
			t.Fatalf("root command is missing %s", name)
		}
	}
	for _, command := range root.Commands() {
		switch command.Name() {
		case "retrieval-worker":
			for _, flag := range []string{"database-url", "tenant-id", "profile-id", "embedding-base-url", "embedding-api-key-env", "embedding-model", "embedding-dimensions", "once", "poll-interval", "batch-size"} {
				if command.Flags().Lookup(flag) == nil {
					t.Fatalf("retrieval-worker is missing --%s", flag)
				}
			}
		case "retrieval-status", "retrieval-rebuild":
			for _, flag := range []string{"database-url", "tenant-id", "profile-id"} {
				if command.Flags().Lookup(flag) == nil {
					t.Fatalf("%s is missing --%s", command.Name(), flag)
				}
			}
		}
	}
}

func TestRetrievalStatusAndRebuildCommandsUseTenantScopedJSON(t *testing.T) {
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
	tenantID := "retrieval-command-json"
	governance := runtime.NewGovernanceService(store, tenantID)
	if _, err := governance.ConfirmWorkspace(context.Background(), "/fixtures/retrieval-command-json"); err != nil {
		t.Fatal(err)
	}
	if _, err := governance.AddSource(context.Background(), "/fixtures/retrieval-command-json", runtime.GovernanceWriteRequest{
		OperationID: "retrieval-command-source",
		MemoryKey:   "release.approval",
		Content:     "Rollback requires two maintainers.",
		SourceRef:   "fixture:retrieval-command-json",
	}); err != nil {
		t.Fatal(err)
	}
	worker, err := runtime.NewProjectionWorker(store, commandTestEmbedder{}, runtime.ProjectionWorkerOptions{
		TenantID: tenantID,
		Profile: runtime.RetrievalProfile{
			ID:         runtime.ProductionRetrievalProfileID,
			BaseURL:    "https://api.siliconflow.cn/v1",
			Model:      "BAAI/bge-m3",
			Dimensions: 1024,
		},
		BatchSize: 8,
	})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := worker.RunOnce(context.Background()); err != nil {
		t.Fatal(err)
	}

	statusCommand := newRetrievalStatusCommand()
	var statusOutput bytes.Buffer
	statusCommand.SetOut(&statusOutput)
	statusCommand.SetArgs([]string{"--database-url", databaseURL, "--tenant-id", tenantID})
	if err := statusCommand.Execute(); err != nil {
		t.Fatal(err)
	}
	var status runtime.ProjectionStatus
	if err := json.Unmarshal(statusOutput.Bytes(), &status); err != nil {
		t.Fatalf("status output is not JSON: %v\n%s", err, statusOutput.String())
	}
	if status.TenantID != tenantID || status.ProfileID != runtime.ProductionRetrievalProfileID || status.VectorCount != 1 || status.Lag != 0 {
		t.Fatalf("unexpected retrieval status: %#v", status)
	}

	rebuildCommand := newRetrievalRebuildCommand()
	var rebuildOutput bytes.Buffer
	rebuildCommand.SetOut(&rebuildOutput)
	rebuildCommand.SetArgs([]string{"--database-url", databaseURL, "--tenant-id", tenantID})
	if err := rebuildCommand.Execute(); err != nil {
		t.Fatal(err)
	}
	var rebuilt runtime.ProjectionStatus
	if err := json.Unmarshal(rebuildOutput.Bytes(), &rebuilt); err != nil {
		t.Fatalf("rebuild output is not JSON: %v\n%s", err, rebuildOutput.String())
	}
	if rebuilt.LastEventID != 0 || rebuilt.VectorCount != 0 || rebuilt.Status != "idle" || rebuilt.Lag == 0 {
		t.Fatalf("unexpected rebuilt status: %#v", rebuilt)
	}
	resolution, err := store.ResolveWorkspace(context.Background(), tenantID, runtime.WorkspaceAnchor{RepoRoot: "/fixtures/retrieval-command-json"})
	if err != nil {
		t.Fatal(err)
	}
	matches, err := store.SearchActiveMemory(context.Background(), tenantID, resolution.ContinuityID, "rollback maintainers", 5)
	if err != nil {
		t.Fatal(err)
	}
	if len(matches) != 1 || !strings.Contains(matches[0].Content, "two maintainers") {
		t.Fatalf("rebuild changed authority or lexical projection: %#v", matches)
	}
}

func TestRetrievalWorkerRejectsOwnerRoleWithoutLeakingConfiguration(t *testing.T) {
	databaseURL := os.Getenv("VERMORY_TEST_DATABASE_URL")
	if databaseURL == "" {
		t.Skip("VERMORY_TEST_DATABASE_URL is not set")
	}
	store, err := runtime.OpenStore(context.Background(), databaseURL)
	if err != nil {
		t.Fatal(err)
	}
	if err := store.Migrate(context.Background()); err != nil {
		store.Close()
		t.Fatal(err)
	}
	store.Close()
	t.Setenv("W09_WORKER_TEST_KEY", "worker-secret-value")
	command := newRetrievalWorkerCommand()
	command.SetOut(&bytes.Buffer{})
	command.SetErr(&bytes.Buffer{})
	command.SetArgs([]string{
		"--database-url", databaseURL,
		"--tenant-id", "retrieval-worker-role",
		"--embedding-api-key-env", "W09_WORKER_TEST_KEY",
		"--once",
	})
	err = command.Execute()
	if !errors.Is(err, runtime.ErrUnsafeRuntimeRole) {
		t.Fatalf("worker accepted an owner/admin role: %v", err)
	}
	if strings.Contains(err.Error(), databaseURL) || strings.Contains(err.Error(), "worker-secret-value") {
		t.Fatalf("worker error leaked configuration: %v", err)
	}
}

type commandTestEmbedder struct{}

func (commandTestEmbedder) Embed(context.Context, string) ([]float32, error) {
	vector := make([]float32, 1024)
	vector[0] = 1
	return vector, nil
}

func TestLexicalRetrievalIgnoresEmbeddingConfigurationAndCredential(t *testing.T) {
	options := retrievalRuntimeOptions{
		Mode:                runtime.RetrievalLexical,
		ProfileID:           "ignored-profile",
		EmbeddingBaseURL:    "not-a-url",
		EmbeddingAPIKeyEnv:  "W09_MISSING_LEXICAL_KEY",
		EmbeddingModel:      "ignored-model",
		EmbeddingDimensions: -1,
	}
	retriever, err := buildRuntimeRetriever(nil, options)
	if err != nil {
		t.Fatalf("lexical retrieval touched embedding configuration: %v", err)
	}
	if retriever != nil {
		t.Fatalf("lexical default unexpectedly built a provider retriever: %#v", retriever)
	}
}

func TestSemanticRetrievalRequiresFrozenProfileAndNamedCredential(t *testing.T) {
	t.Setenv("W09_PRESENT_KEY", "secret-value-that-must-not-leak")
	valid := defaultRetrievalRuntimeOptions()
	valid.Mode = runtime.RetrievalVector
	valid.EmbeddingAPIKeyEnv = "W09_PRESENT_KEY"
	for name, mutate := range map[string]func(*retrievalRuntimeOptions){
		"mode":       func(options *retrievalRuntimeOptions) { options.Mode = "hybrid" },
		"profile":    func(options *retrievalRuntimeOptions) { options.ProfileID = "other" },
		"base URL":   func(options *retrievalRuntimeOptions) { options.EmbeddingBaseURL = "https://example.com/v1" },
		"model":      func(options *retrievalRuntimeOptions) { options.EmbeddingModel = "other" },
		"dimensions": func(options *retrievalRuntimeOptions) { options.EmbeddingDimensions = 768 },
	} {
		t.Run(name, func(t *testing.T) {
			options := valid
			mutate(&options)
			if _, err := options.validateSemantic(); err == nil {
				t.Fatalf("invalid semantic options were accepted: %#v", options)
			}
		})
	}
	if _, err := valid.validateSemantic(); err != nil {
		t.Fatal(err)
	}

	missing := valid
	missing.EmbeddingAPIKeyEnv = "W09_MISSING_KEY"
	_, err := missing.validateSemantic()
	if err == nil || strings.Contains(err.Error(), "secret-value-that-must-not-leak") {
		t.Fatalf("missing credential error was absent or leaked a secret: %v", err)
	}
}
