package adapters

import (
	_ "embed"

	"github.com/shouni/gemini-reviewer-core/pkg/domain"
	coreprompts "github.com/shouni/gemini-reviewer-core/pkg/prompts"
)

const (
	modeDetail  = "detail"
	modeRelease = "release"
)

var (
	//go:embed prompts/prompt_detail.md
	detailPrompt string
	//go:embed prompts/prompt_release.md
	releasePrompt string
)

// NewPromptAdapter は domain.PromptBuilder のインスタンスを構築します。
func NewPromptAdapter() (domain.PromptBuilder, error) {
	templates := map[string]string{
		modeDetail:  detailPrompt,
		modeRelease: releasePrompt,
	}
	return coreprompts.NewBuilder(templates)
}
