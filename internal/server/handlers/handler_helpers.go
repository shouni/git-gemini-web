package handlers

import (
	"bytes"
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"regexp"
	"strings"

	"git-gemini-web/internal/config"
	"git-gemini-web/internal/domain"
)

var (
	// gitURLRegexp は、GitリポジトリURLの形式をチェックします。
	gitURLRegexp = regexp.MustCompile(`^((https?|git|ssh):\/\/|git@)[^ \t\n\r\f\v;\|&]+\.git$`)
	// gitBranchRegexp は、ブランチ名の命名規則をチェックします。
	gitBranchRegexp = regexp.MustCompile(`^[\w.-]+(/[\w.-]+)*$`)
)

// renderForm はテンプレートの表示を一括管理するヘルパーメソッドです。
func (h *Handler) renderForm(w http.ResponseWriter, status int, data ReviewFormPageData) {
	var buf bytes.Buffer
	if err := h.template.Execute(&buf, data); err != nil {
		slog.Error("テンプレート実行エラー", "error", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.WriteHeader(status)
	if _, err := buf.WriteTo(w); err != nil {
		slog.Error("レスポンス書き込みエラー", "error", err)
	}
}

// validateReviewRequest は入力内容が正しいかまとめてチェックする。
func (h *Handler) validateReviewRequest(req domain.ReviewRequest) error {
	if req.RepoURL == "" || req.BaseBranch == "" || req.FeatureBranch == "" || req.Mode == "" {
		return fmt.Errorf("すべてのフィールドを入力してください。")
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
		return fmt.Errorf("形式が不正です。")
	}
	if strings.Contains(branchName, "..") || strings.Contains(branchName, "//") {
		return fmt.Errorf("'..' または '//' は使用できません。")
	}
	if strings.HasSuffix(branchName, "/") || strings.HasSuffix(branchName, ".") {
		return fmt.Errorf("末尾に '/' や '.' は使用できません。")
	}
	return nil
}

// generateSignedResultURL は GCS のパスから署名付きURLを作るヘルパーです。
func (h *Handler) generateSignedResultURL(ctx context.Context, gcsPath string) (string, error) {
	urlSigner, err := h.ioFactory.URLSigner()
	if err != nil {
		return "", err
	}

	fullGSPath := fmt.Sprintf("gs://%s/%s", h.cfg.GCSBucket, gcsPath)
	return urlSigner.GenerateSignedURL(ctx, fullGSPath, "GET", config.SignedURLExpiration)
}
