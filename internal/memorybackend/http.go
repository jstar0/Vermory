package memorybackend

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"
)

const maxErrorBody = 8 << 10

type remoteClient struct {
	baseURL string
	client  *http.Client
	headers map[string]string
}

func newRemoteClient(baseURL string, client *http.Client, headers map[string]string) (remoteClient, error) {
	baseURL = strings.TrimRight(strings.TrimSpace(baseURL), "/")
	parsed, err := url.Parse(baseURL)
	if err != nil || parsed.Scheme == "" || parsed.Host == "" {
		return remoteClient{}, fmt.Errorf("invalid backend base URL %q", baseURL)
	}
	if client == nil {
		client = &http.Client{Timeout: 30 * time.Second}
	}
	return remoteClient{baseURL: baseURL, client: client, headers: headers}, nil
}

func (c remoteClient) doJSON(ctx context.Context, method, path string, body, out any) error {
	var reader io.Reader
	if body != nil {
		payload, err := json.Marshal(body)
		if err != nil {
			return fmt.Errorf("encode request: %w", err)
		}
		reader = bytes.NewReader(payload)
	}
	req, err := http.NewRequestWithContext(ctx, method, c.baseURL+path, reader)
	if err != nil {
		return fmt.Errorf("create request: %w", err)
	}
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	for key, value := range c.headers {
		if value != "" {
			req.Header.Set(key, value)
		}
	}
	resp, err := c.client.Do(req)
	if err != nil {
		return fmt.Errorf("%s %s: %w", method, path, err)
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		message, _ := io.ReadAll(io.LimitReader(resp.Body, maxErrorBody))
		return fmt.Errorf("%s %s returned %s: %s", method, path, resp.Status, strings.TrimSpace(string(message)))
	}
	if out == nil || resp.StatusCode == http.StatusNoContent {
		_, _ = io.Copy(io.Discard, resp.Body)
		return nil
	}
	if err := json.NewDecoder(resp.Body).Decode(out); err != nil {
		if errors.Is(err, io.EOF) {
			return nil
		}
		return fmt.Errorf("decode %s %s response: %w", method, path, err)
	}
	return nil
}

func recordMetadata(record Record) map[string]any {
	metadata := make(map[string]any, len(record.Metadata)+7)
	for key, value := range record.Metadata {
		metadata[key] = value
	}
	metadata["record_id"] = record.ID
	metadata["tenant_id"] = record.Scope.TenantID
	metadata["continuity_id"] = record.Scope.ContinuityID
	metadata["continuity_line"] = record.Scope.ContinuityLine
	metadata["source_id"] = record.SourceID
	metadata["source_version"] = record.SourceVersion
	metadata["lifecycle_status"] = record.Status
	return metadata
}

func recordFromProvider(scope Scope, content string, metadata map[string]any) Record {
	record := Record{
		ID:            stringValue(metadata["record_id"]),
		Scope:         scope,
		SourceID:      stringValue(metadata["source_id"]),
		SourceVersion: intValue(metadata["source_version"]),
		Status:        stringValue(metadata["lifecycle_status"]),
		Content:       content,
		Metadata:      map[string]string{},
	}
	if record.Status == "" {
		record.Status = "active"
	}
	reserved := map[string]bool{
		"record_id": true, "tenant_id": true, "continuity_id": true, "continuity_line": true,
		"source_id": true, "source_version": true, "lifecycle_status": true,
	}
	for key, value := range metadata {
		if !reserved[key] {
			record.Metadata[key] = stringValue(value)
		}
	}
	if len(record.Metadata) == 0 {
		record.Metadata = nil
	}
	return record
}

func stringValue(value any) string {
	switch value := value.(type) {
	case string:
		return value
	case json.Number:
		return value.String()
	case float64:
		return fmt.Sprintf("%g", value)
	case nil:
		return ""
	default:
		return fmt.Sprint(value)
	}
}

func intValue(value any) int {
	var number int
	_, _ = fmt.Sscan(stringValue(value), &number)
	return number
}

func activeResult(query Query, record Record) bool {
	return query.IncludeHistory || record.Status == "active"
}

func truncateResults(results []Result, limit int) []Result {
	if limit > 0 && len(results) > limit {
		return results[:limit]
	}
	return results
}

func opaqueScopeTag(scope Scope) string {
	sum := sha256.Sum256([]byte(scope.TenantID + "\x00" + scope.ContinuityID))
	return fmt.Sprintf("cm_%x", sum[:16])
}
