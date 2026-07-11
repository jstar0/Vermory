package main

import "testing"

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
