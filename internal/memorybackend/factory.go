package memorybackend

import (
	"context"
	"fmt"
	"net/http"
	"strings"
)

type OpenConfig struct {
	Name             string
	BaseURL          string
	APIKey           string
	DatabaseURL      string
	EmbeddingBaseURL string
	EmbeddingAPIKey  string
	EmbeddingModel   string
	Dimensions       int
	HTTPClient       *http.Client
}

func OpenBackend(ctx context.Context, config OpenConfig) (Backend, func(), error) {
	cleanup := func() {}
	switch strings.ToLower(strings.TrimSpace(config.Name)) {
	case "mem0":
		backend, err := NewMem0Backend(Mem0Config{BaseURL: config.BaseURL, APIKey: config.APIKey, Client: config.HTTPClient})
		return backend, cleanup, err
	case "supermemory":
		backend, err := NewSupermemoryBackend(SupermemoryConfig{BaseURL: config.BaseURL, APIKey: config.APIKey, Client: config.HTTPClient})
		return backend, cleanup, err
	case "memos":
		backend, err := NewMemOSBackend(MemOSConfig{BaseURL: config.BaseURL, APIKey: config.APIKey, Client: config.HTTPClient})
		return backend, cleanup, err
	case "native":
		backend, err := NewNativeBackend(ctx, NativeConfig{
			DatabaseURL: config.DatabaseURL, EmbeddingBaseURL: config.EmbeddingBaseURL,
			EmbeddingAPIKey: config.EmbeddingAPIKey, EmbeddingModel: config.EmbeddingModel,
			Dimensions: config.Dimensions, HTTPClient: config.HTTPClient,
		})
		if err != nil {
			return nil, cleanup, err
		}
		return backend, backend.Close, nil
	default:
		return nil, cleanup, fmt.Errorf("unsupported memory backend %q", config.Name)
	}
}
