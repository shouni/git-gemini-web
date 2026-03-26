package domain

import (
	"context"

	"github.com/shouni/gemini-reviewer-core/ports"
)

// Pipeline は、レビュー要求を処理するために実行される一連のプロセスを表します。
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

// PromptGenerator は、AIプロンプトを生成するインターフェースです。
type PromptGenerator interface {
	GenerateReview(mode, diff string) (string, error)
	GenerateErrorReport(ctx context.Context, params ErrorReportParams) (string, error)
	GenerateSkipReport(req ReviewRequest) (string, error)
}

// Notifier は、生成されたコンテンツまたはエラーに関する通知を指定されたターゲットまたはチャネルに送信するためのインターフェイスです。
type Notifier interface {
	// Notify は、パブリック URL やストレージ URL などのメタデータを含む通知をターゲットに送信します。
	Notify(ctx context.Context, publicURL, storageURI string, req ReviewRequest) error
}

// GitFactory は、リクエスト固有の情報に基づいて GitService を生成する契約を定義します。
type GitFactory interface {
	Create(repoURL string, baseBranch string) ports.GitService
}
