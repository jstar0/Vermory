package reality

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
)

const fixtureLockFilename = "fixture-lock.json"

type FixtureLock struct {
	Version      int          `json:"version"`
	CaseID       string       `json:"case_id"`
	ManifestHash string       `json:"manifest_sha256"`
	EventsHash   string       `json:"events_sha256"`
	Files        []LockedFile `json:"files"`
}

type LockedFile struct {
	Path   string `json:"path"`
	SHA256 string `json:"sha256"`
	Bytes  int64  `json:"bytes"`
}

type FreezeReport struct {
	CaseID     string `json:"case_id"`
	LockPath   string `json:"lock_path"`
	LockSHA256 string `json:"lock_sha256"`
}

func FreezeCase(dir string) (FreezeReport, error) {
	c, err := LoadCase(dir)
	if err != nil {
		return FreezeReport{}, err
	}
	lock, err := buildFixtureLock(c)
	if err != nil {
		return FreezeReport{}, err
	}
	data, err := marshalIndented(lock)
	if err != nil {
		return FreezeReport{}, err
	}
	lockPath := filepath.Join(dir, fixtureLockFilename)
	if err := writeAtomic(lockPath, data); err != nil {
		return FreezeReport{}, err
	}
	return FreezeReport{
		CaseID:     c.Manifest.ID,
		LockPath:   lockPath,
		LockSHA256: hashBytes(data),
	}, nil
}

func buildFixtureLock(c Case) (FixtureLock, error) {
	manifest, err := os.ReadFile(filepath.Join(c.Directory, "manifest.json"))
	if err != nil {
		return FixtureLock{}, err
	}
	events, err := os.ReadFile(filepath.Join(c.Directory, "events.jsonl"))
	if err != nil {
		return FixtureLock{}, err
	}

	paths := make(map[string]struct{}, len(c.Manifest.Sources))
	for _, source := range c.Manifest.Sources {
		paths[filepath.ToSlash(source.FixturePath)] = struct{}{}
	}
	orderedPaths := make([]string, 0, len(paths))
	for path := range paths {
		orderedPaths = append(orderedPaths, path)
	}
	sort.Strings(orderedPaths)

	files := make([]LockedFile, 0, len(orderedPaths))
	for _, path := range orderedPaths {
		data, err := os.ReadFile(filepath.Join(c.Directory, filepath.FromSlash(path)))
		if err != nil {
			return FixtureLock{}, fmt.Errorf("read fixture %s: %w", path, err)
		}
		files = append(files, LockedFile{Path: path, SHA256: hashBytes(data), Bytes: int64(len(data))})
	}

	return FixtureLock{
		Version:      1,
		CaseID:       c.Manifest.ID,
		ManifestHash: hashBytes(manifest),
		EventsHash:   hashBytes(events),
		Files:        files,
	}, nil
}

func marshalIndented(value any) ([]byte, error) {
	data, err := json.MarshalIndent(value, "", "  ")
	if err != nil {
		return nil, err
	}
	return append(data, '\n'), nil
}

func hashBytes(data []byte) string {
	digest := sha256.Sum256(data)
	return hex.EncodeToString(digest[:])
}

func writeAtomic(path string, data []byte) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	file, err := os.CreateTemp(filepath.Dir(path), ".vermory-*")
	if err != nil {
		return err
	}
	tempPath := file.Name()
	remove := true
	defer func() {
		if remove {
			_ = os.Remove(tempPath)
		}
	}()
	if err := file.Chmod(0o644); err != nil {
		_ = file.Close()
		return err
	}
	if _, err := file.Write(data); err != nil {
		_ = file.Close()
		return err
	}
	if err := file.Sync(); err != nil {
		_ = file.Close()
		return err
	}
	if err := file.Close(); err != nil {
		return err
	}
	if err := os.Rename(tempPath, path); err != nil {
		return err
	}
	remove = false
	return nil
}
