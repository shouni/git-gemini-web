package handlers

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"

	"git-gemini-web/internal/app"
	"git-gemini-web/internal/config"
	"git-gemini-web/internal/domain"
)

type fakeSigner struct {
	url string
	err error
}

func (f *fakeSigner) GenerateSignedURL(_ context.Context, _ string, _ string, _ time.Duration) (string, error) {
	if f.err != nil {
		return "", f.err
	}
	return f.url, nil
}

type fakeEnqueuer struct {
	err      error
	called   bool
	received domain.ReviewRequest
}

func (f *fakeEnqueuer) Enqueue(_ context.Context, payload domain.ReviewRequest) error {
	f.called = true
	f.received = payload
	return f.err
}

func buildTestHandler(t *testing.T, signerErr, enqueueErr error) (*Handler, *fakeEnqueuer) {
	t.Helper()
	enq := &fakeEnqueuer{err: enqueueErr}
	h, err := NewHandler(
		&config.Config{
			GeminiModel:  "gemini-2.5-flash",
			GeminiModels: []string{"gemini-2.5-flash", "gemini-2.5-pro"},
			GCSBucket:    "bucket-a",
		},
		enq,
		&app.RemoteIO{Signer: &fakeSigner{url: "https://signed.example.com/result.html", err: signerErr}},
	)
	if err != nil {
		t.Fatalf("failed to build handler: %v", err)
	}
	h.now = func() time.Time {
		return time.Date(2026, 4, 15, 10, 20, 30, 0, time.UTC)
	}
	return h, enq
}

func newSubmitRequest(body string) *http.Request {
	req := httptest.NewRequest(http.MethodPost, "/submit_review", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	return req
}

func validFormBody() string {
	v := url.Values{}
	v.Set("repo_url", "https://github.com/org/repo.git")
	v.Set("base_branch", "main")
	v.Set("feature_branch", "feature/new-ui")
	v.Set("review_mode", "detail")
	v.Set("gemini_model", "gemini-2.5-flash")
	return v.Encode()
}

func TestHandleReviewSubmit_ParseError(t *testing.T) {
	h, enq := buildTestHandler(t, nil, nil)
	w := httptest.NewRecorder()
	h.HandleReviewSubmit(w, newSubmitRequest("%zz"))

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected status 400, got %d", w.Code)
	}
	if enq.called {
		t.Fatal("enqueue should not be called on parse error")
	}
}

func TestHandleReviewSubmit_ValidationError(t *testing.T) {
	h, enq := buildTestHandler(t, nil, nil)
	w := httptest.NewRecorder()

	v := url.Values{}
	v.Set("repo_url", "https://github.com/org/repo.git")
	v.Set("base_branch", "main")
	v.Set("feature_branch", "feature/new-ui")
	v.Set("review_mode", "invalid-mode")
	v.Set("gemini_model", "gemini-2.5-flash")

	h.HandleReviewSubmit(w, newSubmitRequest(v.Encode()))

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected status 400, got %d", w.Code)
	}
	if enq.called {
		t.Fatal("enqueue should not be called on validation error")
	}
}

func TestHandleReviewSubmit_InvalidGeminiModel(t *testing.T) {
	h, enq := buildTestHandler(t, nil, nil)
	w := httptest.NewRecorder()

	v := url.Values{}
	v.Set("repo_url", "https://github.com/org/repo.git")
	v.Set("base_branch", "main")
	v.Set("feature_branch", "feature/new-ui")
	v.Set("review_mode", "detail")
	v.Set("gemini_model", "gemini-invalid")

	h.HandleReviewSubmit(w, newSubmitRequest(v.Encode()))

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected status 400, got %d", w.Code)
	}
	if enq.called {
		t.Fatal("enqueue should not be called on invalid gemini model")
	}
}

func TestHandleReviewSubmit_SignerError(t *testing.T) {
	h, enq := buildTestHandler(t, errors.New("sign error"), nil)
	w := httptest.NewRecorder()
	h.HandleReviewSubmit(w, newSubmitRequest(validFormBody()))

	if w.Code != http.StatusInternalServerError {
		t.Fatalf("expected status 500, got %d", w.Code)
	}
	if enq.called {
		t.Fatal("enqueue should not be called when signer fails")
	}
}

func TestHandleReviewSubmit_EnqueueError(t *testing.T) {
	h, enq := buildTestHandler(t, nil, errors.New("queue unavailable"))
	w := httptest.NewRecorder()
	h.HandleReviewSubmit(w, newSubmitRequest(validFormBody()))

	if w.Code != http.StatusServiceUnavailable {
		t.Fatalf("expected status 503, got %d", w.Code)
	}
	if !enq.called {
		t.Fatal("enqueue should be called before returning 503")
	}
}

func TestHandleReviewSubmit_Success(t *testing.T) {
	h, enq := buildTestHandler(t, nil, nil)
	w := httptest.NewRecorder()
	h.HandleReviewSubmit(w, newSubmitRequest(validFormBody()))

	if w.Code != http.StatusAccepted {
		t.Fatalf("expected status 202, got %d", w.Code)
	}
	if !enq.called {
		t.Fatal("enqueue should be called on success")
	}
	if enq.received.ModelName != "gemini-2.5-flash" {
		t.Fatalf("unexpected model name: %s", enq.received.ModelName)
	}
	if enq.received.PublicURL != "https://signed.example.com/result.html" {
		t.Fatalf("unexpected public url: %s", enq.received.PublicURL)
	}
	if !strings.Contains(enq.received.StorageURI, "feature-new-ui.html") {
		t.Fatalf("unexpected storage uri: %s", enq.received.StorageURI)
	}
	if body := w.Body.String(); !strings.Contains(body, "https://signed.example.com/result.html") {
		t.Fatalf("response should include result URL, body=%q", body)
	}
}

func TestHandleReviewSubmit_UsesSelectedGeminiModel(t *testing.T) {
	h, enq := buildTestHandler(t, nil, nil)
	w := httptest.NewRecorder()

	v, err := url.ParseQuery(validFormBody())
	if err != nil {
		t.Fatalf("failed to parse valid form body: %v", err)
	}
	v.Set("gemini_model", "gemini-2.5-pro")

	h.HandleReviewSubmit(w, newSubmitRequest(v.Encode()))

	if w.Code != http.StatusAccepted {
		t.Fatalf("expected status 202, got %d", w.Code)
	}
	if enq.received.ModelName != "gemini-2.5-pro" {
		t.Fatalf("unexpected model name: %s", enq.received.ModelName)
	}
}
