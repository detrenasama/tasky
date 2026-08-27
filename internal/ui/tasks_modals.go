package ui

import (
	"github.com/charmbracelet/bubbletea"
)

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

// startEditDescription открывает модалку описания в режиме просмотра
// (Enter в списке). Аналогично openDescModal(false).
func (s *tasksScreen) startEditDescription() {
	s.openDescModal(false)
}

// openDescModal открывает крупную модалку описания. sel=true — сразу в режиме
// визуального выделения (клавиша v в фокусе описания).
func (s *tasksScreen) openDescModal(sel bool) {
	s.lastErr = nil
	s.notice = ""
	s.loadDesc()
	s.descWork = s.desc
	mW, mH := s.descModalDims()
	v := newDescViewer(s.descWork, mW-4, mH-4)
	v.plain = true
	v.scroll = 0
	if sel {
		v.plain = false
		v.cursor = 0
		v.anchor = 0
		s.dmState = dmSelect
	} else {
		s.dmState = dmView
	}
	s.descViewer = v
	s.mode = taskDescModal
}

// descModalDims — размеры модалки описания (~4/5 экрана, крупная).
func (s *tasksScreen) descModalDims() (int, int) {
	fw, fh := s.fullW, s.fullH
	if fw <= 0 {
		fw = 150
	}
	if fh <= 0 {
		fh = 40
	}
	mW := max(fw*4/5, 60)
	if mW > fw-4 {
		mW = max(fw-4, 20)
	}
	mH := max(fh*4/5, 20)
	if mH > fh-4 {
		mH = max(fh-4, 10)
	}
	return mW, mH
}

// refreshDescViewer пересоздаёт descViewer из рабочей копии (после правки/
// удаления), в режиме plain с прокруткой с начала.
func (s *tasksScreen) refreshDescViewer() {
	mW, mH := s.descModalDims()
	v := newDescViewer(s.descWork, mW-4, mH-4)
	v.plain = true
	v.scroll = 0
	s.descViewer = v
}

// deleteDescSelection удаляет выделенный диапазон из рабочей копии описания.
func (s *tasksScreen) deleteDescSelection() {
	rs := []rune(s.descWork)
	a, b := s.descViewer.anchor, s.descViewer.cursor
	if a > b {
		a, b = b, a
	}
	if a < 0 {
		a = 0
	}
	if b > len(rs) {
		b = len(rs)
	}
	s.descWork = string(append(rs[:a], rs[b:]...))
	s.refreshDescViewer()
}

// saveDescWork сохраняет рабочую копию описания в БД в зависимости от
// выбранной задачи/подзадачи.
func (s *tasksScreen) saveDescWork() {
	kind, id := s.selectedKindID()
	if kind == kindTask {
		s.store.UpdateTaskDescription(id, s.descWork)
	} else {
		s.store.UpdateSubtaskDescription(id, s.descWork)
	}
	s.desc = s.descWork
	s.loadDesc()
}

// copyDescSelection копирует выделенное (или весь текст, если выделения нет)
// в буфер обмена и сбрасывает выделение, возвращаясь в режим просмотра.
func (s *tasksScreen) copyDescSelection() {
	text := s.descWork
	if s.dmState == dmSelect {
		text = s.descViewer.selectedText()
	}
	if text == "" {
		text = s.descWork
	}
	if err := copyToClipboard(text); err != nil {
		s.notice = "Не удалось скопировать: " + err.Error()
	} else {
		s.notice = "Скопировано в буфер обмена"
	}
	s.descViewer.plain = true
	s.descViewer.anchor = -1
	s.descViewer.scroll = s.descViewer.lineOfCursor(s.descViewer.layout())
	s.dmState = dmView
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

// updateTasks — диспетчер экран задач: сначала маршрутизация модальных
// режимов (updateTasksModal), затем базовая навигация (updateTasksBase).
func (m *model) updateTasks(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	m.tasks.notice = ""
	if mm, cmd, handled := m.updateTasksModal(msg); handled {
		return mm, cmd
	}
	return m.updateTasksBase(msg)
}
