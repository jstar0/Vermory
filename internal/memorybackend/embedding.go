package memorybackend

import (
	"context"
	"fmt"
	"net/http"
	"strconv"
	"strings"
)

type openAIEmbedder struct {
	remote     remoteClient
	model      string
	dimensions int
}

type Embedder interface {
	Embed(context.Context, string) ([]float32, error)
}

func NewOpenAIEmbedder(baseURL, apiKey, model string, dimensions int, client *http.Client) (Embedder, error) {
	return newOpenAIEmbedder(baseURL, apiKey, model, dimensions, client)
}

func newOpenAIEmbedder(baseURL, apiKey, model string, dimensions int, client *http.Client) (*openAIEmbedder, error) {
	if strings.TrimSpace(model) == "" {
		return nil, fmt.Errorf("embedding model is required")
	}
	if dimensions <= 0 {
		return nil, fmt.Errorf("embedding dimensions must be positive")
	}
	headers := map[string]string{}
	if apiKey != "" {
		headers["Authorization"] = "Bearer " + apiKey
	}
	remote, err := newRemoteClient(baseURL, client, headers)
	if err != nil {
		return nil, err
	}
	return &openAIEmbedder{remote: remote, model: model, dimensions: dimensions}, nil
}

func (e *openAIEmbedder) Embed(ctx context.Context, text string) ([]float32, error) {
	body := map[string]any{"model": e.model, "input": text}
	var response struct {
		Data []struct {
			Embedding []float32 `json:"embedding"`
		} `json:"data"`
	}
	if err := e.remote.doJSON(ctx, http.MethodPost, "/embeddings", body, &response); err != nil {
		return nil, err
	}
	if len(response.Data) != 1 {
		return nil, fmt.Errorf("embedding provider returned %d vectors", len(response.Data))
	}
	if len(response.Data[0].Embedding) != e.dimensions {
		return nil, fmt.Errorf("embedding provider returned %d dimensions, want %d", len(response.Data[0].Embedding), e.dimensions)
	}
	return response.Data[0].Embedding, nil
}

func vectorLiteral(vector []float32) string {
	parts := make([]string, len(vector))
	for index, value := range vector {
		parts[index] = strconv.FormatFloat(float64(value), 'g', -1, 32)
	}
	return "[" + strings.Join(parts, ",") + "]"
}
