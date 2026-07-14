package retrievalablation

import "time"

const (
	ConditionLexical = "lexical_runtime"
	ConditionVector  = "vector_pg"
	ConditionHybrid  = "hybrid_rrf"
	RRFK             = 60
)

type RankedResult struct {
	MemoryID    string  `json:"memory_id"`
	RecordID    string  `json:"record_id"`
	Content     string  `json:"content,omitempty"`
	Score       float64 `json:"score"`
	LexicalRank int     `json:"lexical_rank,omitempty"`
	VectorRank  int     `json:"vector_rank,omitempty"`
	Exact       bool    `json:"exact"`
	Eligible    bool    `json:"eligible"`
}

type QueryMetrics struct {
	HitAt1          float64 `json:"hit_at_1"`
	RecallAtK       float64 `json:"recall_at_k"`
	MRR             float64 `json:"mrr"`
	NDCGAtK         float64 `json:"ndcg_at_k"`
	ForbiddenCount  int     `json:"forbidden_count"`
	IneligibleCount int     `json:"ineligible_count"`
}

type Aggregate struct {
	QueryCount      int           `json:"query_count"`
	HitAt1          float64       `json:"hit_at_1"`
	RecallAtK       float64       `json:"recall_at_k"`
	MRR             float64       `json:"mrr"`
	NDCGAtK         float64       `json:"ndcg_at_k"`
	ForbiddenCount  int           `json:"forbidden_count"`
	IneligibleCount int           `json:"ineligible_count"`
	SearchP50       time.Duration `json:"search_p50"`
	SearchP95       time.Duration `json:"search_p95"`
}

type QueryReport struct {
	QueryID           string         `json:"query_id"`
	Cohorts           []string       `json:"cohorts"`
	Duration          time.Duration  `json:"duration"`
	Results           []RankedResult `json:"results"`
	RejectedResults   []RankedResult `json:"rejected_results,omitempty"`
	Metrics           QueryMetrics   `json:"metrics"`
	DegradedToLexical bool           `json:"degraded_to_lexical"`
	Error             string         `json:"error,omitempty"`
}

type ConditionReport struct {
	Name    string               `json:"name"`
	Queries []QueryReport        `json:"queries"`
	Metrics Aggregate            `json:"metrics"`
	Cohorts map[string]Aggregate `json:"cohorts"`
}

type HardGateReport struct {
	Pass            bool `json:"pass"`
	ForbiddenCount  int  `json:"forbidden_count"`
	IneligibleCount int  `json:"ineligible_count"`
}

type RunFailure struct {
	Condition string `json:"condition"`
	QueryID   string `json:"query_id"`
	Error     string `json:"error"`
}

type Report struct {
	RunID                       string            `json:"run_id"`
	CorpusSHA256                string            `json:"corpus_sha256"`
	Conditions                  []ConditionReport `json:"conditions"`
	HardGates                   HardGateReport    `json:"hard_gates"`
	ProjectionRebuildEquivalent bool              `json:"projection_rebuild_equivalent"`
	Failures                    []RunFailure      `json:"failures,omitempty"`
}
