package postgres

import (
	"context"
	"errors"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"

	"vermory/internal/store/postgres/db"
)

func TestSchemaRejectsCrossProjectLinks(t *testing.T) {
	databaseURL := os.Getenv("CONTEXTMESH_TEST_DATABASE_URL")
	if databaseURL == "" {
		t.Skip("set CONTEXTMESH_TEST_DATABASE_URL to run PostgreSQL integration tests")
	}

	ctx := context.Background()
	conn, err := pgx.Connect(ctx, databaseURL)
	if err != nil {
		t.Fatal(err)
	}
	defer conn.Close(ctx)

	schema := "contextmesh_test_" + strings.ReplaceAll(time.Now().UTC().Format("20060102150405.000000000"), ".", "_")
	if _, err := conn.Exec(ctx, `CREATE SCHEMA `+schema); err != nil {
		t.Fatal(err)
	}
	defer conn.Exec(ctx, `DROP SCHEMA IF EXISTS `+schema+` CASCADE`)

	if _, err := conn.Exec(ctx, `SET search_path TO `+schema+`, public`); err != nil {
		t.Fatal(err)
	}
	applyInitialMigration(t, ctx, conn)

	queries := db.New(conn)
	projectA, err := queries.CreateProject(ctx, db.CreateProjectParams{
		Name:        "A",
		Description: "project A",
		ProjectType: "student_competition",
		Objective:   "test project boundary",
	})
	if err != nil {
		t.Fatal(err)
	}
	projectB, err := queries.CreateProject(ctx, db.CreateProjectParams{
		Name:        "B",
		Description: "project B",
		ProjectType: "student_competition",
		Objective:   "test project boundary",
	})
	if err != nil {
		t.Fatal(err)
	}

	sourceA, err := queries.CreateSource(ctx, db.CreateSourceParams{
		ProjectID:  projectA.ID,
		SourceType: "markdown",
		Title:      "source A",
	})
	if err != nil {
		t.Fatal(err)
	}
	sourceB, err := queries.CreateSource(ctx, db.CreateSourceParams{
		ProjectID:  projectB.ID,
		SourceType: "markdown",
		Title:      "source B",
	})
	if err != nil {
		t.Fatal(err)
	}

	sourceVersionA, err := queries.CreateSourceVersion(ctx, db.CreateSourceVersionParams{
		SourceID:    sourceA.ID,
		Version:     1,
		ArtifactUri: "file:///source-a.md",
		ContentHash: "hash-a",
		ByteSize:    8,
	})
	if err != nil {
		t.Fatal(err)
	}
	sourceVersionB, err := queries.CreateSourceVersion(ctx, db.CreateSourceVersionParams{
		SourceID:    sourceB.ID,
		Version:     1,
		ArtifactUri: "file:///source-b.md",
		ContentHash: "hash-b",
		ByteSize:    8,
	})
	if err != nil {
		t.Fatal(err)
	}

	_, err = queries.CreateClaim(ctx, db.CreateClaimParams{
		ProjectID:       projectA.ID,
		SourceID:        sourceB.ID,
		SourceVersionID: sourceVersionB.ID,
		ClaimType:       "fact",
		Content:         "cross project source must fail",
		Status:          "confirmed",
		VerifiedByUser:  true,
	})
	assertForeignKeyFailure(t, err)
	_, err = queries.CreateClaim(ctx, db.CreateClaimParams{
		ProjectID:       projectA.ID,
		SourceID:        sourceA.ID,
		SourceVersionID: sourceVersionB.ID,
		ClaimType:       "fact",
		Content:         "cross source version must fail",
		Status:          "confirmed",
		VerifiedByUser:  true,
	})
	assertForeignKeyFailure(t, err)

	claimA, err := queries.CreateClaim(ctx, db.CreateClaimParams{
		ProjectID:       projectA.ID,
		SourceID:        sourceA.ID,
		SourceVersionID: sourceVersionA.ID,
		ClaimType:       "fact",
		Content:         "valid project A claim",
		Status:          "confirmed",
		VerifiedByUser:  true,
	})
	if err != nil {
		t.Fatal(err)
	}
	claimB, err := queries.CreateClaim(ctx, db.CreateClaimParams{
		ProjectID:       projectB.ID,
		SourceID:        sourceB.ID,
		SourceVersionID: sourceVersionB.ID,
		ClaimType:       "fact",
		Content:         "valid project B claim",
		Status:          "confirmed",
		VerifiedByUser:  true,
	})
	if err != nil {
		t.Fatal(err)
	}

	capsuleA, err := queries.CreateCapsule(ctx, db.CreateCapsuleParams{
		ProjectID: projectA.ID,
		Title:     "capsule A",
		Purpose:   "boundary test",
		Summary:   "valid capsule",
	})
	if err != nil {
		t.Fatal(err)
	}
	if err := queries.AddClaimToCapsule(ctx, db.AddClaimToCapsuleParams{
		ProjectID:  projectA.ID,
		CapsuleID:  capsuleA.ID,
		ClaimID:    claimA.ID,
		OrderIndex: 0,
	}); err != nil {
		t.Fatal(err)
	}
	err = queries.AddClaimToCapsule(ctx, db.AddClaimToCapsuleParams{
		ProjectID:  projectA.ID,
		CapsuleID:  capsuleA.ID,
		ClaimID:    claimB.ID,
		OrderIndex: 1,
	})
	assertForeignKeyFailure(t, err)

	if _, err := queries.CreatePacket(ctx, db.CreatePacketParams{
		ProjectID:       projectA.ID,
		CapsuleID:       capsuleA.ID,
		TargetProfileID: "team_handoff",
		Title:           "valid packet",
		Body:            "body",
		RedactionCount:  0,
		TokenEstimate:   1,
	}); err != nil {
		t.Fatal(err)
	}
	_, err = queries.CreatePacket(ctx, db.CreatePacketParams{
		ProjectID:       projectB.ID,
		CapsuleID:       capsuleA.ID,
		TargetProfileID: "team_handoff",
		Title:           "cross project packet must fail",
		Body:            "body",
		RedactionCount:  0,
		TokenEstimate:   1,
	})
	assertForeignKeyFailure(t, err)
}

func applyInitialMigration(t *testing.T, ctx context.Context, conn *pgx.Conn) {
	t.Helper()

	content, err := os.ReadFile("migrations/00001_initial.sql")
	if err != nil {
		t.Fatal(err)
	}
	upSQL, _, ok := strings.Cut(string(content), "-- +goose Down")
	if !ok {
		t.Fatal("migration is missing -- +goose Down marker")
	}
	if _, err := conn.Exec(ctx, upSQL); err != nil {
		t.Fatal(err)
	}
}

func assertForeignKeyFailure(t *testing.T, err error) {
	t.Helper()

	if err == nil {
		t.Fatal("expected foreign key violation")
	}
	var pgErr *pgconn.PgError
	if !errors.As(err, &pgErr) {
		t.Fatalf("expected PostgreSQL error, got %T: %v", err, err)
	}
	if pgErr.Code != "23503" {
		t.Fatalf("expected foreign key violation 23503, got %s: %v", pgErr.Code, err)
	}
}
