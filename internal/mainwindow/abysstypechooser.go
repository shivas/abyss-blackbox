package mainwindow

import (
	"bytes"
	"embed"
	"image"
	_ "image/png"

	"github.com/lxn/walk"
	"github.com/shivas/abyss-blackbox/internal/config"
)

//go:embed images/*
var abyssIcons embed.FS

type AbyssTypeChooser struct {
	toolbar   *walk.ToolBar
	cfg       *config.CaptureConfig
	shipTypes []abyssToolbarOption
	tiers     []abyssToolbarOption
	weathers  []abyssToolbarOption
}

type abyssToolbarOption struct {
	Title string
	Icon  *walk.Icon
}

func NewAbyssTypeChooser(t *walk.ToolBar, c *config.CaptureConfig) *AbyssTypeChooser {
	return &AbyssTypeChooser{toolbar: t, cfg: c}
}

func (a *AbyssTypeChooser) Init() {
	a.shipTypes = make([]abyssToolbarOption, 0)
	a.shipTypes = append(a.shipTypes,
		abyssToolbarOption{Title: "Cruiser", Icon: embededIcon("images/Ship.png")},
		abyssToolbarOption{Title: "Destroyers", Icon: embededIcon("images/Ship.png")},
		abyssToolbarOption{Title: "Frigates", Icon: embededIcon("images/Ship.png")},
	)

	a.tiers = make([]abyssToolbarOption, 0)
	a.tiers = append(a.tiers,
		abyssToolbarOption{Title: "Tier 0", Icon: embededIcon("images/0.png")},
		abyssToolbarOption{Title: "Tier 1", Icon: embededIcon("images/1.png")},
		abyssToolbarOption{Title: "Tier 2", Icon: embededIcon("images/2.png")},
		abyssToolbarOption{Title: "Tier 3", Icon: embededIcon("images/3.png")},
		abyssToolbarOption{Title: "Tier 4", Icon: embededIcon("images/4.png")},
		abyssToolbarOption{Title: "Tier 5", Icon: embededIcon("images/5.png")},
		abyssToolbarOption{Title: "Tier 6", Icon: embededIcon("images/6.png")},
	)

	a.weathers = make([]abyssToolbarOption, 0)
	a.weathers = append(a.weathers,
		abyssToolbarOption{Title: "Dark", Icon: embededIcon("images/Dark.png")},
		abyssToolbarOption{Title: "Electrical", Icon: embededIcon("images/Electrical.png")},
		abyssToolbarOption{Title: "Exotic", Icon: embededIcon("images/Exotic.png")},
		abyssToolbarOption{Title: "Firestorm", Icon: embededIcon("images/Firestorm.png")},
		abyssToolbarOption{Title: "Gamma", Icon: embededIcon("images/Gamma.png")},
	)

	for i, s := range a.shipTypes {
		s := s
		shipType := i + 1
		if shipType == a.cfg.AbyssShipType {
			a.toolbar.Actions().At(0).SetText(s.Title)
			a.toolbar.Actions().At(0).SetImage(s.Icon)
		}
		a.toolbar.Actions().At(0).Menu().Actions().At(i).SetText(s.Title)
		a.toolbar.Actions().At(0).Menu().Actions().At(i).SetImage(s.Icon)
		a.toolbar.Actions().At(0).Menu().Actions().At(i).Triggered().Attach(func() {
			a.toolbar.Actions().At(0).SetText(s.Title)
			a.toolbar.Actions().At(0).SetImage(s.Icon)
			a.cfg.AbyssShipType = shipType
			config.Write(a.cfg)
		})
	}

	for i, s := range a.tiers {
		s := s
		tier := i
		if i == a.cfg.AbyssTier {
			a.toolbar.Actions().At(1).SetText(s.Title)
			a.toolbar.Actions().At(1).SetImage(s.Icon)
		}
		a.toolbar.Actions().At(1).Menu().Actions().At(i).SetText(s.Title)
		a.toolbar.Actions().At(1).Menu().Actions().At(i).SetImage(s.Icon)
		a.toolbar.Actions().At(1).Menu().Actions().At(i).Triggered().Attach(func() {
			a.toolbar.Actions().At(1).SetText(s.Title)
			a.toolbar.Actions().At(1).SetImage(s.Icon)
			a.cfg.AbyssTier = tier
			config.Write(a.cfg)
		})
	}

	for i, s := range a.weathers {
		s := s
		weather := s.Title

		if weather == a.cfg.AbyssWeather {
			a.toolbar.Actions().At(2).SetText(s.Title)
			a.toolbar.Actions().At(2).SetImage(s.Icon)
		}
		a.toolbar.Actions().At(2).Menu().Actions().At(i).SetText(s.Title)
		a.toolbar.Actions().At(2).Menu().Actions().At(i).SetImage(s.Icon)
		a.toolbar.Actions().At(2).Menu().Actions().At(i).Triggered().Attach(func() {
			a.toolbar.Actions().At(2).SetText(s.Title)
			a.toolbar.Actions().At(2).SetImage(s.Icon)
			a.cfg.AbyssWeather = s.Title
			config.Write(a.cfg)
		})

	}
}

func embededIcon(name string) *walk.Icon {
	data, err := abyssIcons.ReadFile(name)
	if err != nil {
		panic(err)
	}

	img, _, err := image.Decode(bytes.NewReader(data))
	if err != nil {
		panic(err)
	}

	icon, err := walk.NewIconFromImageForDPI(img, 92)
	if err != nil {
		panic(err)
	}

	return icon
}
