package main

import (
	"fmt"
	"image"
	"image/color"
	"log"
	"os"
	"regexp"
	"time"

	"github.com/disintegration/imaging"
	"github.com/lxn/walk"
	"github.com/lxn/win"
	"github.com/shivas/abyss-blackbox/combatlog"
	"github.com/shivas/abyss-blackbox/internal/charmanager"
	"github.com/shivas/abyss-blackbox/internal/config"
	"github.com/shivas/abyss-blackbox/internal/mainwindow"
	"github.com/shivas/abyss-blackbox/internal/uploader"
	"github.com/shivas/abyss-blackbox/screen"
)

const EVEClientWindowRe = "^EVE -.*$"

var previewChannel chan image.Image
var recordingChannel chan *image.Paletted
var notificationChannel chan NotificationMessage
var recorder *Recorder

func main() {
	var err error

	currentSettings, err := config.Read()
	if err != nil {
		log.Fatal(err)
	}

	previewChannel = make(chan image.Image, 10)
	recordingChannel = make(chan *image.Paletted, 10)
	notificationChannel = make(chan NotificationMessage, 10)

	// combatlog reader init
	clr := combatlog.NewReader(currentSettings.EVEGameLogsFolder)
	recorder = NewRecorder(recordingChannel, currentSettings, notificationChannel, clr)
	recorder.StartLoop()

	defer recorder.StopLoop()

	foundEVEClientWindows, _ := screen.FindWindow(regexp.MustCompile(EVEClientWindowRe))
	foundEVEClientWindows = foundEVEClientWindows.EnsureUniqueNames()

	comboModel := make([]*mainwindow.WindowComboBoxItem, 0)
	for handle, title := range foundEVEClientWindows {
		comboModel = append(comboModel, &mainwindow.WindowComboBoxItem{WindowTitle: title, WindowHandle: handle})
	}

	charManager := charmanager.New(func(s1, s2 string) {
		notificationChannel <- NotificationMessage{Title: s1, Message: s2}
	})

	actions := make(map[string]walk.EventHandler)
	actions["add_character"] = charManager.EventHandlerCharAdd

	armw := mainwindow.NewAbyssRecorderWindow(currentSettings, drawStuff, comboModel, actions, clr)
	_ = charManager.MainWindow(armw).LoadCache() // assign window to control widgets
	charManager.RefreshUI()

	charManager.OnActivateCharacter =
		func(char charmanager.Character) {
			if char.CharacterID > 0 {
				_ = armw.MainWindow.SetTitle("Abyssal.Space Blackbox Recorder - " + char.CharacterName)
			}

			notificationChannel <- NotificationMessage{Title: "Active character set to:", Message: char.CharacterName}

			currentSettings.ActiveCharacter = char.CharacterID
			armw.AutoUploadCheckbox.SetEnabled(char.CharacterID > 0)

			_ = config.Write(currentSettings)
		}

	_ = charManager.SetActiveCharacter(currentSettings.ActiveCharacter)

	if len(foundEVEClientWindows) == 0 {
		walk.MsgBox(armw.MainWindow, "No signed in EVE clients detected", "Please login with atleast one character and then restart this application", walk.MsgBoxIconWarning)
	}

	notificationIcon := CreateNotificationIcon(armw.MainWindow)

	defer func() {
		_ = notificationIcon.Dispose()
	}()

	// notification routine
	go func(nc chan NotificationMessage, ni *walk.NotifyIcon) {
		for msg := range nc {
			_ = ni.ShowMessage(msg.Title, msg.Message)
		}
	}(notificationChannel, notificationIcon)

	recordingButtonHandler := func() {
		if recorder.Status() == RecorderStopped {
			charsChecked := []string{}

			checkboxes := armw.CombatLogCharacterGroup.Children()
			for i := 0; i < checkboxes.Len(); i++ {
				cb, ok := checkboxes.At(i).(*walk.CheckBox)
				if !ok {
					continue
				}

				if cb.Checked() {
					charsChecked = append(charsChecked, cb.Text())
				}
			}

			if len(charsChecked) == 0 {
				walk.MsgBox(armw.MainWindow, "No characters selected", "Please choose atleast one character to capture combat log", walk.MsgBoxIconWarning)
				return
			}

			recorder.Start(charsChecked)

			_ = armw.MainWindow.Menu().Actions().At(0).SetVisible(false)
			armw.CombatLogCharacterGroup.SetEnabled(false)
			armw.CaptureSettingsGroup.SetEnabled(false)
			armw.TestServer.SetEnabled(false)
			_ = armw.RecordingButton.SetText("Stop recording")
		} else {
			filename, errr := recorder.Stop()
			if errr != nil {
				walk.MsgBox(armw.MainWindow, "Error writing recording", errr.Error(), walk.MsgBoxIconWarning)
			}

			char := charManager.ActiveCharacter()

			if armw.AutoUploadCheckbox.Checked() && char != nil && errr == nil {
				go func(fn string, t string) {
					uploadFile, uploadErr := uploader.Upload(fn, t)
					if uploadErr != nil {
						walk.MsgBox(armw.MainWindow, "Record uploading error", uploadErr.Error(), walk.MsgBoxIconWarning)
					} else {
						notificationChannel <- NotificationMessage{Title: "Record uploaded successfully", Message: uploadFile}
					}
				}(filename, char.CharacterToken)
			}

			_ = armw.MainWindow.Menu().Actions().At(0).SetVisible(true)
			armw.CombatLogCharacterGroup.SetEnabled(true)
			armw.CaptureSettingsGroup.SetEnabled(true)
			armw.TestServer.SetEnabled(true)
			_ = armw.RecordingButton.SetText("Start recording")
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
			recorder.GetWeatherStrengthListener(30)()
		case config.HotkeyWeather50:
			recorder.GetWeatherStrengthListener(50)()
		case config.HotkeyWeather70:
			recorder.GetWeatherStrengthListener(70)()
		}
	})

	walk.RegisterGlobalHotKey(armw.MainWindow, config.HotkeyRecoder, currentSettings.RecorderShortcut)
	walk.RegisterGlobalHotKey(armw.MainWindow, config.HotkeyWeather30, currentSettings.Weather30Shortcut)
	walk.RegisterGlobalHotKey(armw.MainWindow, config.HotkeyWeather50, currentSettings.Weather50Shortcut)
	walk.RegisterGlobalHotKey(armw.MainWindow, config.HotkeyWeather70, currentSettings.Weather70Shortcut)

	go func(cw *walk.CustomWidget) {
		t := time.NewTicker(time.Second)

		for range t.C {
			img, errr := screen.CaptureWindowArea(
				foundEVEClientWindows.GetHandleByTitle(currentSettings.EVEClientWindowTitle),
				image.Rectangle{Min: image.Point{X: currentSettings.X, Y: currentSettings.Y}, Max: image.Point{X: currentSettings.X + 255, Y: currentSettings.Y + currentSettings.H}},
			)
			if errr != nil {
				walk.MsgBox(armw.MainWindow, "Error capturing window area, restart of application is needed.", errr.Error(), walk.MsgBoxIconWarning)
				fmt.Printf("error lost capture window: %v", err)
				os.Exit(1) // exit
			}

			img2 := imaging.Grayscale(img)
			img2 = imaging.Invert(img2)
			img3 := paletted(img2, uint32(currentSettings.FilterThreshold))

			if currentSettings.FilteredPreview {
				select {
				case previewChannel <- img3:
				default:
					log.Println("preview channel full, dropping frame")
				}
			} else {
				select {
				case previewChannel <- img:
				default:
					log.Println("preview channel full, dropping frame")
				}
			}

			select {
			case recordingChannel <- img3:
			default:
				log.Println("recorder channel is full, dropping frame")
			}

			win.InvalidateRect(cw.Handle(), nil, false)
		}
	}(armw.CaptureWidget)

	walk.Clipboard().ContentsChanged().Attach(recorder.ClipboardListener)

	defer func() {
		err = config.Write(currentSettings)
		if err != nil {
			log.Fatalf("failed to save settings after main window close: %v", err)
		}
	}()

	armw.MainWindow.Run()
}

type NotificationMessage struct {
	Title   string
	Message string
}

// CreateNotificationIcon creates walk.NotifyIcon that can be used to send notifications to user
func CreateNotificationIcon(mw *walk.MainWindow) *walk.NotifyIcon {
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

// drawStuff draw function for capture preview widget
func drawStuff(canvas *walk.Canvas, updateBounds walk.Rectangle) error {
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

// paletted convert image in NRGBA mode to Paletted image given cutoff as separator between two colors
func paletted(img *image.NRGBA, cutoff uint32) *image.Paletted {
	palette := []color.Color{color.White, color.Black}
	buffimg := image.NewPaletted(img.Rect, palette)

	var threshold = cutoff << 8

	for y := 0; y < img.Rect.Dy(); y++ {
		for x := 0; x < img.Rect.Dx(); x++ {
			c := img.At(x, y)
			r, _, _, _ := c.RGBA()

			if r > threshold {
				buffimg.SetColorIndex(x, y, 0)
				continue
			}

			if r <= threshold {
				buffimg.SetColorIndex(x, y, 1)
				continue
			}
		}
	}

	return buffimg
}
