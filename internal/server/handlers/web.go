package handlers

import (
	"fmt"
	"git-gemini-web/internal/config"
	"git-gemini-web/internal/domain"
	"html/template"
	"log/slog"
	"net/http"
	"regexp"
	"strings"

	"github.com/google/uuid"
	"github.com/shouni/gcp-kit/tasks"
	"github.com/shouni/go-remote-io/pkg/remoteio"
	"github.com/shouni/go-utils/urlpath"
)

var (
	// gitURLRegexp は、http、https、git、ssh プロトコルを含む有効な Git リポジトリ URL と一致します。
	gitURLRegexp = regexp.MustCompile(`^((https?|git|ssh):\/\/|git@)[^ \t\n\r\f\v;\|&]+\.git$`)
	// gitBranchRegexp は、有効な Git ブランチ名と一致します。
	gitBranchRegexp = regexp.MustCompile(`^[\w.-]+(/[\w.-]+)*$`)
)

// ReviewFormPageData はフォームテンプレートに渡すデータ構造です。
type ReviewFormPageData struct {
	Message   string // 成功メッセージ
	Error     string // エラーメッセージ
	ResultURL string // GCSのレビュー結果URL (署名付きURL)
}

// Handler は HTTPリクエストを処理する構造体です。
type Handler struct {
	cfg          config.Config
	taskEnqueuer *tasks.Enqueuer[domain.ReviewRequest]
	ioFactory    remoteio.IOFactory
	template     *template.Template
}

// NewHandler は新しい Handler インスタンスを作成し、依存関係を注入します。
func NewHandler(
	cfg config.Config,
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

// HandleReviewForm は、GET / リクエストを処理し、レビューフォームを表示します。
func (h *Handler) HandleReviewForm(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	if err := h.template.Execute(w, nil); err != nil {
		slog.Error("テンプレート実行エラー", "error", err)
		http.Error(w, "テンプレートの実行に失敗しました。", http.StatusInternalServerError)
	}
}

// validateBranchName は、指定されたブランチ名が Git 命名規則に従って有効かどうかを確認し、無効な場合はエラーを返します。
func validateBranchName(branchName string) error {
	if !gitBranchRegexp.MatchString(branchName) {
		return fmt.Errorf("形式が不正です。許容されない特殊文字が含まれています。")
	}
	if strings.Contains(branchName, "..") || strings.Contains(branchName, "//") {
		return fmt.Errorf("形式が不正です。'..' または '//' は使用できません。")
	}
	if strings.HasSuffix(branchName, "/") || strings.HasSuffix(branchName, ".") {
		return fmt.Errorf("形式が不正です。末尾を'/'や'.'にすることはできません。")
	}
	return nil
}

// HandleReviewSubmit は、POST /submit_review リクエストを処理します。
func (h *Handler) HandleReviewSubmit(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	ctx := r.Context()

	if err := r.ParseForm(); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		h.template.Execute(w, ReviewFormPageData{Error: "リクエストのパースに失敗しました。"})
		return
	}

	repoURL := r.FormValue("repo_url")
	baseBranch := r.FormValue("base_branch")
	featureBranch := r.FormValue("feature_branch")
	mode := r.FormValue("review_mode")

	// 1. 入力チェック
	if repoURL == "" || baseBranch == "" || featureBranch == "" || mode == "" {
		w.WriteHeader(http.StatusBadRequest)
		h.template.Execute(w, ReviewFormPageData{Error: "すべてのフィールドを入力してください。"})
		return
	}

	// 2. バリデーション (URLとブランチ)
	if !gitURLRegexp.MatchString(repoURL) {
		slog.WarnContext(ctx, "Invalid repository URL format provided by user", "url", repoURL)
		w.WriteHeader(http.StatusBadRequest)
		h.template.Execute(w, ReviewFormPageData{Error: "リポジトリURLの形式が不正です。"})
		return
	}
	if err := validateBranchName(baseBranch); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		h.template.Execute(w, ReviewFormPageData{Error: "ベースブランチ名が不正です。"})
		return
	}
	if err := validateBranchName(featureBranch); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		h.template.Execute(w, ReviewFormPageData{Error: "フィーチャーブランチ名が不正です。"})
		return
	}

	// 3. 結果のGCSパス決定と署名付きURL生成
	reviewID := uuid.New().String()
	repoID := urlpath.GenerateGCSKeyName(repoURL)
	gcsPath := fmt.Sprintf("reviews/%s/%s/%s-%s.html", repoID, featureBranch, mode, reviewID)

	urlSigner, err := h.ioFactory.URLSigner()
	if err != nil {
		slog.Error("URLSignerの取得失敗", "error", err)
		http.Error(w, "内部エラーが発生しました。", http.StatusInternalServerError)
		return
	}

	publicURL, err := urlSigner.GenerateSignedURL(
		ctx,
		fmt.Sprintf("gs://%s/%s", h.cfg.GCSBucket, gcsPath),
		"GET",
		config.SignedURLExpiration,
	)
	if err != nil {
		slog.Error("署名付きURLの生成失敗", "error", err)
		http.Error(w, "内部エラーが発生しました。", http.StatusInternalServerError)
		return
	}

	// 4. ライブラリの Enqueuer を使ってタスクを投入する
	reviewReq := domain.ReviewRequest{
		RepoURL:       repoURL,
		BaseBranch:    baseBranch,
		FeatureBranch: featureBranch,
		Mode:          mode,
		GCSBucket:     h.cfg.GCSBucket,
		GCSPath:       gcsPath,
	}

	// taskEnqueuer.Enqueue はペイロードのJSONシリアライズ、OIDCトークンの付与、
	// Cloud Tasks APIの呼び出しを抽象化し、タスク投入処理を簡潔にします。
	if err := h.taskEnqueuer.Enqueue(ctx, reviewReq); err != nil {
		slog.Error("Cloud Tasksへの投入失敗", "error", err)
		h.template.Execute(w, ReviewFormPageData{Error: "レビューの受け付けに失敗しました。時間をおいて再度お試しください。"})
		return
	}

	// 5. ユーザーへの応答
	slog.Info("レビュータスク投入成功", "repo", repoURL, "path", gcsPath)
	w.WriteHeader(http.StatusAccepted)
	h.template.Execute(w, ReviewFormPageData{
		Message:   "✅ レビュータスクを受け付けました。生成完了後、以下のURLから確認できます（15分間有効）。",
		ResultURL: publicURL,
	})
}
