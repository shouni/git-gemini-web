package builder

import (
	"context"
	"fmt"

	coreports "github.com/shouni/gemini-reviewer-core/ports"
	"github.com/shouni/gemini-reviewer-core/publisher"
	"github.com/shouni/gemini-reviewer-core/runner"
	"github.com/shouni/gemini-reviewer-core/workflow"
	"github.com/shouni/go-remote-io/remoteio"

	"github.com/shouni/git-gemini-web/internal/adapters"
	"github.com/shouni/git-gemini-web/internal/app"
	"github.com/shouni/git-gemini-web/internal/config"
	"github.com/shouni/git-gemini-web/internal/domain"
)

// buildPipeline は、実行可能な domain.Pipeline を返します。
func buildPipeline(
	ctx context.Context,
	appCtx *app.Container,
) (domain.Pipeline, error) {
	reviewRunner, err := buildReviewRunner(ctx, appCtx.Config, appCtx.PromptGen)
	if err != nil {
		return nil, fmt.Errorf("ReviewRunnerの構築に失敗: %w", err)
	}

	publishRunner, err := buildPublishRunner(appCtx.RemoteIO.Writer, appCtx.Notifier)
	if err != nil {
		return nil, fmt.Errorf("PublishRunnerの構築に失敗: %w", err)
	}

	coreWorkflow := workflow.New(reviewRunner, publishRunner)
	return adapters.NewCoreWorkflowAdapter(coreWorkflow, appCtx.Config.PipelineTimeout), nil
}

// buildReviewRunner は、実行可能な ports.ReviewRunner のインターフェースを返します。
func buildReviewRunner(
	ctx context.Context,
	cfg *config.Config,
	promptGen coreports.PromptGenerator,
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
	writer remoteio.Writer,
	notifier coreports.Notifier,
) (*runner.PublishRunner, error) {
	converter, err := publisher.NewConverterAdapter()
	if err != nil {
		return nil, fmt.Errorf("converterの初期化に失敗しました: %w", err)
	}
	publishService, err := publisher.New(writer, converter)
	if err != nil {
		return nil, fmt.Errorf("publisherの初期化に失敗しました: %w", err)
	}

	publishRunner := runner.NewPublishRunner(
		publishService,
		notifier,
	)

	return publishRunner, nil
}
