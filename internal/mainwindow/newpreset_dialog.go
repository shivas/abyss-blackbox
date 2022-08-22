package mainwindow

import (
	"log"

	"github.com/shivas/abyss-blackbox/internal/config"

	"github.com/lxn/walk"
	. "github.com/lxn/walk/declarative" // nolint:stylecheck,revive // we needs side effects
)

func RunNewPresetDialog(owner walk.Form, presetValue config.Preset, c *config.CaptureConfig) (int, error) {
	var (
		dlg                *walk.Dialog
		db                 *walk.DataBinder
		acceptPB, cancelPB *walk.PushButton
		PresetNameEdit     *walk.LineEdit
	)

	data := map[string]interface{}{"Name": ""}

	return Dialog{
		AssignTo:      &dlg,
		Title:         "New preset",
		DefaultButton: &acceptPB,
		CancelButton:  &cancelPB,
		DataBinder: DataBinder{
			AssignTo:   &db,
			DataSource: data,
			OnSubmitted: func() {
				log.Printf("new preset saving: %#v with values: %v", data, presetValue)
				if c.Presets == nil {
					c.Presets = make(map[string]config.Preset)
				}

				c.Presets[data["Name"].(string)] = presetValue
			},
		},
		Layout: VBox{},
		Children: []Widget{
			TextLabel{
				Text: "Preset name:",
			},
			LineEdit{
				Text:     Bind("Name"),
				AssignTo: &PresetNameEdit,
			},
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
