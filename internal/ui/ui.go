package ui

import (
	_ "embed"
	"image"
	"image/color"
	"log/slog"
	"net/http"
	"syscall"

	"fyne.io/fyne/v2/canvas"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/app"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/widget"

	"github.com/shivas/abyss-blackbox/internal/app/recorder"
	"github.com/shivas/abyss-blackbox/internal/charmanager"
	"github.com/shivas/abyss-blackbox/internal/config"
	"github.com/shivas/abyss-blackbox/internal/fittings"
	"github.com/shivas/abyss-blackbox/internal/overlay"
	"github.com/shivas/abyss-blackbox/internal/window"
	"github.com/shivas/abyss-blackbox/pkg/combatlog"
)

//go:embed triglogo.png
var logoPNG []byte

type Deps struct {
	Config               *config.CaptureConfig
	Recorder             *recorder.Recorder
	Overlay              *overlay.Overlay
	Chars                *charmanager.CharManager
	Fittings             *fittings.FittingsManager
	Windows              *window.Manager
	CombatLog            *combatlog.Reader
	HTTPClient           *http.Client
	Recording            chan *image.Paletted
	OnClose              func()
	RebindHotkeys        func()
	StartShortcutCapture func(func(config.Shortcut))
	StopShortcutCapture  func()
}

type App struct {
	fyneApp fyne.App
	win     fyne.Window
	deps    Deps

	preview      *Preview
	captureWidth int

	windowTitles  []string
	windowByTitle map[string]syscall.Handle

	windowSelect  *widget.Select
	xEntry        *widget.Entry
	yEntry        *widget.Entry
	hEntry        *widget.Entry
	filteredCheck *widget.Check
	scalingSelect *widget.Select
	overrideCheck *widget.Check
	shipSelect    *iconPicker
	tierSelect    *iconPicker
	weatherSelect *iconPicker
	runnerList    *widget.List
	runners       []*runnerRow
	autoUpload    *widget.Check
	recordBtn     *widget.Button
	addCharBtn    *widget.Button
	manageFitsBtn *widget.Button
	charSelect    *widget.Select
	presetSelect  *widget.Select
	lockables     []fyne.Disableable
}

type runnerRow struct {
	checked bool
	name    string
	ship    string
	fit     string
	icon    image.Image
}

var uiScales = []struct {
	title string
	width int
}{
	{"100", 255},
	{"110", 255},
	{"125", 300},
	{"150", 300},
	{"175", 400},
}

func New(deps Deps) *App {
	a := app.NewWithID("space.abyssal.blackbox")
	a.SetIcon(fyne.NewStaticResource("triglogo.png", logoPNG))

	w := a.NewWindow("Abyssal.Space Blackbox Recorder")
	w.Resize(fyne.NewSize(980, 720))

	ui := &App{
		fyneApp:       a,
		win:           w,
		deps:          deps,
		windowByTitle: make(map[string]syscall.Handle),
		captureWidth:  255,
	}

	for _, s := range uiScales {
		if s.title == deps.Config.EVEClientUIScaling {
			ui.captureWidth = s.width
		}
	}

	previewH := deps.Config.H
	if previewH < 200 {
		previewH = 200
	}
	ui.preview = newPreview(float32(ui.captureWidth), float32(previewH))

	if deps.Chars != nil {
		deps.Chars.OnRefresh = func() {
			fyne.Do(ui.refreshCharacters)
		}
	}

	if deps.Overlay != nil {
		deps.Overlay.OnSettings = func() {
			fyne.Do(ui.showOverlaySettings)
		}
	}

	return ui
}

func (a *App) Window() fyne.Window {
	return a.win
}

func (a *App) App() fyne.App {
	return a.fyneApp
}

func (a *App) ShowAndRun() {
	a.build()
	a.startCapture()

	a.win.SetCloseIntercept(func() {
		if a.deps.OnClose != nil {
			a.deps.OnClose()
		}
		a.win.Close()
	})

	a.win.CenterOnScreen()
	slog.Info("calling window.Show")
	a.win.Show()
	slog.Info("window.Show returned")

	slog.Info("entering fyne run loop")
	a.fyneApp.Run()
}

func (a *App) Notify(title, msg string) {
	if a.deps.Config.SuppressNotifications {
		return
	}

	fyne.Do(func() {
		a.fyneApp.SendNotification(fyne.NewNotification(title, msg))
	})
}

func (a *App) ToggleRecording() {
	fyne.Do(a.toggleRecording)
}

func (a *App) ToggleOverlay() {
	if a.deps.Overlay != nil {
		a.deps.Overlay.ToggleOverlay()
	}
}

func (a *App) SetWeather(strength int) {
	if a.deps.Recorder != nil {
		a.deps.Recorder.GetWeatherStrengthListener(strength)()
	}
}

func (a *App) RefreshActiveCharacter() {
	fyne.Do(a.refreshCharacters)
}

func (a *App) persist() {
	_ = config.Write(a.deps.Config)
}

func (a *App) showError(title, msg string) {
	dialog.ShowError(simpleError(title+": "+msg), a.win)
}

type simpleError string

func (e simpleError) Error() string { return string(e) }

func max(a, b int) int {
	if a > b {
		return a
	}

	return b
}

func (a *App) setLocked(locked bool) {
	for _, w := range a.lockables {
		if locked {
			w.Disable()
		} else {
			w.Enable()
		}
	}
}

type runnerItem struct {
	widget.BaseWidget
	check *widget.Check
	icon  *canvas.Image
	name  *widget.Label
	ship  *widget.Label
	fit   *widget.Label
}

func newRunnerItem() *runnerItem {
	icon := canvas.NewImageFromImage(image.NewRGBA(image.Rect(0, 0, 24, 24)))
	icon.FillMode = canvas.ImageFillContain
	icon.ScaleMode = canvas.ImageScalePixels
	icon.SetMinSize(fyne.NewSize(24, 24))

	r := &runnerItem{
		check: widget.NewCheck("", nil),
		icon:  icon,
		name:  widget.NewLabel("pilot"),
		ship:  widget.NewLabel("ship"),
		fit:   widget.NewLabel("fit"),
	}
	r.ExtendBaseWidget(r)

	return r
}

func (r *runnerItem) CreateRenderer() fyne.WidgetRenderer {
	return widget.NewSimpleRenderer(container.NewBorder(nil, nil,
		container.NewHBox(r.check, r.icon), nil,
		container.NewHBox(r.name, r.ship, r.fit)))
}

func (r *runnerItem) setIcon(im image.Image) {
	if im == nil {
		im = image.NewUniform(color.Transparent)
	}
	r.icon.Image = im
	r.icon.Refresh()
}
