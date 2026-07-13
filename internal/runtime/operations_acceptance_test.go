package runtime

import (
	"context"
	"errors"
	"fmt"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"vermory/internal/authn"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type allProjectionRebuilder interface {
	RebuildAllProjections(context.Context) (int64, error)
}

func TestOperationsRecovery(t *testing.T) {
	databaseURL := os.Getenv("VERMORY_TEST_DATABASE_URL")
	if databaseURL == "" {
		t.Skip("VERMORY_TEST_DATABASE_URL is not set")
	}

	t.Run("migration projection token and role recovery", func(t *testing.T) {
		ctx := context.Background()
		admin, err := OpenStore(ctx, databaseURL)
		if err != nil {
			t.Fatal(err)
		}
		t.Cleanup(admin.Close)
		if err := admin.Migrate(ctx); err != nil {
			t.Fatal(err)
		}
		if err := admin.Migrate(ctx); err != nil {
			t.Fatalf("migration replay failed: %v", err)
		}
		if err := admin.ResetForTest(ctx); err != nil {
			t.Fatal(err)
		}

		var schemaVersion int64
		if err := admin.pool.QueryRow(ctx, `SELECT max(version_id) FROM goose_db_version WHERE is_applied`).Scan(&schemaVersion); err != nil {
			t.Fatal(err)
		}
		if schemaVersion != 9 {
			t.Fatalf("expected schema version 9 after replay, got %d", schemaVersion)
		}

		continuityID, activeContent, staleContent, deletedContent := seedOperationsProjection(t, admin.pool)
		activeToken, revokedToken := seedOperationsTokens(t, admin.pool)
		before := operationsAuthorityFingerprint(t, admin.pool)

		rebuilder, ok := any(admin).(allProjectionRebuilder)
		if !ok {
			t.Fatal("runtime store does not expose an explicit all-projection recovery operation")
		}
		rebuilt, err := rebuilder.RebuildAllProjections(ctx)
		if err != nil {
			t.Fatal(err)
		}
		if rebuilt != 1 {
			t.Fatalf("expected one active projection after rebuild, got %d", rebuilt)
		}
		if after := operationsAuthorityFingerprint(t, admin.pool); after != before {
			t.Fatalf("projection rebuild changed authoritative state: before=%s after=%s", before, after)
		}
		assertOperationsSearch(t, admin, continuityID, activeContent, true)
		assertOperationsSearch(t, admin, continuityID, staleContent, false)
		assertOperationsSearch(t, admin, continuityID, deletedContent, false)

		roleName, runtimeURL := createTenantPoolRole(t, admin.pool, databaseURL, "operations_reopen", "")
		if err := authn.GrantRuntimeRole(ctx, admin.pool, roleName); err != nil {
			t.Fatal(err)
		}
		for attempt := 0; attempt < 2; attempt++ {
			runtimeStore, err := OpenStoreWithOptions(ctx, runtimeURL, StoreOptions{EnforceTenantContext: true})
			if err != nil {
				t.Fatal(err)
			}
			if err := runtimeStore.ValidateRuntimeRole(ctx); err != nil {
				runtimeStore.Close()
				t.Fatalf("runtime role validation failed on open %d: %v", attempt+1, err)
			}
			runtimeStore.Close()
		}

		authPool, err := pgxpool.New(ctx, runtimeURL)
		if err != nil {
			t.Fatal(err)
		}
		t.Cleanup(authPool.Close)
		authenticator := authn.NewPostgresAuthenticator(authPool)
		principal, err := authenticator.Authenticate(ctx, activeToken)
		if err != nil || principal.TenantID != "ops-tenant" {
			t.Fatalf("restored active token metadata did not authenticate: principal=%#v err=%v", principal, err)
		}
		if _, err := authenticator.Authenticate(ctx, revokedToken); !errors.Is(err, authn.ErrAuthenticationFailed) {
			t.Fatalf("revoked token metadata authenticated: %v", err)
		}
	})

	t.Run("database outage has no false receipt and pool recovers", func(t *testing.T) {
		ctx := context.Background()
		dedicatedURL, databaseName, maintenance := createOperationsDatabase(t, databaseURL)
		admin, err := OpenStore(ctx, dedicatedURL)
		if err != nil {
			t.Fatal(err)
		}
		t.Cleanup(admin.Close)
		if err := admin.Migrate(ctx); err != nil {
			t.Fatal(err)
		}

		roleName, runtimeURL := createTenantPoolRole(t, admin.pool, dedicatedURL, "operations_outage", "")
		if err := authn.GrantRuntimeRole(ctx, admin.pool, roleName); err != nil {
			t.Fatal(err)
		}
		runtimeStore, err := OpenStoreWithOptions(ctx, runtimeURL, StoreOptions{EnforceTenantContext: true})
		if err != nil {
			t.Fatal(err)
		}
		t.Cleanup(runtimeStore.Close)
		if err := runtimeStore.ValidateRuntimeRole(ctx); err != nil {
			t.Fatal(err)
		}
		service := NewConversationService(runtimeStore, "ops-outage-tenant", nil, "", ConversationServiceConfig{})
		anchor := ConversationAnchor{Channel: "operations-test", ThreadID: "database-outage"}

		setDatabaseConnections(t, maintenance, databaseName, false)
		outageCtx, cancel := context.WithTimeout(ctx, 2*time.Second)
		receipt, err := service.PrepareExternalTurn(outageCtx, ExternalConversationTurnRequest{
			OperationID: "ops-outage-write",
			Anchor:      anchor,
			Message:     "This write must not receive a success receipt.",
		})
		cancel()
		if err == nil {
			t.Fatalf("database outage returned a successful receipt: %#v", receipt)
		}
		if receipt.ID != "" || receipt.Status != "" {
			t.Fatalf("database outage returned a non-zero receipt: %#v", receipt)
		}

		setDatabaseConnections(t, maintenance, databaseName, true)
		var recovered PreparedConversationTurn
		deadline := time.Now().Add(10 * time.Second)
		for {
			recoveryCtx, recoveryCancel := context.WithTimeout(ctx, time.Second)
			recovered, err = service.PrepareExternalTurn(recoveryCtx, ExternalConversationTurnRequest{
				OperationID: "ops-after-recovery",
				Anchor:      anchor,
				Message:     "The new request should persist after recovery.",
			})
			recoveryCancel()
			if err == nil || time.Now().After(deadline) {
				break
			}
			time.Sleep(100 * time.Millisecond)
		}
		if err != nil {
			t.Fatalf("same runtime pool did not recover: %v", err)
		}
		if recovered.Status != ChatTurnInProgress || recovered.ID == "" {
			t.Fatalf("unexpected recovery receipt: %#v", recovered)
		}

		outageRows := operationsTurnCountEventually(t, admin.pool, "ops-outage-write")
		recoveryRows := operationsTurnCountEventually(t, admin.pool, "ops-after-recovery")
		if outageRows != 0 || recoveryRows != 1 {
			t.Fatalf("unexpected persistence after outage recovery: outage=%d recovery=%d", outageRows, recoveryRows)
		}
	})

	t.Run("trimpath release binary migrates outside repository", func(t *testing.T) {
		dedicatedURL, _, _ := createOperationsDatabase(t, databaseURL)
		moduleRoot, err := filepath.Abs(filepath.Join("..", ".."))
		if err != nil {
			t.Fatal(err)
		}
		binaryPath := filepath.Join(t.TempDir(), "vermory")
		build := exec.Command("go", "build", "-trimpath", "-o", binaryPath, "./cmd/vermory")
		build.Dir = moduleRoot
		if output, err := build.CombinedOutput(); err != nil {
			t.Fatalf("build release binary: %v\n%s", err, output)
		}
		migrate := exec.Command(binaryPath, "database", "migrate", "--database-url", dedicatedURL)
		migrate.Dir = t.TempDir()
		if output, err := migrate.CombinedOutput(); err != nil {
			t.Fatalf("migrate outside repository: %v\n%s", err, output)
		}
		pool, err := pgxpool.New(context.Background(), dedicatedURL)
		if err != nil {
			t.Fatal(err)
		}
		t.Cleanup(pool.Close)
		var schemaVersion int64
		if err := pool.QueryRow(context.Background(), `SELECT max(version_id) FROM goose_db_version WHERE is_applied`).Scan(&schemaVersion); err != nil {
			t.Fatal(err)
		}
		if schemaVersion != 9 {
			t.Fatalf("release migration reached schema %d", schemaVersion)
		}
	})
}

func operationsTurnCountEventually(t *testing.T, pool *pgxpool.Pool, operationID string) int {
	t.Helper()
	deadline := time.Now().Add(5 * time.Second)
	for {
		var count int
		queryCtx, cancel := context.WithTimeout(context.Background(), time.Second)
		err := pool.QueryRow(queryCtx, `SELECT count(*) FROM conversation_turns WHERE operation_id = $1`, operationID).Scan(&count)
		cancel()
		if err == nil {
			return count
		}
		if time.Now().After(deadline) {
			t.Fatalf("count operation %q after database recovery: %v", operationID, err)
		}
		time.Sleep(100 * time.Millisecond)
	}
}

func seedOperationsProjection(t *testing.T, pool *pgxpool.Pool) (string, string, string, string) {
	t.Helper()
	ctx := context.Background()
	var continuityID string
	if err := pool.QueryRow(ctx, `
INSERT INTO continuity_spaces (tenant_id, continuity_line, state)
VALUES ('ops-tenant', 'workspace', 'active')
RETURNING id::text`).Scan(&continuityID); err != nil {
		t.Fatal(err)
	}
	contents := []struct {
		status  string
		content string
	}{
		{status: "superseded", content: "OPS-STALE-2048"},
		{status: "active", content: "OPS-ACTIVE-7319"},
		{status: "deleted", content: "OPS-DELETED-9981"},
	}
	var staleID string
	for index, item := range contents {
		var observationID, memoryID string
		if err := pool.QueryRow(ctx, `
INSERT INTO observations (tenant_id, continuity_id, operation_id, observation_kind, content, source_ref)
VALUES ('ops-tenant', $1::uuid, $2, 'source_update', $3, 'fixture:I02')
RETURNING id::text`, continuityID, fmt.Sprintf("ops-observation-%d", index), item.content).Scan(&observationID); err != nil {
			t.Fatal(err)
		}
		if err := pool.QueryRow(ctx, `
INSERT INTO governed_memories (
  tenant_id, continuity_id, origin_observation_id, memory_kind,
  lifecycle_status, content, supersedes_memory_id
)
VALUES ('ops-tenant', $1::uuid, $2::uuid, 'fact', $3, $4, NULLIF($5, '')::uuid)
RETURNING id::text`, continuityID, observationID, item.status, item.content, staleID).Scan(&memoryID); err != nil {
			t.Fatal(err)
		}
		if index == 0 {
			staleID = memoryID
		}
		if _, err := pool.Exec(ctx, `
INSERT INTO memory_search_documents (memory_id, tenant_id, continuity_id, content, search_document)
VALUES ($1::uuid, 'ops-tenant', $2::uuid, $3, to_tsvector('simple', $3))`, memoryID, continuityID, item.content); err != nil {
			t.Fatal(err)
		}
	}
	return continuityID, contents[1].content, contents[0].content, contents[2].content
}

func seedOperationsTokens(t *testing.T, pool *pgxpool.Pool) (string, string) {
	t.Helper()
	ctx := context.Background()
	active, err := authn.IssueToken(ctx, pool, authn.IssueTokenRequest{
		OperationID: "ops-active-token",
		TenantID:    "ops-tenant",
		SubjectID:   "ops-client",
		Role:        authn.RoleClient,
		ExpiresAt:   time.Now().UTC().Add(time.Hour),
	})
	if err != nil {
		t.Fatal(err)
	}
	revoked, err := authn.IssueToken(ctx, pool, authn.IssueTokenRequest{
		OperationID: "ops-revoked-token",
		TenantID:    "ops-tenant",
		SubjectID:   "ops-revoked-client",
		Role:        authn.RoleClient,
		ExpiresAt:   time.Now().UTC().Add(time.Hour),
	})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := authn.RevokeToken(ctx, pool, authn.RevokeTokenRequest{
		OperationID: "ops-revoke-token",
		PublicID:    revoked.Token.PublicID(),
	}); err != nil {
		t.Fatal(err)
	}
	return active.Token.Reveal(), revoked.Token.Reveal()
}

func operationsAuthorityFingerprint(t *testing.T, pool *pgxpool.Pool) string {
	t.Helper()
	var fingerprint string
	if err := pool.QueryRow(context.Background(), `
WITH authoritative_rows AS (
  SELECT 'continuity_spaces' AS table_name, to_jsonb(row_data)::text AS row_data FROM continuity_spaces row_data
  UNION ALL SELECT 'continuity_bindings', to_jsonb(row_data)::text FROM continuity_bindings row_data
  UNION ALL SELECT 'conversation_bindings', to_jsonb(row_data)::text FROM conversation_bindings row_data
  UNION ALL SELECT 'observations', to_jsonb(row_data)::text FROM observations row_data
  UNION ALL SELECT 'governed_memories', to_jsonb(row_data)::text FROM governed_memories row_data
  UNION ALL SELECT 'memory_deliveries', to_jsonb(row_data)::text FROM memory_deliveries row_data
  UNION ALL SELECT 'conversation_turns', to_jsonb(row_data)::text FROM conversation_turns row_data
  UNION ALL SELECT 'bridge_operations', to_jsonb(row_data)::text FROM bridge_operations row_data
  UNION ALL SELECT 'bridge_events', to_jsonb(row_data)::text FROM bridge_events row_data
  UNION ALL SELECT 'bridge_memory_effects', to_jsonb(row_data)::text FROM bridge_memory_effects row_data
  UNION ALL SELECT 'conversation_links', to_jsonb(row_data)::text FROM conversation_links row_data
  UNION ALL SELECT 'api_tokens', to_jsonb(row_data)::text FROM vermory_auth.api_tokens row_data
)
SELECT md5(COALESCE(string_agg(table_name || ':' || row_data, E'\n' ORDER BY table_name, row_data), ''))
FROM authoritative_rows`).Scan(&fingerprint); err != nil {
		t.Fatal(err)
	}
	return fingerprint
}

func assertOperationsSearch(t *testing.T, store *Store, continuityID, query string, want bool) {
	t.Helper()
	matches, err := store.SearchActiveMemory(context.Background(), "ops-tenant", continuityID, query, 10)
	if err != nil {
		t.Fatal(err)
	}
	found := false
	for _, match := range matches {
		if match.Content == query {
			found = true
			break
		}
	}
	if found != want {
		t.Fatalf("search %q got %#v, want present=%t", query, matches, want)
	}
}

func createOperationsDatabase(t *testing.T, baseURL string) (string, string, *pgxpool.Pool) {
	t.Helper()
	parsed, err := url.Parse(baseURL)
	if err != nil {
		t.Fatal(err)
	}
	databaseName := "vermory_ops_test_" + strings.ReplaceAll(time.Now().UTC().Format("150405.000000000"), ".", "")
	maintenanceURL := *parsed
	maintenanceURL.Path = "/postgres"
	maintenance, err := pgxpool.New(context.Background(), maintenanceURL.String())
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(maintenance.Close)
	if _, err := maintenance.Exec(context.Background(), "CREATE DATABASE "+pgx.Identifier{databaseName}.Sanitize()); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		_, _ = maintenance.Exec(context.Background(), "ALTER DATABASE "+pgx.Identifier{databaseName}.Sanitize()+" ALLOW_CONNECTIONS true")
		_, _ = maintenance.Exec(context.Background(), `SELECT pg_terminate_backend(pid) FROM pg_stat_activity WHERE datname = $1`, databaseName)
		_, _ = maintenance.Exec(context.Background(), "DROP DATABASE IF EXISTS "+pgx.Identifier{databaseName}.Sanitize())
	})
	dedicatedURL := *parsed
	dedicatedURL.Path = "/" + databaseName
	return dedicatedURL.String(), databaseName, maintenance
}

func setDatabaseConnections(t *testing.T, maintenance *pgxpool.Pool, databaseName string, allowed bool) {
	t.Helper()
	state := "false"
	if allowed {
		state = "true"
	}
	if _, err := maintenance.Exec(context.Background(), "ALTER DATABASE "+pgx.Identifier{databaseName}.Sanitize()+" ALLOW_CONNECTIONS "+state); err != nil {
		t.Fatal(err)
	}
	if !allowed {
		if _, err := maintenance.Exec(context.Background(), `SELECT pg_terminate_backend(pid) FROM pg_stat_activity WHERE datname = $1`, databaseName); err != nil {
			t.Fatal(err)
		}
	}
}
