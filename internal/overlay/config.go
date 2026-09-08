package overlay

import (
	"encoding/hex"
	"strings"

	"github.com/shivas/abyss-blackbox/internal/config"
)

type Color struct {
	R, G, B byte
}

func (c Color) COLORREF() uint32 {
	return uint32(c.R) | uint32(c.G)<<8 | uint32(c.B)<<16
}

var (
	PrimaryColor   = Color{194, 176, 226}
	SecondaryColor = Color{239, 108, 0}
	GreenColor     = Color{76, 175, 80}
	YellowColor    = Color{223, 235, 37}
	CyanColor      = Color{0, 137, 123}
	RedColor       = Color{255, 47, 0}
)

type OverlayConfig struct {
	FontFamily          string
	FontSize            int
	Color               Color
	ColorText           string
	BackgroundColor     Color
	BackgroundColorText string
	Spacing             int
}

func FromCaptureConfig(c *config.CaptureConfig) *OverlayConfig {
	o := &OverlayConfig{}
	o.BackgroundColor = ParseColor(c.OverlayConfig.BackgroundColor)
	o.BackgroundColorText = c.OverlayConfig.BackgroundColor
	o.Color = ParseColor(c.OverlayConfig.Color)
	o.ColorText = c.OverlayConfig.Color
	o.FontFamily = c.OverlayConfig.FontFamily
	o.FontSize = c.OverlayConfig.FontSize
	o.Spacing = c.OverlayConfig.Spacing

	return o
}

func AssignToConfig(c *config.CaptureConfig, o *OverlayConfig) {
	c.OverlayConfig.BackgroundColor = o.BackgroundColorText
	c.OverlayConfig.Color = o.ColorText
	c.OverlayConfig.FontFamily = o.FontFamily
	c.OverlayConfig.FontSize = o.FontSize
	c.OverlayConfig.Spacing = o.Spacing
}

func ParseColor(s string) Color {
	def := Color{0, 0, 0}

	if !strings.HasPrefix(s, "#") || len(s) != 7 {
		return def
	}

	return Color{
		R: hexToByte(s[1:3]),
		G: hexToByte(s[3:5]),
		B: hexToByte(s[5:7]),
	}
}

func hexToByte(s string) byte {
	b, err := hex.DecodeString(s)
	if err != nil {
		return 0
	}

	return b[0]
}
