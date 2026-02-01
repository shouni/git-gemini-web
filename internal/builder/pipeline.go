package builder

import (
	"context"
	"fmt"
	"log/slog"

	"git-gemini-web/internal/adapters"
	"git-gemini-web/internal/app"
	"git-gemini-web/internal/config"
	"git-gemini-web/internal/pipeline"
	"git-gemini-web/internal/runner"

	core "github.com/shouni/gemini-reviewer-core/pkg/adapters"
	"github.com/shouni/gemini-reviewer-core/pkg/prompts"
	"github.com/shouni/gemini-reviewer-core/pkg/publisher"
)

// GitAdapterFactoryImpl は、runner.GitAdapterFactory インターフェースを満たす具象型です。
type GitAdapterFactoryImpl struct {
	sshKeyPath       string
	skipHostKeyCheck bool
}

// Create は runner.GitAdapterFactory インターフェースを満たします。
func (f *GitAdapterFactoryImpl) Create(localPath string, baseBranch string) core.GitService {
	skipHostKeyCheckOption := core.WithInsecureSkipHostKeyCheck(f.skipHostKeyCheck)
	baseBranchOption := core.WithBaseBranch(baseBranch)

	return core.NewGitAdapter(
		localPath,
		f.sshKeyPath,
		skipHostKeyCheckOption,
		baseBranchOption,
	)
}

// buildPipeline は ReviewPipeline の新しいインスタンスを生成します。
func buildPipeline(ctx context.Context, appCtx *app.Container) (pipeline.Pipeline, error) {
	reviewRunner, err := buildReviewRunner(ctx, appCtx.Config)
	if err != nil {
		return nil, fmt.Errorf("ReviewRunnerの構築に失敗: %w", err)
	}

	publishRunner, err := buildPublishRunner(ctx, appCtx.RemoteIO, appCtx.SlackNotifier)
	if err != nil {
		return nil, fmt.Errorf("PublishRunnerの構築に失敗: %w", err)
	}

	return pipeline.NewReviewPipeline(reviewRunner, publishRunner), nil
}

// buildReviewRunner は、実行可能な ReviewRunner のインターフェースを返します。
func buildReviewRunner(
	ctx context.Context,
	cfg *config.Config,
) (pipeline.ReviewRunner, error) {
	// 1. Git Factory の構築
	gitFactory := &GitAdapterFactoryImpl{
		sshKeyPath:       cfg.SSHKeyPath,
		skipHostKeyCheck: cfg.SkipHostKeyCheck,
	}
	slog.Debug("GitAdapterFactory を構築しました。", "ssh_path_set", cfg.SSHKeyPath != "")

	// 2. GeminiService (Adapter) の構築
	geminiService, err := core.NewGeminiAdapter(ctx, cfg.GeminiModel)
	if err != nil {
		return nil, fmt.Errorf("Gemini Service の構築に失敗しました: %w", err)
	}
	slog.Debug("GeminiService (Adapter) を構築しました。", "model", cfg.GeminiModel)

	// 3. Prompt Builder の構築
	promptBuilder, err := prompts.NewPromptBuilder()
	if err != nil {
		return nil, fmt.Errorf("Prompt Builder の構築に失敗しました: %w", err)
	}
	slog.Debug("PromptBuilderを構築しました。")

	// 4. 依存関係を注入して Runner を組み立てる
	reviewRunner := runner.NewCodeReviewRunner(
		gitFactory,
		geminiService,
		promptBuilder,
	)

	slog.Debug("ReviewRunner の構築が完了しました。")
	return reviewRunner, nil
}

// buildPublishRunner は、実行可能な PublisherRunner のインターフェースを返します。
func buildPublishRunner(
	ctx context.Context,
	rio *app.RemoteIO,
	slack adapters.SlackNotifier,
) (pipeline.PublisherRunner, error) {
	// Publisher の構築
	htmlRunner, err := publisher.NewMarkdownToHtmlRunner(ctx)
	if err != nil {
		return nil, fmt.Errorf("MarkdownToHtmlRunnerの初期化に失敗しました: %w", err)
	}
	publisherService, err := publisher.NewPublisher(ctx, rio.Factory, htmlRunner)
	if err != nil {
		return nil, fmt.Errorf("Publisherの初期化に失敗しました: %w", err)
	}
	slog.Debug("Publisher を構築しました。")

	// 依存関係を注入して Runner を組み立てる
	publishRunner := runner.NewStoragePublisherRunner(
		publisherService,
		rio.Signer,
		slack,
	)

	slog.Debug("PublishRunner の構築が完了しました。")
	return publishRunner, nil
}
