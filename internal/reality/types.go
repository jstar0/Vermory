package reality

type EvidenceLevel string

const (
	EvidencePublic        EvidenceLevel = "public"
	EvidenceWithheldLocal EvidenceLevel = "withheld_local"
)

type ContinuityLine string

const (
	LineWorkspace      ContinuityLine = "workspace"
	LineConversation   ContinuityLine = "conversation"
	LineGlobalDefaults ContinuityLine = "global_defaults"
	LineBridge         ContinuityLine = "bridge"
	LineSecurity       ContinuityLine = "security"
)

type SourceRef struct {
	ID            string `json:"id"`
	Kind          string `json:"kind"`
	FixturePath   string `json:"fixture_path"`
	OriginalRef   string `json:"original_ref,omitempty"`
	OriginalRev   string `json:"original_revision,omitempty"`
	SHA256        string `json:"sha256"`
	Authorized    bool   `json:"authorized"`
	Anonymization string `json:"anonymization"`
}

type Anchor struct {
	Kind      string `json:"kind"`
	Value     string `json:"value"`
	Ambiguous bool   `json:"ambiguous"`
}

type Expectations struct {
	CurrentFacts    []string `json:"current_facts"`
	ForbiddenFacts  []string `json:"forbidden_facts"`
	AllowedUnknowns []string `json:"allowed_unknowns,omitempty"`
	ExpectedAction  string   `json:"expected_action"`
}

type DownstreamTask struct {
	Prompt              string   `json:"prompt"`
	ArtifactChecks      []string `json:"artifact_checks,omitempty"`
	DeterministicChecks []string `json:"deterministic_checks"`
}

type Manifest struct {
	Version         int              `json:"version"`
	ID              string           `json:"id"`
	Title           string           `json:"title"`
	EvidenceLevel   EvidenceLevel    `json:"evidence_level"`
	ContinuityLines []ContinuityLine `json:"continuity_lines"`
	Pressures       []string         `json:"pressures"`
	Sources         []SourceRef      `json:"sources"`
	Anchors         []Anchor         `json:"anchors"`
	Expectations    Expectations     `json:"expectations"`
	Task            DownstreamTask   `json:"task"`
}

type Event struct {
	ID       string         `json:"id"`
	Sequence int            `json:"sequence"`
	Actor    string         `json:"actor"`
	Channel  string         `json:"channel"`
	SourceID string         `json:"source_id"`
	Content  string         `json:"content"`
	Metadata map[string]any `json:"metadata,omitempty"`
}

type Case struct {
	Directory string
	Manifest  Manifest
	Events    []Event
}

type Violation struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}
