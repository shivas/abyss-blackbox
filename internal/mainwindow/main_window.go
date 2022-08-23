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
	"github.com/shivas/abyss-blackbox/internal/fittings"
)

const presetToolbarAction = 3

type WindowComboBoxItem struct {
	WindowTitle  string
	WindowHandle syscall.Handle
}

type AbyssRecorderWindow struct {
	MainWindow             *walk.MainWindow
	FilteredPreview        *walk.CheckBox
	DataBinder             *walk.DataBinder
	CaptureWidget          *walk.CustomWidget
	XSetting               *walk.NumberEdit
	YSetting               *walk.NumberEdit
	HSetting               *walk.NumberEdit
	RecordingButton        *walk.PushButton
	CaptureWindowComboBox  *walk.ComboBox
	RunnerCharacterGroup   *walk.GroupBox
	RunnerTableView        *walk.TableView
	CaptureSettingsGroup   *walk.GroupBox
	CapturePreviewGroupBox *walk.GroupBox
	TestServer             *walk.CheckBox
	CharacterSwitcherMenu  *walk.Menu
	Toolbar                *walk.ToolBar
	AutoUploadCheckbox     *walk.CheckBox
	SettingsAction         *walk.Action
	PresetSwitcherMenu     *walk.Menu
	PresetSaveButton       *walk.PushButton
	PreviewScrollView      *walk.ScrollView
	AbyssTypeToolbar       *walk.ToolBar
	ManageFittingsButton   *walk.PushButton
	FittingManager         *fittings.FittingsManager
}

// NewAbyssRecorderWindow creates new main window of recorder.
func NewAbyssRecorderWindow(
	c interface{},
	customWidgetPaintFunc walk.PaintFunc,
	comboBoxModel []*WindowComboBoxItem,
	actions map[string]walk.EventHandler,
	clr *combatlog.Reader,
	fm *fittings.FittingsManager,
) *AbyssRecorderWindow {
	obj := AbyssRecorderWindow{FittingManager: fm}

	logFiles, _ := clr.GetLogFiles(time.Now(), time.Duration(24)*time.Hour)
	characterMap := clr.MapCharactersToFiles(logFiles)
	pilots := make([]string, 0)

	for pilot := range characterMap {
		pilots = append(pilots, pilot)
	}

	runnerModel := NewRunnerModel(characterMap, fm)

	if err := (MainWindow{
		AssignTo: &obj.MainWindow,
		Title:    "Abyssal.Space Blackbox Recorder",
		MinSize:  Size{Width: 480, Height: 480},
		Size:     Size{Width: 480, Height: 480},
		Layout:   HBox{MarginsZero: true, Alignment: AlignHNearVNear},
		DataBinder: DataBinder{
			AssignTo:   &obj.DataBinder,
			DataSource: c,
			AutoSubmit: true,
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
				Action{
					Text:        "Add character",
					Image:       14,
					OnTriggered: actions["add_character"],
				},
				Separator{},
				Menu{
					Text:     "Presets",
					Image:    28,
					AssignTo: &obj.PresetSwitcherMenu,
					Items:    []MenuItem{},
					Enabled:  false,
				},
				Separator{},
			},
		},
		MenuItems: []MenuItem{
			Action{
				AssignTo: &obj.SettingsAction,
				Text:     "Settings",
			},
			Action{
				Text: "About",
				OnTriggered: func() {
					_, _ = RunAboutDialog(obj.MainWindow)
				},
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
								Title:    "Capture region",
								Layout:   VBox{SpacingZero: true},
								AssignTo: &obj.CaptureSettingsGroup,
								Children: []Widget{
									Composite{
										Layout: HBox{},
										Children: []Widget{
											TextLabel{
												Text: "EVE window:",
											},
											ComboBox{
												AssignTo:      &obj.CaptureWindowComboBox,
												Model:         comboBoxModel,
												ToolTipText:   "EVE client to capture",
												BindingMember: "WindowTitle",
												DisplayMember: "WindowTitle",
												Value:         Bind("EVEClientWindowTitle"),
												Editable:      false,
											},
											CheckBox{
												AssignTo:  &obj.FilteredPreview,
												Text:      "show filtered preview",
												Alignment: AlignHNearVNear,
												Checked:   Bind("FilteredPreview"),
											},
											HSpacer{},
										},
									},
									Composite{
										Layout: HBox{},
										Children: []Widget{
											TextLabel{
												Text: "X:",
											},
											NumberEdit{
												AssignTo: &obj.XSetting,
												MinSize:  Size{Width: 50, Height: 10},
												Value:    Bind("X"),
											},
											TextLabel{
												Text: "Y:",
											},
											NumberEdit{
												AssignTo: &obj.YSetting,
												MinSize:  Size{Width: 50, Height: 10},
												Value:    Bind("Y"),
											},
											TextLabel{
												Text: "Height:",
											},
											NumberEdit{
												AssignTo: &obj.HSetting,
												Value:    Bind("H"),
												MinSize:  Size{Width: 50, Height: 10},
												OnValueChanged: func() {
													if obj.CaptureWidget != nil {
														_ = obj.CaptureWidget.SetMinMaxSizePixels(walk.Size{Height: int(obj.HSetting.Value()), Width: 255}, walk.Size{})
														obj.CaptureWidget.RequestLayout()
													}
												},
											},
											PushButton{
												AssignTo: &obj.PresetSaveButton,
												Text:     "Save as preset",
											},
											HSpacer{},
										},
									},
								},
							},
							Composite{
								Layout:    HBox{MarginsZero: true},
								Alignment: AlignHNearVNear,
								Children: []Widget{
									GroupBox{
										Title:              "Manual abyss type override",
										Checkable:          true,
										Checked:            Bind("AbyssTypeOverride"),
										Layout:             VBox{},
										AlwaysConsumeSpace: true,
										Alignment:          AlignHNearVNear,
										Children: []Widget{
											ToolBar{
												AssignTo:    &obj.AbyssTypeToolbar,
												ButtonStyle: ToolBarButtonImageBeforeText,
												Items: []MenuItem{
													Menu{
														Text: "Ship",
														Items: []MenuItem{
															Action{
																Text: "Cruiser",
															},
															Action{
																Text: "Destroyers",
															},
															Action{
																Text: "Frigates",
															},
														},
													},
													Menu{
														Text: "Tier",
														Items: []MenuItem{
															Action{
																Text: "T0",
															},
															Action{
																Text: "T1",
															},
															Action{
																Text: "T2",
															},
															Action{
																Text: "T3",
															},
															Action{
																Text: "T4",
															},
															Action{
																Text: "T5",
															},
															Action{
																Text: "T6",
															},
														},
													},
													Menu{
														Text: "Weather",
														Items: []MenuItem{
															Action{
																Text: "Gamma",
															},
															Action{
																Text: "Exotic",
															},
															Action{
																Text: "Dark",
															},
															Action{
																Text: "Firestorm",
															},
															Action{
																Text: "Electrical",
															},
														},
													},
												},
											},
										},
									},
									HSpacer{},
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
									//									HSpacer{},
								},
							},
							GroupBox{
								Title:     "Capture combatlog of characters:",
								Layout:    VBox{},
								Alignment: AlignHNearVNear,
								AssignTo:  &obj.RunnerCharacterGroup,
								Children: []Widget{
									TableView{
										AssignTo:                    &obj.RunnerTableView,
										AlternatingRowBG:            false,
										CheckBoxes:                  true,
										ColumnsOrderable:            true,
										MultiSelection:              false,
										SelectionHiddenWithoutFocus: true,
										AlwaysConsumeSpace:          true,
										MinSize:                     Size{Height: 200},
										LastColumnStretched:         true,
										CustomRowHeight:             34,
										Columns: []TableViewColumn{
											{Title: "Pilot", Width: 150},
											{Title: "Ship"},
											{Title: "Fitting name"},
										},
									},
									PushButton{
										AssignTo: &obj.ManageFittingsButton,
										Text:     "Manage fittings",
										OnClicked: func() {
											_, _ = RunManageFittingsDialog(obj.MainWindow, make(map[string]string), fm, pilots)
											runnerModel.RefreshList()
										},
									},
								},
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
						Title:    "Captured region:",
						Layout:   VBox{},
						AssignTo: &obj.CapturePreviewGroupBox,
						OnSizeChanged: func() {
							h := obj.MainWindow.AsContainerBase().MinSizeHint()
							c := obj.MainWindow.AsContainerBase().Size()
							_ = obj.MainWindow.SetMinMaxSize(h, walk.Size{})
							if c.Height > h.Height {
								_ = obj.MainWindow.AsFormBase().WindowBase.SetSize(h)
							}
						},
						Children: []Widget{
							CustomWidget{
								AssignTo:            &obj.CaptureWidget,
								MinSize:             Size{Width: 255, Height: 1},
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
	}.Create()); err != nil {
		log.Fatal(err)
	}

	_ = obj.RunnerTableView.SetModel(runnerModel)
	obj.RunnerTableView.SetCellStyler(runnerModel)

	settingsChangedHandler := func(c *config.CaptureConfig) {
		_ = config.Write(c)
		clr.SetLogFolder(c.EVEGameLogsFolder)

		logFiles, err := clr.GetLogFiles(time.Now(), time.Duration(24)*time.Hour)
		if err != nil {
			return
		}

		runnerModel := NewRunnerModel(clr.MapCharactersToFiles(logFiles), fm)
		_ = obj.RunnerTableView.SetModel(runnerModel)
		obj.RunnerTableView.SetCellStyler(runnerModel)

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
		_, _ = RunSettingsDialog(obj.MainWindow, c, settingsChangedHandler)
	})

	chooser := NewAbyssTypeChooser(obj.AbyssTypeToolbar, c.(*config.CaptureConfig))
	chooser.Init()

	return &obj
}

func (m *AbyssRecorderWindow) RefreshPresets(c *config.CaptureConfig) {
	_ = m.PresetSwitcherMenu.Actions().Clear()

	for presetName := range c.Presets {
		presetName := presetName
		action := walk.NewAction()
		_ = action.SetText(presetName)
		_ = action.Triggered().Attach(func() {
			modifiers := walk.ModifiersDown()
			if modifiers == walk.ModControl {
				delete(c.Presets, presetName)
				_ = config.Write(c)
				m.RefreshPresets(c)
			} else {
				p := c.Presets[presetName]
				_ = m.XSetting.SetValue(float64(p.X))
				_ = m.YSetting.SetValue(float64(p.Y))
				_ = m.HSetting.SetValue(float64(p.H))
			}
		})

		_ = m.PresetSwitcherMenu.Actions().Add(action)
	}

	if len(c.Presets) > 0 {
		_ = m.Toolbar.Actions().At(presetToolbarAction).SetEnabled(true)
	} else {
		_ = m.Toolbar.Actions().At(presetToolbarAction).SetEnabled(false)
	}
}
