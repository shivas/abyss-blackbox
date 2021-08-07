// +build windows

package screen

import (
	"fmt"
	"image"
	"reflect"
	"regexp"
	"syscall"
	"unsafe"

	"github.com/disintegration/gift"
	"golang.org/x/sys/windows"
)

func init() {
	// We need to call SetProcessDpiAwareness so that Windows API calls will
	// tell us the scale factor for our monitor so that our screenshot works
	// on hi-res displays.
	// PROCESS_PER_MONITOR_DPI_AWARE
	procSetProcessDpiAwareness.Call(uintptr(2)) //nolint:errcheck // no needed
}

func CaptureWindowArea(handle syscall.Handle, rect image.Rectangle) (image.Image, error) {
	return captureWindow(handle, rect)
}

var (
	modUser32          = syscall.NewLazyDLL("User32.dll")
	procGetClientRect  = modUser32.NewProc("GetClientRect")
	procGetDC          = modUser32.NewProc("GetDC")
	procReleaseDC      = modUser32.NewProc("ReleaseDC")
	procEnumWindows    = modUser32.NewProc("EnumWindows")
	procGetWindowTextW = modUser32.NewProc("GetWindowTextW")

	modGdi32   = syscall.NewLazyDLL("Gdi32.dll")
	procBitBlt = modGdi32.NewProc("BitBlt")

	// procCreateCompatibleBitmap = modGdi32.NewProc("CreateCompatibleBitmap")
	procCreateCompatibleDC = modGdi32.NewProc("CreateCompatibleDC")
	procCreateDIBSection   = modGdi32.NewProc("CreateDIBSection")
	procDeleteDC           = modGdi32.NewProc("DeleteDC")
	procDeleteObject       = modGdi32.NewProc("DeleteObject")

	// procGetDeviceCaps          = modGdi32.NewProc("GetDeviceCaps")
	procSelectObject = modGdi32.NewProc("SelectObject")

	modShcore                  = syscall.NewLazyDLL("Shcore.dll")
	procSetProcessDpiAwareness = modShcore.NewProc("SetProcessDpiAwareness")
)

const (
	// BitBlt constants
	bitBltSRCCOPY = 0x00CC0020
)

// Windows RECT structure
type winRect struct {
	Left, Top, Right, Bottom int32
}

// http://msdn.microsoft.com/en-us/library/windows/desktop/dd183375.aspx
type winBITMAPINFO struct {
	BmiHeader winBITMAPINFOHEADER
	BmiColors *winRGBQUAD
}

// http://msdn.microsoft.com/en-us/library/windows/desktop/dd183376.aspx
type winBITMAPINFOHEADER struct {
	BiSize          uint32
	BiWidth         int32
	BiHeight        int32
	BiPlanes        uint16
	BiBitCount      uint16
	BiCompression   uint32
	BiSizeImage     uint32
	BiXPelsPerMeter int32
	BiYPelsPerMeter int32
	BiClrUsed       uint32
	BiClrImportant  uint32
}

// http://msdn.microsoft.com/en-us/library/windows/desktop/dd162938.aspx
type winRGBQUAD struct {
	RgbBlue     byte
	RgbGreen    byte
	RgbRed      byte
	RgbReserved byte
}

// FoundWindows holds map between window handle and window title
type FoundWindows map[syscall.Handle]string

// EnsureUniqueNames renames windows if duplicate names found, possible if Tranquility and Singularity clients running at the same time with same character logged in.
func (fw FoundWindows) EnsureUniqueNames() FoundWindows {
	z := 1

	for i, name1 := range fw {
		for j, name2 := range fw {
			if i == j {
				continue
			}

			if name1 == name2 {
				fw[j] = fmt.Sprintf("%s - %d", name2, z)
				z++
			}
		}
	}

	return fw
}

func (fw FoundWindows) GetHandleByTitle(title string) syscall.Handle {
	for handle, wtitle := range fw {
		if wtitle == title {
			return handle
		}
	}

	return 0
}

func (fw FoundWindows) GetWindowsTitles() []string {
	result := []string{}
	for _, wtitle := range fw {
		result = append(result, wtitle)
	}

	return result
}

// FindWindow enumerates windows and returns map of FoundWindows with handle associated with title matching provided regex
func FindWindow(title *regexp.Regexp) (FoundWindows, error) {
	results := make(FoundWindows)
	cb := syscall.NewCallback(func(h syscall.Handle, p uintptr) uintptr {
		b := make([]uint16, 200)
		_, err := GetWindowText(h, &b[0], int32(len(b)))
		if err != nil {
			// ignore the error
			return 1 // enumeration continue
		}
		windowTitle := windows.UTF16ToString(b)
		if title.Find([]byte(windowTitle)) != nil {
			// note the window
			results[h] = windowTitle
		}
		return 1 // enumeration continue
	})

	err := enumWindows(cb, 0)
	if err != nil {
		return results, err
	}

	if len(results) == 0 {
		return results, fmt.Errorf("no window with title '%s' found", title.String())
	}

	return results, nil
}

// GetWindowText returns window title
func GetWindowText(hwnd syscall.Handle, str *uint16, maxCount int32) (length int32, err error) {
	r0, _, e1 := syscall.Syscall(procGetWindowTextW.Addr(), 3, uintptr(hwnd), uintptr(unsafe.Pointer(str)), uintptr(maxCount))

	length = int32(r0)
	if length == 0 {
		if e1 != 0 {
			err = error(e1)
		} else {
			err = syscall.EINVAL
		}
	}

	return
}

// WindowRect gets the dimensions for a Window handle.
func WindowRect(hwnd syscall.Handle) (image.Rectangle, error) {
	var rect winRect

	ret, _, err := procGetClientRect.Call(uintptr(hwnd), uintptr(unsafe.Pointer(&rect)))
	if ret == 0 {
		return image.Rectangle{}, fmt.Errorf("error getting window dimensions: %s", err)
	}

	return image.Rect(0, 0, int(rect.Right), int(rect.Bottom)), nil
}

// captureWindow captures the desired area from a Window and returns an image.
func captureWindow(handle syscall.Handle, rect image.Rectangle) (image.Image, error) {
	// Get the device context for screenshotting
	dcSrc, _, err := procGetDC.Call(uintptr(handle))
	if dcSrc == 0 {
		return nil, fmt.Errorf("error preparing screen capture: %s", err)
	}

	defer procReleaseDC.Call(0, dcSrc) // nolint:errcheck // it's fine

	// Grab a compatible DC for drawing
	dcDst, _, err := procCreateCompatibleDC.Call(dcSrc)
	if dcDst == 0 {
		return nil, fmt.Errorf("error creating DC for drawing: %s", err)
	}

	defer procDeleteDC.Call(dcDst) // nolint:errcheck // it's fine

	// Determine the width/height of our capture
	width := rect.Dx()
	height := rect.Dy()

	// Get the bitmap we're going to draw onto
	var bitmapInfo winBITMAPINFO
	bitmapInfo.BmiHeader = winBITMAPINFOHEADER{
		BiSize:        uint32(reflect.TypeOf(bitmapInfo.BmiHeader).Size()),
		BiWidth:       int32(width),
		BiHeight:      int32(height),
		BiPlanes:      1,
		BiBitCount:    32,
		BiCompression: 0, // BI_RGB
	}

	bitmapData := unsafe.Pointer(uintptr(0))
	bitmap, _, err := procCreateDIBSection.Call(
		dcDst,
		uintptr(unsafe.Pointer(&bitmapInfo)),
		0,
		uintptr(unsafe.Pointer(&bitmapData)), 0, 0)

	if bitmap == 0 {
		return nil, fmt.Errorf("error creating bitmap for screen capture: %s", err)
	}

	defer procDeleteObject.Call(bitmap) //nolint:errcheck // no needed

	// Select the object and paint it
	procSelectObject.Call(dcDst, bitmap) //nolint:errcheck // no needed
	ret, _, err := procBitBlt.Call(
		dcDst, 0, 0, uintptr(width), uintptr(height),
		dcSrc, uintptr(rect.Min.X), uintptr(rect.Min.Y), bitBltSRCCOPY)

	if ret == 0 {
		return nil, fmt.Errorf("error capturing screen: %s", err)
	}

	// Convert the bitmap to an image.Image. We first start by directly
	// creating a slice. This is unsafe but we know the underlying structure
	// directly.
	var slice []byte
	sliceHdr := (*reflect.SliceHeader)(unsafe.Pointer(&slice))
	sliceHdr.Data = uintptr(bitmapData)
	sliceHdr.Len = width * height * 4
	sliceHdr.Cap = sliceHdr.Len

	// Using the raw data, grab the RGBA data and transform it into an image.RGBA
	imageBytes := make([]byte, len(slice))
	for i := 0; i < len(imageBytes); i += 4 {
		imageBytes[i], imageBytes[i+2], imageBytes[i+1], imageBytes[i+3] = slice[i+2], slice[i], slice[i+1], slice[i+3]
	}

	img := &image.RGBA{Pix: imageBytes, Stride: 4 * width, Rect: image.Rect(0, 0, width, height)}
	dst := image.NewRGBA(img.Bounds())
	gift.New(gift.FlipVertical()).Draw(dst, img)

	return dst, nil
}

func enumWindows(enumFunc, lparam uintptr) (err error) {
	r1, _, e1 := syscall.Syscall(procEnumWindows.Addr(), 2, enumFunc, lparam, 0)
	if r1 == 0 {
		if e1 != 0 {
			err = error(e1)
		} else {
			err = syscall.EINVAL
		}
	}

	return
}
