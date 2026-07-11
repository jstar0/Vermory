package domain

import "time"

type ID string

type ProjectType string

const (
	ProjectTypeStudentCompetition ProjectType = "student_competition"
)

type SourceType string

const (
	SourceTypeMarkdown SourceType = "markdown"
	SourceTypeChat     SourceType = "chat"
	SourceTypeManual   SourceType = "manual"
	SourceTypeWeb      SourceType = "web"
	SourceTypeGit      SourceType = "git"
)

type ClaimType string

const (
	ClaimTypeGoal           ClaimType = "goal"
	ClaimTypeFact           ClaimType = "fact"
	ClaimTypeDecision       ClaimType = "decision"
	ClaimTypeConstraint     ClaimType = "constraint"
	ClaimTypeProgress       ClaimType = "progress"
	ClaimTypeNextStep       ClaimType = "next_step"
	ClaimTypeRisk           ClaimType = "risk"
	ClaimTypeRejectedOption ClaimType = "rejected_option"
	ClaimTypePreference     ClaimType = "preference"
	ClaimTypeHandoffNote    ClaimType = "handoff_note"
)

type ClaimStatus string

const (
	ClaimStatusDraft          ClaimStatus = "draft"
	ClaimStatusConfirmed      ClaimStatus = "confirmed"
	ClaimStatusActive         ClaimStatus = "active"
	ClaimStatusStaleCandidate ClaimStatus = "stale_candidate"
	ClaimStatusConflict       ClaimStatus = "conflict"
	ClaimStatusArchived       ClaimStatus = "archived"
	ClaimStatusDeleted        ClaimStatus = "deleted"
)

type Project struct {
	ID          ID
	Name        string
	Description string
	Type        ProjectType
	Objective   string
	CreatedAt   time.Time
	UpdatedAt   time.Time
}

type Source struct {
	ID        ID
	ProjectID ID
	Type      SourceType
	Title     string
	CreatedAt time.Time
	UpdatedAt time.Time
}

type SourceVersion struct {
	ID          ID
	SourceID    ID
	Version     int
	ArtifactURI string
	ContentHash string
	ByteSize    int64
	CreatedAt   time.Time
}

type Claim struct {
	ID              ID
	ProjectID       ID
	SourceID        ID
	SourceVersionID ID
	Type            ClaimType
	Content         string
	Status          ClaimStatus
	VerifiedByUser  bool
	CreatedAt       time.Time
	UpdatedAt       time.Time
}

type Capsule struct {
	ID        ID
	ProjectID ID
	Title     string
	Purpose   string
	Summary   string
	CreatedAt time.Time
	UpdatedAt time.Time
}

type Packet struct {
	ID              ID
	ProjectID       ID
	CapsuleID       ID
	TargetProfileID TargetProfileID
	Title           string
	Body            string
	RedactionCount  int
	TokenEstimate   int
	CreatedAt       time.Time
}

type AuditLog struct {
	ID        ID
	ProjectID ID
	Action    AuditAction
	TargetID  ID
	CreatedAt time.Time
}

type ContinuitySpace struct {
	ID             ID
	Name           string
	Line           ContinuityLine
	Anchor         string
	AnchorStrength AnchorStrength
}
