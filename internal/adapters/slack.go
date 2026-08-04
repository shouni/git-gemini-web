package adapters

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/shouni/gemini-reviewer-core/ports"
	"github.com/shouni/go-http-kit/httpkit"
	"github.com/shouni/go-notify/notify"
	"github.com/shouni/go-notify/slack"

	"github.com/shouni/git-gemini-web/internal/giturl"
)

// slackTitles はレビュー結果ごとの見出しです。
var slackTitles = notify.Titles{
	Success: "✅ AIコードレビュー結果がアップロードされました。",
	Failure: "❌ AIコードレビューの生成に失敗しました。",
	Skipped: "⏭️ 差分がないため、レビューをスキップしました。",
}

// SlackAdapter は ports.Notifier インターフェースを満たす具象型です。
type SlackAdapter struct {
	pipeline *notify.Pipeline
}

var _ ports.Notifier = (*SlackAdapter)(nil)

// NewSlackAdapter は新しいアダプターインスタンスを作成します。
// webhookURL が空の場合は通知を行わないアダプターを返します。
func NewSlackAdapter(httpClient httpkit.Requester, webhookURL string) (*SlackAdapter, error) {
	notifier, err := slack.NewNotifier(httpClient, webhookURL)
	if err != nil {
		return nil, fmt.Errorf("slackクライアントの初期化に失敗しました: %w", err)
	}

	return &SlackAdapter{
		pipeline: notify.NewPipeline(notifier, slackTitles),
	}, nil
}

// Notify は ports.Notifier インターフェースの実装です。
// 成功時は publicURL をリンク先として、失敗時・スキップ時はその内容を Slack に投稿します。
// 失敗時・スキップ時は Publisher.Publish が実行されず詳細URLが存在しないため、
// 成功時とは異なるメッセージを組み立てます。
func (s *SlackAdapter) Notify(ctx context.Context, outcome ports.ReviewProcessOutcome) error {
	if !s.pipeline.Enabled() {
		slog.InfoContext(ctx, "Slack通知が無効化されているためスキップします。", "storage_uri", outcome.Req.StorageURI)
		return nil
	}

	if err := s.send(ctx, outcome); err != nil {
		return fmt.Errorf("slackへの結果投稿に失敗しました: %w", err)
	}

	slog.InfoContext(ctx, "レビュー結果を Slack に投稿しました。", "public_url", outcome.Req.PublicURL)
	return nil
}

// send は outcome の状態(成功/スキップ/失敗)に応じた通知を送信します。
func (s *SlackAdapter) send(ctx context.Context, outcome ports.ReviewProcessOutcome) error {
	switch {
	case outcome.Error != nil:
		// 詳細URLは存在しない(Publishが行われない)ため含めず、代わりに発生ステップを載せます。
		// エラー内容そのものは Pipeline が末尾に追記します。
		body := buildRepositoryBody(outcome.Req)
		body.Field("発生ステップ", outcome.StepName)
		return s.pipeline.Failure(ctx, body, outcome.Error)

	case outcome.IsSkipped:
		// スキップの理由は見出しが述べているため、理由は渡しません。
		return s.pipeline.Skipped(ctx, buildRepositoryBody(outcome.Req), nil)

	default:
		body := notify.NewBody().Link("詳細URL", outcome.Req.PublicURL, outcome.Req.StorageURI)
		writeRepositoryFields(body, outcome.Req)
		body.Code("モード", outcome.Req.Mode).
			Code("モデル", outcome.Req.ModelName)
		return s.pipeline.Success(ctx, body)
	}
}

// buildRepositoryBody はリポジトリとブランチだけを持つ本文を返します。
func buildRepositoryBody(req ports.ReviewRequest) *notify.Body {
	body := notify.NewBody()
	writeRepositoryFields(body, req)
	return body
}

// writeRepositoryFields はリポジトリと比較ブランチを本文へ追記します。
func writeRepositoryFields(body *notify.Body, req ports.ReviewRequest) {
	body.Code("リポジトリ", giturl.GetRepositoryPath(req.RepoURL))

	if req.BaseBranch == "" && req.FeatureBranch == "" {
		return
	}
	body.Field("ブランチ", fmt.Sprintf("`%s` ← `%s`", req.BaseBranch, req.FeatureBranch))
}
