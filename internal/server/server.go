package server

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"

	"git-gemini-web/internal/builder"
	"git-gemini-web/internal/config"
	"git-gemini-web/internal/pipeline"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
)

// NewRouter はハンドラーをルーティングに紐付けた http.Handler を返します。
func NewRouter(h *builder.AppHandlers) http.Handler {
	r := chi.NewRouter()

	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)
	r.Use(middleware.CleanPath)

	// A. 公開ルート
	r.Route("/auth", func(r chi.Router) {
		r.Get("/login", h.Auth.Login)
		r.Get("/callback", h.Auth.Callback)
	})

	// B. 認証が必要なルート (Web UI)
	r.Group(func(r chi.Router) {
		r.Use(h.Auth.Middleware)
		r.Get("/", h.Web.HandleReviewForm)
		r.Post("/submit_review", h.Web.HandleReviewSubmit)
	})

	// C. ワーカー専用ルート (OIDC認証)
	r.Group(func(r chi.Router) {
		r.Use(h.Auth.TaskOIDCVerificationMiddleware)
		r.Post("/tasks/execute_review", h.Worker.ProcessTask)
	})

	return r
}

// Run は、設定ロード、バリデーション、サーバーのライフサイクル管理を行う
func Run(ctx context.Context) error {
	// 1. 設定のロード
	cfg := config.LoadConfig()

	// 2. 必須設定のチェックとセキュリティバリデーション
	if err := config.ValidateEssentialConfig(cfg); err != nil {
		return err
	}

	slog.Info("🛠️ サーバー依存関係を構築中...")

	// 3. サーバーハンドラーの構築と依存関係の取得
	appCtx, err := builder.BuildAppContext(ctx, cfg)
	if err != nil {
		return fmt.Errorf("アプリケーションコンテキストの構築に失敗しました: %w", err)
	}

	// リソースを解放する
	defer func() {
		slog.Info("♻️ アプリケーションコンテキストをクローズ中...")
		appCtx.Close()
	}()

	// 4. パイプラインの構築
	reviewPipeline, err := pipeline.NewReviewPipeline(ctx, appCtx)
	if err != nil {
		return fmt.Errorf("reviewPipelineの構築に失敗しました: %w", err)
	}

	// 5. ハンドラーの組み立て (builder/handlers.go を使用)
	appHandlers, err := builder.BuildHandlers(appCtx, reviewPipeline)
	if err != nil {
		return fmt.Errorf("ハンドラーの構築に失敗しました: %w", err)
	}

	// 6. ルーターの作成 (このファイル内の NewRouter を使用)
	router := NewRouter(appHandlers)

	// 7. サーバー起動
	slog.Info("🚀 サーバーを起動中...", "port", cfg.Port)
	// Cloud Runでは SIGTERM を受け取ってからシャットダウンする Graceful Shutdown が推奨されますが、
	// まずはこのシンプルな ListenAndServe でも動作に問題はありません。
	if err := http.ListenAndServe(":"+cfg.Port, router); err != nil && err != http.ErrServerClosed {
		return fmt.Errorf("サーバーの起動に失敗しました: %w", err)
	}

	return nil
}
