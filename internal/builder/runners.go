package builder

import (
	"context"
	"fmt"
	"log/slog"

	"git-gemini-web/internal/app"
	"git-gemini-web/internal/config"
	"git-gemini-web/internal/pipeline"
	"git-gemini-web/internal/runner"

	"github.com/shouni/gemini-reviewer-core/pkg/adapters"
	"github.com/shouni/gemini-reviewer-core/pkg/prompts"
	"github.com/shouni/gemini-reviewer-core/pkg/publisher"
)

// GitAdapterFactoryImpl は、runner.GitAdapterFactory インターフェースを満たす具象型です。
type GitAdapterFactoryImpl struct {
	sshKeyPath       string
	skipHostKeyCheck bool
}

// Create は runner.GitAdapterFactory インターフェースを満たします。
func (f *GitAdapterFactoryImpl) Create(localPath string, baseBranch string) adapters.GitService {
	skipHostKeyCheckOption := adapters.WithInsecureSkipHostKeyCheck(f.skipHostKeyCheck)
	baseBranchOption := adapters.WithBaseBranch(baseBranch)

	return adapters.NewGitAdapter(
		localPath,
		f.sshKeyPath,
		skipHostKeyCheckOption,
		baseBranchOption,
	)
}

// BuildPipeline は ReviewPipeline の新しいインスタンスを生成します。
func BuildPipeline(ctx context.Context, appCtx *app.Container) (pipeline.Pipeline, error) {
	reviewRunner, err := buildReviewRunner(ctx, appCtx.Config)
	if err != nil {
		return nil, fmt.Errorf("ReviewRunnerの構築に失敗: %w", err)
	}

	publishRunner, err := buildPublishRunner(ctx, appCtx)
	if err != nil {
		return nil, fmt.Errorf("PublishRunnerの構築に失敗: %w", err)
	}

	return &pipeline.ReviewPipeline{
		ReviewRunner:  reviewRunner,
		PublishRunner: publishRunner,
	}, nil
}

// buildReviewRunner は、Web Runner の main.go から呼び出され、
// 実行可能な Runner のインターフェースを返します。
func buildReviewRunner(
	ctx context.Context,
	cfg *config.Config,
) (runner.ReviewRunner, error) {

	// 1. Git Factory の構築
	gitFactory := &GitAdapterFactoryImpl{
		sshKeyPath:       cfg.SSHKeyPath,
		skipHostKeyCheck: cfg.SkipHostKeyCheck,
	}
	slog.Debug("GitAdapterFactory を構築しました。", "ssh_path_set", cfg.SSHKeyPath != "")

	// 2. GeminiService (Adapter) の構築
	geminiService, err := adapters.NewGeminiAdapter(ctx, cfg.GeminiModel)
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
	appCtx *app.Container,
) (runner.PublisherRunner, error) {

	// Publisher の構築
	htmlRunner, err := publisher.NewMarkdownToHtmlRunner(ctx)
	if err != nil {
		return nil, fmt.Errorf("MarkdownToHtmlRunnerの初期化に失敗しました: %w", err)
	}
	publisherService, err := publisher.NewPublisher(ctx, appCtx.RemoteIO.Factory, htmlRunner)
	if err != nil {
		return nil, fmt.Errorf("Publisherの初期化に失敗しました: %w", err)
	}
	slog.Debug("Publisher を構築しました。")

	// 依存関係を注入して Runner を組み立てる
	publishRunner := runner.NewStoragePublisherRunner(
		publisherService,
		appCtx.RemoteIO.Signer,
		appCtx.SlackNotifier,
	)

	slog.Debug("PublishRunner の構築が完了しました。")
	return publishRunner, nil
}
