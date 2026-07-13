package identitycli

import (
	"bytes"
	"context"
	"encoding/json"
	"os"
	"strings"
	"testing"
	"time"

	"vermory/internal/authn"
	"vermory/internal/runtime"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/spf13/cobra"
)

func TestIdentityCommandsRequireExplicitAdminDatabaseURL(t *testing.T) {
	root := newTestRoot()
	root.SetArgs([]string{"identity", "token", "inspect", "--public-id", "public123"})
	if err := root.Execute(); err == nil || !strings.Contains(err.Error(), "--database-url is required") {
		t.Fatalf("expected explicit admin URL error, got %v", err)
	}
}

func TestIdentityTokenIssuePrintsSecretOnceAndInspectDoesNot(t *testing.T) {
	databaseURL := resetIdentityCLIStore(t)
	requestFlags := []string{
		"identity", "token", "issue",
		"--database-url", databaseURL,
		"--operation-id", "cli-token-issue-1",
		"--tenant-id", "identity-a",
		"--subject-id", "alice-client",
		"--role", "client",
		"--expires-at", time.Now().UTC().Add(time.Hour).Format(time.RFC3339),
	}
	issueOutput := executeCommand(t, requestFlags...)
	var issued struct {
		Token      string                `json:"token"`
		Inspection authn.TokenInspection `json:"inspection"`
		Replayed   bool                  `json:"replayed"`
	}
	if err := json.Unmarshal(issueOutput.Bytes(), &issued); err != nil {
		t.Fatal(err)
	}
	if issued.Token == "" || issued.Inspection.PublicID == "" || issued.Replayed {
		t.Fatalf("unexpected issue output: %#v", issued)
	}

	replayFlags := append([]string{}, requestFlags...)
	replayOutput := executeCommand(t, replayFlags...)
	if strings.Contains(replayOutput.String(), issued.Token) {
		t.Fatalf("replayed issue exposed secret: %s", replayOutput.String())
	}

	inspectOutput := executeCommand(t, "identity", "token", "inspect", "--database-url", databaseURL, "--public-id", issued.Inspection.PublicID)
	if strings.Contains(inspectOutput.String(), issued.Token) || strings.Contains(inspectOutput.String(), "token_digest") {
		t.Fatalf("inspect output exposed secret material: %s", inspectOutput.String())
	}

	revokeOutput := executeCommand(t, "identity", "token", "revoke", "--database-url", databaseURL, "--operation-id", "cli-token-revoke-1", "--public-id", issued.Inspection.PublicID)
	if strings.Contains(revokeOutput.String(), issued.Token) {
		t.Fatalf("revoke output exposed secret: %s", revokeOutput.String())
	}
}

func TestDatabaseCommandsAreRegisteredAndMigrateIsExplicit(t *testing.T) {
	root := newTestRoot()
	for _, path := range [][]string{{"identity", "token", "issue"}, {"identity", "token", "inspect"}, {"identity", "token", "revoke"}, {"database", "migrate"}, {"database", "grant-runtime"}} {
		if findCommand(root, path...) == nil {
			t.Fatalf("missing command %s", strings.Join(path, " "))
		}
	}
	databaseURL := os.Getenv("VERMORY_TEST_DATABASE_URL")
	if databaseURL == "" {
		t.Skip("VERMORY_TEST_DATABASE_URL is not set")
	}
	output := executeCommand(t, "database", "migrate", "--database-url", databaseURL)
	if !strings.Contains(output.String(), "migrated") {
		t.Fatalf("unexpected migrate output: %s", output.String())
	}
}

func TestDatabaseGrantRuntimeAppliesRestrictedRole(t *testing.T) {
	databaseURL := resetIdentityCLIStore(t)
	pool, err := pgxpool.New(context.Background(), databaseURL)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(pool.Close)
	roleName := "vermory_cli_runtime_" + strings.ReplaceAll(time.Now().UTC().Format("150405.000000000"), ".", "")
	roleSQL := pgx.Identifier{roleName}.Sanitize()
	if _, err := pool.Exec(context.Background(), "CREATE ROLE "+roleSQL+" LOGIN NOSUPERUSER NOBYPASSRLS"); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		_, _ = pool.Exec(context.Background(), "DROP OWNED BY "+roleSQL)
		_, _ = pool.Exec(context.Background(), "DROP ROLE IF EXISTS "+roleSQL)
	})

	output := executeCommand(t, "database", "grant-runtime", "--database-url", databaseURL, "--role", roleName)
	if !strings.Contains(output.String(), `"status":"granted"`) || !strings.Contains(output.String(), roleName) {
		t.Fatalf("unexpected grant-runtime output: %s", output.String())
	}
	var canExecute bool
	if err := pool.QueryRow(context.Background(), `
SELECT has_function_privilege($1, 'vermory_auth.authenticate_token(text,bytea)', 'EXECUTE')`, roleName).Scan(&canExecute); err != nil {
		t.Fatal(err)
	}
	if !canExecute {
		t.Fatal("grant-runtime did not grant authenticate_token execution")
	}
}

func resetIdentityCLIStore(t *testing.T) string {
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
	return databaseURL
}

func executeCommand(t *testing.T, args ...string) *bytes.Buffer {
	t.Helper()
	root := newTestRoot()
	var output bytes.Buffer
	root.SetOut(&output)
	root.SetErr(&output)
	root.SetArgs(args)
	if err := root.Execute(); err != nil {
		t.Fatal(err)
	}
	return &output
}

func newTestRoot() *cobra.Command {
	root := &cobra.Command{Use: "vermory", SilenceErrors: true, SilenceUsage: true}
	root.AddCommand(NewIdentityCommand(), NewDatabaseCommand())
	return root
}

func findCommand(root *cobra.Command, path ...string) *cobra.Command {
	current := root
	for _, name := range path {
		var next *cobra.Command
		for _, command := range current.Commands() {
			if command.Name() == name {
				next = command
				break
			}
		}
		if next == nil {
			return nil
		}
		current = next
	}
	return current
}
