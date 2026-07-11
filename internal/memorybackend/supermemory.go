package memorybackend

import (
	"context"
	"fmt"
	"net/http"
	"time"
)

type SupermemoryConfig struct {
	BaseURL string
	APIKey  string
	Client  *http.Client
}

type SupermemoryBackend struct {
	remote remoteClient
}

type supermemoryItem struct {
	ID          string         `json:"id"`
	Memory      string         `json:"memory"`
	Similarity  float64        `json:"similarity"`
	Metadata    map[string]any `json:"metadata"`
	IsLatest    bool           `json:"isLatest"`
	IsForgotten bool           `json:"isForgotten"`
}

type supermemorySearch struct {
	Results []supermemoryItem `json:"results"`
}

type supermemoryList struct {
	MemoryEntries []supermemoryItem `json:"memoryEntries"`
	Pagination    struct {
		TotalPages int `json:"totalPages"`
	} `json:"pagination"`
}

func NewSupermemoryBackend(config SupermemoryConfig) (*SupermemoryBackend, error) {
	headers := map[string]string{}
	if config.APIKey != "" {
		headers["Authorization"] = "Bearer " + config.APIKey
	}
	remote, err := newRemoteClient(config.BaseURL, config.Client, headers)
	if err != nil {
		return nil, err
	}
	return &SupermemoryBackend{remote: remote}, nil
}

func (b *SupermemoryBackend) Name() string { return "supermemory" }

func (b *SupermemoryBackend) Health(ctx context.Context) error {
	return b.remote.doJSON(ctx, http.MethodGet, "/v4/openapi", nil, &map[string]any{})
}

func (b *SupermemoryBackend) Put(ctx context.Context, record Record) error {
	body := map[string]any{
		"containerTag": opaqueScopeTag(record.Scope),
		"memories": []map[string]any{{
			"content":  record.Content,
			"metadata": recordMetadata(record),
		}},
	}
	return b.remote.doJSON(ctx, http.MethodPost, "/v4/memories", body, &map[string]any{})
}

func (b *SupermemoryBackend) Search(ctx context.Context, query Query) ([]Result, error) {
	items, err := b.searchItems(ctx, query)
	if err != nil {
		return nil, err
	}
	results := make([]Result, 0, len(items))
	for _, item := range items {
		record := recordFromProvider(query.Scope, item.Memory, item.Metadata)
		if record.ID == "" || item.IsForgotten || !activeResult(query, record) {
			continue
		}
		results = append(results, Result{Record: record, Score: item.Similarity})
	}
	return truncateResults(results, query.Limit), nil
}

func (b *SupermemoryBackend) searchItems(ctx context.Context, query Query) ([]supermemoryItem, error) {
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
		"q": query.Text, "containerTag": opaqueScopeTag(query.Scope), "limit": limit,
		"threshold": 0, "searchMode": "memories", "rerank": false, "rewriteQuery": false,
	}
	var response supermemorySearch
	if err := b.remote.doJSON(ctx, http.MethodPost, "/v4/search", body, &response); err != nil {
		return nil, err
	}
	return response.Results, nil
}

func (b *SupermemoryBackend) listScope(ctx context.Context, scope Scope) ([]supermemoryItem, error) {
	var all []supermemoryItem
	for page := 1; ; page++ {
		body := map[string]any{
			"containerTags": []string{opaqueScopeTag(scope)},
			"page":          page, "limit": 100, "sort": "createdAt", "order": "asc",
		}
		var response supermemoryList
		if err := b.remote.doJSON(ctx, http.MethodPost, "/v4/memories/list", body, &response); err != nil {
			return nil, err
		}
		all = append(all, response.MemoryEntries...)
		if response.Pagination.TotalPages <= page {
			return all, nil
		}
	}
}

func (b *SupermemoryBackend) Update(ctx context.Context, record Record) error {
	item, err := b.findRecord(ctx, record.Scope, record.ID)
	if err != nil {
		return err
	}
	body := map[string]any{
		"id": item.ID, "containerTag": opaqueScopeTag(record.Scope),
		"newContent": record.Content, "metadata": recordMetadata(record),
	}
	return b.remote.doJSON(ctx, http.MethodPatch, "/v4/memories", body, &map[string]any{})
}

func (b *SupermemoryBackend) Delete(ctx context.Context, scope Scope, recordID string) error {
	item, err := b.findRecord(ctx, scope, recordID)
	if err != nil {
		return err
	}
	body := map[string]any{"id": item.ID, "containerTag": opaqueScopeTag(scope), "reason": "ContextMesh delete"}
	return b.remote.doJSON(ctx, http.MethodDelete, "/v4/memories", body, &map[string]any{})
}

func (b *SupermemoryBackend) ResetScope(ctx context.Context, scope Scope) error {
	items, err := b.listScope(ctx, scope)
	if err != nil {
		return err
	}
	for _, item := range items {
		body := map[string]any{"id": item.ID, "containerTag": opaqueScopeTag(scope), "reason": "ContextMesh scope reset"}
		if err := b.remote.doJSON(ctx, http.MethodDelete, "/v4/memories", body, &map[string]any{}); err != nil {
			return err
		}
	}
	return nil
}

func (b *SupermemoryBackend) RebuildScope(ctx context.Context, scope Scope, records []Record) error {
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

func (b *SupermemoryBackend) Stats(context.Context) (Stats, error) {
	return Stats{
		MeasuredAt: time.Now().UTC(),
		Extra:      map[string]any{"provider": "supermemory", "record_count_available": false},
	}, nil
}

func (b *SupermemoryBackend) findRecord(ctx context.Context, scope Scope, recordID string) (supermemoryItem, error) {
	items, err := b.listScope(ctx, scope)
	if err != nil {
		return supermemoryItem{}, err
	}
	for _, item := range items {
		if stringValue(item.Metadata["record_id"]) == recordID && !item.IsForgotten {
			return item, nil
		}
	}
	return supermemoryItem{}, fmt.Errorf("supermemory record %q not found in scope %s/%s", recordID, scope.TenantID, scope.ContinuityID)
}
