package assets

import (
	"embed"
	"sync"

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

	cachedPrompts map[string]string
	once          sync.Once
)

// LoadPrompts は埋め込まれたプロンプトファイルを読み込みます。
func LoadPrompts() (map[string]string, error) {
	return resource.Load(promptFiles, promptDir, promptPrefix)
}

// LoadReports は埋め込まれたレポートファイルを読み込みます。
func LoadReports() (map[string]string, error) {
	return resource.Load(reportFiles, promptDir, reportPrefix)
}

// IsValidMode は、指定されたモード名に対応するプロンプトファイルが存在するか確認します。
func IsValidMode(mode string) bool {
	once.Do(func() {
		// 初回だけ読み込んでキャッシュするのだ
		p, err := LoadPrompts()
		if err == nil {
			cachedPrompts = p
		}
	})

	if cachedPrompts == nil {
		return false
	}
	_, ok := cachedPrompts[mode]
	return ok
}
