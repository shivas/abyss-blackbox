package ui

import (
	"fmt"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/dialog"

	"github.com/shivas/abyss-blackbox/internal/app/recorder"
	"github.com/shivas/abyss-blackbox/internal/config"
	"github.com/shivas/abyss-blackbox/internal/overlay"
	"github.com/shivas/abyss-blackbox/internal/uploader"
)

func (a *App) toggleRecording() {
	rec := a.deps.Recorder
	if rec == nil {
		return
	}

	if rec.Status() == recorder.RecorderStopped {
		checked := a.checkedCharacters()
		if len(checked) == 0 {
			dialog.ShowInformation("No characters selected", "Please choose at least one character to capture combat log", a.win)
			return
		}
		if len(checked) > 3 {
			dialog.ShowInformation("Too many characters selected", "Please choose up to 3 characters to capture combat log", a.win)
			return
		}

		rec.Start(checked)

		if a.deps.Overlay != nil {
			a.deps.Overlay.ChangeProperty(overlay.Autoupload, fmt.Sprintf("Autoupload enabled: %t", a.autoUpload.Checked), &overlay.CyanColor)
			if a.deps.Config.AbyssTypeOverride {
				a.deps.Overlay.ChangeProperty(overlay.Override, fmt.Sprintf("Abyss type override: %s", tierOverrideToString(a.deps.Config)), &overlay.SecondaryColor)
			} else {
				a.deps.Overlay.ChangeProperty(overlay.Override, "Abyss type detection: heuristics", &overlay.CyanColor)
			}
		}

		a.setLocked(true)
		a.recordBtn.SetText("Stop recording")
		return
	}

	filename, err := rec.Stop(a.deps.Fittings)
	if err != nil {
		dialog.ShowError(err, a.win)
	}

	char := a.deps.Chars.ActiveCharacter()
	if a.autoUpload.Checked && char != nil && err == nil {
		go func(fn string) {
			uploadFile, uploadErr := uploader.Upload(a.deps.HTTPClient, fn)
			if uploadErr != nil {
				fyne.Do(func() { dialog.ShowError(uploadErr, a.win) })
				return
			}
			a.Notify("Record uploaded successfully", uploadFile)
			if a.deps.Overlay != nil {
				a.deps.Overlay.ChangeProperty(overlay.TODO, "Record uploaded successfully", &overlay.GreenColor)
			}
		}(filename)
	}

	a.setLocked(false)
	a.recordBtn.SetText("Start recording")

	if a.deps.Overlay != nil {
		a.deps.Overlay.ChangeProperty(overlay.Status, "Recorder on standby", &overlay.YellowColor)
		a.deps.Overlay.ChangeProperty(overlay.Weather, "", nil)
	}
}

func tierOverrideToString(c *config.CaptureConfig) string {
	var ship string
	switch c.AbyssShipType {
	case 1:
		ship = "Cruiser"
	case 2:
		ship = "Destroyers"
	case 3:
		ship = "Frigates"
	}

	return fmt.Sprintf("%s T%d %s", ship, c.AbyssTier, c.AbyssWeather)
}
