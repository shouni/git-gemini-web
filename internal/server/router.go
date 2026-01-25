package server

import (
	"net/http"

	"git-gemini-web/internal/builder"

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
