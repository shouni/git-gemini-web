package domain

import "context"

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

// Notifier は、生成されたコンテンツまたはエラーに関する通知を指定されたターゲットまたはチャネルに送信するためのインターフェイスです。
type Notifier interface {
	// Notify は、パブリック URL やストレージ URL などのメタデータを含む通知をターゲットに送信します。
	Notify(ctx context.Context, publicURL, storageURI string, req ReviewRequest) error
}

// PromptBuilder は、プロンプト文字列を生成する責務を定義します。
type PromptBuilder interface {
	Build(mode string, content string) (string, error)
}

// CodeReviewAI は、AIとの通信機能の抽象化を提供し、DIで使用されます。
type CodeReviewAI interface {
	// ReviewCodeDiff は完成されたプロンプトを基にAIにレビューを依頼します。
	ReviewCodeDiff(ctx context.Context, finalPrompt string) (string, error)
}
