package app

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"sort"
	"strings"
	"time"

	"vermory/internal/artifact"
	"vermory/internal/benchmark"
	vermoryruntime "vermory/internal/runtime"
)

const (
	longMemEvalRetrievalBaseline = "plain_token_overlap"
	longMemEvalRetrievalVermory  = "vermory_lexical"
	longMemEvalRetrievalChannel  = "benchmark_longmemeval_retrieval"
	longMemEvalRetrievalLimit    = 12
)

type LongMemEvalRetrievalOptions struct {
	QualificationPath      string
	ExecutionPath          string
	SourceDatasetPath      string
	DatabaseURL            string
	ArtifactRoot           string
	RunID                  string
	ImplementationRevision string
	Resume                 bool
}

type LongMemEvalRetrievalReport struct {
	RunID                  string                                              `json:"run_id"`
	Benchmark              string                                              `json:"benchmark"`
	EvaluationTarget       benchmark.EvaluationTarget                          `json:"evaluation_target"`
	ExecutionScope         benchmark.ExecutionScope                            `json:"execution_scope"`
	ClaimScope             benchmark.ClaimScope                                `json:"claim_scope"`
	DatasetSHA256          string                                              `json:"dataset_sha256"`
	RecordSetSHA256        string                                              `json:"record_set_sha256"`
	ImplementationRevision string                                              `json:"implementation_revision"`
	SourceSummary          benchmark.LongMemEvalSummary                        `json:"source_summary"`
	RecordCount            int                                                 `json:"record_count"`
	ScoredRecordCount      int                                                 `json:"scored_record_count"`
	ImportedMemoryCount    int                                                 `json:"imported_memory_count"`
	Conditions             []string                                            `json:"conditions"`
	Results                []LongMemEvalRetrievalRecordResult                  `json:"results"`
	Aggregates             map[string]LongMemEvalRetrievalAggregate            `json:"aggregates"`
	QuestionTypeAggregates map[string]map[string]LongMemEvalRetrievalAggregate `json:"question_type_aggregates"`
	Failures               []LongMemEvalRetrievalFailure                       `json:"failures"`
	Artifacts              map[string]string                                   `json:"artifacts"`
	NonClaims              []string                                            `json:"non_claims"`
}

type LongMemEvalRetrievalRecordResult struct {
	SchemaVersion          string                                `json:"schema_version"`
	RunID                  string                                `json:"run_id"`
	ImplementationRevision string                                `json:"implementation_revision"`
	DatasetSHA256          string                                `json:"dataset_sha256"`
	RecordSetSHA256        string                                `json:"record_set_sha256"`
	RecordID               string                                `json:"record_id"`
	QuestionType           string                                `json:"question_type"`
	Abstention             bool                                  `json:"abstention"`
	Status                 string                                `json:"status"`
	ContinuityID           string                                `json:"continuity_id,omitempty"`
	ImportedMemoryCount    int                                   `json:"imported_memory_count"`
	Conditions             []LongMemEvalRetrievalConditionResult `json:"conditions,omitempty"`
	Error                  string                                `json:"error,omitempty"`
}

type LongMemEvalRetrievalConditionResult struct {
	Condition            string                            `json:"condition"`
	Status               string                            `json:"status"`
	Classification       string                            `json:"classification"`
	RankedOccurrenceKeys []string                          `json:"ranked_occurrence_keys"`
	RankedSessionIDs     []string                          `json:"ranked_session_ids"`
	MetricAt5            *benchmark.SessionRetrievalMetric `json:"metric_at_5,omitempty"`
	MetricAt10           *benchmark.SessionRetrievalMetric `json:"metric_at_10,omitempty"`
	MetricAt12           *benchmark.SessionRetrievalMetric `json:"metric_at_12,omitempty"`
	LatencyMilliseconds  int64                             `json:"latency_ms"`
	Error                string                            `json:"error,omitempty"`
}

type LongMemEvalRetrievalAggregate struct {
	Count int                          `json:"count"`
	At5   benchmark.RetrievalAggregate `json:"at_5"`
	At10  benchmark.RetrievalAggregate `json:"at_10"`
	At12  benchmark.RetrievalAggregate `json:"at_12"`
}

type LongMemEvalRetrievalFailure struct {
	RecordID     string `json:"record_id"`
	QuestionType string `json:"question_type"`
	Error        string `json:"error"`
}

type longMemEvalSessionOccurrence struct {
	Key     string
	RawID   string
	Session benchmark.LongMemEvalSession
}

func RunLongMemEvalRetrieval(ctx context.Context, opts LongMemEvalRetrievalOptions) (LongMemEvalRetrievalReport, error) {
	if strings.TrimSpace(opts.DatabaseURL) == "" {
		return LongMemEvalRetrievalReport{}, errors.New("benchmark-longmemeval-retrieval requires database-url")
	}
	if strings.TrimSpace(opts.SourceDatasetPath) == "" {
		return LongMemEvalRetrievalReport{}, errors.New("benchmark-longmemeval-retrieval requires source-dataset-path")
	}
	if strings.TrimSpace(opts.ExecutionPath) == "" {
		return LongMemEvalRetrievalReport{}, errors.New("benchmark-longmemeval-retrieval requires execution-path")
	}
	if strings.TrimSpace(opts.ArtifactRoot) == "" {
		opts.ArtifactRoot = "./artifacts"
	}

	root, err := projectRoot()
	if err != nil {
		return LongMemEvalRetrievalReport{}, err
	}
	executionPath := resolveBenchmarkPath(root, opts.ExecutionPath)
	execution, err := benchmark.LoadExecution(executionPath)
	if err != nil {
		return LongMemEvalRetrievalReport{}, err
	}
	qualificationPath := strings.TrimSpace(opts.QualificationPath)
	if qualificationPath == "" {
		qualificationPath = execution.QualificationPath
	}
	qualification, err := benchmark.LoadQualification(resolveBenchmarkPath(root, qualificationPath))
	if err != nil {
		return LongMemEvalRetrievalReport{}, err
	}
	if err := benchmark.ValidateExecution(qualification, execution); err != nil {
		return LongMemEvalRetrievalReport{}, err
	}
	if execution.EvaluationTarget != benchmark.EvaluationTargetRetrieval || execution.ExecutionScope != benchmark.ExecutionScopeFull {
		return LongMemEvalRetrievalReport{}, fmt.Errorf("LongMemEval retrieval runner requires a full retrieval execution")
	}
	conditions := longMemEvalRetrievalConditions()
	if !slices.Equal(execution.Conditions, conditions) {
		return LongMemEvalRetrievalReport{}, fmt.Errorf("LongMemEval retrieval conditions are %v, want %v", execution.Conditions, conditions)
	}

	sourcePath := resolveBenchmarkPath(root, opts.SourceDatasetPath)
	if err := benchmark.VerifyFileSHA256(sourcePath, qualification.Dataset.SHA256); err != nil {
		return LongMemEvalRetrievalReport{}, fmt.Errorf("verify LongMemEval-S source: %w", err)
	}
	info, err := os.Stat(sourcePath)
	if err != nil {
		return LongMemEvalRetrievalReport{}, err
	}
	if info.Size() != qualification.Dataset.SizeBytes {
		return LongMemEvalRetrievalReport{}, fmt.Errorf("LongMemEval-S source size is %d, want %d", info.Size(), qualification.Dataset.SizeBytes)
	}
	summary, err := benchmark.ScanLongMemEval(sourcePath, nil)
	if err != nil {
		return LongMemEvalRetrievalReport{}, err
	}
	if err := validateLongMemEvalRetrievalSummary(qualification, execution, summary); err != nil {
		return LongMemEvalRetrievalReport{}, err
	}

	runID := chooseRunID(opts.RunID, "longmemeval-s-full-retrieval")
	if err := validateLongMemEvalRetrievalSegment(runID, "run ID"); err != nil {
		return LongMemEvalRetrievalReport{}, err
	}
	implementationRevision := strings.TrimSpace(opts.ImplementationRevision)
	if implementationRevision == "" {
		implementationRevision = buildVCSRevision()
	}
	tenantID := "benchmark:" + runID
	store, err := vermoryruntime.OpenStore(ctx, opts.DatabaseURL)
	if err != nil {
		return LongMemEvalRetrievalReport{}, err
	}
	defer store.Close()
	if err := store.Migrate(ctx); err != nil {
		return LongMemEvalRetrievalReport{}, err
	}
	retriever, err := vermoryruntime.NewRetrievalCoordinator(store, nil, vermoryruntime.RetrievalProfile{})
	if err != nil {
		return LongMemEvalRetrievalReport{}, err
	}
	artifactStore := artifact.NewLocalStore(opts.ArtifactRoot)
	prefix := filepath.ToSlash(filepath.Join("benchmarks", runID))
	results := make([]LongMemEvalRetrievalRecordResult, 0, summary.RecordCount)

	_, err = benchmark.ScanLongMemEval(sourcePath, func(record benchmark.LongMemEvalRecord) error {
		checkpointPath, err := longMemEvalRetrievalCheckpointPath(opts.ArtifactRoot, runID, record.QuestionID)
		if err != nil {
			return err
		}
		if _, statErr := os.Stat(checkpointPath); statErr == nil {
			if !opts.Resume {
				return fmt.Errorf("checkpoint already exists for record %q; use --resume", record.QuestionID)
			}
			checkpoint, err := loadLongMemEvalRetrievalCheckpoint(checkpointPath)
			if err != nil {
				return err
			}
			if err := validateLongMemEvalRetrievalCheckpoint(checkpoint, runID, implementationRevision, execution, record.QuestionID, conditions); err != nil {
				return err
			}
			results = append(results, checkpoint)
			return nil
		} else if !errors.Is(statErr, os.ErrNotExist) {
			return statErr
		}

		checkpoint, runErr := runLongMemEvalRetrievalRecord(ctx, store, retriever, tenantID, runID, implementationRevision, execution, record)
		if runErr != nil {
			checkpoint.Status = "runtime_failure"
			checkpoint.Error = runErr.Error()
		}
		key := filepath.ToSlash(filepath.Join(prefix, "checkpoints", record.QuestionID+".json"))
		if _, err := putJSONArtifact(ctx, artifactStore, key, checkpoint); err != nil {
			return err
		}
		results = append(results, checkpoint)
		return nil
	})
	if err != nil {
		return LongMemEvalRetrievalReport{}, err
	}

	report := finalizeLongMemEvalRetrievalReport(runID, implementationRevision, qualification, execution, summary, results)
	if err := writeLongMemEvalRetrievalArtifacts(ctx, artifactStore, opts.ArtifactRoot, prefix, qualification, execution, &report); err != nil {
		return LongMemEvalRetrievalReport{}, err
	}
	if len(report.Failures) != 0 || report.ScoredRecordCount != execution.ExpectedScoredRecordCount {
		return report, fmt.Errorf("LongMemEval retrieval completed with %d runtime failures and %d/%d scored records; report=%s",
			len(report.Failures), report.ScoredRecordCount, execution.ExpectedScoredRecordCount, report.Artifacts["report"])
	}
	return report, nil
}

func longMemEvalRetrievalConditions() []string {
	return []string{longMemEvalRetrievalBaseline, longMemEvalRetrievalVermory}
}

func validateLongMemEvalRetrievalSummary(qualification benchmark.Qualification, execution benchmark.ExecutionManifest, summary benchmark.LongMemEvalSummary) error {
	if summary.RecordCount != qualification.Dataset.RecordCount {
		return fmt.Errorf("LongMemEval-S source has %d records, want %d", summary.RecordCount, qualification.Dataset.RecordCount)
	}
	if summary.RecordSetSHA256 != execution.RecordSetSHA256 {
		return fmt.Errorf("LongMemEval-S record-set digest is %s, want %s", summary.RecordSetSHA256, execution.RecordSetSHA256)
	}
	if summary.SessionCount != execution.ExpectedSessionCount {
		return fmt.Errorf("LongMemEval-S source has %d sessions, want %d", summary.SessionCount, execution.ExpectedSessionCount)
	}
	if summary.TurnCount != execution.ExpectedTurnCount {
		return fmt.Errorf("LongMemEval-S source has %d turns, want %d", summary.TurnCount, execution.ExpectedTurnCount)
	}
	if summary.ScoredRecordCount != execution.ExpectedScoredRecordCount {
		return fmt.Errorf("LongMemEval-S source has %d scored records, want %d", summary.ScoredRecordCount, execution.ExpectedScoredRecordCount)
	}
	return nil
}

func runLongMemEvalRetrievalRecord(
	ctx context.Context,
	store *vermoryruntime.Store,
	retriever *vermoryruntime.RetrievalCoordinator,
	tenantID, runID, implementationRevision string,
	execution benchmark.ExecutionManifest,
	record benchmark.LongMemEvalRecord,
) (checkpoint LongMemEvalRetrievalRecordResult, err error) {
	checkpoint = LongMemEvalRetrievalRecordResult{
		SchemaVersion:          "longmemeval-retrieval-checkpoint/v1",
		RunID:                  runID,
		ImplementationRevision: implementationRevision,
		DatasetSHA256:          execution.DatasetSHA256,
		RecordSetSHA256:        execution.RecordSetSHA256,
		RecordID:               record.QuestionID,
		QuestionType:           record.QuestionType,
		Abstention:             strings.HasSuffix(record.QuestionID, "_abs"),
		Status:                 "completed",
	}
	anchor := vermoryruntime.ConversationAnchor{Channel: longMemEvalRetrievalChannel, ThreadID: runID + ":" + record.QuestionID}
	resolution, err := store.ResolveOrCreateConversation(ctx, tenantID, anchor)
	if err != nil {
		return checkpoint, err
	}
	checkpoint.ContinuityID = resolution.ContinuityID
	occurrences := longMemEvalRetrievalOccurrences(record)
	byMemoryID := make(map[string]longMemEvalSessionOccurrence, len(occurrences))
	for _, occurrence := range occurrences {
		receipt, err := store.CommitGovernedObservation(ctx, tenantID, resolution.ContinuityID, vermoryruntime.CommitObservationRequest{
			OperationID: fmt.Sprintf("%s:%s:source:%06d:%s", runID, record.QuestionID, occurrence.Session.Position, occurrence.RawID),
			Kind:        vermoryruntime.ObservationKindSourceUpdate,
			Content:     occurrence.Session.SemanticText(),
			SourceRef:   fmt.Sprintf("longmemeval-s:%s:%s:%06d:%s", execution.DatasetSHA256, record.QuestionID, occurrence.Session.Position, occurrence.RawID),
		})
		if err != nil {
			return checkpoint, err
		}
		if receipt.Memory.Status != "active" {
			return checkpoint, fmt.Errorf("session occurrence %s was not activated", occurrence.Key)
		}
		byMemoryID[receipt.Memory.MemoryID] = occurrence
		checkpoint.ImportedMemoryCount++
	}

	authority, err := store.ListGovernedMemories(ctx, tenantID, resolution.ContinuityID)
	if err != nil {
		return checkpoint, err
	}
	if len(authority) != len(occurrences) {
		return checkpoint, fmt.Errorf("continuity %s has %d governed memories, want %d", resolution.ContinuityID, len(authority), len(occurrences))
	}
	active := make(map[string]struct{}, len(authority))
	for _, memory := range authority {
		if memory.LifecycleStatus != "active" {
			return checkpoint, fmt.Errorf("continuity %s contains non-active memory %s", resolution.ContinuityID, memory.ID)
		}
		active[memory.ID] = struct{}{}
	}
	for memoryID := range byMemoryID {
		if _, exists := active[memoryID]; !exists {
			return checkpoint, fmt.Errorf("imported memory %s is missing from active authority", memoryID)
		}
	}

	baselineStarted := time.Now()
	baselineSessions := benchmark.RetrieveSessions(record, longMemEvalRetrievalLimit)
	baselineKeys := make([]string, 0, len(baselineSessions))
	baselineIDs := make([]string, 0, len(baselineSessions))
	for _, session := range baselineSessions {
		baselineKeys = append(baselineKeys, longMemEvalRetrievalOccurrenceKey(session.Position, session.ID))
		baselineIDs = append(baselineIDs, session.ID)
	}
	baseline, err := buildLongMemEvalRetrievalCondition(record, longMemEvalRetrievalBaseline, baselineKeys, baselineIDs, time.Since(baselineStarted))
	if err != nil {
		return checkpoint, err
	}

	vermoryStarted := time.Now()
	retrieved, err := retriever.Retrieve(ctx, vermoryruntime.RetrievalRequest{
		TenantID:      tenantID,
		ContinuityIDs: []string{resolution.ContinuityID},
		Query:         record.Question,
		Limit:         longMemEvalRetrievalLimit,
		Mode:          vermoryruntime.RetrievalLexical,
	})
	if err != nil {
		return checkpoint, err
	}
	vermoryKeys := make([]string, 0, len(retrieved.Memories))
	vermoryIDs := make([]string, 0, len(retrieved.Memories))
	for _, memory := range retrieved.Memories {
		if _, exists := active[memory.ID]; !exists {
			return checkpoint, fmt.Errorf("retrieval returned memory %s outside active continuity authority", memory.ID)
		}
		occurrence, exists := byMemoryID[memory.ID]
		if !exists {
			return checkpoint, fmt.Errorf("retrieval returned unmapped memory %s", memory.ID)
		}
		vermoryKeys = append(vermoryKeys, occurrence.Key)
		vermoryIDs = append(vermoryIDs, occurrence.RawID)
	}
	vermoryResult, err := buildLongMemEvalRetrievalCondition(record, longMemEvalRetrievalVermory, vermoryKeys, vermoryIDs, time.Since(vermoryStarted))
	if err != nil {
		return checkpoint, err
	}
	checkpoint.Conditions = []LongMemEvalRetrievalConditionResult{baseline, vermoryResult}
	return checkpoint, nil
}

func longMemEvalRetrievalOccurrences(record benchmark.LongMemEvalRecord) []longMemEvalSessionOccurrence {
	occurrences := make([]longMemEvalSessionOccurrence, 0, len(record.HaystackSessions))
	for position, turns := range record.HaystackSessions {
		session := benchmark.LongMemEvalSession{
			ID:       record.HaystackSessionIDs[position],
			Date:     record.HaystackDates[position],
			Turns:    append([]benchmark.LongMemEvalTurn(nil), turns...),
			Position: position,
		}
		occurrences = append(occurrences, longMemEvalSessionOccurrence{
			Key:     longMemEvalRetrievalOccurrenceKey(position, session.ID),
			RawID:   session.ID,
			Session: session,
		})
	}
	return occurrences
}

func longMemEvalRetrievalOccurrenceKey(position int, rawID string) string {
	return fmt.Sprintf("%06d:%s", position, rawID)
}

func buildLongMemEvalRetrievalCondition(record benchmark.LongMemEvalRecord, condition string, keys, ids []string, latency time.Duration) (LongMemEvalRetrievalConditionResult, error) {
	result := LongMemEvalRetrievalConditionResult{
		Condition:            condition,
		Status:               "completed",
		RankedOccurrenceKeys: append([]string(nil), keys...),
		RankedSessionIDs:     append([]string(nil), ids...),
		LatencyMilliseconds:  latency.Milliseconds(),
	}
	if strings.HasSuffix(record.QuestionID, "_abs") {
		result.Classification = "abstention_unscored"
		return result, nil
	}
	at5, err := benchmark.EvaluateSessionRetrieval(ids, record.AnswerSessionIDs, 5)
	if err != nil {
		return LongMemEvalRetrievalConditionResult{}, err
	}
	at10, err := benchmark.EvaluateSessionRetrieval(ids, record.AnswerSessionIDs, 10)
	if err != nil {
		return LongMemEvalRetrievalConditionResult{}, err
	}
	at12, err := benchmark.EvaluateSessionRetrieval(ids, record.AnswerSessionIDs, 12)
	if err != nil {
		return LongMemEvalRetrievalConditionResult{}, err
	}
	result.MetricAt5 = &at5
	result.MetricAt10 = &at10
	result.MetricAt12 = &at12
	switch {
	case at12.RecallAll == 1:
		result.Classification = "all_evidence_retrieved"
	case at12.RecallAny == 1:
		result.Classification = "partial_evidence_retrieved"
	default:
		result.Classification = "no_evidence_retrieved"
	}
	return result, nil
}

func longMemEvalRetrievalCheckpointPath(root, runID, recordID string) (string, error) {
	if err := validateLongMemEvalRetrievalSegment(recordID, "record ID"); err != nil {
		return "", err
	}
	absoluteRoot, err := filepath.Abs(root)
	if err != nil {
		return "", err
	}
	return filepath.Join(absoluteRoot, "benchmarks", runID, "checkpoints", recordID+".json"), nil
}

func validateLongMemEvalRetrievalSegment(value, label string) error {
	value = strings.TrimSpace(value)
	if value == "" || value == "." || value == ".." || filepath.Base(value) != value {
		return fmt.Errorf("invalid LongMemEval retrieval %s %q", label, value)
	}
	return nil
}

func loadLongMemEvalRetrievalCheckpoint(path string) (LongMemEvalRetrievalRecordResult, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return LongMemEvalRetrievalRecordResult{}, err
	}
	var checkpoint LongMemEvalRetrievalRecordResult
	if err := json.Unmarshal(data, &checkpoint); err != nil {
		return LongMemEvalRetrievalRecordResult{}, fmt.Errorf("decode LongMemEval retrieval checkpoint: %w", err)
	}
	return checkpoint, nil
}

func validateLongMemEvalRetrievalCheckpoint(checkpoint LongMemEvalRetrievalRecordResult, runID, implementationRevision string, execution benchmark.ExecutionManifest, recordID string, conditions []string) error {
	if checkpoint.SchemaVersion != "longmemeval-retrieval-checkpoint/v1" {
		return fmt.Errorf("checkpoint schema_version is invalid for record %q", recordID)
	}
	if checkpoint.RunID != runID {
		return fmt.Errorf("checkpoint run_id is %q, want %q", checkpoint.RunID, runID)
	}
	if checkpoint.ImplementationRevision != implementationRevision {
		return fmt.Errorf("checkpoint implementation_revision is %q, want %q", checkpoint.ImplementationRevision, implementationRevision)
	}
	if checkpoint.DatasetSHA256 != execution.DatasetSHA256 {
		return fmt.Errorf("checkpoint dataset_sha256 is %q, want %q", checkpoint.DatasetSHA256, execution.DatasetSHA256)
	}
	if checkpoint.RecordSetSHA256 != execution.RecordSetSHA256 {
		return fmt.Errorf("checkpoint record_set_sha256 is %q, want %q", checkpoint.RecordSetSHA256, execution.RecordSetSHA256)
	}
	if checkpoint.RecordID != recordID {
		return fmt.Errorf("checkpoint record_id is %q, want %q", checkpoint.RecordID, recordID)
	}
	checkpointConditions := make([]string, 0, len(checkpoint.Conditions))
	for _, condition := range checkpoint.Conditions {
		checkpointConditions = append(checkpointConditions, condition.Condition)
	}
	if checkpoint.Status == "completed" && !slices.Equal(checkpointConditions, conditions) {
		return fmt.Errorf("checkpoint conditions are %v, want %v", checkpointConditions, conditions)
	}
	return nil
}

func finalizeLongMemEvalRetrievalReport(runID, implementationRevision string, qualification benchmark.Qualification, execution benchmark.ExecutionManifest, summary benchmark.LongMemEvalSummary, results []LongMemEvalRetrievalRecordResult) LongMemEvalRetrievalReport {
	sort.Slice(results, func(i, j int) bool { return results[i].RecordID < results[j].RecordID })
	report := LongMemEvalRetrievalReport{
		RunID:                  runID,
		Benchmark:              execution.Benchmark,
		EvaluationTarget:       execution.EvaluationTarget,
		ExecutionScope:         execution.ExecutionScope,
		ClaimScope:             execution.ClaimScope,
		DatasetSHA256:          execution.DatasetSHA256,
		RecordSetSHA256:        execution.RecordSetSHA256,
		ImplementationRevision: implementationRevision,
		SourceSummary:          summary,
		RecordCount:            len(results),
		Conditions:             longMemEvalRetrievalConditions(),
		Results:                results,
		Aggregates:             make(map[string]LongMemEvalRetrievalAggregate),
		QuestionTypeAggregates: make(map[string]map[string]LongMemEvalRetrievalAggregate),
		Artifacts:              make(map[string]string),
		NonClaims:              append([]string(nil), execution.NonClaims...),
	}
	allMetrics := make(map[string]map[int][]benchmark.SessionRetrievalMetric)
	byType := make(map[string]map[string]map[int][]benchmark.SessionRetrievalMetric)
	for _, result := range results {
		report.ImportedMemoryCount += result.ImportedMemoryCount
		if result.Status != "completed" {
			report.Failures = append(report.Failures, LongMemEvalRetrievalFailure{RecordID: result.RecordID, QuestionType: result.QuestionType, Error: result.Error})
			continue
		}
		if result.Abstention {
			continue
		}
		recordComplete := true
		for _, condition := range result.Conditions {
			if condition.Status != "completed" || condition.MetricAt5 == nil || condition.MetricAt10 == nil || condition.MetricAt12 == nil {
				recordComplete = false
				continue
			}
			if allMetrics[condition.Condition] == nil {
				allMetrics[condition.Condition] = make(map[int][]benchmark.SessionRetrievalMetric)
			}
			allMetrics[condition.Condition][5] = append(allMetrics[condition.Condition][5], *condition.MetricAt5)
			allMetrics[condition.Condition][10] = append(allMetrics[condition.Condition][10], *condition.MetricAt10)
			allMetrics[condition.Condition][12] = append(allMetrics[condition.Condition][12], *condition.MetricAt12)
			if byType[result.QuestionType] == nil {
				byType[result.QuestionType] = make(map[string]map[int][]benchmark.SessionRetrievalMetric)
			}
			if byType[result.QuestionType][condition.Condition] == nil {
				byType[result.QuestionType][condition.Condition] = make(map[int][]benchmark.SessionRetrievalMetric)
			}
			byType[result.QuestionType][condition.Condition][5] = append(byType[result.QuestionType][condition.Condition][5], *condition.MetricAt5)
			byType[result.QuestionType][condition.Condition][10] = append(byType[result.QuestionType][condition.Condition][10], *condition.MetricAt10)
			byType[result.QuestionType][condition.Condition][12] = append(byType[result.QuestionType][condition.Condition][12], *condition.MetricAt12)
		}
		if recordComplete && len(result.Conditions) == len(report.Conditions) {
			report.ScoredRecordCount++
		}
	}
	for _, condition := range report.Conditions {
		report.Aggregates[condition] = aggregateLongMemEvalRetrieval(allMetrics[condition])
	}
	for questionType, conditions := range byType {
		report.QuestionTypeAggregates[questionType] = make(map[string]LongMemEvalRetrievalAggregate)
		for _, condition := range report.Conditions {
			report.QuestionTypeAggregates[questionType][condition] = aggregateLongMemEvalRetrieval(conditions[condition])
		}
	}
	_ = qualification
	return report
}

func aggregateLongMemEvalRetrieval(metrics map[int][]benchmark.SessionRetrievalMetric) LongMemEvalRetrievalAggregate {
	return LongMemEvalRetrievalAggregate{
		Count: len(metrics[10]),
		At5:   benchmark.AggregateSessionRetrieval(metrics[5]),
		At10:  benchmark.AggregateSessionRetrieval(metrics[10]),
		At12:  benchmark.AggregateSessionRetrieval(metrics[12]),
	}
}

func writeLongMemEvalRetrievalArtifacts(ctx context.Context, store *artifact.LocalStore, artifactRoot, prefix string, qualification benchmark.Qualification, execution benchmark.ExecutionManifest, report *LongMemEvalRetrievalReport) error {
	sourceURI, err := putJSONArtifact(ctx, store, filepath.ToSlash(filepath.Join(prefix, "source.json")), map[string]any{
		"qualification": qualification,
		"execution":     execution,
		"summary":       report.SourceSummary,
	})
	if err != nil {
		return err
	}
	report.Artifacts["source"] = sourceURI

	var jsonl strings.Builder
	for _, result := range report.Results {
		line, err := json.Marshal(result)
		if err != nil {
			return err
		}
		jsonl.Write(line)
		jsonl.WriteByte('\n')
	}
	resultsURI, err := putTextArtifact(ctx, store, filepath.ToSlash(filepath.Join(prefix, "retrieval-results.jsonl")), jsonl.String())
	if err != nil {
		return err
	}
	report.Artifacts["retrieval_results"] = resultsURI
	scoresURI, err := putJSONArtifact(ctx, store, filepath.ToSlash(filepath.Join(prefix, "scores.json")), map[string]any{
		"run_id":                   report.RunID,
		"evaluation_target":        report.EvaluationTarget,
		"claim_scope":              report.ClaimScope,
		"source_summary":           report.SourceSummary,
		"aggregates":               report.Aggregates,
		"question_type_aggregates": report.QuestionTypeAggregates,
		"results":                  report.Results,
	})
	if err != nil {
		return err
	}
	report.Artifacts["scores"] = scoresURI
	failuresURI, err := putJSONArtifact(ctx, store, filepath.ToSlash(filepath.Join(prefix, "failure-ledger.json")), report.Failures)
	if err != nil {
		return err
	}
	report.Artifacts["failure_ledger"] = failuresURI
	reportURI, err := putTextArtifact(ctx, store, filepath.ToSlash(filepath.Join(prefix, "report.md")), markdownLongMemEvalRetrievalReport(*report))
	if err != nil {
		return err
	}
	report.Artifacts["report"] = reportURI

	finalExecution := execution
	finalExecution.RunID = report.RunID
	finalExecution.ImplementationRev = report.ImplementationRevision
	finalExecution.Artifacts = copyStringMap(report.Artifacts)
	manifestURI, err := localArtifactURI(artifactRoot, filepath.ToSlash(filepath.Join(prefix, "execution-manifest.json")))
	if err != nil {
		return err
	}
	finalExecution.Artifacts["execution_manifest"] = manifestURI
	if err := benchmark.ValidateExecution(qualification, finalExecution); err != nil {
		return err
	}
	manifestURI, err = putJSONArtifact(ctx, store, filepath.ToSlash(filepath.Join(prefix, "execution-manifest.json")), finalExecution)
	if err != nil {
		return err
	}
	report.Artifacts["execution_manifest"] = manifestURI
	_, err = putJSONArtifact(ctx, store, filepath.ToSlash(filepath.Join(prefix, "report.json")), report)
	return err
}

func markdownLongMemEvalRetrievalReport(report LongMemEvalRetrievalReport) string {
	var builder strings.Builder
	fmt.Fprintf(&builder, "# LongMemEval-S Full Retrieval Qualification\n\n")
	fmt.Fprintf(&builder, "- Run ID: `%s`\n", report.RunID)
	fmt.Fprintf(&builder, "- Scope: `%s` / `%s`\n", report.ExecutionScope, report.ClaimScope)
	fmt.Fprintf(&builder, "- Records: `%d` total / `%d` scored\n", report.RecordCount, report.ScoredRecordCount)
	fmt.Fprintf(&builder, "- Imported governed memories: `%d`\n", report.ImportedMemoryCount)
	fmt.Fprintf(&builder, "- Runtime failures: `%d`\n\n", len(report.Failures))
	builder.WriteString("## Aggregate Retrieval\n\n")
	builder.WriteString("| Condition | K | Recall any | Recall all | nDCG | MRR |\n")
	builder.WriteString("|---|---:|---:|---:|---:|---:|\n")
	for _, condition := range report.Conditions {
		aggregate := report.Aggregates[condition]
		for _, row := range []struct {
			k int
			a benchmark.RetrievalAggregate
		}{{5, aggregate.At5}, {10, aggregate.At10}, {12, aggregate.At12}} {
			fmt.Fprintf(&builder, "| `%s` | %d | %.4f | %.4f | %.4f | %.4f |\n", condition, row.k, row.a.MeanRecallAny, row.a.MeanRecallAll, row.a.MeanNDCG, row.a.MeanMRR)
		}
	}
	builder.WriteString("\n## Non-Claims\n\n")
	for _, nonClaim := range report.NonClaims {
		builder.WriteString("- " + nonClaim + "\n")
	}
	return builder.String()
}
