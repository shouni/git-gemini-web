// Package adapters は、Gemini / Git / Slack / ストレージのクライアントを
// go-review-kit のポート実装として提供します。
package adapters

import (
	"context"
	"fmt"

	"github.com/shouni/go-review-kit/gemini"
	"github.com/shouni/go-review-kit/review"

	"github.com/shouni/git-gemini-web/internal/config"
)

// NewReviewer は review.Reviewer のインスタンスを構築します。
func NewReviewer(ctx context.Context, cfg *config.Config) (review.Reviewer, error) {
	reviewer, err := gemini.New(ctx, gemini.Options{
		ProjectID: cfg.ProjectID,
	})
	if err != nil {
		return nil, fmt.Errorf("レビュアーアダプターの構築に失敗しました: %w", err)
	}
	return reviewer, nil
}
