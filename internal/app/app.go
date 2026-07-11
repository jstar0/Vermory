package app

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/jackc/pgx/v5/pgtype"
	_ "github.com/jackc/pgx/v5/stdlib"
	"github.com/pressly/goose/v3"

	"vermory/internal/artifact"
	"vermory/internal/domain"
	"vermory/internal/eval"
	"vermory/internal/governance"
	"vermory/internal/packet"
	"vermory/internal/store/postgres"
	"vermory/internal/store/postgres/db"
)

const selfCaseID = "001-contextmesh-bluebridge-preparation"

type SelfCaseOptions struct {
	DatabaseURL  string
	ArtifactRoot string
}

type fixtureClaim struct {
	Type    domain.ClaimType `json:"type"`
	Content string           `json:"content"`
}

func RunSelfCase(ctx context.Context, opts SelfCaseOptions) error {
	if strings.TrimSpace(opts.DatabaseURL) == "" {
		return errors.New("database-url is required")
	}
	if strings.TrimSpace(opts.ArtifactRoot) == "" {
		opts.ArtifactRoot = "./artifacts"
	}

	root, err := projectRoot()
	if err != nil {
		return err
	}
	if err := migrate(ctx, opts.DatabaseURL, filepath.Join(root, "internal", "store", "postgres", "migrations")); err != nil {
		return err
	}

	store, err := postgres.Open(ctx, opts.DatabaseURL)
	if err != nil {
		return err
	}
	defer store.Close()

	caseDir := filepath.Join(root, "casebook", "cases", selfCaseID)
	sourceBytes, err := os.ReadFile(filepath.Join(caseDir, "source.md"))
	if err != nil {
		return err
	}
	fixtureClaims, err := readFixtureClaims(filepath.Join(caseDir, "claims.json"))
	if err != nil {
		return err
	}
	tasks, err := readTasks(filepath.Join(caseDir, "tasks.json"))
	if err != nil {
		return err
	}
	if len(tasks) == 0 {
		return errors.New("self-case has no WCEF tasks")
	}

	artifactStore := artifact.NewLocalStore(opts.ArtifactRoot)
	sourceArtifact, err := artifactStore.Put(ctx, "sources/contextmesh/source.md", sourceBytes)
	if err != nil {
		return err
	}

	project, err := store.Queries.CreateProject(ctx, db.CreateProjectParams{
		Name:        "ContextMesh",
		Description: "多平台 AI 工作流连续性上下文治理平台",
		ProjectType: string(domain.ProjectTypeStudentCompetition),
		Objective:   "Build source-bound context capsules and platform packets for AI workflows.",
	})
	if err != nil {
		return err
	}

	source, err := store.Queries.CreateSource(ctx, db.CreateSourceParams{
		ProjectID:  project.ID,
		SourceType: string(domain.SourceTypeMarkdown),
		Title:      "ContextMesh Bluebridge Preparation",
	})
	if err != nil {
		return err
	}

	sourceVersion, err := store.Queries.CreateSourceVersion(ctx, db.CreateSourceVersionParams{
		SourceID:    source.ID,
		Version:     1,
		ArtifactUri: sourceArtifact.URI,
		ContentHash: sourceArtifact.SHA256,
		ByteSize:    sourceArtifact.ByteSize,
	})
	if err != nil {
		return err
	}
	if err := audit(ctx, store.Queries, project.ID, "source_import", sourceVersion.ID); err != nil {
		return err
	}

	claims, dbClaims, err := createConfirmedClaims(ctx, store.Queries, project.ID, source.ID, sourceVersion.ID, fixtureClaims)
	if err != nil {
		return err
	}

	capsule := governance.BuildCapsule("ContextMesh current state", "self-case", claims)
	if _, err := artifactStore.Put(ctx, "capsules/self-case.md", []byte(capsule.Summary)); err != nil {
		return err
	}

	dbCapsule, err := store.Queries.CreateCapsule(ctx, db.CreateCapsuleParams{
		ProjectID: project.ID,
		Title:     capsule.Title,
		Purpose:   capsule.Purpose,
		Summary:   capsule.Summary,
	})
	if err != nil {
		return err
	}
	if err := audit(ctx, store.Queries, project.ID, "capsule_build", dbCapsule.ID); err != nil {
		return err
	}
	for i, claim := range dbClaims {
		if err := store.Queries.AddClaimToCapsule(ctx, db.AddClaimToCapsuleParams{
			ProjectID:  project.ID,
			CapsuleID:  dbCapsule.ID,
			ClaimID:    claim.ID,
			OrderIndex: int32(i),
		}); err != nil {
			return err
		}
	}

	packetClaims := preparePacketClaims(claims)
	teamPacket := packet.Build(packet.ProfileTeamHandoff, "ContextMesh", "团队交接", packetClaims)
	generalPacket := packet.Build(packet.ProfileGeneralChineseChat, "ContextMesh", "中文通用 AI 平台", packetClaims)
	if err := exportPacket(ctx, artifactStore, store.Queries, project.ID, dbCapsule.ID, "packets/team-handoff.md", teamPacket); err != nil {
		return err
	}
	if err := exportPacket(ctx, artifactStore, store.Queries, project.ID, dbCapsule.ID, "packets/general-chat.md", generalPacket); err != nil {
		return err
	}

	score := eval.ScoreOutput(tasks[0], generalPacket.Body)
	report := eval.MarkdownReport(tasks[0], score)
	scoreBytes, err := json.MarshalIndent(score, "", "  ")
	if err != nil {
		return err
	}

	reportArtifact, err := artifactStore.Put(ctx, "wcef-runs/self-case/report.md", []byte(report))
	if err != nil {
		return err
	}
	scoresArtifact, err := artifactStore.Put(ctx, "wcef-runs/self-case/scores.json", scoreBytes)
	if err != nil {
		return err
	}

	wcefRun, err := store.Queries.CreateWCEFRun(ctx, db.CreateWCEFRunParams{
		ProjectID:         project.ID,
		CaseID:            selfCaseID,
		ReportArtifactUri: reportArtifact.URI,
		ScoresArtifactUri: scoresArtifact.URI,
	})
	if err != nil {
		return err
	}
	return audit(ctx, store.Queries, project.ID, "wcef_run", wcefRun.ID)
}

func migrate(ctx context.Context, databaseURL string, migrationsDir string) error {
	sqlDB, err := sql.Open("pgx", databaseURL)
	if err != nil {
		return err
	}
	defer sqlDB.Close()

	if err := goose.SetDialect("postgres"); err != nil {
		return err
	}
	return goose.UpContext(ctx, sqlDB, migrationsDir)
}

func readFixtureClaims(path string) ([]fixtureClaim, error) {
	content, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var claims []fixtureClaim
	if err := json.Unmarshal(content, &claims); err != nil {
		return nil, err
	}
	return claims, nil
}

func readTasks(path string) ([]eval.Task, error) {
	content, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var tasks []eval.Task
	if err := json.Unmarshal(content, &tasks); err != nil {
		return nil, err
	}
	return tasks, nil
}

func createConfirmedClaims(ctx context.Context, queries *db.Queries, projectID, sourceID, sourceVersionID pgtype.UUID, fixtureClaims []fixtureClaim) ([]domain.Claim, []db.Claim, error) {
	claims := make([]domain.Claim, 0, len(fixtureClaims))
	dbClaims := make([]db.Claim, 0, len(fixtureClaims))
	for _, item := range fixtureClaims {
		created, err := queries.CreateClaim(ctx, db.CreateClaimParams{
			ProjectID:       projectID,
			SourceID:        sourceID,
			SourceVersionID: sourceVersionID,
			ClaimType:       string(item.Type),
			Content:         item.Content,
			Status:          string(domain.ClaimStatusConfirmed),
			VerifiedByUser:  true,
		})
		if err != nil {
			return nil, nil, err
		}
		if err := audit(ctx, queries, projectID, "claim_confirm", created.ID); err != nil {
			return nil, nil, err
		}
		claims = append(claims, domain.Claim{
			ID:             domain.ID(created.ID.String()),
			Type:           item.Type,
			Content:        item.Content,
			Status:         domain.ClaimStatusConfirmed,
			VerifiedByUser: true,
		})
		dbClaims = append(dbClaims, created)
	}
	return claims, dbClaims, nil
}

func exportPacket(ctx context.Context, artifactStore artifact.Store, queries *db.Queries, projectID, capsuleID pgtype.UUID, key string, item packet.Packet) error {
	if _, err := artifactStore.Put(ctx, key, []byte(item.Body)); err != nil {
		return err
	}
	created, err := queries.CreatePacket(ctx, db.CreatePacketParams{
		ProjectID:       projectID,
		CapsuleID:       capsuleID,
		TargetProfileID: string(item.ProfileID),
		Title:           item.Title,
		Body:            item.Body,
		RedactionCount:  int32(item.RedactionCount),
		TokenEstimate:   int32(item.TokenEstimate),
	})
	if err != nil {
		return err
	}
	return audit(ctx, queries, projectID, "packet_export", created.ID)
}

func preparePacketClaims(claims []domain.Claim) []domain.Claim {
	prepared := make([]domain.Claim, 0, len(claims))
	for _, claim := range claims {
		claim.Content = adaptLegacyExclusionClause(claim.Content)
		prepared = append(prepared, claim)
	}
	return prepared
}

func adaptLegacyExclusionClause(content string) string {
	const sourceClause = "The product core is a self-designed Context Governance Engine, not MemOS, jstarctl, or a wrapper around a third-party memory framework."
	const packetClause = "The product core is a self-designed Context Governance Engine rather than a wrapper around a third-party memory framework."
	return strings.ReplaceAll(content, sourceClause, packetClause)
}

func audit(ctx context.Context, queries *db.Queries, projectID pgtype.UUID, action string, targetID pgtype.UUID) error {
	_, err := queries.CreateAuditLog(ctx, db.CreateAuditLogParams{
		ProjectID: projectID,
		Action:    action,
		TargetID:  targetID,
	})
	return err
}

func projectRoot() (string, error) {
	dir, err := os.Getwd()
	if err != nil {
		return "", err
	}
	for {
		if _, err := os.Stat(filepath.Join(dir, "go.mod")); err == nil {
			return dir, nil
		}
		next := filepath.Dir(dir)
		if next == dir {
			return "", fmt.Errorf("could not locate project root from %s", dir)
		}
		dir = next
	}
}
