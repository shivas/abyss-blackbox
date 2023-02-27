package mainwindow

import (
	"image"
	"image/color"
	"os"
	"time"

	"github.com/disintegration/imaging"
	"github.com/lxn/walk"
	"github.com/lxn/win"
	"github.com/shivas/abyss-blackbox/internal/config"
	"github.com/shivas/abyss-blackbox/internal/screen"
	"golang.org/x/exp/slog"
)

func CustomWidgetDrawLoop(
	cw *walk.CustomWidget,
	mw *walk.MainWindow,
	capturer screen.ScreenCapturer,
	currentSettings *config.CaptureConfig,
	previewChannel chan image.Image,
	recordingChannel chan *image.Paletted,

) {
	t := time.NewTicker(time.Second)
	defer t.Stop()

	for range t.C {
		img, err := capturer.CaptureWindowArea()
		if err != nil {
			walk.MsgBox(mw, "Error capturing window area, restart of application is needed.", err.Error(), walk.MsgBoxIconWarning)
			slog.Error("error lost capture window: %v", err)
			os.Exit(1) // exit
		}

		img2 := imaging.Grayscale(img)
		img2 = imaging.Invert(img2)
		img3 := paletted(img2, uint32(currentSettings.FilterThreshold))

		if currentSettings.FilteredPreview {
			select {
			case previewChannel <- img3:
			default:
				slog.Debug("preview channel full, dropping frame")
			}
		} else {
			select {
			case previewChannel <- img:
			default:
				slog.Debug("preview channel full, dropping frame")
			}
		}

		select {
		case recordingChannel <- img3:
		default:
			slog.Debug("recorder channel is full, dropping frame")
		}

		win.InvalidateRect(cw.Handle(), nil, false)
	}
}

// paletted convert image in NRGBA mode to Paletted image given cutoff as separator between two colors
func paletted(img *image.NRGBA, cutoff uint32) *image.Paletted {
	palette := []color.Color{color.White, color.Black}
	buffimg := image.NewPaletted(img.Rect, palette)

	threshold := cutoff << 8

	for y := 0; y < img.Rect.Dy(); y++ {
		for x := 0; x < img.Rect.Dx(); x++ {
			c := img.At(x, y)
			r, _, _, _ := c.RGBA()

			if r > threshold {
				buffimg.SetColorIndex(x, y, 0)
				continue
			}

			if r <= threshold {
				buffimg.SetColorIndex(x, y, 1)
				continue
			}
		}
	}

	return buffimg
}
