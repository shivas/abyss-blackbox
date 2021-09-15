package config

import (
	"encoding/json"
	"os"
	"os/user"
	"path/filepath"
	"sync"

	"github.com/lxn/walk"
)

const (
	HotkeyRecoder = iota + 1
	HotkeyWeather30
	HotkeyWeather50
	HotkeyWeather70
)

type Preset struct {
	X, Y, H int
}

type CaptureConfig struct {
	sync.Mutex
	X, Y, H                 int
	Presets                 map[string]Preset
	AppRoot                 string
	Recordings              string
	FilterThreshold         int
	FilteredPreview         bool
	EVEClientWindowTitle    string
	EVEGameLogsFolder       string
	TestServer              bool
	RecorderShortcutText    string
	RecorderShortcut        walk.Shortcut
	Weather30ShortcutText   string
	Weather30Shortcut       walk.Shortcut
	Weather50ShortcutText   string
	Weather50Shortcut       walk.Shortcut
	Weather70ShortcutText   string
	Weather70Shortcut       walk.Shortcut
	LootRecordDiscriminator string
	ActiveCharacter         int32
	AutoUpload              bool
	AbyssTypeOverride       bool
	AbyssShipType           int
	AbyssTier               int
	AbyssWeather            string
}

// SetRecorderShortcut satisfies ShortcutSetter interface.
func (c *CaptureConfig) SetRecorderShortcut(shorcutType int, s walk.Shortcut) {
	switch shorcutType {
	case HotkeyRecoder:
		c.RecorderShortcut = s
		c.RecorderShortcutText = s.String()

	case HotkeyWeather30:
		c.Weather30Shortcut = s
		c.Weather30ShortcutText = s.String()

	case HotkeyWeather50:
		c.Weather50Shortcut = s
		c.Weather50ShortcutText = s.String()

	case HotkeyWeather70:
		c.Weather70Shortcut = s
		c.Weather70ShortcutText = s.String()
	}
}

// Read reads configuration from json file, or creates one if file doesn't exist
func Read() (*CaptureConfig, error) {
	appDir, err := filepath.Abs(filepath.Dir(os.Args[0]))
	if err != nil {
		return nil, err
	}

	var c *CaptureConfig

	load := true
	settingsFilename := filepath.Join(appDir, "settings.json")

	defaultRecorderShortcut := walk.Shortcut{Modifiers: walk.ModControl | walk.ModAlt, Key: walk.KeyEnd}
	defaultWeather30Shortcut := walk.Shortcut{Modifiers: walk.ModControl | walk.ModAlt, Key: walk.KeyNumpad3}
	defaultWeather50Shortcut := walk.Shortcut{Modifiers: walk.ModControl | walk.ModAlt, Key: walk.KeyNumpad5}
	defaultWeather70Shortcut := walk.Shortcut{Modifiers: walk.ModControl | walk.ModAlt, Key: walk.KeyNumpad7}

	_, err = os.Stat(settingsFilename)
	if os.IsNotExist(err) {
		// fetch current user folder and try to point to Gamelogs folder
		usr, errr := user.Current()
		if errr != nil {
			return nil, errr
		}

		eveGameLogsFolder := filepath.Join(usr.HomeDir, "Documents", "EVE", "logs", "Gamelogs")

		c = &CaptureConfig{
			AppRoot:                 appDir,
			X:                       10,
			Y:                       10,
			H:                       400,
			Presets:                 make(map[string]Preset),
			Recordings:              filepath.Join(appDir, "recordings"),
			FilterThreshold:         110,
			FilteredPreview:         false,
			AbyssTypeOverride:       false,
			EVEGameLogsFolder:       eveGameLogsFolder,
			RecorderShortcutText:    defaultRecorderShortcut.String(),
			RecorderShortcut:        defaultRecorderShortcut,
			Weather30ShortcutText:   defaultWeather30Shortcut.String(),
			Weather30Shortcut:       defaultWeather30Shortcut,
			Weather50ShortcutText:   defaultWeather50Shortcut.String(),
			Weather50Shortcut:       defaultWeather50Shortcut,
			Weather70ShortcutText:   defaultWeather70Shortcut.String(),
			Weather70Shortcut:       defaultWeather70Shortcut,
			LootRecordDiscriminator: "Quafe",
			AbyssShipType:           1,
			AbyssTier:               0,
			AbyssWeather:            "Dark",
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

		if c.RecorderShortcutText == "" {
			c.RecorderShortcut = defaultRecorderShortcut
			c.RecorderShortcutText = defaultRecorderShortcut.String()
		}

		if c.Weather30ShortcutText == "" {
			c.Weather30Shortcut = defaultWeather30Shortcut
			c.Weather30ShortcutText = defaultWeather30Shortcut.String()
		}

		if c.Weather50ShortcutText == "" {
			c.Weather50Shortcut = defaultWeather50Shortcut
			c.Weather50ShortcutText = defaultWeather50Shortcut.String()
		}

		if c.Weather70ShortcutText == "" {
			c.Weather70Shortcut = defaultWeather70Shortcut
			c.Weather70ShortcutText = defaultWeather70Shortcut.String()
		}

		if c.LootRecordDiscriminator == "" {
			c.LootRecordDiscriminator = "Quafe"
		}

		if c.AbyssShipType == 0 {
			c.AbyssShipType = 1
		}

		if c.AbyssWeather == "" {
			c.AbyssWeather = "Dark"
		}
	}

	return c, nil
}

// Write saves configuration to json file
func Write(c *CaptureConfig) error {
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
