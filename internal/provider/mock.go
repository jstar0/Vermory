package provider

import (
	"context"
	"encoding/json"
	"strings"
)

type Mock struct {
	Output string
}

func (m Mock) Generate(_ context.Context, req GenerateRequest) (GenerateResponse, error) {
	output := strings.TrimSpace(m.Output)
	if output == "" {
		output = req.Prompt
		if strings.TrimSpace(req.ContextPacket) != "" {
			output += "\n" + req.ContextPacket
		}
	}
	raw, _ := json.Marshal(map[string]string{
		"provider": "mock",
		"model":    req.Model,
		"output":   output,
	})
	return GenerateResponse{
		Output:      output,
		RawArtifact: raw,
		Model:       req.Model,
	}, nil
}
