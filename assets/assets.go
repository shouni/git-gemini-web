package assets

import (
	"embed"
	"log/slog"
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
	mu            sync.RWMutex
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
	mu.RLock()
	if cachedPrompts != nil {
		defer mu.RUnlock()
		_, ok := cachedPrompts[mode]
		return ok
	}
	mu.RUnlock()

	mu.Lock()
	defer mu.Unlock()
	// Double-checked locking
	if cachedPrompts == nil {
		p, err := LoadPrompts()
		if err != nil {
			slog.Error("failed to load prompts for validation", "error", err)
			return false
		}
		cachedPrompts = p
	}

	_, ok := cachedPrompts[mode]
	return ok
}
