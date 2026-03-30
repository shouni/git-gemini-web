package adapters

import (
	"fmt"

	"github.com/google/uuid"
	"github.com/shouni/gemini-reviewer-core/git"
	"github.com/shouni/gemini-reviewer-core/ports"
	"github.com/shouni/go-utils/urlpath"

	"git-gemini-web/internal/config"
)

// GitFactory は、ports.GitFactory インターフェースを満たす具象型です。
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

// Create は ports.GitFactory インターフェースを満たします。
func (g *GitFactory) Create(repoURL, baseBranch string) ports.GitService {
	localPath := g.generateLocalPath(repoURL)
	opts := []git.Option{
		git.WithInsecureSkipHostKeyCheck(g.skipHostKeyCheck),
		git.WithBaseBranch(baseBranch),
	}

	return git.NewGitAdapter(localPath, g.sshKeyPath, opts...)
}

// generateLocalPath はリポジトリURLから実行ごとにユニークなローカルパスを生成します。
func (g *GitFactory) generateLocalPath(repoURL string) string {
	const baseRepoDirName = "reviewer-repos"
	basePath := urlpath.SanitizeURLToUniquePath(repoURL, baseRepoDirName)
	uniqueID := uuid.New().String()

	return fmt.Sprintf("%s-%s", basePath, uniqueID)
}
