package ui

import (
	"log/slog"
	"slices"
	"sort"
	"strconv"
	"syscall"
	"time"

	"image/color"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/layout"
	"fyne.io/fyne/v2/widget"

	"github.com/shivas/abyss-blackbox/internal/window"
)

func (a *App) build() {
	slog.Info("building main window")
	a.collectWindows()
	a.refreshRunners()

	a.win.SetMainMenu(a.makeMainMenu())
	slog.Info("setting window content")
	split := container.NewHSplit(a.makeLeftPane(), a.makePreviewPane())
	split.SetOffset(0.42)
	a.win.SetContent(container.NewBorder(a.makeToolbar(), nil, nil, nil, split))
	slog.Info("window content set")
}

func (a *App) collectWindows() {
	a.windowTitles = nil
	a.windowByTitle = make(map[string]syscall.Handle)
	if a.deps.Windows == nil {
		return
	}

	found := a.deps.Windows.GetEVEClientWindows(window.SupportedWindowsFilter)
	if len(found) == 0 {
		found = a.deps.Windows.GetEVEClientWindows(nil)
		if len(found) > 0 {
			slog.Warn("no English-filtered EVE windows, listing all detected clients", slog.Int("count", len(found)))
		}
	}

	for handle, title := range found {
		a.windowTitles = append(a.windowTitles, title)
		a.windowByTitle[title] = handle
	}
	sort.Strings(a.windowTitles)
}

func (a *App) makeMainMenu() *fyne.MainMenu {
	settings := fyne.NewMenuItem("Settings", a.showSettings)
	about := fyne.NewMenuItem("About", a.showAbout)

	return fyne.NewMainMenu(
		fyne.NewMenu("App", settings, about),
	)
}

func (a *App) makeToolbar() fyne.CanvasObject {
	a.charSelect = widget.NewSelect([]string{}, func(name string) {
		for _, ch := range a.deps.Chars.Characters() {
			if ch.CharacterName != name {
				continue
			}
			if active := a.deps.Chars.ActiveCharacter(); active != nil && active.CharacterID == ch.CharacterID {
				return
			}
			_ = a.deps.Chars.SetActiveCharacter(ch.CharacterID)
			return
		}
	})
	a.charSelect.PlaceHolder = "Switch character"

	a.addCharBtn = widget.NewButton("Add character", func() {
		a.addCharBtn.Disable()
		go a.deps.Chars.EventHandlerCharAdd()
	})

	removeChar := widget.NewButton("Remove", func() {
		if a.charSelect.Selected == "" {
			return
		}
		for _, ch := range a.deps.Chars.Characters() {
			if ch.CharacterName == a.charSelect.Selected {
				a.deps.Chars.RemoveCharacter(ch.CharacterID)
				return
			}
		}
	})

	a.presetSelect = widget.NewSelect(nil, func(name string) {
		p, ok := a.deps.Config.Presets[name]
		if !ok {
			return
		}
		a.deps.Config.X, a.deps.Config.Y, a.deps.Config.H = p.X, p.Y, p.H
		a.xEntry.SetText(strconv.Itoa(p.X))
		a.yEntry.SetText(strconv.Itoa(p.Y))
		a.hEntry.SetText(strconv.Itoa(p.H))
		a.preview.SetSize(float32(a.captureWidth), float32(max(p.H, 200)))
		a.persist()
	})
	a.presetSelect.PlaceHolder = "Presets"

	savePreset := widget.NewButton("Save preset", a.showNewPreset)
	delPreset := widget.NewButton("Delete preset", func() {
		name := a.presetSelect.Selected
		if name == "" {
			return
		}
		delete(a.deps.Config.Presets, name)
		a.persist()
		a.refreshPresets()
	})

	overlayBtn := widget.NewButton("Overlay", a.ToggleOverlay)

	a.refreshCharacters()
	a.refreshPresets()

	return container.NewHBox(
		widget.NewLabel("Character:"),
		a.charSelect,
		a.addCharBtn,
		removeChar,
		layout.NewSpacer(),
		a.presetSelect,
		savePreset,
		delPreset,
		overlayBtn,
	)
}

func (a *App) makeLeftPane() fyne.CanvasObject {
	cfg := a.deps.Config

	a.windowSelect = widget.NewSelect(a.windowTitles, func(title string) {
		cfg.EVEClientWindowTitle = title
		if handle, ok := a.windowByTitle[title]; ok && a.deps.Windows != nil {
			cfg.TestServer = a.deps.Windows.IsTestingServer(handle)
		}
		a.persist()
	})
	a.windowSelect.PlaceHolder = "No EVE client found"
	a.applyWindowSelection()
	refreshWindows := widget.NewButton("Refresh", a.refreshEVEWindows)

	a.filteredCheck = widget.NewCheck("show filtered preview", func(v bool) {
		cfg.FilteredPreview = v
		a.persist()
	})
	a.filteredCheck.SetChecked(cfg.FilteredPreview)

	a.xEntry = newIntEntry(cfg.X, func(v int) {
		cfg.X = v
		a.persist()
	})
	a.yEntry = newIntEntry(cfg.Y, func(v int) {
		cfg.Y = v
		a.persist()
	})
	a.hEntry = newIntEntry(cfg.H, func(v int) {
		cfg.H = v
		a.preview.SetSize(float32(a.captureWidth), float32(max(v, 200)))
		a.persist()
	})

	scaleTitles := make([]string, 0, len(uiScales))
	for _, s := range uiScales {
		scaleTitles = append(scaleTitles, s.title)
	}
	a.scalingSelect = widget.NewSelect(scaleTitles, func(title string) {
		cfg.EVEClientUIScaling = title
		for _, s := range uiScales {
			if s.title == title {
				a.captureWidth = s.width
				a.preview.SetSize(float32(s.width), float32(max(cfg.H, 200)))
			}
		}
		a.persist()
	})
	a.scalingSelect.SetSelected(cfg.EVEClientUIScaling)

	captureForm := widget.NewForm(
		widget.NewFormItem("EVE window", container.NewBorder(nil, nil, nil, refreshWindows, a.windowSelect)),
		widget.NewFormItem("X", a.xEntry),
		widget.NewFormItem("Y", a.yEntry),
		widget.NewFormItem("Height", a.hEntry),
		widget.NewFormItem("UI scaling", a.scalingSelect),
	)

	shipIdx := cfg.AbyssShipType - 1
	a.shipSelect = newIconPicker(shipTypeOptions(), shipIdx, func(i int) {
		cfg.AbyssShipType = i + 1
		a.persist()
	})

	a.tierSelect = newIconPicker(tierOptions(), cfg.AbyssTier, func(i int) {
		cfg.AbyssTier = i
		a.persist()
	})

	weathers := weatherOptions()
	weatherIdx := 0
	for i, w := range weathers {
		if w.label == cfg.AbyssWeather {
			weatherIdx = i
			break
		}
	}
	a.weatherSelect = newIconPicker(weathers, weatherIdx, func(i int) {
		cfg.AbyssWeather = weathers[i].label
		a.persist()
	})

	overrideBox := widget.NewForm(
		widget.NewFormItem("Ship", a.shipSelect),
		widget.NewFormItem("Tier", a.tierSelect),
		widget.NewFormItem("Weather", a.weatherSelect),
	)
	a.overrideCheck = widget.NewCheck("Manual abyss type override", func(v bool) {
		cfg.AbyssTypeOverride = v
		if v {
			a.shipSelect.Enable()
			a.tierSelect.Enable()
			a.weatherSelect.Enable()
		} else {
			a.shipSelect.Disable()
			a.tierSelect.Disable()
			a.weatherSelect.Disable()
		}
		a.persist()
	})
	a.overrideCheck.SetChecked(cfg.AbyssTypeOverride)
	if !cfg.AbyssTypeOverride {
		a.shipSelect.Disable()
		a.tierSelect.Disable()
		a.weatherSelect.Disable()
	}

	a.runnerList = widget.NewList(
		func() int { return len(a.runners) },
		func() fyne.CanvasObject {
			return newRunnerItem()
		},
		func(id widget.ListItemID, obj fyne.CanvasObject) {
			if id < 0 || id >= len(a.runners) {
				return
			}
			row := a.runners[id]
			item := obj.(*runnerItem)
			item.check.OnChanged = nil
			item.check.SetChecked(row.checked)
			item.check.OnChanged = func(v bool) {
				if id >= 0 && id < len(a.runners) {
					a.runners[id].checked = v
				}
			}
			item.setIcon(row.icon)
			item.name.SetText(row.name)
			if row.ship != "" {
				item.ship.SetText(row.ship)
			} else {
				item.ship.SetText("—")
			}
			if row.fit != "" {
				item.fit.SetText(row.fit)
			} else {
				item.fit.SetText("no fit")
			}
		},
	)
	runnerPad := canvas.NewRectangle(color.Transparent)
	runnerPad.SetMinSize(fyne.NewSize(280, 160))
	runnerBox := container.NewStack(runnerPad, a.runnerList)

	a.manageFitsBtn = widget.NewButton("Manage fittings", a.showFittings)

	a.autoUpload = widget.NewCheck("Upload file after recording complete", func(v bool) {
		cfg.AutoUpload = v
		a.persist()
	})
	a.autoUpload.SetChecked(cfg.AutoUpload)
	a.autoUpload.Disable()

	a.recordBtn = widget.NewButton("Start recording", a.toggleRecording)
	a.recordBtn.Importance = widget.HighImportance

	a.lockables = []fyne.Disableable{
		a.windowSelect, a.xEntry, a.yEntry, a.hEntry, a.scalingSelect,
		a.overrideCheck, a.shipSelect, a.tierSelect, a.weatherSelect,
		a.manageFitsBtn, a.filteredCheck,
	}

	top := container.NewVBox(
		widget.NewCard("Capture region", "", container.NewVBox(captureForm, a.filteredCheck)),
		widget.NewCard("Abyss type", "", container.NewVBox(a.overrideCheck, overrideBox)),
	)
	bottom := container.NewVBox(a.manageFitsBtn, a.autoUpload, a.recordBtn)

	return container.NewBorder(top, bottom, nil, nil,
		widget.NewCard("Combat log characters", "", runnerBox),
	)
}

func (a *App) makePreviewPane() fyne.CanvasObject {
	return widget.NewCard("Captured region", "", container.NewScroll(container.NewCenter(a.preview.Canvas())))
}

func (a *App) refreshCharacters() {
	if a.charSelect == nil || a.deps.Chars == nil {
		return
	}

	chars := a.deps.Chars.Characters()
	names := make([]string, 0, len(chars))
	for _, ch := range chars {
		names = append(names, ch.CharacterName)
	}
	sort.Strings(names)
	a.charSelect.Options = names
	a.charSelect.Refresh()

	if a.addCharBtn != nil {
		a.addCharBtn.Enable()
	}

	active := a.deps.Chars.ActiveCharacter()
	if active != nil && active.CharacterID > 0 {
		if a.charSelect.Selected != active.CharacterName {
			a.charSelect.SetSelected(active.CharacterName)
		}
		a.win.SetTitle("Abyssal.Space Blackbox Recorder - " + active.CharacterName)
		if a.autoUpload != nil {
			a.autoUpload.Enable()
		}
	}
}

func (a *App) applyWindowSelection() {
	if a.windowSelect == nil {
		return
	}

	cfg := a.deps.Config
	switch {
	case cfg.EVEClientWindowTitle != "" && slices.Contains(a.windowTitles, cfg.EVEClientWindowTitle):
		a.windowSelect.SetSelected(cfg.EVEClientWindowTitle)
	case len(a.windowTitles) > 0:
		a.windowSelect.SetSelected(a.windowTitles[0])
		cfg.EVEClientWindowTitle = a.windowTitles[0]
	default:
		a.windowSelect.ClearSelected()
	}
}

func (a *App) refreshEVEWindows() {
	if a.deps.Windows != nil {
		if err := a.deps.Windows.Refresh(); err != nil {
			slog.Info("eve window refresh", slog.Any("err", err))
		}
	}

	a.collectWindows()
	if a.windowSelect != nil {
		a.windowSelect.Options = a.windowTitles
		a.windowSelect.Refresh()
		a.applyWindowSelection()
	}
}

func (a *App) refreshPresets() {
	if a.presetSelect == nil {
		return
	}

	names := make([]string, 0, len(a.deps.Config.Presets))
	for name := range a.deps.Config.Presets {
		names = append(names, name)
	}
	sort.Strings(names)
	a.presetSelect.Options = names
	a.presetSelect.Refresh()
}

func (a *App) refreshRunners() {
	if a.deps.CombatLog == nil {
		return
	}

	logFiles, err := a.deps.CombatLog.GetLogFiles(time.Now(), 24*time.Hour)
	if err != nil {
		return
	}

	chars := a.deps.CombatLog.MapCharactersToFiles(logFiles)
	a.runners = a.runners[:0]

	for name := range chars {
		row := &runnerRow{name: name}
		if a.deps.Fittings != nil {
			if fit := a.deps.Fittings.GetFittingForPilot(name); fit != nil {
				row.fit = fit.FittingName
				row.ship = fit.ShipName
				row.icon = fit.ShipPicture()
			}
		}
		a.runners = append(a.runners, row)
	}

	sort.Slice(a.runners, func(i, j int) bool {
		return a.runners[i].name < a.runners[j].name
	})

	if a.runnerList != nil {
		a.runnerList.Refresh()
	}
}

func (a *App) checkedCharacters() []string {
	var out []string
	for _, r := range a.runners {
		if r.checked {
			out = append(out, r.name)
		}
	}

	return out
}

func newIntEntry(val int, onChange func(int)) *widget.Entry {
	e := widget.NewEntry()
	e.SetText(strconv.Itoa(val))
	e.OnChanged = func(s string) {
		n, err := strconv.Atoi(s)
		if err != nil {
			return
		}
		onChange(n)
	}

	return e
}
