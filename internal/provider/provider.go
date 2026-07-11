package provider

import "context"

type GenerateRequest struct {
	Model         string
	System        string
	Prompt        string
	ContextPacket string
	MaxTokens     int
}

type GenerateResponse struct {
	Output      string
	RawArtifact []byte
	Model       string
}

type Provider interface {
	Generate(ctx context.Context, req GenerateRequest) (GenerateResponse, error)
}
