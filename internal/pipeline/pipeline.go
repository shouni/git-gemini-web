package pipeline

import (
	"context"
	"fmt"
	"log/slog"

	"git-gemini-web/internal/domain"
)

// ReviewPipeline はパイプラインの実行に必要な外部依存関係を保持するサービス構造体です。
type ReviewPipeline struct {
	reviewer  domain.ReviewRunner
	publisher domain.PublishRunner
}

func NewReviewPipeline(reviewer domain.ReviewRunner, publisher domain.PublishRunner) *ReviewPipeline {
	return &ReviewPipeline{
		reviewer:  reviewer,
		publisher: publisher,
	}
}

// Execute はレビューリクエストの全工程（実行から公開まで）をオーケストレートします。
func (p *ReviewPipeline) Execute(ctx context.Context, payload domain.ReviewRequest) error {
	// 1. レビュー実行（中間結果 Outcome を取得）
	outcome := p.reviewer.Run(ctx, payload)

	// 2. 結果のパブリッシュ（GCS保存、Slack通知、エラーレポート生成など）
	// Outcome 内にエラーが含まれていても、publisher が適切に処理して error を返します。
	result, err := p.publisher.Run(ctx, payload, outcome)
	if err != nil {
		return fmt.Errorf("publish runner execution failed for repo %s: %w", payload.RepoURL, err)
	}

	// 3. 正常終了のログ記録（ステータスが Success または Skipped の場合）
	slog.InfoContext(ctx, "Review pipeline completed successfully.",
		"repo_url", payload.RepoURL,
		"status", result.Status,
		"gcs_uri", result.GCSURI,
	)

	return nil
}
