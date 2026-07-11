package memorybackend

import (
	"context"
	"sort"
	"strings"
	"sync"
	"testing"
	"time"
)

func TestLifecycleSuitePassesForCompliantBackend(t *testing.T) {
	backend := newTestBackend(false)
	report := RunLifecycleSuite(context.Background(), backend)
	if !report.Pass {
		t.Fatalf("expected pass, got %+v", report.Gates)
	}
}

func TestLifecycleSuiteRejectsCrossScopeLeakage(t *testing.T) {
	backend := newTestBackend(true)
	report := RunLifecycleSuite(context.Background(), backend)
	if report.Pass {
		t.Fatal("expected leakage to fail the suite")
	}
	for _, gate := range report.Gates {
		if gate.Name == "cross_scope_isolation" && !gate.Pass {
			return
		}
	}
	t.Fatal("expected cross_scope_isolation failure")
}

type testBackend struct {
	leak    bool
	mu      sync.RWMutex
	records map[string]Record
}

func newTestBackend(leak bool) *testBackend {
	return &testBackend{leak: leak, records: map[string]Record{}}
}

func (b *testBackend) Name() string { return "test" }

func (b *testBackend) Health(context.Context) error { return nil }

func (b *testBackend) Put(_ context.Context, record Record) error {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.records[key(record.Scope, record.ID)] = record
	return nil
}

func (b *testBackend) Search(_ context.Context, query Query) ([]Result, error) {
	b.mu.RLock()
	defer b.mu.RUnlock()
	var results []Result
	for _, record := range b.records {
		if record.Scope.TenantID != query.Scope.TenantID {
			continue
		}
		if !b.leak && record.Scope.ContinuityID != query.Scope.ContinuityID {
			continue
		}
		if !query.IncludeHistory && record.Status != "active" {
			continue
		}
		if query.Text != "" && !overlaps(query.Text, record.Content) {
			continue
		}
		results = append(results, Result{Record: record, Score: 1})
	}
	sort.Slice(results, func(i, j int) bool { return results[i].Record.ID < results[j].Record.ID })
	if query.Limit > 0 && len(results) > query.Limit {
		results = results[:query.Limit]
	}
	return results, nil
}

func (b *testBackend) Update(_ context.Context, record Record) error {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.records[key(record.Scope, record.ID)] = record
	return nil
}

func (b *testBackend) Delete(_ context.Context, scope Scope, recordID string) error {
	b.mu.Lock()
	defer b.mu.Unlock()
	delete(b.records, key(scope, recordID))
	return nil
}

func (b *testBackend) ResetScope(_ context.Context, scope Scope) error {
	b.mu.Lock()
	defer b.mu.Unlock()
	for id, record := range b.records {
		if record.Scope.TenantID == scope.TenantID && record.Scope.ContinuityID == scope.ContinuityID {
			delete(b.records, id)
		}
	}
	return nil
}

func (b *testBackend) RebuildScope(ctx context.Context, scope Scope, records []Record) error {
	if err := b.ResetScope(ctx, scope); err != nil {
		return err
	}
	for _, record := range records {
		if err := b.Put(ctx, record); err != nil {
			return err
		}
	}
	return nil
}

func (b *testBackend) Stats(context.Context) (Stats, error) {
	b.mu.RLock()
	defer b.mu.RUnlock()
	return Stats{RecordCount: len(b.records), MeasuredAt: time.Now().UTC()}, nil
}

func key(scope Scope, recordID string) string {
	return scope.TenantID + "/" + scope.ContinuityID + "/" + recordID
}

func overlaps(query, content string) bool {
	content = strings.ToLower(content)
	for _, token := range strings.Fields(strings.ToLower(query)) {
		if strings.Contains(content, strings.Trim(token, "#.,:;")) {
			return true
		}
	}
	return false
}
