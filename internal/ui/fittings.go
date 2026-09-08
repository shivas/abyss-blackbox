package ui

import (
	"context"
	"fmt"
	"image"
	"image/color"
	"sort"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/widget"

	fittingspb "github.com/shivas/abyss-blackbox/internal/fittings/pb"
)

func (a *App) showFittings() {
	fm := a.deps.Fittings
	if fm == nil {
		return
	}

	selected := 0
	list := widget.NewList(
		func() int {
			return fm.Count()
		},
		func() fyne.CanvasObject {
			return newFittingItem()
		},
		func(id widget.ListItemID, obj fyne.CanvasObject) {
			obj.(*fittingItem).set(fm.GetByID(id))
		},
	)
	list.OnSelected = func(id widget.ListItemID) {
		selected = id
	}

	pilots := a.pilotNames()
	assignSelect := widget.NewSelect(pilots, nil)
	assignSelect.PlaceHolder = "Pilot"

	assignBtn := widget.NewButton("Assign selected", func() {
		if assignSelect.Selected == "" {
			return
		}
		fit := fm.GetByID(selected)
		if fit == nil {
			return
		}
		fm.AssignFittingToCharacter(fit, assignSelect.Selected)
		a.refreshRunners()
		if a.runnerList != nil {
			a.runnerList.Refresh()
		}
	})

	unassignBtn := widget.NewButton("Unassign pilot", func() {
		if assignSelect.Selected == "" {
			return
		}
		fm.AssignFittingToCharacter(nil, assignSelect.Selected)
		a.refreshRunners()
		if a.runnerList != nil {
			a.runnerList.Refresh()
		}
	})

	deleteBtn := widget.NewButton("Delete fitting", func() {
		if fm.GetByID(selected) == nil {
			return
		}
		fm.DeleteFitting(selected)
		list.UnselectAll()
		list.Refresh()
	})

	importClip := widget.NewButton("Import EFT from clipboard", func() {
		eft := a.fyneApp.Clipboard().Content()
		if eft == "" {
			dialog.ShowInformation("Clipboard empty", "Copy an EFT fit first.", a.win)
			return
		}
		_, _, err := fm.AddFitting(&fittingspb.FittingRecord{Source: "manual", EFT: eft})
		if err != nil {
			dialog.ShowError(err, a.win)
			return
		}
		list.Refresh()
	})

	importTracker := widget.NewButton("Import from abysstracker.com", nil)
	importTracker.Disable()
	go func() {
		providers := fm.AvailableProviders(context.Background())
		tracker := providers["abysstracker.com"]
		fyne.Do(func() {
			if !tracker.Available {
				if tracker.Err != nil {
					importTracker.SetText("Tracker unavailable")
				}
				return
			}
			importTracker.Enable()
			importTracker.OnTapped = func() {
				importTracker.Disable()
				go func() {
					err := fm.ImportFittings(context.Background(), "abysstracker.com", func(current, max int) {
						fyne.Do(func() {
							importTracker.SetText(fmt.Sprintf("Importing %d of %d", current, max))
						})
					})
					fyne.Do(func() {
						if err != nil {
							dialog.ShowError(err, a.win)
						}
						importTracker.SetText("Import from abysstracker.com")
						importTracker.Enable()
						list.Refresh()
					})
				}()
			}
		})
	}()

	clear := widget.NewButton("Clear assignments", func() {
		fm.ClearAssignments()
		a.refreshRunners()
		if a.runnerList != nil {
			a.runnerList.Refresh()
		}
	})

	side := container.NewVBox(
		widget.NewLabelWithStyle("Import", fyne.TextAlignLeading, fyne.TextStyle{Bold: true}),
		importClip,
		importTracker,
		widget.NewSeparator(),
		widget.NewLabelWithStyle("Assign", fyne.TextAlignLeading, fyne.TextStyle{Bold: true}),
		assignSelect,
		assignBtn,
		unassignBtn,
		deleteBtn,
		clear,
	)
	sidePad := canvas.NewRectangle(color.Transparent)
	sidePad.SetMinSize(fyne.NewSize(240, 1))
	sideBox := container.NewStack(sidePad, container.NewPadded(side))

	body := container.NewPadded(container.NewBorder(nil, nil, nil, sideBox, list))

	d := dialog.NewCustomConfirm("Manage fittings", "Save fittings", "Cancel", body, func(ok bool) {
		if ok {
			_ = fm.PersistCache()
		}
	}, a.win)
	d.Resize(fyne.NewSize(780, 500))
	d.Show()
}

type fittingItem struct {
	widget.BaseWidget
	icon *canvas.Image
	name *widget.Label
	meta *widget.Label
}

func newFittingItem() *fittingItem {
	icon := canvas.NewImageFromImage(image.NewRGBA(image.Rect(0, 0, 24, 24)))
	icon.FillMode = canvas.ImageFillContain
	icon.ScaleMode = canvas.ImageScalePixels
	icon.SetMinSize(fyne.NewSize(24, 24))

	name := widget.NewLabel("fitting")
	name.Truncation = fyne.TextTruncateEllipsis
	meta := widget.NewLabel("meta")
	meta.Truncation = fyne.TextTruncateEllipsis

	f := &fittingItem{icon: icon, name: name, meta: meta}
	f.ExtendBaseWidget(f)

	return f
}

func (f *fittingItem) CreateRenderer() fyne.WidgetRenderer {
	return widget.NewSimpleRenderer(container.NewBorder(nil, nil, f.icon, nil, container.NewVBox(f.name, f.meta)))
}

func (f *fittingItem) set(fit *fittingspb.FittingRecord) {
	if fit == nil {
		f.name.SetText("")
		f.meta.SetText("")
		f.icon.Image = image.NewUniform(color.Transparent)
		f.icon.Refresh()
		return
	}

	f.name.SetText(fit.FittingName)
	f.meta.SetText(fmt.Sprintf("%s  ·  %s", fit.ShipName, fit.Source))
	f.icon.Image = fit.ShipPicture()
	f.icon.Refresh()
}

func (a *App) pilotNames() []string {
	names := make([]string, 0, len(a.runners))
	for _, r := range a.runners {
		names = append(names, r.name)
	}
	sort.Strings(names)

	return names
}
