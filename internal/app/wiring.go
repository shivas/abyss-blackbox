package app

import (
	"errors"
	"image"
	"log/slog"
	"os"
	"sync"

	"github.com/shivas/abyss-blackbox/internal/app/api/client"
	"github.com/shivas/abyss-blackbox/internal/app/domain"
	"github.com/shivas/abyss-blackbox/internal/app/recorder"
	"github.com/shivas/abyss-blackbox/internal/charmanager"
	"github.com/shivas/abyss-blackbox/internal/config"
	"github.com/shivas/abyss-blackbox/internal/fittings"
	"github.com/shivas/abyss-blackbox/internal/fittings/provider"
	"github.com/shivas/abyss-blackbox/internal/overlay"
	"github.com/shivas/abyss-blackbox/internal/ui"
	"github.com/shivas/abyss-blackbox/internal/win32"
	"github.com/shivas/abyss-blackbox/internal/window"
	"github.com/shivas/abyss-blackbox/pkg/combatlog"
)

func Run() (err error) {
	slog.Info("loading settings")
	currentSettings, err := config.Read()
	if err != nil {
		slog.Error("failed loading settings", slog.Any("err", err))
		return err
	}

	recordingChannel := make(chan *image.Paletted, 10)
	notificationChannel := make(chan domain.NotificationMessage, 10)

	slog.Info("enumerating EVE windows")
	windowsManager, err := window.NewManager()
	if err != nil && !errors.Is(err, window.ErrNoWindowsFound) {
		slog.Error("window manager initialization failed", slog.Any("err", err))
		return err
	}

	overlayManager := overlay.New(
		overlay.FromCaptureConfig(currentSettings),
		currentSettings,
	)
	defer overlayManager.Close()

	overlayManager.ChangeProperty(overlay.Status, "Recorder on standby", &overlay.YellowColor)

	clr := combatlog.NewReader(currentSettings.EVEGameLogsFolder)
	rec := recorder.NewRecorder(recordingChannel, currentSettings, notificationChannel, clr, overlayManager)
	rec.StartLoop()
	defer rec.StopLoop()

	charManager := charmanager.New(func(s1, s2 string) {
		notificationChannel <- domain.NotificationMessage{Title: s1, Message: s2}
	})
	_ = charManager.LoadCache()

	authenticatedHTTPClient := client.New(charManager)
	fittingsManager := fittings.NewManager(authenticatedHTTPClient, provider.NewTrackerFittingsProvider(authenticatedHTTPClient))

	var (
		eventsMu sync.Mutex
		events   *win32.NativeEvents
	)

	closeEvents := func() {
		eventsMu.Lock()
		e := events
		events = nil
		eventsMu.Unlock()
		if e != nil {
			e.Close()
		}
	}
	defer closeEvents()

	var fyneUI *ui.App

	bindHotkeys := func(e *win32.NativeEvents) {
		if e == nil {
			return
		}
		_ = e.RegisterHotKey(config.HotkeyRecoder, currentSettings.RecorderShortcut, func() {
			if fyneUI != nil {
				fyneUI.ToggleRecording()
			}
		})
		_ = e.RegisterHotKey(config.HotkeyWeather30, currentSettings.Weather30Shortcut, func() {
			rec.GetWeatherStrengthListener(30)()
		})
		_ = e.RegisterHotKey(config.HotkeyWeather50, currentSettings.Weather50Shortcut, func() {
			rec.GetWeatherStrengthListener(50)()
		})
		_ = e.RegisterHotKey(config.HotkeyWeather70, currentSettings.Weather70Shortcut, func() {
			rec.GetWeatherStrengthListener(70)()
		})
		_ = e.RegisterHotKey(config.Overlay, currentSettings.OverlayShortcut, func() {
			overlayManager.ToggleOverlay()
		})
	}

	slog.Info("creating fyne window")
	fyneUI = ui.New(ui.Deps{
		Config:     currentSettings,
		Recorder:   rec,
		Overlay:    overlayManager,
		Chars:      charManager,
		Fittings:   fittingsManager,
		Windows:    windowsManager,
		CombatLog:  clr,
		HTTPClient: authenticatedHTTPClient,
		Recording:  recordingChannel,
		RebindHotkeys: func() {
			eventsMu.Lock()
			e := events
			eventsMu.Unlock()
			bindHotkeys(e)
		},
		StartShortcutCapture: func(fn func(config.Shortcut)) {
			eventsMu.Lock()
			e := events
			eventsMu.Unlock()
			if e != nil {
				e.StartKeyCapture(fn)
			}
		},
		StopShortcutCapture: func() {
			eventsMu.Lock()
			e := events
			eventsMu.Unlock()
			if e != nil {
				e.StopKeyCapture()
			}
		},
		OnClose: func() {
			overlayManager.Close()
			closeEvents()
			if werr := config.Write(currentSettings); werr != nil {
				slog.Error("failed to save settings after main window close", slog.Any("err", werr))
				os.Exit(1)
			}
		},
	})

	charManager.OnActivateCharacter = func(char charmanager.Character) {
		if char.CharacterID > 0 {
			notificationChannel <- domain.NotificationMessage{Title: "Active character set to:", Message: char.CharacterName}
		}

		currentSettings.ActiveCharacter = char.CharacterID
		_ = config.Write(currentSettings)

		if fyneUI != nil {
			fyneUI.RefreshActiveCharacter()
		}
	}

	_ = charManager.SetActiveCharacter(currentSettings.ActiveCharacter)

	go func() {
		for msg := range notificationChannel {
			if fyneUI != nil {
				fyneUI.Notify(msg.Title, msg.Message)
			}
		}
	}()

	// Win32 GetMessage/hotkey window must start after GLFW has created its window.
	// Starting it first can hang glfw.CreateWindow on Windows (no UI appears).
	fyneUI.App().Lifecycle().SetOnStarted(func() {
		slog.Info("fyne started, creating native hotkey/clipboard window")
		e, nerr := win32.NewNativeEvents()
		if nerr != nil {
			slog.Error("native events window failed", slog.Any("err", nerr))
			return
		}
		e.SetClipboardHandler(rec.OnClipboardText)
		eventsMu.Lock()
		events = e
		eventsMu.Unlock()
		bindHotkeys(e)
	})

	slog.Info("showing main window")
	fyneUI.ShowAndRun()

	return nil
}
