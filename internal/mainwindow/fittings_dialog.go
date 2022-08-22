package mainwindow

import (
	"fmt"
	"log"

	"github.com/lxn/walk"
	. "github.com/lxn/walk/declarative" // nolint:stylecheck,revive // we needs side effects
)

func RunManageFittingsDialog(owner walk.Form, conf interface{}) (int, error) {
	var (
		dlg                *walk.Dialog
		db                 *walk.DataBinder
		acceptPB, cancelPB *walk.PushButton
	)

	return Dialog{
		AssignTo:      &dlg,
		Title:         "Manage fittings",
		DefaultButton: &acceptPB,
		CancelButton:  &cancelPB,
		DataBinder: DataBinder{
			AssignTo:   &db,
			DataSource: conf,
			OnSubmitted: func() {
				fmt.Printf("config submitted")
			},
		},
		MinSize: Size{Width: 500, Height: 300},
		Layout:  VBox{},
		Children: []Widget{
			Composite{
				Layout: HBox{},
				Children: []Widget{
					HSpacer{},
					PushButton{
						AssignTo: &acceptPB,
						Text:     "OK",
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
