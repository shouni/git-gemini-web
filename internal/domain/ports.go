package domain

import "context"

// Pipeline レビュー要求を処理するために実行される一連のプロセスを表します。
type Pipeline interface {
	// Execute 指定されたコンテキスト内で指定されたレビュー要求を処理し、操作が失敗した場合はエラーを返します。
	Execute(ctx context.Context, payload ReviewRequest) error
}

// ReviewRunner は、レビュー要求に対して実際のレビュー処理（AI分析等）を実行するインターフェースです。
type ReviewRunner interface {
	Run(ctx context.Context, req ReviewRequest) ReviewProcessOutcome
}

// PublisherRunner は、レビュー結果の公開処理を実行する責務を持つインターフェースです。
type PublisherRunner interface {
	Run(ctx context.Context, req ReviewRequest, outcome ReviewProcessOutcome) (ReviewResult, error)
}

// SlackNotifier は Slack への通知機能を提供する契約を定義します。
// publicURL は外部からアクセス可能なリンク (署名済みURLなど) を示し、
// storageURI は内部的なストレージの場所 (s3://... など) を示します。
type SlackNotifier interface {
	Notify(ctx context.Context, publicURL, storageURI string, req ReviewRequest) error
}
