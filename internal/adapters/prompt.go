package adapters

import (
	"context"
	"fmt"
	"log/slog"

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

// promptBuilder は、フォーマット済みのプロンプトを作成するためのインターフェース
type promptBuilder interface {
	Build(mode string, data any) (string, error)
}

// PromptAdapter は、さまざまなモードとデータに基づいてプロンプトを生成する役割を担います。
type PromptAdapter struct {
	reviewMode promptBuilder
	report     promptBuilder
}

// NewPromptAdapter は動的に読み込んだテンプレートを使用して Builder を構築します。
func NewPromptAdapter() (*PromptAdapter, error) {
	reviewTemplates, err := assets.LoadPrompts()
	if err != nil {
		return nil, fmt.Errorf("レビューテンプレートの読み込みに失敗: %w", err)
	}
	reportTemplates, err := assets.LoadReports()
	if err != nil {
		return nil, fmt.Errorf("レポートテンプレートの読み込みに失敗: %w", err)
	}

	reviewMode, err := prompts.NewBuilder(reviewTemplates)
	if err != nil {
		return nil, fmt.Errorf("レビュービルダーの構築に失敗: %w", err)
	}
	report, err := prompts.NewBuilder(reportTemplates)
	if err != nil {
		return nil, fmt.Errorf("レポートビルダーの構築に失敗: %w", err)
	}

	return &PromptAdapter{
		reviewMode: reviewMode,
		report:     report,
	}, nil
}

// GenerateReview はコードレビューのMarkdownレポートを生成します。
func (pa *PromptAdapter) GenerateReview(mode, codeDiff string) (string, error) {
	data := reviewData{
		DiffContent: codeDiff,
	}
	prompt, err := pa.reviewMode.Build(mode, data)
	if err != nil {
		return "", fmt.Errorf("レビューテンプレートの実行に失敗: %w", err)
	}
	return prompt, nil
}

// GenerateErrorReport はエラー発生時にユーザーに提示するMarkdownレポートを生成します。
func (pa *PromptAdapter) GenerateErrorReport(
	ctx context.Context,
	params domain.ErrorReportParams,
) (string, error) {
	// テンプレートを使用してリッチなMarkdownを生成
	markdown, buildErr := pa.executeErrorMarkdown(params)
	if buildErr != nil {
		slog.ErrorContext(ctx, "致命的エラー: エラーレポートMarkdownの生成に失敗しました", "error", buildErr)

		// フォールバック用のプレーンテキストレポート
		fallbackMarkdown := fmt.Sprintf(
			"## ❌ AIコードレビュー: 処理失敗レポート (生成失敗)\n\n"+
				"パイプラインの実行中にエラーが発生し、詳細レポートの生成にも失敗しました。\n\n"+
				"- **発生ステップ:** %s\n"+
				"- **元のエラー:** `%v`\n"+
				"- **レポート生成エラー:** `%v`",
			params.StepName, params.OriginalErr, buildErr,
		)
		return fallbackMarkdown, fmt.Errorf("エラーレポート生成失敗: %w", buildErr)
	}
	// 成功時はレポート生成エラーなしとして nil を返す
	return markdown, nil
}

// GenerateSkipReport は埋め込まれたテンプレートからスキップメッセージを生成します。
func (pa *PromptAdapter) GenerateSkipReport(req domain.ReviewRequest) (string, error) {
	data := reportData{
		BaseBranch:    req.BaseBranch,
		FeatureBranch: req.FeatureBranch,
	}
	prompt, err := pa.report.Build(skipReport, data)
	if err != nil {
		return "", fmt.Errorf("スキップテンプレートの実行に失敗: %w", err)
	}
	return prompt, nil
}

// executeErrorMarkdown は埋め込まれたテンプレートからエラーレポートを生成します。
func (pa *PromptAdapter) executeErrorMarkdown(params domain.ErrorReportParams) (string, error) {
	data := reportData{
		StepName:        params.StepName,
		ErrorMessage:    params.OriginalErr.Error(),
		DurationSeconds: params.Duration.Seconds(),
		RepoURL:         params.Req.RepoURL,
		BaseBranch:      params.Req.BaseBranch,
		FeatureBranch:   params.Req.FeatureBranch,
	}
	prompt, err := pa.report.Build(errorReport, data)
	if err != nil {
		return "", fmt.Errorf("エラーテンプレートの実行に失敗: %w", err)
	}
	return prompt, nil
}
