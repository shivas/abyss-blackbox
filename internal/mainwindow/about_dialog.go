package mainwindow

//nolint:revive,stylecheck // side effects

import (
	"bytes"
	"image"
	"log"

	"github.com/lxn/walk"
	. "github.com/lxn/walk/declarative"
	"github.com/pkg/browser"

	_ "embed"

	_ "image/png"

	"github.com/shivas/abyss-blackbox/internal/version"
)

//go:embed triglogo.png
var logo []byte

func RunAboutDialog(owner walk.Form) (int, error) {
	var (
		dlg      *walk.Dialog
		acceptPB *walk.PushButton
		appLogo  *walk.ImageView
	)

	png, _, _ := image.Decode(bytes.NewReader(logo))
	img, _ := walk.NewBitmapFromImageForDPI(png, 92)

	err := Dialog{
		AssignTo:      &dlg,
		Title:         "About",
		DefaultButton: &acceptPB,
		Layout:        VBox{},
		Children: []Widget{
			Composite{
				Layout: HBox{},
				Children: []Widget{
					ImageView{
						AssignTo: &appLogo,
						Image:    img,
					},
					Composite{
						Layout: VBox{},
						Children: []Widget{
							TextLabel{
								Text: "Version: " + version.RecorderVersion,
							},
							TextLabel{
								Text: "Go: " + version.GoVersion,
							},
							LinkLabel{
								Alignment: AlignHNearVCenter,
								Text:      `Telemetry site: <a id="tracker" href="https://abyssal.space">https://abyssal.space</a>`,
								OnLinkActivated: func(link *walk.LinkLabelLink) {
									_ = browser.OpenURL(link.URL())
								},
							},
							LinkLabel{
								Alignment: AlignHNearVCenter,
								Text:      `Recorder releases: <a id="github" href="https://github.com/shivas/abyss-blackbox/releases">https://github.com/shivas/abyss-blackbox</a>`,
								OnLinkActivated: func(link *walk.LinkLabelLink) {
									_ = browser.OpenURL(link.URL())
								},
							},
							LinkLabel{
								Alignment: AlignHNearVCenter,
								Text:      `For help join <a id="this" href="https://discord.gg/qFcywP6WUK">Abyssal Lurkers Discord</a> #abyss-telemetry channel.`,
								OnLinkActivated: func(link *walk.LinkLabelLink) {
									_ = browser.OpenURL(link.URL())
								},
							},
						},
					},
				},
			},
			Composite{
				Layout: HBox{},
				Children: []Widget{
					HSpacer{},
					PushButton{
						AssignTo: &acceptPB,
						Text:     "OK",
						OnClicked: func() {
							dlg.Accept()
						},
					},
				},
			},
		},
	}.Create(owner)

	if err != nil {
		log.Fatal(err)
	}

	// win.SetWindowPos(dlg.Handle(),
	// 	win.HWND_TOPMOST, 0, 0, 0, 0,
	// 	win.SWP_NOACTIVATE|win.SWP_NOMOVE|win.SWP_NOSIZE)

	// flag := win.GetWindowLong(dlg.Handle(), win.GWL_STYLE) // Gets current style
	// flag |= win.WS_OVERLAPPED                              // always on top
	// // flag &= ^win.WS_BORDER                                 // no border(min/max/close)
	// // flag &= ^win.WS_THICKFRAME                             // fixed size
	// flag &= ^win.WS_SIZEBOX
	// flag &= ^win.WS_CAPTION
	// win.SetWindowLong(dlg.Handle(), win.GWL_STYLE, flag)

	// flag2 := win.GetWindowLong(dlg.Handle(), win.GWL_EXSTYLE)
	// flag2 |= win.WS_EX_NOACTIVATE
	// //flag2 |= win.WS_EX_TOOLWINDOW
	// flag2 |= win.WS_EX_TOPMOST
	// flag2 |= win.WS_EX_LAYERED
	// flag2 |= win.WS_EX_TRANSPARENT
	// //flag2 |= win.WS_EX_STATICEDGE
	// //flag2 |= win.WS_EX_NOPARENTNOTIFY

	// win.SetWindowLong(dlg.Handle(), win.GWL_EXSTYLE, flag2)

	// scb, err := walk.NewSolidColorBrush(walk.RGB(0, 0, 0))
	// if err != nil {
	// 	panic(err)
	// }
	// defer scb.Dispose()
	// //	win.SetBkColor(win.HDC(dlg.Handle()), win.COLORREF(scb.Color()))
	// dlg.SetBackground(scb)

	return dlg.Run(), nil
}
