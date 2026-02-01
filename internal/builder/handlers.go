package builder

import (
	"fmt"
	"git-gemini-web/internal/app"
	"net/url"

	"git-gemini-web/internal/domain"
	"git-gemini-web/internal/server/handlers"

	"github.com/shouni/gcp-kit/auth"
	"github.com/shouni/gcp-kit/worker"
)

const defaultSessionName = "git-gemini-session"

// AppHandlers は生成されたすべての HTTP ハンドラーを保持する構造体です。
type AppHandlers struct {
	Auth   *auth.Handler
	Web    *handlers.Handler
	Worker *worker.Handler[domain.ReviewRequest]
}

// BuildHandlers は依存関係を注入し、各エンドポイント用のハンドラーを生成します。
func BuildHandlers(
	appCtx *app.Container,
) (*AppHandlers, error) {
	// Auth ハンドラーの生成
	authHandler, err := createAuthHandler(appCtx)
	if err != nil {
		return nil, err
	}

	// Web UI ハンドラーの生成
	webHandler, err := handlers.NewHandler(appCtx.Config, appCtx.TaskEnqueuer, appCtx.RemoteIO)
	if err != nil {
		return nil, fmt.Errorf("WebHandlerの初期化失敗: %w", err)
	}

	// Worker ハンドラーの生成
	workerHandler := worker.NewHandler[domain.ReviewRequest](appCtx.Pipeline)

	return &AppHandlers{
		Auth:   authHandler,
		Web:    webHandler,
		Worker: workerHandler,
	}, nil
}

// createAuthHandler は、アプリケーション コンテキスト設定で構成された認証ハンドラーを初期化して返します。
func createAuthHandler(appCtx *app.Container) (*auth.Handler, error) {
	cfg := appCtx.Config
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
		IsSecureCookie:    appCtx.HTTPClient.IsSecureServiceURL(cfg.ServiceURL),
		AllowedEmails:     cfg.AllowedEmails,
		AllowedDomains:    cfg.AllowedDomains,
		TaskAudienceURL:   cfg.TaskAudienceURL,
	})
}
