package casebook

func ValidateBenchmarkCoverage(entries []BenchmarkMapEntry) BenchmarkCoverageReport {
	report := BenchmarkCoverageReport{
		Total:  len(entries),
		ByLine: map[string]int{},
	}

	for _, entry := range entries {
		report.ByLine[string(entry.Line)]++
		benchmarkName := string(entry.Benchmark)

		if benchmarkLevelRank(entry.EvaluationLevel) >= benchmarkLevelRank(BenchmarkEvaluationLevelTranslatedTask) {
			report.TranslatedOrBetter++
		} else {
			report.MissingTranslatedTask = append(report.MissingTranslatedTask, benchmarkName)
		}

		if entry.EvaluationLevel == BenchmarkEvaluationLevelExecutableEvaluation {
			report.ExecutableCount++
			if len(entry.CaseIDs) == 0 {
				report.ExecutableWithoutCases = append(report.ExecutableWithoutCases, benchmarkName)
			}
		}
	}

	return report
}

func benchmarkLevelRank(level BenchmarkEvaluationLevel) int {
	switch level {
	case BenchmarkEvaluationLevelExecutableEvaluation:
		return 3
	case BenchmarkEvaluationLevelTranslatedTask:
		return 2
	case BenchmarkEvaluationLevelReference:
		return 1
	default:
		return 0
	}
}
