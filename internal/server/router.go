package server

import (
	"log/slog"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/gorilla/csrf"

	"git-gemini-web/internal/builder"
	"git-gemini-web/internal/config"
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
		[]byte(cfg.SessionEncryptKey),
		csrf.Path("/"),
		csrf.ErrorHandler(csrfErrorHandler()),
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

func csrfErrorHandler() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if reason := csrf.FailureReason(r); reason != nil {
			slog.WarnContext(r.Context(), "csrf validation failed", "error", reason)
		}

		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		w.WriteHeader(http.StatusForbidden)
		_, _ = w.Write([]byte(`<!DOCTYPE html>
<html lang="ja">
<head>
  <meta charset="UTF-8">
  <meta name="viewport" content="width=device-width, initial-scale=1.0">
  <title>セッションエラー</title>
</head>
<body>
  <h1>セッションが無効です</h1>
  <p>フォームの有効期限が切れたか、トークンが一致しませんでした。</p>
  <p>ページを再読み込みして、もう一度送信してください。</p>
  <p><a href="/">フォームに戻る</a></p>
</body>
</html>`))
	})
}
