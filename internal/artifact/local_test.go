package artifact

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"net/url"
	"os"
	"path/filepath"
	"testing"
)

func TestLocalStoreWritesAndHashesArtifact(t *testing.T) {
	root := t.TempDir()
	store := NewLocalStore(root)
	content := []byte("hello contextmesh")
	result, err := store.Put(context.Background(), "sources/project/source/v1/raw.md", content)
	if err != nil {
		t.Fatal(err)
	}
	if result.ByteSize != 17 {
		t.Fatalf("expected byte size 17, got %d", result.ByteSize)
	}
	sum := sha256.Sum256(content)
	if result.SHA256 != hex.EncodeToString(sum[:]) {
		t.Fatalf("unexpected sha256 hash: %s", result.SHA256)
	}
	written, err := os.ReadFile(result.Path)
	if err != nil {
		t.Fatal(err)
	}
	if string(written) != string(content) {
		t.Fatalf("unexpected file content: %q", written)
	}
	parsedURI, err := url.Parse(result.URI)
	if err != nil {
		t.Fatal(err)
	}
	if parsedURI.Scheme != "file" {
		t.Fatalf("expected file URI, got %q", result.URI)
	}
}

func TestLocalStoreRejectsPathTraversal(t *testing.T) {
	root := t.TempDir()
	store := NewLocalStore(root)

	if _, err := store.Put(context.Background(), "../escape.txt", []byte("x")); err == nil {
		t.Fatal("expected path traversal error")
	}
	if _, err := store.Put(context.Background(), "/tmp/escape.txt", []byte("x")); err == nil {
		t.Fatal("expected absolute path error")
	}
}

func TestLocalStoreEscapesFileURI(t *testing.T) {
	root := t.TempDir()
	store := NewLocalStore(root)

	result, err := store.Put(context.Background(), "sources/project/raw #1%.md", []byte("x"))
	if err != nil {
		t.Fatal(err)
	}

	parsedURI, err := url.Parse(result.URI)
	if err != nil {
		t.Fatal(err)
	}
	if parsedURI.Scheme != "file" {
		t.Fatalf("expected file URI, got %q", result.URI)
	}
	if parsedURI.Path != result.Path {
		t.Fatalf("expected URI path %q to round-trip to artifact path %q", parsedURI.Path, result.Path)
	}
	if result.URI == "file://"+result.Path {
		t.Fatalf("expected escaped file URI, got raw concatenation %q", result.URI)
	}
}

func TestLocalStoreDoesNotCommitFinalArtifactWhenContextCanceled(t *testing.T) {
	root := t.TempDir()
	store := NewLocalStore(root)
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	_, err := store.Put(ctx, "sources/project/source/v1/raw.md", []byte("x"))
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("expected context canceled, got %v", err)
	}
	if _, statErr := os.Stat(filepath.Join(root, "sources/project/source/v1/raw.md")); !errors.Is(statErr, os.ErrNotExist) {
		t.Fatalf("expected final artifact not to exist, got %v", statErr)
	}
}
