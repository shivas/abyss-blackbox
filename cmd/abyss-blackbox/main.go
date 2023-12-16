package main

import (
	"log/slog"
	"os"

	"github.com/shivas/abyss-blackbox/internal/app"
)

func main() {
	logger := slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelDebug}))
	slog.SetDefault(logger)

	err := app.Run()
	if err != nil {
		slog.Error("error: ", err)
	}
}
