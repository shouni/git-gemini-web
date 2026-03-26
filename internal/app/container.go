package app

import (
	"log/slog"

	"github.com/shouni/gcp-kit/tasks"
	"github.com/shouni/go-http-kit/httpkit"
	"github.com/shouni/go-remote-io/remoteio"

	"git-gemini-web/internal/config"
	"git-gemini-web/internal/domain"
)

// Container はアプリケーションの依存関係（DIコンテナ）を保持します。
type Container struct {
	Config *config.Config
	// I/O and Storage
	RemoteIO *RemoteIO
	// Asynchronous Task
	TaskEnqueuer *tasks.Enqueuer[domain.ReviewRequest]
	// Business Logic
	Pipeline domain.Pipeline
	// External Adapters
	HTTPClient    httpkit.Requester
	SlackNotifier domain.Notifier
	PromptGen     domain.PromptGenerator
}

// RemoteIO は外部ストレージ操作に関するコンポーネントをまとめます。
type RemoteIO struct {
	Factory remoteio.IOFactory
	Writer  remoteio.OutputWriter
	Signer  remoteio.URLSigner
}

// Close は、RemoteIO が保持する Factory などの内部リソースを解放します。
func (r *RemoteIO) Close() error {
	if r.Factory != nil {
		return r.Factory.Close()
	}
	return nil
}

// Close は、Container が保持するすべての外部接続リソースを安全に解放します。
func (c *Container) Close() {
	// RemoteIO のリソース解放を委譲
	if c.RemoteIO != nil {
		if err := c.RemoteIO.Close(); err != nil {
			slog.Error("failed to close RemoteIO", "error", err)
		}
	}

	// TaskEnqueuer のリソース解放
	if c.TaskEnqueuer != nil {
		if err := c.TaskEnqueuer.Close(); err != nil {
			slog.Error("failed to close task enqueuer", "error", err)
		}
	}
}
