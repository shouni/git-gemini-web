package server

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/shouni/gcp-kit/auth"
	"github.com/shouni/gcp-kit/worker"

	"git-gemini-web/internal/app"
	"git-gemini-web/internal/builder"
	"git-gemini-web/internal/config"
	"git-gemini-web/internal/domain"
	"git-gemini-web/internal/server/handlers"
)

type noopTaskEnqueuer struct{}

func (noopTaskEnqueuer) Enqueue(context.Context, domain.ReviewRequest) error { return nil }

type noopPipeline struct{}

func (noopPipeline) Execute(context.Context, domain.ReviewRequest) error { return nil }

func newRouterForTest(t *testing.T) http.Handler {
	t.Helper()

	cfg := &config.Config{
		ServiceURL:         "https://service.example.com",
		TaskAudienceURL:    "https://service.example.com",
		GoogleClientID:     "client-id",
		GoogleClientSecret: "client-secret",
		SessionSecret:      "1234567890abcdef",
		SessionEncryptKey:  "1234567890123456",
		AllowedEmails:      []string{"tester@example.com"},
	}

	authHandler, err := auth.NewHandler(auth.Config{
		ClientID:          cfg.GoogleClientID,
		ClientSecret:      cfg.GoogleClientSecret,
		RedirectURL:       cfg.ServiceURL + "/auth/callback",
		SessionAuthKey:    cfg.SessionSecret,
		SessionEncryptKey: cfg.SessionEncryptKey,
		SessionName:       "test-session",
		IsSecureCookie:    true,
		AllowedEmails:     cfg.AllowedEmails,
		TaskAudienceURL:   cfg.TaskAudienceURL,
	})
	if err != nil {
		t.Fatalf("failed to create auth handler: %v", err)
	}

	webHandler, err := handlers.NewHandler(cfg, noopTaskEnqueuer{}, &app.RemoteIO{})
	if err != nil {
		t.Fatalf("failed to create web handler: %v", err)
	}

	workerHandler := worker.NewHandler[domain.ReviewRequest](noopPipeline{})

	appHandlers := &builder.AppHandlers{
		Auth:   authHandler,
		Web:    webHandler,
		Worker: workerHandler,
	}
	return NewRouter(appHandlers, cfg)
}

func TestNewRouter_RouteReachabilityAndGuards(t *testing.T) {
	r := newRouterForTest(t)

	tests := []struct {
		name    string
		method  string
		path    string
		allowed map[int]bool
	}{
		{
			name:   "auth login is reachable",
			method: http.MethodGet,
			path:   "/auth/login",
			allowed: map[int]bool{
				http.StatusFound:             true,
				http.StatusTemporaryRedirect: true,
			},
		},
		{
			name:   "root requires auth",
			method: http.MethodGet,
			path:   "/",
			allowed: map[int]bool{
				http.StatusFound:        true,
				http.StatusUnauthorized: true,
				http.StatusForbidden:    true,
			},
		},
		{
			name:   "worker route requires oidc",
			method: http.MethodPost,
			path:   "/tasks/execute_review",
			allowed: map[int]bool{
				http.StatusUnauthorized: true,
				http.StatusForbidden:    true,
			},
		},
		{
			name:   "unknown route is 404",
			method: http.MethodGet,
			path:   "/not-found",
			allowed: map[int]bool{
				http.StatusNotFound: true,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(tt.method, tt.path, nil)
			w := httptest.NewRecorder()
			r.ServeHTTP(w, req)

			if !tt.allowed[w.Code] {
				t.Fatalf("unexpected status for %s %s: got %d", tt.method, tt.path, w.Code)
			}
		})
	}
}
