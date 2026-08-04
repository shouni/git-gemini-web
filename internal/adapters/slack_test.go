package adapters

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/shouni/gemini-reviewer-core/ports"
	"github.com/shouni/go-notify/notify"
)

// recordingNotifier は送信された notify.Message を記録するフェイクです。
// Slack 記法への変換は go-notify 側の責務なので、ここでは git-gemini-web が
// 組み立てた見出しと本文だけを検証します。
type recordingNotifier struct {
	got []notify.Message
}

// Notify は notify.Notifier を実装し、受け取った Message を記録します。
func (r *recordingNotifier) Notify(_ context.Context, msg notify.Message) error {
	r.got = append(r.got, msg)
	return nil
}

// last は最後に送信された Message を返します。
func (r *recordingNotifier) last(t *testing.T) notify.Message {
	t.Helper()
	if len(r.got) == 0 {
		t.Fatal("通知が送信されていません")
	}
	return r.got[len(r.got)-1]
}

// newTestSlackAdapter は記録用 Notifier を差し込んだアダプターを返します。
func newTestSlackAdapter() (*SlackAdapter, *recordingNotifier) {
	rec := &recordingNotifier{}
	return &SlackAdapter{pipeline: notify.NewPipeline(rec, slackTitles)}, rec
}

// testReviewRequest はテストで使う共通のレビュー要求です。
func testReviewRequest() ports.ReviewRequest {
	return ports.ReviewRequest{
		RepoURL:       "git@github.com:org/repo.git",
		BaseBranch:    "main",
		FeatureBranch: "feature/new-ui",
		Mode:          "code",
		ModelName:     "gemini-2.5-pro",
		StorageURI:    "gs://bucket/reviews/repo.html",
		PublicURL:     "https://signed.example.com/repo.html",
	}
}

func TestNotifySuccessIncludesModelName(t *testing.T) {
	adapter, rec := newTestSlackAdapter()

	err := adapter.Notify(context.Background(), ports.ReviewProcessOutcome{Req: testReviewRequest()})
	if err != nil {
		t.Fatalf("Notify() error = %v", err)
	}

	msg := rec.last(t)
	if msg.Title != slackTitles.Success {
		t.Errorf("Title = %q, want %q", msg.Title, slackTitles.Success)
	}

	for _, want := range []string{
		"**詳細URL:** [gs://bucket/reviews/repo.html](https://signed.example.com/repo.html)",
		"**リポジトリ:** `org/repo`",
		"**ブランチ:** `main` ← `feature/new-ui`",
		"**モード:** `code`",
		"**モデル:** `gemini-2.5-pro`",
	} {
		if !strings.Contains(msg.Body, want) {
			t.Fatalf("slack content should contain %q, got:\n%s", want, msg.Body)
		}
	}
}

// TestNotifyFailureIncludesStepAndCause は、失敗通知が発生ステップと原因を含み、
// 詳細URLを含まないことを検証します。失敗時は Publish が実行されないため、
// 詳細URLを出すとリンク先が存在しません。
func TestNotifyFailureIncludesStepAndCause(t *testing.T) {
	adapter, rec := newTestSlackAdapter()

	outcome := ports.ReviewProcessOutcome{
		Req:      testReviewRequest(),
		StepName: "AI レビュー生成",
		Error:    errors.New("Gemini がタイムアウトしました"),
	}
	if err := adapter.Notify(context.Background(), outcome); err != nil {
		t.Fatalf("Notify() error = %v", err)
	}

	msg := rec.last(t)
	if msg.Title != slackTitles.Failure {
		t.Errorf("Title = %q, want %q", msg.Title, slackTitles.Failure)
	}
	if !strings.Contains(msg.Body, "**発生ステップ:** AI レビュー生成") {
		t.Errorf("Body = %q, want the failing step", msg.Body)
	}
	if !strings.Contains(msg.Body, "**エラー内容:**\nGemini がタイムアウトしました") {
		t.Errorf("Body = %q, want the cause", msg.Body)
	}
	if strings.Contains(msg.Body, "詳細URL") {
		t.Errorf("Body = %q, want no result link on failure", msg.Body)
	}
}

// TestNotifySkippedOmitsReasonSection は、スキップ通知がリポジトリとブランチだけを
// 載せることを検証します。スキップの理由は見出しが述べているため、
// 「理由: N/A」という中身のない行は出しません。
func TestNotifySkippedOmitsReasonSection(t *testing.T) {
	adapter, rec := newTestSlackAdapter()

	outcome := ports.ReviewProcessOutcome{Req: testReviewRequest(), IsSkipped: true}
	if err := adapter.Notify(context.Background(), outcome); err != nil {
		t.Fatalf("Notify() error = %v", err)
	}

	msg := rec.last(t)
	if msg.Title != slackTitles.Skipped {
		t.Errorf("Title = %q, want %q", msg.Title, slackTitles.Skipped)
	}
	if strings.Contains(msg.Body, "理由") {
		t.Errorf("Body = %q, 理由の節は出さない想定です", msg.Body)
	}
	if !strings.Contains(msg.Body, "**リポジトリ:** `org/repo`") {
		t.Errorf("Body = %q, want the repository", msg.Body)
	}
	if strings.Contains(msg.Body, "詳細URL") {
		t.Errorf("Body = %q, want no result link when skipped", msg.Body)
	}
}

// TestNotifyErrorTakesPrecedenceOverSkipped は、両方立っている場合に失敗として
// 扱われることを検証します。
func TestNotifyErrorTakesPrecedenceOverSkipped(t *testing.T) {
	adapter, rec := newTestSlackAdapter()

	outcome := ports.ReviewProcessOutcome{
		Req:       testReviewRequest(),
		IsSkipped: true,
		Error:     errors.New("boom"),
	}
	if err := adapter.Notify(context.Background(), outcome); err != nil {
		t.Fatalf("Notify() error = %v", err)
	}

	if msg := rec.last(t); msg.Title != slackTitles.Failure {
		t.Errorf("Title = %q, want %q", msg.Title, slackTitles.Failure)
	}
}

// TestWriteRepositoryFieldsOmitsEmptyBranches は、ブランチ情報が無い場合に
// 行ごと省かれることを検証します。
func TestWriteRepositoryFieldsOmitsEmptyBranches(t *testing.T) {
	body := notify.NewBody()
	writeRepositoryFields(body, ports.ReviewRequest{RepoURL: "git@github.com:org/repo.git"})

	got := body.String()
	if strings.Contains(got, "ブランチ") {
		t.Errorf("body = %q, ブランチの行は出さない想定です", got)
	}
	if !strings.Contains(got, "**リポジトリ:** `org/repo`") {
		t.Errorf("body = %q, want the repository", got)
	}
}

// TestNewSlackAdapterDisabledWhenWebhookURLEmpty は、Webhook URL が未設定なら
// エラーにならず通知が無効化されることを検証します。
func TestNewSlackAdapterDisabledWhenWebhookURLEmpty(t *testing.T) {
	adapter, err := NewSlackAdapter(nil, "")
	if err != nil {
		t.Fatalf("NewSlackAdapter() error = %v", err)
	}
	if adapter.pipeline.Enabled() {
		t.Fatal("Webhook URL 未設定なのに通知が有効になっています")
	}

	if err := adapter.Notify(context.Background(), ports.ReviewProcessOutcome{Req: testReviewRequest()}); err != nil {
		t.Errorf("Notify() = %v, want nil", err)
	}
}

// TestNewSlackAdapterRequiresHTTPClientWhenWebhookSet は、Webhook URL があるのに
// HTTP クライアントが nil の場合はエラーになることを検証します。
func TestNewSlackAdapterRequiresHTTPClientWhenWebhookSet(t *testing.T) {
	if _, err := NewSlackAdapter(nil, "https://hooks.slack.example/webhook"); err == nil {
		t.Fatal("HTTPクライアントが nil なのにエラーになりません")
	}
}

// TestNotifySetsLevel は、3 つの結果それぞれが種別を伴って送信されることを検証します。
// Slack 側はこれを attachment の色に落とすため、見出しの絵文字とは別に必要です。
func TestNotifySetsLevel(t *testing.T) {
	tests := []struct {
		name    string
		outcome ports.ReviewProcessOutcome
		want    notify.Level
	}{
		{
			name:    "成功",
			outcome: ports.ReviewProcessOutcome{Req: testReviewRequest()},
			want:    notify.LevelSuccess,
		},
		{
			name:    "失敗",
			outcome: ports.ReviewProcessOutcome{Req: testReviewRequest(), Error: errors.New("boom")},
			want:    notify.LevelFailure,
		},
		{
			name:    "スキップ",
			outcome: ports.ReviewProcessOutcome{Req: testReviewRequest(), IsSkipped: true},
			want:    notify.LevelSkipped,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			adapter, rec := newTestSlackAdapter()

			if err := adapter.Notify(context.Background(), tt.outcome); err != nil {
				t.Fatalf("Notify failed: %v", err)
			}
			if got := rec.last(t).Level; got != tt.want {
				t.Errorf("Level = %v, want %v", got, tt.want)
			}
		})
	}
}
