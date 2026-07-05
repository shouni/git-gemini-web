package handlers

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"strings"

	"github.com/shouni/git-gemini-web/internal/config"
	"github.com/shouni/git-gemini-web/internal/domain"
)

// HandleReviewSubmit は POST リクエストを処理します。
func (h *Handler) HandleReviewSubmit(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	if err := r.ParseForm(); err != nil {
		data := defaultReviewFormPageData()
		data.Error = "リクエストのパースに失敗しました。"
		h.renderForm(w, r, http.StatusBadRequest, data)
		return
	}

	// 1. フォーム値の取得
	// Note: ModelNameなどは共通設定から取得
	req := domain.ReviewRequest{
		RepoURL:       strings.TrimSpace(r.PostFormValue("repo_url")),
		BaseBranch:    strings.TrimSpace(r.PostFormValue("base_branch")),
		FeatureBranch: strings.TrimSpace(r.PostFormValue("feature_branch")),
		Mode:          r.PostFormValue("review_mode"),
		ModelName:     r.PostFormValue("gemini_model"),
	}

	// 2. 入力バリデーション
	if err := h.validateReviewRequest(req); err != nil {
		h.renderForm(w, r, http.StatusBadRequest, reviewFormPageData(req, ReviewFormPageData{
			Error: err.Error(),
		}))
		return
	}

	// 3. 保存先 URI の決定
	req.StorageURI = h.cfg.StorageURI(req.RepoURL, req.FeatureBranch, h.now())

	// 4. 結果表示用の署名付きURLを事前に生成し、Request に保持させる
	publicURL, err := h.generateSignedResultURL(ctx, req.StorageURI)
	if err != nil {
		slog.ErrorContext(ctx, "署名付きURLの生成失敗", "error", err)
		h.renderForm(w, r, http.StatusInternalServerError, reviewFormPageData(req, ReviewFormPageData{
			Error: "内部サーバーエラーが発生しました。",
		}))
		return
	}
	req.PublicURL = publicURL

	// 5. Cloud Tasks へのタスク投入
	if err := h.taskEnqueuer.Enqueue(ctx, req); err != nil {
		slog.ErrorContext(ctx, "Cloud Tasksへの投入失敗", "error", err, "repo", req.RepoURL)
		h.renderForm(w, r, http.StatusServiceUnavailable, reviewFormPageData(req, ReviewFormPageData{
			Error: "現在レビューの受け付けができません。時間をおいて再度お試しください。",
		}))
		return
	}

	// 6. 成功応答を返します
	slog.InfoContext(ctx, "レビュータスク投入成功", "repo", req.RepoURL, "storage_uri", req.StorageURI)
	h.renderForm(w, r, http.StatusAccepted, reviewFormPageData(req, ReviewFormPageData{
		Message:   fmt.Sprintf("✅ レビュータスクを受け付けました。生成完了後、以下のURLから確認できます（%s有効）。", config.SignedURLExpiration.String()),
		ResultURL: req.PublicURL,
	}))
}

func defaultReviewFormPageData() ReviewFormPageData {
	return ReviewFormPageData{
		BaseBranch:    defaultBaseBranch,
		FeatureBranch: defaultFeatureBranch,
		ReviewMode:    defaultReviewMode,
	}
}

func reviewFormPageData(req domain.ReviewRequest, data ReviewFormPageData) ReviewFormPageData {
	data.RepoURL = req.RepoURL
	data.BaseBranch = req.BaseBranch
	data.FeatureBranch = req.FeatureBranch
	data.ReviewMode = req.Mode
	data.GeminiModel = req.ModelName
	return data
}

// generateSignedResultURL は StorageURI から署名付きURLを作るヘルパーです。
func (h *Handler) generateSignedResultURL(ctx context.Context, storageURI string) (string, error) {
	return h.remoteIO.Signer.GenerateSignedURL(ctx, storageURI, "GET", config.SignedURLExpiration)
}
