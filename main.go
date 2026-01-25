package main

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"os"

	"git-gemini-web/internal/builder"
	"git-gemini-web/internal/config"
	"git-gemini-web/internal/pipeline"
)

func main() {
	slog.SetDefault(slog.New(slog.NewJSONHandler(os.Stdout, nil)))

	if err := run(context.Background()); err != nil {
		slog.Error("アプリケーションの実行に失敗しました", "error", err)
		os.Exit(1)
	}
}

// run は、設定ロード、バリデーション、サーバーのライフサイクル管理を行う
func run(ctx context.Context) error {
	// 1. 設定のロード
	cfg := config.LoadConfig()

	// 2. 必須設定のチェックとセキュリティバリデーション
	if err := config.ValidateEssentialConfig(cfg); err != nil {
		return err
	}

	slog.Info("🛠️ サーバー依存関係を構築中...")

	// 3. サーバーハンドラーの構築と依存関係の取得
	appCtx, err := builder.BuildAppContext(ctx, cfg)
	if err != nil {
		return fmt.Errorf("failed to build application context: %w", err)
	}

	// リソースを解放する
	defer func() {
		slog.Info("♻️ アプリケーションコンテキストをクローズ中...")
		appCtx.Close()
	}()

	// 4. パイプラインの構築
	reviewPipeline, err := pipeline.NewReviewPipeline(ctx, appCtx)
	if err != nil {
		return fmt.Errorf("failed to build reviewPipeline: %w", err)
	}

	// 5. サーバーハンドラーの作成
	handler, err := builder.NewServerHandler(ctx, appCtx, reviewPipeline)
	if err != nil {
		return fmt.Errorf("サーバーの構築に失敗しました: %w", err)
	}

	// 6. サーバー起動
	slog.Info("🚀 サーバーを起動中...", "port", cfg.Port)
	if err := http.ListenAndServe(":"+cfg.Port, handler); err != nil && err != http.ErrServerClosed {
		return fmt.Errorf("サーバーの起動に失敗しました: %w", err)
	}

	return nil
}
