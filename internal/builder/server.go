package builder

import (
	"context"
	"fmt"
	"net/http"
	"net/url"

	"git-gemini-web/internal/domain"
	"git-gemini-web/internal/server"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/shouni/gcp-kit/auth"
	"github.com/shouni/gcp-kit/worker"
)

const defaultSessionName = "git-gemini-session"

// taskExecutor は、非同期タスクを受け取りレビュー処理のパイプラインを実行するインターフェースです。
type taskExecutor interface {
	Execute(ctx context.Context, payload domain.ReviewRequest) error
}

// NewServerHandler は、アプリケーションのすべての依存関係を構築し、ルーティングを設定します。
func NewServerHandler(ctx context.Context, appCtx *AppContext, reviewPipeline taskExecutor) (http.Handler, error) {
	// 1. 各種ハンドラーの初期化
	authHandler, err := createAuthHandler(appCtx)
	if err != nil {
		return nil, fmt.Errorf("authHandlerの初期化に失敗しました: %w", err)
	}

	webHandler, err := server.NewHandler(appCtx.Config, appCtx.TaskEnqueuer, appCtx.IOFactory)
	if err != nil {
		return nil, fmt.Errorf("WebHandlerの初期化に失敗しました: %w", err)
	}

	workerHandler := worker.NewHandler[domain.ReviewRequest](reviewPipeline)

	// 2. ルーターの構築
	r := chi.NewRouter()

	// 標準ミドルウェア
	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)
	r.Use(middleware.CleanPath) // //path// を /path に正規化

	// --- A. 公開ルート (認証フロー) ---
	r.Route("/auth", func(r chi.Router) {
		r.Get("/login", authHandler.Login)
		r.Get("/callback", authHandler.Callback)
	})

	// --- B. 認証が必要なルート (Web UI) ---
	r.Group(func(r chi.Router) {
		r.Use(authHandler.Middleware)

		r.Get("/", webHandler.HandleReviewForm)
		r.Post("/submit_review", webHandler.HandleReviewSubmit)
	})

	// --- C. ワーカー専用ルート (OIDC認証で保護) ---
	// 誰でも叩ける状態を避け、Cloud Tasks からのリクエストのみを許可します
	r.Group(func(r chi.Router) {
		r.Use(authHandler.TaskOIDCVerificationMiddleware)
		r.Post("/tasks/execute_review", workerHandler.ProcessTask)
	})

	return r, nil
}

// createAuthHandler は AppContext から認証ライブラリ用の設定を構築し、ハンドラーを生成します。
func createAuthHandler(appCtx *AppContext) (*auth.Handler, error) {
	cfg := appCtx.Config
	redirectURL, err := url.JoinPath(cfg.ServiceURL, "/auth/callback")
	if err != nil {
		return nil, fmt.Errorf("リダイレクトURLの構築に失敗しました: %w", err)
	}

	isSecure := appCtx.HTTPClient.IsSecureServiceURL(cfg.ServiceURL)

	return auth.NewHandler(auth.Config{
		ClientID:          cfg.GoogleClientID,
		ClientSecret:      cfg.GoogleClientSecret,
		RedirectURL:       redirectURL,
		SessionAuthKey:    cfg.SessionSecret,
		SessionEncryptKey: cfg.SessionEncryptKey,
		SessionName:       defaultSessionName,
		IsSecureCookie:    isSecure,
		AllowedEmails:     cfg.AllowedEmails,
		AllowedDomains:    cfg.AllowedDomains,
		TaskAudienceURL:   cfg.TaskAudienceURL,
	})
}
