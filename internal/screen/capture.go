package screen

import (
	"image"

	"github.com/shivas/abyss-blackbox/internal/config"
	"github.com/shivas/abyss-blackbox/internal/window"
)

type ScreenCapturer interface {
	CaptureWindowArea() (image.Image, error)
}

func NewDirectX11Capture(cfg *config.CaptureConfig, manager *window.Manager, captureWidth func() int) *DirectX11Capture {
	return &DirectX11Capture{
		cfg:            cfg,
		manager:        manager,
		captureWidthFn: captureWidth,
	}
}

type DirectX11Capture struct {
	cfg            *config.CaptureConfig
	manager        *window.Manager
	captureWidthFn func() int
}

func (c *DirectX11Capture) CaptureWindowArea() (image.Image, error) {
	rect := image.Rectangle{Min: image.Point{X: c.cfg.X, Y: c.cfg.Y}, Max: image.Point{X: c.cfg.X + c.captureWidthFn(), Y: c.cfg.Y + c.cfg.H}}
	return captureWindow(
		c.manager.GetHandleByTitle(c.cfg.EVEClientWindowTitle),
		rect,
	)
}
