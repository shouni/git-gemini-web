package adapters

import (
	"context"

	coreports "github.com/shouni/gemini-reviewer-core/ports"
	"github.com/shouni/gemini-reviewer-core/workflow"

	"git-gemini-web/internal/domain"
)

// CoreWorkflowAdapter は core workflow を domain.Pipeline として公開する ACL です。
type CoreWorkflowAdapter struct {
	core *workflow.Workflow
}

// NewCoreWorkflowAdapter は core workflow をラップしたアダプターを返します。
func NewCoreWorkflowAdapter(core *workflow.Workflow) *CoreWorkflowAdapter {
	return &CoreWorkflowAdapter{core: core}
}

// Execute は domain モデルを core モデルへ変換して実行します。
func (a *CoreWorkflowAdapter) Execute(ctx context.Context, req domain.ReviewRequest) error {
	return a.core.Execute(ctx, toCoreReviewRequest(req))
}

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
