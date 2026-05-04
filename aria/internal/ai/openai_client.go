package ai

import (
	"github.com/Deonkar/Aria/aria/internal/config"
	"github.com/openai/openai-go"
	"github.com/openai/openai-go/option"
)

// NewOpenAIClient builds a client; optional OPENAI_BASE_URL / custom header are applied from cfg.
func NewOpenAIClient(cfg *config.Config) openai.Client {
	var opts []option.RequestOption
	if cfg.OpenAIAPIKeyHeader != "" {
		opts = append(opts, option.WithHeader(cfg.OpenAIAPIKeyHeader, cfg.OpenAIAPIKey))
	} else {
		opts = append(opts, option.WithAPIKey(cfg.OpenAIAPIKey))
	}
	if cfg.OpenAIBaseURL != "" {
		opts = append(opts, option.WithBaseURL(cfg.OpenAIBaseURL))
	}
	return openai.NewClient(opts...)
}
