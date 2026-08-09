package adapters

import (
	"context"
	"time"

	coreports "github.com/shouni/gemini-reviewer-core/ports"
	"github.com/shouni/gemini-reviewer-core/workflow"

	"github.com/shouni/git-gemini-web/internal/domain"
)

// CoreWorkflowAdapter は core workflow を domain.Pipeline として公開する ACL です。
type CoreWorkflowAdapter struct {
	core    *workflow.Workflow
	timeout time.Duration
}

// NewCoreWorkflowAdapter は core workflow をラップしたアダプターを返します。
//
// timeout はレビュー 1 件の実行時間の上限（PIPELINE_TIMEOUT）です。0 以下なら無制限。
func NewCoreWorkflowAdapter(core *workflow.Workflow, timeout time.Duration) *CoreWorkflowAdapter {
	return &CoreWorkflowAdapter{core: core, timeout: timeout}
}

// Execute は domain モデルを core モデルへ変換して実行します。
//
// ★ ここで締切を被せることで、**Cloud Tasks より先にアプリが諦めます**。
//
// Cloud Tasks に先を越されるとプロセスごと SIGTERM になり、失敗レポートも Slack 通知も
// 残りません（review-queue は max_attempts = 1 なので再試行も来ず、タスクは黙って
// 失われます）。自分で諦めれば、打ち切りの理由が Outcome に載って publisher まで届きます。
//
// 締切が公開処理を巻き込まないことは core 側が保証しています
// （workflow.Execute が context を切り離す）。
func (a *CoreWorkflowAdapter) Execute(ctx context.Context, req domain.ReviewRequest) error {
	if a.timeout > 0 {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, a.timeout)
		defer cancel()
	}
	return a.core.Execute(ctx, toCoreReviewRequest(req))
}

// toCoreReviewRequest は domain.ReviewRequest を coreports.ReviewRequest に変換します。
func toCoreReviewRequest(req domain.ReviewRequest) coreports.ReviewRequest {
	return coreports.ReviewRequest{
		RepoURL:       req.RepoURL,
		BaseBranch:    req.BaseBranch,
		FeatureBranch: req.FeatureBranch,
		Mode:          req.Mode,
		ModelName:     req.ModelName,
		StorageURI:    req.StorageURI,
		PublicURL:     req.PublicURL,
	}
}
