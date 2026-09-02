package ui

import (
	"github.com/charmbracelet/bubbles/list"
	"github.com/detrenasama/tasky/internal/db"
	"github.com/detrenasama/tasky/internal/ui/theme"
	"strings"
	"time"
)

func (s *tasksScreen) loadData() {
	if s.projIdx < 0 || s.projIdx >= len(s.projects) {
		s.tasks = nil
		s.subs = nil
		s.buildItems()
		s.loadInfo()
		s.loadDesc()
		return
	}
	s.tasks, _ = s.store.TasksByProject(s.currentProjectID())
	s.subs, _ = s.store.SubtasksByProject(s.currentProjectID())
	s.checklistCounts, _ = s.store.ChecklistCounts(s.currentProjectID())
	s.journalTexts, _ = s.store.JournalTexts(s.currentProjectID())
	s.tagsMap, _ = s.store.TagsByProject(s.currentProjectID())
	s.tagsText = make(map[int64]string, len(s.tagsMap))
	for taskID, tg := range s.tagsMap {
		parts := make([]string, 0, len(tg))
		for _, t := range tg {
			parts = append(parts, t.Text)
		}
		s.tagsText[taskID] = strings.Join(parts, " ")
	}
	s.buildItems()
	s.loadInfo()
	s.loadDesc()
}

func (s *tasksScreen) loadInfo() {
	s.run, _ = s.store.RunningSession()
	s.entries = nil
	s.history = nil
	s.checklistItems = nil
	kind, id := s.selectedKindID()
	if id == 0 {
		return
	}
	if kind == kindSubtask {
		s.entries, _ = s.store.TimeEntriesBySubtask(id)
		s.checklistItems, _ = s.store.ChecklistItems(id)
	}
	s.history, _ = s.store.StatusHistory(dbOwner(kind), id)
}

// loadDesc подгружает описание, ссылки и (для подзадачи) записи журнала
// выбранного элемента и пересобирает колонку описания.
func (s *tasksScreen) loadDesc() {
	kind, id := s.selectedKindID()
	s.desc = ""
	s.links = nil
	s.journal = nil
	switch kind {
	case kindTask:
		if id != 0 {
			s.desc, _ = s.store.TaskDescription(id)
			s.links, _ = s.store.TaskLinks(id)
		}
	case kindSubtask:
		if id != 0 {
			s.desc, _ = s.store.SubtaskDescription(id)
			s.links, _ = s.store.SubtaskLinks(id)
			s.journal, _ = s.store.JournalEntries(id)
		}
	}
	items := make([]list.Item, len(s.links))
	for i, l := range s.links {
		items[i] = linkItem{l}
	}
	s.linkList.SetItems(items)
	// разворачиваем список ссылок на все элементы, чтобы bubbles/list не
	// прыгал страницами (в модалке список ссылок не ограничен окном).
	sizeListHeight(&s.linkList, s.linkDelegate, len(s.links), 8)
	s.refreshDesc()
}

// refreshDesc собирает контент viewport колонки описания: для задачи —
// описание и ссылки; для подзадачи — блок «Описание» (1/3) и блок «Журнал»
// (2/3), разделённые линией.
func (s *tasksScreen) refreshDesc() {
	w := max(s.descV.Width, 1)
	kind, id := s.selectedKindID()
	var body string
	switch {
	case id == 0:
		body = strings.Join([]string{theme.Faint("Выберите задачу или подзадачу.")}, "\n")
	case kind == kindTask:
		body = strings.Join(append([]string{theme.Faint("Описание")}, s.descBody(w)...), "\n")
	case kind == kindSubtask:
		h1, h2 := s.descSplit()
		descBlock := padLines(strings.Join(append([]string{theme.Faint("Описание")}, s.descBody(w)...), "\n"), w, h1)
		journalBlock := padLines(strings.Join(append([]string{theme.Faint("Журнал")}, s.journalBody(w)...), "\n"), w, h2)
		divider := theme.Faint(strings.Repeat("─", w))
		body = descBlock + "\n" + divider + "\n" + journalBlock
	}
	s.descV.SetContent(body)
}

// descSplit делит высоту колонки описания на блок описания и блок журнала
// в пропорции 1:2 (минус строка-разделитель).
func (s *tasksScreen) descSplit() (int, int) {
	H := max(s.descV.Height, 3)
	h1 := max((H-1)/3, 2)
	if H-h1-1 < 3 {
		h1 = H - 4
		if h1 < 2 {
			h1 = 2
		}
	}
	return h1, max(H-h1-1, 0)
}

// descBody — строки описания и ссылок выбранного элемента.
func (s *tasksScreen) descBody(w int) []string {
	var body []string
	desc := strings.TrimSpace(s.desc)
	if desc == "" {
		body = append(body, theme.Faint("Описание пустое. Нажмите Enter, чтобы добавить."))
	} else {
		body = append(body, wrapText(desc, w))
	}
	if len(s.links) > 0 {
		body = append(body, "", theme.Faint("Ссылки:"))
		for _, l := range s.links {
			label := l.Name
			if label == "" {
				label = l.URL
			}
			body = append(body, truncateWEnd(linkStyle.Render("• "+label), w))
		}
	}
	return body
}

// journalBody — строки записей журнала в хронологическом порядке: штамп
// даты/времени и текст записи.
func (s *tasksScreen) journalBody(w int) []string {
	if len(s.journal) == 0 {
		return []string{theme.Faint("Записей нет. Ctrl+J — добавить.")}
	}
	var body []string
	for _, e := range s.journal {
		body = append(body, theme.Faint(e.CreatedAt.Format("02.01.2006 15:04")))
		body = append(body, wrapText(e.Text, w))
		body = append(body, "")
	}
	return body
}

func (s *tasksScreen) currentProjectID() int64 {
	if s.projIdx < 0 || s.projIdx >= len(s.projects) {
		return 0
	}
	return s.projects[s.projIdx].ID
}

func (s *tasksScreen) currentProjectName() string {
	if s.projIdx < 0 || s.projIdx >= len(s.projects) {
		return ""
	}
	return s.projects[s.projIdx].Name
}

func (s *tasksScreen) buildItems() {
	if q := strings.ToLower(strings.TrimSpace(s.searchQuery)); q != "" {
		s.buildSearchItems(q)
		return
	}
	s.items = []list.Item{}
	for _, t := range s.tasks {
		if s.hiddenDue(t) {
			continue
		}
		s.items = append(s.items, taskItem{t: t, expanded: s.expanded[t.ID], scr: s})
		if s.expanded[t.ID] {
			for _, st := range s.subs {
				if st.TaskID == t.ID {
					s.items = append(s.items, subItem{st: st, scr: s})
				}
			}
		}
	}
	s.finishBuildItems()
}

// hiddenDue возвращает true, если задачу нужно скрыть: статус завершённого
// типа, задача завершена hideDays и более дней назад. Поиск (buildSearchItems)
// фильтр не применяет — скрытые задачи остаются доступны через поиск.
func (s *tasksScreen) hiddenDue(t db.Task) bool {
	if s.hideDays <= 0 || t.CompletedAt == nil {
		return false
	}
	st, ok := s.statusDef(t.Status)
	if !ok || st.Type != "done" {
		return false
	}
	return s.now.Sub(*t.CompletedAt) >= time.Duration(s.hideDays)*24*time.Hour
}

// buildSearchItems собирает дерево по запросу q (регистронезависимо):
// задача попадает, если совпадает её название или описание (тогда видны все
// её подзадачи); подзадача — если совпадает название, описание или текст
// записей журнала (тогда видна только она).
func (s *tasksScreen) buildSearchItems(q string) {
	s.items = []list.Item{}
	for _, t := range s.tasks {
		taskMatch := strings.Contains(strings.ToLower(t.Title), q) ||
			strings.Contains(strings.ToLower(t.Description), q) ||
			strings.Contains(strings.ToLower(s.tagsText[t.ID]), q)
		var subs []db.SubtaskWithTime
		if taskMatch {
			for _, st := range s.subs {
				if st.TaskID == t.ID {
					subs = append(subs, st)
				}
			}
		} else {
			for _, st := range s.subs {
				if st.TaskID == t.ID && s.subMatches(st, q) {
					subs = append(subs, st)
				}
			}
		}
		if taskMatch || len(subs) > 0 {
			s.items = append(s.items, taskItem{t: t, expanded: true, scr: s})
			for _, st := range subs {
				s.items = append(s.items, subItem{st: st, scr: s})
			}
		}
	}
	s.finishBuildItems()
}

func (s *tasksScreen) subMatches(st db.SubtaskWithTime, q string) bool {
	return strings.Contains(strings.ToLower(st.Title), q) ||
		strings.Contains(strings.ToLower(st.Description), q) ||
		strings.Contains(strings.ToLower(s.journalTexts[st.ID]), q)
}

func (s *tasksScreen) finishBuildItems() {
	selKind, selID := s.selectedKindID()
	s.list.SetItems(s.items)
	if len(s.items) > 0 {
		idx := s.indexByKindID(selKind, selID)
		if idx < 0 {
			idx = 0
		}
		s.list.Select(idx)
	}
	s.sizeList()
	s.syncScroll()
}

// sizeList «разворачивает» bubbles/list на все элементы, чтобы он не
// прыгал страницами, и обновляет верхнюю видимую строку.
func (s *tasksScreen) sizeList() {
	sizeListHeight(&s.list, s.listDelegate, len(s.items), s.listH)
}

// syncScroll удерживает курсор в отступе listScrollOff от краёв окна.
func (s *tasksScreen) syncScroll() {
	syncListTop(&s.list, &s.listTop, s.listDelegate, len(s.items), s.listH)
}

// listView возвращает только видимое окно списка (плавная прокрутка).
func (s *tasksScreen) listView() string {
	if len(s.items) == 0 {
		return strings.Repeat("\n", max(s.listH-1, 0))
	}
	return clipList(s.list, s.listTop, s.listH)
}

func (s *tasksScreen) selectedKindID() (paneKind, int64) {
	switch item := s.list.SelectedItem().(type) {
	case taskItem:
		return kindTask, item.t.ID
	case subItem:
		return kindSubtask, item.st.ID
	}
	return kindTask, 0
}

func (s *tasksScreen) indexByKindID(kind paneKind, id int64) int {
	if id == 0 {
		return -1
	}
	for i, item := range s.items {
		switch it := item.(type) {
		case taskItem:
			if kind == kindTask && it.t.ID == id {
				return i
			}
		case subItem:
			if kind == kindSubtask && it.st.ID == id {
				return i
			}
		}
	}
	return -1
}

func (s *tasksScreen) selectByKindID(kind paneKind, id int64) {
	idx := s.indexByKindID(kind, id)
	if idx < 0 {
		return
	}
	s.list.Select(idx)
	s.syncScroll()
}

func (s *tasksScreen) selectedKind() paneKind {
	switch s.list.SelectedItem().(type) {
	case taskItem:
		return kindTask
	case subItem:
		return kindSubtask
	}
	return kindTask
}

// selectedTaskID возвращает ID задачи: выбранной или родителя выбранной подзадачи.
func (s *tasksScreen) selectedTaskID() int64 {
	switch item := s.list.SelectedItem().(type) {
	case taskItem:
		return item.t.ID
	case subItem:
		return item.st.TaskID
	}
	return 0
}

// expandTask раскрывает выбранную задачу (→).
func (s *tasksScreen) expandTask() {
	item, ok := s.list.SelectedItem().(taskItem)
	if !ok {
		return
	}
	if s.searchQuery != "" {
		return
	}
	if s.expanded[item.t.ID] {
		return
	}
	s.expanded[item.t.ID] = true
	s.buildItems()
}

// collapseTask сворачивает выбранную задачу (←).
func (s *tasksScreen) collapseTask() {
	item, ok := s.list.SelectedItem().(taskItem)
	if !ok {
		return
	}
	if s.searchQuery != "" {
		return
	}
	if !s.expanded[item.t.ID] {
		return
	}
	s.expanded[item.t.ID] = false
	s.buildItems()
}

// toggleTask переключает свёрнуто/развёрнуто выбранной задачи (tab).
func (s *tasksScreen) toggleTask() {
	item, ok := s.list.SelectedItem().(taskItem)
	if !ok {
		return
	}
	if s.searchQuery != "" {
		return
	}
	s.expanded[item.t.ID] = !s.expanded[item.t.ID]
	s.buildItems()
}

func (s *tasksScreen) canCreate(kind paneKind) bool {
	if kind == kindTask {
		return s.projIdx >= 0 && s.projIdx < len(s.projects)
	}
	return s.selectedTaskID() != 0
}

func (s *tasksScreen) canDelete() bool {
	return s.list.Index() >= 0 && len(s.items) > 0
}

func (s *tasksScreen) switchProject(dir int) {
	if len(s.projects) < 2 {
		return
	}
	s.projIdx = (s.projIdx + dir + len(s.projects)) % len(s.projects)
	s.loadData()
}

// moveSelected перемещает выбранную задачу/подзадачу на одну позицию
// (dir = -1 вверх, +1 вниз) и сохраняет выделение.
func (s *tasksScreen) moveSelected(dir int) {
	kind, id := s.selectedKindID()
	if id == 0 {
		return
	}
	switch kind {
	case kindTask:
		idx := -1
		for i, t := range s.tasks {
			if t.ID == id {
				idx = i
				break
			}
		}
		if idx < 0 || idx+dir < 0 || idx+dir >= len(s.tasks) {
			return
		}
		if err := s.store.MoveTask(id, dir); err != nil {
			s.lastErr = err
			return
		}
	case kindSubtask:
		parentID := s.selectedTaskID()
		idx, siblings := -1, 0
		for _, st := range s.subs {
			if st.TaskID != parentID {
				continue
			}
			if st.ID == id {
				idx = siblings
			}
			siblings++
		}
		if idx < 0 || idx+dir < 0 || idx+dir >= siblings {
			return
		}
		if err := s.store.MoveSubtask(id, dir); err != nil {
			s.lastErr = err
			return
		}
	}
	s.lastErr = nil
	s.loadData()
	s.selectByKindID(kind, id)
	s.loadInfo()
	s.loadDesc()
}

func (s *tasksScreen) toggleTimer() {
	item, ok := s.list.SelectedItem().(subItem)
	if !ok {
		return
	}
	now := time.Now()
	if item.st.ActiveSince != nil {
		s.store.StopSession(item.st.ID, now)
	} else {
		s.store.StartSession(item.st.ID, now)
	}
	s.now = now
	s.loadData()
	s.today, _ = s.store.TodayTotal(now)
	s.weekly, _ = s.store.WeeklyTotal(now)
}

func (s *tasksScreen) createItem(kind paneKind, title string) (int64, error) {
	if kind == kindTask {
		t, err := s.store.CreateTask(s.currentProjectID(), title)
		return t.ID, err
	}
	st, err := s.store.CreateSubtask(s.selectedTaskID(), title)
	return st.ID, err
}

func (s *tasksScreen) deleteItem(kind paneKind, id int64) {
	if kind == kindTask {
		s.store.DeleteTask(id)
		return
	}
	s.store.DeleteSubtask(id)
}
