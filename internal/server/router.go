package server

import (
	"bytes"
	"html/template"
	"log/slog"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"

	"git-gemini-web/assets"
	"git-gemini-web/internal/builder"
	"git-gemini-web/internal/config"
)

const (
	layoutTemplatePath           = "templates/layout.html"
	crossOriginErrorTemplatePath = "templates/csrf_error.html"
)

// NewRouter はハンドラーをルーティングに紐付けた http.Handler を返します。
func NewRouter(h *builder.AppHandlers, cfg *config.Config) http.Handler {
	r := chi.NewRouter()

	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)
	r.Use(middleware.CleanPath)

	// 同一オリジンのブラウザ送信だけを許可する。
	crossOriginProtection := http.NewCrossOriginProtection()
	crossOriginProtection.SetDenyHandler(crossOriginErrorHandler())

	// A. 公開ルート（認証もCSRFも不要なログイン周り）
	r.Route("/auth", func(r chi.Router) {
		r.Get("/login", h.Auth.Login)
		r.Get("/callback", h.Auth.Callback)
	})

	// B. 認証が必要なルート (Web UI)
	r.Group(func(r chi.Router) {
		r.Use(h.Auth.Middleware)             // 先にユーザー認証
		r.Use(crossOriginProtection.Handler) // 次に同一オリジン保護を適用

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

func crossOriginErrorHandler() http.Handler {
	tmpl := template.Must(template.ParseFS(
		assets.Templates,
		layoutTemplatePath,
		crossOriginErrorTemplatePath,
	))

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		slog.WarnContext(r.Context(), "cross-origin request blocked", "method", r.Method, "path", r.URL.Path)

		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		w.WriteHeader(http.StatusForbidden)

		var buf bytes.Buffer
		if err := tmpl.ExecuteTemplate(&buf, "layout.html", nil); err != nil {
			slog.ErrorContext(r.Context(), "failed to render cross-origin error template", "error", err)
			w.Header().Set("Content-Type", "text/plain; charset=utf-8")
			_, _ = w.Write([]byte("送信元を確認できなかったため、リクエストをブロックしました。ページを開き直して再送信してください。"))
			return
		}

		_, _ = buf.WriteTo(w)
	})
}
