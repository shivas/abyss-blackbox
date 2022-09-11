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

type OverlayConfig struct {
	FontFamily      string
	FontSize        int
	Color           walk.Color
	BackgroundColor walk.Color
}

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
	text string
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
				Status:     {text: "status text"},
				Weather:    {text: ""},
				TODO:       {text: "Long text message can be here"},
				Override:   {text: "Manual override text"},
				Autoupload: {text: "Autoupload status"},
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

func (o *Overlay) ChangeProperty(prop WidgetProperty, text string) {
	o.overlayState.Lock()

	o.overlayState.items[prop] = stateItem{text: text}

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
		win.SetWindowPos(handle,
			win.HWND_TOPMOST, int32(o.position.X), int32(o.position.Y), int32(o.position.Height), int32(o.position.Width),
			win.SWP_NOACTIVATE)

		p := o.position
		p.Y -= 32

		if p.Y < 0 {
			p.Y = 0
		}

		_ = o.overlayWindow.Dialog.SetBoundsPixels(p)

		o.overlayWindow.Dialog.BoundsChanged().Attach(func() {
			o.position = o.overlayWindow.Dialog.Bounds()
			o.position.Y += 32
			o.position.X += 8
		})

		flags := win.GetWindowLong(handle, win.GWL_EXSTYLE)
		flags |= win.WS_EX_NOACTIVATE
		flags |= win.WS_EX_TOPMOST
		flags |= win.WS_EX_NOPARENTNOTIFY

		win.SetWindowLong(handle, win.GWL_EXSTYLE, flags)

		o.overlayWindow.Dialog.Closing().Once(func(canceled *bool, reason walk.CloseReason) {
			o.captureConfig.OverlayPosition = o.position
			_ = config.Write(o.captureConfig)
			o.mode = ModeLive
			o.reopen = true
		})
	}

	if o.mode == ModeLive {
		win.SetWindowPos(handle,
			win.HWND_TOPMOST, int32(o.position.X), int32(o.position.Y), int32(o.position.Height), int32(o.position.Width),
			win.SWP_NOACTIVATE|win.SWP_NOMOVE|win.SWP_NOSIZE)
		o.overlayWindow.Dialog.SetBoundsPixels(o.position)
		o.overlayWindow.Dialog.SetMinMaxSizePixels(walk.Size{Width: o.position.Width - 16, Height: o.position.Height - 40}, walk.Size{Width: o.position.Width - 16, Height: o.position.Height - 40})
		// o.overlayWindow.SetBounds(o.position)

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

		menu, _ := walk.NewMenu()
		menu.Actions().Add(placementModeAction)
		menu.Actions().Add(closeAction)

		o.overlayWindow.Dialog.SetContextMenu(menu)

		flag := win.GetWindowLong(handle, win.GWL_STYLE) // Gets current style
		flag |= win.WS_OVERLAPPED                        // always on top
		flag &= ^win.WS_SIZEBOX
		flag &= ^win.WS_CAPTION
		win.SetWindowLong(handle, win.GWL_STYLE, flag)

		flag2 := win.GetWindowLong(handle, win.GWL_EXSTYLE)
		flag2 |= win.WS_EX_NOACTIVATE
		flag2 |= win.WS_EX_TOPMOST
		flag2 |= win.WS_EX_NOPARENTNOTIFY

		win.SetWindowLong(handle, win.GWL_EXSTYLE, flag2)
	}

	o.overlayWindow.Dialog.Run()
}
