package adapters

import (
	"context"
	"errors"
	"testing"

	"github.com/shouni/go-review-kit/review"

	"github.com/shouni/git-gemini-web/internal/domain"
)

// 成功時は、判定とタイトルと結果の場所を記録します。これが履歴一覧の 1 行になります。
func TestStatusRecorderRecordsSuccess(t *testing.T) {
	store := &stubStore{}
	recorder := NewStatusRecorder(store)

	report := testReport()
	if err := recorder.Notify(context.Background(), notification(review.StatusSucceeded, &report, nil)); err != nil {
		t.Fatalf("Notify() error = %v", err)
	}

	if len(store.saved) != 1 {
		t.Fatalf("記録件数 = %d, want 1", len(store.saved))
	}

	got := store.saved[0]
	if got.State != "succeeded" {
		t.Errorf("State = %q, want succeeded", got.State)
	}
	if got.Outcome != review.StatusSucceeded {
		t.Errorf("Outcome = %q, want %q", got.Outcome, review.StatusSucceeded)
	}
	if got.Title != report.Title {
		t.Errorf("Title = %q, want %q", got.Title, report.Title)
	}
	if got.Decision != review.DecisionMinor {
		t.Errorf("Decision = %q, want %q", got.Decision, review.DecisionMinor)
	}
	if !got.HasReport() {
		t.Error("結果の場所が記録されていません")
	}
}

// スキップは succeeded として記録し、Outcome で区別します。成果物は無いので
// 結果の場所は記録しません。
func TestStatusRecorderRecordsSkipped(t *testing.T) {
	store := &stubStore{}
	recorder := NewStatusRecorder(store)

	if err := recorder.Notify(context.Background(), notification(review.StatusSkipped, nil, nil)); err != nil {
		t.Fatalf("Notify() error = %v", err)
	}

	got := store.saved[0]
	if got.State != "succeeded" {
		t.Errorf("State = %q, want succeeded", got.State)
	}
	if got.Outcome != review.StatusSkipped {
		t.Errorf("Outcome = %q, want %q", got.Outcome, review.StatusSkipped)
	}
	if got.HasReport() {
		t.Error("スキップなのに結果の場所が記録されています")
	}
}

// 失敗こそ記録が要ります。旧実装では失敗が Slack にしか残りませんでした。
func TestStatusRecorderRecordsFailure(t *testing.T) {
	store := &stubStore{}
	recorder := NewStatusRecorder(store)

	cause := review.WrapStep(review.StepReview, errors.New("timeout"))
	if err := recorder.Notify(context.Background(), notification(review.StatusFailed, nil, cause)); err != nil {
		t.Fatalf("Notify() error = %v", err)
	}

	got := store.saved[0]
	if got.State != "failed" {
		t.Errorf("State = %q, want failed", got.State)
	}
	if got.Error == "" {
		t.Error("失敗理由が記録されていません")
	}
}

// ジョブIDが無ければ書き込み先が決まりません。落とさずに記録だけ諦めます。
func TestStatusRecorderSkipsWithoutJobID(t *testing.T) {
	store := &stubStore{}
	recorder := NewStatusRecorder(store)

	n := notification(review.StatusSucceeded, nil, nil)
	n.Request.JobID = ""

	if err := recorder.Notify(context.Background(), n); err != nil {
		t.Fatalf("Notify() error = %v", err)
	}
	if len(store.saved) != 0 {
		t.Fatalf("記録件数 = %d, want 0", len(store.saved))
	}
}

// 複数の通知先へ配り、片方が失敗しても残りは呼ばれること。
func TestMultiNotifierCallsAll(t *testing.T) {
	failing := &countingNotifier{err: errors.New("slack down")}
	succeeding := &countingNotifier{}

	multi := MultiNotifier{failing, nil, succeeding}

	err := multi.Notify(context.Background(), notification(review.StatusSucceeded, nil, nil))
	if err == nil {
		t.Fatal("失敗したことが呼び出し側へ伝わっていません")
	}
	if !errors.Is(err, failing.err) {
		t.Errorf("原因まで辿れません: %v", err)
	}
	if succeeding.calls != 1 {
		t.Errorf("後続の通知回数 = %d, want 1", succeeding.calls)
	}
}

type countingNotifier struct {
	err   error
	calls int
}

func (c *countingNotifier) Notify(context.Context, review.Notification) error {
	c.calls++
	return c.err
}

// 記録が Notifier として動くことを型で確かめます。
var _ review.Notifier = (*StatusRecorder)(nil)
var _ domain.StatusStore = (*stubStore)(nil)
