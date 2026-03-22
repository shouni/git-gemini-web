package adapters

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/shouni/gemini-reviewer-core/git"
	"github.com/shouni/gemini-reviewer-core/ports"
	"github.com/shouni/go-utils/urlpath"

	"git-gemini-web/internal/config"
)

const baseRepoDirName = "reviewer-repos"

// GitFactoryAdapter は、runner.GitAdapterFactory インターフェースを満たす具象型です。
type GitFactoryAdapter struct {
	sshKeyPath       string
	skipHostKeyCheck bool
}

func NewGitFactoryAdapter(cfg *config.Config) *GitFactoryAdapter {
	return &GitFactoryAdapter{
		sshKeyPath:       cfg.SSHKeyPath,
		skipHostKeyCheck: cfg.SkipHostKeyCheck,
	}
}

// Create は runner.GitAdapterFactory インターフェースを満たします。
func (g *GitFactoryAdapter) Create(repoURL, baseBranch string) ports.GitService {
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

// CloneAndDiff Gitリポジトリのクローンまたは更新後に、リポジトリ内の2つのブランチ間のコード差分を取得します。
func (g *GitFactoryAdapter) CloneAndDiff(ctx context.Context, gitService ports.GitService, repoURL, base, feat string) (string, string, error) {
	var stepName string
	defer g.cleanupGit(ctx, gitService)

	stepName = "リポジトリの準備"
	slog.InfoContext(ctx, "1. リポジトリをクローン/更新中", "repo_url", repoURL)
	if err := gitService.CloneOrUpdate(ctx, repoURL); err != nil {
		return "", stepName, fmt.Errorf("リポジトリの準備に失敗: %w", err)
	}

	slog.InfoContext(ctx, "2. フィーチャーブランチの存在を確認中", "branch", feat)
	exists, err := gitService.CheckRefExists(ctx, feat)
	if err != nil {
		return "", stepName, fmt.Errorf("ブランチ存在確認に失敗: %w", err)
	}
	if !exists {
		return "", stepName, fmt.Errorf("指定されたフィーチャーブランチ '%s' がリモートに存在しません。", feat)
	}

	// 3. 差分の取得
	stepName = "コード差分取得"
	slog.InfoContext(ctx, "3.コード差分取得中", "branch", feat)
	codeDiff, err := gitService.GetCodeDiff(ctx, base, feat)
	if err != nil {
		return "", stepName, err
	}

	return codeDiff, stepName, nil
}

// cleanupGit は、Git リソースのクリーンアップを処理し、クリーンアップ操作が失敗した場合は警告をログに記録します。
func (g *GitFactoryAdapter) cleanupGit(ctx context.Context, git ports.GitService) {
	if err := git.Cleanup(ctx); err != nil {
		slog.WarnContext(ctx, "Gitリソースのクリーンアップに失敗しました", "error", err)
	}
}
