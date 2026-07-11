package runner

import (
	"vermory/internal/artifact"
	"vermory/internal/eval"
)

type BaselineID string

const (
	BaselineNoContext         BaselineID = "no_context"
	BaselineStaleContext      BaselineID = "stale_context"
	BaselinePlainSummary      BaselineID = "plain_summary"
	BaselineContextMeshPacket BaselineID = "contextmesh_packet"
)

type EvaluationOptions struct {
	RunID          string
	ProviderMode   string
	ProviderName   string
	Model          string
	SystemPrompt   string
	Task           eval.Task
	StaleContext   string
	PlainSummary   string
	ContextPacket  string
	MaxTokens      int
	ArtifactPrefix string
	ArtifactStore  artifact.Store
}

type ChatContractRequest struct {
	ThreadID       string
	SystemFrame    string
	History        []string
	ContinuityView string
	CurrentTurn    string
}

type ConversationEvaluationOptions struct {
	RunID          string
	ProviderMode   string
	ProviderName   string
	Model          string
	SystemFrame    string
	ThreadID       string
	ThreadHistory  []string
	ContinuityView string
	CurrentTurn    string
	MaxTokens      int
	ArtifactPrefix string
	ArtifactStore  artifact.Store
	Task           eval.Task
}

type BaselineResult struct {
	Baseline     BaselineID `json:"baseline"`
	ProviderMode string     `json:"provider_mode"`
	ProviderName string     `json:"provider_name"`
	Model        string     `json:"model"`
	Output       string     `json:"output"`
	Score        eval.Score `json:"score"`
	InputURI     string     `json:"input_uri"`
	PacketURI    string     `json:"packet_uri,omitempty"`
	OutputURI    string     `json:"output_uri"`
	RawURI       string     `json:"raw_uri,omitempty"`
	ScoreURI     string     `json:"score_uri"`
}

type ConversationEvaluationResult struct {
	RunID        string     `json:"run_id"`
	TaskID       string     `json:"task_id"`
	ProviderMode string     `json:"provider_mode"`
	ProviderName string     `json:"provider_name"`
	Model        string     `json:"model"`
	Output       string     `json:"output"`
	Score        eval.Score `json:"score"`
	InputURI     string     `json:"input_uri"`
	OutputURI    string     `json:"output_uri"`
	RawURI       string     `json:"raw_uri,omitempty"`
	ScoreURI     string     `json:"score_uri"`
}

type EvaluationReport struct {
	RunID        string           `json:"run_id"`
	TaskID       string           `json:"task_id"`
	ProviderMode string           `json:"provider_mode"`
	ProviderName string           `json:"provider_name"`
	Model        string           `json:"model"`
	Results      []BaselineResult `json:"results"`
	ReportURI    string           `json:"report_uri,omitempty"`
}
