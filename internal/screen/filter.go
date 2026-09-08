package screen

import (
	"image"
	"image/color"

	"github.com/disintegration/imaging"
)

// FilterFrame returns the raw capture and the two-color paletted frame used for recording.
func FilterFrame(img image.Image, threshold uint32) (preview image.Image, recording *image.Paletted) {
	gray := imaging.Grayscale(img)
	gray = imaging.Invert(gray)

	return img, Paletted(gray, threshold)
}

// Paletted converts an NRGBA image to a two-color paletted image using cutoff as separator.
func Paletted(img *image.NRGBA, cutoff uint32) *image.Paletted {
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

			buffimg.SetColorIndex(x, y, 1)
		}
	}

	return buffimg
}
