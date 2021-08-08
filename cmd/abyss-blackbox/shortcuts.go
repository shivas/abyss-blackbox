package main

import (
	"github.com/lxn/walk"
	"github.com/lxn/win"
)

// GetShortcutRecordingHandler creates handler for shortcut recording.
func GetShortcutRecordingHandler(
	e *walk.LineEdit,
	b *walk.PushButton,
	m *walk.MainWindow,
	id int,
	settings *captureConfig,
	shortcut walk.Shortcut,
) func() {
	return func() {
		if !e.Enabled() { // start recording
			e.SetEnabled(true)
			_ = e.SetFocus()
			_ = b.SetText("Save")

			if !win.UnregisterHotKey(m.Handle(), id) {
				walk.MsgBox(m, "Failed unregistering hotkey", "failed unregistering key, please restart application", walk.MsgBoxIconWarning)
				return
			}
		} else { // persist new shortcut and rebind
			e.SetEnabled(false)
			_ = b.SetText("Record shortcut")
			if !walk.RegisterGlobalHotKey(m, id, shortcut) {
				walk.MsgBox(m, "Failed registering new hotkey", "failed registering new shortcut key, please restart application", walk.MsgBoxIconWarning)
				return
			}
			_ = writeConfig(settings)
		}
	}
}
