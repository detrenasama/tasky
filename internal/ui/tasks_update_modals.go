package ui

import (
	"fmt"
	"strings"
	"time"

	"github.com/charmbracelet/bubbletea"
)

// updateTasksModal маршрутизирует клавиши по модальным режимам экрана задач.
// Возвращает (_, _, false), если текущий режим — не модальный (навигация
// обрабатывается в updateTasksBase).
func (m *model) updateTasksModal(msg tea.KeyMsg) (tea.Model, tea.Cmd, bool) {
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
		return m, cmd, true
	case taskTitleEdit:
		var cmd tea.Cmd
		s.input, cmd = s.input.Update(msg)
		switch msg.String() {
		case "enter":
			title := strings.TrimSpace(s.input.Value())
			if title == "" {
				s.lastErr = fmt.Errorf("название не может быть пустым")
				return m, cmd, true
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
		return m, cmd, true
	case taskConfirm:
		switch msg.String() {
		case "y", "enter":
			s.deleteItem(s.confirmKind, s.confirmID)
			s.mode = taskBrowse
			s.loadData()
		case "n", "esc":
			s.mode = taskBrowse
		}
		return m, nil, true
	case taskDescModal:
		switch s.dmState {
		case dmView, dmSelect:
			switch msg.String() {
			case "up", "k":
				if s.dmState == dmView {
					s.descViewer.scrollUp(1)
				} else {
					s.descViewer.moveUp()
				}
			case "down", "j":
				if s.dmState == dmView {
					s.descViewer.scrollDown(1)
				} else {
					s.descViewer.moveDown()
				}
			case "pgup":
				s.descViewer.scrollUp(s.descViewer.height)
			case "pgdown":
				s.descViewer.scrollDown(s.descViewer.height)
			case "left", "h":
				if s.dmState == dmSelect {
					s.descViewer.moveLeft()
				}
			case "right", "l":
				if s.dmState == dmSelect {
					s.descViewer.moveRight()
				}
			case "e":
				s.descText.SetValue(s.descWork)
				s.descText.Focus()
				s.dmState = dmEdit
			case "E", "shift+e":
				path, cmd, err := openInEditor(s.descWork)
				if err != nil {
					s.notice = "Не удалось открыть редактор: " + err.Error()
					return m, nil, true
				}
				s.extEditPath = path
				s.extEditMode = 2
				return m, cmd, true
			case "alt+enter":
				path, cmd, err := openInEditor(s.descWork)
				if err != nil {
					s.notice = "Не удалось открыть редактор: " + err.Error()
					return m, nil, true
				}
				s.extEditPath = path
				s.extEditMode = 2
				return m, cmd, true
			case "v":
				if s.dmState == dmView {
					lines := s.descViewer.layout()
					top := s.descViewer.scroll
					if top < 0 || top >= len(lines) {
						top = 0
					}
					s.descViewer.plain = false
					s.descViewer.cursor = lines[top].start
					s.descViewer.anchor = -1
					s.dmState = dmSelect
				} else {
					s.descViewer.plain = true
					s.descViewer.anchor = -1
					s.descViewer.scroll = s.descViewer.lineOfCursor(s.descViewer.layout())
					s.dmState = dmView
				}
			case " ":
				if s.dmState == dmSelect {
					s.descViewer.anchor = s.descViewer.cursor
				}
			case "enter":
				if s.dmState == dmSelect {
					s.copyDescSelection()
				}
			case "y":
				s.copyDescSelection()
			case "d":
				if s.dmState == dmSelect && s.descViewer.anchor >= 0 {
					s.deleteDescSelection()
					s.dmState = dmView
				}
			case "ctrl+s":
				s.saveDescWork()
				s.dmState = dmView
				s.refreshDescViewer()
			case "esc":
				if s.descWork != s.desc {
					s.dmPrev = s.dmState
					s.dmState = dmDiscard
				} else {
					s.mode = taskBrowse
				}
			}
			return m, nil, true
		case dmEdit:
			var cmd tea.Cmd
			s.descText, cmd = s.descText.Update(msg)
			switch msg.String() {
			case "ctrl+s":
				s.descWork = s.descText.Value()
				s.saveDescWork()
				s.descText.Blur()
				s.dmState = dmView
				s.refreshDescViewer()
			case "esc":
				if s.descText.Value() != s.descWork {
					s.dmPrev = dmEdit
					s.dmState = dmDiscard
				} else {
					s.descText.Blur()
					s.dmState = dmView
				}
			}
			return m, cmd, true
		case dmDiscard:
			switch msg.String() {
			case "y":
				s.descWork = s.desc
				s.refreshDescViewer()
				s.dmState = dmView
			case "n", "esc":
				s.dmState = s.dmPrev
			}
			return m, nil, true
		}
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
		return m, cmd, true
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
			return m, nil, true
		case "e":
			if item, ok := s.linkList.SelectedItem().(linkItem); ok {
				s.startLinkEdit(item.l.ID)
			}
			return m, nil, true
		case "n":
			s.startLinkEdit(0)
			return m, nil, true
		case "d":
			if item, ok := s.linkList.SelectedItem().(linkItem); ok {
				s.confirmLinkID = item.l.ID
				s.mode = taskLinkConfirm
			}
			return m, nil, true
		case "esc":
			s.mode = taskBrowse
			s.lastErr = nil
		}
		return m, cmd, true
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
		return m, nil, true
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
			if s.journalText.Value() != s.journalOrig {
				s.mode = taskJournalDiscard
				break
			}
			s.journalText.Blur()
			s.mode = taskBrowse
		}
		return m, cmd, true
	case taskJournalDiscard:
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
		case "y":
			s.journalText.Blur()
			s.mode = taskBrowse
			s.loadDesc()
			s.descV.GotoBottom()
		case "n", "esc":
			s.mode = taskJournal
			s.journalText.Focus()
		}
		return m, nil, true
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
		return m, nil, true
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
		return m, cmd, true
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
		return m, cmd, true
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
		return m, nil, true
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
		return m, nil, true
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
		return m, nil, true
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
		return m, nil, true
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
		return m, nil, true
	case taskChecklist:
		mm, cmd := s.updateChecklist(m, msg)
		return mm, cmd, true
	case taskTimeList:
		switch msg.String() {
		case "up", "k":
			s.timePick.move(-1)
		case "down", "j":
			s.timePick.move(1)
		case "pgup":
			s.timePick.move(-s.timePick.visible)
		case "pgdown":
			s.timePick.move(s.timePick.visible)
		case "enter":
			s.startTimeEdit()
		case "d":
			s.startTimeDelete()
		case "esc":
			s.mode = taskBrowse
		}
		return m, nil, true
	case taskTimeEdit:
		switch msg.String() {
		case "left", "h":
			base := (s.timeField / 5) * 5
			off := s.timeField % 5
			s.timeField = base + (off-1+5)%5
		case "right", "l":
			base := (s.timeField / 5) * 5
			off := s.timeField % 5
			s.timeField = base + (off+1)%5
		case "tab":
			if s.editHasEnd {
				if s.timeField < 5 {
					s.timeField += 5
				} else {
					s.timeField -= 5
				}
			}
		case "up":
			s.adjustTimeField(s.timeField, 1)
		case "down":
			s.adjustTimeField(s.timeField, -1)
		case "shift+up":
			s.adjustTimeField(s.timeField, s.shiftDelta())
		case "shift+down":
			s.adjustTimeField(s.timeField, -s.shiftDelta())
		case "enter":
			err := s.store.UpdateTimeEntry(s.editTimeID, s.editStart, s.editEnd)
			s.lastErr = err
			if err == nil {
				if kind, id := s.selectedKindID(); kind == kindSubtask {
					s.entries, _ = s.store.TimeEntriesBySubtask(id)
				}
				s.rebuildTimePick()
				s.timePick.sel = 0
				for i, it := range s.timePick.items {
					if it.value == s.editTimeID {
						s.timePick.sel = i
						break
					}
				}
				s.timePick.clampScroll()
				s.loadInfo()
				s.mode = taskTimeList
			}
		case "esc":
			s.lastErr = nil
			s.mode = taskTimeList
		}
		return m, nil, true
	case taskTimeDelete:
		switch msg.String() {
		case "y", "enter":
			if err := s.store.DeleteTimeEntry(s.confirmTimeID); err != nil {
				s.lastErr = err
			} else {
				s.lastErr = nil
				if kind, id := s.selectedKindID(); kind == kindSubtask {
					s.entries, _ = s.store.TimeEntriesBySubtask(id)
				}
				s.rebuildTimePick()
			}
			s.mode = taskTimeList
		case "n", "esc":
			s.mode = taskTimeList
		}
		return m, nil, true
	}
	return m, nil, false
}
