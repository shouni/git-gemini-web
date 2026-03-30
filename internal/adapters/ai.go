package adapters

import (
	"context"
	"fmt"

	"github.com/shouni/gemini-reviewer-core/ai"
	"github.com/shouni/gemini-reviewer-core/ports"

	"git-gemini-web/internal/config"
)

// NewCodeReviewAI は ports.CodeReviewAI のインスタンスを構築します。
func NewCodeReviewAI(ctx context.Context, cfg *config.Config) (ports.CodeReviewAI, error) {
	opt := ai.GeminiOptions{
		ProjectID: cfg.ProjectID,
	}
	codeReviewAI, err := ai.NewGeminiAdapter(ctx, opt)
	if err != nil {
		return nil, fmt.Errorf("CodeReviewAIアダプターの構築に失敗しました: %w", err)
	}
	return codeReviewAI, nil
}
