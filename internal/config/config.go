// Package config は、環境変数からアプリケーション設定を読み込み・検証します。
package config

import (
	"time"
)

const (
	// DefaultHTTPTimeout は外部HTTP通信のデフォルトタイムアウトです。
	DefaultHTTPTimeout = 30 * time.Second
	// SignedURLExpiration は生成物の署名付きURLの有効期限です。
	SignedURLExpiration = 30 * time.Minute
	// DefaultGeminiModel はレビュー生成に使用するデフォルトのGeminiモデルです。
	DefaultGeminiModel = "gemini-3.6-flash"
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
	GeminiModel         string
	GeminiModels        []string
	SSHKeyPath          string
	SkipHostKeyCheck    bool

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
	geminiModels := parseCommaSeparatedList(getEnv("GEMINI_MODEL", DefaultGeminiModel))
	if len(geminiModels) == 0 {
		geminiModels = []string{DefaultGeminiModel}
	}

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
		GeminiModel:         geminiModels[0],
		GeminiModels:        geminiModels,
		SSHKeyPath:          getEnv("SSH_KEY_PATH", "~/.ssh/id_rsa"),
		SkipHostKeyCheck:    getEnvAsBool("SKIP_HOST_KEY_CHECK", false),

		// OAuth & Session
		GoogleClientID:     getEnv("GOOGLE_CLIENT_ID", ""),
		GoogleClientSecret: getEnv("GOOGLE_CLIENT_SECRET", ""),
		SessionSecret:      getEnv("SESSION_SECRET", ""),
		SessionEncryptKey:  getEnv("SESSION_ENCRYPT_KEY", ""),

		AllowedEmails:  parseCommaSeparatedList(allowedEmails),
		AllowedDomains: parseCommaSeparatedList(allowedDomains),
	}
}
