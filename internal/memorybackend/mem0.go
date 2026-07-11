package memorybackend

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
	"time"
)

type Mem0Config struct {
	BaseURL string
	APIKey  string
	Client  *http.Client
}

type Mem0Backend struct {
	remote remoteClient
}

type mem0Item struct {
	ID       string         `json:"id"`
	Memory   string         `json:"memory"`
	Score    float64        `json:"score"`
	Metadata map[string]any `json:"metadata"`
}

type mem0Results struct {
	Results []mem0Item `json:"results"`
}

func NewMem0Backend(config Mem0Config) (*Mem0Backend, error) {
	headers := map[string]string{}
	if config.APIKey != "" {
		headers["X-API-Key"] = config.APIKey
	}
	remote, err := newRemoteClient(config.BaseURL, config.Client, headers)
	if err != nil {
		return nil, err
	}
	return &Mem0Backend{remote: remote}, nil
}

func (b *Mem0Backend) Name() string { return "mem0" }

func (b *Mem0Backend) Health(ctx context.Context) error {
	return b.remote.doJSON(ctx, http.MethodGet, "/configure", nil, &map[string]any{})
}

func (b *Mem0Backend) Put(ctx context.Context, record Record) error {
	body := map[string]any{
		"messages": []map[string]string{{"role": "user", "content": record.Content}},
		"user_id":  record.Scope.TenantID,
		"agent_id": record.Scope.ContinuityID,
		"metadata": recordMetadata(record),
		"infer":    false,
	}
	return b.remote.doJSON(ctx, http.MethodPost, "/memories", body, &mem0Results{})
}

func (b *Mem0Backend) Search(ctx context.Context, query Query) ([]Result, error) {
	items, err := b.searchItems(ctx, query)
	if err != nil {
		return nil, err
	}
	results := make([]Result, 0, len(items))
	for _, item := range items {
		record := recordFromProvider(query.Scope, item.Memory, item.Metadata)
		if record.ID == "" || !activeResult(query, record) {
			continue
		}
		results = append(results, Result{Record: record, Score: item.Score})
	}
	return truncateResults(results, query.Limit), nil
}

func (b *Mem0Backend) searchItems(ctx context.Context, query Query) ([]mem0Item, error) {
	if query.Text == "" {
		return b.listScope(ctx, query.Scope)
	}
	limit := query.Limit * 4
	if limit < 20 {
		limit = 20
	}
	if limit > 100 {
		limit = 100
	}
	body := map[string]any{
		"query": query.Text,
		"filters": map[string]string{
			"user_id":  query.Scope.TenantID,
			"agent_id": query.Scope.ContinuityID,
		},
		"top_k": limit,
	}
	var response mem0Results
	if err := b.remote.doJSON(ctx, http.MethodPost, "/search", body, &response); err != nil {
		return nil, err
	}
	return response.Results, nil
}

func (b *Mem0Backend) listScope(ctx context.Context, scope Scope) ([]mem0Item, error) {
	values := url.Values{}
	values.Set("user_id", scope.TenantID)
	values.Set("agent_id", scope.ContinuityID)
	values.Set("top_k", "1000")
	var response mem0Results
	if err := b.remote.doJSON(ctx, http.MethodGet, "/memories?"+values.Encode(), nil, &response); err != nil {
		return nil, err
	}
	return response.Results, nil
}

func (b *Mem0Backend) Update(ctx context.Context, record Record) error {
	item, err := b.findRecord(ctx, record.Scope, record.ID)
	if err != nil {
		return err
	}
	body := map[string]any{"text": record.Content, "metadata": recordMetadata(record)}
	return b.remote.doJSON(ctx, http.MethodPut, "/memories/"+url.PathEscape(item.ID), body, &map[string]any{})
}

func (b *Mem0Backend) Delete(ctx context.Context, scope Scope, recordID string) error {
	item, err := b.findRecord(ctx, scope, recordID)
	if err != nil {
		return err
	}
	return b.remote.doJSON(ctx, http.MethodDelete, "/memories/"+url.PathEscape(item.ID), nil, &map[string]any{})
}

func (b *Mem0Backend) ResetScope(ctx context.Context, scope Scope) error {
	values := url.Values{}
	values.Set("user_id", scope.TenantID)
	values.Set("agent_id", scope.ContinuityID)
	return b.remote.doJSON(ctx, http.MethodDelete, "/memories?"+values.Encode(), nil, &map[string]any{})
}

func (b *Mem0Backend) RebuildScope(ctx context.Context, scope Scope, records []Record) error {
	if err := b.ResetScope(ctx, scope); err != nil {
		return err
	}
	for _, record := range records {
		if record.Status != "active" {
			continue
		}
		if err := b.Put(ctx, record); err != nil {
			return err
		}
	}
	return nil
}

func (b *Mem0Backend) Stats(ctx context.Context) (Stats, error) {
	stats := Stats{MeasuredAt: time.Now().UTC(), Extra: map[string]any{"provider": "mem0"}}
	var response mem0Results
	if err := b.remote.doJSON(ctx, http.MethodGet, "/memories?top_k=1000", nil, &response); err == nil {
		stats.RecordCount = len(response.Results)
		stats.Extra["record_count_available"] = true
	} else {
		stats.Extra["record_count_available"] = false
	}
	return stats, nil
}

func (b *Mem0Backend) findRecord(ctx context.Context, scope Scope, recordID string) (mem0Item, error) {
	items, err := b.listScope(ctx, scope)
	if err != nil {
		return mem0Item{}, err
	}
	for _, item := range items {
		if stringValue(item.Metadata["record_id"]) == recordID {
			return item, nil
		}
	}
	return mem0Item{}, fmt.Errorf("mem0 record %q not found in scope %s/%s", recordID, scope.TenantID, scope.ContinuityID)
}
