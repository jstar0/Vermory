package memorybackend

import (
	"context"
	"strings"
	"testing"
)

func TestOpenBackendRejectsUnknownBackend(t *testing.T) {
	_, _, err := OpenBackend(context.Background(), OpenConfig{Name: "mystery"})
	if err == nil || !strings.Contains(err.Error(), "unsupported memory backend") {
		t.Fatalf("expected unsupported backend error, got %v", err)
	}
}
