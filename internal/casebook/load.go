package casebook

import (
	"encoding/json"
	"os"
	"path/filepath"
)

func LoadBenchmarkMap(path string) ([]BenchmarkMapEntry, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	var entries []BenchmarkMapEntry
	if err := json.Unmarshal(data, &entries); err != nil {
		return nil, err
	}

	return entries, nil
}

func LoadCase(dir string) (Case, error) {
	sourcePath := filepath.Join(dir, "source.md")
	sourceBytes, err := os.ReadFile(sourcePath)
	if err != nil {
		return Case{}, err
	}

	tasks, err := loadJSON[[]Task](filepath.Join(dir, "tasks.json"))
	if err != nil {
		return Case{}, err
	}

	claims, err := loadJSON[[]Claim](filepath.Join(dir, "claims.json"))
	if err != nil {
		return Case{}, err
	}

	return Case{
		ID:        filepath.Base(dir),
		Directory: dir,
		SourceMD:  string(sourceBytes),
		Tasks:     tasks,
		Claims:    claims,
	}, nil
}

func loadJSON[T any](path string) (T, error) {
	var value T

	data, err := os.ReadFile(path)
	if err != nil {
		return value, err
	}

	if err := json.Unmarshal(data, &value); err != nil {
		return value, err
	}

	return value, nil
}
