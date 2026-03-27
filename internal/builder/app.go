package builder

import (
	"context"
	"fmt"
	"io"
	"net/url"

	"github.com/shouni/gcp-kit/tasks"
	"github.com/shouni/gemini-reviewer-core/ports"
	"github.com/shouni/go-http-kit/httpkit"
	"github.com/shouni/go-remote-io/remoteio/gcs"

	"git-gemini-web/internal/adapters"
	"git-gemini-web/internal/app"
	"git-gemini-web/internal/config"
)

// BuildContainer は外部サービスとの接続を確立し、依存関係を組み立てた app.Container を返します。
func BuildContainer(ctx context.Context, cfg *config.Config) (container *app.Container, err error) {
	var resources []io.Closer
	defer func() {
		if err != nil {
			for _, r := range resources {
				if r != nil {
					_ = r.Close()
				}
			}
		}
	}()

	// 1. HttpClient
	httpClient := httpkit.New(config.DefaultHTTPTimeout)

	// 2. I/O Infrastructure
	rio, err := buildRemoteIO(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to initialize IO components: %w", err)
	}
	resources = append(resources, rio)

	// 3. Task Enqueuer
	enqueuer, err := buildTaskEnqueuer(ctx, cfg)
	if err != nil {
		return nil, fmt.Errorf("failed to initialize task enqueuer: %w", err)
	}
	resources = append(resources, enqueuer)

	// 3. Prompt Adapter の構築
	promptGen, err := adapters.NewPromptAdapter()
	if err != nil {
		return nil, fmt.Errorf("PromptAdapter の構築に失敗しました: %w", err)
	}

	// 4. Slack Adapter
	slack, err := adapters.NewSlackAdapter(httpClient, cfg.SlackWebhookURL)
	if err != nil {
		return nil, fmt.Errorf("failed to initialize Slack adapter: %w", err)
	}

	appCtx := &app.Container{
		Config:       cfg,
		RemoteIO:     rio,
		TaskEnqueuer: enqueuer,
		HTTPClient:   httpClient,
		PromptGen:    promptGen,
		Notifier:     slack,
	}

	// 5. Pipeline (Core Logic)
	reviewPipeline, err := buildPipeline(ctx, appCtx.Config, appCtx.RemoteIO, appCtx.Notifier, appCtx.PromptGen)
	if err != nil {
		return nil, fmt.Errorf("failed to initialize review pipeline: %w", err)
	}
	appCtx.Pipeline = reviewPipeline

	return appCtx, nil
}

// buildRemoteIO は、GCS ベースの I/O コンポーネントを初期化します。
func buildRemoteIO(ctx context.Context) (*app.RemoteIO, error) {
	factory, err := gcs.New(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to create GCS factory: %w", err)
	}
	w, err := factory.OutputWriter()
	if err != nil {
		return nil, fmt.Errorf("failed to create output writer: %w", err)
	}
	s, err := factory.URLSigner()
	if err != nil {
		return nil, fmt.Errorf("failed to create URL signer: %w", err)
	}
	return &app.RemoteIO{
		Factory: factory,
		Writer:  w,
		Signer:  s,
	}, nil
}

// buildTaskEnqueuer は、Cloud Tasks エンキューアを初期化します。
func buildTaskEnqueuer(ctx context.Context, cfg *config.Config) (*tasks.Enqueuer[ports.ReviewRequest], error) {
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
	}
	return tasks.NewEnqueuer[ports.ReviewRequest](ctx, taskCfg)
}
