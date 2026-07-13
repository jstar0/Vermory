package runtime

import (
	"context"
	"strings"
	"testing"
)

func TestIdentityRLSMigrationCreatesRestrictedAuthSchema(t *testing.T) {
	store := openTestStore(t)
	ctx := context.Background()

	var tableExists bool
	if err := store.pool.QueryRow(ctx, `
SELECT to_regclass('vermory_auth.api_tokens') IS NOT NULL`).Scan(&tableExists); err != nil {
		t.Fatal(err)
	}
	if !tableExists {
		t.Fatal("vermory_auth.api_tokens is missing")
	}

	rows, err := store.pool.Query(ctx, `
SELECT column_name, data_type
FROM information_schema.columns
WHERE table_schema = 'vermory_auth' AND table_name = 'api_tokens'
ORDER BY ordinal_position`)
	if err != nil {
		t.Fatal(err)
	}
	defer rows.Close()
	columns := map[string]string{}
	for rows.Next() {
		var name, dataType string
		if err := rows.Scan(&name, &dataType); err != nil {
			t.Fatal(err)
		}
		columns[name] = dataType
	}
	for _, required := range []string{
		"id", "issue_operation_id", "request_fingerprint", "public_id", "token_digest",
		"tenant_id", "subject_id", "role", "status", "expires_at", "created_at",
		"revoked_at", "revoke_operation_id",
	} {
		if _, ok := columns[required]; !ok {
			t.Fatalf("auth token column %q is missing: %#v", required, columns)
		}
	}
	for name := range columns {
		if strings.Contains(name, "secret") || strings.Contains(name, "raw_token") {
			t.Fatalf("auth schema stores forbidden secret column %q", name)
		}
	}
	if columns["token_digest"] != "bytea" {
		t.Fatalf("token_digest must be bytea, got %q", columns["token_digest"])
	}

	var securityDefiner bool
	var functionConfig []string
	var publicExecuteGrants int
	if err := store.pool.QueryRow(ctx, `
SELECT p.prosecdef,
       COALESCE(p.proconfig, ARRAY[]::text[]),
       (SELECT count(*)
        FROM aclexplode(COALESCE(p.proacl, acldefault('f', p.proowner))) acl
        WHERE acl.grantee = 0 AND acl.privilege_type = 'EXECUTE')
FROM pg_proc p
JOIN pg_namespace n ON n.oid = p.pronamespace
WHERE n.nspname = 'vermory_auth' AND p.proname = 'authenticate_token'`,
	).Scan(&securityDefiner, &functionConfig, &publicExecuteGrants); err != nil {
		t.Fatal(err)
	}
	if !securityDefiner {
		t.Fatal("authenticate_token must be SECURITY DEFINER")
	}
	if publicExecuteGrants != 0 {
		t.Fatal("PUBLIC must not execute authenticate_token")
	}
	if !containsString(functionConfig, "search_path=pg_catalog, vermory_auth") {
		t.Fatalf("authenticate_token has unsafe search path: %#v", functionConfig)
	}

	var publicTablePrivileges int
	if err := store.pool.QueryRow(ctx, `
SELECT count(*)
FROM pg_class c
JOIN pg_namespace n ON n.oid = c.relnamespace
CROSS JOIN LATERAL aclexplode(COALESCE(c.relacl, acldefault('r', c.relowner))) acl
WHERE n.nspname = 'vermory_auth' AND c.relname = 'api_tokens' AND acl.grantee = 0`,
	).Scan(&publicTablePrivileges); err != nil {
		t.Fatal(err)
	}
	if publicTablePrivileges != 0 {
		t.Fatalf("PUBLIC has auth table privileges: %d", publicTablePrivileges)
	}

	var publicSchemaPrivileges int
	if err := store.pool.QueryRow(ctx, `
SELECT count(*)
FROM pg_namespace n
CROSS JOIN LATERAL aclexplode(COALESCE(n.nspacl, acldefault('n', n.nspowner))) acl
WHERE n.nspname = 'vermory_auth' AND acl.grantee = 0`,
	).Scan(&publicSchemaPrivileges); err != nil {
		t.Fatal(err)
	}
	if publicSchemaPrivileges != 0 {
		t.Fatalf("PUBLIC has auth schema privileges: %d", publicSchemaPrivileges)
	}

	var checkDefinitions []string
	if err := store.pool.QueryRow(ctx, `
SELECT COALESCE(array_agg(pg_get_constraintdef(c.oid) ORDER BY c.conname), ARRAY[]::text[])
FROM pg_constraint c
WHERE c.conrelid = 'vermory_auth.api_tokens'::regclass AND c.contype = 'c'`,
	).Scan(&checkDefinitions); err != nil {
		t.Fatal(err)
	}
	checks := strings.Join(checkDefinitions, " ")
	for _, expected := range []string{"client", "operator", "owner", "active", "revoked"} {
		if !strings.Contains(checks, expected) {
			t.Fatalf("auth token checks do not constrain %q: %s", expected, checks)
		}
	}
}

func TestIdentityRLSMigrationEnablesEveryServedTenantTable(t *testing.T) {
	store := openTestStore(t)
	ctx := context.Background()
	tables := []string{
		"continuity_spaces",
		"continuity_bindings",
		"conversation_bindings",
		"observations",
		"governed_memories",
		"memory_deliveries",
		"memory_search_documents",
		"conversation_turns",
		"bridge_operations",
		"bridge_events",
		"bridge_memory_effects",
		"conversation_links",
	}
	for _, table := range tables {
		var enabled bool
		var policyCount int
		if err := store.pool.QueryRow(ctx, `
SELECT c.relrowsecurity,
       (SELECT count(*) FROM pg_policy policy WHERE policy.polrelid = c.oid)
FROM pg_class c
JOIN pg_namespace n ON n.oid = c.relnamespace
WHERE n.nspname = 'public' AND c.relname = $1`, table).Scan(&enabled, &policyCount); err != nil {
			t.Fatalf("inspect RLS for %s: %v", table, err)
		}
		if !enabled || policyCount == 0 {
			t.Fatalf("table %s is not protected by RLS: enabled=%v policies=%d", table, enabled, policyCount)
		}
		var usingExpression, checkExpression string
		if err := store.pool.QueryRow(ctx, `
SELECT pg_get_expr(polqual, polrelid), pg_get_expr(polwithcheck, polrelid)
FROM pg_policy
WHERE polrelid = ('public.' || $1)::regclass
ORDER BY polname
LIMIT 1`, table).Scan(&usingExpression, &checkExpression); err != nil {
			t.Fatalf("inspect RLS policy for %s: %v", table, err)
		}
		for _, expression := range []string{usingExpression, checkExpression} {
			if !strings.Contains(expression, "vermory.tenant_id") || !strings.Contains(expression, "tenant_id") {
				t.Fatalf("table %s has unsafe tenant policy expression %q", table, expression)
			}
		}
	}

	var publicLegacyPrivileges int
	if err := store.pool.QueryRow(ctx, `
SELECT count(*)
FROM pg_class c
JOIN pg_namespace n ON n.oid = c.relnamespace
CROSS JOIN LATERAL aclexplode(COALESCE(c.relacl, acldefault('r', c.relowner))) acl
WHERE n.nspname = 'public'
  AND c.relname = ANY($1::text[])
  AND acl.grantee = 0`, []string{
		"projects", "sources", "claims", "capsules", "packets", "wcef_runs",
	}).Scan(&publicLegacyPrivileges); err != nil {
		t.Fatal(err)
	}
	if publicLegacyPrivileges != 0 {
		t.Fatalf("PUBLIC has legacy table privileges: %d", publicLegacyPrivileges)
	}
}

func TestIdentityRLSMigrationAddsTenantAwareForeignKeys(t *testing.T) {
	store := openTestStore(t)
	ctx := context.Background()
	constraints := []string{
		"continuity_bindings_tenant_continuity_fk",
		"conversation_bindings_tenant_continuity_fk",
		"observations_tenant_continuity_fk",
		"governed_memories_tenant_continuity_fk",
		"governed_memories_tenant_origin_observation_fk",
		"governed_memories_tenant_supersedes_memory_fk",
		"memory_deliveries_tenant_continuity_fk",
		"memory_search_documents_tenant_memory_fk",
		"memory_search_documents_tenant_continuity_fk",
		"conversation_turns_tenant_continuity_fk",
		"conversation_turns_tenant_user_observation_fk",
		"conversation_turns_tenant_delivery_fk",
		"conversation_turns_tenant_assistant_observation_fk",
		"bridge_operations_tenant_source_continuity_fk",
		"bridge_operations_tenant_target_continuity_fk",
		"bridge_events_tenant_bridge_fk",
		"bridge_memory_effects_tenant_bridge_fk",
		"bridge_memory_effects_tenant_source_memory_fk",
		"bridge_memory_effects_tenant_target_memory_fk",
		"conversation_links_tenant_bridge_fk",
		"conversation_links_tenant_primary_continuity_fk",
		"conversation_links_tenant_linked_continuity_fk",
	}
	for _, name := range constraints {
		var validated bool
		var definition string
		if err := store.pool.QueryRow(ctx, `
SELECT convalidated, pg_get_constraintdef(oid)
FROM pg_constraint
WHERE conname = $1`, name).Scan(&validated, &definition); err != nil {
			t.Fatalf("inspect constraint %s: %v", name, err)
		}
		if !validated || !strings.Contains(definition, "tenant_id") {
			t.Fatalf("constraint %s is not a validated tenant-aware FK: validated=%v definition=%q", name, validated, definition)
		}
	}
}

func containsString(values []string, expected string) bool {
	for _, value := range values {
		if value == expected {
			return true
		}
	}
	return false
}
