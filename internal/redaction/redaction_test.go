package redaction

import (
	"strings"
	"testing"
)

func TestRedactSecrets(t *testing.T) {
	secret := "sk-" + strings.Repeat("a", 20)
	input := "key " + secret + " email user@example.com phone 13800138000"
	result := Redact(input)
	if result.Text == input {
		t.Fatal("expected text to be redacted")
	}
	if result.Count != 3 {
		t.Fatalf("expected 3 redactions, got %d", result.Count)
	}
	if ContainsSensitive(result.Text) {
		t.Fatal("redacted text still contains sensitive content")
	}
}
