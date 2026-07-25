package adapters

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"strings"

	"github.com/shouni/gemini-reviewer-core/ports"
	"github.com/shouni/git-gemini-web/internal/giturl"
	"github.com/shouni/go-http-kit/httpkit"
	"github.com/shouni/go-notifier/pkg/slack"
)

// SlackAdapter は ports.Notifier インターフェースを満たす具象型です。
type SlackAdapter struct {
	slackClient *slack.Client
	webhookURL  string
}

// NewSlackAdapter は新しいアダプターインスタンスを作成します。
func NewSlackAdapter(httpClient httpkit.Requester, webhookURL string) (*SlackAdapter, error) {
	if webhookURL == "" {
		// オプショナル機能として扱い、空のままインスタンスを返す
		return &SlackAdapter{}, nil
	}

	if httpClient == nil {
		return nil, errors.New("http client cannot be nil")
	}

	client, err := slack.NewClient(httpClient, webhookURL)
	if err != nil {
		return nil, fmt.Errorf("slackクライアントの初期化に失敗しました: %w", err)
	}

	return &SlackAdapter{
		slackClient: client,
		webhookURL:  webhookURL,
	}, nil
}

// Notify は ports.Notifier インターフェースの実装です。
// 成功時は publicURL をリンク先として、失敗時・スキップ時はその内容を Slack に投稿します。
// 失敗時・スキップ時は Publisher.Publish が実行されず詳細URLが存在しないため、
// 成功時とは異なるメッセージを組み立てます。
func (s *SlackAdapter) Notify(ctx context.Context, outcome ports.ReviewProcessOutcome) error {
	if s.webhookURL == "" || s.slackClient == nil {
		slog.Info("Slack通知が無効化されているか、クライアントが未初期化のためスキップします。", "storage_uri", outcome.Req.StorageURI)
		return nil
	}

	title, content := s.buildMessage(outcome)

	if err := s.slackClient.SendTextWithHeader(ctx, title, content); err != nil {
		return fmt.Errorf("slackへの結果投稿に失敗しました: %w", err)
	}

	slog.Info("レビュー結果を Slack に投稿しました。", "public_url", outcome.Req.PublicURL)
	return nil
}

// buildMessage は outcome の状態(成功/スキップ/失敗)に応じたタイトルと本文を組み立てます。
func (s *SlackAdapter) buildMessage(outcome ports.ReviewProcessOutcome) (title, content string) {
	switch {
	case outcome.Error != nil:
		return "❌ AIコードレビューの生成に失敗しました。", s.buildErrorContent(outcome)
	case outcome.IsSkipped:
		return "⏭️ 差分がないため、レビューをスキップしました。", s.buildSkipContent(outcome.Req)
	default:
		return "✅ AIコードレビュー結果がアップロードされました。", s.buildSlackContent(outcome.Req)
	}
}

// buildSlackContent は投稿メッセージの本文を組み立てます。
// publicURLをメッセージ内のリンク先URL、storageURIをそのリンクの表示テキストとして使用します。
func (s *SlackAdapter) buildSlackContent(req ports.ReviewRequest) string {
	repoPath := giturl.GetRepositoryPath(req.RepoURL)
	content := fmt.Sprintf(
		"*詳細URL:* <%s|%s>\n"+
			"*リポジトリ:* `%s`\n"+
			"*ブランチ:* `%s` ← `%s`\n"+
			"*モード:* `%s`\n"+
			"*モデル:* `%s`",
		req.PublicURL,
		req.StorageURI,
		repoPath,
		req.BaseBranch,
		req.FeatureBranch,
		req.Mode,
		req.ModelName,
	)
	return strings.TrimSpace(content)
}

// buildErrorContent は、失敗時の投稿メッセージの本文を組み立てます。
// 詳細URLは存在しない(Publishが行われない)ため含めず、代わりに発生ステップと
// エラー内容そのものを含めます。
func (s *SlackAdapter) buildErrorContent(outcome ports.ReviewProcessOutcome) string {
	repoPath := giturl.GetRepositoryPath(outcome.Req.RepoURL)
	content := fmt.Sprintf(
		"*リポジトリ:* `%s`\n"+
			"*ブランチ:* `%s` ← `%s`\n"+
			"*発生ステップ:* %s\n"+
			"*エラー内容:* ```%v```",
		repoPath,
		outcome.Req.BaseBranch,
		outcome.Req.FeatureBranch,
		outcome.StepName,
		outcome.Error,
	)
	return strings.TrimSpace(content)
}

// buildSkipContent は、スキップ時の投稿メッセージの本文を組み立てます。
func (s *SlackAdapter) buildSkipContent(req ports.ReviewRequest) string {
	repoPath := giturl.GetRepositoryPath(req.RepoURL)
	content := fmt.Sprintf(
		"*リポジトリ:* `%s`\n"+
			"*ブランチ:* `%s` ← `%s`",
		repoPath,
		req.BaseBranch,
		req.FeatureBranch,
	)
	return strings.TrimSpace(content)
}
