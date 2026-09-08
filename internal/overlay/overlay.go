package overlay

import (
	"sync"

	"github.com/shivas/abyss-blackbox/internal/config"
)

type Mode int

const (
	ModePlacement Mode = iota
	ModeLive
)

type Overlay struct {
	mu            sync.Mutex
	hwnd          uintptr
	visible       bool
	reopen        bool
	skipModeFlip  bool
	mode          Mode
	config        *OverlayConfig
	overlayState  *overlayState
	captureConfig *config.CaptureConfig
	OnSettings    func()
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
	color *Color
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
	if o.Visible() {
		o.Close()
		return
	}

	o.Show()
}

func (o *Overlay) Visible() bool {
	o.mu.Lock()
	defer o.mu.Unlock()

	return o.visible
}

func (o *Overlay) ChangeProperty(prop WidgetProperty, text string, color *Color) {
	o.overlayState.Lock()
	o.overlayState.items[prop] = stateItem{text: text, color: color}
	o.overlayState.Unlock()
	o.invalidate()
}

func (o *Overlay) ApplyConfig(c *OverlayConfig) {
	o.mu.Lock()
	o.config = c
	o.mu.Unlock()
	o.invalidate()
}

func (o *Overlay) Config() *OverlayConfig {
	return o.config
}
