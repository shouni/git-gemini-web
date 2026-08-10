// Package config は、環境変数からアプリケーション設定を読み込み・検証します。
package config

import (
	"time"
)

const (
	// DefaultHTTPTimeout は外部HTTP通信のデフォルトタイムアウトです。
	DefaultHTTPTimeout = 30 * time.Second

	// TaskDispatchDeadline は Cloud Tasks がワーカーの応答を待つ上限です。
	// 未指定だと既定 10 分が効き、Cloud Run の timeout を伸ばしても効きません。
	TaskDispatchDeadline = 10 * time.Minute

	// DefaultPipelineTimeout はレビュー 1 件の実行時間の上限の既定値です。
	// 実測 3.9〜9.2 秒に対する余裕として 5m。他アプリの 25m より短いのは意図的です。
	DefaultPipelineTimeout = 5 * time.Minute
)

// Config は環境変数からアプリケーション設定を読み込む構造体です。
type Config struct {
	ServiceURL          string // Cloud RunのURL (例: https://myapp.run.app)
	Port                string
	ProjectID           string
	LocationID          string
	QueueID             string
	TaskAudienceURL     string // OIDC トークンの検証に使用する Audience URL
	ServiceAccountEmail string
	GCSBucket           string
	SlackWebhookURL     string
	GeminiAPIKey        string

	// GeminiModels は GEMINI_MODELS（カンマ区切り）で指定するモデル名の一覧です。
	GeminiModels []string

	SSHKeyPath string

	// PipelineTimeout はレビュー 1 件（clone〜AI〜公開）の実行時間の上限です。
	// 0 以下は無制限を意味します。詳細は DefaultPipelineTimeout のコメント。
	PipelineTimeout time.Duration
	// pipelineTimeoutErr は PIPELINE_TIMEOUT の解析に失敗したときの理由です。
	// LoadConfig はエラーを返さない契約なのでここへ持ち越し、
	// ValidateEssentialConfig が起動時に落とします（黙って既定値へ落ちない）。
	pipelineTimeoutErr error

	// OAuth & Session Settings
	GoogleClientID     string
	GoogleClientSecret string
	// SessionSecret はセッションデータのHMAC署名用シークレットキーです。
	SessionSecret string
	// SessionEncryptKey はセッションデータのAES暗号化用シークレットキーです。 16, 24, 32 バイトのいずれかである必要があります。
	SessionEncryptKey string

	// Auth Settings
	AllowedEmails  []string
	AllowedDomains []string
}

// LoadConfig は環境変数から設定を読み込みます。
func LoadConfig() *Config {
	serviceURL := getEnv("SERVICE_URL", "http://localhost:8080")
	allowedEmails := getEnv("ALLOWED_EMAILS", "")
	allowedDomains := getEnv("ALLOWED_DOMAINS", "")
	pipelineTimeout, pipelineTimeoutErr := parseDurationEnv("PIPELINE_TIMEOUT", DefaultPipelineTimeout)

	return &Config{
		ServiceURL:          serviceURL,
		Port:                getEnv("PORT", "8080"),
		ProjectID:           getEnv("GCP_PROJECT_ID", "your-gcp-project"),
		LocationID:          getEnv("GCP_LOCATION_ID", "asia-northeast1"),
		QueueID:             getEnv("CLOUD_TASKS_QUEUE_ID", "review-queue"),
		TaskAudienceURL:     getEnv("TASK_AUDIENCE_URL", serviceURL),
		ServiceAccountEmail: getEnv("SERVICE_ACCOUNT_EMAIL", ""),
		GCSBucket:           getEnv("GCS_REVIEW_BUCKET", "your-review-archive-bucket"),
		SlackWebhookURL:     getEnv("SLACK_WEBHOOK_URL", ""),
		GeminiAPIKey:        getEnv("GEMINI_API_KEY", ""),
		GeminiModels:        parseCommaSeparatedList(getEnv("GEMINI_MODELS", "")),
		SSHKeyPath:          getEnv("SSH_KEY_PATH", "~/.ssh/id_rsa"),
		PipelineTimeout:     pipelineTimeout,
		pipelineTimeoutErr:  pipelineTimeoutErr,

		// OAuth & Session
		GoogleClientID:     getEnv("GOOGLE_CLIENT_ID", ""),
		GoogleClientSecret: getEnv("GOOGLE_CLIENT_SECRET", ""),
		SessionSecret:      getEnv("SESSION_SECRET", ""),
		SessionEncryptKey:  getEnv("SESSION_ENCRYPT_KEY", ""),

		AllowedEmails:  parseCommaSeparatedList(allowedEmails),
		AllowedDomains: parseCommaSeparatedList(allowedDomains),
	}
}
