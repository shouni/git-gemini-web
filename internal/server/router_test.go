package server

import (
	"context"
	"net/http"
	"net/http/httptest"
	"net/url"
	"regexp"
	"strings"
	"testing"

	"github.com/gorilla/sessions"
	"github.com/shouni/gcp-kit/auth"
	"github.com/shouni/gcp-kit/worker"

	"github.com/shouni/git-gemini-web/internal/app"
	"github.com/shouni/git-gemini-web/internal/builder"
	"github.com/shouni/git-gemini-web/internal/config"
	"github.com/shouni/git-gemini-web/internal/domain"
	"github.com/shouni/git-gemini-web/internal/server/handlers"
)

type noopTaskEnqueuer struct{}

func (noopTaskEnqueuer) Enqueue(context.Context, domain.ReviewRequest) error { return nil }
func (noopTaskEnqueuer) Close() error                                        { return nil }

type noopPipeline struct{}

func (noopPipeline) Execute(context.Context, domain.ReviewRequest) error { return nil }

func authenticatedSessionCookie(t *testing.T) *http.Cookie {
	t.Helper()

	store := sessions.NewCookieStore([]byte("1234567890abcdef"), []byte("1234567890123456"))
	store.Options = &sessions.Options{
		Path:     "/",
		HttpOnly: true,
		Secure:   true,
		SameSite: http.SameSiteLaxMode,
	}

	req := httptest.NewRequest(http.MethodGet, "https://service.example.com/", nil)
	w := httptest.NewRecorder()
	session, err := store.Get(req, "test-session")
	if err != nil {
		t.Fatalf("failed to create session: %v", err)
	}
	session.Values[auth.DefaultUserSessionKey] = "tester@example.com"
	if err := session.Save(req, w); err != nil {
		t.Fatalf("failed to save session: %v", err)
	}

	cookies := w.Result().Cookies()
	if len(cookies) == 0 {
		t.Fatal("session cookie was not set")
	}
	return cookies[0]
}

func newRouterForTest(t *testing.T) http.Handler {
	t.Helper()

	cfg := &config.Config{
		ServiceURL:          "https://service.example.com",
		TaskAudienceURL:     "https://service.example.com",
		ServiceAccountEmail: "tasks@example.iam.gserviceaccount.com",
		GoogleClientID:      "client-id",
		GoogleClientSecret:  "client-secret",
		SessionSecret:       "1234567890abcdef",
		SessionEncryptKey:   "1234567890123456",
		AllowedEmails:       []string{"tester@example.com"},
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
		// TaskAudienceURL を設定する場合、許可サービスアカウントの指定は必須。
		AllowedTaskServiceAccounts: []string{cfg.ServiceAccountEmail},
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
	return NewRouter(appHandlers)
}

func TestNewRouter_RouteReachabilityAndGuards(t *testing.T) {
	r := newRouterForTest(t)

	tests := []struct {
		name         string
		method       string
		path         string
		expectedCode int
	}{
		{
			name:         "auth login is reachable",
			method:       http.MethodGet,
			path:         "/auth/login",
			expectedCode: http.StatusTemporaryRedirect,
		},
		{
			name:         "root requires auth",
			method:       http.MethodGet,
			path:         "/",
			expectedCode: http.StatusFound,
		},
		{
			name:         "worker route requires oidc",
			method:       http.MethodPost,
			path:         "/tasks/execute_review",
			expectedCode: http.StatusUnauthorized,
		},
		{
			name:         "unknown route is 404",
			method:       http.MethodGet,
			path:         "/not-found",
			expectedCode: http.StatusNotFound,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(tt.method, tt.path, nil)
			w := httptest.NewRecorder()
			r.ServeHTTP(w, req)

			if w.Code != tt.expectedCode {
				t.Fatalf("unexpected status for %s %s: got %d, want %d", tt.method, tt.path, w.Code, tt.expectedCode)
			}
		})
	}
}

func TestNewRouter_FormRendersCSRFTokenAndSubmitUsesIt(t *testing.T) {
	r := newRouterForTest(t)
	sessionCookie := authenticatedSessionCookie(t)

	getReq := httptest.NewRequest(http.MethodGet, "https://service.example.com/", nil)
	getReq.AddCookie(sessionCookie)
	getW := httptest.NewRecorder()
	r.ServeHTTP(getW, getReq)

	if getW.Code != http.StatusOK {
		t.Fatalf("GET / unexpected status: got %d want %d", getW.Code, http.StatusOK)
	}

	tokenMatch := regexp.MustCompile(`name="csrf_token" value="([^"]+)"`).FindStringSubmatch(getW.Body.String())
	if len(tokenMatch) != 2 {
		t.Fatalf("csrf token was not rendered in form: %s", getW.Body.String())
	}
	csrfToken := tokenMatch[1]

	for _, cookie := range getW.Result().Cookies() {
		if cookie.Name == sessionCookie.Name {
			sessionCookie = cookie
			break
		}
	}

	form := url.Values{}
	form.Set("csrf_token", csrfToken)
	postReq := httptest.NewRequest(http.MethodPost, "https://service.example.com/submit_review", strings.NewReader(form.Encode()))
	postReq.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	postReq.Header.Set("Origin", "https://service.example.com")
	postReq.AddCookie(sessionCookie)
	postW := httptest.NewRecorder()
	r.ServeHTTP(postW, postReq)

	if postW.Code != http.StatusBadRequest {
		t.Fatalf("POST with rendered csrf token should reach form validation: got %d want %d", postW.Code, http.StatusBadRequest)
	}
}
