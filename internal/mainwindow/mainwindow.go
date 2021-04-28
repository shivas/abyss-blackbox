package mainwindow

import (
	"log"
	"syscall"
	"time"

	"github.com/lxn/walk"
	. "github.com/lxn/walk/declarative"
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
	EVEGameLogsFolderLabel  *walk.TextLabel
	ChooseLogDirButton      *walk.PushButton
	TestServer              *walk.CheckBox
}

func NewAbyssRecorderWindow(config interface{}, customWidgetPaintFunc walk.PaintFunc, comboBoxModel []*WindowComboBoxItem) *AbyssRecorderWindow {
	obj := AbyssRecorderWindow{}

	if err := (MainWindow{
		AssignTo: &obj.MainWindow,
		Title:    "Abyssal.Space Blackbox Recorder",
		MinSize:  Size{Width: 320, Height: 240},
		Size:     Size{Width: 400, Height: 600},
		Layout:   HBox{MarginsZero: false},
		DataBinder: DataBinder{
			AssignTo:        &obj.DataBinder,
			DataSource:      config,
			AutoSubmit:      true,
			AutoSubmitDelay: 1 * time.Second,
		},
		Children: []Widget{
			Composite{
				Layout: HBox{},
				Children: []Widget{
					Composite{
						Layout:    VBox{},
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
								Title:     "Server flag",
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
								Title:     "Combat log capture",
								Layout:    VBox{},
								Alignment: AlignHNearVNear,
								Children: []Widget{
									TextLabel{
										Text:     Bind("EVEGameLogsFolder"),
										AssignTo: &obj.EVEGameLogsFolderLabel,
									},
									PushButton{
										Text:     "Choose",
										AssignTo: &obj.ChooseLogDirButton,
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
									PushButton{
										AssignTo: &obj.RecordingButton,
										MinSize:  Size{Height: 40},
										Text:     "Start recording",
									},
								},
							},
						},
					},
					GroupBox{
						Title:  "Captured region:",
						Layout: VBox{},
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
	}.Create()); err != nil {
		log.Fatal(err)
	}

	return &obj
}
