//go:build windows && !386

package screen

import "syscall"

func init() {
	// We need to call SetProcessDpiAwareness so that Windows API calls will
	// tell us the scale factor for our monitor so that our screenshot works
	// on hi-res displays.
	// PROCESS_PER_MONITOR_DPI_AWARE
	procSetProcessDpiAwareness.Call(uintptr(2)) //nolint:errcheck // no needed
}

var modShcore = syscall.NewLazyDLL("Shcore.dll")
var procSetProcessDpiAwareness = modShcore.NewProc("SetProcessDpiAwareness")
