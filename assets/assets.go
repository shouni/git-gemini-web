package assets

import (
	"embed"
)

var (
	// DetailPrompt は詳細レビュー用のプロンプトテンプレートです。
	//go:embed prompts/prompt_detail.md
	DetailPrompt string

	// ReleasePrompt はリリース判定用のプロンプトテンプレートです。
	//go:embed prompts/prompt_release.md
	ReleasePrompt string

	// SkipReportTemplate はスキップ時のレポートテンプレートです。
	//go:embed prompts/skip_report.md
	SkipReportTemplate string

	// ErrorReportTemplate はエラー発生時のレポートテンプレートです。
	//go:embed prompts/error_report.md
	ErrorReportTemplate string

	// Templates は、HTMLテンプレートです。
	//go:embed templates/*.html
	Templates embed.FS
)
