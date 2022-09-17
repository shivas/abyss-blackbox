package overlay

import (
	"encoding/hex"
	"strings"

	"github.com/lxn/walk"
	"github.com/shivas/abyss-blackbox/internal/config"
)

type OverlayConfig struct {
	FontFamily          string
	FontSize            int
	Color               walk.Color
	ColorText           string
	BackgroundColor     walk.Color
	BackgroundColorText string
	Spacing             int
}

func FromCaptureConfig(c *config.CaptureConfig) *OverlayConfig {
	o := &OverlayConfig{}
	o.BackgroundColor = parseColor(c.OverlayConfig.BackgroundColor)
	o.BackgroundColorText = c.OverlayConfig.BackgroundColor
	o.Color = parseColor(c.OverlayConfig.Color)
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

func parseColor(s string) walk.Color {
	def := walk.RGB(0, 0, 0)

	if !strings.HasPrefix(s, "#") || len(s) != 7 { // #RRGGBB
		return def
	}

	r := hexToByte(s[1:3])
	g := hexToByte(s[3:5])
	b := hexToByte(s[5:7])

	return walk.RGB(r, g, b)
}

func hexToByte(s string) byte {
	b, err := hex.DecodeString(s)
	if err != nil {
		return 0
	}

	return b[0]
}
