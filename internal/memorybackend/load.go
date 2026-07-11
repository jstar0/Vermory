package memorybackend

import (
	"context"
	"fmt"
	"sync"
	"time"
)

type LoadOptions struct {
	Records     int
	Queries     int
	Concurrency int
	ScopeSuffix string
}

type LoadReport struct {
	Backend          string        `json:"backend"`
	Pass             bool          `json:"pass"`
	Scope            Scope         `json:"scope"`
	RecordsRequested int           `json:"records_requested"`
	ResetDuration    time.Duration `json:"reset_duration"`
	WritesSucceeded  int           `json:"writes_succeeded"`
	WriteErrors      []string      `json:"write_errors,omitempty"`
	IngestDuration   time.Duration `json:"ingest_duration"`
	Queries          int           `json:"queries"`
	QueryHits        int           `json:"query_hits"`
	Recall           float64       `json:"recall"`
	SearchP50        time.Duration `json:"search_p50"`
	SearchP95        time.Duration `json:"search_p95"`
	Stats            Stats         `json:"stats"`
	Duration         time.Duration `json:"duration"`
}

func RunLoadSuite(ctx context.Context, backend Backend, options LoadOptions) LoadReport {
	started := time.Now()
	if options.Records <= 0 {
		options.Records = 200
	}
	if options.Queries <= 0 {
		options.Queries = 20
	}
	if options.Concurrency <= 0 {
		options.Concurrency = 4
	}
	if options.ScopeSuffix == "" {
		options.ScopeSuffix = "default"
	}
	scope := Scope{TenantID: "bakeoff-load", ContinuityID: "workspace:load:" + options.ScopeSuffix, ContinuityLine: "workspace"}
	report := LoadReport{
		Backend: backend.Name(), Pass: true, Scope: scope,
		RecordsRequested: options.Records, Queries: options.Queries,
	}
	resetStarted := time.Now()
	resetErr := backend.ResetScope(ctx, scope)
	report.ResetDuration = time.Since(resetStarted)
	if resetErr != nil {
		report.Pass = false
		report.WriteErrors = append(report.WriteErrors, resetErr.Error())
		report.Duration = time.Since(started)
		return report
	}

	jobs := make(chan Record)
	errors := make(chan error, options.Records)
	var workers sync.WaitGroup
	ingestStarted := time.Now()
	for range options.Concurrency {
		workers.Add(1)
		go func() {
			defer workers.Done()
			for record := range jobs {
				if err := backend.Put(ctx, record); err != nil {
					errors <- err
				}
			}
		}()
	}
	for index := range options.Records {
		jobs <- loadRecord(scope, index)
	}
	close(jobs)
	workers.Wait()
	close(errors)
	report.IngestDuration = time.Since(ingestStarted)
	for err := range errors {
		report.WriteErrors = append(report.WriteErrors, err.Error())
	}
	report.WritesSucceeded = options.Records - len(report.WriteErrors)
	if len(report.WriteErrors) > 0 {
		report.Pass = false
	}

	searchDurations := make([]time.Duration, 0, options.Queries)
	for queryIndex := range options.Queries {
		target := loadQueryTarget(options.Records, options.Queries, queryIndex)
		started := time.Now()
		results, err := backend.Search(ctx, Query{
			Scope: scope,
			Text:  fmt.Sprintf("模块 module-%04d 的 production feature flag 和 owner 是什么？", target),
			Limit: 10,
		})
		searchDurations = append(searchDurations, time.Since(started))
		if err != nil {
			report.Pass = false
			continue
		}
		expectedFlag := fmt.Sprintf("ff_%04d", target)
		expectedOwner := fmt.Sprintf("#team-%02d", target%23)
		if requireContent(results, expectedFlag, expectedOwner) == nil {
			report.QueryHits++
		} else {
			report.Pass = false
		}
	}
	if report.Queries > 0 {
		report.Recall = float64(report.QueryHits) / float64(report.Queries)
	}
	for i := 1; i < len(searchDurations); i++ {
		for j := i; j > 0 && searchDurations[j] < searchDurations[j-1]; j-- {
			searchDurations[j], searchDurations[j-1] = searchDurations[j-1], searchDurations[j]
		}
	}
	report.SearchP50 = percentileDuration(searchDurations, 0.50)
	report.SearchP95 = percentileDuration(searchDurations, 0.95)
	stats, err := backend.Stats(ctx)
	if err != nil {
		report.Pass = false
	} else {
		report.Stats = stats
	}
	report.Duration = time.Since(started)
	return report
}

func loadRecord(scope Scope, index int) Record {
	return Record{
		ID: fmt.Sprintf("module-%04d", index), Scope: scope,
		SourceID: "repo:services-catalog", SourceVersion: 1, Status: "active",
		Content: fmt.Sprintf(
			"模块 module-%04d 的 production feature flag 是 ff_%04d，owner 是 #team-%02d，请求超时为 %dms。",
			index, index, index%23, 500+(index%17)*100,
		),
	}
}

func loadQueryTarget(records, queries, queryIndex int) int {
	if queries <= 1 {
		return records / 2
	}
	return queryIndex * (records - 1) / (queries - 1)
}
