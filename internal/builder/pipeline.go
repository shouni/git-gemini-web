package builder

import (
	"context"
	"fmt"

	"github.com/shouni/go-review-kit/pipeline"

	"github.com/shouni/git-gemini-web/internal/adapters"
	"github.com/shouni/git-gemini-web/internal/app"
	"github.com/shouni/git-gemini-web/internal/domain"
)

// buildPipeline は、実行可能な domain.Pipeline を返します。
func buildPipeline(ctx context.Context, appCtx *app.Container) (domain.Pipeline, error) {
	sources, err := adapters.NewDiffSourceFactory(appCtx.Config.SSHKeyPath)
	if err != nil {
		return nil, err
	}

	reviewer, err := adapters.NewReviewer(ctx, appCtx.Config)
	if err != nil {
		return nil, err
	}

	publisher, err := adapters.NewReportPublisher(appCtx.RemoteIO.Writer, appCtx.Layout)
	if err != nil {
		return nil, fmt.Errorf("ReportPublisher の構築に失敗しました: %w", err)
	}

	core, err := pipeline.New(pipeline.Deps{
		Sources:   sources,
		Prompts:   appCtx.PromptGen,
		Reviewer:  reviewer,
		Publisher: publisher,
		Notifier:  appCtx.Notifier,
	})
	if err != nil {
		return nil, fmt.Errorf("パイプラインの構築に失敗しました: %w", err)
	}

	return adapters.NewReviewPipeline(core, appCtx.StatusStore, appCtx.Config.PipelineTimeout), nil
}
