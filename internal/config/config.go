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

	// TaskDispatchDeadline は Cloud Tasks がワーカーの応答を待つ上限です。
	//
	// 「待つ時間」ではなく **ワーカーの実行時間の実効上限** で、これを超えると
	// 処理中でも Cloud Tasks は待受を打ち切ります。未指定だと既定の 10 分が効き、
	// Cloud Run の timeout をいくら伸ばしても 10 分で切られます。
	// HTTP ターゲットの上限である 30 分を取り、Cloud Run の timeout と揃えます。
	//
	// タイムアウトの三段（下記）の**真ん中**。builder/task.go が Cloud Tasks へ渡します。
	TaskDispatchDeadline = 30 * time.Minute

	// DefaultPipelineTimeout はレビュー 1 件の実行時間の上限の既定値です。
	//
	// ★ タイムアウトは 3 つあり、この大小関係を守ります。
	//
	//     PIPELINE_TIMEOUT  <  dispatch deadline  <=  Cloud Run の timeout
	//        (このアプリ)         (Cloud Tasks)          (ap-infra)
	//
	//   実効上限を決めるのは**いちばん小さい値**で、他は飾りになります。
	//   PIPELINE_TIMEOUT をいちばん短く取るのが要点で、**アプリが自分で先に諦める**ことで
	//   失敗レポートを GCS へ書き、Slack へ通知してから終われます。逆順にすると先に
	//   Cloud Tasks が打ち切り、プロセスごと SIGTERM になるため通知が一切残りません
	//   （review-queue は max_attempts = 1 なので再試行も来ず、タスクは黙って失われます）。
	//
	//   フリートの他アプリ（ap-comp / ap-mv / ap-story / lyric-video）と同じ 25m です。
	DefaultPipelineTimeout = 25 * time.Minute
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
	geminiModels := parseCommaSeparatedList(getEnv("GEMINI_MODEL", DefaultGeminiModel))
	if len(geminiModels) == 0 {
		geminiModels = []string{DefaultGeminiModel}
	}

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
		GeminiModel:         geminiModels[0],
		GeminiModels:        geminiModels,
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
