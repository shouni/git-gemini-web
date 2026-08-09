package adapters

import (
	"context"
	"testing"
	"time"

	coreports "github.com/shouni/gemini-reviewer-core/ports"
	"github.com/shouni/gemini-reviewer-core/workflow"

	"github.com/shouni/git-gemini-web/internal/domain"
)

type stubReviewer struct {
	seen func(ctx context.Context)
}

func (s stubReviewer) Run(ctx context.Context, req coreports.ReviewRequest) coreports.ReviewProcessOutcome {
	s.seen(ctx)
	return coreports.ReviewProcessOutcome{Req: req}
}

type stubPublisher struct {
	seen func(ctx context.Context)
}

func (s stubPublisher) Run(ctx context.Context, _ coreports.ReviewProcessOutcome) (coreports.ReviewResult, error) {
	s.seen(ctx)
	return coreports.ReviewResult{Status: coreports.ReviewStatusSuccess}, nil
}

// ★ アダプタがレビューに締切を被せていること。
//
// これが無いと Cloud Tasks の dispatch deadline が先に来て、プロセスごと SIGTERM になり
// 失敗レポートも Slack 通知も残らない（config.DefaultPipelineTimeout のコメント参照）。
func TestCoreWorkflowAdapter_レビューに締切を被せる(t *testing.T) {
	t.Parallel()

	var reviewDeadline time.Time
	var hasDeadline bool
	core := workflow.New(
		stubReviewer{seen: func(ctx context.Context) { reviewDeadline, hasDeadline = ctx.Deadline() }},
		stubPublisher{seen: func(_ context.Context) {}},
	)

	const timeout = 25 * time.Minute
	before := time.Now()
	if err := NewCoreWorkflowAdapter(core, timeout).Execute(context.Background(), domain.ReviewRequest{}); err != nil {
		t.Fatalf("Execute() failed: %v", err)
	}

	if !hasDeadline {
		t.Fatal("レビューの context に締切が無い（PIPELINE_TIMEOUT が効いていない）")
	}
	// 締切がおおよそ timeout 後にあること。
	if d := reviewDeadline.Sub(before); d < timeout-time.Minute || d > timeout+time.Minute {
		t.Errorf("締切までの時間 = %s, want ≒ %s", d, timeout)
	}
}

// 0 以下は無制限。ローカルでの長時間デバッグ用の逃げ道が塞がっていないこと。
func TestCoreWorkflowAdapter_ゼロなら無制限(t *testing.T) {
	t.Parallel()

	var hasDeadline bool
	core := workflow.New(
		stubReviewer{seen: func(ctx context.Context) { _, hasDeadline = ctx.Deadline() }},
		stubPublisher{seen: func(_ context.Context) {}},
	)

	if err := NewCoreWorkflowAdapter(core, 0).Execute(context.Background(), domain.ReviewRequest{}); err != nil {
		t.Fatalf("Execute() failed: %v", err)
	}
	if hasDeadline {
		t.Error("timeout=0 なのに締切が付いている")
	}
}

// 締切で打ち切られても公開は走ること（core v1.11.2 の切り離しが効いているかの結合確認）。
func TestCoreWorkflowAdapter_打ち切り後も公開は走る(t *testing.T) {
	t.Parallel()

	var publishCtxErr error
	published := false
	core := workflow.New(
		stubReviewer{seen: func(ctx context.Context) {
			<-ctx.Done() // 締切まで待つ = レビューが打ち切られた状態を作る
		}},
		stubPublisher{seen: func(ctx context.Context) {
			published = true
			publishCtxErr = ctx.Err()
		}},
	)

	if err := NewCoreWorkflowAdapter(core, 10*time.Millisecond).Execute(context.Background(), domain.ReviewRequest{}); err != nil {
		t.Fatalf("Execute() failed: %v", err)
	}

	if !published {
		t.Fatal("レビュー打ち切り後に公開が走っていない（通知が出ない）")
	}
	if publishCtxErr != nil {
		t.Errorf("公開の context が期限切れ: %v", publishCtxErr)
	}
}
