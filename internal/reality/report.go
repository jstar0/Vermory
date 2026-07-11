package reality

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

type ValidationReport struct {
	RunID      string         `json:"run_id"`
	CaseRoot   string         `json:"case_root"`
	Pass       bool           `json:"pass"`
	Cases      int            `json:"cases"`
	ByLine     map[string]int `json:"by_line"`
	ByEvidence map[string]int `json:"by_evidence"`
	Results    []CaseResult   `json:"results"`
}

type CaseResult struct {
	CaseID        string           `json:"case_id"`
	Pass          bool             `json:"pass"`
	EvidenceLevel EvidenceLevel    `json:"evidence_level,omitempty"`
	Lines         []ContinuityLine `json:"continuity_lines,omitempty"`
	Pressures     []string         `json:"pressures,omitempty"`
	LockSHA256    string           `json:"lock_sha256,omitempty"`
	Violations    []Violation      `json:"violations,omitempty"`
}

type ValidationArtifacts struct {
	JSONPath     string `json:"json_path"`
	MarkdownPath string `json:"markdown_path"`
}

func ValidateRoot(root string) ValidationReport {
	report := ValidationReport{
		CaseRoot:   root,
		Pass:       true,
		ByLine:     make(map[string]int),
		ByEvidence: make(map[string]int),
	}
	entries, err := os.ReadDir(root)
	if err != nil {
		report.Pass = false
		report.Results = []CaseResult{{
			CaseID: filepath.Base(root),
			Pass:   false,
			Violations: []Violation{{
				Code:    "case_root_unreadable",
				Message: err.Error(),
			}},
		}}
		return report
	}

	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		report.Cases++
		caseDir := filepath.Join(root, entry.Name())
		result := validateFrozenCase(caseDir, entry.Name())
		if result.EvidenceLevel != "" {
			report.ByEvidence[string(result.EvidenceLevel)]++
		}
		for _, line := range result.Lines {
			report.ByLine[string(line)]++
		}
		if !result.Pass {
			report.Pass = false
		}
		report.Results = append(report.Results, result)
	}
	sort.Slice(report.Results, func(i, j int) bool { return report.Results[i].CaseID < report.Results[j].CaseID })
	return report
}

func validateFrozenCase(dir, fallbackID string) CaseResult {
	result := CaseResult{CaseID: fallbackID}
	c, err := LoadCase(dir)
	if err != nil {
		if strings.Contains(err.Error(), "sealed evidence cannot be loaded") {
			result.Violations = append(result.Violations, Violation{
				Code:    "local_sealed_evidence",
				Message: "sealed evidence cannot be loaded from a readable local case",
			})
		} else if strings.HasPrefix(err.Error(), "case validation failed: ") {
			codes := strings.Split(strings.TrimPrefix(err.Error(), "case validation failed: "), ",")
			for _, code := range codes {
				result.Violations = append(result.Violations, Violation{Code: code, Message: "case validation failed"})
			}
		} else {
			result.Violations = append(result.Violations, Violation{Code: "case_load_failed", Message: err.Error()})
		}
	} else {
		result.CaseID = c.Manifest.ID
		result.EvidenceLevel = c.Manifest.EvidenceLevel
		result.Lines = append([]ContinuityLine(nil), c.Manifest.ContinuityLines...)
		result.Pressures = append([]string(nil), c.Manifest.Pressures...)
	}

	lockPath := filepath.Join(dir, fixtureLockFilename)
	lockData, lockErr := os.ReadFile(lockPath)
	if lockErr != nil {
		result.Violations = append(result.Violations, Violation{Code: "fixture_lock_missing", Message: lockErr.Error()})
	} else {
		result.LockSHA256 = hashBytes(lockData)
		var lock FixtureLock
		if err := decodeStrictJSON(lockData, &lock); err != nil {
			result.Violations = append(result.Violations, Violation{Code: "fixture_lock_invalid", Message: err.Error()})
		} else {
			result.Violations = append(result.Violations, verifyFixtureLock(dir, result.CaseID, lock)...)
		}
	}

	sort.SliceStable(result.Violations, func(i, j int) bool {
		if result.Violations[i].Code == result.Violations[j].Code {
			return result.Violations[i].Message < result.Violations[j].Message
		}
		return result.Violations[i].Code < result.Violations[j].Code
	})
	result.Pass = len(result.Violations) == 0
	return result
}

func verifyFixtureLock(dir, caseID string, lock FixtureLock) []Violation {
	var violations []Violation
	addMismatch := func(message string) {
		violations = append(violations, Violation{Code: "fixture_lock_mismatch", Message: message})
	}
	if lock.Version != 1 {
		addMismatch(fmt.Sprintf("lock version is %d, expected 1", lock.Version))
	}
	if lock.CaseID != caseID {
		addMismatch(fmt.Sprintf("lock case id is %q, expected %q", lock.CaseID, caseID))
	}
	verifyLockedBytes(filepath.Join(dir, "manifest.json"), lock.ManifestHash, -1, "manifest.json", addMismatch)
	verifyLockedBytes(filepath.Join(dir, "events.jsonl"), lock.EventsHash, -1, "events.jsonl", addMismatch)
	for _, file := range lock.Files {
		rel := filepath.FromSlash(file.Path)
		clean := filepath.Clean(rel)
		if file.Path == "" || filepath.IsAbs(rel) || clean != rel || clean == ".." || strings.HasPrefix(clean, ".."+string(filepath.Separator)) {
			addMismatch(fmt.Sprintf("locked path %q is invalid", file.Path))
			continue
		}
		path := filepath.Join(dir, clean)
		within, err := pathWithin(dir, path)
		if err != nil || !within {
			addMismatch(fmt.Sprintf("locked path %q leaves the case directory", file.Path))
			continue
		}
		verifyLockedBytes(path, file.SHA256, file.Bytes, file.Path, addMismatch)
	}
	return violations
}

func verifyLockedBytes(path, expectedHash string, expectedBytes int64, label string, addMismatch func(string)) {
	data, err := os.ReadFile(path)
	if err != nil {
		addMismatch(fmt.Sprintf("%s cannot be read: %v", label, err))
		return
	}
	if actual := hashBytes(data); actual != expectedHash {
		addMismatch(fmt.Sprintf("%s hash is %s, expected %s", label, actual, expectedHash))
	}
	if expectedBytes >= 0 && int64(len(data)) != expectedBytes {
		addMismatch(fmt.Sprintf("%s size is %d, expected %d", label, len(data), expectedBytes))
	}
}

func WriteValidationArtifacts(artifactRoot string, report ValidationReport) (ValidationArtifacts, error) {
	if report.RunID == "" || filepath.Base(report.RunID) != report.RunID || report.RunID == "." || report.RunID == ".." {
		return ValidationArtifacts{}, fmt.Errorf("invalid run id %q", report.RunID)
	}
	dir := filepath.Join(artifactRoot, "reality-validation", report.RunID)
	jsonPath := filepath.Join(dir, "report.json")
	markdownPath := filepath.Join(dir, "report.md")
	jsonData, err := marshalIndented(report)
	if err != nil {
		return ValidationArtifacts{}, err
	}
	if err := writeAtomic(jsonPath, jsonData); err != nil {
		return ValidationArtifacts{}, err
	}
	if err := writeAtomic(markdownPath, []byte(renderValidationMarkdown(report))); err != nil {
		return ValidationArtifacts{}, err
	}
	return ValidationArtifacts{JSONPath: jsonPath, MarkdownPath: markdownPath}, nil
}

func renderValidationMarkdown(report ValidationReport) string {
	var builder strings.Builder
	fmt.Fprintf(&builder, "# Reality Validation: %s\n\n", report.RunID)
	fmt.Fprintf(&builder, "- Pass: `%t`\n", report.Pass)
	fmt.Fprintf(&builder, "- Case root: `%s`\n", report.CaseRoot)
	fmt.Fprintf(&builder, "- Cases: `%d`\n\n", report.Cases)

	builder.WriteString("## Coverage\n\n")
	for _, key := range sortedMapKeys(report.ByLine) {
		fmt.Fprintf(&builder, "- Continuity `%s`: %d\n", key, report.ByLine[key])
	}
	for _, key := range sortedMapKeys(report.ByEvidence) {
		fmt.Fprintf(&builder, "- Evidence `%s`: %d\n", key, report.ByEvidence[key])
	}

	builder.WriteString("\n## Cases\n\n")
	for _, result := range report.Results {
		fmt.Fprintf(&builder, "### %s\n\n", result.CaseID)
		fmt.Fprintf(&builder, "- Pass: `%t`\n", result.Pass)
		if result.LockSHA256 != "" {
			fmt.Fprintf(&builder, "- Lock SHA-256: `%s`\n", result.LockSHA256)
		}
		for _, violation := range result.Violations {
			fmt.Fprintf(&builder, "- Violation `%s`: %s\n", violation.Code, violation.Message)
		}
		builder.WriteByte('\n')
	}
	return builder.String()
}

func sortedMapKeys(values map[string]int) []string {
	keys := make([]string, 0, len(values))
	for key := range values {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	return keys
}
