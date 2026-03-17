package runner

import (
	"context"
	_ "embed"
	"fmt"
	"log/slog"
	"time"

	coredom "github.com/shouni/gemini-reviewer-core/pkg/domain"
	coreprompts "github.com/shouni/gemini-reviewer-core/pkg/prompts"

	"git-gemini-web/internal/domain"
)

// ----------------------------------------------------------------------
// テンプレート埋め込みと初期化
// ----------------------------------------------------------------------

const (
	skipReport  = "skip_report"
	errorReport = "error_report"
)

//go:embed prompts/skip_report.md
var skipReportTemplate string

//go:embed prompts/error_report.md
var errorReportTemplate string

// reportBuilder は、レポート生成プロンプトの構築と管理に使用される PromptBuilder のインスタンスです。
var reportBuilder coredom.PromptBuilder

// reportData は、エラーレポートやスキップレポートのテンプレートに渡すデータを集約するための内部構造体です。
type reportData struct {
	StepName        string
	ErrorMessage    string
	DurationSeconds float64
	RepoURL         string
	BaseBranch      string
	FeatureBranch   string
}

// init 関数でテンプレートを一度だけパースし、エラーを捕捉します。
// 同じパッケージ内のどのファイルに書いても、init() はパッケージロード時に実行されます。
func init() {
	var err error
	templates := map[string]string{
		skipReport:  skipReportTemplate,
		errorReport: errorReportTemplate,
	}
	reportBuilder, err = coreprompts.NewBuilder(templates)
	if err != nil {
		panic(fmt.Sprintf("Failed to parse report templates: %v", err))
	}
}

// ----------------------------------------------------------------------
// ヘルパー関数 (Markdownレポート実行)
// ----------------------------------------------------------------------
// generateErrorReport はエラー発生時にユーザーに提示するMarkdownレポートを生成します。
func generateErrorReport(
	ctx context.Context,
	originalErr error,
	req domain.ReviewRequest,
	duration time.Duration,
	stepName string,
) (string, error) {
	// テンプレートを使用してリッチなMarkdownを生成
	markdown, buildErr := executeErrorMarkdown(originalErr, req, duration, stepName)
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

// executeSkipMarkdown は埋め込まれたテンプレートからスキップメッセージを生成します。
func executeSkipMarkdown(req domain.ReviewRequest) (string, error) {
	data := reportData{
		BaseBranch:    req.BaseBranch,
		FeatureBranch: req.FeatureBranch,
	}
	prompt, err := reportBuilder.Build(skipReport, data)
	if err != nil {
		return "", fmt.Errorf("スキップテンプレートの実行に失敗: %w", err)
	}
	return prompt, nil
}

// executeErrorMarkdown は埋め込まれたテンプレートからエラーレポートを生成します。
func executeErrorMarkdown(err error, req domain.ReviewRequest, duration time.Duration, stepName string) (string, error) {
	data := reportData{
		StepName:        stepName,
		ErrorMessage:    err.Error(),
		DurationSeconds: duration.Seconds(),
		RepoURL:         req.RepoURL,
		BaseBranch:      req.BaseBranch,
		FeatureBranch:   req.FeatureBranch,
	}
	prompt, err := reportBuilder.Build(errorReport, data)
	if err != nil {
		return "", fmt.Errorf("エラーテンプレートの実行に失敗: %w", err)
	}
	return prompt, nil
}
