package mainwindow

import (
	"context"
	"fmt"
	"log"
	"sort"
	"time"

	"github.com/lxn/walk"
	. "github.com/lxn/walk/declarative" //nolint:stylecheck,revive // we needs side effects
	"github.com/shivas/abyss-blackbox/internal/fittings"
	"golang.org/x/exp/slog"

	fittingspb "github.com/shivas/abyss-blackbox/internal/fittings/pb"
)

func RunManageFittingsDialog(owner walk.Form, conf interface{}, fm *fittings.FittingsManager, pilots []string) (int, error) {
	var (
		dlg                      *walk.Dialog
		db                       *walk.DataBinder
		acceptPB, cancelPB       *walk.PushButton
		fittingsView             *walk.TableView
		importClipboardButton    *walk.PushButton
		importAbyssTrackerButton *walk.PushButton
		clearAssignmentsButton   *walk.PushButton
		importGroupBox           *walk.GroupBox
		assignMenu               *walk.Menu
		unassignMenu             *walk.Menu
		fittingsModel            *fittings.FittingsModel
	)

	go func() {
		time.Sleep(100 * time.Millisecond)

		providers := fm.AvailableProviders(context.Background())
		slog.Debug("available providers", slog.Any("providers", providers))

		tracker := providers["abysstracker.com"]
		if importAbyssTrackerButton != nil {
			importAbyssTrackerButton.SetEnabled(tracker.Available)

			if tracker.Err != nil {
				_ = importAbyssTrackerButton.SetToolTipText(tracker.Err.Error())
			}

			originalButtonText := importAbyssTrackerButton.Text()

			if tracker.Available {
				importAbyssTrackerButton.Clicked().Attach(func() {
					func(original string) {
						importAbyssTrackerButton.SetEnabled(false)
						defer func(text string) {
							fittingsModel.PublishRowsReset()
							importAbyssTrackerButton.SetEnabled(true)
							_ = importAbyssTrackerButton.SetText(text)
						}(original)

						err := fm.ImportFittings(context.Background(), "abysstracker.com", func(current, max int) {
							_ = importAbyssTrackerButton.SetText(fmt.Sprintf("importing %d of %d", current, max))
						})
						if err != nil {
							slog.Error("error importing fit", err)
						}
					}(originalButtonText)
				})
			}
		}
	}()

	pilotActions := make([]MenuItem, 0)
	pilotUnassignActions := make([]MenuItem, 0)

	sort.Strings(pilots)

	for _, name := range pilots {
		name := name

		pilotActions = append(pilotActions, Action{
			Text: name,
			OnTriggered: func() {
				if fittingsView.CurrentIndex() < 0 {
					return
				}

				fit := fm.GetByID(fittingsView.CurrentIndex())
				if fit == nil {
					return
				}

				fm.AssignFittingToCharacter(fit, name)
				slog.Info("Assigned fit to character", slog.String("fitting", fit.FittingName), slog.String("character", name))
			},
		})

		pilotUnassignActions = append(pilotUnassignActions, Action{
			Text: name,
			OnTriggered: func() {
				fm.AssignFittingToCharacter(nil, name)
				slog.Info("Unassigned fit for character", slog.String("character", name))
			},
		})
	}

	fittingsModel = fm.Model()

	return Dialog{
		AssignTo:      &dlg,
		Title:         "Manage fittings",
		DefaultButton: &acceptPB,
		CancelButton:  &cancelPB,
		DataBinder: DataBinder{
			AssignTo:   &db,
			DataSource: conf,
			OnSubmitted: func() {
				_ = fm.PersistCache()
				slog.Info("fittings saved")
			},
		},
		MinSize: Size{Width: 600, Height: 400},
		Layout:  VBox{},
		Children: []Widget{
			Composite{
				Layout: HBox{},
				Children: []Widget{
					TableView{
						AssignTo:                    &fittingsView,
						Model:                       fittingsModel,
						AlternatingRowBG:            false,
						CheckBoxes:                  false,
						ColumnsOrderable:            true,
						MultiSelection:              false,
						SelectionHiddenWithoutFocus: true,
						AlwaysConsumeSpace:          true,
						MinSize:                     Size{Height: 200, Width: 500},
						LastColumnStretched:         true,
						CustomRowHeight:             34,
						Columns: []TableViewColumn{
							{Title: "Fitting name"},
							{Title: "Ship"},
							{Title: "FFH"},
							{Title: "Source"},
							{Title: "Source ID"},
						},
						ContextMenuItems: []MenuItem{
							Menu{
								AssignTo: &assignMenu,
								Text:     "Assign to pilot",
								Items:    pilotActions,
							},
							Menu{
								AssignTo: &unassignMenu,
								Text:     "Unassign pilot fitting",
								Items:    pilotUnassignActions,
							},
							Separator{},
							Action{
								Text: "Delete fitting",
								OnTriggered: func() {
									if fittingsView.CurrentIndex() < 0 {
										return
									}

									fit := fm.GetByID(fittingsView.CurrentIndex())
									if fit == nil {
										return
									}

									fm.DeleteFitting(fittingsView.CurrentIndex())
									fittingsModel.PublishRowsReset()
								},
							},
						},
					},
					Composite{
						Layout:    VBox{},
						Alignment: AlignHNearVNear,
						Children: []Widget{
							GroupBox{
								AssignTo: &importGroupBox,
								Title:    "Import fittings",
								Layout:   VBox{},
								Children: []Widget{
									PushButton{
										AssignTo: &importClipboardButton,
										Text:     "Import EFT from clipboard",
										OnClicked: func() {
											eft, err := walk.Clipboard().Text()
											if err != nil {
												return
											}

											_, _, err = fm.AddFitting(&fittingspb.FittingRecord{Source: "manual", EFT: eft})
											if err != nil {
												walk.MsgBox(dlg, "Error importing EFT from clipboard", err.Error(), walk.MsgBoxIconWarning)
											}

											fittingsModel.PublishRowsReset()
										},
									},
									PushButton{
										AssignTo: &importAbyssTrackerButton,
										Text:     "Import my fits from abysstracker.com",
										Enabled:  false,
									},
								},
							},
							PushButton{
								AssignTo: &clearAssignmentsButton,
								Text:     "Clear all fit assignments",
								OnClicked: func() {
									fm.ClearAssignments()
								},
							},
							TextLabel{
								Text: "To perform actions on fits, please use right-click!",
							},
						},
					},
				},
			},
			Composite{
				Layout: HBox{},
				Children: []Widget{
					PushButton{
						AssignTo: &acceptPB,
						Text:     "Save fittings",
						OnClicked: func() {
							if err := db.Submit(); err != nil {
								log.Print(err)
								return
							}

							dlg.Accept()
						},
					},
					PushButton{
						AssignTo:  &cancelPB,
						Text:      "Cancel",
						OnClicked: func() { dlg.Cancel() },
					},
				},
			},
		},
	}.Run(owner)
}
