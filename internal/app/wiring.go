package app

import (
	"errors"
	"fmt"
	"image"
	"log"

	"github.com/lxn/walk"
	"golang.org/x/exp/slog"

	"github.com/shivas/abyss-blackbox/internal/app/api/client"
	"github.com/shivas/abyss-blackbox/internal/app/domain"
	"github.com/shivas/abyss-blackbox/internal/app/recorder"
	"github.com/shivas/abyss-blackbox/internal/charmanager"
	"github.com/shivas/abyss-blackbox/internal/config"
	"github.com/shivas/abyss-blackbox/internal/fittings"
	"github.com/shivas/abyss-blackbox/internal/fittings/provider"
	"github.com/shivas/abyss-blackbox/internal/mainwindow"
	"github.com/shivas/abyss-blackbox/internal/overlay"
	"github.com/shivas/abyss-blackbox/internal/screen"
	"github.com/shivas/abyss-blackbox/internal/uploader"
	"github.com/shivas/abyss-blackbox/internal/window"
	"github.com/shivas/abyss-blackbox/pkg/combatlog"
)

var (
	previewChannel      chan image.Image
	recordingChannel    chan *image.Paletted
	notificationChannel chan domain.NotificationMessage
	rec                 *recorder.Recorder
)

func Run() (err error) {
	currentSettings, err := config.Read()
	if err != nil {
		slog.Error("failed loading settings", err)
		return err
	}

	previewChannel = make(chan image.Image, 10)
	recordingChannel = make(chan *image.Paletted, 10)
	notificationChannel = make(chan domain.NotificationMessage, 10)

	windowsManager, err := window.NewManager()
	if err != nil && !errors.Is(err, window.ErrNoWindowsFound) {
		slog.Error("window manager initialization failed", err)
		return err
	}

	overlayManager := overlay.New(
		overlay.FromCaptureConfig(currentSettings),
		currentSettings,
	)
	defer overlayManager.Close()

	overlayManager.ChangeProperty(overlay.Status, "Recorder on standby", &overlay.YellowColor)

	// combatlog reader init
	clr := combatlog.NewReader(currentSettings.EVEGameLogsFolder)
	rec = recorder.NewRecorder(recordingChannel, currentSettings, notificationChannel, clr, overlayManager)
	rec.StartLoop()

	defer rec.StopLoop()

	comboModel := make([]*mainwindow.WindowComboBoxItem, 0)
	for handle, title := range windowsManager.GetEVEClientWindows(window.SupportedWindowsFilter) {
		comboModel = append(comboModel, &mainwindow.WindowComboBoxItem{WindowTitle: title, WindowHandle: handle})
	}

	charManager := charmanager.New(func(s1, s2 string) {
		notificationChannel <- domain.NotificationMessage{Title: s1, Message: s2}
	})

	authenticatedHTTPClient := client.New(charManager)
	fittingsManager := fittings.NewManager(authenticatedHTTPClient, provider.NewTrackerFittingsProvider(authenticatedHTTPClient))

	actions := make(map[string]walk.EventHandler)
	actions["add_character"] = charManager.EventHandlerCharAdd
	actions["show_overlay"] = func() {
		overlayManager.ToggleOverlay()
	}

	armw := mainwindow.NewAbyssRecorderWindow(
		currentSettings,
		mainwindow.WidgetDrawFn(previewChannel, recordingChannel),
		comboModel,
		actions,
		clr,
		fittingsManager,
		windowsManager,
	)
	_ = charManager.MainWindow(armw).LoadCache() // assign window to control widgets
	charManager.RefreshUI()
	armw.MainWindow.Closing().Once(func(canceled *bool, reason walk.CloseReason) {
		overlayManager.Close()
	})

	charManager.OnActivateCharacter = func(char charmanager.Character) {
		if char.CharacterID > 0 {
			_ = armw.MainWindow.SetTitle("Abyssal.Space Blackbox Recorder - " + char.CharacterName)
		}

		notificationChannel <- domain.NotificationMessage{Title: "Active character set to:", Message: char.CharacterName}

		currentSettings.ActiveCharacter = char.CharacterID
		armw.AutoUploadCheckbox.SetEnabled(char.CharacterID > 0)

		_ = config.Write(currentSettings)
	}

	_ = charManager.SetActiveCharacter(currentSettings.ActiveCharacter)

	if len(windowsManager.GetEVEClientWindows(window.SupportedWindowsFilter)) == 0 {
		walk.MsgBox(armw.MainWindow, "Error: No supported EVE client windows detected",
			`To use this application, you need to log in with at least one character.

 If you already have a game running and a character logged in, it means that your EVE client is either using DirectX 12 or a language other than English.

 Please reconfigure EVE client accordingly and restart this application.`, walk.MsgBoxIconWarning)
	}

	notificationIcon := createNotificationIcon(armw.MainWindow)

	defer func() {
		_ = notificationIcon.Dispose()
	}()

	// notification routine
	go func(nc chan domain.NotificationMessage, ni *walk.NotifyIcon) {
		for msg := range nc {
			if !currentSettings.SuppressNotifications {
				_ = ni.ShowMessage(msg.Title, msg.Message)
			}
		}
	}(notificationChannel, notificationIcon)

	recordingButtonHandler := func() {
		if rec.Status() == recorder.RecorderStopped {
			if runnerModel, ok := armw.RunnerTableView.Model().(*mainwindow.RunnerModel); ok {
				checkedChars := runnerModel.GetCheckedCharacters()

				if len(checkedChars) == 0 {
					walk.MsgBox(armw.MainWindow, "No characters selected", "Please choose atleast one character to capture combat log", walk.MsgBoxIconWarning)
					return
				}

				if len(checkedChars) > 3 {
					walk.MsgBox(armw.MainWindow, "Too much characters selected", "Please choose up-to 3 characters to capture combat log", walk.MsgBoxIconWarning)
					return
				}

				rec.Start(checkedChars)
			}

			overlayManager.ChangeProperty(overlay.Autoupload, fmt.Sprintf("Autoupload enabled: %t", armw.AutoUploadCheckbox.Checked()), &overlay.CyanColor)

			if currentSettings.AbyssTypeOverride {
				overlayManager.ChangeProperty(overlay.Override, fmt.Sprintf("Abyss type override: %s", tierOverrideToString(currentSettings)), &overlay.SecondaryColor)
			} else {
				overlayManager.ChangeProperty(overlay.Override, "Abyss type detection: heuristics", &overlay.CyanColor)
			}

			_ = armw.MainWindow.Menu().Actions().At(0).SetVisible(false)
			armw.RunnerCharacterGroup.SetEnabled(false)
			armw.RunnerTableView.SetEnabled(false)
			armw.ManageFittingsButton.SetEnabled(false)
			armw.CaptureSettingsGroup.SetEnabled(false)
			//armw.TestServer.SetEnabled(false)
			_ = armw.Toolbar.Actions().At(3).SetEnabled(false)
			_ = armw.RecordingButton.SetText("Stop recording")
		} else {
			filename, errr := rec.Stop(armw.FittingManager)
			if errr != nil {
				walk.MsgBox(armw.MainWindow, "Error writing recording", errr.Error(), walk.MsgBoxIconWarning)
			}

			char := charManager.ActiveCharacter()

			if armw.AutoUploadCheckbox.Checked() && char != nil && errr == nil {
				go func(fn string) {
					uploadFile, uploadErr := uploader.Upload(authenticatedHTTPClient, fn)
					if uploadErr != nil {
						walk.MsgBox(armw.MainWindow, "Record uploading error", uploadErr.Error(), walk.MsgBoxIconWarning)
					} else {
						notificationChannel <- domain.NotificationMessage{Title: "Record uploaded successfully", Message: uploadFile}
						overlayManager.ChangeProperty(overlay.TODO, "Record uploaded successfully", &overlay.GreenColor)
					}
				}(filename)
			}

			_ = armw.MainWindow.Menu().Actions().At(0).SetVisible(true)
			armw.RunnerCharacterGroup.SetEnabled(true)
			armw.RunnerTableView.SetEnabled(true)
			armw.ManageFittingsButton.SetEnabled(true)
			armw.CaptureSettingsGroup.SetEnabled(true)
			//armw.TestServer.SetEnabled(true)
			_ = armw.Toolbar.Actions().At(3).SetEnabled(true)
			_ = armw.RecordingButton.SetText("Start recording")

			overlayManager.ChangeProperty(overlay.Status, "Recorder on standby", &overlay.YellowColor)
			overlayManager.ChangeProperty(overlay.Weather, "", nil)
		}
	}

	armw.RecordingButton.Clicked().Attach(recordingButtonHandler)
	armw.PresetSaveButton.Clicked().Attach(func() {
		p := config.Preset{X: currentSettings.X, Y: currentSettings.Y, H: currentSettings.H}
		_, _ = mainwindow.RunNewPresetDialog(armw.MainWindow, p, currentSettings)
		_ = config.Write(currentSettings)
		armw.RefreshPresets(currentSettings)
	})

	armw.RefreshPresets(currentSettings)

	armw.MainWindow.Hotkey().Attach(func(hkid int) {
		switch hkid {
		case config.HotkeyRecoder:
			recordingButtonHandler()
		case config.HotkeyWeather30:
			rec.GetWeatherStrengthListener(30)()
		case config.HotkeyWeather50:
			rec.GetWeatherStrengthListener(50)()
		case config.HotkeyWeather70:
			rec.GetWeatherStrengthListener(70)()
		case config.Overlay:
			overlayManager.ToggleOverlay()
		}
	})

	walk.RegisterGlobalHotKey(armw.MainWindow, config.HotkeyRecoder, currentSettings.RecorderShortcut)
	walk.RegisterGlobalHotKey(armw.MainWindow, config.HotkeyWeather30, currentSettings.Weather30Shortcut)
	walk.RegisterGlobalHotKey(armw.MainWindow, config.HotkeyWeather50, currentSettings.Weather50Shortcut)
	walk.RegisterGlobalHotKey(armw.MainWindow, config.HotkeyWeather70, currentSettings.Weather70Shortcut)
	walk.RegisterGlobalHotKey(armw.MainWindow, config.Overlay, currentSettings.OverlayShortcut)

	go mainwindow.CustomWidgetDrawLoop(
		armw.CaptureWidget,
		armw.MainWindow,
		screen.NewDirectX11Capture(currentSettings, windowsManager, func() int { return armw.RecorderWidth }),
		currentSettings,
		previewChannel,
		recordingChannel,
	)

	walk.Clipboard().ContentsChanged().Attach(rec.ClipboardListener)

	defer func() {
		err = config.Write(currentSettings)
		if err != nil {
			log.Fatalf("failed to save settings after main window close: %v", err)
		}
	}()

	armw.MainWindow.Run()

	return nil
}

// createNotificationIcon creates walk.NotifyIcon that can be used to send notifications to user
func createNotificationIcon(mw *walk.MainWindow) *walk.NotifyIcon {
	// We load our icon from a file.
	icon, err := walk.Resources.Icon("7")
	if err != nil {
		log.Fatal(err)
	}

	// Create the notify icon and make sure we clean it up on exit.
	ni, err := walk.NewNotifyIcon(mw)
	if err != nil {
		log.Fatal(err)
	}

	// Set the icon and a tool tip text.
	if err := ni.SetIcon(icon); err != nil {
		log.Fatal(err)
	}

	if err := ni.SetToolTip("Click for info or use the context menu to exit."); err != nil {
		log.Fatal(err)
	}

	// We put an exit action into the context menu.
	exitAction := walk.NewAction()
	if err := exitAction.SetText("E&xit"); err != nil {
		log.Fatal(err)
	}

	exitAction.Triggered().Attach(func() { walk.App().Exit(0) })

	if err := ni.ContextMenu().Actions().Add(exitAction); err != nil {
		log.Fatal(err)
	}

	if err := ni.SetVisible(true); err != nil {
		log.Fatal(err)
	}

	return ni
}

func tierOverrideToString(c *config.CaptureConfig) string {
	var ship string

	switch c.AbyssShipType {
	case 1:
		ship = "Cruiser"
	case 2:
		ship = "Destroyers"
	case 3:
		ship = "Frigates"
	}

	return fmt.Sprintf("%s T%d %s", ship, c.AbyssTier, c.AbyssWeather)
}
