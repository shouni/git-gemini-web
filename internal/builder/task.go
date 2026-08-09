package builder

import (
	"context"
	"fmt"
	"net/url"

	"github.com/shouni/gcp-kit/tasks"

	"github.com/shouni/git-gemini-web/internal/config"
	"github.com/shouni/git-gemini-web/internal/domain"
)

// buildTaskEnqueuer は、Cloud Tasks エンキューアを初期化します。
func buildTaskEnqueuer(ctx context.Context, cfg *config.Config) (*tasks.Enqueuer[domain.ReviewRequest], error) {
	workerURL, err := url.JoinPath(cfg.ServiceURL, "/tasks/execute_review")
	if err != nil {
		return nil, fmt.Errorf("failed to build worker URL: %w", err)
	}

	taskCfg := tasks.Config{
		ProjectID:           cfg.ProjectID,
		LocationID:          cfg.LocationID,
		QueueID:             cfg.QueueID,
		WorkerURL:           workerURL,
		ServiceAccountEmail: cfg.ServiceAccountEmail,
		Audience:            cfg.TaskAudienceURL,
		// ★ 未指定だと Cloud Tasks の既定 10 分が実効上限になり、Cloud Run の timeout を
		//   いくら伸ばしても 10 分で打ち切られます。2026-08-10 まで指定が無く、
		//   Cloud Run 側の 600s と偶然一致しているだけの状態でした。
		//   PIPELINE_TIMEOUT をこれより短く取り、アプリが自分で先に諦めます。
		DispatchDeadline: config.TaskDispatchDeadline,
	}
	return tasks.NewEnqueuer[domain.ReviewRequest](ctx, taskCfg)
}
