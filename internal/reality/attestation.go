package reality

import (
	"crypto/ed25519"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"
)

type Attestation struct {
	Version              int            `json:"version"`
	EvaluatorID          string         `json:"evaluator_id"`
	SuiteVersion         string         `json:"suite_version"`
	ImplementationDigest string         `json:"implementation_digest"`
	RunAt                time.Time      `json:"run_at"`
	HardGatesPass        bool           `json:"hard_gates_pass"`
	Counts               map[string]int `json:"counts"`
	FailureCategories    []string       `json:"failure_categories,omitempty"`
	Signature            string         `json:"signature"`
}

type attestationWire struct {
	Version              *int            `json:"version"`
	EvaluatorID          *string         `json:"evaluator_id"`
	SuiteVersion         *string         `json:"suite_version"`
	ImplementationDigest *string         `json:"implementation_digest"`
	RunAt                *time.Time      `json:"run_at"`
	HardGatesPass        *bool           `json:"hard_gates_pass"`
	Counts               *map[string]int `json:"counts"`
	FailureCategories    []string        `json:"failure_categories,omitempty"`
	Signature            *string         `json:"signature"`
}

type unsignedAttestation struct {
	Version              int            `json:"version"`
	EvaluatorID          string         `json:"evaluator_id"`
	SuiteVersion         string         `json:"suite_version"`
	ImplementationDigest string         `json:"implementation_digest"`
	RunAt                time.Time      `json:"run_at"`
	HardGatesPass        bool           `json:"hard_gates_pass"`
	Counts               map[string]int `json:"counts"`
	FailureCategories    []string       `json:"failure_categories,omitempty"`
}

func VerifyAttestation(data []byte, publicKey ed25519.PublicKey) (Attestation, error) {
	if len(publicKey) != ed25519.PublicKeySize {
		return Attestation{}, fmt.Errorf("invalid Ed25519 public key length %d", len(publicKey))
	}

	var wire attestationWire
	if err := decodeStrictJSON(data, &wire); err != nil {
		return Attestation{}, fmt.Errorf("decode attestation: %w", err)
	}
	attestation, err := validateAttestationWire(wire)
	if err != nil {
		return Attestation{}, err
	}
	signature, err := base64.StdEncoding.DecodeString(attestation.Signature)
	if err != nil {
		return Attestation{}, fmt.Errorf("decode signature: %w", err)
	}
	if len(signature) != ed25519.SignatureSize {
		return Attestation{}, fmt.Errorf("invalid Ed25519 signature length %d", len(signature))
	}
	payload, err := canonicalAttestationPayload(attestation)
	if err != nil {
		return Attestation{}, err
	}
	if !ed25519.Verify(publicKey, payload, signature) {
		return Attestation{}, errors.New("attestation signature verification failed")
	}
	return attestation, nil
}

func validateAttestationWire(wire attestationWire) (Attestation, error) {
	if wire.Version == nil {
		return Attestation{}, errors.New("version is required")
	}
	if *wire.Version != 1 {
		return Attestation{}, fmt.Errorf("attestation version %d is unsupported", *wire.Version)
	}
	if wire.EvaluatorID == nil || strings.TrimSpace(*wire.EvaluatorID) == "" {
		return Attestation{}, errors.New("evaluator_id is required")
	}
	if wire.SuiteVersion == nil || strings.TrimSpace(*wire.SuiteVersion) == "" {
		return Attestation{}, errors.New("suite_version is required")
	}
	if wire.ImplementationDigest == nil || !validSHA256(*wire.ImplementationDigest) {
		return Attestation{}, errors.New("implementation_digest is required and must be a lowercase SHA-256 digest")
	}
	if wire.RunAt == nil || wire.RunAt.IsZero() {
		return Attestation{}, errors.New("run_at is required")
	}
	if wire.HardGatesPass == nil {
		return Attestation{}, errors.New("hard_gates_pass is required")
	}
	if wire.Counts == nil || len(*wire.Counts) == 0 {
		return Attestation{}, errors.New("counts is required and must not be empty")
	}
	for key, value := range *wire.Counts {
		if strings.TrimSpace(key) == "" || value < 0 {
			return Attestation{}, errors.New("counts keys must be non-empty and values must be non-negative")
		}
	}
	if wire.Signature == nil || strings.TrimSpace(*wire.Signature) == "" {
		return Attestation{}, errors.New("signature is required")
	}

	return Attestation{
		Version:              *wire.Version,
		EvaluatorID:          *wire.EvaluatorID,
		SuiteVersion:         *wire.SuiteVersion,
		ImplementationDigest: *wire.ImplementationDigest,
		RunAt:                *wire.RunAt,
		HardGatesPass:        *wire.HardGatesPass,
		Counts:               *wire.Counts,
		FailureCategories:    wire.FailureCategories,
		Signature:            *wire.Signature,
	}, nil
}

func canonicalAttestationPayload(attestation Attestation) ([]byte, error) {
	return json.Marshal(unsignedAttestation{
		Version:              attestation.Version,
		EvaluatorID:          attestation.EvaluatorID,
		SuiteVersion:         attestation.SuiteVersion,
		ImplementationDigest: attestation.ImplementationDigest,
		RunAt:                attestation.RunAt,
		HardGatesPass:        attestation.HardGatesPass,
		Counts:               attestation.Counts,
		FailureCategories:    attestation.FailureCategories,
	})
}

func validSHA256(value string) bool {
	if len(value) != 64 || strings.ToLower(value) != value {
		return false
	}
	decoded, err := hex.DecodeString(value)
	return err == nil && len(decoded) == 32
}
