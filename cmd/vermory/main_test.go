package main

import (
	"strings"
	"testing"
)

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
		if command.Name() != "eval-self-case" && command.Name() != "eval-casebook" && command.Name() != "eval-matrix" && command.Name() != "probe-provider" && command.Name() != "acceptance-report" {
			continue
		}
		flag := command.Flags().Lookup("provider")
		if flag == nil || !strings.Contains(flag.Usage, "grok-cli") {
			t.Fatalf("command %q must advertise grok-cli provider support", command.Name())
		}
	}
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
