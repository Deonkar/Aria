package ai

import (
	"context"
	"fmt"

	"github.com/openai/openai-go"
	"github.com/openai/openai-go/packages/param"
)

func EmbedText(ctx context.Context, client openai.Client, model, text string) ([]float32, int, int, error) {
	if text == "" {
		return nil, 0, 0, fmt.Errorf("empty embed text")
	}
	resp, err := client.Embeddings.New(ctx, openai.EmbeddingNewParams{
		Model: openai.EmbeddingModel(model),
		Input: openai.EmbeddingNewParamsInputUnion{
			OfString: param.NewOpt(text),
		},
	})
	if err != nil {
		return nil, 0, 0, err
	}
	if len(resp.Data) == 0 {
		return nil, 0, 0, fmt.Errorf("no embedding returned")
	}
	vec64 := resp.Data[0].Embedding
	out := make([]float32, len(vec64))
	for i, v := range vec64 {
		out[i] = float32(v)
	}
	promptTok := int(resp.Usage.PromptTokens)
	totalTok := int(resp.Usage.TotalTokens)
	return out, promptTok, totalTok, nil
}
