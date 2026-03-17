package assets

import _ "embed"

var (
	//go:embed prompts/prompt_detail.md
	DetailPrompt string
	//go:embed prompts/prompt_release.md
	ReleasePrompt string
	//go:embed prompts/skip_report.md
	SkipReportTemplate string
	//go:embed prompts/error_report.md
	ErrorReportTemplate string
)
