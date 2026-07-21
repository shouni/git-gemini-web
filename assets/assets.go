// Package assets は、プロンプトテンプレート等を埋め込みリソースとして提供します。
package assets

import (
	"embed"
	"log/slog"
	"sort"
	"strings"
	"sync"

	"github.com/shouni/go-prompt-kit/resource"
)

const (
	promptDir             = "prompts"
	modeDescriptionPrefix = "<!-- mode-description:"
	metadataSuffix        = "-->"
)

var (
	// promptFiles はプロンプトテンプレートです。ディレクトリ内は現在プロンプトのみのため、
	// ファイル名のprefixは不要です（ファイル名がそのままモード名になります）。
	//go:embed prompts/*.md
	promptFiles embed.FS

	// Templates は、HTMLテンプレートです。
	//go:embed templates/*.html
	Templates embed.FS

	cachedPrompts map[string]promptTemplate
	mu            sync.RWMutex
)

type promptTemplate struct {
	body        string
	description string
}

// ReviewMode は、フォームに表示するレビューモードのメタデータです。
type ReviewMode struct {
	Name        string
	Description string
}

// LoadPrompts は埋め込まれたプロンプトファイルを読み込みます。
func LoadPrompts() (map[string]string, error) {
	if err := ensurePromptCache(); err != nil {
		return nil, err
	}

	mu.RLock()
	defer mu.RUnlock()

	prompts := make(map[string]string, len(cachedPrompts))
	for mode, prompt := range cachedPrompts {
		prompts[mode] = prompt.body
	}
	return prompts, nil
}

// AvailableModes は、埋め込まれたレビュープロンプトから利用可能なモード名を返します。
func AvailableModes() ([]ReviewMode, error) {
	if err := ensurePromptCache(); err != nil {
		return nil, err
	}

	mu.RLock()
	modes := make([]ReviewMode, 0, len(cachedPrompts))
	for mode, prompt := range cachedPrompts {
		modes = append(modes, ReviewMode{
			Name:        mode,
			Description: prompt.description,
		})
	}
	mu.RUnlock()

	sort.Slice(modes, func(i, j int) bool {
		return modes[i].Name < modes[j].Name
	})
	return modes, nil
}

// IsValidMode は、指定されたモード名に対応するプロンプトファイルが存在するか確認します。
func IsValidMode(mode string) bool {
	if err := ensurePromptCache(); err != nil {
		slog.Error("failed to load prompts for validation", "error", err)
		return false
	}

	mu.RLock()
	defer mu.RUnlock()
	_, ok := cachedPrompts[mode]
	return ok
}

func ensurePromptCache() error {
	mu.RLock()
	if cachedPrompts != nil {
		mu.RUnlock()
		return nil
	}
	mu.RUnlock()

	mu.Lock()
	defer mu.Unlock()
	// Double-checked locking
	if cachedPrompts == nil {
		p, err := resource.Load(promptFiles, promptDir, "")
		if err != nil {
			return err
		}
		parsedPrompts := make(map[string]promptTemplate, len(p))
		for mode, body := range p {
			description, promptBody := parsePromptMetadata(mode, body)
			parsedPrompts[mode] = promptTemplate{
				body:        promptBody,
				description: description,
			}
		}
		cachedPrompts = parsedPrompts
	}

	return nil
}

func parsePromptMetadata(mode, body string) (string, string) {
	trimmed := strings.TrimLeft(body, "\ufeff \t\r\n")
	if !strings.HasPrefix(trimmed, modeDescriptionPrefix) {
		return mode, trimmed
	}

	end := strings.Index(trimmed, metadataSuffix)
	if end < len(modeDescriptionPrefix) {
		return mode, trimmed
	}

	description := strings.TrimSpace(trimmed[len(modeDescriptionPrefix):end])
	if description == "" {
		return mode, trimmed
	}

	promptBody := strings.TrimLeft(trimmed[end+len(metadataSuffix):], "\r\n")
	return description, promptBody
}
