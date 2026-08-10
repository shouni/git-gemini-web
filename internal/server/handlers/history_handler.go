package handlers

import (
	"errors"
	"log/slog"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"
	"github.com/shouni/go-job-kit/jobstatus"
	"github.com/shouni/go-utils/jobid"

	"github.com/shouni/git-gemini-web/internal/domain"
)

const (
	// historyBasePath は履歴ページのパスです。詳細URLの組み立てにも使います。
	historyBasePath = "/history"
	// defaultPerPage は履歴一覧の 1 ページあたりの件数です。
	defaultPerPage = 20
	// maxPerPage は per_page に許容する上限です。1 ページの読み取り件数がそのまま
	// ストレージへの往復数になるため、際限なく増やせないようにします。
	maxPerPage = 100
)

// HistoryPageData は履歴一覧テンプレートに渡すデータです。
type HistoryPageData struct {
	Items []domain.JobStatus
	Meta  domain.PageMeta
	Error string
}

// ReviewDetailPageData は履歴詳細テンプレートに渡すデータです。
type ReviewDetailPageData struct {
	Detail domain.ReviewDetail
	Error  string
}

// HandleHistory は履歴一覧を表示します。
func (h *Handler) HandleHistory(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	page := positiveIntParam(r, "page", 1)
	perPage := clamp(positiveIntParam(r, "per_page", defaultPerPage), 1, maxPerPage)

	result, err := h.history.List(ctx, page, perPage)
	if err != nil {
		slog.ErrorContext(ctx, "レビュー履歴の取得に失敗しました", "error", err)
		h.render(w, r, http.StatusInternalServerError, "history.html", HistoryPageData{
			Error: "履歴を取得できませんでした。時間をおいて再度お試しください。",
		})
		return
	}

	h.render(w, r, http.StatusOK, "history.html", HistoryPageData{
		Items: result.Items,
		Meta:  result.Meta,
	})
}

// HandleReviewDetail は履歴 1 件の詳細を表示します。
func (h *Handler) HandleReviewDetail(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	// ジョブ ID はストレージのパス要素になるため、URL から受け取った時点で検証します。
	rawJobID := chi.URLParam(r, "jobID")
	safeJobID, err := jobid.Sanitize(rawJobID)
	if err != nil {
		slog.WarnContext(ctx, "不正なジョブIDを受け取りました", "job_id", rawJobID, "error", err)
		h.render(w, r, http.StatusBadRequest, "review_detail.html", ReviewDetailPageData{
			Error: "ジョブIDの形式が不正です。",
		})
		return
	}

	detail, err := h.history.Get(ctx, safeJobID)
	if err != nil {
		// 未記録と読み取り失敗はどちらも ErrNotFound で返ります。切り分けられるよう、
		// 包まれた原因をログへ残してから 404 を返します。
		if errors.Is(err, jobstatus.ErrNotFound) {
			slog.WarnContext(ctx, "レビュー履歴が見つかりません", "job_id", safeJobID, "error", err)
			h.render(w, r, http.StatusNotFound, "review_detail.html", ReviewDetailPageData{
				Error: "指定されたレビューは見つかりませんでした。",
			})
			return
		}

		slog.ErrorContext(ctx, "レビュー履歴の取得に失敗しました", "job_id", safeJobID, "error", err)
		h.render(w, r, http.StatusInternalServerError, "review_detail.html", ReviewDetailPageData{
			Error: "レビューを取得できませんでした。",
		})
		return
	}

	h.render(w, r, http.StatusOK, "review_detail.html", ReviewDetailPageData{Detail: detail})
}

// positiveIntParam はクエリから正の整数を読み取ります。
// 未指定・不正値・0 以下はすべて既定値へ落とします。ページ番号の指定ミスで
// 画面がエラーになるより、1 ページ目を出すほうが扱いやすいためです。
func positiveIntParam(r *http.Request, name string, fallback int) int {
	raw := r.URL.Query().Get(name)
	if raw == "" {
		return fallback
	}

	value, err := strconv.Atoi(raw)
	if err != nil || value <= 0 {
		return fallback
	}
	return value
}

func clamp(value, low, high int) int {
	return min(max(value, low), high)
}
