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
	"git-gemini-web/internal/server/handlers"
)

const (
	layoutTemplatePath           = "templates/layout.html"
	crossOriginErrorTemplatePath = "templates/cross_origin_error.html"
)

// NewRouter は、ミドルウェアとルーティングを統合した http.Handler を構築します。
func NewRouter(h *builder.AppHandlers) http.Handler {
	r := chi.NewRouter()
	setupCommonMiddleware(r)
	setupRoutes(r, h)

	return r
}

// setupCommonMiddleware は、標準的なミドルウェアを構成します。
func setupCommonMiddleware(r *chi.Mux) {
	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)
	r.Use(middleware.CleanPath)
}

// setupRoutes は、各コンポーネントのハンドラーをルーティングに登録します。
func setupRoutes(r chi.Router, h *builder.AppHandlers) {
	// --- 1. 公開ルート (ヘルスチェック) ---
	r.Get("/healthz", func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("ok"))
	})

	if h == nil {
		slog.Warn("AppHandlers is nil, skipping application routes registration")
		return
	}

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
		if h.Auth == nil {
			if h.Web != nil {
				slog.Error("Auth handler is nil, skipping protected web routes")
			}
			return
		}

		r.Use(h.Auth.Middleware)

		// CSRFトークンがなければ自動生成してセッションに保存するミドルウェア
		r.Use(func(next http.Handler) http.Handler {
			return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				csrfToken := h.Auth.GetCSRFTokenFromSession(r)
				if csrfToken == "" && r.Method == http.MethodGet {
					token, err := h.Auth.GenerateAndSaveCSRFToken(w, r)
					if err != nil {
						slog.Error("Failed to auto-generate CSRF token", "error", err)
						http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
						return
					}
					csrfToken = token
				}
				r = r.WithContext(handlers.WithCSRFToken(r.Context(), csrfToken))
				next.ServeHTTP(w, r)
			})
		})

		r.Use(crossOriginProtection.Handler)

		r.Get("/", h.Web.HandleReviewForm)
		r.Post("/submit_review", h.Web.HandleReviewSubmit)
	})

	// C. ワーカー専用ルート (OIDC認証)
	r.Group(func(r chi.Router) {
		if h.Auth == nil {
			if h.Worker != nil {
				slog.Error("Auth handler is nil, skipping worker routes")
			}
			return
		}

		r.Use(h.Auth.TaskOIDCVerificationMiddleware)

		if h.Worker != nil {
			r.Post("/tasks/execute_review", h.Worker.ProcessTask)
		}
	})
}

// crossOriginErrorHandler returns a handler for requests blocked by cross-origin protection.
func crossOriginErrorHandler() http.Handler {
	tmpl := template.Must(template.ParseFS(
		assets.Templates,
		layoutTemplatePath,
		crossOriginErrorTemplatePath,
	))

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		writeCrossOriginErrorResponse(w, r, tmpl)
	})
}

// writeCrossOriginErrorResponse renders a forbidden response for blocked cross-origin requests.
func writeCrossOriginErrorResponse(w http.ResponseWriter, r *http.Request, tmpl *template.Template) {
	slog.WarnContext(r.Context(), "cross-origin request blocked", "method", r.Method, "path", r.URL.Path)

	var buf bytes.Buffer
	if err := tmpl.ExecuteTemplate(&buf, "layout.html", nil); err != nil {
		slog.ErrorContext(r.Context(), "failed to render cross-origin error template", "error", err)
		w.Header().Set("Content-Type", "text/plain; charset=utf-8")
		w.WriteHeader(http.StatusForbidden)
		_, _ = w.Write([]byte("送信元を確認できなかったため、リクエストをブロックしました。ページを開き直して再送信してください。"))
		return
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.WriteHeader(http.StatusForbidden)
	_, _ = buf.WriteTo(w)
}
