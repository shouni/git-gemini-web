package builder

import (
	"context"
	"fmt"

	"github.com/shouni/gemini-reviewer-core/publisher"

	"git-gemini-web/internal/adapters"
	"git-gemini-web/internal/app"
	"git-gemini-web/internal/config"
	"git-gemini-web/internal/domain"
	"git-gemini-web/internal/pipeline"
	"git-gemini-web/internal/runner"
)

// buildPipeline は ReviewPipeline の新しいインスタンスを生成します。
func buildPipeline(ctx context.Context, cfg *config.Config, rio *app.RemoteIO, slack domain.Notifier) (domain.Pipeline, error) {
	reviewRunner, err := buildReviewRunner(ctx, cfg)
	if err != nil {
		return nil, fmt.Errorf("ReviewRunnerの構築に失敗: %w", err)
	}

	publishRunner, err := buildPublishRunner(ctx, rio, slack)
	if err != nil {
		return nil, fmt.Errorf("PublishRunnerの構築に失敗: %w", err)
	}

	return pipeline.NewReviewPipeline(reviewRunner, publishRunner), nil
}

// buildReviewRunner は、実行可能な ReviewRunner のインターフェースを返します。
func buildReviewRunner(
	ctx context.Context,
	cfg *config.Config,
) (domain.ReviewRunner, error) {
	// 1. Git Factory の構築
	gitFactory := adapters.NewGitFactory(cfg)

	// 2. codeReviewAI の構築
	codeReviewAI, err := adapters.NewCodeReviewAI(ctx, cfg)
	if err != nil {
		return nil, err
	}

	// 3. Prompt Builder の構築
	promptBuilder, err := adapters.NewPromptAdapter()
	if err != nil {
		return nil, fmt.Errorf("Prompt Builder の構築に失敗しました: %w", err)
	}

	// 4. 依存関係を注入して Runner を組み立てる
	reviewRunner := runner.NewReviewRunner(
		gitFactory,
		codeReviewAI,
		promptBuilder,
	)

	return reviewRunner, nil
}

// buildPublishRunner は、実行可能な PublisherRunner のインターフェースを返します。
func buildPublishRunner(
	ctx context.Context,
	rio *app.RemoteIO,
	slack domain.Notifier,
) (domain.PublisherRunner, error) {
	if rio == nil {
		return nil, fmt.Errorf("RemoteIO が設定されていません")
	}

	htmlRunner, err := publisher.NewMarkdownConverterAdapter()
	if err != nil {
		return nil, fmt.Errorf("MarkdownToHtmlRunnerの初期化に失敗しました: %w", err)
	}
	publisherService, err := publisher.NewPublisher(rio.Writer, htmlRunner)
	if err != nil {
		return nil, fmt.Errorf("Publisherの初期化に失敗しました: %w", err)
	}

	publishRunner := runner.NewPublisherRunner(
		publisherService,
		rio.Signer,
		slack,
	)

	return publishRunner, nil
}
