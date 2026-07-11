package memorybackend

import (
	"context"
	"time"
)

type Scope struct {
	TenantID       string `json:"tenant_id"`
	ContinuityID   string `json:"continuity_id"`
	ContinuityLine string `json:"continuity_line"`
}

type Record struct {
	ID            string            `json:"id"`
	Scope         Scope             `json:"scope"`
	SourceID      string            `json:"source_id"`
	SourceVersion int               `json:"source_version"`
	Status        string            `json:"status"`
	Content       string            `json:"content"`
	Metadata      map[string]string `json:"metadata,omitempty"`
}

type Query struct {
	Scope          Scope  `json:"scope"`
	Text           string `json:"text"`
	Limit          int    `json:"limit"`
	IncludeHistory bool   `json:"include_history"`
}

type Result struct {
	Record Record  `json:"record"`
	Score  float64 `json:"score"`
}

type Stats struct {
	RecordCount int            `json:"record_count"`
	DiskBytes   int64          `json:"disk_bytes"`
	IdleRSS     int64          `json:"idle_rss_bytes"`
	MeasuredAt  time.Time      `json:"measured_at"`
	Extra       map[string]any `json:"extra,omitempty"`
}

// Backend is the minimum contract required for a ContextMesh memory engine.
// PostgreSQL remains authoritative; implementations are disposable indexes.
type Backend interface {
	Name() string
	Health(ctx context.Context) error
	Put(ctx context.Context, record Record) error
	Search(ctx context.Context, query Query) ([]Result, error)
	Update(ctx context.Context, record Record) error
	Delete(ctx context.Context, scope Scope, recordID string) error
	ResetScope(ctx context.Context, scope Scope) error
	RebuildScope(ctx context.Context, scope Scope, records []Record) error
	Stats(ctx context.Context) (Stats, error)
}
