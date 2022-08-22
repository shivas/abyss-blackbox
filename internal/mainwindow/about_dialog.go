package mainwindow

import (
	"bytes"

	"github.com/lxn/walk"
	. "github.com/lxn/walk/declarative" // nolint:stylecheck,revive // we needs side effects
	"github.com/pkg/browser"

	_ "embed"
	"image"
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
	img, _ := walk.NewBitmapFromImage(png)

	return (Dialog{
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
	}).Run(owner)
}
