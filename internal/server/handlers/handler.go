package handlers

import (
	"fmt"
	"html/template"
	"log/slog"
	"net/http"
	"strings"

	"git-gemini-web/internal/config"
	"git-gemini-web/internal/domain"

	"github.com/google/uuid"
	"github.com/shouni/gcp-kit/tasks"
	"github.com/shouni/go-remote-io/pkg/remoteio"
	"github.com/shouni/go-utils/urlpath"
)

// ReviewFormPageData はフォームテンプレートに渡すデータ構造です。
type ReviewFormPageData struct {
	Message   string
	Error     string
	ResultURL string
}

// Handler は HTTPリクエストを処理する構造体です。
type Handler struct {
	cfg          *config.Config
	taskEnqueuer *tasks.Enqueuer[domain.ReviewRequest]
	ioFactory    remoteio.IOFactory
	template     *template.Template
}

// NewHandler は新しい Handler インスタンスを作成します。
func NewHandler(
	cfg *config.Config,
	taskEnqueuer *tasks.Enqueuer[domain.ReviewRequest],
	ioFactory remoteio.IOFactory,
) (*Handler, error) {
	tmpl, err := template.ParseFiles(cfg.TemplatePath)
	if err != nil {
		return nil, fmt.Errorf("テンプレートパース失敗: %w", err)
	}

	return &Handler{
		cfg:          cfg,
		taskEnqueuer: taskEnqueuer,
		ioFactory:    ioFactory,
		template:     tmpl,
	}, nil
}

// HandleReviewForm は GET リクエストに対してフォームを表示します。
func (h *Handler) HandleReviewForm(w http.ResponseWriter, r *http.Request) {
	h.renderForm(w, http.StatusOK, ReviewFormPageData{})
}

// HandleReviewSubmit は POST リクエストを処理します。
func (h *Handler) HandleReviewSubmit(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	if err := r.ParseForm(); err != nil {
		h.renderForm(w, http.StatusBadRequest, ReviewFormPageData{Error: "リクエストのパースに失敗しました。"})
		return
	}

	// 1. フォーム値の取得
	req := domain.ReviewRequest{
		RepoURL:       strings.TrimSpace(r.FormValue("repo_url")),
		BaseBranch:    strings.TrimSpace(r.FormValue("base_branch")),
		FeatureBranch: strings.TrimSpace(r.FormValue("feature_branch")),
		Mode:          r.FormValue("review_mode"),
		GCSBucket:     h.cfg.GCSBucket,
	}

	// 2. 入力バリデーション
	if err := h.validateReviewRequest(req); err != nil {
		h.renderForm(w, http.StatusBadRequest, ReviewFormPageData{Error: err.Error()})
		return
	}

	// 3. ID生成と保存先パスの決定
	reviewID := uuid.New().String()
	repoID := urlpath.GenerateGCSKeyName(req.RepoURL)
	req.GCSPath = fmt.Sprintf("reviews/%s/%s/%s-%s.html", repoID, req.FeatureBranch, req.Mode, reviewID)

	// 4. 結果表示用の署名付きURLを事前に生成します
	publicURL, err := h.generateSignedResultURL(ctx, req.GCSPath)
	if err != nil {
		slog.Error("署名付きURLの生成失敗", "error", err)
		http.Error(w, "内部サーバーエラーが発生しました。", http.StatusInternalServerError)
		return
	}

	// 5. Cloud Tasks へのタスク投入
	if err := h.taskEnqueuer.Enqueue(ctx, req); err != nil {
		slog.Error("Cloud Tasksへの投入失敗", "error", err, "repo", req.RepoURL)
		h.renderForm(w, http.StatusServiceUnavailable, ReviewFormPageData{
			Error: "現在レビューの受け付けができません。時間をおいて再度お試しください。",
		})
		return
	}

	// 6. 成功応答を返します
	slog.Info("レビュータスク投入成功", "repo", req.RepoURL, "gcs_path", req.GCSPath)
	h.renderForm(w, http.StatusAccepted, ReviewFormPageData{
		Message:   "✅ レビュータスクを受け付けました。生成完了後、以下のURLから確認できます（15分間有効）。",
		ResultURL: publicURL,
	})
}
