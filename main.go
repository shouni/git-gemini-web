package main

import (
	"context"
	"log/slog"
	"os"
	"os/signal"
	"syscall"

	"git-gemini-web/internal/config"
	"git-gemini-web/internal/server"
)

func main() {
	// 1. 構造化ログの設定
	slog.SetDefault(slog.New(slog.NewJSONHandler(os.Stdout, nil)))

	// 2. シグナルを監視するコンテキスト
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	// 3. 設定のロードとバリデーション
	cfg := config.LoadConfig()
	if err := cfg.ValidateEssentialConfig(); err != nil {
		slog.Error("必須設定のバリデーションに失敗しました", "error", err)
		os.Exit(1)
	}

	// 4. サーバー実行
	if err := server.Run(ctx, cfg); err != nil {
		slog.Error("アプリケーションが異常終了しました", "error", err)
		os.Exit(1)
	}
}
