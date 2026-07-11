package app

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"path/filepath"
	"sort"
	"strings"
	"sync"

	"vermory/internal/artifact"
	"vermory/internal/eval"
	"vermory/internal/packet"
	"vermory/internal/provider"
	"vermory/internal/runner"
)

type EvalMatrixOptions struct {
	ArtifactRoot string
	Provider     string
	BaseURL      string
	APIKeyEnv    string
	RunID        string
	Models       []string
	MaxTokens    int
	Concurrency  int
}

type MatrixTaskReport struct {
	TaskID       string `json:"task_id"`
	ReportURI    string `json:"report_uri"`
	ProviderName string `json:"provider_name"`
	Model        string `json:"model"`
	Status       string `json:"status"`
	Error        string `json:"error,omitempty"`
}

type MatrixModelReport struct {
	Model string             `json:"model"`
	Tasks []MatrixTaskReport `json:"tasks"`
}

type MatrixReport struct {
	RunID        string              `json:"run_id"`
	ProviderMode string              `json:"provider_mode"`
	ProviderName string              `json:"provider_name"`
	Models       []MatrixModelReport `json:"models"`
	ReportURI    string              `json:"report_uri,omitempty"`
}

func EvalMatrix(ctx context.Context, opts EvalMatrixOptions) (MatrixReport, error) {
	return evalMatrixWithFactory(ctx, opts, func(model string) (provider.Provider, string, string, string, error) {
		return buildProvider(EvalSelfCaseOptions{
			ArtifactRoot: opts.ArtifactRoot,
			Provider:     opts.Provider,
			BaseURL:      opts.BaseURL,
			APIKeyEnv:    opts.APIKeyEnv,
			Model:        model,
			MaxTokens:    opts.MaxTokens,
		})
	})
}

func evalMatrixWithFactory(ctx context.Context, opts EvalMatrixOptions, providerFactory func(model string) (provider.Provider, string, string, string, error)) (MatrixReport, error) {
	models := compactModels(opts.Models)
	if len(models) == 0 {
		return MatrixReport{}, errors.New("eval-matrix requires at least one model")
	}
	if strings.TrimSpace(opts.ArtifactRoot) == "" {
		opts.ArtifactRoot = "./artifacts"
	}
	runID := chooseRunID(opts.RunID, "matrix")

	root, err := projectRoot()
	if err != nil {
		return MatrixReport{}, err
	}
	caseDir := filepath.Join(root, "casebook", "cases", selfCaseID)
	fixtureClaims, err := readFixtureClaims(filepath.Join(caseDir, "claims.json"))
	if err != nil {
		return MatrixReport{}, err
	}
	tasks, err := readTasks(filepath.Join(caseDir, "tasks.json"))
	if err != nil {
		return MatrixReport{}, err
	}
	if len(tasks) == 0 {
		return MatrixReport{}, errors.New("self-case has no WCEF tasks")
	}

	packetClaims := preparePacketClaims(toConfirmedDomainClaims(fixtureClaims))
	contextPacket := packet.Build(packet.ProfileCodingAgent, "ContextMesh", "AI 工具直连评测", packetClaims)
	plain := plainSummary(packetClaims)
	stale := defaultStaleContext()
	if opts.MaxTokens <= 0 {
		opts.MaxTokens = 256
	}
	if opts.Concurrency <= 0 {
		opts.Concurrency = 4
	}

	report := MatrixReport{
		RunID:  runID,
		Models: make([]MatrixModelReport, 0, len(models)),
	}
	store := artifact.NewLocalStore(opts.ArtifactRoot)
	modelReports := make(map[string]*MatrixModelReport, len(models))
	for _, model := range models {
		_, providerMode, providerName, resolvedModel, err := providerFactory(model)
		if err != nil {
			return MatrixReport{}, err
		}
		report.ProviderMode = providerMode
		report.ProviderName = providerName
		modelReports[model] = &MatrixModelReport{
			Model: resolvedModel,
			Tasks: make([]MatrixTaskReport, 0, len(tasks)),
		}
	}

	type job struct {
		model string
		task  eval.Task
	}
	type result struct {
		model string
		task  MatrixTaskReport
	}

	jobs := make(chan job)
	results := make(chan result, len(models)*len(tasks))
	errs := make(chan error, 1)
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	var wg sync.WaitGroup
	for range opts.Concurrency {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for job := range jobs {
				llm, providerMode, providerName, resolvedModel, err := providerFactory(job.model)
				if err != nil {
					select {
					case errs <- err:
					default:
					}
					cancel()
					return
				}
				taskRunID := strings.Join([]string{runID, sanitizePathSegment(job.model), job.task.ID}, "__")
				evalReport, err := runner.RunEvaluation(ctx, llm, runner.EvaluationOptions{
					RunID:          taskRunID,
					ProviderMode:   providerMode,
					ProviderName:   providerName,
					Model:          resolvedModel,
					SystemPrompt:   "你是一个真实模型评测对象。请只根据用户任务和提供的上下文作答，不要编造未给出的项目事实。",
					Task:           job.task,
					StaleContext:   stale,
					PlainSummary:   plain,
					ContextPacket:  contextPacket.Body,
					MaxTokens:      opts.MaxTokens,
					ArtifactPrefix: "matrix-runs",
					ArtifactStore:  store,
				})
				taskReport := MatrixTaskReport{
					TaskID:       job.task.ID,
					ProviderName: providerName,
					Model:        resolvedModel,
					Status:       "ok",
				}
				if err != nil {
					taskReport.Status = "error"
					taskReport.Error = err.Error()
				} else {
					taskReport.ReportURI = evalReport.ReportURI
				}
				select {
				case results <- result{model: job.model, task: taskReport}:
				case <-ctx.Done():
					return
				}
			}
		}()
	}

	go func() {
		defer close(jobs)
		for _, model := range models {
			for _, task := range tasks {
				select {
				case jobs <- job{model: model, task: task}:
				case <-ctx.Done():
					return
				}
			}
		}
	}()

	expected := len(models) * len(tasks)
	for i := 0; i < expected; i++ {
		select {
		case err := <-errs:
			if err != nil {
				return MatrixReport{}, err
			}
		case result := <-results:
			modelReports[result.model].Tasks = append(modelReports[result.model].Tasks, result.task)
		}
	}
	wg.Wait()

	for _, model := range models {
		modelReport := modelReports[model]
		sort.Slice(modelReport.Tasks, func(i, j int) bool {
			return modelReport.Tasks[i].TaskID < modelReport.Tasks[j].TaskID
		})
		report.Models = append(report.Models, *modelReport)
	}

	reportBytes, err := json.MarshalIndent(report, "", "  ")
	if err != nil {
		return MatrixReport{}, err
	}
	if _, err := store.Put(ctx, strings.Join([]string{"matrix-runs", runID, "report.json"}, "/"), reportBytes); err != nil {
		return MatrixReport{}, err
	}
	md := markdownMatrixReport(report, tasks)
	mdArtifact, err := store.Put(ctx, strings.Join([]string{"matrix-runs", runID, "report.md"}, "/"), []byte(md))
	if err != nil {
		return MatrixReport{}, err
	}
	report.ReportURI = mdArtifact.URI
	return report, nil
}

func markdownMatrixReport(report MatrixReport, tasks []eval.Task) string {
	var b strings.Builder
	b.WriteString("# ContextMesh Matrix Report\n\n")
	b.WriteString(fmt.Sprintf("- Run ID: `%s`\n", report.RunID))
	b.WriteString(fmt.Sprintf("- Provider mode: `%s`\n", report.ProviderMode))
	b.WriteString(fmt.Sprintf("- Provider: `%s`\n\n", report.ProviderName))
	b.WriteString("This matrix records model-by-task evaluation runs. Each cell points to a full four-baseline report artifact.\n\n")
	b.WriteString("| Model | Task Count | Task IDs |\n")
	b.WriteString("| --- | ---: | --- |\n")
	for _, model := range report.Models {
		ids := make([]string, 0, len(model.Tasks))
		for _, task := range model.Tasks {
			ids = append(ids, task.TaskID)
		}
		b.WriteString(fmt.Sprintf("| `%s` | %d | %s |\n", model.Model, len(model.Tasks), strings.Join(ids, ", ")))
	}
	b.WriteString("\n## Task Catalog\n\n")
	for _, task := range tasks {
		b.WriteString(fmt.Sprintf("- `%s`: %s\n", task.ID, task.Prompt))
	}
	return b.String()
}

func sanitizePathSegment(input string) string {
	replacer := strings.NewReplacer("/", "_", " ", "_", ":", "_")
	return replacer.Replace(input)
}
