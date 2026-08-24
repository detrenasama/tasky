package ui

import (
	"fmt"
	"github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/detrenasama/tasky/internal/ui/theme"
	"strings"
	"time"
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

// startNewTask открывает ввод названия новой задачи (гвард canCreate).
func (s *tasksScreen) startNewTask() {
	if s.canCreate(kindTask) {
		s.inputKind = kindTask
		s.lastErr = nil
		s.mode = taskInput
		s.input.Focus()
	}
}

// startNewSubtask открывает ввод названия новой подзадачи (гвард canCreate).
func (s *tasksScreen) startNewSubtask() {
	if s.canCreate(kindSubtask) {
		s.inputKind = kindSubtask
		s.lastErr = nil
		s.mode = taskInput
		s.input.Focus()
	}
}

// startDelete открывает подтверждение удаления выбранного элемента.
func (s *tasksScreen) startDelete() {
	if !s.canDelete() {
		return
	}
	s.confirmKind = s.selectedKind()
	switch s.confirmKind {
	case kindTask:
		if item, ok := s.list.SelectedItem().(taskItem); ok {
			s.confirmID = item.t.ID
		}
	case kindSubtask:
		if item, ok := s.list.SelectedItem().(subItem); ok {
			s.confirmID = item.st.ID
		}
	}
	s.mode = taskConfirm
}

// startEditTitle открывает модалку изменения названия выбранного элемента:
// textinput префиллен текущим названием, Enter сохраняет, Esc отменяет.
func (s *tasksScreen) startEditTitle() {
	if !s.canDelete() {
		return
	}
	s.inputKind = s.selectedKind()
	switch s.inputKind {
	case kindTask:
		if item, ok := s.list.SelectedItem().(taskItem); ok {
			s.editID = item.t.ID
			s.input.SetValue(item.t.Title)
		}
	case kindSubtask:
		if item, ok := s.list.SelectedItem().(subItem); ok {
			s.editID = item.st.ID
			s.input.SetValue(item.st.Title)
		}
	}
	if s.editID == 0 {
		return
	}
	s.lastErr = nil
	s.input.CursorEnd()
	s.mode = taskTitleEdit
	s.input.Focus()
}

// startEditDescription открывает инлайн-редактирование описания выбранной
// задачи/подзадачи (Enter в списке — то же, что раньше «e» в колонке описания).
func (s *tasksScreen) startEditDescription() {
	s.lastErr = nil
	s.descText.SetValue(s.desc)
	s.mode = taskDescEdit
	s.descText.Focus()
}

// startLinkEdit открывает модалку ссылки выбранного элемента: id=0 — новая,
// иначе — правка существующей (поля префилливаются из списка ссылок).
func (s *tasksScreen) startLinkEdit(id int64) {
	s.lastErr = nil
	s.editLinkID = id
	s.linkName.SetValue("")
	s.linkInput.SetValue("")
	for _, l := range s.links {
		if l.ID == id {
			s.linkName.SetValue(l.Name)
			s.linkInput.SetValue(l.URL)
		}
	}
	s.mode = taskLinkEdit
	s.linkName.Focus()
	s.linkInput.Blur()
}

// openLinks открывает список ссылок выбранного элемента.
func (s *tasksScreen) openLinks() {
	s.lastErr = nil
	s.mode = taskLinks
}

// startSearch открывает модалку поиска по проекту.
func (s *tasksScreen) startSearch() {
	s.lastErr = nil
	s.searchInput.SetValue(s.searchQuery)
	s.searchInput.Focus()
	s.mode = taskSearch
}

func (m *model) updateTasks(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	s := m.tasks

	switch s.mode {
	case taskInput:
		var cmd tea.Cmd
		s.input, cmd = s.input.Update(msg)
		switch msg.String() {
		case "enter":
			title := strings.TrimSpace(s.input.Value())
			if title != "" {
				created, err := s.createItem(s.inputKind, title)
				s.lastErr = err
				if err == nil {
					if s.inputKind == kindSubtask {
						s.expanded[s.selectedTaskID()] = true
					}
					s.loadData()
					s.selectByKindID(s.inputKind, created)
					s.loadInfo()
					s.loadDesc()
					s.descV.GotoTop()
				}
			}
			s.input.SetValue("")
			s.input.Blur()
			s.mode = taskBrowse
		case "esc":
			s.input.SetValue("")
			s.input.Blur()
			s.mode = taskBrowse
		}
		return m, cmd
	case taskTitleEdit:
		var cmd tea.Cmd
		s.input, cmd = s.input.Update(msg)
		switch msg.String() {
		case "enter":
			title := strings.TrimSpace(s.input.Value())
			if title == "" {
				s.lastErr = fmt.Errorf("название не может быть пустым")
				return m, cmd
			}
			var err error
			if s.inputKind == kindTask {
				err = s.store.UpdateTaskTitle(s.editID, title)
			} else {
				err = s.store.UpdateSubtaskTitle(s.editID, title)
			}
			s.lastErr = err
			if err == nil {
				s.loadData()
				s.selectByKindID(s.inputKind, s.editID)
				s.loadInfo()
				s.loadDesc()
				s.descV.GotoTop()
			}
			s.input.SetValue("")
			s.input.Blur()
			s.mode = taskBrowse
		case "esc":
			s.input.SetValue("")
			s.input.Blur()
			s.mode = taskBrowse
		}
		return m, cmd
	case taskConfirm:
		switch msg.String() {
		case "y", "enter":
			s.deleteItem(s.confirmKind, s.confirmID)
			s.mode = taskBrowse
			s.loadData()
		case "n", "esc":
			s.mode = taskBrowse
		}
		return m, nil
	case taskDescEdit:
		var cmd tea.Cmd
		s.descText, cmd = s.descText.Update(msg)
		switch msg.String() {
		case "ctrl+s":
			kind, id := s.selectedKindID()
			if kind == kindTask {
				s.store.UpdateTaskDescription(id, s.descText.Value())
			} else {
				s.store.UpdateSubtaskDescription(id, s.descText.Value())
			}
			s.descText.Blur()
			s.mode = taskBrowse
			s.loadDesc()
		case "esc":
			s.descText.Blur()
			s.mode = taskBrowse
		}
		return m, cmd
	case taskLinkEdit:
		var cmd tea.Cmd
		if s.linkName.Focused() {
			s.linkName, cmd = s.linkName.Update(msg)
		} else {
			s.linkInput, cmd = s.linkInput.Update(msg)
		}
		switch msg.String() {
		case "tab":
			if s.linkName.Focused() {
				s.linkName.Blur()
				s.linkInput.Focus()
			} else {
				s.linkInput.Blur()
				s.linkName.Focus()
			}
		case "enter":
			if s.linkInput.Focused() {
				url := strings.TrimSpace(s.linkInput.Value())
				name := strings.TrimSpace(s.linkName.Value())
				if url != "" {
					var err error
					if s.editLinkID != 0 {
						kind, _ := s.selectedKindID()
						if kind == kindTask {
							err = s.store.UpdateTaskLink(s.editLinkID, name, url)
						} else {
							err = s.store.UpdateSubtaskLink(s.editLinkID, name, url)
						}
					} else {
						kind, id := s.selectedKindID()
						if kind == kindTask {
							_, err = s.store.CreateTaskLink(id, name, url)
						} else {
							_, err = s.store.CreateSubtaskLink(id, name, url)
						}
					}
					s.lastErr = err
					if err == nil {
						s.editLinkID = 0
						s.loadDesc()
					}
				}
				s.linkName.SetValue("")
				s.linkName.Blur()
				s.linkInput.SetValue("")
				s.linkInput.Blur()
				if s.lastErr == nil {
					s.mode = taskLinks
				} else {
					s.mode = taskBrowse
				}
			} else {
				// Enter на названии — перейти к адресу
				s.linkName.Blur()
				s.linkInput.Focus()
			}
		case "esc":
			s.editLinkID = 0
			s.linkName.SetValue("")
			s.linkName.Blur()
			s.linkInput.SetValue("")
			s.linkInput.Blur()
			s.mode = taskLinks
		}
		return m, cmd
	case taskLinks:
		var cmd tea.Cmd
		s.linkList, cmd = s.linkList.Update(msg)
		switch msg.String() {
		case "enter":
			if item, ok := s.linkList.SelectedItem().(linkItem); ok {
				if err := openURL(item.l.URL); err != nil {
					s.lastErr = err
				}
			}
			return m, nil
		case "e":
			if item, ok := s.linkList.SelectedItem().(linkItem); ok {
				s.startLinkEdit(item.l.ID)
			}
			return m, nil
		case "n":
			s.startLinkEdit(0)
			return m, nil
		case "d":
			if item, ok := s.linkList.SelectedItem().(linkItem); ok {
				s.confirmLinkID = item.l.ID
				s.mode = taskLinkConfirm
			}
			return m, nil
		case "esc":
			s.mode = taskBrowse
			s.lastErr = nil
		}
		return m, cmd
	case taskLinkConfirm:
		switch msg.String() {
		case "y", "enter":
			kind, _ := s.selectedKindID()
			if kind == kindTask {
				s.store.DeleteTaskLink(s.confirmLinkID)
			} else {
				s.store.DeleteSubtaskLink(s.confirmLinkID)
			}
			s.mode = taskLinks
			s.loadDesc()
		case "n", "esc":
			s.mode = taskLinks
		}
		return m, nil
	case taskJournal:
		var cmd tea.Cmd
		s.journalText, cmd = s.journalText.Update(msg)
		switch msg.String() {
		case "ctrl+s":
			text := strings.TrimSpace(s.journalText.Value())
			kind, id := s.selectedKindID()
			if kind == kindSubtask && id != 0 && text != "" {
				if s.journalEditID != 0 {
					s.store.UpdateJournalEntry(s.journalEditID, text)
				} else {
					s.store.CreateJournalEntry(id, text)
				}
			}
			s.journalText.Blur()
			s.mode = taskBrowse
			s.loadDesc()
			s.descV.GotoBottom()
		case "esc":
			s.journalText.Blur()
			s.mode = taskBrowse
		}
		return m, cmd
	case taskStatusPick:
		switch msg.String() {
		case "up":
			s.statusPick.move(-1)
		case "down":
			s.statusPick.move(1)
		case "pgup":
			s.statusPick.move(-s.statusPick.visible)
		case "pgdown":
			s.statusPick.move(s.statusPick.visible)
		case "enter":
			if it, ok := s.statusPick.selected(); ok {
				for _, st := range s.statuses {
					if st.ID == it.value {
						s.applyStatus(s.statusKind, s.statusID, st)
						break
					}
				}
			}
		case "esc":
			s.mode = taskBrowse
		}
		return m, nil
	case taskStatusNote:
		var cmd tea.Cmd
		s.statusNote, cmd = s.statusNote.Update(msg)
		switch msg.String() {
		case "ctrl+s":
			note := strings.TrimSpace(s.statusNote.Value())
			if s.statusTarget != nil {
				if err := s.store.SetStatus(dbOwner(s.statusKind), s.statusID,
					s.statusTarget.Name, note, time.Now()); err == nil {
					s.now = time.Now()
					s.loadData()
				} else {
					s.lastErr = err
				}
			}
			s.statusNote.Blur()
			s.mode = taskBrowse
		case "esc":
			s.statusNote.Blur()
			s.mode = taskBrowse
		}
		return m, cmd
	case taskSearch:
		var cmd tea.Cmd
		s.searchInput, cmd = s.searchInput.Update(msg)
		switch msg.String() {
		case "enter":
			s.searchQuery = strings.TrimSpace(s.searchInput.Value())
			s.searchInput.Blur()
			s.mode = taskBrowse
			s.buildItems()
			s.loadInfo()
			s.loadDesc()
		case "esc":
			s.searchQuery = ""
			s.searchInput.Blur()
			s.mode = taskBrowse
			s.buildItems()
			s.loadInfo()
			s.loadDesc()
		default:
			// живой фильтр по мере ввода
			s.searchQuery = strings.TrimSpace(s.searchInput.Value())
			s.buildItems()
			s.loadInfo()
			s.loadDesc()
		}
		return m, cmd
	case taskTags:
		switch msg.String() {
		case "up":
			s.tagPick.move(-1)
		case "down":
			s.tagPick.move(1)
		case "pgup":
			s.tagPick.move(-s.tagPick.visible)
		case "pgdown":
			s.tagPick.move(s.tagPick.visible)
		case "n":
			s.openTagEdit(0)
		case "enter":
			if it, ok := s.tagPick.selected(); ok {
				s.openTagEdit(it.value)
			}
		case "d":
			if it, ok := s.tagPick.selected(); ok {
				s.tagConfirmID = it.value
				s.mode = taskTagConfirm
			}
		case "esc":
			s.lastErr = nil
			s.mode = taskBrowse
		}
		return m, nil
	case taskTagEdit:
		switch msg.String() {
		case "up":
			s.tagEditFocus = (s.tagEditFocus + 2) % 3
			s.focusTagEditField()
		case "down", "tab":
			s.tagEditFocus = (s.tagEditFocus + 1) % 3
			s.focusTagEditField()
		case "enter":
			switch s.tagEditFocus {
			case 0:
				s.tagTypePick.sel = 0
				for i, tt := range s.tagTypes {
					if tt.ID == s.tagEditType {
						s.tagTypePick.sel = i
					}
				}
				s.tagTypePick.clampScroll()
				s.mode = taskTagTypePick
			case 1:
				s.tagEditFocus = 2
				s.focusTagEditField()
			case 2:
				s.saveTagEdit()
			}
		case "ctrl+s":
			s.saveTagEdit()
		case "esc":
			s.lastErr = nil
			s.mode = taskTags
		default:
			switch s.tagEditFocus {
			case 1:
				s.tagEditText, _ = s.tagEditText.Update(msg)
			case 2:
				s.tagEditURL, _ = s.tagEditURL.Update(msg)
			}
		}
		return m, nil
	case taskTagTypePick:
		switch msg.String() {
		case "up":
			s.tagTypePick.move(-1)
		case "down":
			s.tagTypePick.move(1)
		case "pgup":
			s.tagTypePick.move(-s.tagTypePick.visible)
		case "pgdown":
			s.tagTypePick.move(s.tagTypePick.visible)
		case "enter":
			if it, ok := s.tagTypePick.selected(); ok {
				s.tagEditType = it.value
			}
			s.mode = taskTagEdit
		case "esc":
			s.mode = taskTagEdit
		}
		return m, nil
	case taskTagConfirm:
		switch msg.String() {
		case "y", "enter":
			s.store.DeleteTag(s.tagConfirmID)
			s.loadTags()
			s.loadData()
			s.mode = taskTags
		case "n", "esc":
			s.mode = taskTags
		}
		return m, nil
	case taskChecklistConfirm:
		switch msg.String() {
		case "y", "enter":
			s.store.DeleteChecklistItem(s.checklistConfirmID)
			kind, id := s.selectedKindID()
			if kind == kindSubtask {
				s.checklistLoad(id)
			}
			s.loadData()
			s.mode = taskChecklist
		case "n", "esc":
			s.mode = taskChecklist
		}
		return m, nil
	case taskChecklist:
		return s.updateChecklist(m, msg)
	}

	switch msg.String() {
	case "tab":
		// переключение фокуса между панелями упразднено
		return m, nil
	case "enter":
		s.startEditDescription()
		return m, nil
	case "right":
		s.expandTask()
		return m, nil
	case "left":
		kind, id := s.selectedKindID()
		if kind == kindSubtask {
			s.selectByKindID(kindTask, s.selectedTaskID())
		} else if id != 0 {
			s.expanded[id] = false
			s.buildItems()
		}
		s.loadInfo()
		s.loadDesc()
		return m, nil
	case "pgup", "pgdown":
		s.descV, _ = s.descV.Update(msg)
		return m, nil
	case "e":
		s.startEditTitle()
		return m, nil
	case "o":
		s.openLinks()
		return m, nil
	case "ctrl+j":
		s.startJournal()
		return m, nil
	case "j":
		s.editTodayJournal()
		return m, nil
	case "/":
		s.lastErr = nil
		s.searchInput.SetValue(s.searchQuery)
		s.searchInput.Focus()
		s.mode = taskSearch
		return m, nil
	case "g":
		s.openTags()
		return m, nil
	case "i":
		s.openChecklist()
		return m, nil
	}

	// стрелки вверх/вниз — навигация по списку (через s.list.Update ниже);
	// прокрутка описания — PgUp/PgDn (независимо от фокуса, см. выше).

	switch msg.String() {
	case "esc":
		if s.searchQuery != "" {
			s.searchQuery = ""
			s.buildItems()
			s.loadInfo()
			s.loadDesc()
		}
		return m, nil
	case "ctrl+l":
		s.toggleTimer()
		return m, nil
	case "[":
		s.switchProject(-1)
		return m, nil
	case "]":
		s.switchProject(1)
		return m, nil
	case "ctrl+up":
		s.moveSelected(-1)
		return m, nil
	case "ctrl+down":
		s.moveSelected(1)
		return m, nil
	case "n":
		s.startNewTask()
		return m, nil
	case "a":
		s.startNewSubtask()
		return m, nil
	case "d":
		s.startDelete()
		return m, nil
	case "x":
		s.shiftStatus(1)
		return m, nil
	case "z":
		s.shiftStatus(-1)
		return m, nil
	case "c":
		s.openStatusPick()
		return m, nil
	}
	var cmd tea.Cmd
	beforeKind, beforeID := s.selectedKindID()
	s.list, cmd = s.list.Update(msg)
	afterKind, afterID := s.selectedKindID()
	if beforeKind != afterKind || beforeID != afterID {
		s.loadInfo()
		s.loadDesc()
		s.descV.GotoTop()
	}
	return m, cmd
}
