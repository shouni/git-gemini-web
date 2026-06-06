package handlers

import (
	"bytes"
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"regexp"
	"slices"
	"strings"

	"git-gemini-web/assets"
	"git-gemini-web/internal/config"
	"git-gemini-web/internal/domain"
)

const (
	repoURLPattern    = `^((https?|git|ssh)://|git@)[a-zA-Z0-9_./:@-]+\.git$`
	branchPattern     = `^[a-zA-Z0-9_.-]+(/[a-zA-Z0-9_.-]+)*$`
	csrfTokenField    = "csrf_token"
	defaultReviewMode = "detail"
	defaultBaseBranch = "main"
	defaultHeadBranch = "develop"
)

var (
	// gitURLRegexp は、GitリポジトリURLの形式をチェックします。
	gitURLRegexp = regexp.MustCompile(repoURLPattern)
	// gitBranchRegexp は、ブランチ名の命名規則をチェックします。
	gitBranchRegexp = regexp.MustCompile(branchPattern)
)

// renderForm はテンプレートの表示を一括管理するヘルパーメソッドです。
func (h *Handler) renderForm(w http.ResponseWriter, r *http.Request, status int, data ReviewFormPageData) {
	applyFormValueDefaults(r, &data)

	data.RepoURLPattern = repoURLPattern
	data.BranchPattern = branchPattern
	data.CSRFTokenField = csrfTokenField
	if len(data.ReviewModes) == 0 {
		data.ReviewModes = reviewModeOptions(r.Context(), data.ReviewMode)
	}
	if len(data.GeminiModels) == 0 {
		data.GeminiModels = h.geminiModelOptions(data.GeminiModel)
	}
	if data.CSRFToken == "" {
		data.CSRFToken = csrfTokenFromContext(r.Context())
	}

	var buf bytes.Buffer
	if err := h.template.ExecuteTemplate(&buf, "layout.html", data); err != nil {
		slog.ErrorContext(r.Context(), "テンプレート実行エラー", "error", err, "templateName", "layout.html")
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.WriteHeader(status)
	if _, err := buf.WriteTo(w); err != nil {
		slog.ErrorContext(r.Context(), "レスポンス書き込みエラー", "error", err)
	}
}

func applyFormValueDefaults(r *http.Request, data *ReviewFormPageData) {
	if data.RepoURL == "" {
		data.RepoURL = strings.TrimSpace(r.PostFormValue("repo_url"))
	}
	if data.BaseBranch == "" {
		data.BaseBranch = strings.TrimSpace(r.PostFormValue("base_branch"))
	}
	if data.FeatureBranch == "" {
		data.FeatureBranch = strings.TrimSpace(r.PostFormValue("feature_branch"))
	}
	if data.ReviewMode == "" {
		data.ReviewMode = r.PostFormValue("review_mode")
	}
	if data.GeminiModel == "" {
		data.GeminiModel = r.PostFormValue("gemini_model")
	}

	if r.Method == http.MethodGet {
		if data.BaseBranch == "" {
			data.BaseBranch = defaultBaseBranch
		}
		if data.FeatureBranch == "" {
			data.FeatureBranch = defaultHeadBranch
		}
	}
}

func reviewModeOptions(ctx context.Context, selectedMode string) []ReviewModeOption {
	modes, err := assets.AvailableModes()
	if err != nil {
		slog.ErrorContext(ctx, "レビューモード一覧の読み込みに失敗しました", "error", err)
		if selectedMode == "" {
			selectedMode = defaultReviewMode
		}
		return []ReviewModeOption{{
			Value:       selectedMode,
			Description: selectedMode,
			Selected:    true,
		}}
	}
	if selectedMode == "" {
		selectedMode = defaultReviewMode
	}

	options := make([]ReviewModeOption, 0, len(modes))
	hasSelected := false
	for _, mode := range modes {
		selected := mode.Name == selectedMode
		if selected {
			hasSelected = true
		}
		options = append(options, ReviewModeOption{
			Value:       mode.Name,
			Description: mode.Description,
			Selected:    selected,
		})
	}
	if len(options) > 0 && !hasSelected {
		options[0].Selected = true
	}
	return options
}

func (h *Handler) geminiModelOptions(selectedModel string) []GeminiModelOption {
	models := h.configuredGeminiModels()
	if selectedModel == "" || !slices.Contains(models, selectedModel) {
		selectedModel = models[0]
	}

	options := make([]GeminiModelOption, 0, len(models))
	for _, model := range models {
		options = append(options, GeminiModelOption{
			Value:    model,
			Selected: model == selectedModel,
		})
	}
	return options
}

func (h *Handler) configuredGeminiModels() []string {
	if h.cfg == nil {
		return []string{config.DefaultGeminiModel}
	}
	if len(h.cfg.GeminiModels) > 0 {
		return h.cfg.GeminiModels
	}
	if h.cfg.GeminiModel != "" {
		return []string{h.cfg.GeminiModel}
	}
	return []string{config.DefaultGeminiModel}
}

// validateReviewRequest は入力内容が正しいかまとめてチェックする。
func (h *Handler) validateReviewRequest(req domain.ReviewRequest) error {
	if req.RepoURL == "" || req.BaseBranch == "" || req.FeatureBranch == "" || req.Mode == "" || req.ModelName == "" {
		return fmt.Errorf("すべてのフィールドを入力してください。")
	}

	// レビューモードの動的バリデーション
	if !assets.IsValidMode(req.Mode) {
		return fmt.Errorf("不正なレビューモードです: %s", req.Mode)
	}

	if !slices.Contains(h.configuredGeminiModels(), req.ModelName) {
		return fmt.Errorf("不正なGeminiモデルです: %s", req.ModelName)
	}

	if !gitURLRegexp.MatchString(req.RepoURL) {
		return fmt.Errorf("リポジトリURLの形式が不正です。")
	}

	if err := validateBranchName(req.BaseBranch); err != nil {
		return fmt.Errorf("ベースブランチ名: %w", err)
	}

	if err := validateBranchName(req.FeatureBranch); err != nil {
		return fmt.Errorf("フィーチャーブランチ名: %w", err)
	}

	return nil
}

// validateBranchName は Git のブランチ名として正当かどうかを判定する。
func validateBranchName(branchName string) error {
	if !gitBranchRegexp.MatchString(branchName) {
		return fmt.Errorf("形式が不正です。英数字、ハイフン、ドット、スラッシュのみ使用可能です。")
	}
	if strings.Contains(branchName, "..") || strings.Contains(branchName, "//") {
		return fmt.Errorf("'..' または '//' は使用できません。")
	}
	if strings.HasSuffix(branchName, "/") || strings.HasSuffix(branchName, ".") {
		return fmt.Errorf("末尾に '/' や '.' は使用できません。")
	}
	return nil
}
