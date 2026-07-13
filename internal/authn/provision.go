package authn

import (
	"context"
	"errors"
	"fmt"
	"regexp"
	"strings"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

var roleIdentifierPattern = regexp.MustCompile(`^[A-Za-z_][A-Za-z0-9_$]{0,62}$`)

var servedTables = []string{
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

var forbiddenRuntimeTables = []string{
	"vermory_auth.api_tokens",
	"public.projects",
	"public.sources",
	"public.source_versions",
	"public.claims",
	"public.capsules",
	"public.capsule_claims",
	"public.packets",
	"public.audit_logs",
	"public.wcef_runs",
}

func GrantRuntimeRole(ctx context.Context, pool *pgxpool.Pool, roleName string) error {
	if pool == nil {
		return errors.New("grant runtime role: database pool is required")
	}
	if !roleIdentifierPattern.MatchString(roleName) {
		return invalidRequest("invalid PostgreSQL role identifier")
	}
	var canLogin, superuser, bypassRLS bool
	if err := pool.QueryRow(ctx, `
SELECT rolcanlogin, rolsuper, rolbypassrls
FROM pg_roles
WHERE rolname = $1`, roleName).Scan(&canLogin, &superuser, &bypassRLS); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return invalidRequest("PostgreSQL role does not exist")
		}
		return fmt.Errorf("inspect runtime role: %w", err)
	}
	if !canLogin || superuser || bypassRLS {
		return invalidRequest("PostgreSQL role is not a restricted login role")
	}

	var ownedTables int
	if err := pool.QueryRow(ctx, `
SELECT count(*)
FROM pg_class c
JOIN pg_namespace n ON n.oid = c.relnamespace
JOIN pg_roles r ON r.oid = c.relowner
WHERE r.rolname = $1
  AND n.nspname IN ('public', 'vermory_auth')
  AND c.relkind IN ('r', 'p')`, roleName).Scan(&ownedTables); err != nil {
		return fmt.Errorf("inspect runtime role ownership: %w", err)
	}
	if ownedTables != 0 {
		return invalidRequest("PostgreSQL runtime role owns application tables")
	}

	tx, err := pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("begin runtime role grant: %w", err)
	}
	defer tx.Rollback(ctx)
	roleSQL := pgx.Identifier{roleName}.Sanitize()
	servedSQL := make([]string, 0, len(servedTables))
	for _, table := range servedTables {
		servedSQL = append(servedSQL, pgx.Identifier{"public", table}.Sanitize())
	}
	forbiddenSQL := make([]string, 0, len(forbiddenRuntimeTables))
	for _, table := range forbiddenRuntimeTables {
		parts := strings.SplitN(table, ".", 2)
		forbiddenSQL = append(forbiddenSQL, pgx.Identifier{parts[0], parts[1]}.Sanitize())
	}
	statements := []string{
		"GRANT USAGE ON SCHEMA public, vermory_auth TO " + roleSQL,
		"GRANT SELECT, INSERT, UPDATE, DELETE ON TABLE " + strings.Join(servedSQL, ", ") + " TO " + roleSQL,
		"GRANT USAGE, SELECT ON SEQUENCE public.observations_observation_seq_seq TO " + roleSQL,
		"GRANT EXECUTE ON FUNCTION vermory_auth.authenticate_token(TEXT, BYTEA) TO " + roleSQL,
		"REVOKE ALL PRIVILEGES ON TABLE " + strings.Join(forbiddenSQL, ", ") + " FROM " + roleSQL,
	}
	for _, statement := range statements {
		if _, err := tx.Exec(ctx, statement); err != nil {
			return fmt.Errorf("apply runtime role grant: %w", err)
		}
	}

	for _, table := range forbiddenRuntimeTables {
		var canRead bool
		if err := tx.QueryRow(ctx, `SELECT has_table_privilege($1, $2, 'SELECT')`, roleName, table).Scan(&canRead); err != nil {
			return fmt.Errorf("verify runtime role boundary: %w", err)
		}
		if canRead {
			return invalidRequest("PostgreSQL role inherits forbidden table access")
		}
	}
	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("commit runtime role grant: %w", err)
	}
	return nil
}
