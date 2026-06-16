package adapters

import (
	"strings"
	"testing"

	"github.com/shouni/gemini-reviewer-core/ports"
)

func TestBuildSlackContentIncludesModelName(t *testing.T) {
	adapter := &SlackAdapter{}
	req := ports.ReviewRequest{
		RepoURL:       "git@github.com:org/repo.git",
		BaseBranch:    "main",
		FeatureBranch: "feature/new-ui",
		Mode:          "detail",
		ModelName:     "gemini-2.5-pro",
		StorageURI:    "gs://bucket/reviews/repo.html",
		PublicURL:     "https://signed.example.com/repo.html",
	}

	content := adapter.buildSlackContent(req)

	for _, want := range []string{
		"*詳細URL:* <https://signed.example.com/repo.html|gs://bucket/reviews/repo.html>",
		"*リポジトリ:* `org/repo`",
		"*ブランチ:* `main` ← `feature/new-ui`",
		"*モード:* `detail`",
		"*モデル:* `gemini-2.5-pro`",
	} {
		if !strings.Contains(content, want) {
			t.Fatalf("slack content should contain %q, got:\n%s", want, content)
		}
	}
}
