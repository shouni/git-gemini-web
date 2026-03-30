package builder

import (
	"fmt"
	"net/url"

	"github.com/shouni/gcp-kit/auth"
	"github.com/shouni/gcp-kit/worker"
	"github.com/shouni/gemini-reviewer-core/ports"

	"git-gemini-web/internal/app"
	"git-gemini-web/internal/config"
	"git-gemini-web/internal/server/handlers"
)

const defaultSessionName = "git-gemini-session"

// AppHandlers は生成されたすべての HTTP ハンドラーを保持する構造体です。
type AppHandlers struct {
	Auth   *auth.Handler
	Web    *handlers.Handler
	Worker *worker.Handler[ports.ReviewRequest]
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

	// Worker ハンドラーの生成
	workerHandler := worker.NewHandler[ports.ReviewRequest](appCtx.Pipeline)

	return &AppHandlers{
		Auth:   authHandler,
		Web:    webHandler,
		Worker: workerHandler,
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
		TaskAudienceURL:   cfg.TaskAudienceURL,
	})
}
