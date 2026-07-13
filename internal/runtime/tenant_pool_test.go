package runtime

import (
	"context"
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	"vermory/internal/authn"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

func TestTenantPoolFiltersOmittedQueriesAndClearsReusedConnections(t *testing.T) {
	admin, databaseURL := openTenantPoolAdmin(t)
	ctx := context.Background()
	seedTenantContinuity(t, admin.pool, "identity-a")
	seedTenantContinuity(t, admin.pool, "identity-b")

	roleName, runtimeURL := createTenantPoolRole(t, admin.pool, databaseURL, "runtime", "")
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

	if err := runtimeStore.pool.QueryRow(ctx, `SELECT count(*) FROM continuity_spaces`).Scan(new(int)); err == nil {
		t.Fatal("missing tenant context did not fail closed")
	}
	for _, tenantID := range []string{"identity-a", "identity-b", "identity-a", "identity-b"} {
		tenantCtx, err := withTenantContext(ctx, tenantID)
		if err != nil {
			t.Fatal(err)
		}
		var tenants string
		if err := runtimeStore.pool.QueryRow(tenantCtx, `
SELECT string_agg(DISTINCT tenant_id, ',' ORDER BY tenant_id)
FROM continuity_spaces`).Scan(&tenants); err != nil {
			t.Fatal(err)
		}
		if tenants != tenantID {
			t.Fatalf("tenant %s saw rows for %q", tenantID, tenants)
		}
	}

	var wait sync.WaitGroup
	errorsByTenant := make(chan error, 40)
	for _, tenantID := range []string{"identity-a", "identity-b"} {
		tenantID := tenantID
		wait.Add(1)
		go func() {
			defer wait.Done()
			for range 20 {
				tenantCtx, err := withTenantContext(ctx, tenantID)
				if err != nil {
					errorsByTenant <- err
					return
				}
				var visible string
				if err := runtimeStore.pool.QueryRow(tenantCtx, `SELECT max(tenant_id) FROM continuity_spaces`).Scan(&visible); err != nil {
					errorsByTenant <- err
					return
				}
				if visible != tenantID {
					errorsByTenant <- fmt.Errorf("tenant %s observed %q", tenantID, visible)
					return
				}
			}
		}()
	}
	wait.Wait()
	close(errorsByTenant)
	for err := range errorsByTenant {
		t.Fatal(err)
	}

	resolution, err := runtimeStore.ResolveWorkspace(ctx, "identity-a", WorkspaceAnchor{RepoRoot: "/runtime/method-scoping"})
	if err != nil {
		t.Fatal(err)
	}
	if resolution.Status != ResolutionNeedsConfirmation {
		t.Fatalf("store method did not execute under tenant context: %#v", resolution)
	}
}

func TestTenantPoolRejectsCrossTenantForeignKeys(t *testing.T) {
	admin, databaseURL := openTenantPoolAdmin(t)
	ctx := context.Background()
	graphA := seedTenantGraph(t, admin.pool, "identity-a", "a")
	graphB := seedTenantGraph(t, admin.pool, "identity-b", "b")

	roleName, runtimeURL := createTenantPoolRole(t, admin.pool, databaseURL, "foreign_key", "")
	if err := authn.GrantRuntimeRole(ctx, admin.pool, roleName); err != nil {
		t.Fatal(err)
	}
	runtimeStore, err := OpenStoreWithOptions(ctx, runtimeURL, StoreOptions{EnforceTenantContext: true})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(runtimeStore.Close)
	tenantB, err := withTenantContext(ctx, "identity-b")
	if err != nil {
		t.Fatal(err)
	}

	attacks := []struct {
		name string
		sql  string
		args []any
	}{
		{
			name: "continuity",
			sql: `INSERT INTO continuity_bindings (continuity_id, tenant_id, repo_root, binding_state)
              VALUES ($1::uuid, 'identity-b', '/attack/a-continuity', 'ambiguous')`,
			args: []any{graphA.continuityID},
		},
		{
			name: "memory",
			sql: `INSERT INTO memory_search_documents (memory_id, tenant_id, continuity_id, content, search_document)
              VALUES ($1::uuid, 'identity-b', $2::uuid, 'attack', to_tsvector('simple', 'attack'))`,
			args: []any{graphA.memoryID, graphB.continuityID},
		},
		{
			name: "delivery",
			sql: `INSERT INTO conversation_turns (
                tenant_id, continuity_id, operation_id, status, user_observation_id,
                delivery_id, request_fingerprint
              ) VALUES (
                'identity-b', $1::uuid, 'attack-delivery', 'in_progress', $2::uuid,
                $3::uuid, repeat('a', 64)
              )`,
			args: []any{graphB.continuityID, graphB.observationID, graphA.deliveryID},
		},
		{
			name: "bridge",
			sql: `INSERT INTO bridge_events (tenant_id, bridge_id, event_type, operation_id)
              VALUES ('identity-b', $1::uuid, 'created', 'attack-bridge')`,
			args: []any{graphA.bridgeID},
		},
	}
	for _, attack := range attacks {
		if _, err := runtimeStore.pool.Exec(tenantB, attack.sql, attack.args...); err == nil {
			t.Fatalf("cross-tenant %s reference was accepted", attack.name)
		}
	}
}

func TestRuntimeRoleValidationRejectsUnsafeIdentities(t *testing.T) {
	admin, databaseURL := openTenantPoolAdmin(t)
	ctx := context.Background()
	if err := admin.ValidateRuntimeRole(ctx); err == nil {
		t.Fatal("admin/table-owner identity passed runtime validation")
	}

	runtimeRole, runtimeURL := createTenantPoolRole(t, admin.pool, databaseURL, "valid", "")
	if err := authn.GrantRuntimeRole(ctx, admin.pool, runtimeRole); err != nil {
		t.Fatal(err)
	}
	runtimeStore, err := OpenStoreWithOptions(ctx, runtimeURL, StoreOptions{EnforceTenantContext: true})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(runtimeStore.Close)
	if err := runtimeStore.ValidateRuntimeRole(ctx); err != nil {
		t.Fatalf("restricted runtime role was rejected: %v", err)
	}

	_, bypassURL := createTenantPoolRole(t, admin.pool, databaseURL, "bypass", "BYPASSRLS")
	bypassStore, err := OpenStore(ctx, bypassURL)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(bypassStore.Close)
	if err := bypassStore.ValidateRuntimeRole(ctx); err == nil {
		t.Fatal("BYPASSRLS identity passed runtime validation")
	}

	leakyRole, leakyURL := createTenantPoolRole(t, admin.pool, databaseURL, "legacy_access", "")
	if _, err := admin.pool.Exec(ctx, "GRANT SELECT ON public.projects TO "+pgx.Identifier{leakyRole}.Sanitize()); err != nil {
		t.Fatal(err)
	}
	leakyStore, err := OpenStore(ctx, leakyURL)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(leakyStore.Close)
	if err := leakyStore.ValidateRuntimeRole(ctx); err == nil {
		t.Fatal("identity with legacy table access passed runtime validation")
	}

	ownerRole, ownerURL := createTenantPoolRole(t, admin.pool, databaseURL, "owner", "")
	var originalOwner string
	if err := admin.pool.QueryRow(ctx, `
SELECT pg_get_userbyid(relowner)
FROM pg_class
WHERE oid = 'public.continuity_spaces'::regclass`).Scan(&originalOwner); err != nil {
		t.Fatal(err)
	}
	if _, err := admin.pool.Exec(ctx, "ALTER TABLE public.continuity_spaces OWNER TO "+pgx.Identifier{ownerRole}.Sanitize()); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		_, _ = admin.pool.Exec(context.Background(), "ALTER TABLE public.continuity_spaces OWNER TO "+pgx.Identifier{originalOwner}.Sanitize())
	})
	ownerStore, err := OpenStore(ctx, ownerURL)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(ownerStore.Close)
	if err := ownerStore.ValidateRuntimeRole(ctx); err == nil {
		t.Fatal("served-table owner identity passed runtime validation")
	}
}

func TestTenantPoolAllTenantStoreMethodsAttachContext(t *testing.T) {
	files, err := filepath.Glob("*_store.go")
	if err != nil {
		t.Fatal(err)
	}
	fileSet := token.NewFileSet()
	for _, path := range files {
		parsed, err := parser.ParseFile(fileSet, path, nil, 0)
		if err != nil {
			t.Fatal(err)
		}
		for _, declaration := range parsed.Decls {
			function, ok := declaration.(*ast.FuncDecl)
			if !ok || function.Recv == nil || function.Body == nil || !ast.IsExported(function.Name.Name) || !hasNamedParameter(function, "tenantID") {
				continue
			}
			attached := false
			ast.Inspect(function.Body, func(node ast.Node) bool {
				call, ok := node.(*ast.CallExpr)
				if !ok {
					return true
				}
				identifier, ok := call.Fun.(*ast.Ident)
				if ok && identifier.Name == "withTenantContext" {
					attached = true
					return false
				}
				return true
			})
			if !attached {
				t.Errorf("%s.%s accepts tenantID without attaching runtime tenant context", path, function.Name.Name)
			}
		}
	}
}

type tenantGraph struct {
	continuityID  string
	observationID string
	memoryID      string
	deliveryID    string
	bridgeID      string
}

func openTenantPoolAdmin(t *testing.T) (*Store, string) {
	t.Helper()
	databaseURL := os.Getenv("VERMORY_TEST_DATABASE_URL")
	if databaseURL == "" {
		t.Skip("VERMORY_TEST_DATABASE_URL is not set")
	}
	store, err := OpenStore(context.Background(), databaseURL)
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
	return store, databaseURL
}

func seedTenantContinuity(t *testing.T, pool *pgxpool.Pool, tenantID string) string {
	t.Helper()
	var continuityID string
	if err := pool.QueryRow(context.Background(), `
INSERT INTO continuity_spaces (tenant_id, continuity_line, state)
VALUES ($1, 'workspace', 'active')
RETURNING id::text`, tenantID).Scan(&continuityID); err != nil {
		t.Fatal(err)
	}
	return continuityID
}

func seedTenantGraph(t *testing.T, pool *pgxpool.Pool, tenantID, suffix string) tenantGraph {
	t.Helper()
	ctx := context.Background()
	graph := tenantGraph{continuityID: seedTenantContinuity(t, pool, tenantID)}
	if err := pool.QueryRow(ctx, `
INSERT INTO observations (tenant_id, continuity_id, operation_id, observation_kind, content)
VALUES ($1, $2::uuid, $3, 'source_update', $4)
RETURNING id::text`, tenantID, graph.continuityID, "seed-observation-"+suffix, "seed "+suffix).Scan(&graph.observationID); err != nil {
		t.Fatal(err)
	}
	if err := pool.QueryRow(ctx, `
INSERT INTO governed_memories (
  tenant_id, continuity_id, origin_observation_id, memory_kind, lifecycle_status, content
)
VALUES ($1, $2::uuid, $3::uuid, 'fact', 'active', $4)
RETURNING id::text`, tenantID, graph.continuityID, graph.observationID, "memory "+suffix).Scan(&graph.memoryID); err != nil {
		t.Fatal(err)
	}
	if err := pool.QueryRow(ctx, `
INSERT INTO memory_deliveries (tenant_id, continuity_id, operation_id, task, context_body)
VALUES ($1, $2::uuid, $3, 'seed task', 'seed context')
RETURNING id::text`, tenantID, graph.continuityID, "seed-delivery-"+suffix).Scan(&graph.deliveryID); err != nil {
		t.Fatal(err)
	}
	if err := pool.QueryRow(ctx, `
INSERT INTO bridge_operations (
  tenant_id, operation_id, action, status, source_continuity_id,
  source_anchor, target_profile, title, request_fingerprint
)
VALUES ($1, $2, 'export', 'active', $3::uuid, $4, 'team_handoff', 'seed', $5)
RETURNING id::text`, tenantID, "seed-bridge-"+suffix, graph.continuityID, "/seed/"+suffix, "fingerprint-"+suffix).Scan(&graph.bridgeID); err != nil {
		t.Fatal(err)
	}
	return graph
}

func createTenantPoolRole(t *testing.T, pool *pgxpool.Pool, databaseURL, suffix, attributes string) (string, string) {
	t.Helper()
	roleName := "vermory_pool_" + suffix + "_" + strings.ReplaceAll(time.Now().UTC().Format("150405.000000000"), ".", "")
	roleSQL := pgx.Identifier{roleName}.Sanitize()
	statement := "CREATE ROLE " + roleSQL + " LOGIN PASSWORD 'vermory-test-only' NOSUPERUSER " + attributes
	if _, err := pool.Exec(context.Background(), statement); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		_, _ = pool.Exec(context.Background(), "DROP OWNED BY "+roleSQL)
		_, _ = pool.Exec(context.Background(), "DROP ROLE IF EXISTS "+roleSQL)
	})
	parsed, err := url.Parse(databaseURL)
	if err != nil {
		t.Fatal(err)
	}
	parsed.User = url.UserPassword(roleName, "vermory-test-only")
	query := parsed.Query()
	query.Set("pool_max_conns", "2")
	parsed.RawQuery = query.Encode()
	return roleName, parsed.String()
}

func hasNamedParameter(function *ast.FuncDecl, name string) bool {
	for _, field := range function.Type.Params.List {
		for _, identifier := range field.Names {
			if identifier.Name == name {
				return true
			}
		}
	}
	return false
}
