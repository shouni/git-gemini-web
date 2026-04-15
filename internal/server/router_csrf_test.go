package server

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestCSRFErrorHandler_ReturnsHTML(t *testing.T) {
	h := csrfErrorHandler()
	req := httptest.NewRequest(http.MethodPost, "/submit_review", nil)
	w := httptest.NewRecorder()

	h.ServeHTTP(w, req)

	if w.Code != http.StatusForbidden {
		t.Fatalf("unexpected status: got %d, want %d", w.Code, http.StatusForbidden)
	}
	if ct := w.Header().Get("Content-Type"); !strings.Contains(ct, "text/html") {
		t.Fatalf("unexpected content-type: %s", ct)
	}
	body := w.Body.String()
	if !strings.Contains(body, "セッションが無効です") {
		t.Fatalf("unexpected body: %s", body)
	}
	if !strings.Contains(body, "フォームに戻る") {
		t.Fatalf("unexpected body: %s", body)
	}
}
