package fittings

import (
	"log"
	"sort"

	"github.com/lxn/walk"
)

type FittingsModel struct {
	walk.TableModelBase
	walk.SorterBase
	sortColumn int
	sortOrder  walk.SortOrder
	fm         *FittingsManager
	checks     map[int]bool
	font       *walk.Font
}

func (m *FittingsManager) Model() *FittingsModel {
	fnt, err := walk.NewFont("MS Shell Dlg 2", 8, 0)
	if err != nil {
		log.Fatal(err)
	}

	return &FittingsModel{fm: m, checks: make(map[int]bool), font: fnt}
}

func (m *FittingsModel) RowCount() int {
	return len(m.fm.cache.Fittings)
}

// Called by the TableView when it needs the text to display for a given cell.
func (m *FittingsModel) Value(row, col int) interface{} {
	item := m.fm.cache.Fittings[row]

	switch col {
	case 0:
		return item.FittingName

	case 1:
		return item.ShipName

	case 2:
		return item.FFH
	}

	panic("unexpected col")
}

// Called by the TableView to retrieve if a given row is checked.
func (m *FittingsModel) Checked(row int) bool {
	return m.checks[row]
}

// Called by the TableView when the user toggled the check box of a given row.
func (m *FittingsModel) SetChecked(row int, checked bool) error {
	m.checks[row] = checked

	return nil
}

// Called by the TableView to sort the model.
func (m *FittingsModel) Sort(col int, order walk.SortOrder) error {
	m.sortColumn, m.sortOrder = col, order

	sort.SliceStable(m.fm.cache.Fittings, func(i, j int) bool {
		a, b := m.fm.cache.Fittings[i], m.fm.cache.Fittings[j]

		c := func(ls bool) bool {
			if m.sortOrder == walk.SortAscending {
				return ls
			}

			return !ls
		}

		switch m.sortColumn {
		case 0:
			return c(a.FittingName < b.FittingName)

		case 1:
			return c(a.ShipName < b.ShipName)

		case 2, 3:
			return c(a.FFH < b.FFH)
		}

		panic("unreachable")
	})

	return m.SorterBase.Sort(col, order)
}

func (m *FittingsModel) StyleCell(style *walk.CellStyle) {
	item := m.fm.cache.Fittings[style.Row()]

	//	if item.checked {
	if style.Row()%2 == 0 {
		style.BackgroundColor = walk.RGB(255, 255, 255)
	} else {
		style.BackgroundColor = walk.RGB(244, 244, 244)
	}
	// }

	if style.Col() == 1 {
		if canvas := style.Canvas(); canvas != nil {
			bounds := style.Bounds()
			bounds.Height = 32
			bounds.Width = 32
			canvas.DrawBitmapWithOpacity(item.ShipImage(), bounds, 255)

			bounds = style.Bounds()
			bounds.X += 34
			bounds.Y += 10
			bounds.Width -= 34
			canvas.DrawText(item.ShipName, m.font, 0, bounds, walk.TextLeft)
		}
	}
}
