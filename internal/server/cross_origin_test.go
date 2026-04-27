package server

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestCrossOriginErrorHandler_ReturnsForbiddenPage(t *testing.T) {
	req := httptest.NewRequest(http.MethodPost, "/submit_review", nil)
	w := httptest.NewRecorder()

	crossOriginErrorHandler().ServeHTTP(w, req)

	if w.Code != http.StatusForbidden {
		t.Fatalf("unexpected status: got %d want %d", w.Code, http.StatusForbidden)
	}
	body := w.Body.String()
	if !strings.Contains(body, "送信元が許可されていません") {
		t.Fatalf("unexpected body: %s", body)
	}
}
