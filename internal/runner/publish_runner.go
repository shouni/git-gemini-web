package runner

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"time"

	"git-gemini-web/internal/config"
	"git-gemini-web/internal/domain"

	core "github.com/shouni/gemini-reviewer-core/pkg/domain"
	"github.com/shouni/go-remote-io/pkg/remoteio"
)

// PublishRunner は、レビュー結果の公開処理を実行する具象構造体です。
type PublishRunner struct {
	publisher core.Publisher
	urlSigner remoteio.URLSigner
	notifier  domain.Notifier
}

// NewPublisherRunner は PublishRunner の新しいインスタンスを作成します。
func NewPublisherRunner(publisher core.Publisher, urlSigner remoteio.URLSigner, notifier domain.Notifier) *PublishRunner {
	return &PublishRunner{
		publisher: publisher,
		urlSigner: urlSigner,
		notifier:  notifier,
	}
}

// Run は 最終結果の構築、エラーハンドリング、GCSへの公開といった後処理を一元的に担います。
func (p *PublishRunner) Run(
	ctx context.Context,
	req domain.ReviewRequest,
	outcome domain.ReviewProcessOutcome,
) (domain.ReviewResult, error) {
	finalDuration := time.Since(outcome.StartTime)
	var result domain.ReviewResult
	err := outcome.Error

	if err != nil {
		// 1. エラー発生時: エラーレポートを生成
		slog.ErrorContext(ctx, "エラーレポートをGCSに公開準備中", "step", outcome.StepName, "error", err)

		var reportErr error
		outcome.ReviewMarkdown, reportErr = generateErrorReport(ctx, err, req, finalDuration, outcome.StepName)

		// レポート生成自体のエラーがあれば結合する
		if !errors.Is(reportErr, err) {
			err = errors.Join(err, reportErr)
		}
		result = domain.NewFailureResult(req, err, finalDuration)

	} else {
		// 2. 成功またはスキップ時
		var finalMessage string
		if outcome.IsSkipped {
			finalMessage = "差分が存在しないため、レビューをスキップしました。"
			slog.InfoContext(ctx, "✅ レビューパイプライン完了 (スキップ)", "duration", finalDuration)
		} else {
			finalMessage = "AIコードレビューが正常に完了しました。"
			slog.InfoContext(ctx, "✅ レビューパイプライン完了", "duration", finalDuration)
		}
		result = domain.NewSuccessResult(req, finalMessage, finalDuration)
	}

	// 3. GCSへのパブリッシュ実行 (MarkdownをHTMLに変換して保存)
	publishErr := p.publish(ctx, req, outcome.ReviewMarkdown)

	// 公開フェーズのエラーハンドリング
	if publishErr != nil {
		if err == nil {
			err = publishErr
			result = domain.NewFailureResult(req, err, finalDuration)
		} else {
			err = errors.Join(err, publishErr)
		}
	}

	return result, err
}

// publish は公開処理のパイプライン全体を実行します。
func (p *PublishRunner) publish(ctx context.Context, req domain.ReviewRequest, reviewMarkdown string) error {
	// ReviewDataを構築
	reviewData := core.ReviewData{
		RepoURL:        req.RepoURL,
		BaseBranch:     req.BaseBranch,
		FeatureBranch:  req.FeatureBranch,
		ReviewMarkdown: reviewMarkdown,
	}

	// GCSへの公開
	storageURI := fmt.Sprintf("gs://%s/%s", req.GCSBucket, req.GCSPath)
	publishErr := p.publisher.Publish(ctx, storageURI, reviewData)

	// Early Return: 公開エラーが発生した場合、即座にエラーを返して終了
	if publishErr != nil {
		slog.ErrorContext(ctx, "致命的エラー: レビュー結果の公開に失敗しました", "error", publishErr)
		return fmt.Errorf("レビュー結果の公開に失敗: %w", publishErr)
	}

	// --- 公開成功後の処理 ---

	// 署名付きURLの取得。失敗しても処理は継続する（WarnログとstorageURIの利用）
	publicURL, urlErr := p.getPublicURL(ctx, storageURI)
	if urlErr != nil {
		slog.Warn("公開URLの生成に失敗しました...", "error", urlErr)
		publicURL = storageURI
	}

	// Slackへの通知
	slog.InfoContext(ctx, "レビュー結果(またはエラーレポート)をSlackに通知中")
	if notifyErr := p.notifier.Notify(ctx, publicURL, storageURI, req); notifyErr != nil {
		// 通知は非致命的エラーとして警告ログを出し、処理は継続する
		slog.WarnContext(ctx, "Slack通知に失敗しました", "error", notifyErr)
	}

	return nil
}

// getPublicURL は URI に応じて署名付きURLを生成するか、公開URLに変換します。
func (p *PublishRunner) getPublicURL(ctx context.Context, storageURI string) (string, error) {
	signedURL, err := p.urlSigner.GenerateSignedURL(ctx, storageURI, "GET", config.SignedURLExpiration)
	if err != nil {
		return "", fmt.Errorf("GCS 署名付きURLの生成に失敗しました: %w", err)
	}
	// URL全体を出力せず、成功した事実のみ、あるいはマスクして出力する
	slog.DebugContext(ctx, "GCS 署名付きURLの生成に成功", "uri_path", storageURI)

	return signedURL, nil
}
