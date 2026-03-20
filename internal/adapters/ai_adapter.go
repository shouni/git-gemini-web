package adapters

import (
	"context"
	"fmt"

	coreAdapters "github.com/shouni/gemini-reviewer-core/pkg/adapters"
	"github.com/shouni/gemini-reviewer-core/pkg/ports"

	"git-gemini-web/internal/config"
)

// NewCodeReviewAI は domain.CodeReviewAI のインスタンスを構築します。
func NewCodeReviewAI(ctx context.Context, cfg *config.Config) (ports.CodeReviewAI, error) {
	opt := coreAdapters.GeminiOptions{
		ProjectID: cfg.ProjectID,
	}
	codeReviewAI, err := coreAdapters.NewGeminiAdapter(ctx, opt)
	if err != nil {
		return nil, fmt.Errorf("CodeReviewAIアダプターの構築に失敗しました: %w", err)
	}
	return codeReviewAI, nil
}
