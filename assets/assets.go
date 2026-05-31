package assets

import (
	"embed"
	"log/slog"
	"sort"
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

// AvailableModes は、埋め込まれたレビュープロンプトから利用可能なモード名を返します。
func AvailableModes() ([]string, error) {
	prompts, err := loadCachedPrompts()
	if err != nil {
		return nil, err
	}

	modes := make([]string, 0, len(prompts))
	for mode := range prompts {
		modes = append(modes, mode)
	}
	sort.Strings(modes)
	return modes, nil
}

// IsValidMode は、指定されたモード名に対応するプロンプトファイルが存在するか確認します。
func IsValidMode(mode string) bool {
	prompts, err := loadCachedPrompts()
	if err != nil {
		slog.Error("failed to load prompts for validation", "error", err)
		return false
	}

	_, ok := prompts[mode]
	return ok
}

func loadCachedPrompts() (map[string]string, error) {
	mu.RLock()
	if cachedPrompts != nil {
		prompts := cachedPrompts
		mu.RUnlock()
		return prompts, nil
	}
	mu.RUnlock()

	mu.Lock()
	defer mu.Unlock()
	// Double-checked locking
	if cachedPrompts == nil {
		p, err := LoadPrompts()
		if err != nil {
			return nil, err
		}
		cachedPrompts = p
	}

	return cachedPrompts, nil
}
