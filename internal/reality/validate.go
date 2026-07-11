package reality

import (
	"bufio"
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

const maxEventLineBytes = 16 * 1024 * 1024

func LoadCase(dir string) (Case, error) {
	manifestPath := filepath.Join(dir, "manifest.json")
	var manifest Manifest
	if err := decodeStrictJSONFile(manifestPath, &manifest); err != nil {
		return Case{}, fmt.Errorf("decode manifest: %w", err)
	}
	if manifest.EvidenceLevel == "sealed" {
		return Case{}, errors.New("sealed evidence cannot be loaded from a readable local case")
	}

	events, err := decodeEvents(filepath.Join(dir, "events.jsonl"))
	if err != nil {
		return Case{}, err
	}

	c := Case{
		Directory: dir,
		Manifest:  manifest,
		Events:    events,
	}
	violations := ValidateCase(c)
	if len(violations) != 0 {
		codes := make([]string, 0, len(violations))
		for _, violation := range violations {
			codes = append(codes, violation.Code)
		}
		return Case{}, fmt.Errorf("case validation failed: %s", strings.Join(codes, ","))
	}
	return c, nil
}

func ValidateCase(c Case) []Violation {
	var violations []Violation
	add := func(code, message string) {
		violations = append(violations, Violation{Code: code, Message: message})
	}

	if c.Manifest.Version != 1 {
		add("version_unsupported", "manifest version must be 1")
	}
	if strings.TrimSpace(c.Manifest.ID) == "" {
		add("id_required", "manifest id is required")
	}
	if strings.TrimSpace(c.Manifest.Title) == "" {
		add("title_required", "manifest title is required")
	}
	if c.Manifest.EvidenceLevel != EvidencePublic && c.Manifest.EvidenceLevel != EvidenceWithheldLocal {
		add("evidence_level_invalid", "local evidence level must be public or withheld_local")
	}
	if len(c.Manifest.ContinuityLines) == 0 {
		add("continuity_lines_required", "at least one continuity line is required")
	}
	for _, line := range c.Manifest.ContinuityLines {
		if !validContinuityLine(line) {
			add("continuity_line_invalid", fmt.Sprintf("unsupported continuity line %q", line))
		}
	}
	if len(c.Manifest.Pressures) == 0 {
		add("pressures_required", "at least one pressure is required")
	}
	if len(c.Manifest.Sources) == 0 {
		add("sources_required", "at least one source is required")
	}
	if len(c.Manifest.Anchors) == 0 {
		add("anchors_required", "at least one anchor is required")
	}
	if len(c.Manifest.Expectations.CurrentFacts) == 0 {
		add("current_facts_required", "at least one current fact is required")
	}
	if len(c.Manifest.Expectations.ForbiddenFacts) == 0 {
		add("forbidden_facts_required", "at least one forbidden fact is required")
	}
	if strings.TrimSpace(c.Manifest.Expectations.ExpectedAction) == "" {
		add("expected_action_required", "expected action is required")
	}
	if strings.TrimSpace(c.Manifest.Task.Prompt) == "" {
		add("task_prompt_required", "downstream task prompt is required")
	}
	if len(c.Manifest.Task.DeterministicChecks) == 0 {
		add("deterministic_checks_required", "at least one deterministic check is required")
	}

	sourceIDs := make(map[string]struct{}, len(c.Manifest.Sources))
	for i, source := range c.Manifest.Sources {
		if strings.TrimSpace(source.ID) == "" {
			add("source_id_required", fmt.Sprintf("source %d has no id", i))
		} else if _, exists := sourceIDs[source.ID]; exists {
			add("duplicate_source_id", fmt.Sprintf("source id %q is duplicated", source.ID))
		} else {
			sourceIDs[source.ID] = struct{}{}
		}
		if strings.TrimSpace(source.Kind) == "" {
			add("source_kind_required", fmt.Sprintf("source %q has no kind", source.ID))
		}
		if !source.Authorized {
			add("source_not_authorized", fmt.Sprintf("source %q is not authorized", source.ID))
		}
		if strings.TrimSpace(source.Anonymization) == "" {
			add("source_anonymization_required", fmt.Sprintf("source %q has no anonymization statement", source.ID))
		}
		validateFixture(c.Directory, source, add)
	}

	if len(c.Events) == 0 {
		add("events_required", "events.jsonl must contain at least one event")
	}
	eventIDs := make(map[string]struct{}, len(c.Events))
	for i, event := range c.Events {
		if strings.TrimSpace(event.ID) == "" {
			add("event_id_required", fmt.Sprintf("event at sequence %d has no id", event.Sequence))
		} else if _, exists := eventIDs[event.ID]; exists {
			add("duplicate_event_id", fmt.Sprintf("event id %q is duplicated", event.ID))
		} else {
			eventIDs[event.ID] = struct{}{}
		}
		expectedSequence := i + 1
		if event.Sequence != expectedSequence {
			add("event_sequence_invalid", fmt.Sprintf("event %q has sequence %d, expected %d", event.ID, event.Sequence, expectedSequence))
		}
		if _, exists := sourceIDs[event.SourceID]; !exists {
			add("event_source_unknown", fmt.Sprintf("event %q references unknown source %q", event.ID, event.SourceID))
		}
		if strings.TrimSpace(event.Actor) == "" {
			add("event_actor_required", fmt.Sprintf("event %q has no actor", event.ID))
		}
		if strings.TrimSpace(event.Channel) == "" {
			add("event_channel_required", fmt.Sprintf("event %q has no channel", event.ID))
		}
		if strings.TrimSpace(event.Content) == "" {
			add("event_content_required", fmt.Sprintf("event %q has no content", event.ID))
		}
	}

	validateFactSets(c.Manifest.Expectations, add)
	sort.SliceStable(violations, func(i, j int) bool {
		if violations[i].Code == violations[j].Code {
			return violations[i].Message < violations[j].Message
		}
		return violations[i].Code < violations[j].Code
	})
	return violations
}

func decodeStrictJSONFile(path string, target any) error {
	data, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	return decodeStrictJSON(data, target)
}

func decodeStrictJSON(data []byte, target any) error {
	decoder := json.NewDecoder(bytes.NewReader(data))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(target); err != nil {
		return err
	}
	var trailing any
	if err := decoder.Decode(&trailing); !errors.Is(err, io.EOF) {
		if err == nil {
			return errors.New("multiple JSON values")
		}
		return err
	}
	return nil
}

func decodeEvents(path string) ([]Event, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("open events: %w", err)
	}
	defer file.Close()

	var events []Event
	scanner := bufio.NewScanner(file)
	scanner.Buffer(make([]byte, 64*1024), maxEventLineBytes)
	line := 0
	for scanner.Scan() {
		line++
		data := bytes.TrimSpace(scanner.Bytes())
		if len(data) == 0 {
			return nil, fmt.Errorf("decode events line %d: empty lines are not allowed", line)
		}
		var event Event
		if err := decodeStrictJSON(data, &event); err != nil {
			return nil, fmt.Errorf("decode events line %d: %w", line, err)
		}
		events = append(events, event)
	}
	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("scan events: %w", err)
	}
	if len(events) == 0 {
		return nil, errors.New("events.jsonl must contain at least one event")
	}
	return events, nil
}

func validContinuityLine(line ContinuityLine) bool {
	switch line {
	case LineWorkspace, LineConversation, LineGlobalDefaults, LineBridge, LineSecurity:
		return true
	default:
		return false
	}
}

func validateFixture(caseDir string, source SourceRef, add func(string, string)) {
	rel := filepath.FromSlash(source.FixturePath)
	clean := filepath.Clean(rel)
	if source.FixturePath == "" || filepath.IsAbs(rel) || clean == "." || clean != rel || clean == ".." || strings.HasPrefix(clean, ".."+string(filepath.Separator)) {
		add("fixture_path_invalid", fmt.Sprintf("source %q fixture path must be clean and relative", source.ID))
		return
	}

	caseAbs, err := filepath.Abs(caseDir)
	if err != nil {
		add("fixture_path_invalid", fmt.Sprintf("source %q case directory cannot be resolved", source.ID))
		return
	}
	fixturePath := filepath.Join(caseAbs, clean)
	within, err := pathWithin(caseAbs, fixturePath)
	if err != nil || !within {
		add("fixture_path_invalid", fmt.Sprintf("source %q fixture leaves the case directory", source.ID))
		return
	}

	data, err := os.ReadFile(fixturePath)
	if err != nil {
		add("fixture_unreadable", fmt.Sprintf("source %q fixture cannot be read: %v", source.ID, err))
		return
	}
	digest := sha256.Sum256(data)
	actual := hex.EncodeToString(digest[:])
	if source.SHA256 != actual {
		add("fixture_hash_mismatch", fmt.Sprintf("source %q fixture hash is %s, declared %s", source.ID, actual, source.SHA256))
	}
}

func pathWithin(root, target string) (bool, error) {
	realRoot, err := filepath.EvalSymlinks(root)
	if err != nil {
		return false, err
	}
	realTarget, err := filepath.EvalSymlinks(target)
	if err != nil {
		return false, err
	}
	rel, err := filepath.Rel(realRoot, realTarget)
	if err != nil {
		return false, err
	}
	return rel != ".." && !strings.HasPrefix(rel, ".."+string(filepath.Separator)), nil
}

func validateFactSets(expectations Expectations, add func(string, string)) {
	current := make(map[string]struct{}, len(expectations.CurrentFacts))
	for _, fact := range expectations.CurrentFacts {
		if _, exists := current[fact]; exists {
			add("duplicate_current_fact", fmt.Sprintf("current fact %q is duplicated", fact))
		}
		current[fact] = struct{}{}
	}
	forbidden := make(map[string]struct{}, len(expectations.ForbiddenFacts))
	for _, fact := range expectations.ForbiddenFacts {
		if _, exists := forbidden[fact]; exists {
			add("duplicate_forbidden_fact", fmt.Sprintf("forbidden fact %q is duplicated", fact))
		}
		forbidden[fact] = struct{}{}
		if _, exists := current[fact]; exists {
			add("fact_expectation_conflict", fmt.Sprintf("fact %q is both current and forbidden", fact))
		}
	}
}
