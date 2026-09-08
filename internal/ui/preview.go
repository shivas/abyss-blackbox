package ui

import (
	"image"
	"sync"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
)

type Preview struct {
	mu  sync.Mutex
	img *canvas.Image
}

func newPreview(width, height float32) *Preview {
	blank := image.NewRGBA(image.Rect(0, 0, int(width), int(height)))
	im := canvas.NewImageFromImage(blank)
	im.FillMode = canvas.ImageFillOriginal
	im.ScaleMode = canvas.ImageScalePixels

	return &Preview{img: im}
}

func (p *Preview) Canvas() fyne.CanvasObject {
	return p.img
}

func (p *Preview) SetSize(_, _ float32) {
	// ImageFillOriginal sizes from capture pixels on the next SetImage.
}

func (p *Preview) SetImage(im image.Image) {
	p.mu.Lock()
	p.img.Image = im
	p.mu.Unlock()

	fyne.Do(func() {
		p.img.Refresh()
	})
}
