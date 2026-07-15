package runtime

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
	"time"
)

type scopedHNSWCase struct {
	Version                string                      `json:"version"`
	ID                     string                      `json:"id"`
	ProfileName            string                      `json:"profile_name"`
	SourceProfile          string                      `json:"source_profile"`
	TenantCount            int                         `json:"tenant_count"`
	ContinuitiesPerTenant  int                         `json:"continuities_per_tenant"`
	RecordsPerContinuity   int                         `json:"records_per_continuity"`
	ActiveMemoryCount      int                         `json:"active_memory_count"`
	QueryClientCount       int                         `json:"query_client_count"`
	VectorQueriesPerClient int                         `json:"vector_queries_per_client"`
	RequestedVectorQueries int                         `json:"requested_vector_queries"`
	SnapshotPageSize       int                         `json:"snapshot_page_size"`
	PoolMaxConnections     int                         `json:"pool_max_connections"`
	ProfileID              string                      `json:"profile_id"`
	PGVectorMinimum        string                      `json:"pgvector_minimum"`
	Baseline               scopedHNSWBaseline          `json:"baseline"`
	CalibratedLimits       serverScaleCalibratedLimits `json:"calibrated_limits"`
	HardGates              []string                    `json:"hard_gates"`
}

type scopedHNSWBaseline struct {
	ImplementationRevision string `json:"implementation_revision"`
	EffectiveVectorQueries int    `json:"effective_vector_queries"`
	ControlledFallbacks    int    `json:"controlled_lexical_fallbacks"`
	CorrectTargetResults   int    `json:"correct_target_results"`
	CrossScopeLeaks        int    `json:"cross_scope_leaks"`
}

type scopedHNSWResult struct {
	QuerySamples    int
	EffectiveVector int
	Degraded        int
	P50             time.Duration
	P95             time.Duration
	P99             time.Duration
	VectorDuration  time.Duration
	DatabaseSize    int64
}

func TestScopedHNSWCaseIsFrozen(t *testing.T) {
	manifest := loadScopedHNSWCase(t)
	if manifest.Version != "1" || manifest.ID != "W13-scoped-hnsw-recall-profile" ||
		manifest.ProfileName != "scoped-hnsw-recall-v1" ||
		manifest.SourceProfile != "W12-server-qualification-scale-profile" {
		t.Fatalf("unexpected W13 identity: %#v", manifest)
	}
	if manifest.TenantCount*manifest.ContinuitiesPerTenant*manifest.RecordsPerContinuity != manifest.ActiveMemoryCount {
		t.Fatalf("W13 active-memory arithmetic drifted: %#v", manifest)
	}
	if manifest.QueryClientCount*manifest.VectorQueriesPerClient != manifest.RequestedVectorQueries ||
		manifest.RequestedVectorQueries != 550 || manifest.ProfileID != ProductionRetrievalProfileID {
		t.Fatalf("W13 query/profile contract drifted: %#v", manifest)
	}
	if manifest.PGVectorMinimum != "0.8.0" || manifest.PoolMaxConnections != 64 || len(manifest.HardGates) != 8 {
		t.Fatalf("W13 hard-gate contract drifted: %#v", manifest)
	}
	if manifest.Baseline.EffectiveVectorQueries != 138 || manifest.Baseline.ControlledFallbacks != 412 ||
		manifest.Baseline.CorrectTargetResults != 550 || manifest.Baseline.CrossScopeLeaks != 0 {
		t.Fatalf("W13 W12 baseline drifted: %#v", manifest.Baseline)
	}
}

func TestScopedHNSWMediumProfile(t *testing.T) {
	if os.Getenv("VERMORY_SCOPED_HNSW_MEDIUM_PROFILE") != "1" {
		t.Skip("VERMORY_SCOPED_HNSW_MEDIUM_PROFILE=1 is required")
	}
	manifest := serverScaleCase{
		TenantCount: 2, ContinuitiesPerTenant: 25, RecordsPerContinuity: 200,
		ActiveMemoryCount: 10000, QueryClientCount: 20, QueriesPerClient: 10,
		SnapshotPageSize: 500, PoolMaxConnections: 32, ProfileID: ProductionRetrievalProfileID,
		CalibratedLimits: serverScaleCalibratedLimits{
			VectorSnapshotSeconds: 300, QueryP95MS: 2000, QueryP99MS: 5000, DatabaseSizeGiB: 5,
		},
	}
	result := runScopedHNSWProfile(t, manifest, true)
	assertScopedHNSWResult(t, result, manifest.QueryClientCount*manifest.QueriesPerClient, manifest.CalibratedLimits)
	t.Logf("scoped HNSW medium result=%s", marshalScopedHNSWResult(result))
}

func TestScopedHNSWRecallProfile(t *testing.T) {
	if os.Getenv("VERMORY_SCOPED_HNSW_PROFILE") != "1" {
		t.Skip("VERMORY_SCOPED_HNSW_PROFILE=1 is required")
	}
	frozen := loadScopedHNSWCase(t)
	manifest := serverScaleCase{
		TenantCount: frozen.TenantCount, ContinuitiesPerTenant: frozen.ContinuitiesPerTenant,
		RecordsPerContinuity: frozen.RecordsPerContinuity, ActiveMemoryCount: frozen.ActiveMemoryCount,
		QueryClientCount: frozen.QueryClientCount, QueriesPerClient: frozen.VectorQueriesPerClient,
		SnapshotPageSize: frozen.SnapshotPageSize, PoolMaxConnections: frozen.PoolMaxConnections,
		ProfileID: frozen.ProfileID, CalibratedLimits: frozen.CalibratedLimits,
	}
	result := runScopedHNSWProfile(t, manifest, false)
	assertScopedHNSWResult(t, result, frozen.RequestedVectorQueries, frozen.CalibratedLimits)
	t.Logf("scoped HNSW evidence=%s", marshalScopedHNSWResult(result))
}

func runScopedHNSWProfile(t *testing.T, manifest serverScaleCase, forceHNSW bool) scopedHNSWResult {
	t.Helper()
	cluster := startDisposablePostgres18(t)
	defer cluster.stop(t, "fast")

	ctx := context.Background()
	store, err := OpenStore(ctx, cluster.databaseURL+"&pool_max_conns="+itoa(manifest.PoolMaxConnections))
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()
	if err := store.Migrate(ctx); err != nil {
		t.Fatal(err)
	}
	assertPGVectorMinimum(t, store, "0.8.0")

	dataset := prepareServerScaleContinuities(t, store, manifest)
	seedServerScaleAuthority(t, store, manifest, dataset)
	if rows, err := store.RebuildAllProjections(ctx); err != nil {
		t.Fatal(err)
	} else if rows != int64(manifest.ActiveMemoryCount) {
		t.Fatalf("lexical rows=%d want %d", rows, manifest.ActiveMemoryCount)
	}
	dataset.Records, dataset.RecordsByID = loadCurrentServerScaleRecords(t, store, manifest)

	embedder := &serverScaleEmbedder{}
	vectorStarted := time.Now()
	results := rebuildServerScaleVectors(t, store, manifest, dataset, productionRetrievalProfile(t), embedder)
	vectorDuration := time.Since(vectorStarted)
	if embedder.calls.Load() != int64(manifest.ActiveMemoryCount) {
		t.Fatalf("snapshot embedding calls=%d want %d", embedder.calls.Load(), manifest.ActiveMemoryCount)
	}
	for _, result := range results {
		if result.Projected != result.Scanned || result.SkippedChanged != 0 || result.Lag != 0 {
			t.Fatalf("unexpected snapshot result: %#v", result)
		}
	}
	if _, err := store.pool.Exec(ctx, `ANALYZE memory_vector_documents`); err != nil {
		t.Fatal(err)
	}
	if forceHNSW {
		if _, err := store.pool.Exec(ctx, `DROP INDEX memory_vector_documents_scope_idx`); err != nil {
			t.Fatal(err)
		}
		if _, err := store.pool.Exec(ctx, `ALTER TABLE memory_vector_documents DROP CONSTRAINT memory_vector_documents_pkey`); err != nil {
			t.Fatal(err)
		}
		if _, err := store.pool.Exec(ctx, `ALTER DATABASE vermory_outbox_fault SET enable_seqscan = off`); err != nil {
			t.Fatal(err)
		}
		store.pool.Reset()
	}
	coordinators := newServerScaleCoordinators(t, store, dataset, productionRetrievalProfile(t), embedder)
	measurements := make(chan serverScaleQueryMeasurement, manifest.QueryClientCount*manifest.QueriesPerClient)
	errorsCh := make(chan error, manifest.QueryClientCount*manifest.QueriesPerClient)
	runServerScaleQueryPhase(
		ctx, manifest, dataset, coordinators, 0, manifest.QueriesPerClient, measurements, errorsCh,
	)
	close(measurements)
	close(errorsCh)
	for queryErr := range errorsCh {
		if queryErr != nil {
			t.Fatal(queryErr)
		}
	}
	latencies, degraded, effectiveVector := collectServerScaleMeasurements(measurements)
	assertServerScaleScopes(t, store, manifest, dataset, coordinators)
	assertAllServerScaleTenantsCurrent(t, store, manifest, dataset)

	var databaseSize int64
	if err := store.pool.QueryRow(ctx, `SELECT pg_database_size(current_database())`).Scan(&databaseSize); err != nil {
		t.Fatal(err)
	}
	return scopedHNSWResult{
		QuerySamples: len(latencies), EffectiveVector: effectiveVector, Degraded: degraded,
		P50: percentileDuration(latencies, 50), P95: percentileDuration(latencies, 95),
		P99: percentileDuration(latencies, 99), VectorDuration: vectorDuration, DatabaseSize: databaseSize,
	}
}

func assertScopedHNSWResult(t *testing.T, result scopedHNSWResult, expected int, limits serverScaleCalibratedLimits) {
	t.Helper()
	if result.QuerySamples != expected {
		t.Fatalf("query samples=%d want %d", result.QuerySamples, expected)
	}
	if result.EffectiveVector != expected || result.Degraded != 0 {
		t.Fatalf("scoped HNSW effective/degraded=%d/%d want %d/0", result.EffectiveVector, result.Degraded, expected)
	}
	if result.P95 > time.Duration(limits.QueryP95MS)*time.Millisecond ||
		result.P99 > time.Duration(limits.QueryP99MS)*time.Millisecond {
		t.Fatalf("query latency exceeded profile: p95=%s p99=%s", result.P95, result.P99)
	}
	if result.VectorDuration > time.Duration(limits.VectorSnapshotSeconds)*time.Second {
		t.Fatalf("vector snapshot=%s exceeds %ds", result.VectorDuration, limits.VectorSnapshotSeconds)
	}
	if result.DatabaseSize > int64(limits.DatabaseSizeGiB)<<30 {
		t.Fatalf("database size=%d exceeds %d GiB", result.DatabaseSize, limits.DatabaseSizeGiB)
	}
}

func assertScopedHNSWIndexPlan(
	t *testing.T,
	store *Store,
	manifest serverScaleCase,
	dataset serverScaleDataset,
	embedder *serverScaleEmbedder,
) {
	t.Helper()
	tenantID := dataset.Tenants[0]
	continuityID := dataset.Continuities[tenantID][0]
	marker := serverScaleMarker(0, 0, 1)
	queryVector, err := embedder.Embed(context.Background(), marker)
	if err != nil {
		t.Fatal(err)
	}
	rows, err := store.pool.Query(context.Background(), `
EXPLAIN (COSTS OFF)
WITH candidates AS (
  SELECT document.memory_id, document.content_sha256,
         document.embedding <=> $4::vector AS distance,
         CASE origin.observation_kind
           WHEN 'user_correction' THEN 4
           WHEN 'user_confirmation' THEN 4
           WHEN 'source_update' THEN 3
           WHEN 'bridge_promote' THEN 2
           ELSE 1
         END AS authority_rank
  FROM memory_vector_documents document
  JOIN governed_memories memory
    ON memory.tenant_id = $2 AND memory.id = document.memory_id
  JOIN observations origin
    ON origin.tenant_id = $2 AND origin.id = memory.origin_observation_id
  WHERE document.profile_id = $1
    AND document.tenant_id = $2
    AND document.continuity_id = ANY($3::uuid[])
  ORDER BY document.embedding <=> $4::vector, document.memory_id
  LIMIT $5
)
SELECT memory.id::text, memory.content
FROM candidates candidate
JOIN governed_memories memory
  ON memory.tenant_id = $2 AND memory.id = candidate.memory_id
WHERE memory.continuity_id = ANY($3::uuid[])
  AND memory.memory_kind = 'fact'
  AND memory.lifecycle_status = 'active'
  AND memory.content <> '[redacted]'
  AND encode(digest(convert_to(memory.content, 'UTF8'), 'sha256'), 'hex') = candidate.content_sha256
ORDER BY candidate.distance, candidate.authority_rank DESC, memory.id
LIMIT $6`, manifest.ProfileID, tenantID, []string{continuityID}, retrievalVectorLiteral(queryVector), 20, 1)
	if err != nil {
		t.Fatal(err)
	}
	defer rows.Close()
	var plan strings.Builder
	for rows.Next() {
		var line string
		if err := rows.Scan(&line); err != nil {
			t.Fatal(err)
		}
		plan.WriteString(line)
		plan.WriteByte('\n')
	}
	if err := rows.Err(); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(plan.String(), "memory_vector_documents_embedding_hnsw_idx") {
		t.Fatalf("fixture did not use the HNSW index:\n%s", plan.String())
	}
}

func assertPGVectorMinimum(t *testing.T, store *Store, minimum string) {
	t.Helper()
	var version string
	if err := store.pool.QueryRow(context.Background(), `SELECT extversion FROM pg_extension WHERE extname = 'vector'`).Scan(&version); err != nil {
		t.Fatal(err)
	}
	if version < minimum {
		t.Fatalf("pgvector version=%s want >= %s", version, minimum)
	}
}

func marshalScopedHNSWResult(result scopedHNSWResult) string {
	payload, _ := json.Marshal(map[string]any{
		"query_samples": result.QuerySamples, "effective_vector_queries": result.EffectiveVector,
		"degraded_queries": result.Degraded, "query_p50_ms": result.P50.Microseconds() / 1000.0,
		"query_p95_ms": result.P95.Microseconds() / 1000.0, "query_p99_ms": result.P99.Microseconds() / 1000.0,
		"vector_snapshot_ms": result.VectorDuration.Milliseconds(), "database_size_bytes": result.DatabaseSize,
	})
	return string(payload)
}

func loadScopedHNSWCase(t *testing.T) scopedHNSWCase {
	t.Helper()
	root, err := filepath.Abs(filepath.Join("..", ".."))
	if err != nil {
		t.Fatal(err)
	}
	payload, err := os.ReadFile(filepath.Join(root, "runtime", "cases", "W13-scoped-hnsw-recall-profile", "case.json"))
	if err != nil {
		t.Fatal(err)
	}
	var manifest scopedHNSWCase
	decoder := json.NewDecoder(strings.NewReader(string(payload)))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&manifest); err != nil {
		t.Fatal(err)
	}
	return manifest
}

func itoa(value int) string {
	return strconv.Itoa(value)
}
