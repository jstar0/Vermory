package retrievalablation

import (
	"context"
	"fmt"
	"sort"
	"strings"
	"time"

	"vermory/internal/memorybackend"
	"vermory/internal/runtime"
)

type Options struct {
	RunID                  string
	Corpus                 Corpus
	DatabaseURL            string
	CorpusPath             string
	RepositoryRoot         string
	EmbeddingBaseURL       string
	EmbeddingAPIKey        string
	EmbeddingModel         string
	EmbeddingDimensions    int
	ImplementationRevision string
}

func RunWithDependencies(
	ctx context.Context,
	options Options,
	store *runtime.Store,
	backend memorybackend.Backend,
) (Report, error) {
	if strings.TrimSpace(options.RunID) == "" {
		return Report{}, fmt.Errorf("run_id is required")
	}
	if len(options.Corpus.Queries) == 0 {
		return Report{}, fmt.Errorf("retrieval corpus requires queries")
	}
	corpusHash, err := CorpusSHA256(options.Corpus)
	if err != nil {
		return Report{}, err
	}
	seeded, err := SeedCorpus(ctx, store, backend, options.Corpus, options.RunID)
	if err != nil {
		return Report{}, err
	}
	conditions, failures, err := executeConditions(ctx, store, backend, options.Corpus, seeded)
	if err != nil {
		return Report{}, err
	}
	rebuildEquivalent, err := rebuildAndCompare(ctx, store, backend, options.Corpus, seeded, conditions)
	if err != nil {
		return Report{}, err
	}
	report := Report{
		RunID:                       options.RunID,
		CorpusSHA256:                corpusHash,
		Conditions:                  conditions,
		ProjectionRebuildEquivalent: rebuildEquivalent,
		Failures:                    failures,
	}
	for _, condition := range conditions {
		report.HardGates.ForbiddenCount += condition.Metrics.ForbiddenCount
		report.HardGates.IneligibleCount += condition.Metrics.IneligibleCount
	}
	report.HardGates.Pass = report.HardGates.IneligibleCount == 0 && rebuildEquivalent
	return report, nil
}

func executeConditions(
	ctx context.Context,
	store *runtime.Store,
	backend memorybackend.Backend,
	corpus Corpus,
	seeded SeededCorpus,
) ([]ConditionReport, []RunFailure, error) {
	lexicalReport := ConditionReport{Name: ConditionLexical}
	vectorReport := ConditionReport{Name: ConditionVector}
	hybridReport := ConditionReport{Name: ConditionHybrid}
	failures := make([]RunFailure, 0)
	memoryToRecord := make(map[string]string, len(seeded.Records))
	for recordID, seededRecord := range seeded.Records {
		memoryToRecord[seededRecord.MemoryID] = recordID
	}

	for _, query := range corpus.Queries {
		scope, exists := seeded.Scopes[query.ScopeID]
		if !exists {
			return nil, nil, fmt.Errorf("query %q references unseeded scope %q", query.ID, query.ScopeID)
		}
		active, err := activeMemorySet(ctx, store, scope)
		if err != nil {
			return nil, nil, fmt.Errorf("load active authority for query %q: %w", query.ID, err)
		}

		lexicalStarted := time.Now()
		lexicalMemories, err := store.SearchActiveMemory(ctx, scope.TenantID, scope.ContinuityID, query.Text, 12)
		lexicalDuration := time.Since(lexicalStarted)
		if err != nil {
			return nil, nil, fmt.Errorf("lexical query %q: %w", query.ID, err)
		}
		lexicalResults := make([]RankedResult, 0, len(lexicalMemories))
		for index, memory := range lexicalMemories {
			lexicalResults = append(lexicalResults, RankedResult{
				MemoryID: memory.ID, RecordID: memoryToRecord[memory.ID], Content: memory.Content,
				LexicalRank: index + 1, Eligible: active[memory.ID],
			})
		}
		lexicalDelivered := truncateRanked(lexicalResults, query.Limit)
		lexicalQuery := QueryReport{
			QueryID: query.ID, Cohorts: append([]string(nil), query.Cohorts...), Duration: lexicalDuration,
			Results: lexicalDelivered, Metrics: ScoreQuery(query, lexicalDelivered),
		}
		lexicalReport.Queries = append(lexicalReport.Queries, lexicalQuery)

		vectorStarted := time.Now()
		vectorBackendResults, vectorErr := backend.Search(ctx, memorybackend.Query{
			Scope: scope.BackendScope, Text: query.Text, Limit: vectorCandidateLimit(query.Limit),
		})
		vectorDuration := time.Since(vectorStarted)
		if vectorErr != nil {
			message := vectorErr.Error()
			vectorReport.Queries = append(vectorReport.Queries, QueryReport{
				QueryID: query.ID, Cohorts: append([]string(nil), query.Cohorts...), Duration: vectorDuration, Error: message,
			})
			fallback := FallbackToLexical(lexicalResults, query.Limit)
			hybridReport.Queries = append(hybridReport.Queries, QueryReport{
				QueryID: query.ID, Cohorts: append([]string(nil), query.Cohorts...), Duration: lexicalDuration + vectorDuration,
				Results: fallback, Metrics: ScoreQuery(query, fallback), DegradedToLexical: true,
			})
			failures = append(failures, RunFailure{Condition: ConditionVector, QueryID: query.ID, Error: message})
			continue
		}

		vectorEligible := make([]RankedResult, 0, len(vectorBackendResults))
		vectorRejected := make([]RankedResult, 0)
		for index, backendResult := range vectorBackendResults {
			recordID := backendResult.Record.Metadata["record_id"]
			if recordID == "" {
				recordID = memoryToRecord[backendResult.Record.ID]
			}
			result := RankedResult{
				MemoryID: backendResult.Record.ID, RecordID: recordID, Content: backendResult.Record.Content,
				Score: backendResult.Score, VectorRank: index + 1, Eligible: active[backendResult.Record.ID],
			}
			if result.Eligible {
				vectorEligible = append(vectorEligible, result)
			} else {
				vectorRejected = append(vectorRejected, result)
			}
		}
		vectorDelivered := truncateRanked(vectorEligible, query.Limit)
		vectorMetrics := ScoreQuery(query, vectorDelivered)
		vectorMetrics.IneligibleCount += len(vectorRejected)
		vectorReport.Queries = append(vectorReport.Queries, QueryReport{
			QueryID: query.ID, Cohorts: append([]string(nil), query.Cohorts...), Duration: vectorDuration,
			Results: vectorDelivered, RejectedResults: vectorRejected, Metrics: vectorMetrics,
		})

		fusionStarted := time.Now()
		hybridResults := FuseRRF(query.Text, query.Limit, lexicalResults, vectorEligible)
		hybridDuration := lexicalDuration + vectorDuration + time.Since(fusionStarted)
		hybridReport.Queries = append(hybridReport.Queries, QueryReport{
			QueryID: query.ID, Cohorts: append([]string(nil), query.Cohorts...), Duration: hybridDuration,
			Results: hybridResults, Metrics: ScoreQuery(query, hybridResults),
		})
	}

	for _, report := range []*ConditionReport{&lexicalReport, &vectorReport, &hybridReport} {
		report.Metrics = AggregateMetrics(report.Queries)
		report.Cohorts = AggregateCohorts(report.Queries)
	}
	return []ConditionReport{lexicalReport, vectorReport, hybridReport}, failures, nil
}

func rebuildAndCompare(
	ctx context.Context,
	store *runtime.Store,
	backend memorybackend.Backend,
	corpus Corpus,
	seeded SeededCorpus,
	before []ConditionReport,
) (bool, error) {
	scopeIDs := make([]string, 0, len(seeded.Scopes))
	for scopeID := range seeded.Scopes {
		scopeIDs = append(scopeIDs, scopeID)
	}
	sort.Strings(scopeIDs)
	for _, scopeID := range scopeIDs {
		records := append([]memorybackend.Record(nil), seeded.BackendRecords[scopeID]...)
		sort.Slice(records, func(i, j int) bool { return records[i].ID < records[j].ID })
		if err := backend.RebuildScope(ctx, seeded.Scopes[scopeID].BackendScope, records); err != nil {
			return false, fmt.Errorf("rebuild vector scope %q: %w", scopeID, err)
		}
	}
	after, _, err := executeConditions(ctx, store, backend, corpus, seeded)
	if err != nil {
		return false, err
	}
	return comparableConditionResults(before) == comparableConditionResults(after), nil
}

func comparableConditionResults(conditions []ConditionReport) string {
	type queryResult struct {
		Condition string
		QueryID   string
		Results   []string
		Rejected  []string
		Degraded  bool
		Error     string
	}
	comparable := make([]queryResult, 0)
	for _, condition := range conditions {
		if condition.Name == ConditionLexical {
			continue
		}
		for _, query := range condition.Queries {
			comparable = append(comparable, queryResult{
				Condition: condition.Name,
				QueryID:   query.QueryID,
				Results:   recordIDs(query.Results),
				Rejected:  recordIDs(query.RejectedResults),
				Degraded:  query.DegradedToLexical,
				Error:     query.Error,
			})
		}
	}
	return fmt.Sprintf("%#v", comparable)
}

func activeMemorySet(ctx context.Context, store *runtime.Store, scope SeededScope) (map[string]bool, error) {
	memories, err := store.ListGovernedMemories(ctx, scope.TenantID, scope.ContinuityID)
	if err != nil {
		return nil, err
	}
	active := make(map[string]bool, len(memories))
	for _, memory := range memories {
		if memory.LifecycleStatus == "active" {
			active[memory.ID] = true
		}
	}
	return active, nil
}

func vectorCandidateLimit(limit int) int {
	candidates := limit * 4
	if candidates < 20 {
		candidates = 20
	}
	if candidates > 100 {
		candidates = 100
	}
	return candidates
}

func truncateRanked(results []RankedResult, limit int) []RankedResult {
	if limit <= 0 || len(results) == 0 {
		return nil
	}
	if limit > len(results) {
		limit = len(results)
	}
	return append([]RankedResult(nil), results[:limit]...)
}

func recordIDs(results []RankedResult) []string {
	ids := make([]string, len(results))
	for index, result := range results {
		ids[index] = result.RecordID
	}
	return ids
}
