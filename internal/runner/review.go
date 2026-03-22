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
	codeDiff, gitStep, err := r.gitFactory.CloneAndDiff(ctx, gitService, req.RepoURL, req.BaseBranch, req.FeatureBranch)
	if err != nil {
		outcome.StepName = gitStep
		outcome.Error = err
		return outcome
	}

	// 2. 差分がない場合のスキップ処理
	if len(codeDiff) == 0 {
		outcome.StepName = "差分チェック"
		outcome.IsSkipped = true
		markdown, err := executeSkipMarkdown(req)
		outcome.ReviewMarkdown = markdown
		outcome.Error = err // 生成失敗の可能性も含める
		return outcome
	}

	// 3. AIによるレビュー生成
	outcome.StepName = "Gemini API呼び出し"
	markdown, err := r.executeAIReview(ctx, req.Mode, codeDiff, req.ModelName)
	outcome.ReviewMarkdown = markdown
	outcome.Error = err

	return outcome
}

// --- 内部補助メソッド ---

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
