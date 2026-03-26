package adapters

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/shouni/go-prompt-kit/prompts"

	"git-gemini-web/assets"
	"git-gemini-web/internal/domain"
)

const (
	skipReport  = "skip"
	errorReport = "error"
)

// reviewData はレビュープロンプトのテンプレートに渡すデータ構造です。
type reviewData struct {
	DiffContent string
}

// reportData は、エラーレポートやスキップレポートのテンプレートに渡すデータを集約するための内部構造体です。
type reportData struct {
	StepName        string
	ErrorMessage    string
	DurationSeconds float64
	RepoURL         string
	BaseBranch      string
	FeatureBranch   string
}

type PromptAdapter struct {
	promptBuilder *prompts.Builder
}

// NewPromptAdapter は動的に読み込んだテンプレートを使用して Builder を構築します。
func NewPromptAdapter() (*PromptAdapter, error) {
	templates, err := assets.LoadPrompts()
	if err != nil {
		return nil, err
	}
	b, err := prompts.NewBuilder(templates)
	if err != nil {
		return nil, err
	}

	return &PromptAdapter{
		promptBuilder: b,
	}, nil
}

// GenerateReview はコードレビューのMarkdownレポートを生成します。
func (pa *PromptAdapter) GenerateReview(mode, codeDiff string) (string, error) {
	data := reviewData{
		DiffContent: codeDiff,
	}

	prompt, err := pa.promptBuilder.Build(mode, data)
	if err != nil {
		return "", fmt.Errorf("スキップテンプレートの実行に失敗: %w", err)
	}
	return prompt, nil
}

// GenerateErrorReport はエラー発生時にユーザーに提示するMarkdownレポートを生成します。
func (pa *PromptAdapter) GenerateErrorReport(
	ctx context.Context,
	originalErr error,
	req domain.ReviewRequest,
	duration time.Duration,
	stepName string,
) (string, error) {
	// テンプレートを使用してリッチなMarkdownを生成
	markdown, buildErr := pa.executeErrorMarkdown(originalErr, req, duration, stepName)
	if buildErr != nil {
		slog.ErrorContext(ctx, "致命的エラー: エラーレポートMarkdownの生成に失敗しました", "error", buildErr)

		// フォールバック用のプレーンテキストレポート
		fallbackMarkdown := fmt.Sprintf(
			"## ❌ AIコードレビュー: 処理失敗レポート (生成失敗)\n\n"+
				"パイプラインの実行中にエラーが発生し、詳細レポートの生成にも失敗しました。\n\n"+
				"- **発生ステップ:** %s\n"+
				"- **元のエラー:** `%v`\n"+
				"- **レポート生成エラー:** `%v`",
			stepName, originalErr, buildErr,
		)
		return fallbackMarkdown, fmt.Errorf("エラーレポート生成失敗: %w", buildErr)
	}
	return markdown, originalErr
}

// ExecuteSkipMarkdown は埋め込まれたテンプレートからスキップメッセージを生成します。
func (pa *PromptAdapter) ExecuteSkipMarkdown(req domain.ReviewRequest) (string, error) {
	data := reportData{
		BaseBranch:    req.BaseBranch,
		FeatureBranch: req.FeatureBranch,
	}
	prompt, err := pa.promptBuilder.Build(skipReport, data)
	if err != nil {
		return "", fmt.Errorf("スキップテンプレートの実行に失敗: %w", err)
	}
	return prompt, nil
}

// executeErrorMarkdown は埋め込まれたテンプレートからエラーレポートを生成します。
func (pa *PromptAdapter) executeErrorMarkdown(err error, req domain.ReviewRequest, duration time.Duration, stepName string) (string, error) {
	data := reportData{
		StepName:        stepName,
		ErrorMessage:    err.Error(),
		DurationSeconds: duration.Seconds(),
		RepoURL:         req.RepoURL,
		BaseBranch:      req.BaseBranch,
		FeatureBranch:   req.FeatureBranch,
	}
	prompt, err := pa.promptBuilder.Build(errorReport, data)
	if err != nil {
		return "", fmt.Errorf("エラーテンプレートの実行に失敗: %w", err)
	}
	return prompt, nil
}
