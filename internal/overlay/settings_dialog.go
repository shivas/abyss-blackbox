package overlay

import (
	"log"

	"github.com/lxn/walk"
	. "github.com/lxn/walk/declarative" // nolint:stylecheck,revive // we needs side effects
)

type ShortcutSetter interface {
	SetRecorderShortcut(int, walk.Shortcut)
}

func RunSettingsDialog(owner walk.Form, conf interface{}, onSettingsSubmit func(c *OverlayConfig), invalidateFunc func()) (int, error) {
	var (
		dlg                         *walk.Dialog
		db                          *walk.DataBinder
		applyPB, acceptPB, cancelPB *walk.PushButton
		fontFamilyEdit              *walk.LineEdit
		fontSizeEdit                *walk.NumberEdit
		spacingEdit                 *walk.NumberEdit
		backgroundColorEdit         *walk.LineEdit
	)

	return Dialog{
		AssignTo:      &dlg,
		Title:         "Overlay settings",
		DefaultButton: &acceptPB,
		CancelButton:  &cancelPB,
		DataBinder: DataBinder{
			AssignTo:   &db,
			DataSource: conf,
			OnSubmitted: func() {
				if c, ok := conf.(*OverlayConfig); ok {
					onSettingsSubmit(c)
				}
			},
		},
		// MinSize: Size{Width: 500, Height: 300},
		Layout: VBox{},
		Children: []Widget{
			GroupBox{
				Title:     "Font settings:",
				Layout:    VBox{SpacingZero: true},
				Alignment: AlignHNearVNear,
				Children: []Widget{
					Composite{
						Layout: HBox{},
						Children: []Widget{
							TextLabel{
								Text: "Font family:",
							},
							LineEdit{
								Text:     Bind("FontFamily"),
								AssignTo: &fontFamilyEdit,
							},
							HSpacer{},
						},
					},
					Composite{
						Layout: HBox{},
						Children: []Widget{
							TextLabel{
								Text: "Font size:",
							},
							NumberEdit{
								Value:    Bind("FontSize"),
								AssignTo: &fontSizeEdit,
							},
							HSpacer{},
						},
					},
					Composite{
						Layout: HBox{},
						Children: []Widget{
							TextLabel{
								Text: "Spacing (between lines):",
							},
							NumberEdit{
								Value:    Bind("Spacing"),
								AssignTo: &spacingEdit,
							},
							HSpacer{},
						},
					},
					Composite{
						Layout: HBox{},
						Children: []Widget{
							TextLabel{
								Text: "Background color (hex):",
							},
							LineEdit{
								Text:     Bind("BackgroundColorText"),
								AssignTo: &backgroundColorEdit,
							},
							HSpacer{},
						},
					},
				},
			},
			Composite{
				Layout: HBox{},
				Children: []Widget{
					HSpacer{},
					PushButton{
						AssignTo: &applyPB,
						Text:     "Apply",
						OnClicked: func() {
							if err := db.Submit(); err != nil {
								log.Print(err)
								return
							}
							invalidateFunc()
						},
					},

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
