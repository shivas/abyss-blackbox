package mainwindow

import (
	"log"
	"sort"

	"github.com/lxn/walk"
	"github.com/shivas/abyss-blackbox/combatlog"
	"github.com/shivas/abyss-blackbox/internal/fittings"
)

type Runner struct {
	Index         int
	CharacterName string
	FittingName   string
	FittingID     string
	ShipType      string
	ShipBitmap    *walk.Bitmap
	checked       bool
}

type RunnerModel struct {
	walk.TableModelBase
	walk.SorterBase
	sortColumn int
	sortOrder  walk.SortOrder
	items      []*Runner
	fm         *fittings.FittingsManager
	characters map[string]combatlog.CombatLogFile
	font       *walk.Font
}

func NewRunnerModel(characters map[string]combatlog.CombatLogFile, fm *fittings.FittingsManager) *RunnerModel {
	m := new(RunnerModel)
	m.fm = fm
	m.characters = characters
	fnt, err := walk.NewFont("MS Shell Dlg 2", 8, 0)
	if err != nil {
		log.Fatal(err)
	}
	m.font = fnt
	m.RefreshList()
	return m
}

func (m *RunnerModel) RefreshList() {
	m.items = make([]*Runner, 0, len(m.characters))
	i := 0

	for k := range m.characters {
		fitting := m.fm.GetFittingForPilot(k)
		if fitting != nil {
			m.items = append(m.items, &Runner{Index: i, CharacterName: k, FittingName: fitting.FittingName, ShipType: fitting.ShipName, ShipBitmap: fitting.ShipImage()})
		} else {
			m.items = append(m.items, &Runner{Index: i, CharacterName: k, FittingName: "", ShipType: "", ShipBitmap: nil})
		}

		i++
	}
	m.PublishRowsReset()
	_ = m.Sort(m.sortColumn, m.sortOrder)
}

// Called by the TableView from SetModel and every time the model publishes a
// RowsReset event.
func (m *RunnerModel) RowCount() int {
	return len(m.items)
}

// Called by the TableView when it needs the text to display for a given cell.
func (m *RunnerModel) Value(row, col int) interface{} {
	item := m.items[row]

	switch col {
	case 0:
		return item.CharacterName

	case 1:
		return item.ShipType

	case 2:
		return item.FittingName
	}

	panic("unexpected col")
}

// Called by the TableView to sort the model.
func (m *RunnerModel) Sort(col int, order walk.SortOrder) error {
	m.sortColumn, m.sortOrder = col, order

	sort.SliceStable(m.items, func(i, j int) bool {
		a, b := m.items[i], m.items[j]

		c := func(ls bool) bool {
			if m.sortOrder == walk.SortAscending {
				return ls
			}

			return !ls
		}

		switch m.sortColumn {
		case 0:
			return c(a.CharacterName < b.CharacterName)

		case 1:
			return c(a.ShipType < b.ShipType)

		case 2, 3:
			return c(a.FittingName < b.FittingName)
		}

		panic("unreachable")
	})

	return m.SorterBase.Sort(col, order)
}

func (m *RunnerModel) StyleCell(style *walk.CellStyle) {
	item := m.items[style.Row()]

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
			canvas.DrawBitmapWithOpacity(item.ShipBitmap, bounds, 255)

			bounds = style.Bounds()
			bounds.X += 34
			bounds.Y += 10
			bounds.Width -= 34
			canvas.DrawText(item.ShipType, m.font, 0, bounds, walk.TextLeft)
		}
	}
}

// Called by the TableView to retrieve if a given row is checked.
func (m *RunnerModel) Checked(row int) bool {
	return m.items[row].checked
}

// Called by the TableView when the user toggled the check box of a given row.
func (m *RunnerModel) SetChecked(row int, checked bool) error {
	m.items[row].checked = checked

	return nil
}

func (m *RunnerModel) GetCheckedCharacters() (characters []string) {
	for _, v := range m.items {
		if v.checked {
			characters = append(characters, v.CharacterName)
		}
	}

	return
}
