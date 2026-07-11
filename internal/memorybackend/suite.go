package memorybackend

import (
	"context"
	"fmt"
	"sort"
	"strings"
	"time"
)

type GateResult struct {
	Name    string `json:"name"`
	Pass    bool   `json:"pass"`
	Details string `json:"details,omitempty"`
}

type LifecycleReport struct {
	Backend   string        `json:"backend"`
	Pass      bool          `json:"pass"`
	StartedAt time.Time     `json:"started_at"`
	Duration  time.Duration `json:"duration"`
	Gates     []GateResult  `json:"gates"`
	Stats     Stats         `json:"stats"`
}

func RunLifecycleSuite(ctx context.Context, backend Backend) LifecycleReport {
	started := time.Now().UTC()
	report := LifecycleReport{
		Backend:   backend.Name(),
		Pass:      true,
		StartedAt: started,
	}

	addGate := func(name string, err error) {
		gate := GateResult{Name: name, Pass: err == nil}
		if err != nil {
			gate.Details = err.Error()
			report.Pass = false
		}
		report.Gates = append(report.Gates, gate)
	}

	if err := backend.Health(ctx); err != nil {
		addGate("health", err)
		report.Duration = time.Since(started)
		return report
	}
	addGate("health", nil)

	webScope := Scope{TenantID: "bakeoff", ContinuityID: "workspace:web-checkout", ContinuityLine: "workspace"}
	opsScope := Scope{TenantID: "bakeoff", ContinuityID: "workspace:ops-console", ContinuityLine: "workspace"}
	for _, scope := range []Scope{webScope, opsScope} {
		if err := backend.ResetScope(ctx, scope); err != nil {
			addGate("setup_reset", err)
			report.Duration = time.Since(started)
			return report
		}
	}

	webRecord := Record{
		ID:            "web-current",
		Scope:         webScope,
		SourceID:      "repo:web-checkout",
		SourceVersion: 1,
		Status:        "active",
		Content:       "web-checkout uses checkout_eta_v2 and owner #checkout-ops for delivery estimate loading.",
	}
	opsRecord := Record{
		ID:            "ops-current",
		Scope:         opsScope,
		SourceID:      "repo:ops-console",
		SourceVersion: 1,
		Status:        "active",
		Content:       "ops-console uses ops_exception_queue_refresh and owner #ops-console for shipment exception review.",
	}
	for _, record := range []Record{webRecord, opsRecord} {
		if err := backend.Put(ctx, record); err != nil {
			addGate("put", err)
			report.Duration = time.Since(started)
			return report
		}
	}
	addGate("put", nil)

	results, err := backend.Search(ctx, Query{Scope: webScope, Text: "checkout feature flag and owner", Limit: 10})
	if err == nil {
		err = requireContent(results, "checkout_eta_v2", "#checkout-ops")
	}
	addGate("current_fact_recall", err)
	addGate("cross_scope_isolation", forbidContent(results, "ops_exception_queue_refresh", "#ops-console"))

	oldRecord := Record{
		ID:            "database-old",
		Scope:         webScope,
		SourceID:      "decision:database",
		SourceVersion: 1,
		Status:        "active",
		Content:       "MongoDB is the current database for web-checkout.",
	}
	if err := backend.Put(ctx, oldRecord); err != nil {
		addGate("supersede_setup", err)
	} else {
		oldRecord.Status = "superseded"
		newRecord := Record{
			ID:            "database-current",
			Scope:         webScope,
			SourceID:      "decision:database",
			SourceVersion: 2,
			Status:        "active",
			Content:       "PostgreSQL is the current database for web-checkout.",
		}
		if err := backend.Update(ctx, oldRecord); err != nil {
			addGate("supersede_update", err)
		} else if err := backend.Put(ctx, newRecord); err != nil {
			addGate("supersede_put", err)
		} else {
			current, searchErr := backend.Search(ctx, Query{Scope: webScope, Text: "current database", Limit: 10})
			if searchErr == nil {
				searchErr = requireContent(current, "PostgreSQL")
			}
			if searchErr == nil {
				searchErr = forbidContent(current, "MongoDB")
			}
			addGate("superseded_current_suppression", searchErr)

			if deleteErr := backend.Delete(ctx, webScope, newRecord.ID); deleteErr != nil {
				addGate("delete", deleteErr)
			} else {
				deleted, deletedErr := backend.Search(ctx, Query{Scope: webScope, Text: "PostgreSQL current database", Limit: 10, IncludeHistory: true})
				if deletedErr == nil {
					deletedErr = forbidContent(deleted, "PostgreSQL")
				}
				addGate("delete_residue", deletedErr)
			}
		}
	}

	beforeReset, err := activeIDs(ctx, backend, webScope)
	if err != nil {
		addGate("rebuild_snapshot", err)
	} else {
		activeRecords := []Record{webRecord}
		if err := backend.ResetScope(ctx, webScope); err != nil {
			addGate("scope_reset", err)
		} else {
			empty, searchErr := backend.Search(ctx, Query{Scope: webScope, Text: "checkout", Limit: 100, IncludeHistory: true})
			if searchErr == nil && len(empty) != 0 {
				searchErr = fmt.Errorf("scope reset left %d records", len(empty))
			}
			addGate("scope_reset", searchErr)
			if rebuildErr := backend.RebuildScope(ctx, webScope, activeRecords); rebuildErr != nil {
				addGate("rebuild", rebuildErr)
			} else {
				afterRebuild, rebuildErr := activeIDs(ctx, backend, webScope)
				if rebuildErr == nil && !sameStrings(beforeReset, afterRebuild) {
					rebuildErr = fmt.Errorf("active record IDs changed: before=%v after=%v", beforeReset, afterRebuild)
				}
				addGate("rebuild_equivalence", rebuildErr)
			}
		}
	}

	stats, err := backend.Stats(ctx)
	addGate("stats", err)
	report.Stats = stats
	report.Duration = time.Since(started)
	return report
}

func activeIDs(ctx context.Context, backend Backend, scope Scope) ([]string, error) {
	results, err := backend.Search(ctx, Query{Scope: scope, Text: "", Limit: 100})
	if err != nil {
		return nil, err
	}
	ids := make([]string, 0, len(results))
	for _, result := range results {
		ids = append(ids, result.Record.ID)
	}
	sort.Strings(ids)
	return ids, nil
}

func requireContent(results []Result, phrases ...string) error {
	joined := joinContent(results)
	for _, phrase := range phrases {
		if !strings.Contains(strings.ToLower(joined), strings.ToLower(phrase)) {
			return fmt.Errorf("missing required content %q", phrase)
		}
	}
	return nil
}

func forbidContent(results []Result, phrases ...string) error {
	joined := joinContent(results)
	for _, phrase := range phrases {
		if strings.Contains(strings.ToLower(joined), strings.ToLower(phrase)) {
			return fmt.Errorf("retrieved forbidden content %q", phrase)
		}
	}
	return nil
}

func joinContent(results []Result) string {
	var values []string
	for _, result := range results {
		values = append(values, result.Record.Content)
	}
	return strings.Join(values, "\n")
}

func sameStrings(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}
