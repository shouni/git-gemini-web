package builder

import (
	"fmt"
	"net/url"

	"github.com/shouni/gcp-kit/auth"
	"github.com/shouni/gcp-kit/worker"

	"github.com/shouni/git-gemini-web/internal/app"
	"github.com/shouni/git-gemini-web/internal/config"
	"github.com/shouni/git-gemini-web/internal/domain"
	"github.com/shouni/git-gemini-web/internal/server/handlers"
)

const defaultSessionName = "git-gemini-session"

// AppHandlers は生成されたすべての HTTP ハンドラーを保持する構造体です。
type AppHandlers struct {
	Auth   *auth.Handler
	Web    *handlers.Handler
	Worker *worker.Handler[domain.ReviewRequest]
	// TaskAuth は Cloud Tasks からの OIDC を検証します。Auth と違い OAuth 設定を
	// 必要としないため、検証だけを担う独立した部品として持ちます。
	TaskAuth *auth.TaskVerifier
}

// BuildHandlers は依存関係を注入し、各エンドポイント用のハンドラーを生成します。
func BuildHandlers(
	appCtx *app.Container,
) (*AppHandlers, error) {
	// Auth ハンドラーの生成
	authHandler, err := createAuthHandler(appCtx.Config)
	if err != nil {
		return nil, err
	}

	// Web UI ハンドラーの生成
	webHandler, err := handlers.NewHandler(appCtx.Config, appCtx.TaskEnqueuer, appCtx.RemoteIO)
	if err != nil {
		return nil, fmt.Errorf("WebHandlerの初期化失敗: %w", err)
	}

	// Worker ハンドラーと、その入口を守る Cloud Tasks OIDC 検証器の生成。
	// audience と発行元サービスアカウントの両方が揃わないと検証は常に失敗する
	// （fail-closed）ため、起動時に構成を確かめておきます。
	workerHandler := worker.NewHandler[domain.ReviewRequest](appCtx.Pipeline)
	taskAuth := auth.NewTaskVerifier(appCtx.Config.TaskAudienceURL, []string{appCtx.Config.ServiceAccountEmail})
	if !taskAuth.Configured() {
		return nil, fmt.Errorf("cloud Tasks の OIDC 検証を構成できません: TASK_AUDIENCE_URL と SERVICE_ACCOUNT_EMAIL の両方が必要です")
	}

	return &AppHandlers{
		Auth:     authHandler,
		Web:      webHandler,
		Worker:   workerHandler,
		TaskAuth: taskAuth,
	}, nil
}

// createAuthHandler は、提供された設定(Config)に基づいて認証ハンドラーを初期化して返します。
func createAuthHandler(cfg *config.Config) (*auth.Handler, error) {
	redirectURL, err := url.JoinPath(cfg.ServiceURL, "/auth/callback")
	if err != nil {
		return nil, fmt.Errorf("リダイレクトURLの構築失敗: %w", err)
	}

	return auth.NewHandler(auth.Config{
		ClientID:          cfg.GoogleClientID,
		ClientSecret:      cfg.GoogleClientSecret,
		RedirectURL:       redirectURL,
		SessionAuthKey:    cfg.SessionSecret,
		SessionEncryptKey: cfg.SessionEncryptKey,
		SessionName:       defaultSessionName,
		IsSecureCookie:    cfg.IsSecureServiceURL(),
		AllowedEmails:     cfg.AllowedEmails,
		AllowedDomains:    cfg.AllowedDomains,
	})
}
