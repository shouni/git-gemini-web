package assets

import (
	"strings"
	"testing"
)

func resetPromptCache(t *testing.T) {
	t.Helper()

	mu.Lock()
	cachedPrompts = nil
	mu.Unlock()

	t.Cleanup(func() {
		mu.Lock()
		cachedPrompts = nil
		mu.Unlock()
	})
}

func TestAvailableModesReadsPromptMetadata(t *testing.T) {
	resetPromptCache(t)

	modes, err := AvailableModes()
	if err != nil {
		t.Fatalf("AvailableModes failed: %v", err)
	}

	got := make(map[string]string, len(modes))
	for _, mode := range modes {
		got[mode.Name] = mode.Description
	}

	want := map[string]string{
		"article": "技術記事・ドキュメント品質レビュー",
		"detail":  "詳細な品質レビュー",
		"release": "リリース可否判定",
	}
	for mode, description := range want {
		if got[mode] != description {
			t.Fatalf("unexpected description for %s: got %q want %q", mode, got[mode], description)
		}
	}
}

func TestLoadPromptsStripsModeDescriptionMetadata(t *testing.T) {
	resetPromptCache(t)

	prompts, err := LoadPrompts()
	if err != nil {
		t.Fatalf("LoadPrompts failed: %v", err)
	}

	detail := prompts["detail"]
	if strings.Contains(detail, "mode-description:") {
		t.Fatalf("metadata should be stripped from prompt body: %q", detail[:80])
	}
	if !strings.HasPrefix(detail, "# ") {
		t.Fatalf("prompt body should start with markdown heading: %q", detail[:80])
	}
}

func TestLoadPromptsReturnsCopy(t *testing.T) {
	resetPromptCache(t)

	prompts, err := LoadPrompts()
	if err != nil {
		t.Fatalf("LoadPrompts failed: %v", err)
	}
	prompts["detail"] = "mutated"

	reloaded, err := LoadPrompts()
	if err != nil {
		t.Fatalf("LoadPrompts reload failed: %v", err)
	}
	if reloaded["detail"] == "mutated" {
		t.Fatal("LoadPrompts should not expose the cached map to callers")
	}
}
