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
	// promptFiles はプロンプトテンプレートです。
	//go:embed prompts/prompt_*.md
	promptFiles embed.FS

	// reportFiles はレポートテンプレートです。
	//go:embed prompts/report_*.md
	reportFiles embed.FS

	// Templates は、HTMLテンプレートです。
	//go:embed templates/*.html
	Templates embed.FS
)

// LoadPrompts は埋め込まれたプロンプトファイルを読み込みます。
func LoadPrompts() (map[string]string, error) {
	return resource.Load(promptFiles, promptDir, promptPrefix)
}

// LoadReports は埋め込まれたレポートファイルを読み込みます。
func LoadReports() (map[string]string, error) {
	return resource.Load(reportFiles, promptDir, reportPrefix)
}
