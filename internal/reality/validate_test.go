package reality

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestLoadAndValidatePublicCase(t *testing.T) {
	c, err := LoadCase("../../reality/testdata/valid-public")
	if err != nil {
		t.Fatal(err)
	}
	if got := ValidateCase(c); len(got) != 0 {
		t.Fatalf("expected valid case, got violations: %#v", got)
	}
}

func TestRejectsLocalCaseClaimingSealedEvidence(t *testing.T) {
	_, err := LoadCase("../../reality/testdata/invalid-local-sealed")
	if err == nil || !strings.Contains(err.Error(), "sealed evidence cannot be loaded from a readable local case") {
		t.Fatalf("expected sealed-evidence rejection, got %v", err)
	}
}

func TestRequiresExpectedAndForbiddenBehavior(t *testing.T) {
	c := loadValidCase(t)
	c.Manifest.Expectations.ForbiddenFacts = nil
	violations := ValidateCase(c)
	assertViolationCode(t, violations, "forbidden_facts_required")
}

func TestRejectsUnknownManifestFields(t *testing.T) {
	dir := copyCaseFixture(t, "../../reality/testdata/valid-public")
	path := filepath.Join(dir, "manifest.json")
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	data = []byte(strings.Replace(string(data), `"version": 1,`, `"version": 1, "unknown": true,`, 1))
	if err := os.WriteFile(path, data, 0o600); err != nil {
		t.Fatal(err)
	}

	_, err = LoadCase(dir)
	if err == nil || !strings.Contains(err.Error(), "unknown field") {
		t.Fatalf("expected unknown-field rejection, got %v", err)
	}
}

func TestRejectsInvalidEventGraph(t *testing.T) {
	tests := []struct {
		name string
		edit func(*Case)
		code string
	}{
		{
			name: "duplicate source",
			edit: func(c *Case) { c.Manifest.Sources = append(c.Manifest.Sources, c.Manifest.Sources[0]) },
			code: "duplicate_source_id",
		},
		{
			name: "duplicate event",
			edit: func(c *Case) {
				c.Events = append(c.Events, Event{ID: c.Events[0].ID, Sequence: 2, Actor: "user", Channel: "chat", SourceID: c.Events[0].SourceID, Content: "duplicate"})
			},
			code: "duplicate_event_id",
		},
		{
			name: "unknown event source",
			edit: func(c *Case) { c.Events[0].SourceID = "missing" },
			code: "event_source_unknown",
		},
		{
			name: "noncontiguous sequence",
			edit: func(c *Case) { c.Events[0].Sequence = 2 },
			code: "event_sequence_invalid",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := loadValidCase(t)
			tt.edit(&c)
			assertViolationCode(t, ValidateCase(c), tt.code)
		})
	}
}

func TestRejectsInvalidFixtureEvidence(t *testing.T) {
	tests := []struct {
		name string
		edit func(*Case)
		code string
	}{
		{
			name: "absolute path",
			edit: func(c *Case) { c.Manifest.Sources[0].FixturePath = "/tmp/source.md" },
			code: "fixture_path_invalid",
		},
		{
			name: "path traversal",
			edit: func(c *Case) { c.Manifest.Sources[0].FixturePath = "../source.md" },
			code: "fixture_path_invalid",
		},
		{
			name: "unauthorized source",
			edit: func(c *Case) { c.Manifest.Sources[0].Authorized = false },
			code: "source_not_authorized",
		},
		{
			name: "missing anonymization",
			edit: func(c *Case) { c.Manifest.Sources[0].Anonymization = "" },
			code: "source_anonymization_required",
		},
		{
			name: "hash mismatch",
			edit: func(c *Case) { c.Manifest.Sources[0].SHA256 = strings.Repeat("0", 64) },
			code: "fixture_hash_mismatch",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := loadValidCase(t)
			tt.edit(&c)
			assertViolationCode(t, ValidateCase(c), tt.code)
		})
	}
}

func TestRejectsFactDeclaredCurrentAndForbidden(t *testing.T) {
	c := loadValidCase(t)
	c.Manifest.Expectations.ForbiddenFacts = append(c.Manifest.Expectations.ForbiddenFacts, c.Manifest.Expectations.CurrentFacts[0])
	assertViolationCode(t, ValidateCase(c), "fact_expectation_conflict")
}

func loadValidCase(t *testing.T) Case {
	t.Helper()
	c, err := LoadCase("../../reality/testdata/valid-public")
	if err != nil {
		t.Fatal(err)
	}
	return c
}

func assertViolationCode(t *testing.T, violations []Violation, code string) {
	t.Helper()
	for _, violation := range violations {
		if violation.Code == code {
			return
		}
	}
	t.Fatalf("expected violation %q, got %#v", code, violations)
}

func copyCaseFixture(t *testing.T, source string) string {
	t.Helper()
	destination := t.TempDir()
	entries, err := os.ReadDir(source)
	if err != nil {
		t.Fatal(err)
	}
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		data, err := os.ReadFile(filepath.Join(source, entry.Name()))
		if err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(destination, entry.Name()), data, 0o600); err != nil {
			t.Fatal(err)
		}
	}
	return destination
}
