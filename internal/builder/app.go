package builder

import (
	"context"
	"fmt"
	"log/slog"
	"net/url"

	"git-gemini-web/internal/adapters"
	"git-gemini-web/internal/config"
	"git-gemini-web/internal/domain"

	"github.com/shouni/gcp-kit/tasks"
	"github.com/shouni/go-http-kit/pkg/httpkit"
	"github.com/shouni/go-remote-io/pkg/gcsfactory"
	"github.com/shouni/go-remote-io/pkg/remoteio"
)

// AppContext はアプリケーションの依存関係を保持します。
type AppContext struct {
	Config        config.Config
	HTTPClient    httpkit.ClientInterface
	IOFactory     remoteio.IOFactory
	TaskEnqueuer  *tasks.Enqueuer[domain.ReviewRequest]
	SlackNotifier adapters.SlackNotifier
}

// BuildAppContext は外部サービスとの接続を確立し、依存関係を組み立てます。
func BuildAppContext(ctx context.Context, cfg config.Config) (*AppContext, error) {
	// 1. 基盤クライアントの初期化
	httpClient := httpkit.New(config.DefaultHTTPTimeout)

	// 2. I/O インフラ (GCS等) の初期化
	ioFactory, err := gcsfactory.New(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to create GCS factory: %w", err)
	}

	// 3. Cloud Tasks Enqueuer の初期化
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
	taskEnqueuer, err := tasks.NewEnqueuer[domain.ReviewRequest](ctx, taskCfg)
	if err != nil {
		return nil, fmt.Errorf("failed to initialize task enqueuer: %w", err)
	}

	// 4. Slack アダプター
	slackNotifier := adapters.NewSlackAdapter(httpClient, cfg.SlackWebhookURL)

	return &AppContext{
		Config:        cfg,
		HTTPClient:    httpClient,
		IOFactory:     ioFactory,
		TaskEnqueuer:  taskEnqueuer,
		SlackNotifier: slackNotifier,
	}, nil
}

// Close は、AppContextが保持するすべてのリソース（クライアント接続など）を解放します。
func (a *AppContext) Close() {
	if a.IOFactory != nil {
		if err := a.IOFactory.Close(); err != nil {
			slog.Error("failed to close IOFactory", "error", err)
		}
	}
	if a.TaskEnqueuer != nil {
		if err := a.TaskEnqueuer.Close(); err != nil {
			slog.Error("failed to close task enqueuer", "error", err)
		}
	}
}
