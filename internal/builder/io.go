package builder

import (
	"context"
	"fmt"

	"github.com/shouni/go-remote-io/remoteio/gcs"

	"git-gemini-web/internal/app"
)

// buildRemoteIO は、I/O コンポーネントを初期化します。
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
