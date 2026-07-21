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
		"code":    "詳細なコード品質レビュー",
		"novel":   "小説原稿の詳細レビュー",
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

	code := prompts["code"]
	if strings.Contains(code, "mode-description:") {
		t.Fatalf("metadata should be stripped from prompt body: %q", code[:80])
	}
	if !strings.HasPrefix(code, "# ") {
		t.Fatalf("prompt body should start with markdown heading: %q", code[:80])
	}
}

func TestLoadPromptsReturnsCopy(t *testing.T) {
	resetPromptCache(t)

	prompts, err := LoadPrompts()
	if err != nil {
		t.Fatalf("LoadPrompts failed: %v", err)
	}
	prompts["code"] = "mutated"

	reloaded, err := LoadPrompts()
	if err != nil {
		t.Fatalf("LoadPrompts reload failed: %v", err)
	}
	if reloaded["code"] == "mutated" {
		t.Fatal("LoadPrompts should not expose the cached map to callers")
	}
}

func TestLoadFindingsFormat(t *testing.T) {
	got, err := LoadFindingsFormat()
	if err != nil {
		t.Fatalf("LoadFindingsFormat failed: %v", err)
	}
	for _, want := range []string{"severity", "file", "excerpt", "message", "suggestion"} {
		if !strings.Contains(got, want) {
			t.Errorf("LoadFindingsFormat() missing %q:\n%s", want, got)
		}
	}
}

func TestLoadVerdictFormat(t *testing.T) {
	got, err := LoadVerdictFormat()
	if err != nil {
		t.Fatalf("LoadVerdictFormat failed: %v", err)
	}
	for _, want := range []string{"decision", "reason"} {
		if !strings.Contains(got, want) {
			t.Errorf("LoadVerdictFormat() missing %q:\n%s", want, got)
		}
	}
}

func TestParsePromptMetadataTrimsLeadingNoiseWithoutMetadata(t *testing.T) {
	description, body := parsePromptMetadata("custom", "\ufeff\n\n# Custom Prompt")

	if description != "custom" {
		t.Fatalf("unexpected description: got %q want %q", description, "custom")
	}
	if body != "# Custom Prompt" {
		t.Fatalf("unexpected body: got %q", body)
	}
}
