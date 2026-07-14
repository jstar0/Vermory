package main

import (
	"bytes"
	"encoding/json"
	"strings"
	"testing"

	"vermory/internal/brand"
)

func TestVersionCommandUsesStableBuildMetadata(t *testing.T) {
	originalVersion, originalRevision, originalBuildDate := brand.Version, brand.Revision, brand.BuildDate
	brand.Version = "0.1.0-alpha.1"
	brand.Revision = "abc123"
	brand.BuildDate = "2026-07-14T00:00:00Z"
	t.Cleanup(func() {
		brand.Version, brand.Revision, brand.BuildDate = originalVersion, originalRevision, originalBuildDate
	})

	command := newRootCommand()
	var output bytes.Buffer
	command.SetOut(&output)
	command.SetArgs([]string{"version"})
	if err := command.Execute(); err != nil {
		t.Fatal(err)
	}

	var got map[string]any
	if err := json.Unmarshal(output.Bytes(), &got); err != nil {
		t.Fatalf("version output is not JSON: %v\n%s", err, output.String())
	}
	for key, want := range map[string]string{
		"version":    "0.1.0-alpha.1",
		"revision":   "abc123",
		"build_date": "2026-07-14T00:00:00Z",
	} {
		if got[key] != want {
			t.Fatalf("version field %s = %#v, want %q", key, got[key], want)
		}
	}
	if value, ok := got["go_version"].(string); !ok || value == "" {
		t.Fatalf("missing go_version: %#v", got)
	}
}

func TestRootVersionFlagUsesBrandVersion(t *testing.T) {
	originalVersion := brand.Version
	brand.Version = "0.1.0-alpha.1"
	t.Cleanup(func() { brand.Version = originalVersion })

	command := newRootCommand()
	var output bytes.Buffer
	command.SetOut(&output)
	command.SetArgs([]string{"--version"})
	if err := command.Execute(); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(output.String(), "0.1.0-alpha.1") {
		t.Fatalf("root version output did not use brand version: %q", output.String())
	}
}

func TestRealityAttestationCLIExposesVerifyButNoSignCommand(t *testing.T) {
	verifyFound := false
	for _, command := range newRootCommand().Commands() {
		switch command.Name() {
		case "reality-attestation-verify":
			verifyFound = true
		case "reality-attestation-sign":
			t.Fatal("local sealed attestation signing must not be exposed")
		}
	}
	if !verifyFound {
		t.Fatal("expected reality-attestation-verify command")
	}
}

func TestExperiment0CLIIsRegistered(t *testing.T) {
	for _, command := range newRootCommand().Commands() {
		if command.Name() == "experiment-0" {
			return
		}
	}
	t.Fatal("expected experiment-0 command")
}

func TestProviderCommandsAdvertiseGrokCLI(t *testing.T) {
	for _, command := range newRootCommand().Commands() {
		if command.Name() != "eval-self-case" && command.Name() != "eval-casebook" && command.Name() != "eval-matrix" && command.Name() != "probe-provider" && command.Name() != "acceptance-report" && command.Name() != "benchmark-longmemeval" {
			continue
		}
		flag := command.Flags().Lookup("provider")
		if flag == nil || !strings.Contains(flag.Usage, "grok-cli") {
			t.Fatalf("command %q must advertise grok-cli provider support", command.Name())
		}
	}
}

func TestBenchmarkLongMemEvalCommandIsRegistered(t *testing.T) {
	for _, command := range newRootCommand().Commands() {
		if command.Name() != "benchmark-longmemeval" {
			continue
		}
		for _, flagName := range []string{
			"database-url",
			"source-dataset",
			"qualification",
			"execution",
			"artifact-root",
			"provider",
			"base-url",
			"api-key-env",
			"model",
			"run-id",
			"implementation-revision",
		} {
			if command.Flags().Lookup(flagName) == nil {
				t.Fatalf("benchmark-longmemeval must expose --%s", flagName)
			}
		}
		return
	}
	t.Fatal("expected benchmark-longmemeval command")
}

func TestMCPStdioCommandIsRegistered(t *testing.T) {
	for _, command := range newRootCommand().Commands() {
		if command.Name() != "mcp-stdio" {
			continue
		}
		for _, flagName := range []string{"database-url", "tenant-id"} {
			if command.Flags().Lookup(flagName) == nil {
				t.Fatalf("mcp-stdio must expose --%s", flagName)
			}
		}
		return
	}
	t.Fatal("expected mcp-stdio command")
}

func TestOperatorCommandsAreRegistered(t *testing.T) {
	names := map[string]bool{}
	for _, command := range newRootCommand().Commands() {
		names[command.Name()] = true
	}
	for _, want := range []string{"workspace", "memory"} {
		if !names[want] {
			t.Fatalf("expected root command %q", want)
		}
	}
}

func TestIdentityAndDatabaseCommandsAreRegistered(t *testing.T) {
	names := map[string]bool{}
	for _, command := range newRootCommand().Commands() {
		names[command.Name()] = true
	}
	for _, want := range []string{"identity", "database"} {
		if !names[want] {
			t.Fatalf("expected root command %q", want)
		}
	}
}

func TestAuthenticatedServeCommandIsRegistered(t *testing.T) {
	for _, command := range newRootCommand().Commands() {
		if command.Name() == "serve" {
			return
		}
	}
	t.Fatal("expected authenticated serve command")
}

func TestOperatorMemoryForgetHasNoFreeTextFlag(t *testing.T) {
	root := newRootCommand()
	for _, parent := range root.Commands() {
		if parent.Name() != "memory" {
			continue
		}
		for _, child := range parent.Commands() {
			if child.Name() != "forget" {
				continue
			}
			if child.Flags().Lookup("reason") != nil || child.Flags().Lookup("content") != nil {
				t.Fatal("forget command must not accept free-text deletion content")
			}
			return
		}
	}
	t.Fatal("expected memory forget command")
}

func TestOperatorSourceRevisionCommandIsRegistered(t *testing.T) {
	root := newRootCommand()
	for _, parent := range root.Commands() {
		if parent.Name() != "memory" {
			continue
		}
		for _, child := range parent.Commands() {
			if child.Name() != "revise-source" {
				continue
			}
			for _, flag := range []string{"repo-root", "operation-id", "memory-id", "source-ref", "content"} {
				if child.Flags().Lookup(flag) == nil {
					t.Fatalf("revise-source is missing --%s", flag)
				}
			}
			return
		}
	}
	t.Fatal("expected memory revise-source command")
}
