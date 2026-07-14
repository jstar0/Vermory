package runtime

import (
	"context"
	"fmt"
	"net/url"
	"sort"
	"strings"
	"time"
)

const ProductionRetrievalProfileID = "siliconflow-bge-m3-1024-v1"

type RetrievalMode string

const (
	RetrievalLexical RetrievalMode = "lexical"
	RetrievalShadow  RetrievalMode = "shadow"
	RetrievalVector  RetrievalMode = "vector"
)

type RetrievalRequest struct {
	OperationID   string
	TenantID      string
	ContinuityIDs []string
	Query         string
	Limit         int
	Mode          RetrievalMode
}

func (r RetrievalRequest) normalized() (RetrievalRequest, error) {
	r.OperationID = strings.TrimSpace(r.OperationID)
	r.TenantID = strings.TrimSpace(r.TenantID)
	r.Query = strings.TrimSpace(r.Query)
	if r.Mode == "" {
		r.Mode = RetrievalLexical
	}
	if r.Mode != RetrievalLexical && r.Mode != RetrievalShadow && r.Mode != RetrievalVector {
		return RetrievalRequest{}, fmt.Errorf("retrieval mode must be lexical, shadow, or vector")
	}
	if r.Mode != RetrievalLexical && r.OperationID == "" {
		return RetrievalRequest{}, fmt.Errorf("retrieval operation ID is required")
	}
	if r.TenantID == "" {
		return RetrievalRequest{}, fmt.Errorf("retrieval tenant ID is required")
	}
	if len(r.ContinuityIDs) == 0 || len(r.ContinuityIDs) > 50 {
		return RetrievalRequest{}, fmt.Errorf("retrieval requires between 1 and 50 continuity IDs")
	}
	continuityIDs := make([]string, 0, len(r.ContinuityIDs))
	seen := make(map[string]struct{}, len(r.ContinuityIDs))
	for _, continuityID := range r.ContinuityIDs {
		continuityID = strings.TrimSpace(continuityID)
		if continuityID == "" {
			return RetrievalRequest{}, fmt.Errorf("retrieval continuity ID is required")
		}
		if _, exists := seen[continuityID]; exists {
			return RetrievalRequest{}, fmt.Errorf("retrieval continuity IDs must be unique")
		}
		seen[continuityID] = struct{}{}
		continuityIDs = append(continuityIDs, continuityID)
	}
	sort.Strings(continuityIDs)
	r.ContinuityIDs = continuityIDs
	if r.Query == "" {
		return RetrievalRequest{}, fmt.Errorf("retrieval query is required")
	}
	if r.Limit < 0 {
		return RetrievalRequest{}, fmt.Errorf("retrieval limit cannot be negative")
	}
	if r.Limit == 0 {
		r.Limit = defaultContextItems
	}
	if r.Limit > maxContextItems {
		r.Limit = maxContextItems
	}
	return r, nil
}

type RetrievalResult struct {
	Memories  []Memory
	Effective RetrievalMode
	Degraded  bool
	AuditID   string
}

type MemoryRetriever interface {
	Retrieve(context.Context, RetrievalRequest) (RetrievalResult, error)
}

type RetrievalProfile struct {
	ID         string
	BaseURL    string
	Model      string
	Dimensions int
}

func (p RetrievalProfile) Validate() error {
	if strings.TrimSpace(p.ID) != ProductionRetrievalProfileID {
		return fmt.Errorf("retrieval profile must be %s", ProductionRetrievalProfileID)
	}
	baseURL := strings.TrimRight(strings.TrimSpace(p.BaseURL), "/")
	parsed, err := url.Parse(baseURL)
	if err != nil || parsed.Scheme != "https" || parsed.Host == "" || parsed.User != nil {
		return fmt.Errorf("embedding base URL is invalid or contains credentials")
	}
	if baseURL != "https://api.siliconflow.cn/v1" {
		return fmt.Errorf("embedding base URL must use direct SiliconFlow v1")
	}
	if strings.TrimSpace(p.Model) != "BAAI/bge-m3" {
		return fmt.Errorf("embedding model must be BAAI/bge-m3")
	}
	if p.Dimensions != 1024 {
		return fmt.Errorf("embedding dimensions must be 1024")
	}
	return nil
}

type Embedder interface {
	Embed(context.Context, string) ([]float32, error)
}

type ProjectionStatus struct {
	TenantID      string     `json:"tenant_id"`
	ProfileID     string     `json:"profile_id"`
	LastEventID   int64      `json:"last_event_id"`
	LatestEventID int64      `json:"latest_event_id"`
	Lag           int64      `json:"lag"`
	Status        string     `json:"status"`
	AttemptCount  int        `json:"attempt_count"`
	LastErrorCode string     `json:"last_error_code,omitempty"`
	LastAttemptAt *time.Time `json:"last_attempt_at,omitempty"`
	VectorCount   int64      `json:"vector_count"`
}

type ProjectionEvent struct {
	EventID          int64
	TenantID         string
	ContinuityID     string
	MemoryID         string
	DesiredState     string
	AuthorityVersion time.Time
}

type ProjectionWorkerOptions struct {
	TenantID     string
	Profile      RetrievalProfile
	BatchSize    int
	PollInterval time.Duration
}

func (o *ProjectionWorkerOptions) normalize() error {
	o.TenantID = strings.TrimSpace(o.TenantID)
	if o.TenantID == "" {
		return fmt.Errorf("projection worker tenant ID is required")
	}
	if err := o.Profile.Validate(); err != nil {
		return err
	}
	if o.BatchSize <= 0 {
		o.BatchSize = 32
	}
	if o.BatchSize > 256 {
		o.BatchSize = 256
	}
	if o.PollInterval <= 0 {
		o.PollInterval = time.Second
	}
	return nil
}

type ProjectionRunResult struct {
	Processed      int    `json:"processed"`
	LastEventID    int64  `json:"last_event_id"`
	LatestEventID  int64  `json:"latest_event_id"`
	Lag            int64  `json:"lag"`
	Status         string `json:"status"`
	FailureCode    string `json:"failure_code,omitempty"`
	AlreadyRunning bool   `json:"already_running"`
}

type projectionRunError struct {
	code string
}

func (e projectionRunError) Error() string {
	return "projection worker: " + e.code
}
