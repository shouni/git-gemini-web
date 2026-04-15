package handlers

import (
	"context"
	"fmt"
	"html/template"
	"net/http"
	"time"

	"git-gemini-web/assets"
	"git-gemini-web/internal/app"
	"git-gemini-web/internal/config"
	"git-gemini-web/internal/domain"
)

// ReviewFormPageData はフォームテンプレートに渡すデータ構造です。
type ReviewFormPageData struct {
	Message        string
	Error          string
	ResultURL      string
	CSRFToken      string
	RepoURLPattern string
	BranchPattern  string
}

type reviewTaskEnqueuer interface {
	Enqueue(ctx context.Context, payload domain.ReviewRequest) error
}

// Handler は HTTPリクエストを処理する構造体です。
type Handler struct {
	cfg          *config.Config
	taskEnqueuer reviewTaskEnqueuer
	remoteIO     *app.RemoteIO
	template     *template.Template
	now          func() time.Time
}

// NewHandler は新しい Handler インスタンスを作成します。
func NewHandler(
	cfg *config.Config,
	taskEnqueuer reviewTaskEnqueuer,
	remoteIO *app.RemoteIO,
) (*Handler, error) {
	tmpl, err := template.ParseFS(assets.Templates, "templates/*.html")
	if err != nil {
		return nil, fmt.Errorf("テンプレートパース失敗: %w", err)
	}

	return &Handler{
		cfg:          cfg,
		taskEnqueuer: taskEnqueuer,
		remoteIO:     remoteIO,
		template:     tmpl,
		now:          time.Now,
	}, nil
}

// HandleReviewForm は GET リクエストに対してフォームを表示します。
func (h *Handler) HandleReviewForm(w http.ResponseWriter, r *http.Request) {
	h.renderForm(w, r, http.StatusOK, ReviewFormPageData{})
}
