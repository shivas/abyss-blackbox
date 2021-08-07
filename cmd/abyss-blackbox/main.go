package main

import (
	"encoding/json"
	"fmt"
	"image"
	"image/color"
	"log"
	"os"
	"os/user"
	"path/filepath"
	"regexp"
	"sync"
	"time"

	"github.com/disintegration/imaging"
	"github.com/lxn/walk"
	"github.com/lxn/win"
	"github.com/shivas/abyss-blackbox/combatlog"
	"github.com/shivas/abyss-blackbox/internal/mainwindow"
	"github.com/shivas/abyss-blackbox/screen"
)

const EVEClientWindowRe = "^EVE -.*$"

var previewChannel chan image.Image
var recordingChannel chan *image.Paletted
var notificationChannel chan NotificationMessage
var recorder *Recorder

type captureConfig struct {
	sync.Mutex
	X, Y, H              int
	AppRoot              string
	Recordings           string
	FilterThreshold      int
	FilteredPreview      bool
	EVEClientWindowTitle string
	EVEGameLogsFolder    string
	TestServer           bool
	RecorderShortcutText string
	RecorderShortcut     walk.Shortcut
}

// SetRecorderShortcut satisfies ShortcutSetter interface.
func (c *captureConfig) SetRecorderShortcut(s walk.Shortcut) {
	c.RecorderShortcut = s
	c.RecorderShortcutText = s.String()
}

func main() {
	var err error

	currentSettings, err := readConfig()
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

	logFiles, _ := clr.GetLogFiles(time.Now(), time.Duration(24)*time.Hour)
	charMap := clr.MapCharactersToFiles(logFiles)

	armw := mainwindow.NewAbyssRecorderWindow(currentSettings, drawStuff, comboModel)
	mainwindow.BuildSettingsWidget(charMap, armw.CombatLogCharacterGroup)

	if len(foundEVEClientWindows) == 0 {
		walk.MsgBox(armw.MainWindow, "No signed in EVE clients detected", "Please login with atleast one character and then restart this application", walk.MsgBoxIconWarning)
	}

	armw.ChooseLogDirButton.Clicked().Attach(func() {
		fd := walk.FileDialog{}
		accepted, _ := fd.ShowBrowseFolder(armw.MainWindow)
		if accepted {
			_ = armw.EVEGameLogsFolderLabel.SetText(fd.FilePath)
			clr.SetLogFolder(fd.FilePath)
			logFiles, err = clr.GetLogFiles(time.Now(), time.Duration(24)*time.Hour)
			if err != nil {
				return
			}
			charMap := clr.MapCharactersToFiles(logFiles)
			mainwindow.BuildSettingsWidget(charMap, armw.CombatLogCharacterGroup)
		}
	})

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
			armw.CombatLogCharacterGroup.SetEnabled(false)
			armw.CaptureSettingsGroup.SetEnabled(false)
			armw.ChooseLogDirButton.SetEnabled(false)
			armw.TestServer.SetEnabled(false)
			_ = armw.RecordingButton.SetText("Stop recording")
		} else {
			err = recorder.Stop()
			if err != nil {
				walk.MsgBox(armw.MainWindow, "Error writing recording", err.Error(), walk.MsgBoxIconWarning)
			}
			armw.CombatLogCharacterGroup.SetEnabled(true)
			armw.CaptureSettingsGroup.SetEnabled(true)
			armw.ChooseLogDirButton.SetEnabled(true)
			armw.TestServer.SetEnabled(true)
			_ = armw.RecordingButton.SetText("Start recording")
		}
	}

	armw.RecordingButton.Clicked().Attach(recordingButtonHandler)
	armw.MainWindow.Hotkey().Attach(func(hkid int) {
		if hkid == 1 {
			recordingButtonHandler()
		}
	})

	walk.RegisterGlobalHotKey(armw.MainWindow, 1, currentSettings.RecorderShortcut)

	shortcutRecorderHandler := func() {
		if !armw.RecorderShortcutEdit.Enabled() { // start recording
			armw.RecorderShortcutEdit.SetEnabled(true)
			_ = armw.RecorderShortcutEdit.SetFocus()
			_ = armw.RecorderShortcutRecordButton.SetText("Save")

			if !win.UnregisterHotKey(armw.MainWindow.Handle(), 1) {
				walk.MsgBox(armw.MainWindow, "Failed unregistering hotkey", "failed unregistering key, please restart application", walk.MsgBoxIconWarning)
				return
			}
		} else { // persist new shortcut and rebind
			armw.RecorderShortcutEdit.SetEnabled(false)
			_ = armw.RecorderShortcutRecordButton.SetText("Record shortcut")
			if !walk.RegisterGlobalHotKey(armw.MainWindow, 1, currentSettings.RecorderShortcut) {
				walk.MsgBox(armw.MainWindow, "Failed registering new hotkey", "failed registering new shortcut key, please restart application", walk.MsgBoxIconWarning)
				return
			}
			_ = writeConfig(currentSettings)
		}
	}

	armw.RecorderShortcutRecordButton.Clicked().Attach(shortcutRecorderHandler)

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
		err = writeConfig(currentSettings)
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

// readConfig reads configuration from json file, or creates one if file doesn't exist
func readConfig() (*captureConfig, error) {
	appDir, err := filepath.Abs(filepath.Dir(os.Args[0]))
	if err != nil {
		return nil, err
	}

	var c *captureConfig

	load := true
	settingsFilename := filepath.Join(appDir, "settings.json")

	_, err = os.Stat(settingsFilename)
	if os.IsNotExist(err) {
		// fetch current user folder and try to point to Gamelogs folder
		usr, errr := user.Current()
		if errr != nil {
			return nil, errr
		}

		eveGameLogsFolder := filepath.Join(usr.HomeDir, "Documents", "EVE", "logs", "Gamelogs")

		defaultShortcut := walk.Shortcut{Modifiers: walk.ModControl | walk.ModAlt, Key: walk.KeyEnd}

		c = &captureConfig{
			AppRoot:              appDir,
			X:                    10,
			Y:                    10,
			H:                    400,
			Recordings:           filepath.Join(appDir, "recordings"),
			FilterThreshold:      110,
			FilteredPreview:      false,
			EVEGameLogsFolder:    eveGameLogsFolder,
			RecorderShortcutText: defaultShortcut.String(),
			RecorderShortcut:     defaultShortcut,
		}
		load = false
	} else if err != nil {
		return c, err
	}

	if load {
		f, err := os.Open(settingsFilename)
		if err != nil {
			return c, err
		}
		defer f.Close()

		err = json.NewDecoder(f).Decode(&c)
		if err != nil {
			return c, err
		}
	}

	return c, nil
}

// writeConfig saves configuration to json file
func writeConfig(c *captureConfig) error {
	settingsFilename := filepath.Join(c.AppRoot, "settings.json")

	f, err := os.Create(settingsFilename)
	if err != nil {
		return err
	}
	defer f.Close()

	encoder := json.NewEncoder(f)
	encoder.SetIndent("", "  ")

	err = encoder.Encode(c)
	if err != nil {
		return err
	}

	return nil
}
