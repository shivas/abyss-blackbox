package main

import (
	"os"

	"golang.org/x/exp/slog"

	"github.com/shivas/abyss-blackbox/internal/app"
)

func main() {
	slog.SetDefault(slog.New(slog.HandlerOptions{Level: slog.LevelDebug}.NewTextHandler(os.Stdout)))

	err := app.Run()
	if err != nil {
		slog.Error("error: ", err)
	}
}
