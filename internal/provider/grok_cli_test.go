package provider

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestGrokCLIProviderRunsIsolatedSingleTurnAndCapturesJSON(t *testing.T) {
	dir := t.TempDir()
	capturePath := filepath.Join(dir, "arguments.txt")
	commandPath := filepath.Join(dir, "grok")
	script := fmt.Sprintf("#!/bin/sh\nprintf '%%s\\n' \"$@\" > %q\nprintf '%%s\\n' '{\"text\":\"grok response\",\"modelUsage\":{\"grok-4.5\":{}}}'\n", capturePath)
	if err := os.WriteFile(commandPath, []byte(script), 0o700); err != nil {
		t.Fatalf("write fake grok command: %v", err)
	}

	client := NewGrokCLI(GrokCLIConfig{Command: commandPath})
	resp, err := client.Generate(context.Background(), GenerateRequest{
		Model:         "grok-4.5",
		System:        "system instructions",
		Prompt:        "finish the task",
		ContextPacket: "confirmed context packet",
		MaxTokens:     64,
	})
	if err != nil {
		t.Fatalf("Generate returned error: %v", err)
	}
	if resp.Output != "grok response" {
		t.Fatalf("unexpected output: %q", resp.Output)
	}
	if resp.Model != "grok-4.5" {
		t.Fatalf("unexpected model: %q", resp.Model)
	}
	if !json.Valid(resp.RawArtifact) {
		t.Fatalf("raw artifact should be the Grok JSON response, got %q", resp.RawArtifact)
	}

	arguments, err := os.ReadFile(capturePath)
	if err != nil {
		t.Fatalf("read captured arguments: %v", err)
	}
	for _, want := range []string{
		"--no-memory",
		"--disable-web-search",
		"--no-plan",
		"--no-subagents",
		"--max-turns",
		"1",
		"--permission-mode",
		"dontAsk",
		"--output-format",
		"json",
		"--model",
		"grok-4.5",
		"system instructions",
		"confirmed context packet",
		"finish the task",
	} {
		if !strings.Contains(string(arguments), want) {
			t.Fatalf("expected Grok invocation to contain %q, got %q", want, arguments)
		}
	}
}
