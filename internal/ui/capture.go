package ui

import (
	"log/slog"
	"time"

	"github.com/shivas/abyss-blackbox/internal/screen"
)

func (a *App) startCapture() {
	if a.deps.Windows == nil || a.deps.Config == nil {
		return
	}

	capturer := screen.NewDirectX11Capture(a.deps.Config, a.deps.Windows, func() int {
		return a.captureWidth
	})

	go func() {
		t := time.NewTicker(time.Second)
		defer t.Stop()

		for range t.C {
			img, err := capturer.CaptureWindowArea()
			if err != nil {
				slog.Error("error capturing window", slog.Any("err", err))
				continue
			}

			raw, pal := screen.FilterFrame(img, uint32(a.deps.Config.FilterThreshold))
			if a.deps.Config.FilteredPreview {
				a.preview.SetImage(pal)
			} else {
				a.preview.SetImage(raw)
			}

			if a.deps.Recording != nil {
				select {
				case a.deps.Recording <- pal:
				default:
					slog.Debug("recorder channel is full, dropping frame")
				}
			}
		}
	}()
}
