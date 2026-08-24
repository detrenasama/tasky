package ui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/detrenasama/tasky/internal/ui/theme"
)

// checkColor возвращает цвет индикатора чек-листа по статусу.
func checkColor(status string) string {
	switch status {
	case "new":
		return theme.StatusPalette[4] // жёлтый
	case "in_progress":
		return theme.StatusPalette[2] // синий
	case "done":
		return theme.StatusPalette[0] // зелёный
	case "cancelled":
		return theme.StatusPalette[6] // серый
	default:
		return theme.StatusPalette[6]
	}
}

// checkMark — символ индикатора чек-листа по статусу.
func checkMark(status string) string {
	switch status {
	case "new":
		return "[ ]"
	case "in_progress":
		return "[•]"
	case "done":
		return "[✓]"
	case "cancelled":
		return "[-]"
	default:
		return "[ ]"
	}
}

// checkIndicator — окрашенный индикатор чек-листа перед строкой.
func checkIndicator(status string) string {
	return lipgloss.NewStyle().Foreground(lipgloss.Color(checkColor(status))).Render(checkMark(status))
}

// openChecklist открывает редактор чек-листа выбранной подзадачи.
func (s *tasksScreen) openChecklist() {
	kind, id := s.selectedKindID()
	if kind != kindSubtask || id == 0 {
		return
	}
	s.lastErr = nil
	s.checklistLoad(id)
	s.mode = taskChecklist
}

// checklistLoad перечитывает элементы чек-листа подзадачи и собирает список модалки.
func (s *tasksScreen) checklistLoad(subtaskID int64) {
	selVal := int64(0)
	if s.checklistPick.sel >= 0 && s.checklistPick.sel < len(s.checklistPick.items) {
		selVal = s.checklistPick.items[s.checklistPick.sel].value
	}
	s.checklistItems, _ = s.store.ChecklistItems(subtaskID)
	items := make([]pickItem, 0, len(s.checklistItems))
	for _, ci := range s.checklistItems {
		items = append(items, pickItem{value: ci.ID,
			label: checkIndicator(ci.Status) + " " + ci.Text})
	}
	s.checklistPick.items = items
	ns := 0
	for i, it := range items {
		if it.value == selVal {
			ns = i
			break
		}
	}
	s.checklistPick.sel = ns
	s.checklistPick.clampScroll()
}

// checklistStatusOf возвращает статус элемента чек-листа по id.
func (s *tasksScreen) checklistStatusOf(id int64) string {
	for _, ci := range s.checklistItems {
		if ci.ID == id {
			return ci.Status
		}
	}
	return "new"
}

// checklistTextOf возвращает текст элемента чек-листа по id.
func (s *tasksScreen) checklistTextOf(id int64) string {
	for _, ci := range s.checklistItems {
		if ci.ID == id {
			return ci.Text
		}
	}
	return ""
}

// checklistReload перечитывает чек-лист и счётчики для значка в списке.
func (s *tasksScreen) checklistReload() {
	kind, id := s.selectedKindID()
	if kind == kindSubtask && id != 0 {
		s.checklistLoad(id)
	}
	s.loadData()
}

// checklistCycle переключает статус выделенного элемента на dir (+1/-1)
// линейно, без зацикливания: new → in_progress → done → cancelled; на краях
// переключение останавливается (с new влево и с cancelled вправо ничего не делает).
func (s *tasksScreen) checklistCycle(dir int) {
	it, ok := s.checklistPick.selected()
	if !ok {
		return
	}
	order := []string{"new", "in_progress", "done", "cancelled"}
	cur := s.checklistStatusOf(it.value)
	idx := 0
	for i, st := range order {
		if st == cur {
			idx = i
			break
		}
	}
	next := idx + dir
	if next < 0 {
		next = 0
	} else if next >= len(order) {
		next = len(order) - 1
	}
	idx = next
	if err := s.store.SetChecklistItemStatus(it.value, order[idx]); err != nil {
		s.lastErr = err
		return
	}
	s.checklistReload()
}

// updateChecklist обрабатывает клавиши модального окна чек-листа.
func (s *tasksScreen) updateChecklist(m *model, msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	if s.checklistNew {
		switch msg.String() {
		case "enter":
			text := strings.TrimSpace(s.checklistInput.Value())
			if text == "" {
				s.lastErr = fmt.Errorf("текст пункта не может быть пустым")
				return m, nil
			}
			kind, id := s.selectedKindID()
			if kind != kindSubtask || id == 0 {
				s.checklistNew = false
				s.mode = taskBrowse
				return m, nil
			}
			var err error
			if s.checklistEditID == 0 {
				_, err = s.store.CreateChecklistItem(id, text)
			} else {
				err = s.store.UpdateChecklistItemText(s.checklistEditID, text)
			}
			if err != nil {
				s.lastErr = err
				return m, nil
			}
			s.checklistNew = false
			s.checklistEditID = 0
			s.lastErr = nil
			s.checklistReload()
			return m, nil
		case "esc":
			s.checklistNew = false
			s.checklistEditID = 0
			s.lastErr = nil
			return m, nil
		}
		var cmd tea.Cmd
		s.checklistInput, cmd = s.checklistInput.Update(msg)
		return m, cmd
	}

	switch msg.String() {
	case "up":
		s.checklistPick.move(-1)
	case "down":
		s.checklistPick.move(1)
	case "enter":
		if it, ok := s.checklistPick.selected(); ok {
			next := "done"
			if s.checklistStatusOf(it.value) == "done" {
				next = "new"
			}
			if err := s.store.SetChecklistItemStatus(it.value, next); err != nil {
				s.lastErr = err
			} else {
				s.checklistReload()
			}
		}
	case "right":
		s.checklistCycle(1)
	case "left":
		s.checklistCycle(-1)
	case "ctrl+up":
		if it, ok := s.checklistPick.selected(); ok {
			if err := s.store.MoveChecklistItem(it.value, -1); err != nil {
				s.lastErr = err
			} else {
				s.checklistReload()
			}
		}
	case "ctrl+down":
		if it, ok := s.checklistPick.selected(); ok {
			if err := s.store.MoveChecklistItem(it.value, 1); err != nil {
				s.lastErr = err
			} else {
				s.checklistReload()
			}
		}
	case "e":
		if it, ok := s.checklistPick.selected(); ok {
			s.checklistEditID = it.value
			s.checklistNew = true
			s.checklistInput.SetValue(s.checklistTextOf(it.value))
			s.checklistInput.CursorEnd()
			s.checklistInput.Focus()
			s.lastErr = nil
		}
	case "n":
		s.checklistEditID = 0
		s.checklistNew = true
		s.checklistInput.SetValue("")
		s.checklistInput.Focus()
		s.lastErr = nil
	case "d":
		if it, ok := s.checklistPick.selected(); ok {
			s.checklistConfirmID = it.value
			s.mode = taskChecklistConfirm
		}
	case "esc":
		s.checklistNew = false
		s.checklistEditID = 0
		s.mode = taskBrowse
		s.loadData()
	}
	return m, nil
}
