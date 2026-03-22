package runner

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/shouni/gemini-reviewer-core/ports"

	"git-gemini-web/internal/domain"
)

const (
	emptyAPIResponseMessage = "Gemini APIは応答しましたが、空の結果を返しました。"
)

// TemplateData はレビュープロンプトのテンプレートに渡すデータ構造です。
type TemplateData struct {
	DiffContent string
}

// ReviewRunner は domain.ReviewRunner インターフェースの実装です。
type ReviewRunner struct {
	gitFactory    domain.GitFactory
	codeReviewAI  ports.CodeReviewAI
	promptBuilder domain.PromptBuilder
}

// NewReviewRunner は ReviewRunner の新しいインスタンスを作成します。
func NewReviewRunner(
	gitFactory domain.GitFactory,
	codeReviewAI ports.CodeReviewAI,
	pb domain.PromptBuilder,
) *ReviewRunner {
	return &ReviewRunner{
		gitFactory:    gitFactory,
		codeReviewAI:  codeReviewAI,
		promptBuilder: pb,
	}
}

// Run はレビューのメインフローを実行します。
func (r *ReviewRunner) Run(ctx context.Context, req domain.ReviewRequest) domain.ReviewProcessOutcome {
	outcome := domain.ReviewProcessOutcome{
		StartTime: time.Now(),
	}

	// 1. Git リソースの生成, 差分の取得
	gitService := r.gitFactory.Create(req.RepoURL, req.BaseBranch)
	defer r.cleanupGit(ctx, gitService)

	// 2. リポジトリの準備
	outcome.StepName = "リポジトリの準備"
	if err := r.prepareRepository(ctx, gitService, req.RepoURL, req.FeatureBranch); err != nil {
		outcome.Error = err
		return outcome
	}

	// 3. 差分の取得
	outcome.StepName = "コード差分取得"
	codeDiff, err := gitService.GetCodeDiff(ctx, req.BaseBranch, req.FeatureBranch)
	if err != nil {
		outcome.Error = err
		return outcome
	}

	// 4. 差分がない場合のスキップ処理
	if len(codeDiff) == 0 {
		outcome.StepName = "差分チェック"
		outcome.IsSkipped = true
		markdown, err := executeSkipMarkdown(req)
		outcome.ReviewMarkdown = markdown
		outcome.Error = err // 生成失敗の可能性も含める
		return outcome
	}

	// 5. AIによるレビュー生成
	outcome.StepName = "Gemini API呼び出し"
	markdown, err := r.executeAIReview(ctx, req.Mode, codeDiff, req.ModelName)
	outcome.ReviewMarkdown = markdown
	outcome.Error = err

	return outcome
}

// --- 内部補助メソッド ---

// prepareRepository は、リポジトリを複製し、機能ブランチが存在するかどうかを確認します。
func (r *ReviewRunner) prepareRepository(ctx context.Context, git ports.GitService, repoURL, branch string) error {
	slog.InfoContext(ctx, "1. リポジトリをクローン/更新中", "repo_url", repoURL)
	if err := git.CloneOrUpdate(ctx, repoURL); err != nil {
		return fmt.Errorf("リポジトリの準備に失敗: %w", err)
	}

	slog.InfoContext(ctx, "2. フィーチャーブランチの存在を確認中", "branch", branch)
	exists, err := git.CheckRefExists(ctx, branch)
	if err != nil {
		return fmt.Errorf("ブランチ存在確認に失敗: %w", err)
	}
	if !exists {
		return fmt.Errorf("指定されたフィーチャーブランチ '%s' がリモートに存在しません。", branch)
	}
	return nil
}

// executeAIReview は、指定されたdiffとモードでプロンプトを生成し、AIによるコードレビューを実行します。
func (r *ReviewRunner) executeAIReview(ctx context.Context, mode, codeDiff, model string) (string, error) {
	slog.InfoContext(ctx, "AIプロンプトを生成・API呼び出し中", "mode", mode)
	data := TemplateData{
		DiffContent: codeDiff,
	}
	prompt, err := r.promptBuilder.Build(mode, data)
	if err != nil {
		return "", fmt.Errorf("プロンプト生成に失敗: %w", err)
	}

	content, err := r.codeReviewAI.ReviewCodeDiff(ctx, model, prompt)
	if err != nil {
		return "", fmt.Errorf("Gemini API呼び出しに失敗: %w", err)
	}

	if content == "" {
		slog.WarnContext(ctx, "Gemini API returned an empty response without error.")
		return emptyAPIResponseMessage, nil
	}
	return content, nil
}

// cleanupGit は、Git リソースのクリーンアップを処理し、クリーンアップ操作が失敗した場合は警告をログに記録します。
func (r *ReviewRunner) cleanupGit(ctx context.Context, git ports.GitService) {
	if err := git.Cleanup(ctx); err != nil {
		slog.WarnContext(ctx, "Gitリソースのクリーンアップに失敗しました", "error", err)
	}
}
