package pipeline

import (
	"context"
	"fmt"
	"log/slog"

	"git-gemini-web/internal/app"
	"git-gemini-web/internal/builder"
	"git-gemini-web/internal/domain"
	"git-gemini-web/internal/runner"
)

// ReviewPipeline はパイプラインの実行に必要な外部依存関係を保持するサービス構造体です。
type ReviewPipeline struct {
	reviewRunner  runner.ReviewRunner
	publishRunner runner.PublisherRunner
}

// NewReviewPipeline は ReviewPipeline の新しいインスタンスを生成します。
func NewReviewPipeline(ctx context.Context, appCtx *app.Container) (*ReviewPipeline, error) {
	reviewRunner, err := builder.BuildReviewRunner(ctx, appCtx.Config)
	if err != nil {
		return nil, fmt.Errorf("ReviewRunnerの構築に失敗: %w", err)
	}

	publishRunner, err := builder.BuildPublishRunner(ctx, appCtx)
	if err != nil {
		return nil, fmt.Errorf("PublishRunnerの構築に失敗: %w", err)
	}

	return &ReviewPipeline{
		reviewRunner:  reviewRunner,
		publishRunner: publishRunner,
	}, nil
}

// Execute はレビューリクエストの全工程（実行から公開まで）をオーケストレートします。
func (p *ReviewPipeline) Execute(ctx context.Context, payload domain.ReviewRequest) error {
	// 1. レビュー実行（中間結果 Outcome を取得）
	outcome := p.reviewRunner.Run(ctx, payload)

	// 2. 結果のパブリッシュ（GCS保存、Slack通知、エラーレポート生成など）
	// Outcome 内にエラーが含まれていても、publishRunner が適切に処理して error を返します。
	result, err := p.publishRunner.Run(ctx, payload, outcome)
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
