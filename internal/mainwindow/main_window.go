package mainwindow

import (
	"image"
	"log"
	"syscall"
	"time"

	"github.com/lxn/walk"
	. "github.com/lxn/walk/declarative" //nolint:stylecheck,revive // we needs side effects
	"github.com/lxn/win"
	"github.com/shivas/abyss-blackbox/internal/app/domain"
	"github.com/shivas/abyss-blackbox/internal/config"
	"github.com/shivas/abyss-blackbox/internal/fittings"
	"github.com/shivas/abyss-blackbox/pkg/combatlog"
)

const presetToolbarAction = 3

type WindowComboBoxItem struct {
	WindowTitle  string
	WindowHandle syscall.Handle
}

type UIScalingComboBoxItem struct {
	UIScalingTitle string
	RecorderWidth  int
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
	UIScalingComboBox      *walk.ComboBox
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
	RecorderWidth          int
}

// NewAbyssRecorderWindow creates new main window of recorder.
func NewAbyssRecorderWindow(
	c interface{},
	customWidgetPaintFunc walk.PaintFunc,
	comboBoxModel []*WindowComboBoxItem,
	actions map[string]walk.EventHandler,
	clr *combatlog.Reader,
	fm *fittings.FittingsManager,
	serverProvider domain.ServerProvider,
) *AbyssRecorderWindow {
	obj := AbyssRecorderWindow{FittingManager: fm}

	UIScalingItems := []*UIScalingComboBoxItem{
		{UIScalingTitle: "100", RecorderWidth: 255},
		{UIScalingTitle: "110", RecorderWidth: 255},
		{UIScalingTitle: "125", RecorderWidth: 300},
		{UIScalingTitle: "150", RecorderWidth: 300},
		{UIScalingTitle: "175", RecorderWidth: 400},
	}

	obj.RecorderWidth = 255

	for _, v := range UIScalingItems {
		if v.UIScalingTitle == c.(*config.CaptureConfig).EVEClientUIScaling {
			obj.RecorderWidth = v.RecorderWidth
		}
	}

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
					Image:    16,
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
					Image:    23,
					AssignTo: &obj.PresetSwitcherMenu,
					Items:    []MenuItem{},
					Enabled:  false,
				},
				Separator{},
				Action{
					Text:        "Overlay",
					Image:       25,
					OnTriggered: actions["show_overlay"],
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
														_ = obj.CaptureWidget.SetMinMaxSizePixels(walk.Size{Height: int(obj.HSetting.Value()), Width: obj.RecorderWidth}, walk.Size{})
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
									Composite{
										Layout: HBox{},
										Children: []Widget{
											TextLabel{
												Text: "EVE client UI scaling:",
											},
											ComboBox{
												AssignTo:      &obj.UIScalingComboBox,
												Model:         UIScalingItems,
												ToolTipText:   "EVE client UI scaling setting, used to adjust recorder width to fit all entities. If using large font in EVE, set this to maximum value.",
												BindingMember: "UIScalingTitle",
												DisplayMember: "UIScalingTitle",
												Value:         Bind("EVEClientUIScaling"),
												Editable:      false,
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
										Visible:   false,
										Title:     "Server flag:",
										Layout:    VBox{},
										Alignment: AlignHNearVNear,
										Children: []Widget{
											CheckBox{
												AssignTo:  &obj.TestServer,
												Text:      "Test Server (Singularity)",
												Alignment: AlignHNearVNear,
												Checked:   Bind("TestServer"),
												Enabled:   false,
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
								MinSize:             Size{Width: obj.RecorderWidth, Height: 1},
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
		win.UnregisterHotKey(obj.MainWindow.Handle(), config.Overlay)

		walk.RegisterGlobalHotKey(obj.MainWindow, config.HotkeyRecoder, c.RecorderShortcut)
		walk.RegisterGlobalHotKey(obj.MainWindow, config.HotkeyWeather30, c.Weather30Shortcut)
		walk.RegisterGlobalHotKey(obj.MainWindow, config.HotkeyWeather50, c.Weather50Shortcut)
		walk.RegisterGlobalHotKey(obj.MainWindow, config.HotkeyWeather70, c.Weather70Shortcut)
		walk.RegisterGlobalHotKey(obj.MainWindow, config.Overlay, c.OverlayShortcut)
	}

	obj.UIScalingComboBox.CurrentIndexChanged().Attach(func() {
		if obj.UIScalingComboBox.CurrentIndex() < 0 && obj.UIScalingComboBox.CurrentIndex() > len(UIScalingItems)-1 {
			return
		}

		for _, v := range UIScalingItems {
			if v.UIScalingTitle == UIScalingItems[obj.UIScalingComboBox.CurrentIndex()].UIScalingTitle {
				_ = obj.CapturePreviewGroupBox.SetMinMaxSize(walk.Size{Width: v.RecorderWidth, Height: c.(*config.CaptureConfig).H}, walk.Size{})
				_ = obj.CaptureWidget.SetWidth(v.RecorderWidth)
				obj.RecorderWidth = v.RecorderWidth
			}
		}
	})

	obj.SettingsAction.Triggered().Attach(func() {
		_, _ = RunSettingsDialog(obj.MainWindow, c, settingsChangedHandler)
	})

	obj.CaptureWindowComboBox.CurrentIndexChanged().Attach(func() {
		if obj.CaptureWindowComboBox.CurrentIndex() < 0 && obj.CaptureWindowComboBox.CurrentIndex() > len(comboBoxModel)-1 {
			return
		}

		handle := comboBoxModel[obj.CaptureWindowComboBox.CurrentIndex()].WindowHandle
		obj.TestServer.SetChecked(serverProvider.IsTestingServer(handle))
	})

	chooser := NewAbyssTypeChooser(obj.AbyssTypeToolbar, c.(*config.CaptureConfig))
	chooser.Init()

	return &obj
}

// DrawStuff returns draw function main window preview custom widget.
func WidgetDrawFn(
	previewChannel chan image.Image,
	recordingChannel chan *image.Paletted,
) walk.PaintFunc {
	return func(canvas *walk.Canvas, updateBounds walk.Rectangle) error {
		select {
		case img := <-previewChannel:
			bmp, err := walk.NewBitmapFromImageForDPI(img, 96)
			if err != nil {
				return err
			}

			defer bmp.Dispose()

			err = canvas.DrawImagePixels(bmp, walk.Point{X: 0, Y: 0})
			if err != nil {
				return err
			}
		default:
			return nil
		}

		return nil
	}
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
