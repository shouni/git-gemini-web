package adapters

import (
	"fmt"

	"github.com/google/uuid"
	"github.com/shouni/gemini-reviewer-core/git"
	"github.com/shouni/gemini-reviewer-core/ports"
	"github.com/shouni/go-utils/giturl"

	"github.com/shouni/git-gemini-web/internal/config"
)

// GitFactory は、ports.GitFactory インターフェースを満たす具象型です。
type GitFactory struct {
	sshKeyPath       string
	skipHostKeyCheck bool
}

// コンパイル時に ports.GitFactory インターフェースの実装を保証します
var _ ports.GitFactory = (*GitFactory)(nil)

// NewGitFactory は、ports.GitFactory インターフェースを満たす GitFactory を生成します。
func NewGitFactory(cfg *config.Config) *GitFactory {
	return &GitFactory{
		sshKeyPath:       cfg.SSHKeyPath,
		skipHostKeyCheck: cfg.SkipHostKeyCheck,
	}
}

// Create は ports.GitFactory インターフェースを満たします。
func (g *GitFactory) Create(repoURL, baseBranch string) ports.GitService {
	localPath := g.generateLocalPath(repoURL)
	return git.NewAdapter(
		localPath,
		g.sshKeyPath,
		git.WithInsecureSkipHostKeyCheck(g.skipHostKeyCheck),
		git.WithBaseBranch(baseBranch),
	)
}

// generateLocalPath はリポジトリURLから実行ごとにユニークなローカルパスを生成します。
func (g *GitFactory) generateLocalPath(repoURL string) string {
	const baseRepoDirName = "reviewer-repos"
	basePath := giturl.SanitizeURLToUniquePath(repoURL, baseRepoDirName)
	uniqueID := uuid.New().String()

	return fmt.Sprintf("%s-%s", basePath, uniqueID)
}
