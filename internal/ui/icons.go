package ui

import (
	"embed"

	"fyne.io/fyne/v2"
)

//go:embed images/*.png
var abyssIconFS embed.FS

func abyssIcon(name string) fyne.Resource {
	data, err := abyssIconFS.ReadFile("images/" + name)
	if err != nil {
		return nil
	}

	return fyne.NewStaticResource(name, data)
}

func shipTypeOptions() []iconOption {
	ship := abyssIcon("Ship.png")
	return []iconOption{
		{label: "Cruiser", icon: ship},
		{label: "Destroyers", icon: ship},
		{label: "Frigates", icon: ship},
	}
}

func tierOptions() []iconOption {
	return []iconOption{
		{label: "T0", icon: abyssIcon("0.png")},
		{label: "T1", icon: abyssIcon("1.png")},
		{label: "T2", icon: abyssIcon("2.png")},
		{label: "T3", icon: abyssIcon("3.png")},
		{label: "T4", icon: abyssIcon("4.png")},
		{label: "T5", icon: abyssIcon("5.png")},
		{label: "T6", icon: abyssIcon("6.png")},
	}
}

func weatherOptions() []iconOption {
	return []iconOption{
		{label: "Dark", icon: abyssIcon("Dark.png")},
		{label: "Electrical", icon: abyssIcon("Electrical.png")},
		{label: "Exotic", icon: abyssIcon("Exotic.png")},
		{label: "Firestorm", icon: abyssIcon("Firestorm.png")},
		{label: "Gamma", icon: abyssIcon("Gamma.png")},
	}
}
