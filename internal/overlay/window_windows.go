//go:build windows

package overlay

import (
	"runtime"
	"syscall"
	"unsafe"

	"github.com/shivas/abyss-blackbox/internal/config"
	"golang.org/x/sys/windows"
)

var (
	modUser32 = windows.NewLazySystemDLL("user32.dll")
	modGdi32  = windows.NewLazySystemDLL("gdi32.dll")

	procRegisterClassExW  = modUser32.NewProc("RegisterClassExW")
	procCreateWindowExW   = modUser32.NewProc("CreateWindowExW")
	procDestroyWindow     = modUser32.NewProc("DestroyWindow")
	procDefWindowProcW    = modUser32.NewProc("DefWindowProcW")
	procGetMessageW       = modUser32.NewProc("GetMessageW")
	procTranslateMessage  = modUser32.NewProc("TranslateMessage")
	procDispatchMessageW  = modUser32.NewProc("DispatchMessageW")
	procPostMessageW      = modUser32.NewProc("PostMessageW")
	procPostQuitMessage   = modUser32.NewProc("PostQuitMessage")
	procInvalidateRect    = modUser32.NewProc("InvalidateRect")
	procGetClientRect     = modUser32.NewProc("GetClientRect")
	procGetWindowRect     = modUser32.NewProc("GetWindowRect")
	procClientToScreen    = modUser32.NewProc("ClientToScreen")
	procSetWindowPos      = modUser32.NewProc("SetWindowPos")
	procBeginPaint        = modUser32.NewProc("BeginPaint")
	procEndPaint          = modUser32.NewProc("EndPaint")
	procFillRect          = modUser32.NewProc("FillRect")
	procDrawTextW         = modUser32.NewProc("DrawTextW")
	procCreatePopupMenu   = modUser32.NewProc("CreatePopupMenu")
	procAppendMenuW       = modUser32.NewProc("AppendMenuW")
	procTrackPopupMenu    = modUser32.NewProc("TrackPopupMenu")
	procGetCursorPos      = modUser32.NewProc("GetCursorPos")
	procDestroyMenu       = modUser32.NewProc("DestroyMenu")
	procSetWindowLongPtrW = modUser32.NewProc("SetWindowLongPtrW")
	procGetWindowLongPtrW = modUser32.NewProc("GetWindowLongPtrW")
	procCreateSolidBrush  = modGdi32.NewProc("CreateSolidBrush")
	procDeleteObject      = modGdi32.NewProc("DeleteObject")
	procCreateFontW       = modGdi32.NewProc("CreateFontW")
	procSelectObject      = modGdi32.NewProc("SelectObject")
	procSetTextColor      = modGdi32.NewProc("SetTextColor")
	procSetBkMode         = modGdi32.NewProc("SetBkMode")
	procGetModuleHandleW  = windows.NewLazySystemDLL("kernel32.dll").NewProc("GetModuleHandleW")
)

const (
	wmDestroy      = 0x0002
	wmSize         = 0x0005
	wmPaint        = 0x000F
	wmClose        = 0x0010
	wmEraseBkgnd   = 0x0014
	wmCommand      = 0x0111
	wmContextMenu  = 0x007B
	wmUserInvalidate = 0x0401

	wsExNoActivate     = 0x08000000
	wsExTopmost        = 0x00000008
	wsExToolwindow     = 0x00000080
	wsPopup            = 0x80000000
	wsVisible          = 0x10000000
	wsOverlappedWindow = 0x00CF0000
	csHRedraw          = 0x0002
	csVRedraw          = 0x0001
	swpNoActivate      = 0x0010
	swpNoMove          = 0x0002
	swpNoSize          = 0x0001
	swpFrameChanged    = 0x0020
	hwndTopmost        = ^uintptr(0)
	gwlpUserData       = ^uintptr(20) // GWLP_USERDATA = -21
	dtLeft             = 0x0000
	dtWordbreak        = 0x0010
	transparentBk      = 1
	fwNormal           = 400
	fwBold             = 700
	defaultCharset     = 1
	outDefaultPrecis   = 0
	clipDefaultPrecis  = 0
	defaultQuality     = 0
	defaultPitch       = 0
	mfString           = 0x0000
	tpmRightButton     = 0x0002
	cmdClose           = 1
	cmdPlacement       = 2
	cmdSettings        = 3
	className          = "AbyssBlackboxOverlay"
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

type rect struct {
	left, top, right, bottom int32
}

type point struct {
	x, y int32
}

type paintStruct struct {
	hdc         uintptr
	erase       int32
	rcPaint     rect
	restore     int32
	incUpdate   int32
	rgbReserved [32]byte
}

var overlayWndProc = syscall.NewCallback(overlayWndProcImpl)

func (o *Overlay) Show() {
	o.mu.Lock()
	if o.visible {
		o.mu.Unlock()
		return
	}
	o.visible = true
	o.mu.Unlock()

	go o.loop()
}

func (o *Overlay) Close() {
	o.mu.Lock()
	hwnd := o.hwnd
	o.reopen = false
	o.skipModeFlip = true
	o.mu.Unlock()

	if hwnd != 0 {
		procPostMessageW.Call(hwnd, wmClose, 0, 0)
	}
}

func (o *Overlay) invalidate() {
	o.mu.Lock()
	hwnd := o.hwnd
	o.mu.Unlock()

	if hwnd != 0 {
		procInvalidateRect.Call(hwnd, 0, 1)
	}
}

func (o *Overlay) loop() {
	runtime.LockOSThread()
	defer runtime.UnlockOSThread()

	defer func() {
		o.mu.Lock()
		o.visible = false
		o.hwnd = 0
		reopen := o.reopen
		o.reopen = false
		o.mu.Unlock()

		if reopen {
			o.Show()
		}
	}()

	hInstance, _, _ := procGetModuleHandleW.Call(0)
	clsName, _ := syscall.UTF16PtrFromString(className)
	title, _ := syscall.UTF16PtrFromString("Overlay")

	cls := wndClassEx{
		style:         csHRedraw | csVRedraw,
		lpfnWndProc:   overlayWndProc,
		hInstance:     hInstance,
		lpszClassName: clsName,
	}
	cls.cbSize = uint32(unsafe.Sizeof(cls))
	procRegisterClassExW.Call(uintptr(unsafe.Pointer(&cls)))

	pos := o.captureConfig.OverlayPosition
	if pos.Width <= 0 {
		pos.Width = 280
	}
	if pos.Height <= 0 {
		pos.Height = 180
	}

	var exStyle, style uintptr
	if o.mode == ModePlacement {
		exStyle = wsExTopmost
		style = wsOverlappedWindow | wsVisible
	} else {
		exStyle = wsExTopmost | wsExNoActivate | wsExToolwindow
		style = wsPopup | wsVisible
	}

	hwnd, _, _ := procCreateWindowExW.Call(
		exStyle,
		uintptr(unsafe.Pointer(clsName)),
		uintptr(unsafe.Pointer(title)),
		style,
		uintptr(int32(pos.X)),
		uintptr(int32(pos.Y)),
		uintptr(int32(pos.Width)),
		uintptr(int32(pos.Height)),
		0, 0, hInstance, 0,
	)
	if hwnd == 0 {
		return
	}

	procSetWindowLongPtrW.Call(hwnd, gwlpUserData, uintptr(unsafe.Pointer(o)))

	o.mu.Lock()
	o.hwnd = hwnd
	o.mu.Unlock()

	if o.mode == ModeLive {
		procSetWindowPos.Call(hwnd, hwndTopmost, 0, 0, 0, 0, swpNoActivate|swpNoMove|swpNoSize|swpFrameChanged)
		setClientScreenRect(hwnd, pos)
	} else {
		setClientScreenRect(hwnd, pos)
	}

	procInvalidateRect.Call(hwnd, 0, 1)

	var m msg
	for {
		ret, _, _ := procGetMessageW.Call(uintptr(unsafe.Pointer(&m)), 0, 0, 0)
		if int32(ret) <= 0 {
			break
		}
		procTranslateMessage.Call(uintptr(unsafe.Pointer(&m)))
		procDispatchMessageW.Call(uintptr(unsafe.Pointer(&m)))
	}
}

func overlayFromHWND(hwnd uintptr) *Overlay {
	p, _, _ := procGetWindowLongPtrW.Call(hwnd, gwlpUserData)
	if p == 0 {
		return nil
	}

	return (*Overlay)(unsafe.Pointer(p))
}

func overlayWndProcImpl(hwnd uintptr, msg uint32, wParam, lParam uintptr) uintptr {
	o := overlayFromHWND(hwnd)

	switch msg {
	case wmEraseBkgnd:
		return 1
	case wmPaint:
		if o != nil {
			o.paint(hwnd)
		}
		return 0
	case wmContextMenu:
		if o != nil && o.mode == ModeLive {
			o.showMenu(hwnd)
		}
		return 0
	case wmCommand:
		if o != nil {
			o.handleCommand(int(wParam & 0xFFFF))
		}
		return 0
	case wmClose:
		if o != nil {
			o.onClosing(hwnd)
		}
		procDestroyWindow.Call(hwnd)
		return 0
	case wmDestroy:
		procPostQuitMessage.Call(0)
		return 0
	}

	r, _, _ := procDefWindowProcW.Call(hwnd, uintptr(msg), wParam, lParam)

	return r
}

func (o *Overlay) onClosing(hwnd uintptr) {
	pos := clientScreenRect(hwnd)
	o.captureConfig.OverlayPosition = pos
	_ = config.Write(o.captureConfig)

	o.mu.Lock()
	defer o.mu.Unlock()
	if o.skipModeFlip {
		o.skipModeFlip = false
		return
	}
	if o.mode == ModePlacement {
		o.mode = ModeLive
		o.reopen = true
	}
}

func (o *Overlay) handleCommand(id int) {
	switch id {
	case cmdClose:
		o.mu.Lock()
		o.reopen = false
		o.skipModeFlip = true
		hwnd := o.hwnd
		o.mu.Unlock()
		procPostMessageW.Call(hwnd, wmClose, 0, 0)
	case cmdPlacement:
		o.mu.Lock()
		o.mode = ModePlacement
		o.reopen = true
		o.skipModeFlip = true
		hwnd := o.hwnd
		o.mu.Unlock()
		procPostMessageW.Call(hwnd, wmClose, 0, 0)
	case cmdSettings:
		if o.OnSettings != nil {
			o.OnSettings()
		}
	}
}

func (o *Overlay) showMenu(hwnd uintptr) {
	menu, _, _ := procCreatePopupMenu.Call()
	if menu == 0 {
		return
	}
	defer procDestroyMenu.Call(menu)

	appendMenu(menu, cmdPlacement, "Switch to placement mode")
	appendMenu(menu, cmdSettings, "Customization settings")
	appendMenu(menu, cmdClose, "Close")

	var pt point
	procGetCursorPos.Call(uintptr(unsafe.Pointer(&pt)))
	procTrackPopupMenu.Call(menu, tpmRightButton, uintptr(pt.x), uintptr(pt.y), 0, hwnd, 0)
}

func appendMenu(menu uintptr, id int, text string) {
	p, _ := syscall.UTF16PtrFromString(text)
	procAppendMenuW.Call(menu, mfString, uintptr(id), uintptr(unsafe.Pointer(p)))
}

func (o *Overlay) paint(hwnd uintptr) {
	var ps paintStruct
	hdc, _, _ := procBeginPaint.Call(hwnd, uintptr(unsafe.Pointer(&ps)))
	if hdc == 0 {
		return
	}
	defer procEndPaint.Call(hwnd, uintptr(unsafe.Pointer(&ps)))

	var rc rect
	procGetClientRect.Call(hwnd, uintptr(unsafe.Pointer(&rc)))

	brush, _, _ := procCreateSolidBrush.Call(uintptr(o.config.BackgroundColor.COLORREF()))
	if brush != 0 {
		procFillRect.Call(hdc, uintptr(unsafe.Pointer(&rc)), brush)
		procDeleteObject.Call(brush)
	}

	procSetBkMode.Call(hdc, transparentBk)

	bounds := rc
	bounds.left += 5
	bounds.top += 5
	bounds.right -= 5
	bounds.bottom -= 5

	headerFont := createFont(o.config.FontFamily, 9, fwBold)
	bodyFont := createFont(o.config.FontFamily, o.config.FontSize, fwNormal)
	defer procDeleteObject.Call(headerFont)
	defer procDeleteObject.Call(bodyFont)

	drawLine(hdc, headerFont, PrimaryColor, "abyssal.space telemetry overlay", &bounds)
	bounds.top += int32(o.config.Spacing)

	o.overlayState.Lock()
	defer o.overlayState.Unlock()

	for _, prop := range []WidgetProperty{Status, Weather, TODO, Override, Autoupload} {
		item := o.overlayState.items[prop]
		col := o.config.Color
		if item.color != nil {
			col = *item.color
		}
		drawLine(hdc, bodyFont, col, item.text, &bounds)
		bounds.top += int32(o.config.Spacing)
	}
}

func createFont(family string, size, weight int) uintptr {
	if size <= 0 {
		size = 9
	}
	if family == "" {
		family = "Verdana"
	}

	height := -int32(size * 96 / 72)
	name, _ := syscall.UTF16PtrFromString(family)
	h, _, _ := procCreateFontW.Call(
		uintptr(height), 0, 0, 0,
		uintptr(weight),
		0, 0, 0,
		defaultCharset,
		outDefaultPrecis,
		clipDefaultPrecis,
		defaultQuality,
		defaultPitch,
		uintptr(unsafe.Pointer(name)),
	)

	return h
}

func drawLine(hdc, font uintptr, color Color, text string, bounds *rect) {
	if text == "" {
		return
	}

	procSelectObject.Call(hdc, font)
	procSetTextColor.Call(hdc, uintptr(color.COLORREF()))
	p, _ := syscall.UTF16PtrFromString(text)
	r := *bounds
	procDrawTextW.Call(hdc, uintptr(unsafe.Pointer(p)), ^uintptr(0), uintptr(unsafe.Pointer(&r)), dtLeft|dtWordbreak)
}

func clientScreenRect(hwnd uintptr) config.Rectangle {
	var cr rect
	procGetClientRect.Call(hwnd, uintptr(unsafe.Pointer(&cr)))

	origin := point{}
	procClientToScreen.Call(hwnd, uintptr(unsafe.Pointer(&origin)))

	return config.Rectangle{
		X:      int(origin.x),
		Y:      int(origin.y),
		Width:  int(cr.right - cr.left),
		Height: int(cr.bottom - cr.top),
	}
}

func chromeInsets(hwnd uintptr) (left, top, right, bottom int) {
	var wr rect
	procGetWindowRect.Call(hwnd, uintptr(unsafe.Pointer(&wr)))
	client := clientScreenRect(hwnd)

	left = client.X - int(wr.left)
	top = client.Y - int(wr.top)
	right = int(wr.right-wr.left) - client.Width - left
	bottom = int(wr.bottom-wr.top) - client.Height - top

	return
}

func setClientScreenRect(hwnd uintptr, client config.Rectangle) {
	if client.Width <= 0 || client.Height <= 0 {
		return
	}

	left, top, right, bottom := chromeInsets(hwnd)
	procSetWindowPos.Call(
		hwnd, hwndTopmost,
		uintptr(int32(client.X-left)),
		uintptr(int32(client.Y-top)),
		uintptr(int32(client.Width+left+right)),
		uintptr(int32(client.Height+top+bottom)),
		swpNoActivate,
	)
}
