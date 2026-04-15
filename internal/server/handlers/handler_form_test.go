package handlers

import (
	"html"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"git-gemini-web/internal/app"
	"git-gemini-web/internal/config"
)

func TestHandleReviewForm_RendersValidationPatterns(t *testing.T) {
	h, err := NewHandler(&config.Config{}, &fakeEnqueuer{}, &app.RemoteIO{})
	if err != nil {
		t.Fatalf("failed to create handler: %v", err)
	}

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	w := httptest.NewRecorder()
	h.HandleReviewForm(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("unexpected status: got %d want %d", w.Code, http.StatusOK)
	}
	body := html.UnescapeString(w.Body.String())
	if !strings.Contains(body, `name="repo_url"`) ||
		!strings.Contains(body, `pattern="`+repoURLPattern+`"`) {
		t.Fatalf("repo url pattern not rendered: %s", body)
	}
	if !strings.Contains(body, `name="base_branch"`) ||
		!strings.Contains(body, `name="feature_branch"`) ||
		!strings.Contains(body, `pattern="`+branchPattern+`"`) {
		t.Fatalf("branch pattern not rendered: %s", body)
	}
}
