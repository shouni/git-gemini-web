package handlers

import (
	"strings"
	"testing"

	"git-gemini-web/internal/domain"
)

func TestValidateBranchName(t *testing.T) {
	tests := []struct {
		name    string
		branch  string
		wantErr bool
	}{
		{name: "simple valid", branch: "main", wantErr: false},
		{name: "path valid", branch: "feature/new-ui", wantErr: false},
		{name: "contains space", branch: "feature/new ui", wantErr: true},
		{name: "contains double dot", branch: "feature..bad", wantErr: true},
		{name: "contains double slash", branch: "feature//bad", wantErr: true},
		{name: "ends with slash", branch: "feature/", wantErr: true},
		{name: "ends with dot", branch: "feature.", wantErr: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateBranchName(tt.branch)
			if tt.wantErr && err == nil {
				t.Fatalf("expected error for %q", tt.branch)
			}
			if !tt.wantErr && err != nil {
				t.Fatalf("unexpected error for %q: %v", tt.branch, err)
			}
		})
	}
}

func TestValidateReviewRequest(t *testing.T) {
	h := &Handler{}
	valid := domain.ReviewRequest{
		RepoURL:       "https://github.com/org/repo.git",
		BaseBranch:    "main",
		FeatureBranch: "feature/new-ui",
		Mode:          "detail",
	}

	tests := []struct {
		name    string
		mutate  func(*domain.ReviewRequest)
		wantErr string
	}{
		{name: "valid", mutate: func(r *domain.ReviewRequest) {}},
		{
			name:    "missing required",
			mutate:  func(r *domain.ReviewRequest) { r.RepoURL = "" },
			wantErr: "すべてのフィールド",
		},
		{
			name:    "invalid mode",
			mutate:  func(r *domain.ReviewRequest) { r.Mode = "unknown" },
			wantErr: "不正なレビューモード",
		},
		{
			name:    "invalid repo url",
			mutate:  func(r *domain.ReviewRequest) { r.RepoURL = "https://github.com/org/repo" },
			wantErr: "URLの形式",
		},
		{
			name:    "invalid base branch",
			mutate:  func(r *domain.ReviewRequest) { r.BaseBranch = "main..bad" },
			wantErr: "ベースブランチ名",
		},
		{
			name:    "invalid feature branch",
			mutate:  func(r *domain.ReviewRequest) { r.FeatureBranch = "feature/" },
			wantErr: "フィーチャーブランチ名",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := valid
			tt.mutate(&req)

			err := h.validateReviewRequest(req)
			if tt.wantErr == "" && err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if tt.wantErr != "" {
				if err == nil {
					t.Fatalf("expected error containing %q, got nil", tt.wantErr)
				}
				if !strings.Contains(err.Error(), tt.wantErr) {
					t.Fatalf("error %q does not contain %q", err.Error(), tt.wantErr)
				}
			}
		})
	}
}
