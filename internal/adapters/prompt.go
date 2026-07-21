package adapters

import (
	"fmt"

	"github.com/shouni/go-prompt-kit/prompts"

	"github.com/shouni/git-gemini-web/assets"
)

// reviewData はレビュープロンプトのテンプレートに渡すデータ構造です。
type reviewData struct {
	DiffContent string
}

// promptBuilder は、フォーマット済みのプロンプトを作成するためのインターフェース
type promptBuilder interface {
	Build(mode string, data any) (string, error)
}

// PromptAdapter は、さまざまなモードとデータに基づいてプロンプトを生成する役割を担います。
type PromptAdapter struct {
	reviewBuilder promptBuilder
}

// NewPromptAdapter は動的に読み込んだテンプレートを使用して Builder を構築します。
func NewPromptAdapter() (*PromptAdapter, error) {
	reviewTemplates, err := assets.LoadPrompts()
	if err != nil {
		return nil, fmt.Errorf("レビューテンプレートの読み込みに失敗: %w", err)
	}

	review, err := prompts.NewBuilder(reviewTemplates)
	if err != nil {
		return nil, fmt.Errorf("レビュービルダーの構築に失敗: %w", err)
	}

	return &PromptAdapter{
		reviewBuilder: review,
	}, nil
}

// GenerateReview はコードレビューのプロンプトを生成します。
func (pa *PromptAdapter) GenerateReview(mode, codeDiff string) (string, error) {
	data := reviewData{
		DiffContent: codeDiff,
	}
	prompt, err := pa.reviewBuilder.Build(mode, data)
	if err != nil {
		return "", fmt.Errorf("レビューテンプレートの実行に失敗: %w", err)
	}
	return prompt, nil
}
