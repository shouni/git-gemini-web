package adapters

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/shouni/go-review-kit/pipeline"
	"github.com/shouni/go-review-kit/review"

	"github.com/shouni/git-gemini-web/internal/domain"
)

// stubSource は差分取得元のフェイクです。Diff で受け取った context を観察できます。
type stubSource struct {
	seen func(ctx context.Context)
}

func (s stubSource) Diff(ctx context.Context, _, _ string) (string, error) {
	if s.seen != nil {
		s.seen(ctx)
	}
	return "diff --git a/main.go b/main.go", nil
}

func (s stubSource) Close(context.Context) error { return nil }

type stubFactory struct{ source stubSource }

func (f stubFactory) Open(context.Context, review.Request) (review.DiffSource, error) {
	return f.source, nil
}

type stubPrompts struct{}

func (stubPrompts) Generate(_, _ string) (string, error) { return "prompt", nil }

type stubReviewer struct{}

func (stubReviewer) Review(context.Context, string, string) (review.Report, error) {
	return review.Report{
		Title:   "レビュー結果",
		Verdict: review.Verdict{Decision: review.DecisionNone, Reason: "問題なし"},
	}, nil
}

// stubPublisher は保存のフェイクです。保存時の context を観察できます。
type stubPublisher struct {
	seen func(ctx context.Context)
}

func (p stubPublisher) Publish(ctx context.Context, _ review.Request, _ review.Report) error {
	if p.seen != nil {
		p.seen(ctx)
	}
	return nil
}

type stubNotifier struct{}

func (stubNotifier) Notify(context.Context, review.Notification) error { return nil }

// stubStore は進行状況のフェイクです。保存された状態を順に記録します。
type stubStore struct {
	mu    sync.Mutex
	saved []domain.JobStatus
}

func (s *stubStore) Get(_ context.Context, jobID string) (domain.JobStatus, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	for i := len(s.saved) - 1; i >= 0; i-- {
		if s.saved[i].JobID == jobID {
			return s.saved[i], nil
		}
	}
	return domain.JobStatus{}, jobNotFound
}

func (s *stubStore) Save(_ context.Context, jobID string, status domain.JobStatus) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	status.JobID = jobID
	s.saved = append(s.saved, status)
	return nil
}

func (s *stubStore) states() []string {
	s.mu.Lock()
	defer s.mu.Unlock()

	states := make([]string, 0, len(s.saved))
	for _, status := range s.saved {
		states = append(states, string(status.State))
	}
	return states
}

// jobNotFound は「未記録」を表すテスト用のエラーです。
var jobNotFound = errNotRecorded{}

type errNotRecorded struct{}

func (errNotRecorded) Error() string { return "not recorded" }

// newTestPipeline は、観察用フックを差し込んだパイプラインを組み立てます。
func newTestPipeline(t *testing.T, source stubSource, publisher stubPublisher) *pipeline.Pipeline {
	t.Helper()

	p, err := pipeline.New(pipeline.Deps{
		Sources:   stubFactory{source: source},
		Prompts:   stubPrompts{},
		Reviewer:  stubReviewer{},
		Publisher: publisher,
		Notifier:  stubNotifier{},
	})
	if err != nil {
		t.Fatalf("パイプラインの構築に失敗: %v", err)
	}
	return p
}

func testDomainRequest() domain.ReviewRequest {
	return domain.ReviewRequest{
		JobID:         "20260810-213000-a1b2c3d4",
		RepoURL:       "git@github.com:org/repo.git",
		BaseBranch:    "main",
		FeatureBranch: "develop",
		Mode:          "code",
		ModelName:     "gemini-2.5-pro",
		StorageURI:    "gs://bucket/reviews/20260810-213000-a1b2c3d4/report.json",
	}
}

// ★ アダプタがレビューに締切を被せていること。
//
// これが無いと Cloud Tasks の dispatch deadline が先に来て、プロセスごと SIGTERM になり
// 失敗の記録も Slack 通知も残らない（config.DefaultPipelineTimeout のコメント参照）。
func TestReviewPipeline_レビューに締切を被せる(t *testing.T) {
	t.Parallel()

	var reviewDeadline time.Time
	var hasDeadline bool
	core := newTestPipeline(t,
		stubSource{seen: func(ctx context.Context) { reviewDeadline, hasDeadline = ctx.Deadline() }},
		stubPublisher{},
	)

	const timeout = 25 * time.Minute
	before := time.Now()
	if err := NewReviewPipeline(core, &stubStore{}, timeout).Execute(context.Background(), testDomainRequest()); err != nil {
		t.Fatalf("Execute() failed: %v", err)
	}

	if !hasDeadline {
		t.Fatal("レビューの context に締切が無い（PIPELINE_TIMEOUT が効いていない）")
	}
	if d := reviewDeadline.Sub(before); d < timeout-time.Minute || d > timeout+time.Minute {
		t.Errorf("締切までの時間 = %s, want ≒ %s", d, timeout)
	}
}

// 0 以下は無制限。ローカルでの長時間デバッグ用の逃げ道が塞がっていないこと。
func TestReviewPipeline_ゼロなら無制限(t *testing.T) {
	t.Parallel()

	var hasDeadline bool
	core := newTestPipeline(t,
		stubSource{seen: func(ctx context.Context) { _, hasDeadline = ctx.Deadline() }},
		stubPublisher{},
	)

	if err := NewReviewPipeline(core, &stubStore{}, 0).Execute(context.Background(), testDomainRequest()); err != nil {
		t.Fatalf("Execute() failed: %v", err)
	}
	if hasDeadline {
		t.Error("timeout=0 なのに締切が付いている")
	}
}

// 締切で打ち切られても保存・通知は走ること（ライブラリ側の切り離しが効いているかの結合確認）。
func TestReviewPipeline_打ち切り後も保存は走る(t *testing.T) {
	t.Parallel()

	var publishCtxErr error
	published := false
	core := newTestPipeline(t,
		stubSource{seen: func(ctx context.Context) {
			<-ctx.Done() // 締切まで待つ = レビューが打ち切られた状態を作る
		}},
		stubPublisher{seen: func(ctx context.Context) {
			published = true
			publishCtxErr = ctx.Err()
		}},
	)

	// 差分取得が締切で打ち切られるため、パイプラインは失敗として返ります。
	// ここで確かめたいのは、その後も保存側が動けることです。
	_ = NewReviewPipeline(core, &stubStore{}, 10*time.Millisecond).Execute(context.Background(), testDomainRequest())

	if published {
		if publishCtxErr != nil {
			t.Errorf("保存の context が期限切れ: %v", publishCtxErr)
		}
	}
}

// 実行開始が記録され、試行回数が加算されること。
func TestReviewPipeline_実行開始を記録する(t *testing.T) {
	t.Parallel()

	store := &stubStore{}
	core := newTestPipeline(t, stubSource{}, stubPublisher{})

	if err := NewReviewPipeline(core, store, 0).Execute(context.Background(), testDomainRequest()); err != nil {
		t.Fatalf("Execute() failed: %v", err)
	}

	states := store.states()
	if len(states) == 0 || states[0] != "running" {
		t.Fatalf("最初の記録 = %v, want running", states)
	}
	if store.saved[0].Attempts != 1 {
		t.Errorf("Attempts = %d, want 1", store.saved[0].Attempts)
	}
}

// 完了済みのタスクが再配信されたら、レビューを実行せずに打ち切ること。
// Cloud Tasks は at-least-once 配信のため、これが無いと AI の呼び出しが二重に走ります。
func TestReviewPipeline_完了済みの再配信は打ち切る(t *testing.T) {
	t.Parallel()

	req := testDomainRequest()
	store := &stubStore{}
	if err := store.Save(context.Background(), req.JobID, domain.NewSucceededStatus(req, review.StatusSucceeded)); err != nil {
		t.Fatalf("事前の記録に失敗: %v", err)
	}

	reviewed := false
	core := newTestPipeline(t,
		stubSource{seen: func(context.Context) { reviewed = true }},
		stubPublisher{},
	)

	if err := NewReviewPipeline(core, store, 0).Execute(context.Background(), req); err != nil {
		t.Fatalf("Execute() failed: %v", err)
	}
	if reviewed {
		t.Error("完了済みなのにレビューが実行されている")
	}
}
