package server

import (
	"net/http"

	"git-gemini-web/internal/builder"
	"git-gemini-web/internal/config"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/gorilla/csrf"
)

// NewRouter はハンドラーをルーティングに紐付けた http.Handler を返します。
// CSRF設定のために config.Config を引数に追加するのが望ましいのだ。
func NewRouter(h *builder.AppHandlers, cfg *config.Config) http.Handler {
	r := chi.NewRouter()

	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)
	r.Use(middleware.CleanPath)

	// CSRFミドルウェアの設定
	CSRF := csrf.Protect(
		[]byte(cfg.SessionSecret),
		csrf.Path("/"),
	)

	// A. 公開ルート（認証もCSRFも不要なログイン周り）
	r.Route("/auth", func(r chi.Router) {
		r.Get("/login", h.Auth.Login)
		r.Get("/callback", h.Auth.Callback)
	})

	// B. 認証が必要なルート (Web UI)
	r.Group(func(r chi.Router) {
		r.Use(h.Auth.Middleware) // 先にユーザー認証
		r.Use(CSRF)              // 次にCSRF保護を適用

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
