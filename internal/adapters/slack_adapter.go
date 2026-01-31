package adapters

import (
	"context"
	"fmt"
	"log/slog"
	"strings"

	"git-gemini-web/internal/domain"

	"github.com/shouni/go-http-kit/pkg/httpkit"
	"github.com/shouni/go-notifier/pkg/factory"
	"github.com/shouni/go-notifier/pkg/slack"
	"github.com/shouni/go-utils/urlpath"
)

// --- 定数と内部構造体 ---

// SlackNotifier は Slack への通知機能を提供する契約を定義します。
// publicURL は外部からアクセス可能なリンク (署名済みURLなど) を示し、
// storageURI は内部的なストレージの場所 (s3://... など) を示します。
type SlackNotifier interface {
	Notify(ctx context.Context, publicURL, storageURI string, req domain.ReviewRequest) error
}

// SlackAdapter は SlackNotifier インターフェースを満たす具象型です。
type SlackAdapter struct {
	webhookURL  string // Webhook URLを保持
	slackClient *slack.Client
}

// NewSlackAdapter は新しいアダプターインスタンスを作成します。
func NewSlackAdapter(httpClient httpkit.ClientInterface, webhookURL string) (*SlackAdapter, error) {
	if webhookURL == "" {
		return &SlackAdapter{webhookURL: webhookURL}, nil
	}
	client, err := factory.GetSlackClient(httpClient)
	if err != nil {
		return nil, fmt.Errorf("Slackクライアントの初期化に失敗しました: %w", err)
	}

	return &SlackAdapter{
		webhookURL:  webhookURL,
		slackClient: client,
	}, nil
}

// Notify は SlackNotifier インターフェースの実装です。
// publicURL をリンク先として、Slack に投稿します。
func (s *SlackAdapter) Notify(ctx context.Context, publicURL, storageURI string, req domain.ReviewRequest) error {

	// 1. Slack 認証情報の取得とスキップチェック
	if s.webhookURL == "" {
		slog.Info("SLACK_WEBHOOK_URL が設定されていません。Slack通知をスキップします。", "storage_uri", storageURI)
		return nil
	}

	// 2. Slack に投稿するメッセージを作成
	title := "✅ AIコードレビュー結果がアップロードされました。"
	content := s.buildSlackContent(publicURL, storageURI, req)

	// 3. Slack投稿処理を実行
	if err := s.slackClient.SendTextWithHeader(ctx, title, content); err != nil {
		return fmt.Errorf("Slackへの結果URL投稿に失敗しました: %w", err)
	}

	slog.Info("レビュー結果のURLを Slack に投稿しました。", "public_url", publicURL)
	return nil
}

// buildSlackContent は投稿メッセージの本文を組み立てます。
// publicURLをメッセージ内のリンク先URL、storageURIをそのリンクの表示テキストとして使用します。
func (s *SlackAdapter) buildSlackContent(publicURL, storageURI string, req domain.ReviewRequest) string {
	repoPath := urlpath.GetRepositoryPath(req.RepoURL)
	content := fmt.Sprintf(
		"**詳細URL:** <%s|%s>\n"+
			"**リポジトリ:** `%s`\n"+
			"**ブランチ:** `%s` ← `%s`\n"+
			"**モード:** `%s`",
		publicURL,
		storageURI,
		repoPath,
		req.BaseBranch,
		req.FeatureBranch,
		req.Mode,
	)
	return strings.TrimSpace(content)
}
