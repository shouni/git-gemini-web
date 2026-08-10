package adapters

import (
	"context"
	"log/slog"
	"time"

	"github.com/shouni/go-job-kit/jobstatus"
	"github.com/shouni/go-review-kit/pipeline"

	"github.com/shouni/git-gemini-web/internal/domain"
)

// ReviewPipeline は go-review-kit のパイプラインを domain.Pipeline として公開する ACL です。
// あわせて、ワーカー側でしか分からない進行状況（実行開始・再配信）を記録します。
type ReviewPipeline struct {
	core     *pipeline.Pipeline
	recorder *jobstatus.Recorder[domain.JobStatus]
	timeout  time.Duration
}

var _ domain.Pipeline = (*ReviewPipeline)(nil)

// NewReviewPipeline は ReviewPipeline を構築します。
//
// timeout はレビュー 1 件の実行時間の上限（PIPELINE_TIMEOUT）です。0 以下なら無制限。
func NewReviewPipeline(core *pipeline.Pipeline, store domain.StatusStore, timeout time.Duration) *ReviewPipeline {
	return &ReviewPipeline{
		core:     core,
		recorder: jobstatus.NewRecorder(store),
		timeout:  timeout,
	}
}

// Execute は domain モデルをライブラリのモデルへ変換して実行します。
//
// ★ ここで締切を被せることで、**Cloud Tasks より先にアプリが諦めます**。
//
// Cloud Tasks に先を越されるとプロセスごと SIGTERM になり、失敗の記録も Slack 通知も
// 残りません（review-queue は max_attempts = 1 なので再試行も来ず、タスクは黙って
// 失われます）。自分で諦めれば、打ち切りの理由が結末に載って記録と通知まで届きます。
//
// 締切が保存・通知・後始末を巻き込まないことはライブラリ側が保証しています。
func (p *ReviewPipeline) Execute(ctx context.Context, req domain.ReviewRequest) error {
	if p.skipRedelivery(ctx, req.JobID) {
		return nil
	}
	p.recordRunning(ctx, req)

	if p.timeout > 0 {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, p.timeout)
		defer cancel()
	}

	// 結末（成功・スキップ・失敗）の記録は StatusRecorder が Notifier として行うため、
	// ここでは結果を捨ててエラーだけをワーカーへ返します。
	_, err := p.core.Run(ctx, toReviewRequest(req))
	return err
}

// skipRedelivery は、既に成功しているジョブの再配信を打ち切ってよいかを返します。
//
// Cloud Tasks は at-least-once 配信です。通知の失敗などでワーカーがエラーを返すと同じ
// タスクが再配信され、AI の呼び出しコストがそのまま二重に発生します。
func (p *ReviewPipeline) skipRedelivery(ctx context.Context, jobID string) bool {
	if jobID == "" {
		return false
	}

	done, err := p.recorder.AlreadySucceeded(ctx, jobID)
	if err != nil {
		// 読めない場合は未完了として先へ進めます。記録が読めないことを理由に
		// レビューを止めるより、二重実行のほうがまだ回復可能なためです。
		slog.WarnContext(ctx, "完了済みかどうかを確認できませんでした", "job_id", jobID, "error", err)
		return false
	}
	if done {
		slog.InfoContext(ctx, "完了済みのタスクが再配信されたため打ち切ります", "job_id", jobID)
	}
	return done
}

// recordRunning は、処理を開始したことを記録します。
func (p *ReviewPipeline) recordRunning(ctx context.Context, req domain.ReviewRequest) {
	if req.JobID == "" {
		return
	}

	p.recorder.Record(ctx, req.JobID, domain.NewRunningStatus(req), func(next, prev *domain.JobStatus) {
		// Attempts と QueuedAt は Recorder が前回の記録から引き継いだあとに呼ばれます。
		next.Attempts++
		domain.CarryOverExtras(next, prev)
	})
}
