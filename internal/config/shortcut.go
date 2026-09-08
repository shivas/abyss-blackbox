package config

import (
	"strconv"
	"strings"
)

// Modifiers matches the historical walk.Modifiers bit layout used in settings.json.
const (
	ModShift Modifiers = 1 << iota
	ModControl
	ModAlt
)

type Modifiers byte

func (m Modifiers) String() string {
	var parts []string
	if m&ModAlt != 0 {
		parts = append(parts, "Alt")
	}
	if m&ModControl != 0 {
		parts = append(parts, "Ctrl")
	}
	if m&ModShift != 0 {
		parts = append(parts, "Shift")
	}

	return strings.Join(parts, "+")
}

// ToHotKey converts walk-style modifiers to Win32 RegisterHotKey MOD_* flags.
func (m Modifiers) ToHotKey() uint32 {
	var mod uint32
	if m&ModAlt != 0 {
		mod |= 0x0001 // MOD_ALT
	}
	if m&ModControl != 0 {
		mod |= 0x0002 // MOD_CONTROL
	}
	if m&ModShift != 0 {
		mod |= 0x0004 // MOD_SHIFT
	}

	return mod
}

// Key is a Win32 virtual-key code. JSON stays compatible with walk.Key.
type Key uint16

type Shortcut struct {
	Modifiers Modifiers
	Key       Key
}

func (s Shortcut) String() string {
	key := s.Key.String()
	if s.Modifiers == 0 {
		return key
	}

	return s.Modifiers.String() + "+" + key
}

func (k Key) String() string {
	if name, ok := keyNames[k]; ok {
		return name
	}

	return "Key" + strconv.Itoa(int(k))
}

// Common virtual-key codes used by the recorder and settings UI.
const (
	KeyEnd      Key = 0x23
	KeyInsert   Key = 0x2D
	KeyDelete   Key = 0x2E
	KeyHome     Key = 0x24
	KeyPrior    Key = 0x21
	KeyNext     Key = 0x22
	KeyLeft     Key = 0x25
	KeyUp       Key = 0x26
	KeyRight    Key = 0x27
	KeyDown     Key = 0x28
	KeySpace    Key = 0x20
	KeyEscape   Key = 0x1B
	KeyReturn   Key = 0x0D
	KeyTab      Key = 0x09
	KeyBack     Key = 0x08
	KeyF1       Key = 0x70
	KeyF2       Key = 0x71
	KeyF3       Key = 0x72
	KeyF4       Key = 0x73
	KeyF5       Key = 0x74
	KeyF6       Key = 0x75
	KeyF7       Key = 0x76
	KeyF8       Key = 0x77
	KeyF9       Key = 0x78
	KeyF10      Key = 0x79
	KeyF11      Key = 0x7A
	KeyF12      Key = 0x7B
	KeyNumpad0  Key = 0x60
	KeyNumpad1  Key = 0x61
	KeyNumpad2  Key = 0x62
	KeyNumpad3  Key = 0x63
	KeyNumpad4  Key = 0x64
	KeyNumpad5  Key = 0x65
	KeyNumpad6  Key = 0x66
	KeyNumpad7  Key = 0x67
	KeyNumpad8  Key = 0x68
	KeyNumpad9  Key = 0x69
	Key0        Key = 0x30
	Key1        Key = 0x31
	Key2        Key = 0x32
	Key3        Key = 0x33
	Key4        Key = 0x34
	Key5        Key = 0x35
	Key6        Key = 0x36
	Key7        Key = 0x37
	Key8        Key = 0x38
	Key9        Key = 0x39
	KeyA        Key = 0x41
	KeyB        Key = 0x42
	KeyC        Key = 0x43
	KeyD        Key = 0x44
	KeyE        Key = 0x45
	KeyF        Key = 0x46
	KeyG        Key = 0x47
	KeyH        Key = 0x48
	KeyI        Key = 0x49
	KeyJ        Key = 0x4A
	KeyK        Key = 0x4B
	KeyL        Key = 0x4C
	KeyM        Key = 0x4D
	KeyN        Key = 0x4E
	KeyO        Key = 0x4F
	KeyP        Key = 0x50
	KeyQ        Key = 0x51
	KeyR        Key = 0x52
	KeyS        Key = 0x53
	KeyT        Key = 0x54
	KeyU        Key = 0x55
	KeyV        Key = 0x56
	KeyW        Key = 0x57
	KeyX        Key = 0x58
	KeyY        Key = 0x59
	KeyZ        Key = 0x5A
)

var keyNames = map[Key]string{
	KeyEnd:     "End",
	KeyInsert:  "Insert",
	KeyDelete:  "Delete",
	KeyHome:    "Home",
	KeyPrior:   "PageUp",
	KeyNext:    "PageDown",
	KeyLeft:    "Left",
	KeyUp:      "Up",
	KeyRight:   "Right",
	KeyDown:    "Down",
	KeySpace:   "Space",
	KeyEscape:  "Escape",
	KeyReturn:  "Return",
	KeyTab:     "Tab",
	KeyBack:    "Back",
	KeyF1:      "F1",
	KeyF2:      "F2",
	KeyF3:      "F3",
	KeyF4:      "F4",
	KeyF5:      "F5",
	KeyF6:      "F6",
	KeyF7:      "F7",
	KeyF8:      "F8",
	KeyF9:      "F9",
	KeyF10:     "F10",
	KeyF11:     "F11",
	KeyF12:     "F12",
	KeyNumpad0: "Numpad0",
	KeyNumpad1: "Numpad1",
	KeyNumpad2: "Numpad2",
	KeyNumpad3: "Numpad3",
	KeyNumpad4: "Numpad4",
	KeyNumpad5: "Numpad5",
	KeyNumpad6: "Numpad6",
	KeyNumpad7: "Numpad7",
	KeyNumpad8: "Numpad8",
	KeyNumpad9: "Numpad9",
	Key0:       "0",
	Key1:       "1",
	Key2:       "2",
	Key3:       "3",
	Key4:       "4",
	Key5:       "5",
	Key6:       "6",
	Key7:       "7",
	Key8:       "8",
	Key9:       "9",
	KeyA:       "A",
	KeyB:       "B",
	KeyC:       "C",
	KeyD:       "D",
	KeyE:       "E",
	KeyF:       "F",
	KeyG:       "G",
	KeyH:       "H",
	KeyI:       "I",
	KeyJ:       "J",
	KeyK:       "K",
	KeyL:       "L",
	KeyM:       "M",
	KeyN:       "N",
	KeyO:       "O",
	KeyP:       "P",
	KeyQ:       "Q",
	KeyR:       "R",
	KeyS:       "S",
	KeyT:       "T",
	KeyU:       "U",
	KeyV:       "V",
	KeyW:       "W",
	KeyX:       "X",
	KeyY:       "Y",
	KeyZ:       "Z",
}

// Rectangle is stored in native screen pixels (same JSON as walk.Rectangle).
type Rectangle struct {
	X, Y, Width, Height int
}
