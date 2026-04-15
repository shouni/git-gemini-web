package server

import (
	"bytes"
	"html/template"
	"log/slog"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/gorilla/csrf"

	"git-gemini-web/assets"
	"git-gemini-web/internal/builder"
	"git-gemini-web/internal/config"
)

const csrfErrorTemplatePath = "templates/csrf_error.html"

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
	tmpl, err := template.ParseFS(assets.Templates, csrfErrorTemplatePath)
	if err != nil {
		panic("failed to parse csrf error template: " + err.Error())
	}

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if reason := csrf.FailureReason(r); reason != nil {
			slog.WarnContext(r.Context(), "csrf validation failed", "error", reason)
		}

		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		w.WriteHeader(http.StatusForbidden)

		var buf bytes.Buffer
		if err := tmpl.Execute(&buf, nil); err != nil {
			slog.ErrorContext(r.Context(), "failed to render csrf error template", "error", err)
			_, _ = w.Write([]byte("セッションが無効です。ページを再読み込みして再送信してください。"))
			return
		}

		_, _ = buf.WriteTo(w)
	})
}
