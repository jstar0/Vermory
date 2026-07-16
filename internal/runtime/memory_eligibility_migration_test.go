package runtime

import (
	"context"
	"database/sql"
	"os"
	"strings"
	"testing"

	storepostgres "vermory/internal/store/postgres"

	"github.com/pressly/goose/v3"
)

func TestMemoryEligibilitySchema(t *testing.T) {
	store := openTestStore(t)
	ctx := context.Background()
	version, err := store.SchemaVersion(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if version != 18 {
		t.Fatalf("schema version=%d want 18", version)
	}

	for _, column := range []struct {
		table    string
		name     string
		nullable string
	}{
		{table: "governed_memories", name: "valid_from", nullable: "YES"},
		{table: "governed_memories", name: "valid_until", nullable: "YES"},
		{table: "memory_deliveries", name: "eligibility_as_of", nullable: "NO"},
		{table: "memory_retrieval_runs", name: "eligibility_as_of", nullable: "NO"},
	} {
		var dataType, nullable string
		if err := store.pool.QueryRow(ctx, `
SELECT data_type, is_nullable
FROM information_schema.columns
WHERE table_schema = 'public' AND table_name = $1 AND column_name = $2`,
			column.table, column.name).Scan(&dataType, &nullable); err != nil {
			t.Fatalf("read %s.%s: %v", column.table, column.name, err)
		}
		if dataType != "timestamp with time zone" || nullable != column.nullable {
			t.Fatalf("%s.%s type/nullability=%s/%s", column.table, column.name, dataType, nullable)
		}
	}

	var memoryConstraints string
	if err := store.pool.QueryRow(ctx, `
SELECT string_agg(pg_get_constraintdef(oid), ' ' ORDER BY conname)
FROM pg_constraint
WHERE conrelid = 'public.governed_memories'::regclass`).Scan(&memoryConstraints); err != nil {
		t.Fatal(err)
	}
	for _, fragment := range []string{"archived", "valid_until", "valid_from"} {
		if !strings.Contains(memoryConstraints, fragment) {
			t.Fatalf("governed memory constraints lack %q: %s", fragment, memoryConstraints)
		}
	}

	var functionVolatility string
	if err := store.pool.QueryRow(ctx, `
SELECT provolatile::text
FROM pg_proc
WHERE oid = 'public.memory_is_eligible(text,text,timestamp with time zone,timestamp with time zone,timestamp with time zone)'::regprocedure`).Scan(&functionVolatility); err != nil {
		t.Fatal(err)
	}
	if functionVolatility != "i" {
		t.Fatalf("memory_is_eligible volatility=%q want immutable", functionVolatility)
	}

	var rlsEnabled, rlsForced bool
	if err := store.pool.QueryRow(ctx, `
SELECT relrowsecurity, relforcerowsecurity
FROM pg_class
WHERE oid = 'public.memory_eligibility_operations'::regclass`).Scan(&rlsEnabled, &rlsForced); err != nil {
		t.Fatal(err)
	}
	if !rlsEnabled || rlsForced {
		t.Fatalf("eligibility operation RLS enabled/forced=%t/%t want true/false", rlsEnabled, rlsForced)
	}
	var policyCount int
	if err := store.pool.QueryRow(ctx, `
SELECT count(*)
FROM pg_policies
WHERE schemaname = 'public' AND tablename = 'memory_eligibility_operations'
  AND qual LIKE '%vermory.tenant_id%'
  AND with_check LIKE '%vermory.tenant_id%'`).Scan(&policyCount); err != nil {
		t.Fatal(err)
	}
	if policyCount != 1 {
		t.Fatalf("eligibility operation tenant policy count=%d want 1", policyCount)
	}
	var publicPrivileges bool
	if err := store.pool.QueryRow(ctx, `
SELECT has_table_privilege('public', 'public.memory_eligibility_operations', 'SELECT')
    OR has_table_privilege('public', 'public.memory_eligibility_operations', 'INSERT')
    OR has_table_privilege('public', 'public.memory_eligibility_operations', 'UPDATE')
    OR has_table_privilege('public', 'public.memory_eligibility_operations', 'DELETE')`).Scan(&publicPrivileges); err != nil {
		t.Fatal(err)
	}
	if publicPrivileges {
		t.Fatal("memory_eligibility_operations retained PUBLIC privileges")
	}

	assertMemoryEligibilityOperationConstraints(t, store)
}

func assertMemoryEligibilityOperationConstraints(t *testing.T, store *Store) {
	t.Helper()
	var constraints string
	if err := store.pool.QueryRow(context.Background(), `
SELECT string_agg(pg_get_constraintdef(oid), ' ' ORDER BY conname)
FROM pg_constraint
WHERE conrelid = 'public.memory_eligibility_operations'::regclass`).Scan(&constraints); err != nil {
		t.Fatal(err)
	}
	for _, fragment := range []string{
		"UNIQUE (tenant_id, operation_id)",
		"action = ANY (ARRAY['set_validity'::text, 'archive'::text])",
		"length(request_fingerprint) = 64",
		"FOREIGN KEY (tenant_id, continuity_id, memory_id)",
	} {
		if !strings.Contains(constraints, fragment) {
			t.Fatalf("eligibility operation constraints lack %q: %s", fragment, constraints)
		}
	}
}

func TestMemoryEligibilityMigrationUpDown(t *testing.T) {
	store := openTestStore(t)
	ctx := context.Background()
	db := openMemoryEligibilityMigrationDB(t)
	if err := goose.DownToContext(ctx, db, "migrations", 17); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if err := goose.UpToContext(context.Background(), db, "migrations", 18); err != nil {
			t.Errorf("restore schema 18: %v", err)
		}
	})
	var tableExists bool
	if err := store.pool.QueryRow(ctx, `SELECT to_regclass('public.memory_eligibility_operations') IS NOT NULL`).Scan(&tableExists); err != nil {
		t.Fatal(err)
	}
	if tableExists {
		t.Fatal("schema 18 operation table survived downgrade")
	}
	var functionExists bool
	if err := store.pool.QueryRow(ctx, `
SELECT to_regprocedure('public.memory_is_eligible(text,text,timestamp with time zone,timestamp with time zone,timestamp with time zone)') IS NOT NULL`).Scan(&functionExists); err != nil {
		t.Fatal(err)
	}
	if functionExists {
		t.Fatal("schema 18 eligibility function survived downgrade")
	}
	if err := goose.UpToContext(ctx, db, "migrations", 18); err != nil {
		t.Fatal(err)
	}
	assertMemoryEligibilityOperationConstraints(t, store)
}

func openMemoryEligibilityMigrationDB(t *testing.T) *sql.DB {
	t.Helper()
	databaseURL := os.Getenv("VERMORY_TEST_DATABASE_URL")
	if databaseURL == "" {
		t.Skip("VERMORY_TEST_DATABASE_URL is not set")
	}
	db, err := sql.Open("pgx", databaseURL)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = db.Close() })
	if err := goose.SetDialect("postgres"); err != nil {
		t.Fatal(err)
	}
	goose.SetBaseFS(storepostgres.Migrations)
	t.Cleanup(func() { goose.SetBaseFS(nil) })
	return db
}
