package adapters

import (
	"context"
	"log/slog"

	"github.com/shouni/go-job-kit/jobstatus"
	"github.com/shouni/go-review-kit/review"

	"github.com/shouni/git-gemini-web/internal/domain"
)

// StatusRecorder は、レビューの結末を status.json へ記録する review.Notifier です。
//
// 記録を Notifier として実装しているのは、パイプラインが結末を一式（リクエスト・結果・
// レポート・エラー）まとめて渡してくるのが Notify だけだからです。しかも成功・スキップ・
// 失敗のいずれでも必ず 1 回呼ばれるため、記録の取りこぼしが起きません。
// Publisher 側では成功時しか呼ばれず、失敗を記録できません。
type StatusRecorder struct {
	recorder *jobstatus.Recorder[domain.JobStatus]
}

var _ review.Notifier = (*StatusRecorder)(nil)

// NewStatusRecorder は StatusRecorder を構築します。store が nil の場合、記録は行われません。
func NewStatusRecorder(store domain.StatusStore) *StatusRecorder {
	return &StatusRecorder{recorder: jobstatus.NewRecorder(store)}
}

// Notify は、結末に応じた JobStatus を記録します。
//
// 記録の失敗で nil 以外を返さないのは、パイプラインが通知の失敗を全体の失敗として扱わない
// 前提に合わせるためです（jobstatus.Recorder が失敗をログに残します）。
func (r *StatusRecorder) Notify(ctx context.Context, n review.Notification) error {
	if n.Request.JobID == "" {
		slog.WarnContext(ctx, "ジョブIDが無いため進行状況を記録できません", "status", n.Result.Status)
		return nil
	}

	r.recorder.Record(ctx, n.Request.JobID, r.build(n), domain.CarryOverExtras)
	return nil
}

// build は、結末から記録する JobStatus を組み立てます。
func (r *StatusRecorder) build(n review.Notification) domain.JobStatus {
	req := fromReviewRequest(n.Request)

	if n.Result.Status == review.StatusFailed {
		return domain.NewFailedStatus(req, n.Err)
	}

	// スキップもジョブとしては正常終了です。成果物が無いことは Outcome が表します。
	status := domain.NewSucceededStatus(req, n.Result.Status)
	if n.Report != nil {
		status.Title = n.Report.Title
		status.Decision = n.Report.Verdict.Decision
		status.ReportURI = n.Request.StorageURI
	}
	return status
}
