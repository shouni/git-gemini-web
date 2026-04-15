package builder

import (
	"fmt"

	"git-gemini-web/internal/app"

	"github.com/shouni/go-remote-io/remoteio"
)

// buildRemoteIO は、 I/O コンポーネントを初期化します。
func buildRemoteIO(storage remoteio.IOFactory) (*app.RemoteIO, error) {
	w, err := storage.OutputWriter()
	if err != nil {
		return nil, fmt.Errorf("出力ライターの生成に失敗しました: %w", err)
	}
	s, err := storage.URLSigner()
	if err != nil {
		return nil, fmt.Errorf("URL署名器の生成に失敗しました: %w", err)
	}
	return &app.RemoteIO{
		Factory: storage,
		Writer:  w,
		Signer:  s,
	}, nil
}
