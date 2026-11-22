package ui

import (
	"strings"

	"github.com/charmbracelet/bubbletea"
)

// updateTasksJournal обрабатывает модалку записи журнала подзадачи.
func (s *tasksScreen) updateTasksJournal(m *model, msg tea.KeyMsg) (tea.Model, tea.Cmd, bool) {
	if s.mode != taskJournal {
		return m, nil, false
	}
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
		s.esc.Reset()
		s.mode = taskBrowse
		s.loadDesc()
		s.descV.GotoBottom()
	case "esc":
		if !s.esc.Press(s.journalText.Value() != s.journalOrig) {
			break
		}
		s.journalText.Blur()
		s.mode = taskBrowse
		s.loadDesc()
		s.descV.GotoBottom()
	}
	return m, cmd, true
}
