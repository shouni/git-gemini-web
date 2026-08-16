// Package assets は、プロンプトテンプレート等を埋め込みリソースとして提供します。
package assets

import (
	"embed"
	"fmt"
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

	// partialFiles は、複数のプロンプトモードで共有するテキスト断片です。
	// prompts/ とは別ディレクトリに置き、レビューモードの一覧には含めません。
	//go:embed partials/*.md
	partialFiles embed.FS

	// Templates は、HTMLテンプレートです。
	//go:embed templates/*.html
	Templates embed.FS

	// StaticFiles は、ブラウザへ配信するJavaScriptなどの静的ファイルを保持します。
	//go:embed static
	StaticFiles embed.FS
)

// loadPrompts は、埋め込みプロンプトの解析結果を最初の呼び出しで1度だけ構築します。
//
// AvailableModes と IsValidMode はリクエストのたびに呼ばれるため遅延初期化しますが、
// 二重チェックロックを手で書く必要はありません。読み込み元は埋め込みアセットで、
// 失敗するとすれば「モード名の衝突」のように毎回同じ結果になるものだけなので、
// エラーごとキャッシュして再試行しないのが正しい挙動です。
//
// 返すマップは呼び出し側で共有されます。書き換えないでください
// （LoadPrompts と AvailableModes は、いずれも新しい入れ物へ写して返します）。
var loadPrompts = sync.OnceValues(func() (map[string]promptTemplate, error) {
	files, err := resource.Load(promptFiles, promptDir)
	if err != nil {
		return nil, err
	}

	parsed := make(map[string]promptTemplate, len(files))
	for mode, body := range files {
		description, promptBody := parsePromptMetadata(mode, body)
		parsed[mode] = promptTemplate{
			body:        promptBody,
			description: description,
		}
	}
	return parsed, nil
})

type promptTemplate struct {
	body        string
	description string
}

// ReviewMode は、フォームに表示するレビューモードのメタデータです。
type ReviewMode struct {
	Name        string
	Description string
}

// LoadPrompts は埋め込まれたプロンプトの本文をモード名で引けるマップとして返します。
func LoadPrompts() (map[string]string, error) {
	cached, err := loadPrompts()
	if err != nil {
		return nil, err
	}

	prompts := make(map[string]string, len(cached))
	for mode, prompt := range cached {
		prompts[mode] = prompt.body
	}
	return prompts, nil
}

// LoadFindingsFormat は、レビュー指摘のJSONフォーマットを説明する共通テキストを読み込みます。
// 全レビューモードのプロンプトで共有され、AIの構造化出力(findings配列)のスキーマに
// 対応する項目を説明します。
func LoadFindingsFormat() (string, error) {
	return loadPartial("findings_format.md")
}

// LoadVerdictFormat は、判定結果のJSONフォーマット(verdictオブジェクト)を説明する
// 共通テキストを読み込みます。
func LoadVerdictFormat() (string, error) {
	return loadPartial("verdict_format.md")
}

func loadPartial(name string) (string, error) {
	b, err := partialFiles.ReadFile("partials/" + name)
	if err != nil {
		return "", fmt.Errorf("共有テンプレート '%s' の読み込みに失敗: %w", name, err)
	}
	return string(b), nil
}

// AvailableModes は、埋め込まれたレビュープロンプトから利用可能なモード名を返します。
func AvailableModes() ([]ReviewMode, error) {
	cached, err := loadPrompts()
	if err != nil {
		return nil, err
	}

	modes := make([]ReviewMode, 0, len(cached))
	for mode, prompt := range cached {
		modes = append(modes, ReviewMode{
			Name:        mode,
			Description: prompt.description,
		})
	}

	sort.Slice(modes, func(i, j int) bool {
		return modes[i].Name < modes[j].Name
	})
	return modes, nil
}

// IsValidMode は、指定されたモード名に対応するプロンプトファイルが存在するか確認します。
func IsValidMode(mode string) bool {
	cached, err := loadPrompts()
	if err != nil {
		slog.Error("failed to load prompts for validation", "error", err)
		return false
	}

	_, ok := cached[mode]
	return ok
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
