//go:build windows

package screen

import (
	"fmt"
	"image"
	"reflect"
	"syscall"
	"unsafe"

	"github.com/disintegration/gift"
)

func CaptureWindowArea(handle syscall.Handle, rect image.Rectangle) (image.Image, error) {
	return captureWindow(handle, rect)
}

var (
	modUser32     = syscall.NewLazyDLL("User32.dll")
	procGetDC     = modUser32.NewProc("GetDC")
	procReleaseDC = modUser32.NewProc("ReleaseDC")

	modGdi32   = syscall.NewLazyDLL("Gdi32.dll")
	procBitBlt = modGdi32.NewProc("BitBlt")

	// procCreateCompatibleBitmap = modGdi32.NewProc("CreateCompatibleBitmap")
	procCreateCompatibleDC = modGdi32.NewProc("CreateCompatibleDC")
	procCreateDIBSection   = modGdi32.NewProc("CreateDIBSection")
	procDeleteDC           = modGdi32.NewProc("DeleteDC")
	procDeleteObject       = modGdi32.NewProc("DeleteObject")

	// procGetDeviceCaps          = modGdi32.NewProc("GetDeviceCaps")
	procSelectObject = modGdi32.NewProc("SelectObject")
)

const (
	// BitBlt constants
	bitBltSRCCOPY = 0x00CC0020
)

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

// captureWindow captures the desired area from a Window and returns an image.
func captureWindow(handle syscall.Handle, rect image.Rectangle) (image.Image, error) {
	// Get the device context for screenshotting
	dcSrc, _, err := procGetDC.Call(uintptr(handle))
	if dcSrc == 0 {
		return nil, fmt.Errorf("error preparing screen capture: %s", err)
	}

	defer procReleaseDC.Call(0, dcSrc) //nolint:errcheck // it's fine

	// Grab a compatible DC for drawing
	dcDst, _, err := procCreateCompatibleDC.Call(dcSrc)
	if dcDst == 0 {
		return nil, fmt.Errorf("error creating DC for drawing: %s", err)
	}

	defer procDeleteDC.Call(dcDst) //nolint:errcheck // it's fine

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

	zero := uintptr(0)
	bitmapData := unsafe.Pointer(&zero)
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
