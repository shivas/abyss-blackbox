//go:build windows

package win32

import (
	"errors"
	"log/slog"
	"runtime"
	"sync"
	"sync/atomic"
	"syscall"
	"time"
	"unsafe"

	"github.com/shivas/abyss-blackbox/internal/config"
)

// NativeEvents is a hidden message-only window that receives WM_HOTKEY and clipboard updates.
type NativeEvents struct {
	mu           sync.Mutex
	hwnd         uintptr
	hotkeys      map[int]hotkeyBinding
	onClipboard  func(string)
	onKeyCapture func(config.Shortcut)
	hook         uintptr
	ready        chan struct{}
	done         chan struct{}
}

type hotkeyBinding struct {
	shortcut config.Shortcut
	handler  func()
}

type kbdllhook struct {
	vkCode      uint32
	scanCode    uint32
	flags       uint32
	time        uint32
	dwExtraInfo uintptr
}

// NewNativeEvents starts a dedicated OS thread with a message-only window.
func NewNativeEvents() (*NativeEvents, error) {
	n := &NativeEvents{
		hotkeys: make(map[int]hotkeyBinding),
		ready:   make(chan struct{}),
		done:    make(chan struct{}),
	}

	go n.loop()

	select {
	case <-n.ready:
		if n.hwnd == 0 {
			return nil, errors.New("failed to create native events window")
		}
		return n, nil
	case <-time.After(3 * time.Second):
		return nil, errors.New("timeout creating native events window")
	}
}

func (n *NativeEvents) SetClipboardHandler(fn func(string)) {
	n.mu.Lock()
	defer n.mu.Unlock()
	n.onClipboard = fn
}

func (n *NativeEvents) RegisterHotKey(id int, s config.Shortcut, handler func()) error {
	n.mu.Lock()
	n.hotkeys[id] = hotkeyBinding{shortcut: s, handler: handler}
	hwnd := n.hwnd
	n.mu.Unlock()
	if hwnd == 0 {
		return errors.New("native events window not ready")
	}

	procPostMessageW.Call(hwnd, wmAppRegisterHotkey, uintptr(id), 0)
	return nil
}

func (n *NativeEvents) UnregisterHotKey(id int) {
	n.mu.Lock()
	delete(n.hotkeys, id)
	hwnd := n.hwnd
	n.mu.Unlock()
	if hwnd != 0 {
		procPostMessageW.Call(hwnd, wmAppUnregisterHotkey, uintptr(id), 0)
	}
}

func (n *NativeEvents) StartKeyCapture(fn func(config.Shortcut)) {
	n.mu.Lock()
	n.onKeyCapture = fn
	hwnd := n.hwnd
	n.mu.Unlock()
	if hwnd != 0 {
		procPostMessageW.Call(hwnd, wmAppStartCapture, 0, 0)
	}
}

func (n *NativeEvents) StopKeyCapture() {
	n.mu.Lock()
	n.onKeyCapture = nil
	hwnd := n.hwnd
	n.mu.Unlock()
	if hwnd != 0 {
		procPostMessageW.Call(hwnd, wmAppStopCapture, 0, 0)
	}
}

func (n *NativeEvents) Close() {
	n.StopKeyCapture()

	n.mu.Lock()
	hwnd := n.hwnd
	n.mu.Unlock()

	if hwnd != 0 {
		procPostMessageW.Call(hwnd, wmClose, 0, 0)
	}

	select {
	case <-n.done:
	case <-time.After(2 * time.Second):
	}
}

func (n *NativeEvents) loop() {
	runtime.LockOSThread()
	defer runtime.UnlockOSThread()
	defer close(n.done)

	className := utf16Ptr("AbyssBlackboxNativeEvents")
	wndProc := syscall.NewCallback(wndProcDispatch)

	cls := wndClassEx{
		lpfnWndProc:   wndProc,
		hInstance:     moduleHandle(),
		lpszClassName: className,
	}
	cls.cbSize = uint32(unsafe.Sizeof(cls))

	atom, _, err := procRegisterClassExW.Call(uintptr(unsafe.Pointer(&cls)))
	if atom == 0 {
		slog.Error("RegisterClassEx failed", slog.Any("err", err))
	}

	hwnd, _, err := procCreateWindowExW.Call(
		0,
		uintptr(unsafe.Pointer(className)),
		uintptr(unsafe.Pointer(utf16Ptr("AbyssBlackboxEvents"))),
		0,
		0, 0, 0, 0,
		hwndMessage,
		0,
		cls.hInstance,
		0,
	)
	if hwnd == 0 {
		slog.Error("CreateWindowEx message window failed", slog.Any("err", err))
	}

	n.mu.Lock()
	n.hwnd = hwnd
	if hwnd != 0 {
		nativeEventsByHWND.Store(hwnd, n)
	}
	n.mu.Unlock()
	close(n.ready)

	if hwnd == 0 {
		return
	}

	procAddClipboardFormatListener.Call(hwnd)

	var m msg
	for {
		// hwnd=0 is required so this thread also receives WH_KEYBOARD_LL hook messages.
		// This loop runs on a dedicated OS thread, not the GLFW thread.
		ret, _, _ := procGetMessageW.Call(uintptr(unsafe.Pointer(&m)), 0, 0, 0)
		if int32(ret) <= 0 {
			break
		}
		procTranslateMessage.Call(uintptr(unsafe.Pointer(&m)))
		procDispatchMessageW.Call(uintptr(unsafe.Pointer(&m)))
	}

	n.unhookKeyboard()
	nativeEventsByHWND.Delete(hwnd)
}

var nativeEventsByHWND sync.Map

func wndProcDispatch(hwnd, msg, wParam, lParam uintptr) uintptr {
	if v, ok := nativeEventsByHWND.Load(hwnd); ok {
		return v.(*NativeEvents).handleMessage(hwnd, msg, wParam, lParam)
	}

	return defWindowProc(hwnd, uint32(msg), wParam, lParam)
}

func (n *NativeEvents) handleMessage(hwnd, msg, wParam, lParam uintptr) uintptr {
	switch msg {
	case wmHotkey:
		n.mu.Lock()
		fn := n.hotkeys[int(wParam)].handler
		n.mu.Unlock()
		if fn != nil {
			fn()
		}
		return 0
	case wmAppRegisterHotkey:
		n.applyHotkey(hwnd, int(wParam))
		return 0
	case wmAppUnregisterHotkey:
		procUnregisterHotKey.Call(hwnd, wParam)
		return 0
	case wmAppStartCapture:
		n.hookKeyboard()
		return 0
	case wmAppStopCapture:
		n.unhookKeyboard()
		return 0
	case wmClipboardUpdate:
		n.mu.Lock()
		fn := n.onClipboard
		n.mu.Unlock()
		if fn != nil {
			if text := getClipboardText(); text != "" {
				fn(text)
			}
		}
		return 0
	case wmClose:
		n.unhookKeyboard()
		procDestroyWindow.Call(hwnd)
		return 0
	case wmDestroy:
		postQuit()
		return 0
	}

	return defWindowProc(hwnd, uint32(msg), wParam, lParam)
}

func (n *NativeEvents) applyHotkey(hwnd uintptr, id int) {
	n.mu.Lock()
	bind, ok := n.hotkeys[id]
	n.mu.Unlock()
	if !ok || bind.shortcut.Key == 0 {
		procUnregisterHotKey.Call(hwnd, uintptr(id))
		return
	}

	procUnregisterHotKey.Call(hwnd, uintptr(id))
	r, _, err := procRegisterHotKey.Call(hwnd, uintptr(id), uintptr(bind.shortcut.Modifiers.ToHotKey()), uintptr(bind.shortcut.Key))
	if r == 0 {
		slog.Error("RegisterHotKey failed",
			"id", id,
			"shortcut", bind.shortcut.String(),
			slog.Any("err", err),
		)
		return
	}

	slog.Info("registered hotkey", "id", id, "shortcut", bind.shortcut.String())
}

var (
	keyboardHookCB = syscall.NewCallback(keyboardLLProc)
	hookOwner      atomic.Pointer[NativeEvents]
)

func (n *NativeEvents) hookKeyboard() {
	if n.hook != 0 {
		return
	}

	hookOwner.Store(n)
	h, _, err := procSetWindowsHookExW.Call(whKeyboardLL, keyboardHookCB, moduleHandle(), 0)
	if h == 0 {
		slog.Error("SetWindowsHookEx keyboard capture failed", slog.Any("err", err))
		hookOwner.Store(nil)
		return
	}

	n.hook = h
}

func (n *NativeEvents) unhookKeyboard() {
	if n.hook == 0 {
		return
	}

	procUnhookWindowsHookEx.Call(n.hook)
	n.hook = 0
	hookOwner.Store(nil)
}

func keyboardLLProc(nCode, wParam, lParam uintptr) uintptr {
	n := hookOwner.Load()
	if n != nil && int32(nCode) >= 0 && (wParam == wmKeyDown || wParam == wmSysKeyDown) && lParam != 0 {
		kbd := (*kbdllhook)(unsafe.Pointer(lParam))
		if !isModifierVK(kbd.vkCode) {
			s := shortcutFromVK(config.Key(kbd.vkCode))
			n.mu.Lock()
			fn := n.onKeyCapture
			n.mu.Unlock()
			if fn != nil {
				fn(s)
			}
			return 1
		}
	}

	r, _, _ := procCallNextHookEx.Call(0, nCode, wParam, lParam)
	return r
}

func isModifierVK(vk uint32) bool {
	switch vk {
	case vkShift, vkControl, vkMenu, vkLWin, vkRWin, vkLShift, vkRShift, vkLControl, vkRControl, vkLMenu, vkRMenu:
		return true
	default:
		return false
	}
}

func shortcutFromVK(key config.Key) config.Shortcut {
	var mods config.Modifiers
	if keyDown(vkMenu) || keyDown(vkLMenu) || keyDown(vkRMenu) {
		mods |= config.ModAlt
	}
	if keyDown(vkControl) || keyDown(vkLControl) || keyDown(vkRControl) {
		mods |= config.ModControl
	}
	if keyDown(vkShift) || keyDown(vkLShift) || keyDown(vkRShift) {
		mods |= config.ModShift
	}

	return config.Shortcut{Modifiers: mods, Key: key}
}

func keyDown(vk uintptr) bool {
	r, _, _ := procGetAsyncKeyState.Call(vk)
	return r&0x8000 != 0
}
