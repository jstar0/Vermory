package main

import (
	"bytes"
	"context"
	"errors"
	"os"
	"strings"
	"testing"

	"vermory/internal/runtime"
)

func TestServeOptionsRequireRuntimeDatabaseAndTLSForNonLoopback(t *testing.T) {
	validLoopback := serveOptions{DatabaseURL: "postgresql:///vermory", Listen: "127.0.0.1:8788"}
	if err := validLoopback.Validate(); err != nil {
		t.Fatalf("loopback without TLS should be valid: %v", err)
	}
	validTLS := serveOptions{
		DatabaseURL: "postgresql:///vermory",
		Listen:      "0.0.0.0:8788",
		TLSCert:     "/etc/vermory/tls.crt",
		TLSKey:      "/etc/vermory/tls.key",
	}
	if err := validTLS.Validate(); err != nil {
		t.Fatalf("non-loopback with TLS pair should be valid: %v", err)
	}

	tests := map[string]serveOptions{
		"database":  {Listen: "127.0.0.1:8788"},
		"address":   {DatabaseURL: "postgresql:///vermory", Listen: "not-an-address"},
		"cleartext": {DatabaseURL: "postgresql:///vermory", Listen: "0.0.0.0:8788"},
		"cert-only": {DatabaseURL: "postgresql:///vermory", Listen: "127.0.0.1:8788", TLSCert: "/tmp/cert"},
		"key-only":  {DatabaseURL: "postgresql:///vermory", Listen: "127.0.0.1:8788", TLSKey: "/tmp/key"},
	}
	for name, options := range tests {
		t.Run(name, func(t *testing.T) {
			if err := options.Validate(); err == nil {
				t.Fatalf("unsafe serve options were accepted: %#v", options)
			}
		})
	}
}

func TestServeCommandExposesNoTenantOrImplicitMigrationControls(t *testing.T) {
	command := newServeCommand()
	for _, forbidden := range []string{"tenant-id", "migrate", "admin-database-url"} {
		if command.Flags().Lookup(forbidden) != nil {
			t.Fatalf("serve must not expose --%s", forbidden)
		}
	}
	for _, required := range []string{"database-url", "listen", "tls-cert", "tls-key", "provider", "model"} {
		if command.Flags().Lookup(required) == nil {
			t.Fatalf("serve is missing --%s", required)
		}
	}
}

func TestServeRejectsUnsafeDatabaseRoleBeforeListening(t *testing.T) {
	databaseURL := os.Getenv("VERMORY_TEST_DATABASE_URL")
	if databaseURL == "" {
		t.Skip("VERMORY_TEST_DATABASE_URL is not set")
	}
	store, err := runtime.OpenStore(context.Background(), databaseURL)
	if err != nil {
		t.Fatal(err)
	}
	if err := store.Migrate(context.Background()); err != nil {
		store.Close()
		t.Fatal(err)
	}
	store.Close()

	command := newServeCommand()
	command.SetOut(&bytes.Buffer{})
	command.SetErr(&bytes.Buffer{})
	command.SetArgs([]string{"--database-url", databaseURL, "--listen", "127.0.0.1:0", "--provider", "mock"})
	err = command.Execute()
	if !errors.Is(err, runtime.ErrUnsafeRuntimeRole) {
		t.Fatalf("serve did not reject admin/table-owner role: %v", err)
	}
	if err != nil && strings.Contains(err.Error(), databaseURL) {
		t.Fatalf("serve error exposed database URL: %v", err)
	}
}
