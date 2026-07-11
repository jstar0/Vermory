package memorybackend

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"sort"
	"strings"
	"time"
)

type QualitySuite struct {
	Version   string            `json:"version"`
	Scenarios []QualityScenario `json:"scenarios"`
}

type QualityScenario struct {
	ID          string        `json:"id"`
	Description string        `json:"description"`
	Steps       []QualityStep `json:"steps"`
}

type QualityStep struct {
	Op       string        `json:"op"`
	Scope    *Scope        `json:"scope,omitempty"`
	Record   *Record       `json:"record,omitempty"`
	RecordID string        `json:"record_id,omitempty"`
	Records  []Record      `json:"records,omitempty"`
	Query    *QualityQuery `json:"query,omitempty"`
}

type QualityQuery struct {
	Scope          Scope    `json:"scope"`
	Text           string   `json:"text"`
	Limit          int      `json:"limit"`
	IncludeHistory bool     `json:"include_history,omitempty"`
	Required       []string `json:"required,omitempty"`
	Forbidden      []string `json:"forbidden,omitempty"`
}

type QualityQueryResult struct {
	Text           string        `json:"text"`
	Pass           bool          `json:"pass"`
	Duration       time.Duration `json:"duration"`
	ResultCount    int           `json:"result_count"`
	Missing        []string      `json:"missing,omitempty"`
	ForbiddenFound []string      `json:"forbidden_found,omitempty"`
}

type QualityScenarioResult struct {
	ID          string               `json:"id"`
	Description string               `json:"description"`
	Pass        bool                 `json:"pass"`
	Error       string               `json:"error,omitempty"`
	Queries     []QualityQueryResult `json:"queries,omitempty"`
}

type QualityMetrics struct {
	Assertions       int           `json:"assertions"`
	PassedAssertions int           `json:"passed_assertions"`
	RequiredFacts    int           `json:"required_facts"`
	RecalledFacts    int           `json:"recalled_facts"`
	Recall           float64       `json:"recall"`
	ForbiddenLeakage int           `json:"forbidden_leakage"`
	SearchP50        time.Duration `json:"search_p50"`
	SearchP95        time.Duration `json:"search_p95"`
}

type QualityReport struct {
	Backend   string                  `json:"backend"`
	Pass      bool                    `json:"pass"`
	StartedAt time.Time               `json:"started_at"`
	Duration  time.Duration           `json:"duration"`
	Metrics   QualityMetrics          `json:"metrics"`
	Scenarios []QualityScenarioResult `json:"scenarios"`
}

func LoadQualitySuite(path string) (QualitySuite, error) {
	payload, err := os.ReadFile(path)
	if err != nil {
		return QualitySuite{}, fmt.Errorf("read backend casebook: %w", err)
	}
	var suite QualitySuite
	if err := json.Unmarshal(payload, &suite); err != nil {
		return QualitySuite{}, fmt.Errorf("decode backend casebook: %w", err)
	}
	if suite.Version == "" || len(suite.Scenarios) == 0 {
		return QualitySuite{}, fmt.Errorf("backend casebook requires version and scenarios")
	}
	for index, scenario := range suite.Scenarios {
		if scenario.ID == "" || len(scenario.Steps) == 0 {
			return QualitySuite{}, fmt.Errorf("scenario %d requires id and steps", index)
		}
	}
	return suite, nil
}

func RunQualitySuite(ctx context.Context, backend Backend, suite QualitySuite) QualityReport {
	started := time.Now().UTC()
	report := QualityReport{Backend: backend.Name(), Pass: true, StartedAt: started}
	if err := backend.Health(ctx); err != nil {
		report.Pass = false
		report.Scenarios = append(report.Scenarios, QualityScenarioResult{ID: "health", Pass: false, Error: err.Error()})
		report.Duration = time.Since(started)
		return report
	}
	var searchDurations []time.Duration
	for _, scenario := range suite.Scenarios {
		result := QualityScenarioResult{ID: scenario.ID, Description: scenario.Description, Pass: true}
		for _, step := range scenario.Steps {
			if err := runQualityStep(ctx, backend, step, &result, &report.Metrics, &searchDurations); err != nil {
				result.Pass = false
				result.Error = err.Error()
				break
			}
		}
		if !result.Pass {
			report.Pass = false
		}
		report.Scenarios = append(report.Scenarios, result)
	}
	if report.Metrics.RequiredFacts == 0 {
		report.Metrics.Recall = 1
	} else {
		report.Metrics.Recall = float64(report.Metrics.RecalledFacts) / float64(report.Metrics.RequiredFacts)
	}
	sort.Slice(searchDurations, func(i, j int) bool { return searchDurations[i] < searchDurations[j] })
	report.Metrics.SearchP50 = percentileDuration(searchDurations, 0.50)
	report.Metrics.SearchP95 = percentileDuration(searchDurations, 0.95)
	report.Duration = time.Since(started)
	return report
}

func runQualityStep(
	ctx context.Context,
	backend Backend,
	step QualityStep,
	scenario *QualityScenarioResult,
	metrics *QualityMetrics,
	searchDurations *[]time.Duration,
) error {
	switch step.Op {
	case "reset":
		if step.Scope == nil {
			return fmt.Errorf("reset step requires scope")
		}
		return backend.ResetScope(ctx, *step.Scope)
	case "put":
		if step.Record == nil {
			return fmt.Errorf("put step requires record")
		}
		return backend.Put(ctx, *step.Record)
	case "update":
		if step.Record == nil {
			return fmt.Errorf("update step requires record")
		}
		return backend.Update(ctx, *step.Record)
	case "delete":
		if step.Scope == nil || step.RecordID == "" {
			return fmt.Errorf("delete step requires scope and record_id")
		}
		return backend.Delete(ctx, *step.Scope, step.RecordID)
	case "rebuild":
		if step.Scope == nil {
			return fmt.Errorf("rebuild step requires scope")
		}
		return backend.RebuildScope(ctx, *step.Scope, step.Records)
	case "search":
		if step.Query == nil {
			return fmt.Errorf("search step requires query")
		}
		started := time.Now()
		results, err := backend.Search(ctx, Query{
			Scope: step.Query.Scope, Text: step.Query.Text, Limit: step.Query.Limit,
			IncludeHistory: step.Query.IncludeHistory,
		})
		duration := time.Since(started)
		*searchDurations = append(*searchDurations, duration)
		if err != nil {
			return err
		}
		queryResult := evaluateQualityQuery(*step.Query, results)
		queryResult.Duration = duration
		scenario.Queries = append(scenario.Queries, queryResult)
		metrics.Assertions += len(step.Query.Required) + len(step.Query.Forbidden)
		metrics.PassedAssertions += len(step.Query.Required) - len(queryResult.Missing)
		metrics.PassedAssertions += len(step.Query.Forbidden) - len(queryResult.ForbiddenFound)
		metrics.RequiredFacts += len(step.Query.Required)
		metrics.RecalledFacts += len(step.Query.Required) - len(queryResult.Missing)
		metrics.ForbiddenLeakage += len(queryResult.ForbiddenFound)
		if !queryResult.Pass {
			scenario.Pass = false
		}
		return nil
	default:
		return fmt.Errorf("unsupported quality step %q", step.Op)
	}
}

func evaluateQualityQuery(query QualityQuery, results []Result) QualityQueryResult {
	joined := strings.ToLower(joinContent(results))
	result := QualityQueryResult{Text: query.Text, Pass: true, ResultCount: len(results)}
	for _, phrase := range query.Required {
		if !strings.Contains(joined, strings.ToLower(phrase)) {
			result.Missing = append(result.Missing, phrase)
			result.Pass = false
		}
	}
	for _, phrase := range query.Forbidden {
		if strings.Contains(joined, strings.ToLower(phrase)) {
			result.ForbiddenFound = append(result.ForbiddenFound, phrase)
			result.Pass = false
		}
	}
	return result
}

func percentileDuration(values []time.Duration, percentile float64) time.Duration {
	if len(values) == 0 {
		return 0
	}
	index := int(float64(len(values)-1) * percentile)
	return values[index]
}
