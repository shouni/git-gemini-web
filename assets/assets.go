package assets

import (
	"embed"

	"github.com/shouni/go-prompt-kit/resource"
)

const (
	promptDir    = "prompts"
	promptPrefix = "prompt_"
	reportPrefix = "report_"
)

var (
	// PromptFiles はプロントテンプレートです。
	//go:embed prompts/prompt_*.md
	PromptFiles embed.FS

	// ReportFiles はレポートテンプレートです。
	//go:embed prompts/report_*.md
	ReportFiles embed.FS

	// Templates は、HTMLテンプレートです。
	//go:embed templates/*.html
	Templates embed.FS
)

// LoadPrompts は埋め込まれたプロンプトファイルを読み込みます。
func LoadPrompts() (map[string]string, error) {
	return resource.Load(PromptFiles, promptDir, promptPrefix)
}

// LoadReports は埋め込まれたレポートファイルを読み込みます。
func LoadReports() (map[string]string, error) {
	return resource.Load(ReportFiles, promptDir, reportPrefix)
}
