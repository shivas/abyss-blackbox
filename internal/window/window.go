package window

import (
	"errors"
	"fmt"
	"regexp"
	"sync"
	"syscall"
	"unsafe"

	"github.com/shivas/go-windows-hlp/pkg/pcl"
	"golang.org/x/sys/windows"
)

const EVEClientWindowRe = "^EVE -.*$"

var ErrNoWindowsFound = errors.New("no EVE client windows found")

var (
	titleRe = regexp.MustCompile(EVEClientWindowRe)

	modUser32          = syscall.NewLazyDLL("User32.dll")
	procGetWindowTextW = modUser32.NewProc("GetWindowTextW")
)

func NewManager() (*Manager, error) {
	m := &Manager{}

	results, err := m.findEVEWindows()
	m.queryResults = results

	if err != nil {
		return m, err
	}

	m.queryResults.getWindowsAttributes()
	m.queryResults.ensureUniqueNames()

	return m, err
}

type Manager struct {
	queryResults *queryWindowsResult
}

func (m *Manager) GetEVEClientWindows() map[syscall.Handle]string {
	result := make(map[syscall.Handle]string, len(m.queryResults.windows))
	for h, w := range m.queryResults.windows {
		result[h] = w.windowTitle
	}

	return result
}

func (m *Manager) findEVEWindows() (*queryWindowsResult, error) {
	result := &queryWindowsResult{windows: make(map[syscall.Handle]details)}
	cb := syscall.NewCallback(func(h syscall.Handle, p uintptr) uintptr {
		b := make([]uint16, 200)
		_, err := getWindowText(h, &b[0], int32(len(b)))
		if err != nil {
			// ignore the error
			return 1 // enumeration continue
		}
		windowTitle := windows.UTF16ToString(b)
		if titleRe.Find([]byte(windowTitle)) != nil {
			// note the window
			result.windows[h] = details{windowTitle: windowTitle}
		}
		return 1 // enumeration continue
	})

	zero := uintptr(0)

	err := windows.EnumWindows(cb, unsafe.Pointer(&zero))
	if err != nil {
		return result, err
	}

	if len(result.windows) == 0 {
		return result, ErrNoWindowsFound
	}

	return result, nil
}

func (m *Manager) GetHandleByTitle(title string) syscall.Handle {
	for handle, w := range m.queryResults.windows {
		if w.windowTitle == title {
			return handle
		}
	}

	return 0 // fallbacks to 0 - desktop
}

func (m *Manager) IsTestingServer(handle syscall.Handle) bool {
	return m.queryResults.windows[handle].serverName != "tranquility"
}

func (m *Manager) GetWindowsTitles() []string {
	result := []string{}
	for _, w := range m.queryResults.windows {
		result = append(result, w.windowTitle)
	}

	return result
}

// getWindowText returns window title
func getWindowText(hwnd syscall.Handle, str *uint16, maxCount int32) (length int32, err error) {
	r0, _, e1 := syscall.SyscallN(procGetWindowTextW.Addr(), uintptr(hwnd), uintptr(unsafe.Pointer(str)), uintptr(maxCount))

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

type queryWindowsResult struct {
	windows map[syscall.Handle]details
}

// EnsureUniqueNames renames windows if duplicate names found, possible if Tranquility and Singularity clients running at the same time with same character logged in.
func (fw *queryWindowsResult) ensureUniqueNames() {
	for i, w1 := range fw.windows {
		w1.handle = i
		fw.windows[i] = w1

		for j, w2 := range fw.windows {
			if i == j {
				continue
			}

			if w1.windowTitle == w2.windowTitle {
				w2.windowTitle = fmt.Sprintf("%s - %s", w2.windowTitle, w2.serverName)
				fw.windows[j] = w2
			}
		}
	}
}

func (fw *queryWindowsResult) getWindowsAttributes() {
	var wg sync.WaitGroup

	var mu sync.Mutex

	for k, w := range fw.windows {
		wg.Add(1)

		go func(wg *sync.WaitGroup, handle syscall.Handle, d details) {
			defer wg.Done()

			var err error

			cLine, err := pcl.GetCommandLine(windows.HWND(handle))
			if err != nil {
				d.err = err

				return
			}

			var serverRe = regexp.MustCompile(`/server:(\w*)`)
			for i, match := range serverRe.FindAllStringSubmatch(cLine, 1) {
				d.serverName = match[i+1]
			}

			var languageRe = regexp.MustCompile(`/language=(\w*)`)
			for i, match := range languageRe.FindAllStringSubmatch(cLine, 1) {
				d.language = match[i+1]
			}

			var dx12Re = regexp.MustCompile(`/triplatform=(\w*)`)
			for i, match := range dx12Re.FindAllStringSubmatch(cLine, 1) {
				if match[i+1] == "dx12" {
					d.dx12 = true
				}
			}

			mu.Lock()
			defer mu.Unlock()

			fw.windows[handle] = d
		}(&wg, k, w)
	}

	wg.Wait()
}

type details struct {
	handle      syscall.Handle
	windowTitle string
	serverName  string
	dx12        bool
	language    string
	err         error
}
