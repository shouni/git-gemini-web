package builder

import (
	"context"
	"fmt"

	"github.com/shouni/gemini-reviewer-core/ports"
	"github.com/shouni/gemini-reviewer-core/publisher"
	"github.com/shouni/gemini-reviewer-core/runner"
	"github.com/shouni/gemini-reviewer-core/workflow"
	"github.com/shouni/go-remote-io/remoteio"

	"git-gemini-web/internal/adapters"
	"git-gemini-web/internal/app"
	"git-gemini-web/internal/config"
)

// buildPipeline は、実行可能な domain.Pipeline を返します。
func buildPipeline(
	ctx context.Context,
	appCtx *app.Container,
) (*workflow.Workflow, error) {
	reviewRunner, err := buildReviewRunner(ctx, appCtx.Config, appCtx.PromptGen)
	if err != nil {
		return nil, fmt.Errorf("ReviewRunnerの構築に失敗: %w", err)
	}

	publishRunner, err := buildPublishRunner(appCtx.PromptGen, appCtx.RemoteIO.Writer, appCtx.RemoteIO.Signer, appCtx.Notifier)
	if err != nil {
		return nil, fmt.Errorf("PublishRunnerの構築に失敗: %w", err)
	}

	return workflow.New(reviewRunner, publishRunner), nil
}

// buildReviewRunner は、実行可能な ports.ReviewRunner のインターフェースを返します。
func buildReviewRunner(
	ctx context.Context,
	cfg *config.Config,
	promptGen ports.PromptGenerator,
) (*runner.ReviewRunner, error) {
	gitFactory := adapters.NewGitFactory(cfg)
	codeReviewAI, err := adapters.NewCodeReviewAI(ctx, cfg)
	if err != nil {
		return nil, err
	}

	reviewRunner := runner.NewReviewRunner(
		promptGen,
		gitFactory,
		codeReviewAI,
	)

	return reviewRunner, nil
}

// buildPublishRunner は、実行可能な ports.PublishRunner のインターフェースを返します。
func buildPublishRunner(
	promptGen ports.PromptGenerator,
	writer remoteio.OutputWriter,
	signer remoteio.URLSigner,
	notifier ports.Notifier,
) (*runner.PublishRunner, error) {
	converter, err := publisher.NewConverterAdapter()
	if err != nil {
		return nil, fmt.Errorf("MarkdownToHtmlRunnerの初期化に失敗しました: %w", err)
	}
	publishService, err := publisher.New(writer, converter)
	if err != nil {
		return nil, fmt.Errorf("Publisherの初期化に失敗しました: %w", err)
	}

	publishRunner := runner.NewPublishRunner(
		promptGen,
		publishService,
		notifier,
		signer,
	)

	return publishRunner, nil
}
