package ui

import (
	"net/url"
	"strconv"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/storage"
	"fyne.io/fyne/v2/widget"

	"github.com/shivas/abyss-blackbox/internal/config"
	"github.com/shivas/abyss-blackbox/internal/overlay"
	"github.com/shivas/abyss-blackbox/internal/version"
)

func (a *App) showAbout() {
	logo := canvas.NewImageFromResource(fyne.NewStaticResource("triglogo.png", logoPNG))
	logo.SetMinSize(fyne.NewSize(72, 72))
	logo.FillMode = canvas.ImageFillContain

	site, _ := url.Parse("https://abyssal.space")
	releases, _ := url.Parse("https://github.com/shivas/abyss-blackbox/releases")
	discord, _ := url.Parse("https://discord.gg/qFcywP6WUK")

	content := container.NewHBox(
		logo,
		container.NewVBox(
			widget.NewLabel("Version: "+version.RecorderVersion),
			widget.NewLabel("Go: "+version.GoVersion),
			widget.NewHyperlink("https://abyssal.space", site),
			widget.NewHyperlink("Recorder releases", releases),
			widget.NewHyperlink("Abyssal Lurkers Discord", discord),
		),
	)

	d := dialog.NewCustom("About", "OK", content, a.win)
	d.Show()
}

func (a *App) showNewPreset() {
	name := widget.NewEntry()
	name.SetPlaceHolder("Preset name")

	d := dialog.NewForm("New preset", "OK", "Cancel", []*widget.FormItem{
		widget.NewFormItem("Name", name),
	}, func(ok bool) {
		if !ok || name.Text == "" {
			return
		}
		if a.deps.Config.Presets == nil {
			a.deps.Config.Presets = make(map[string]config.Preset)
		}
		a.deps.Config.Presets[name.Text] = config.Preset{
			X: a.deps.Config.X,
			Y: a.deps.Config.Y,
			H: a.deps.Config.H,
		}
		a.persist()
		a.refreshPresets()
		a.presetSelect.SetSelected(name.Text)
	}, a.win)
	d.Show()
}

func (a *App) showSettings() {
	cfg := a.deps.Config

	folder := widget.NewEntry()
	folder.SetText(cfg.EVEGameLogsFolder)
	folder.Disable()

	choose := widget.NewButton("Browse", func() {
		d := dialog.NewFolderOpen(func(uri fyne.ListableURI, err error) {
			if err != nil || uri == nil {
				return
			}
			cfg.EVEGameLogsFolder = uri.Path()
			folder.SetText(uri.Path())
			if a.deps.CombatLog != nil {
				a.deps.CombatLog.SetLogFolder(uri.Path())
			}
			a.refreshRunners()
			if a.runnerList != nil {
				a.runnerList.Refresh()
			}
		}, a.win)
		if u, err := storage.ListerForURI(storage.NewFileURI(cfg.EVEGameLogsFolder)); err == nil {
			d.SetLocation(u)
		}
		d.Show()
	})

	loot := widget.NewEntry()
	loot.SetText(cfg.LootRecordDiscriminator)
	loot.SetPlaceHolder("e.g. Quafe")
	loot.OnChanged = func(s string) {
		cfg.LootRecordDiscriminator = s
	}

	recorder := newShortcutRecorder(a.deps.StartShortcutCapture, a.deps.StopShortcutCapture)
	form := widget.NewForm(
		widget.NewFormItem("Combat logs", container.NewBorder(nil, nil, nil, choose, folder)),
		widget.NewFormItem("Loot item", loot),
		widget.NewFormItem("Start/Stop", recorder.add(cfg.RecorderShortcut, func(s config.Shortcut) {
			cfg.SetRecorderShortcut(config.HotkeyRecoder, s)
		})),
		widget.NewFormItem("Weather 30%", recorder.add(cfg.Weather30Shortcut, func(s config.Shortcut) {
			cfg.SetRecorderShortcut(config.HotkeyWeather30, s)
		})),
		widget.NewFormItem("Weather 50%", recorder.add(cfg.Weather50Shortcut, func(s config.Shortcut) {
			cfg.SetRecorderShortcut(config.HotkeyWeather50, s)
		})),
		widget.NewFormItem("Weather 70%", recorder.add(cfg.Weather70Shortcut, func(s config.Shortcut) {
			cfg.SetRecorderShortcut(config.HotkeyWeather70, s)
		})),
		widget.NewFormItem("Overlay", recorder.add(cfg.OverlayShortcut, func(s config.Shortcut) {
			cfg.SetRecorderShortcut(config.Overlay, s)
		})),
	)

	suppress := widget.NewCheck("Suppress notifications", func(v bool) {
		cfg.SuppressNotifications = v
	})
	suppress.SetChecked(cfg.SuppressNotifications)

	d := dialog.NewCustomConfirm("Settings", "OK", "Cancel", container.NewPadded(container.NewVBox(form, suppress)), func(ok bool) {
		recorder.stop()
		if !ok {
			return
		}
		a.persist()
		if a.deps.RebindHotkeys != nil {
			a.deps.RebindHotkeys()
		}
	}, a.win)
	d.Resize(fyne.NewSize(560, 420))
	d.SetOnClosed(func() {
		recorder.stop()
	})
	d.Show()
}

func (a *App) showOverlaySettings() {
	oc := a.deps.Overlay.Config()
	if oc == nil {
		return
	}

	family := widget.NewEntry()
	family.SetText(oc.FontFamily)
	size := newIntEntry(oc.FontSize, func(v int) { oc.FontSize = v })
	spacing := newIntEntry(oc.Spacing, func(v int) { oc.Spacing = v })
	bg := widget.NewEntry()
	bg.SetText(oc.BackgroundColorText)

	form := widget.NewForm(
		widget.NewFormItem("Font family", family),
		widget.NewFormItem("Font size", size),
		widget.NewFormItem("Spacing", spacing),
		widget.NewFormItem("Background color", bg),
	)

	apply := func() {
		oc.FontFamily = family.Text
		if n, err := strconv.Atoi(size.Text); err == nil {
			oc.FontSize = n
		}
		if n, err := strconv.Atoi(spacing.Text); err == nil {
			oc.Spacing = n
		}
		oc.BackgroundColorText = bg.Text
		oc.BackgroundColor = overlay.ParseColor(bg.Text)
		a.deps.Overlay.ApplyConfig(oc)
		overlay.AssignToConfig(a.deps.Config, oc)
		a.persist()
	}

	d := dialog.NewCustomConfirm("Overlay settings", "OK", "Cancel", container.NewPadded(form), func(ok bool) {
		if ok {
			apply()
		}
	}, a.win)
	d.Resize(fyne.NewSize(420, 280))
	d.Show()
}

type shortcutRecorder struct {
	start  func(func(config.Shortcut))
	stopFn func()
	active *shortcutRow
}

type shortcutRow struct {
	rec     *shortcutRecorder
	value   *widget.Entry
	btn     *widget.Button
	current config.Shortcut
	onSave  func(config.Shortcut)
}

func newShortcutRecorder(start func(func(config.Shortcut)), stop func()) *shortcutRecorder {
	return &shortcutRecorder{start: start, stopFn: stop}
}

func (r *shortcutRecorder) add(initial config.Shortcut, onSave func(config.Shortcut)) fyne.CanvasObject {
	row := &shortcutRow{
		rec:     r,
		value:   widget.NewEntry(),
		btn:     widget.NewButton("Record", nil),
		current: initial,
		onSave:  onSave,
	}
	row.value.SetText(initial.String())
	row.value.Disable()
	row.btn.OnTapped = row.toggle
	return container.NewBorder(nil, nil, nil, row.btn, row.value)
}

func (row *shortcutRow) toggle() {
	if row.rec.active == row {
		row.rec.stop()
		return
	}
	row.rec.stop()
	row.rec.active = row
	row.btn.SetText("Done")
	row.value.SetText("Press a shortcut…")
	if row.rec.start == nil {
		return
	}
	row.rec.start(func(s config.Shortcut) {
		fyne.Do(func() {
			if row.rec.active != row {
				return
			}
			row.current = s
			row.value.SetText(s.String())
			if row.onSave != nil {
				row.onSave(s)
			}
		})
	})
}

func (r *shortcutRecorder) stop() {
	if r.active != nil {
		r.active.btn.SetText("Record")
		if r.active.current.Key != 0 {
			r.active.value.SetText(r.active.current.String())
		}
		r.active = nil
	}
	if r.stopFn != nil {
		r.stopFn()
	}
}
