package mainwindow

import (
	"fmt"
	"log"
	"sort"

	"github.com/lxn/walk"
	. "github.com/lxn/walk/declarative" // nolint:stylecheck,revive // we needs side effects
	"github.com/shivas/abyss-blackbox/internal/fittings"
)

func RunManageFittingsDialog(owner walk.Form, conf interface{}, fm *fittings.FittingsManager, pilots []string) (int, error) {
	var (
		dlg                    *walk.Dialog
		db                     *walk.DataBinder
		acceptPB, cancelPB     *walk.PushButton
		fittingsView           *walk.TableView
		importClipboardButton  *walk.PushButton
		clearAssignmentsButton *walk.PushButton
		importGroupBox         *walk.GroupBox
		assignMenu             *walk.Menu
		fittingsModel          *fittings.FittingsModel
	)

	pilotActions := make([]MenuItem, 0)

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
				fmt.Printf("Assigned fit: %q to character: %q\n", fit.FittingName, name)
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
				fm.PersistCache()
				fmt.Printf("fittings saved")
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
						//						AlwaysConsumeSpace:          true,
						MinSize:             Size{Height: 200},
						LastColumnStretched: true,
						CustomRowHeight:     34,
						Columns: []TableViewColumn{
							{Title: "Fitting name"},
							{Title: "Ship"},
							{Title: "FFH"},
						},
						ContextMenuItems: []MenuItem{
							Menu{
								AssignTo: &assignMenu,
								Text:     "Assign to pilot",
								Items:    pilotActions,
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

											id, _, err := fm.AddFitting(&fittings.FittingRecord{Source: "manual", EFT: eft})
											if err != nil {
												log.Printf("error importing fitting %v\n", err)
											}

											fmt.Printf("added fitting: %d\n", id)
											fittingsView.SetSelectedIndexes([]int{id})
											fittingsModel.PublishRowsReset()

										},
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
