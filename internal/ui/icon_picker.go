package ui

import (
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/widget"
)

type iconOption struct {
	label string
	icon  fyne.Resource
}

type iconPicker struct {
	widget.Button
	options  []iconOption
	selected int
	onChange func(int)
}

func newIconPicker(options []iconOption, selected int, onChange func(int)) *iconPicker {
	if selected < 0 || selected >= len(options) {
		selected = 0
	}

	p := &iconPicker{options: options, selected: selected, onChange: onChange}
	p.ExtendBaseWidget(p)
	p.Alignment = widget.ButtonAlignLeading
	p.IconPlacement = widget.ButtonIconLeadingText
	p.OnTapped = p.showMenu
	p.sync()

	return p
}

func (p *iconPicker) showMenu() {
	if p.Disabled() {
		return
	}

	c := fyne.CurrentApp().Driver().CanvasForObject(p)
	if c == nil {
		return
	}

	items := make([]*fyne.MenuItem, len(p.options))
	for i, opt := range p.options {
		i, opt := i, opt
		item := fyne.NewMenuItem(opt.label, func() {
			p.selected = i
			p.sync()
			if p.onChange != nil {
				p.onChange(i)
			}
		})
		item.Icon = opt.icon
		items[i] = item
	}

	widget.ShowPopUpMenuAtRelativePosition(fyne.NewMenu("", items...), c, fyne.NewPos(0, p.Size().Height), p)
}

func (p *iconPicker) sync() {
	if p.selected < 0 || p.selected >= len(p.options) {
		return
	}
	p.SetText(p.options[p.selected].label)
	p.SetIcon(p.options[p.selected].icon)
}
