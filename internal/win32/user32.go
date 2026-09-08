//go:build windows

package win32

import (
	"syscall"
	"unsafe"

	"golang.org/x/sys/windows"
)

var (
	modUser32   = windows.NewLazySystemDLL("user32.dll")
	modKernel32 = windows.NewLazySystemDLL("kernel32.dll")

	procRegisterClassExW           = modUser32.NewProc("RegisterClassExW")
	procCreateWindowExW            = modUser32.NewProc("CreateWindowExW")
	procDestroyWindow              = modUser32.NewProc("DestroyWindow")
	procDefWindowProcW             = modUser32.NewProc("DefWindowProcW")
	procGetMessageW                = modUser32.NewProc("GetMessageW")
	procTranslateMessage           = modUser32.NewProc("TranslateMessage")
	procDispatchMessageW           = modUser32.NewProc("DispatchMessageW")
	procPostMessageW               = modUser32.NewProc("PostMessageW")
	procPostQuitMessage            = modUser32.NewProc("PostQuitMessage")
	procRegisterHotKey             = modUser32.NewProc("RegisterHotKey")
	procUnregisterHotKey           = modUser32.NewProc("UnregisterHotKey")
	procAddClipboardFormatListener = modUser32.NewProc("AddClipboardFormatListener")
	procSetWindowsHookExW          = modUser32.NewProc("SetWindowsHookExW")
	procUnhookWindowsHookEx        = modUser32.NewProc("UnhookWindowsHookEx")
	procCallNextHookEx             = modUser32.NewProc("CallNextHookEx")
	procGetAsyncKeyState           = modUser32.NewProc("GetAsyncKeyState")
	procOpenClipboard              = modUser32.NewProc("OpenClipboard")
	procCloseClipboard             = modUser32.NewProc("CloseClipboard")
	procGetClipboardData           = modUser32.NewProc("GetClipboardData")
	procGetModuleHandleW           = modKernel32.NewProc("GetModuleHandleW")
	procGlobalLock                 = modKernel32.NewProc("GlobalLock")
	procGlobalUnlock               = modKernel32.NewProc("GlobalUnlock")
)

const (
	wmDestroy             = 0x0002
	wmClose               = 0x0010
	wmKeyDown             = 0x0100
	wmSysKeyDown          = 0x0104
	wmHotkey              = 0x0312
	wmClipboardUpdate     = 0x031D
	wmApp                 = 0x8000
	wmAppRegisterHotkey   = wmApp + 1
	wmAppUnregisterHotkey = wmApp + 2
	wmAppStartCapture     = wmApp + 3
	wmAppStopCapture      = wmApp + 4
	whKeyboardLL          = 13
	vkShift               = 0x10
	vkControl             = 0x11
	vkMenu                = 0x12
	vkLWin                = 0x5B
	vkRWin                = 0x5C
	vkLShift              = 0xA0
	vkRShift              = 0xA1
	vkLControl            = 0xA2
	vkRControl            = 0xA3
	vkLMenu               = 0xA4
	vkRMenu               = 0xA5
	hwndMessage           = ^uintptr(2) // HWND_MESSAGE = -3
	cfUnicodeText         = 13
	wsExNoActivate        = 0x08000000
	wsExTopmost           = 0x00000008
	wsExToolwindow        = 0x00000080
	wsExAppwindow         = 0x00040000
	wsPopup               = 0x80000000
	wsVisible             = 0x10000000
	wsOverlappedWindow    = 0x00CF0000
	csHRedraw             = 0x0002
	csVRedraw             = 0x0001
	swpNoActivate         = 0x0010
	swpNoMove             = 0x0002
	swpNoSize             = 0x0001
	swpFrameChanged       = 0x0020
	hwndTopmost           = ^uintptr(0) // HWND_TOPMOST = -1
)

type wndClassEx struct {
	cbSize        uint32
	style         uint32
	lpfnWndProc   uintptr
	cbClsExtra    int32
	cbWndExtra    int32
	hInstance     uintptr
	hIcon         uintptr
	hCursor       uintptr
	hbrBackground uintptr
	lpszMenuName  *uint16
	lpszClassName *uint16
	hIconSm       uintptr
}

type msg struct {
	hwnd     uintptr
	message  uint32
	_        uint32
	wParam   uintptr
	lParam   uintptr
	time     uint32
	ptX      int32
	ptY      int32
	lPrivate uint32
}

func moduleHandle() uintptr {
	h, _, _ := procGetModuleHandleW.Call(0)
	return h
}

func utf16Ptr(s string) *uint16 {
	p, err := syscall.UTF16PtrFromString(s)
	if err != nil {
		p, _ = syscall.UTF16PtrFromString("abyss")
	}

	return p
}

func defWindowProc(hwnd uintptr, msg uint32, wParam, lParam uintptr) uintptr {
	r, _, _ := procDefWindowProcW.Call(hwnd, uintptr(msg), wParam, lParam)
	return r
}

func postQuit() {
	procPostQuitMessage.Call(0)
}

func getClipboardText() string {
	r, _, _ := procOpenClipboard.Call(0)
	if r == 0 {
		return ""
	}
	defer procCloseClipboard.Call()

	h, _, _ := procGetClipboardData.Call(cfUnicodeText)
	if h == 0 {
		return ""
	}

	ptr, _, _ := procGlobalLock.Call(h)
	if ptr == 0 {
		return ""
	}
	defer procGlobalUnlock.Call(h)

	return windows.UTF16PtrToString((*uint16)(unsafe.Pointer(ptr)))
}
