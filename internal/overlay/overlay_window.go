package overlay

//nolint:revive,stylecheck // side effects

import (
	"github.com/lxn/walk"
	. "github.com/lxn/walk/declarative"
)

var PrimaryColor walk.Color = walk.RGB(194, 176, 226)
var SecondaryColor walk.Color = walk.RGB(239, 108, 0)
var GreenColor walk.Color = walk.RGB(76, 175, 80)
var YellowColor walk.Color = walk.RGB(223, 235, 37)
var CyanColor walk.Color = walk.RGB(0, 137, 123)
var RedColor walk.Color = walk.RGB(255, 47, 0)

type OverlayDialog struct {
	Dialog *walk.Dialog
	Widget *walk.CustomWidget
	config *OverlayConfig
	state  *overlayState
}

func CreateDialog(owner walk.Form, c *OverlayConfig, s *overlayState) *OverlayDialog {
	od := OverlayDialog{config: c, state: s}

	Dialog{
		AssignTo: &od.Dialog,
		Visible:  false,
		Title:    "Overlay",
		//DefaultButton: &acceptPB,
		Layout: VBox{MarginsZero: true, SpacingZero: true},
		Children: []Widget{
			CustomWidget{
				AssignTo:            &od.Widget,
				ClearsBackground:    true,
				InvalidatesOnResize: true,
				Paint:               od.drawStuff,
			},
		},
	}.Create(owner)

	return &od
}

func (o *OverlayDialog) drawStuff(canvas *walk.Canvas, updateBounds walk.Rectangle) error {
	o.state.Lock()
	defer o.state.Unlock()

	bounds := o.Widget.ClientBounds()

	back, err := walk.NewSolidColorBrush(o.config.BackgroundColor)
	if err != nil {
		return err
	}

	canvas.FillRectangle(back, bounds)

	// rectPen, err := walk.NewCosmeticPen(walk.PenSolid, walk.RGB(255, 0, 0))
	// if err != nil {
	// 	return err
	// }

	// defer rectPen.Dispose()

	// if err := canvas.DrawRectanglePixels(rectPen, bounds); err != nil {
	// 	return err
	// }

	bounds.Width -= 10
	bounds.X += 5
	bounds.Height -= 10
	bounds.Y += 5

	headerFont, err := walk.NewFont(o.config.FontFamily, 9, walk.FontBold)
	if err != nil {
		return err
	}
	defer headerFont.Dispose()

	if err = canvas.DrawTextPixels("abyssal.space telemetry overlay", headerFont, PrimaryColor, bounds, walk.TextWordbreak); err != nil {
		return err
	}

	bounds.Y += o.config.FontSize + 5

	font, err := walk.NewFont(o.config.FontFamily, o.config.FontSize, 0)
	if err != nil {
		return err
	}
	defer font.Dispose()

	if err := canvas.DrawTextPixels(o.state.items[Status].text, font, lineColor(o.state.items[Status], o.config.Color), bounds, walk.TextWordbreak); err != nil {
		return err
	}

	bounds.Y += o.config.FontSize + 8
	if err := canvas.DrawTextPixels(o.state.items[Weather].text, font, lineColor(o.state.items[Weather], o.config.Color), bounds, walk.TextWordbreak); err != nil {
		return err
	}

	bounds.Y += o.config.FontSize + 6
	if err := canvas.DrawTextPixels(o.state.items[TODO].text, font, lineColor(o.state.items[TODO], o.config.Color), bounds, walk.TextWordbreak); err != nil {
		return err
	}

	bounds.Y += o.config.FontSize + 10
	if err := canvas.DrawTextPixels(o.state.items[Override].text, font, lineColor(o.state.items[Override], o.config.Color), bounds, walk.TextWordbreak); err != nil {
		return err
	}

	bounds.Y += o.config.FontSize + 6
	if err := canvas.DrawTextPixels(o.state.items[Autoupload].text, font, lineColor(o.state.items[Autoupload], o.config.Color), bounds, walk.TextWordbreak); err != nil {
		return err
	}

	return nil
}

func lineColor(s stateItem, defaultColor walk.Color) (color walk.Color) {
	color = defaultColor
	if s.color != nil {
		color = *s.color
	}

	return
}
