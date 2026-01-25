package main

import (
	"context"
	"log/slog"
	"os"

	"git-gemini-web/internal/server"
)

func main() {
	slog.SetDefault(slog.New(slog.NewJSONHandler(os.Stdout, nil)))

	if err := server.Run(context.Background()); err != nil {
		slog.Error("アプリケーションの実行に失敗しました", "error", err)
		os.Exit(1)
	}
}
