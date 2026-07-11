package artifact

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"net/url"
	"os"
	"path/filepath"
	"strings"
)

var errInvalidKey = errors.New("artifact: invalid key")

type LocalStore struct {
	root string
}

func NewLocalStore(root string) *LocalStore {
	return &LocalStore{root: root}
}

func (s *LocalStore) Put(ctx context.Context, key string, content []byte) (PutResult, error) {
	if err := ctx.Err(); err != nil {
		return PutResult{}, err
	}

	relativePath, err := cleanKey(key)
	if err != nil {
		return PutResult{}, err
	}

	root, err := filepath.Abs(s.root)
	if err != nil {
		return PutResult{}, err
	}
	path := filepath.Join(root, relativePath)
	if err := ctx.Err(); err != nil {
		return PutResult{}, err
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return PutResult{}, err
	}
	if err := writeFileAtomically(ctx, path, content); err != nil {
		return PutResult{}, err
	}

	sum := sha256.Sum256(content)
	return PutResult{
		URI:      fileURI(path),
		Path:     path,
		SHA256:   hex.EncodeToString(sum[:]),
		ByteSize: int64(len(content)),
	}, nil
}

func cleanKey(key string) (string, error) {
	if key == "" || filepath.IsAbs(key) {
		return "", errInvalidKey
	}

	cleaned := filepath.Clean(key)
	if cleaned == "." || cleaned == ".." {
		return "", errInvalidKey
	}

	prefix := ".." + string(filepath.Separator)
	if strings.HasPrefix(cleaned, prefix) {
		return "", errInvalidKey
	}

	return cleaned, nil
}

func writeFileAtomically(ctx context.Context, path string, content []byte) error {
	if err := ctx.Err(); err != nil {
		return err
	}

	file, err := os.CreateTemp(filepath.Dir(path), ".contextmesh-*")
	if err != nil {
		return err
	}
	tempPath := file.Name()
	shouldRemove := true
	defer func() {
		if shouldRemove {
			_ = os.Remove(tempPath)
		}
	}()

	if _, err := file.Write(content); err != nil {
		_ = file.Close()
		return err
	}
	if err := file.Close(); err != nil {
		return err
	}
	if err := ctx.Err(); err != nil {
		return err
	}
	if err := os.Rename(tempPath, path); err != nil {
		return err
	}
	shouldRemove = false
	return nil
}

func fileURI(path string) string {
	return (&url.URL{Scheme: "file", Path: filepath.ToSlash(path)}).String()
}
