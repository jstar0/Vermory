package memorybackend

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"
)

type MemOSConfig struct {
	BaseURL string
	APIKey  string
	Client  *http.Client
}

type MemOSBackend struct {
	remote remoteClient
}

type memosEnvelope struct {
	Code    int             `json:"code"`
	Message string          `json:"message"`
	Data    json.RawMessage `json:"data"`
}

type memosData struct {
	TextMem []struct {
		Memories []memosItem `json:"memories"`
	} `json:"text_mem"`
}

type memosItem struct {
	ID       string         `json:"id"`
	Memory   string         `json:"memory"`
	Metadata map[string]any `json:"metadata"`
}

func NewMemOSBackend(config MemOSConfig) (*MemOSBackend, error) {
	headers := map[string]string{}
	if config.APIKey != "" {
		headers["Authorization"] = "Bearer " + config.APIKey
	}
	remote, err := newRemoteClient(config.BaseURL, config.Client, headers)
	if err != nil {
		return nil, err
	}
	return &MemOSBackend{remote: remote}, nil
}

func (b *MemOSBackend) Name() string { return "memos" }

func (b *MemOSBackend) Health(ctx context.Context) error {
	return b.remote.doJSON(ctx, http.MethodGet, "/openapi.json", nil, &map[string]any{})
}

func (b *MemOSBackend) Put(ctx context.Context, record Record) error {
	body := map[string]any{
		"user_id": record.Scope.TenantID, "session_id": record.Scope.ContinuityID,
		"writable_cube_ids": []string{opaqueScopeTag(record.Scope)},
		"async_mode":        "sync", "mode": "fast",
		"messages": []map[string]string{{"role": "user", "content": record.Content}},
		"info":     memosRecordMetadata(record),
	}
	return b.do(ctx, http.MethodPost, "/product/add", body, nil)
}

func (b *MemOSBackend) Search(ctx context.Context, query Query) ([]Result, error) {
	var items []memosItem
	var err error
	if query.Text == "" {
		items, err = b.listScope(ctx, query.Scope, nil)
	} else {
		items, err = b.searchScope(ctx, query)
	}
	if err != nil {
		return nil, err
	}
	results := make([]Result, 0, len(items))
	for _, item := range items {
		record := memosRecordFromItem(query.Scope, item)
		if record.ID == "" || !activeResult(query, record) {
			continue
		}
		results = append(results, Result{Record: record, Score: floatValue(item.Metadata["relativity"])})
	}
	return truncateResults(results, query.Limit), nil
}

func (b *MemOSBackend) searchScope(ctx context.Context, query Query) ([]memosItem, error) {
	limit := query.Limit * 4
	if limit < 20 {
		limit = 20
	}
	body := map[string]any{
		"query": query.Text, "user_id": query.Scope.TenantID,
		"readable_cube_ids": []string{opaqueScopeTag(query.Scope)},
		"session_id":        query.Scope.ContinuityID, "mode": "fast", "top_k": limit, "relativity": 0,
		"filter": map[string]any{"cm_lifecycle_status": "active"},
		"dedup":  "no", "rerank": false,
	}
	var data memosData
	if err := b.do(ctx, http.MethodPost, "/product/search", body, &data); err != nil {
		return nil, err
	}
	return flattenMemosItems(data), nil
}

func (b *MemOSBackend) listScope(ctx context.Context, scope Scope, filter map[string]any) ([]memosItem, error) {
	body := map[string]any{
		"mem_cube_id": opaqueScopeTag(scope), "user_id": scope.TenantID,
		"include_preference": false, "include_tool_memory": false, "include_skill_memory": false,
		"page": 1, "page_size": 1000,
	}
	if filter != nil {
		body["filter"] = filter
	}
	var data memosData
	if err := b.do(ctx, http.MethodPost, "/product/get_memory", body, &data); err != nil {
		return nil, err
	}
	return flattenMemosItems(data), nil
}

func (b *MemOSBackend) Update(ctx context.Context, record Record) error {
	if err := b.Delete(ctx, record.Scope, record.ID); err != nil {
		return err
	}
	return b.Put(ctx, record)
}

func (b *MemOSBackend) Delete(ctx context.Context, scope Scope, recordID string) error {
	items, err := b.listScope(ctx, scope, map[string]any{"cm_record_id": recordID})
	if err != nil {
		return err
	}
	if len(items) == 0 {
		return fmt.Errorf("memos record %q not found in scope %s/%s", recordID, scope.TenantID, scope.ContinuityID)
	}
	ids := make([]string, 0, len(items))
	for _, item := range items {
		ids = append(ids, item.ID)
	}
	body := map[string]any{"writable_cube_ids": []string{opaqueScopeTag(scope)}, "memory_ids": ids}
	return b.do(ctx, http.MethodPost, "/product/delete_memory", body, nil)
}

func (b *MemOSBackend) ResetScope(ctx context.Context, scope Scope) error {
	items, err := b.listScope(ctx, scope, nil)
	if err != nil {
		return err
	}
	if len(items) == 0 {
		return nil
	}
	ids := make([]string, 0, len(items))
	for _, item := range items {
		ids = append(ids, item.ID)
	}
	body := map[string]any{"writable_cube_ids": []string{opaqueScopeTag(scope)}, "memory_ids": ids}
	return b.do(ctx, http.MethodPost, "/product/delete_memory", body, nil)
}

func (b *MemOSBackend) RebuildScope(ctx context.Context, scope Scope, records []Record) error {
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

func (b *MemOSBackend) Stats(context.Context) (Stats, error) {
	return Stats{MeasuredAt: time.Now().UTC(), Extra: map[string]any{
		"provider": "memos", "record_count_available": false,
	}}, nil
}

func (b *MemOSBackend) do(ctx context.Context, method, path string, body, data any) error {
	var envelope memosEnvelope
	if err := b.remote.doJSON(ctx, method, path, body, &envelope); err != nil {
		return err
	}
	if envelope.Code != 200 {
		return fmt.Errorf("memos %s returned code %d: %s", path, envelope.Code, envelope.Message)
	}
	if len(envelope.Data) > 0 && string(envelope.Data) != "null" {
		var status struct {
			Status string `json:"status"`
		}
		if json.Unmarshal(envelope.Data, &status) == nil && status.Status == "failure" {
			return fmt.Errorf("memos %s failed: %s", path, envelope.Message)
		}
	}
	if data != nil && len(envelope.Data) > 0 && string(envelope.Data) != "null" {
		if err := json.Unmarshal(envelope.Data, data); err != nil {
			return fmt.Errorf("decode memos %s data: %w", path, err)
		}
	}
	return nil
}

func memosRecordMetadata(record Record) map[string]any {
	metadata := map[string]any{
		"cm_record_id": record.ID, "cm_tenant_id": record.Scope.TenantID,
		"cm_continuity_id": record.Scope.ContinuityID, "cm_continuity_line": record.Scope.ContinuityLine,
		"cm_source_id": record.SourceID, "cm_source_version": record.SourceVersion,
		"cm_lifecycle_status": record.Status,
	}
	for key, value := range record.Metadata {
		metadata["cm_meta_"+key] = value
	}
	return metadata
}

func memosRecordFromItem(scope Scope, item memosItem) Record {
	record := Record{
		ID: stringValue(item.Metadata["cm_record_id"]), Scope: scope,
		SourceID:      stringValue(item.Metadata["cm_source_id"]),
		SourceVersion: intValue(item.Metadata["cm_source_version"]),
		Status:        stringValue(item.Metadata["cm_lifecycle_status"]), Content: item.Memory,
		Metadata: map[string]string{},
	}
	if record.Status == "" {
		record.Status = "active"
	}
	for key, value := range item.Metadata {
		if strings.HasPrefix(key, "cm_meta_") {
			record.Metadata[strings.TrimPrefix(key, "cm_meta_")] = stringValue(value)
		}
	}
	if len(record.Metadata) == 0 {
		record.Metadata = nil
	}
	return record
}

func flattenMemosItems(data memosData) []memosItem {
	var items []memosItem
	for _, bucket := range data.TextMem {
		items = append(items, bucket.Memories...)
	}
	return items
}

func floatValue(value any) float64 {
	var number float64
	_, _ = fmt.Sscan(stringValue(value), &number)
	return number
}
