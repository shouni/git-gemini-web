package adapters

import (
	"github.com/shouni/gemini-reviewer-core/pkg/domain"
	coreprompts "github.com/shouni/gemini-reviewer-core/pkg/prompts"

	"git-gemini-web/assets"
)

const (
	modeDetail  = "detail"
	modeRelease = "release"
)

// NewPromptAdapter は domain.PromptBuilder のインスタンスを構築します。
func NewPromptAdapter() (domain.PromptBuilder, error) {
	templates := map[string]string{
		modeDetail:  assets.DetailPrompt,
		modeRelease: assets.ReleasePrompt,
	}
	return coreprompts.NewBuilder(templates)
}
