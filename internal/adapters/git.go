package adapters

import (
	"github.com/shouni/gemini-reviewer-core/git"
	"github.com/shouni/gemini-reviewer-core/ports"
	"github.com/shouni/go-utils/urlpath"

	"git-gemini-web/internal/config"
)

const baseRepoDirName = "reviewer-repos"

// GitFactory は、domain.GitFactory インターフェースを満たす具象型です。
type GitFactory struct {
	sshKeyPath       string
	skipHostKeyCheck bool
}

func NewGitFactory(cfg *config.Config) *GitFactory {
	return &GitFactory{
		sshKeyPath:       cfg.SSHKeyPath,
		skipHostKeyCheck: cfg.SkipHostKeyCheck,
	}
}

// Create は runner.GitAdapterFactory インターフェースを満たします。
func (g *GitFactory) Create(repoURL, baseBranch string) ports.GitService {
	localPath := urlpath.SanitizeURLToUniquePath(repoURL, baseRepoDirName)
	skipHostKeyCheckOption := git.WithInsecureSkipHostKeyCheck(g.skipHostKeyCheck)
	baseBranchOption := git.WithBaseBranch(baseBranch)

	return git.NewGitAdapter(
		localPath,
		g.sshKeyPath,
		skipHostKeyCheckOption,
		baseBranchOption,
	)
}
