package memorybackend

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestOpenAIEmbedderUsesConfiguredModel(t *testing.T) {
	var request map[string]any
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/embeddings" {
			http.Error(w, "unexpected path", http.StatusNotFound)
			return
		}
		defer r.Body.Close()
		if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
			t.Fatal(err)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"data":[{"embedding":[0.1,0.2,0.3],"index":0}],"model":"bge-m3"}`))
	}))
	defer server.Close()

	embedder, err := newOpenAIEmbedder(server.URL, "test-key", "bge-m3", 3, server.Client())
	if err != nil {
		t.Fatal(err)
	}
	vector, err := embedder.Embed(context.Background(), "中文 mixed identifier checkout_eta_v2")
	if err != nil {
		t.Fatal(err)
	}
	if len(vector) != 3 || vector[2] != 0.3 {
		t.Fatalf("unexpected vector: %#v", vector)
	}
	if request["model"] != "bge-m3" || request["input"] != "中文 mixed identifier checkout_eta_v2" {
		t.Fatalf("unexpected embedding request: %#v", request)
	}
	if _, exists := request["dimensions"]; exists {
		t.Fatalf("fixed-dimension embedding request must not send optional dimensions: %#v", request)
	}
}

func TestVectorLiteral(t *testing.T) {
	if got := vectorLiteral([]float32{0.25, -1, 3.5}); got != "[0.25,-1,3.5]" {
		t.Fatalf("unexpected vector literal %q", got)
	}
}
