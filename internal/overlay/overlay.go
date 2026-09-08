package overlay

import (
	"fmt"
	"sync"

	"github.com/lxn/walk"
	"github.com/lxn/win"
	"github.com/shivas/abyss-blackbox/internal/config"
)

type Mode int

const (
	ModePlacement Mode = iota
	ModeLive
)

type Overlay struct {
	overlayWindow *OverlayDialog
	position      walk.Rectangle
	mode          Mode
	config        *OverlayConfig
	overlayState  *overlayState
	reopen        bool
	captureConfig *config.CaptureConfig
}

type WidgetProperty string

const (
	Status     WidgetProperty = "status"
	Weather    WidgetProperty = "weather"
	TODO       WidgetProperty = "todo"
	Override   WidgetProperty = "override"
	Autoupload WidgetProperty = "autoupload"
)

type stateItem struct {
	text  string
	color *walk.Color
}

type overlayState struct {
	sync.Mutex
	items map[WidgetProperty]stateItem
}

func New(c *OverlayConfig, captureConfig *config.CaptureConfig) *Overlay {
	mode := ModePlacement
	if captureConfig.OverlayPosition.Height != 0 {
		mode = ModeLive
	}

	return &Overlay{
		mode:          mode,
		config:        c,
		captureConfig: captureConfig,
		overlayState: &overlayState{
			items: map[WidgetProperty]stateItem{
				Status:     {text: "status text", color: &YellowColor},
				Weather:    {text: ""},
				TODO:       {text: "Long text message can be here"},
				Override:   {text: "Manual override text", color: &CyanColor},
				Autoupload: {text: "Autoupload status", color: &CyanColor},
			},
		},
	}
}

func (o *Overlay) ToggleOverlay() {
	if o.overlayWindow != nil && o.overlayWindow.Dialog != nil {
		o.Close()
	} else {
		o.Show()
	}
}

func (o *Overlay) ChangeProperty(prop WidgetProperty, text string, color *walk.Color) {
	o.overlayState.Lock()

	o.overlayState.items[prop] = stateItem{text: text, color: color}

	o.overlayState.Unlock()

	if o.overlayWindow != nil && o.overlayWindow.Dialog.Visible() {
		_ = o.overlayWindow.Dialog.Invalidate()
	}
}

func (o *Overlay) Close() {
	if o.overlayWindow != nil && o.overlayWindow.Dialog != nil {
		o.overlayWindow.Dialog.Cancel()
		o.overlayWindow.Dialog.Dispose()
		o.overlayWindow.Dialog = nil
		o.overlayWindow = nil
	}
}

func (o *Overlay) Show() {
	if o.overlayWindow != nil && o.overlayWindow.Dialog != nil {
		fmt.Printf("can't show new dialog - already one running")
		return
	}

	defer func() {
		o.overlayWindow = nil
		if o.reopen {
			o.reopen = false
			o.Show()
		}
	}()

	o.position = o.captureConfig.OverlayPosition

	o.overlayWindow = CreateDialog(nil, o.config, o.overlayState)
	handle := o.overlayWindow.Dialog.Handle()

	if o.mode == ModePlacement {
		applyOverlayExStyle(handle)
		setClientScreenRectPixels(o.overlayWindow.Dialog, o.position)

		o.overlayWindow.Dialog.BoundsChanged().Attach(func() {
			// Native client pixels, not Bounds() (96-DPI DIPs).
			o.position = clientScreenRectPixels(o.overlayWindow.Dialog)
		})

		o.overlayWindow.Dialog.Closing().Once(func(canceled *bool, reason walk.CloseReason) {
			o.position = clientScreenRectPixels(o.overlayWindow.Dialog)
			o.captureConfig.OverlayPosition = o.position
			_ = config.Write(o.captureConfig)
			o.mode = ModeLive
			o.reopen = true
		})
	}

	if o.mode == ModeLive {
		stripLiveChrome(handle)
		applyOverlayExStyle(handle)
		win.SetWindowPos(handle, win.HWND_TOPMOST, 0, 0, 0, 0,
			win.SWP_NOACTIVATE|win.SWP_NOMOVE|win.SWP_NOSIZE|win.SWP_FRAMECHANGED)

		setClientScreenRectPixels(o.overlayWindow.Dialog, o.position)
		sz := walk.Size{Width: o.position.Width, Height: o.position.Height}
		_ = o.overlayWindow.Dialog.SetMinMaxSizePixels(sz, sz)

		closeAction := walk.NewAction()
		closeAction.SetText("Close")
		closeAction.Triggered().Once(func() {
			o.overlayWindow.Dialog.Accept()
		})

		placementModeAction := walk.NewAction()
		placementModeAction.SetText("Switch to placement mode")
		placementModeAction.Triggered().Once(func() {
			o.mode = ModePlacement
			o.reopen = true
			o.overlayWindow.Dialog.Accept()
		})

		settingsAction := walk.NewAction()
		settingsAction.SetText("Customization settings")
		settingsAction.Triggered().Once(func() {
			code, _ := RunSettingsDialog(o.overlayWindow.Dialog, o.overlayWindow.config, func(c *OverlayConfig) {
				c.BackgroundColor = parseColor(c.BackgroundColorText)
			}, func() {
				_ = o.overlayWindow.Widget.Invalidate() // redraw
			})

			if code == 1 {
				// persist to configuration
				AssignToConfig(o.captureConfig, o.overlayWindow.config)
				_ = config.Write(o.captureConfig)
			}

			o.reopen = true
			o.overlayWindow.Dialog.Accept()
		})

		menu, _ := walk.NewMenu()
		menu.Actions().Add(placementModeAction)
		menu.Actions().Add(settingsAction)
		menu.Actions().Add(closeAction)

		o.overlayWindow.Dialog.SetContextMenu(menu)
	}

	o.overlayWindow.Dialog.Run()
}

func applyOverlayExStyle(hwnd win.HWND) {
	flags := win.GetWindowLong(hwnd, win.GWL_EXSTYLE)
	flags |= win.WS_EX_NOACTIVATE
	flags |= win.WS_EX_TOPMOST
	flags |= win.WS_EX_NOPARENTNOTIFY
	win.SetWindowLong(hwnd, win.GWL_EXSTYLE, flags)
}

func stripLiveChrome(hwnd win.HWND) {
	style := win.GetWindowLong(hwnd, win.GWL_STYLE)
	style |= win.WS_OVERLAPPED
	style &^= win.WS_SIZEBOX
	style &^= win.WS_CAPTION
	win.SetWindowLong(hwnd, win.GWL_STYLE, style)
}

// clientScreenRectPixels is the dialog client area in native screen pixels.
// OverlayPosition must stay in this space — walk.Bounds() is 96-DPI DIPs.
func clientScreenRectPixels(dlg *walk.Dialog) walk.Rectangle {
	cb := dlg.ClientBoundsPixels()
	origin := win.POINT{}
	if !win.ClientToScreen(dlg.Handle(), &origin) {
		return walk.Rectangle{Width: cb.Width, Height: cb.Height}
	}

	return walk.Rectangle{
		X:      int(origin.X),
		Y:      int(origin.Y),
		Width:  cb.Width,
		Height: cb.Height,
	}
}

func chromeInsetsPixels(dlg *walk.Dialog) (left, top, right, bottom int) {
	outer := dlg.BoundsPixels()
	client := clientScreenRectPixels(dlg)
	left = client.X - outer.X
	top = client.Y - outer.Y
	right = outer.Width - client.Width - left
	bottom = outer.Height - client.Height - top

	return
}

func setClientScreenRectPixels(dlg *walk.Dialog, client walk.Rectangle) {
	if client.Width <= 0 || client.Height <= 0 {
		return
	}

	left, top, right, bottom := chromeInsetsPixels(dlg)
	win.SetWindowPos(dlg.Handle(), win.HWND_TOPMOST,
		int32(client.X-left),
		int32(client.Y-top),
		int32(client.Width+left+right),
		int32(client.Height+top+bottom),
		win.SWP_NOACTIVATE)
}
