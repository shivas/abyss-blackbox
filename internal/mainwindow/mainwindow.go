package mainwindow

import (
	"log"
	"syscall"
	"time"

	"github.com/lxn/walk"
	. "github.com/lxn/walk/declarative" // nolint:stylecheck,revive // we needs side effects
	"github.com/lxn/win"
	"github.com/shivas/abyss-blackbox/combatlog"
	"github.com/shivas/abyss-blackbox/internal/config"
)

type WindowComboBoxItem struct {
	WindowTitle  string
	WindowHandle syscall.Handle
}

type AbyssRecorderWindow struct {
	MainWindow              *walk.MainWindow
	FilteredPreview         *walk.CheckBox
	DataBinder              *walk.DataBinder
	CaptureWidget           *walk.CustomWidget
	HSetting                *walk.NumberEdit
	RecordingButton         *walk.PushButton
	CaptureWindowComboBox   *walk.ComboBox
	CombatLogCharacterGroup *walk.GroupBox
	CaptureSettingsGroup    *walk.GroupBox
	TestServer              *walk.CheckBox
	CharacterSwitcherMenu   *walk.Menu
	Toolbar                 *walk.ToolBar
	AutoUploadCheckbox      *walk.CheckBox
	SettingsAction          *walk.Action
}

// NewAbyssRecorderWindow creates new main window of recorder.
func NewAbyssRecorderWindow(
	c interface{},
	customWidgetPaintFunc walk.PaintFunc,
	comboBoxModel []*WindowComboBoxItem,
	actions map[string]walk.EventHandler,
	clr *combatlog.Reader,
) *AbyssRecorderWindow {
	obj := AbyssRecorderWindow{}

	if err := (MainWindow{
		AssignTo: &obj.MainWindow,
		Title:    "Abyssal.Space Blackbox Recorder",
		MinSize:  Size{Width: 320, Height: 240},
		Size:     Size{Width: 400, Height: 600},
		Layout:   HBox{MarginsZero: true, Alignment: AlignHNearVNear},
		DataBinder: DataBinder{
			AssignTo:        &obj.DataBinder,
			DataSource:      c,
			AutoSubmit:      true,
			AutoSubmitDelay: time.Duration(1) * time.Second,
		},
		ToolBar: ToolBar{
			ButtonStyle: ToolBarButtonImageBeforeText,
			AssignTo:    &obj.Toolbar,
			Items: []MenuItem{
				Menu{
					Text:     "Switch character",
					Image:    21,
					AssignTo: &obj.CharacterSwitcherMenu,
					Items:    []MenuItem{},
					Enabled:  false,
				},
				Separator{},
				Action{
					Text:        "Add character",
					Image:       14,
					OnTriggered: actions["add_character"],
				},
			},
		},
		MenuItems: []MenuItem{
			Action{
				AssignTo: &obj.SettingsAction,
				Text:     "Settings",
			},
			Action{
				Text: "About",
			},
		},
		Children: []Widget{
			Composite{
				Layout: HBox{},
				Children: []Widget{
					Composite{
						Layout:    VBox{MarginsZero: true},
						Alignment: AlignHNearVNear,
						Children: []Widget{
							GroupBox{
								Title:  "Overview settings",
								Layout: VBox{},
								Children: []Widget{
									CheckBox{
										AssignTo:  &obj.FilteredPreview,
										Text:      "show filtered preview",
										Alignment: AlignHNearVNear,
										Checked:   Bind("FilteredPreview"),
									},
									GroupBox{
										Title:    "Capture region",
										Layout:   HBox{},
										AssignTo: &obj.CaptureSettingsGroup,
										Children: []Widget{
											ComboBox{
												AssignTo:      &obj.CaptureWindowComboBox,
												Model:         comboBoxModel,
												ToolTipText:   "EVE client to capture",
												BindingMember: "WindowTitle",
												DisplayMember: "WindowTitle",
												Value:         Bind("EVEClientWindowTitle"),
												Editable:      false,
											},
											TextLabel{
												Text: "X:",
											},
											NumberEdit{
												Value: Bind("X"),
											},
											TextLabel{
												Text: "Y:",
											},
											NumberEdit{
												Value: Bind("Y"),
											},
											TextLabel{
												Text: "Height:",
											},
											NumberEdit{
												AssignTo: &obj.HSetting,
												Value:    Bind("H"),
												OnValueChanged: func() {
													if obj.CaptureWidget != nil {
														_ = obj.CaptureWidget.SetMinMaxSizePixels(walk.Size{Height: int(obj.HSetting.Value()), Width: 255}, walk.Size{})
														obj.CaptureWidget.RequestLayout()
													}
												},
											},
										},
									},
								},
							},
							GroupBox{
								Title:     "Server flag:",
								Layout:    VBox{},
								Alignment: AlignHNearVNear,
								Children: []Widget{
									CheckBox{
										AssignTo:  &obj.TestServer,
										Text:      "Test Server (Singularity)",
										Alignment: AlignHNearVNear,
										Checked:   Bind("TestServer"),
									},
								},
							},
							GroupBox{
								Title:              "Capture combatlog of characters:",
								Layout:             VBox{},
								Alignment:          AlignHNearVNear,
								AssignTo:           &obj.CombatLogCharacterGroup,
								Children:           []Widget{},
								AlwaysConsumeSpace: true,
								MinSize:            Size{Height: 20},
							},
							CheckBox{
								Text:        "Upload file after recording complete",
								AssignTo:    &obj.AutoUploadCheckbox,
								Alignment:   AlignHNearVCenter,
								Enabled:     false,
								Checked:     Bind("AutoUpload"),
								ToolTipText: "Automatically uploads recorded file to active character account.",
							},
							PushButton{
								AssignTo: &obj.RecordingButton,
								MinSize:  Size{Height: 40},
								Text:     "Start recording",
							},
						},
					},
					GroupBox{
						Title:  "Captured region:",
						Layout: VBox{},
						Children: []Widget{
							ScrollView{
								Layout:          VBox{},
								HorizontalFixed: true,
								Children: []Widget{
									CustomWidget{
										AssignTo:            &obj.CaptureWidget,
										MinSize:             Size{Width: 255, Height: 800},
										Paint:               customWidgetPaintFunc,
										InvalidatesOnResize: true,
										ClearsBackground:    true,
										DoubleBuffering:     true,
									},
								},
							},
						},
					},
				},
			},
		},
	}.Create()); err != nil {
		log.Fatal(err)
	}

	logFiles, _ := clr.GetLogFiles(time.Now(), time.Duration(24)*time.Hour)

	charMap := clr.MapCharactersToFiles(logFiles)
	buildRunnerList(charMap, obj.CombatLogCharacterGroup)

	settingsChangedHandler := func(c *config.CaptureConfig) {
		_ = config.Write(c)
		clr.SetLogFolder(c.EVEGameLogsFolder)
		logFiles, err := clr.GetLogFiles(time.Now(), time.Duration(24)*time.Hour)

		if err != nil {
			return
		}

		buildRunnerList(clr.MapCharactersToFiles(logFiles), obj.CombatLogCharacterGroup)

		// rebind hotkeys
		win.UnregisterHotKey(obj.MainWindow.Handle(), config.HotkeyRecoder)
		win.UnregisterHotKey(obj.MainWindow.Handle(), config.HotkeyWeather30)
		win.UnregisterHotKey(obj.MainWindow.Handle(), config.HotkeyWeather50)
		win.UnregisterHotKey(obj.MainWindow.Handle(), config.HotkeyWeather70)

		walk.RegisterGlobalHotKey(obj.MainWindow, config.HotkeyRecoder, c.RecorderShortcut)
		walk.RegisterGlobalHotKey(obj.MainWindow, config.HotkeyWeather30, c.Weather30Shortcut)
		walk.RegisterGlobalHotKey(obj.MainWindow, config.HotkeyWeather50, c.Weather50Shortcut)
		walk.RegisterGlobalHotKey(obj.MainWindow, config.HotkeyWeather70, c.Weather70Shortcut)
	}

	obj.SettingsAction.Triggered().Attach(func() {
		_, _ = RunAnimalDialog(obj.MainWindow, c, settingsChangedHandler)
	})

	return &obj
}

func buildRunnerList(characters map[string]combatlog.CombatLogFile, parent walk.Container) {
	for i := 0; i < parent.Children().Len(); i++ {
		w := parent.Children().At(i)
		w.SetVisible(false)
	}

	for charName := range characters {
		cb, _ := walk.NewCheckBox(parent)
		_ = cb.SetText(charName)
		_ = cb.SetMinMaxSize(walk.Size{Width: 400}, walk.Size{Width: 800})
		_ = cb.SetAlignment(walk.AlignHNearVCenter)
		cb.SetChecked(false)
		_ = parent.Children().Add(cb)
	}
}
