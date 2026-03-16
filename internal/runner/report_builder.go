package runner

import (
	"bytes"
	"context"
	_ "embed"
	"fmt"
	"log/slog"
	"text/template"
	"time"

	"git-gemini-web/internal/domain"
)

// ----------------------------------------------------------------------
// テンプレート埋め込みと初期化
// ----------------------------------------------------------------------

//go:embed prompts/skip_report.md
var skipReportTemplate string

//go:embed prompts/error_report.md
var errorReportTemplate string

var skipTpl *template.Template
var errorTpl *template.Template

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

	// スキップレポートテンプレートのパース
	skipTpl, err = template.New("skip_report").Parse(skipReportTemplate)
	if err != nil {
		panic(fmt.Sprintf("Failed to parse skip report template: %v", err))
	}

	// エラーレポートテンプレートのパース
	errorTpl, err = template.New("error_report").Parse(errorReportTemplate)
	if err != nil {
		panic(fmt.Sprintf("Failed to parse error report template: %v", err))
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
	var buf bytes.Buffer
	if err := skipTpl.Execute(&buf, data); err != nil {
		return "", fmt.Errorf("スキップテンプレートの実行に失敗: %w", err)
	}
	return buf.String(), nil
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
	var buf bytes.Buffer
	if err := errorTpl.Execute(&buf, data); err != nil {
		return "", fmt.Errorf("エラーテンプレートの実行に失敗: %w", err)
	}
	return buf.String(), nil
}
