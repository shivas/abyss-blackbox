package mainwindow

import (
	"github.com/lxn/walk"
	"github.com/shivas/abyss-blackbox/combatlog"
)

func BuildSettingsWidget(characters map[string]combatlog.CombatLogFile, parent walk.Container) {
	for i := 0; i < parent.Children().Len(); i++ {
		parent.Children().At(i).Dispose()
	}
	for charName := range characters {
		cb, _ := walk.NewCheckBox(parent)
		_ = cb.SetText(charName)
		_ = cb.SetMinMaxSize(walk.Size{Width: 400}, walk.Size{Width: 800})
		_ = cb.SetAlignment(walk.AlignHNearVCenter)
		cb.SetChecked(false)
		_ = parent.Children().Add(cb)
	}
}
