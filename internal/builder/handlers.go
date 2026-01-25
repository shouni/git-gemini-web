package builder

import (
	"context"
	"fmt"
	"net/url"

	"git-gemini-web/internal/domain"
	"git-gemini-web/internal/server/handlers"

	"github.com/shouni/gcp-kit/auth"
	"github.com/shouni/gcp-kit/worker"
)

const defaultSessionName = "git-gemini-session"

// taskExecutor は、非同期タスクを受け取りレビュー処理のパイプラインを実行するインターフェースです。
type taskExecutor interface {
	Execute(ctx context.Context, payload domain.ReviewRequest) error
}

// AppHandlers は生成されたすべての HTTP ハンドラーを保持する構造体です。
type AppHandlers struct {
	Auth   *auth.Handler
	Web    *handlers.Handler
	Worker *worker.Handler[domain.ReviewRequest]
}

// BuildHandlers は依存関係を注入し、各エンドポイント用のハンドラーを生成します。
func BuildHandlers(appCtx *AppContext, reviewPipeline taskExecutor) (*AppHandlers, error) {
	// Auth ハンドラーの生成
	authHandler, err := createAuthHandler(appCtx)
	if err != nil {
		return nil, err
	}

	// Web UI ハンドラーの生成
	webHandler, err := handlers.NewHandler(appCtx.Config, appCtx.TaskEnqueuer, appCtx.IOFactory)
	if err != nil {
		return nil, fmt.Errorf("WebHandlerの初期化失敗: %w", err)
	}

	// Worker ハンドラーの生成
	workerHandler := worker.NewHandler[domain.ReviewRequest](reviewPipeline)

	return &AppHandlers{
		Auth:   authHandler,
		Web:    webHandler,
		Worker: workerHandler,
	}, nil
}

func createAuthHandler(appCtx *AppContext) (*auth.Handler, error) {
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
