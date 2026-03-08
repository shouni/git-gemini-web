package adapters

import (
	"context"
	"fmt"

	coreAdapters "github.com/shouni/gemini-reviewer-core/pkg/adapters"
	"github.com/shouni/gemini-reviewer-core/pkg/domain"
)

// NewCodeReviewAI は adapters.CodeReviewAI のインスタンスを構築します。
func NewCodeReviewAI(ctx context.Context, geminiModel string) (domain.CodeReviewAI, error) {
	codeReviewAI, err := coreAdapters.NewGeminiAdapter(ctx, geminiModel)
	if err != nil {
		return nil, fmt.Errorf("CodeReviewAIアダプターの構築に失敗しました: %w", err)
	}
	return codeReviewAI, nil
}
