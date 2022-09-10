package mainwindow

import (
	"log"

	"github.com/shivas/abyss-blackbox/internal/config"

	"github.com/lxn/walk"
	. "github.com/lxn/walk/declarative" // nolint:stylecheck,revive // we needs side effects
)

type ShortcutSetter interface {
	SetRecorderShortcut(int, walk.Shortcut)
}

func RunSettingsDialog(owner walk.Form, conf interface{}, onSettingsSubmit func(c *config.CaptureConfig)) (int, error) {
	var (
		dlg                           *walk.Dialog
		db                            *walk.DataBinder
		acceptPB, cancelPB            *walk.PushButton
		RecorderShortcutEdit          *walk.LineEdit
		RecorderShortcutRecordButton  *walk.PushButton
		Weather30ShortcutEdit         *walk.LineEdit
		Weather30ShortcutRecordButton *walk.PushButton
		Weather50ShortcutEdit         *walk.LineEdit
		Weather50ShortcutRecordButton *walk.PushButton
		Weather70ShortcutEdit         *walk.LineEdit
		Weather70ShortcutRecordButton *walk.PushButton
		OverlayShortcutEdit           *walk.LineEdit
		OverlayShortcutRecordButton   *walk.PushButton
		LootRecordDiscriminatorEdit   *walk.LineEdit
		EVEGameLogsFolderLabel        *walk.TextLabel
		ChooseLogDirButton            *walk.PushButton
	)

	shortcutStringToKey := make(map[string]walk.Shortcut)

	return Dialog{
		AssignTo:      &dlg,
		Title:         "Settings",
		DefaultButton: &acceptPB,
		CancelButton:  &cancelPB,
		DataBinder: DataBinder{
			AssignTo:   &db,
			DataSource: conf,
			OnSubmitted: func() {
				if c, ok := conf.(*config.CaptureConfig); ok {
					if setter, ok2 := conf.(ShortcutSetter); ok2 {
						if key, exists := shortcutStringToKey[c.RecorderShortcutText]; exists {
							setter.SetRecorderShortcut(config.HotkeyRecoder, key)
						}
						if key, exists := shortcutStringToKey[c.Weather30ShortcutText]; exists {
							setter.SetRecorderShortcut(config.HotkeyWeather30, key)
						}
						if key, exists := shortcutStringToKey[c.Weather50ShortcutText]; exists {
							setter.SetRecorderShortcut(config.HotkeyWeather50, key)
						}
						if key, exists := shortcutStringToKey[c.Weather70ShortcutText]; exists {
							setter.SetRecorderShortcut(config.HotkeyWeather70, key)
						}
						if key, exists := shortcutStringToKey[c.OverlayShortcutText]; exists {
							setter.SetRecorderShortcut(config.Overlay, key)
						}
					}
					onSettingsSubmit(c)
				}
			},
		},
		MinSize: Size{Width: 500, Height: 300},
		Layout:  VBox{},
		Children: []Widget{
			GroupBox{
				Title:     "Combat log capture",
				Layout:    VBox{},
				Alignment: AlignHNearVNear,
				Children: []Widget{
					TextLabel{
						Text:     Bind("EVEGameLogsFolder"),
						AssignTo: &EVEGameLogsFolderLabel,
					},
					PushButton{
						Text:     "Choose",
						AssignTo: &ChooseLogDirButton,
						OnClicked: func() {
							fd := walk.FileDialog{}
							accepted, _ := fd.ShowBrowseFolder(owner)
							if accepted {
								_ = EVEGameLogsFolderLabel.SetText(fd.FilePath)
							}
						},
					},
				},
			},
			GroupBox{
				Title:     "Loot recording settings:",
				Layout:    VBox{},
				Alignment: AlignHNearVNear,
				Children: []Widget{
					TextLabel{
						Text: "Ship loot record discriminator item: (quantity in each ship should be different)",
					},
					LineEdit{
						Text:     Bind("LootRecordDiscriminator"),
						AssignTo: &LootRecordDiscriminatorEdit,
					},
					TextLabel{
						Text: "Only used when multiboxing destroyer or frigate runs.",
					},
				},
			},
			GroupBox{
				Title:     "Shortcut configuration:",
				Layout:    VBox{},
				Alignment: AlignHNearVNear,
				Children: []Widget{
					Composite{
						Layout:    HBox{},
						Alignment: AlignHNearVNear,
						Children: []Widget{
							TextLabel{
								Text: "Start/Stop shortcut",
							},
							LineEdit{
								Text:     Bind("RecorderShortcutText"),
								AssignTo: &RecorderShortcutEdit,
								OnKeyPress: func(key walk.Key) {
									shortcut := walk.Shortcut{Modifiers: walk.ModifiersDown(), Key: key}
									_ = RecorderShortcutEdit.SetText(shortcut.String())
									shortcutStringToKey[shortcut.String()] = shortcut
								},
								Enabled:  false,
								ReadOnly: true,
							},
							PushButton{
								AssignTo: &RecorderShortcutRecordButton,
								MinSize:  Size{Height: 20},
								Text:     "Record shortcut",
								OnClicked: func() {
									if !RecorderShortcutEdit.Enabled() { // start recording
										RecorderShortcutEdit.SetEnabled(true)
										_ = RecorderShortcutEdit.SetFocus()
										_ = RecorderShortcutRecordButton.SetText("Save")
									} else { // persist new shortcut and rebind
										RecorderShortcutEdit.SetEnabled(false)
										_ = RecorderShortcutRecordButton.SetText("Record shortcut")
									}
								},
							},
						},
					},
					Composite{
						Layout:    HBox{},
						Alignment: AlignHNearVNear,
						Children: []Widget{
							TextLabel{
								Text: "Weather strength 30%",
							},
							LineEdit{
								Text:     Bind("Weather30ShortcutText"),
								AssignTo: &Weather30ShortcutEdit,
								OnKeyPress: func(key walk.Key) {
									shortcut := walk.Shortcut{Modifiers: walk.ModifiersDown(), Key: key}
									_ = Weather30ShortcutEdit.SetText(shortcut.String())
									shortcutStringToKey[shortcut.String()] = shortcut
								},
								Enabled:  false,
								ReadOnly: true,
							},
							PushButton{
								AssignTo: &Weather30ShortcutRecordButton,
								MinSize:  Size{Height: 20},
								Text:     "Record shortcut",
								OnClicked: func() {
									if !Weather30ShortcutEdit.Enabled() { // start recording
										Weather30ShortcutEdit.SetEnabled(true)
										_ = Weather30ShortcutEdit.SetFocus()
										_ = Weather30ShortcutRecordButton.SetText("Save")
									} else { // persist new shortcut and rebind
										Weather30ShortcutEdit.SetEnabled(false)
										_ = Weather30ShortcutRecordButton.SetText("Record shortcut")
									}
								},
							},
						},
					},
					Composite{
						Layout:    HBox{},
						Alignment: AlignHNearVNear,
						Children: []Widget{
							TextLabel{
								Text: "Weather strength 50%",
							},
							LineEdit{
								Text:     Bind("Weather50ShortcutText"),
								AssignTo: &Weather50ShortcutEdit,
								OnKeyPress: func(key walk.Key) {
									shortcut := walk.Shortcut{Modifiers: walk.ModifiersDown(), Key: key}
									_ = Weather50ShortcutEdit.SetText(shortcut.String())
									shortcutStringToKey[shortcut.String()] = shortcut
								},
								Enabled:  false,
								ReadOnly: true,
							},
							PushButton{
								AssignTo: &Weather50ShortcutRecordButton,
								MinSize:  Size{Height: 20},
								Text:     "Record shortcut",
								OnClicked: func() {
									if !Weather50ShortcutEdit.Enabled() { // start recording
										Weather50ShortcutEdit.SetEnabled(true)
										_ = Weather50ShortcutEdit.SetFocus()
										_ = Weather50ShortcutRecordButton.SetText("Save")
									} else { // persist new shortcut and rebind
										Weather50ShortcutEdit.SetEnabled(false)
										_ = Weather50ShortcutRecordButton.SetText("Record shortcut")
									}
								},
							},
						},
					},
					Composite{
						Layout:    HBox{},
						Alignment: AlignHNearVNear,
						Children: []Widget{
							TextLabel{
								Text: "Weather strength 70%",
							},
							LineEdit{
								Text:     Bind("Weather70ShortcutText"),
								AssignTo: &Weather70ShortcutEdit,
								OnKeyPress: func(key walk.Key) {
									shortcut := walk.Shortcut{Modifiers: walk.ModifiersDown(), Key: key}
									_ = Weather70ShortcutEdit.SetText(shortcut.String())
									shortcutStringToKey[shortcut.String()] = shortcut
								},
								Enabled:  false,
								ReadOnly: true,
							},
							PushButton{
								AssignTo: &Weather70ShortcutRecordButton,
								MinSize:  Size{Height: 20},
								Text:     "Record shortcut",
								OnClicked: func() {
									if !Weather70ShortcutEdit.Enabled() { // start recording
										Weather70ShortcutEdit.SetEnabled(true)
										_ = Weather70ShortcutEdit.SetFocus()
										_ = Weather70ShortcutRecordButton.SetText("Save")
									} else { // persist new shortcut and rebind
										Weather70ShortcutEdit.SetEnabled(false)
										_ = Weather70ShortcutRecordButton.SetText("Record shortcut")
									}
								},
							},
						},
					},
					Composite{
						Layout:    HBox{},
						Alignment: AlignHNearVNear,
						Children: []Widget{
							TextLabel{
								Text: "Toggle overlay",
							},
							LineEdit{
								Text:     Bind("OverlayShortcutText"),
								AssignTo: &OverlayShortcutEdit,
								OnKeyPress: func(key walk.Key) {
									shortcut := walk.Shortcut{Modifiers: walk.ModifiersDown(), Key: key}
									_ = OverlayShortcutEdit.SetText(shortcut.String())
									shortcutStringToKey[shortcut.String()] = shortcut
								},
								Enabled:  false,
								ReadOnly: true,
							},
							PushButton{
								AssignTo: &OverlayShortcutRecordButton,
								MinSize:  Size{Height: 20},
								Text:     "Record shortcut",
								OnClicked: func() {
									if !OverlayShortcutEdit.Enabled() { // start recording
										OverlayShortcutEdit.SetEnabled(true)
										_ = OverlayShortcutEdit.SetFocus()
										_ = OverlayShortcutRecordButton.SetText("Save")
									} else { // persist new shortcut and rebind
										OverlayShortcutEdit.SetEnabled(false)
										_ = OverlayShortcutRecordButton.SetText("Record shortcut")
									}
								},
							},
						},
					},
				},
				AlwaysConsumeSpace: true,
				MinSize:            Size{Height: 20},
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
